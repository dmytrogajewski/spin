// Package ollama provides an Ollama LLM provider implementation.
// Ollama is a tool for running large language models locally with OpenAI-compatible API.
package ollama

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ollama/ollama/api"
	openaisdk "github.com/openai/openai-go"

	"github.com/dmytrogajewski/spin/internal/llm"
)

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

	// Context length detected from model metadata.
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

	// Detect context length on first call.
	if p.detectedCtxLen == 0 {
		p.detectedCtxLen = p.detectContextLength(ctx)
	}

	opts["num_ctx"] = p.detectedCtxLen

	return opts
}

// Complete performs a non-streaming completion request using Ollama's native API.
func (p *Provider) Complete(ctx context.Context, params openaisdk.ChatCompletionNewParams) (*openaisdk.ChatCompletion, error) {
	// Convert OpenAI params to Ollama ChatRequest.
	req := &api.ChatRequest{
		Model: p.model,
	}

	// Convert messages, building tool_call_id -> tool_name mapping for tool result messages.
	if params.Messages.Present {
		toolCallIDToName := buildToolCallIDToNameMap(params.Messages.Value, p.logger, ctx)

		req.Messages = make([]api.Message, len(params.Messages.Value))
		for i, msg := range params.Messages.Value {
			req.Messages[i] = convertMessageToOllama(msg, toolCallIDToName, p.logger, ctx)
		}
		// Debug: log the messages being sent.
		for i, m := range req.Messages {
			p.logger.DebugContext(ctx, "ollama stream message",
				"index", i,
				"role", m.Role,
				"content_len", len(m.Content),
				"tool_calls", len(m.ToolCalls),
				"tool_name", m.ToolName)
		}
	}

	// Convert tools if present.
	if params.Tools.Present && len(params.Tools.Value) > 0 {
		req.Tools = make([]api.Tool, len(params.Tools.Value))
		for i, tool := range params.Tools.Value {
			req.Tools[i] = convertToolToOllama(tool)
		}

		p.logger.DebugContext(ctx, "ollama request with tools", "tool_count", len(req.Tools), "model", p.model)
	}

	// Set options.
	if params.Temperature.Present {
		if req.Options == nil {
			req.Options = make(map[string]any)
		}

		req.Options["temperature"] = params.Temperature.Value
	}

	if params.MaxTokens.Present {
		if req.Options == nil {
			req.Options = make(map[string]any)
		}

		req.Options["num_predict"] = params.MaxTokens.Value
	}

	// Set context window size.
	req.Options = p.setContextOptions(ctx, req.Options)

	// Call Ollama API
	// Note: Ollama sends multiple callbacks even for non-streaming requests
	// We need to accumulate the content from all callbacks.
	var (
		resp         api.ChatResponse
		fullContent  strings.Builder
		fullThinking strings.Builder
	)

	callbackCount := 0

	err := p.client.Chat(ctx, req, func(r api.ChatResponse) error {
		callbackCount++
		resp = r // Keep the last response for metadata.
		// Accumulate content and thinking from all callbacks.
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
	if fullThinking.Len() > 0 {
		resp.Message.Content = "<think>" + fullThinking.String() + "</think>" + fullContent.String()
	} else {
		resp.Message.Content = fullContent.String()
	}

	resp.Message.Thinking = ""

	// Fix tool calls with empty function names by inferring from arguments (same as streaming path).
	if len(resp.Message.ToolCalls) > 0 {
		filtered := resp.Message.ToolCalls[:0]
		for _, tc := range resp.Message.ToolCalls {
			if tc.Function.Name != "" {
				filtered = append(filtered, tc)
			} else if inferred := inferToolName(tc.Function.Arguments, req.Tools, p.logger, ctx); inferred != "" {
				tc.Function.Name = inferred
				p.logger.InfoContext(ctx, "ollama: inferred tool name for nameless tool call",
					"name", inferred, "args", tc.Function.Arguments)
				filtered = append(filtered, tc)
			} else if len(tc.Function.Arguments) > 0 {
				p.logger.WarnContext(ctx, "ollama: dropping tool call with empty name (could not infer)",
					"args", tc.Function.Arguments)
			} else {
				p.logger.DebugContext(ctx, "ollama: filtering phantom tool call (empty name and args)")
			}
		}

		resp.Message.ToolCalls = filtered
	}

	// Debug: Log the Ollama response.
	p.logger.DebugContext(ctx, "Ollama Complete", "callbacks", callbackCount, "content_length", len(resp.Message.Content))

	if len(resp.Message.Content) > 0 {
		preview := resp.Message.Content
		if len(preview) > 100 {
			preview = preview[:100]
		}

		p.logger.DebugContext(ctx, "Ollama Complete response preview", "preview", preview)
	}

	// Convert response to OpenAI format.
	return convertOllamaResponseToOpenAI(resp, p.model, p.logger, ctx), nil
}

// Stream performs a streaming completion request using Ollama's native API.
func (p *Provider) Stream(ctx context.Context, params openaisdk.ChatCompletionNewParams) (<-chan openaisdk.ChatCompletionChunk, error) {
	// Convert OpenAI params to Ollama ChatRequest.
	req := &api.ChatRequest{
		Model:  p.model,
		Stream: new(bool),
	}
	*req.Stream = true

	// Convert messages, building tool_call_id -> tool_name mapping for tool result messages.
	if params.Messages.Present {
		toolCallIDToName := buildToolCallIDToNameMap(params.Messages.Value, p.logger, ctx)

		req.Messages = make([]api.Message, len(params.Messages.Value))
		for i, msg := range params.Messages.Value {
			req.Messages[i] = convertMessageToOllama(msg, toolCallIDToName, p.logger, ctx)
		}
		// Debug: log the messages being sent to Ollama.
		for i, m := range req.Messages {
			tcIDs := make([]string, len(m.ToolCalls))
			for j, tc := range m.ToolCalls {
				tcIDs[j] = tc.Function.Name
			}

			p.logger.DebugContext(ctx, "ollama stream msg",
				"idx", i,
				"role", m.Role,
				"content_len", len(m.Content),
				"tool_calls", tcIDs,
				"tool_name", m.ToolName)
		}
	}

	// Convert tools if present.
	if params.Tools.Present && len(params.Tools.Value) > 0 {
		req.Tools = make([]api.Tool, len(params.Tools.Value))
		for i, tool := range params.Tools.Value {
			req.Tools[i] = convertToolToOllama(tool)
		}

		p.logger.DebugContext(ctx, "ollama stream request with tools", "tool_count", len(req.Tools), "model", p.model)
	}

	// Set options.
	if params.Temperature.Present {
		if req.Options == nil {
			req.Options = make(map[string]any)
		}

		req.Options["temperature"] = params.Temperature.Value
	}

	if params.MaxTokens.Present {
		if req.Options == nil {
			req.Options = make(map[string]any)
		}

		req.Options["num_predict"] = params.MaxTokens.Value
	}

	// Set context window size.
	req.Options = p.setContextOptions(ctx, req.Options)

	// Create channel for chunks.
	chunks := make(chan openaisdk.ChatCompletionChunk, 10)

	// Start streaming in background.
	go func() {
		defer close(chunks)

		chunkID := fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano())
		chunkIndex := 0

		var lastDoneReason string

		err := p.client.Chat(ctx, req, func(resp api.ChatResponse) error {
			// Check context cancellation.
			if ctx.Err() != nil {
				return ctx.Err()
			}

			// Merge Ollama's separate Thinking field into Content as <think> tags.
			// Thinking models (qwen3, kimi2.5, etc.) put reasoning in Message.Thinking
			// and only the final answer in Message.Content. Our sanitizer already handles
			// <think> tags, so wrapping and prepending unifies the pipeline.
			if resp.Message.Thinking != "" {
				resp.Message.Content = "<think>" + resp.Message.Thinking + "</think>" + resp.Message.Content
				resp.Message.Thinking = ""
			}

			// Fix tool calls with empty function names by inferring from arguments.
			// Some models emit tool calls with valid arguments but no function name.
			// Only filter truly phantom calls (empty name AND empty arguments).
			if len(resp.Message.ToolCalls) > 0 {
				filtered := resp.Message.ToolCalls[:0]
				for _, tc := range resp.Message.ToolCalls {
					if tc.Function.Name != "" {
						filtered = append(filtered, tc)
					} else if inferred := inferToolName(tc.Function.Arguments, req.Tools, p.logger, ctx); inferred != "" {
						tc.Function.Name = inferred
						p.logger.InfoContext(ctx, "ollama: inferred tool name for nameless tool call",
							"name", inferred, "args", tc.Function.Arguments)
						filtered = append(filtered, tc)
					} else if len(tc.Function.Arguments) > 0 {
						p.logger.WarnContext(ctx, "ollama: dropping tool call with empty name (could not infer)",
							"chunk_index", chunkIndex,
							"args", tc.Function.Arguments)
					} else {
						p.logger.DebugContext(ctx, "ollama: filtering phantom tool call (empty name and args)",
							"chunk_index", chunkIndex)
					}
				}

				resp.Message.ToolCalls = filtered
			}

			// Track done reason for final chunk handling.
			if resp.Done && resp.DoneReason != "" {
				lastDoneReason = resp.DoneReason
			}

			// Convert to OpenAI chunk and send.
			chunk := convertOllamaChunkToOpenAI(resp, chunkID, p.model, p.logger, ctx)

			select {
			case chunks <- chunk:
				chunkIndex++
			case <-ctx.Done():
				return ctx.Err()
			}

			return nil
		})

		// Handle error - log it and let the channel close with zero chunks.
		// The caller (callLLM) detects 0-chunk streams and returns an error,
		// which allows the retry loop to handle transient failures (e.g. HTTP 500).
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
