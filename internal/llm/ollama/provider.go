package ollama

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/ollama/ollama/api"
	openaisdk "github.com/openai/openai-go"

	"github.com/dmytrogajewski/spin/internal/llm"
)

const maxPreviewLen = 100

// ErrModelIsRequired is a sentinel error.
var ErrModelIsRequired = errors.New("model is required")

const (
	// DefaultBaseURL is the default Ollama API endpoint.
	DefaultBaseURL = "http://localhost:11434"
)

// Config configures the Ollama provider.
type Config struct {
	// BaseURL is the API endpoint URL (default: http://localhost:11434)
	BaseURL string

	// Model is the model name to use (required).
	Model string

	// Timeout is the request timeout (defaults to 5 minutes).
	Timeout time.Duration

	// StreamTimeout optionally bounds streaming calls (default: 30m).
	StreamTimeout time.Duration
}

// Provider implements the Ollama LLM provider using the native Ollama API client.
type Provider struct {
	// Ollama SDK client.
	client *api.Client

	model   string
	baseURL string
	timeout time.Duration
	logger  *slog.Logger

	// Context length detected from model metadata (guarded by ctxLenOnce).
	ctxLenOnce     sync.Once
	detectedCtxLen int
}

// NewProvider creates a new Ollama provider.
func NewProvider(cfg Config) (*Provider, error) {
	if cfg.Model == "" {
		return nil, ErrModelIsRequired
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	// Validate base URL early.
	baseURLParsed, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse base URL: %w", err)
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = llm.DefaultTimeout
	}

	// Create Ollama SDK client.
	ollamaClient := api.NewClient(baseURLParsed, &http.Client{
		Timeout: timeout,
	})

	return &Provider{
		client:  ollamaClient,
		model:   cfg.Model,
		baseURL: baseURL,
		timeout: timeout,
		logger:  slog.Default(),
	}, nil
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "ollama"
}

// Capabilities returns the provider's capabilities.
func (p *Provider) Capabilities() llm.Capabilities {
	return llm.Capabilities{
		Streaming:       true,
		FunctionCalling: true,  // Ollama supports function calling.
		Vision:          false, // Vision not supported yet.
		ContextWindow:   p.detectedCtxLen,
	}
}

// Models lists available models using Ollama SDK.
func (p *Provider) Models(ctx context.Context) ([]openaisdk.Model, error) {
	resp, err := p.client.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list models: %w", err)
	}

	models := make([]openaisdk.Model, 0, len(resp.Models))
	for _, m := range resp.Models {
		models = append(models, openaisdk.Model{
			ID:      m.Name,
			Created: 0, // Ollama doesn't provide creation time.
			Object:  "model",
		})
	}

	return models, nil
}

// FallbackContextLength is used when the model's actual context length cannot be detected.
const FallbackContextLength = 32768

// detectContextLength queries Ollama for the model's maximum context length.
// Falls back to FallbackContextLength if the query fails.
func (p *Provider) detectContextLength(ctx context.Context) int {
	resp, err := p.client.Show(ctx, &api.ShowRequest{Model: p.model})
	if err != nil {
		p.logger.DebugContext(ctx, "ollama: failed to query model info for context length", "model", p.model, "error", err)

		return FallbackContextLength
	}

	// Look for context_length in model_info (key format: "<arch>.context_length").
	for k, v := range resp.ModelInfo {
		if strings.HasSuffix(k, ".context_length") || k == "context_length" {
			if val, ok := v.(float64); ok {
				if int(val) > 0 {
					p.logger.InfoContext(ctx, "ollama: detected model context length", "model", p.model, "context_length", int(val))

					return int(val)
				}
			}
		}
	}

	p.logger.DebugContext(ctx, "ollama: no context_length found in model info, using fallback", "model", p.model)

	return FallbackContextLength
}

// setContextOptions applies num_ctx to the request options.
// Uses autoTuneCtxLen if set, otherwise detects from the model.
func (p *Provider) setContextOptions(ctx context.Context, opts map[string]any) map[string]any {
	if opts == nil {
		opts = make(map[string]any)
	}

	// Detect context length on first call (thread-safe via sync.Once).
	p.ctxLenOnce.Do(func() {
		p.detectedCtxLen = p.detectContextLength(ctx)
	})

	opts["num_ctx"] = p.detectedCtxLen

	return opts
}

// Complete performs a non-streaming completion request using Ollama's native API.
func (p *Provider) Complete(ctx context.Context, params openaisdk.ChatCompletionNewParams) (*openaisdk.ChatCompletion, error) {
	req := p.buildChatRequest(ctx, params, false)

	// Call Ollama API
	// Note: Ollama sends multiple callbacks even for non-streaming requests.
	var (
		resp         api.ChatResponse
		fullContent  strings.Builder
		fullThinking strings.Builder
	)

	callbackCount := 0

	err := p.client.Chat(ctx, req, func(r api.ChatResponse) error {
		callbackCount++

		resp = r // Keep the last response for metadata.
		if r.Message.Content != "" {
			fullContent.WriteString(r.Message.Content)
		}

		if r.Message.Thinking != "" {
			fullThinking.WriteString(r.Message.Thinking)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("ollama chat: %w", err)
	}

	// Use accumulated content, merging thinking into <think> tags.
	resp.Message.Content = mergeThinkingContent(fullThinking.String(), fullContent.String())
	resp.Message.Thinking = ""

	// Fix tool calls with empty function names.
	resp.Message.ToolCalls = filterToolCalls(ctx, resp.Message.ToolCalls, req.Tools, p.logger)

	// Debug: Log the Ollama response.
	p.logger.DebugContext(ctx, "Ollama Complete", "callbacks", callbackCount, "content_length", len(resp.Message.Content))
	logResponsePreview(ctx, p.logger, resp.Message.Content)

	return convertOllamaResponseToOpenAI(ctx, resp, p.model, p.logger), nil
}

// buildChatRequest converts OpenAI params into an Ollama ChatRequest.
func (p *Provider) buildChatRequest(ctx context.Context, params openaisdk.ChatCompletionNewParams, streaming bool) *api.ChatRequest {
	req := &api.ChatRequest{
		Model: p.model,
	}

	if streaming {
		req.Stream = new(bool)
		*req.Stream = true
	}

	p.convertMessages(ctx, params, req)
	p.convertTools(ctx, params, req)
	p.setRequestOptions(ctx, params, req)

	return req
}

// convertMessages converts OpenAI messages to Ollama format on the request.
func (p *Provider) convertMessages(ctx context.Context, params openaisdk.ChatCompletionNewParams, req *api.ChatRequest) {
	if len(params.Messages) == 0 {
		return
	}

	toolCallIDToName := buildToolCallIDToNameMap(ctx, params.Messages, p.logger)

	req.Messages = make([]api.Message, len(params.Messages))
	for i, msg := range params.Messages {
		req.Messages[i] = convertMessageToOllama(ctx, msg, toolCallIDToName, p.logger)
	}

	for i, m := range req.Messages {
		p.logger.DebugContext(ctx, "ollama stream message",
			"index", i,
			"role", m.Role,
			"content_len", len(m.Content),
			"tool_calls", len(m.ToolCalls),
			"tool_name", m.ToolName)
	}
}

// convertTools converts OpenAI tools to Ollama format on the request.
func (p *Provider) convertTools(ctx context.Context, params openaisdk.ChatCompletionNewParams, req *api.ChatRequest) {
	if len(params.Tools) == 0 {
		return
	}

	req.Tools = make([]api.Tool, len(params.Tools))
	for i, tool := range params.Tools {
		req.Tools[i] = convertToolToOllama(tool)
	}

	p.logger.DebugContext(ctx, "ollama request with tools", "tool_count", len(req.Tools), "model", p.model)
}

// setRequestOptions sets temperature, max tokens, and context options on the request.
func (p *Provider) setRequestOptions(ctx context.Context, params openaisdk.ChatCompletionNewParams, req *api.ChatRequest) {
	if params.Temperature.Valid() {
		req.Options = ensureOptionsMap(req.Options)
		req.Options["temperature"] = params.Temperature.Value
	}

	if params.MaxTokens.Valid() {
		req.Options = ensureOptionsMap(req.Options)
		req.Options["num_predict"] = params.MaxTokens.Value
	}

	req.Options = p.setContextOptions(ctx, req.Options)
}

// ensureOptionsMap returns the provided options map or creates a new one if nil.
func ensureOptionsMap(opts map[string]any) map[string]any {
	if opts == nil {
		return make(map[string]any)
	}

	return opts
}

// mergeThinkingContent merges thinking and content strings into a single content string.
func mergeThinkingContent(thinking, content string) string {
	if thinking != "" {
		return "<think>" + thinking + "</think>" + content
	}

	return content
}

// filterToolCalls filters tool calls, inferring names for nameless calls and removing phantom ones.
func filterToolCalls(ctx context.Context, toolCalls []api.ToolCall, tools []api.Tool, logger *slog.Logger) []api.ToolCall {
	if len(toolCalls) == 0 {
		return toolCalls
	}

	filtered := toolCalls[:0]

	for _, tc := range toolCalls {
		if tc.Function.Name != "" {
			filtered = append(filtered, tc)

			continue
		}

		if inferred := inferToolName(ctx, tc.Function.Arguments.ToMap(), tools, logger); inferred != "" {
			tc.Function.Name = inferred
			logger.InfoContext(ctx, "ollama: inferred tool name for nameless tool call",
				"name", inferred, "args", tc.Function.Arguments)
			filtered = append(filtered, tc)
		} else if tc.Function.Arguments.Len() > 0 {
			logger.WarnContext(ctx, "ollama: dropping tool call with empty name (could not infer)",
				"args", tc.Function.Arguments)
		} else {
			logger.DebugContext(ctx, "ollama: filtering phantom tool call (empty name and args)")
		}
	}

	return filtered
}

// logResponsePreview logs a preview of the response content for debugging.
func logResponsePreview(ctx context.Context, logger *slog.Logger, content string) {
	if content == "" {
		return
	}

	preview := content
	if len(preview) > maxPreviewLen {
		preview = preview[:100]
	}

	logger.DebugContext(ctx, "Ollama Complete response preview", "preview", preview)
}

// Stream performs a streaming completion request using Ollama's native API.
func (p *Provider) Stream(ctx context.Context, params openaisdk.ChatCompletionNewParams) (<-chan openaisdk.ChatCompletionChunk, error) {
	req := p.buildChatRequest(ctx, params, true)

	chunks := make(chan openaisdk.ChatCompletionChunk, 10)

	go func() {
		defer close(chunks)

		chunkID := fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano())
		chunkIndex := 0

		var lastDoneReason string

		err := p.client.Chat(ctx, req, func(resp api.ChatResponse) error {
			if ctx.Err() != nil {
				return ctx.Err()
			}

			// Merge thinking into content.
			resp.Message.Content = mergeThinkingContent(resp.Message.Thinking, resp.Message.Content)
			resp.Message.Thinking = ""

			// Fix tool calls with empty function names.
			resp.Message.ToolCalls = filterToolCalls(ctx, resp.Message.ToolCalls, req.Tools, p.logger)

			if resp.Done && resp.DoneReason != "" {
				lastDoneReason = resp.DoneReason
			}

			chunk := convertOllamaChunkToOpenAI(ctx, resp, chunkID, p.model, p.logger)

			select {
			case chunks <- chunk:
				chunkIndex++
			case <-ctx.Done():
				return ctx.Err()
			}

			return nil
		})

		if err != nil && ctx.Err() == nil {
			p.logger.ErrorContext(ctx, "ollama stream error", "error", err, "chunks_sent", chunkIndex, "done_reason", lastDoneReason)
		}

		p.logger.DebugContext(ctx, "ollama stream finished",
			"total_chunks", chunkIndex,
			"done_reason", lastDoneReason)
	}()

	return chunks, nil
}

// Close closes the provider and releases resources.
func (p *Provider) Close() error {
	// Ollama client doesn't require explicit cleanup.
	return nil
}
