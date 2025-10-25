// Package ollama provides an Ollama LLM provider implementation.
// Ollama is a tool for running large language models locally with a simple REST API.
package ollama

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/llm/vram"
)

// vramNewDetector is a test seam for creating a VRAM detector.
// It defaults to vram.NewDetector, but tests may override it.
var vramNewDetector = vram.NewDetector

// newRequirementsCalculator is a test seam for creating a VRAM requirements calculator.
var newRequirementsCalculator = vram.NewRequirementsCalculator

const (
	// DefaultBaseURL is the default Ollama API endpoint.
	DefaultBaseURL = "http://localhost:11434"
)

// Config configures the Ollama provider.
type Config struct {
	// BaseURL is the API endpoint URL (default: http://localhost:11434)
	BaseURL string

	// Model is the model name to use (required)
	Model string

	// Timeout is the request timeout (defaults to 5 minutes, used for non-streaming).
	Timeout time.Duration

	// StreamTimeout optionally bounds a streaming call via context (default: 30m).
	// Set 0 to keep the inherited context deadline only.
	StreamTimeout time.Duration
}

// Provider implements the Ollama LLM provider.
//
// GOROUTINE LIFECYCLE:
// - Stream() spawns one goroutine per streaming request that:
//   - Reads from HTTP response body line-by-line
//   - Parses JSON chunks and sends to the returned channel
//   - Lives until EOF, context cancellation, or error
//   - Automatically cleans up (closes channel and response body)
//
// - PullModel() spawns one goroutine to track pull progress that:
//   - Polls /api/ps endpoint to check if model is available
//   - Lives until model is available or context timeout
//   - Terminates when pull completes or context is cancelled
//
// CONCURRENCY:
// - Stream() and PullModel() are safe to call concurrently
// - Each operation has its own independent goroutine and channel
// - No shared state between concurrent operations
type Provider struct {
	// client is for non-streaming calls with retries/timeouts.
	client *llm.HTTPClient
	// streamClient is for streaming calls — no hard client timeout, ctx controls lifetime.
	streamClient *llm.HTTPClient

	baseURL       string
	model         string
	timeout       time.Duration
	streamTimeout time.Duration

	// errorMapper provides standardized error handling
	errorMapper *llm.ErrorMapper

	// auto-tune fields (optional)
	autoTuneCtxLen    int
	autoTuneGPULayers int

	// autoTuneWarning holds a human-readable note when auto-tune had to degrade settings
	autoTuneWarning string
}

// NewProvider creates a new Ollama provider with automatic retry logic.
func NewProvider(cfg Config) (*Provider, error) {
	if cfg.Model == "" {
		return nil, fmt.Errorf("model is required")
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	baseURL = strings.TrimSuffix(baseURL, "/")

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 5 * time.Minute
	}

	streamTimeout := cfg.StreamTimeout
	if streamTimeout == 0 {
		streamTimeout = 30 * time.Minute
	}

	// Regular (non-streaming) client: retries + deadline
	client := llm.NewHTTPClient(
		llm.WithTimeout(timeout),
		llm.WithMaxRetries(3),
		llm.WithRetryDelay(time.Second),
	)

	// Streaming client: no global timeout, no retries
	streamClient := llm.NewHTTPClient(
		llm.WithTimeout(0),
		llm.WithMaxRetries(0),
		llm.WithRetryDelay(0),
	)

	// Create error mapper for standardized error handling
	errorMapper := llm.NewErrorMapper("ollama")

	return &Provider{
		client:        client,
		streamClient:  streamClient,
		baseURL:       baseURL,
		model:         cfg.Model,
		timeout:       timeout,
		streamTimeout: streamTimeout,
		errorMapper:   errorMapper,
	}, nil
}

// AutoTune queries model info and hardware to set optimal runtime options.
// This is optional; callers may ignore errors and proceed.
func (p *Provider) AutoTune(ctx context.Context, headroomBytes int64) error {
	// Get model metadata for size via /api/tags (first matching)
	req, err := p.newRequest(ctx, http.MethodGet, "/api/tags", nil)
	if err != nil {
		return err
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return p.handleError(resp)
	}
	var tags tagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return err
	}
	var size int64
	modelName := p.model
	for _, m := range tags.Models {
		if m.Name == modelName {
			size = m.Size
			break
		}
	}
	if size == 0 {
		// best-effort: cannot tune without size
		return nil
	}

	det := vramNewDetector(nil)
	calc := newRequirementsCalculator(det, headroomBytes)
	reqs, err := calc.Calculate(size, 4096)
	if err != nil {
		return err
	}
	if reqs != nil {
		if reqs.ContextLength > 0 {
			p.autoTuneCtxLen = reqs.ContextLength
		}
		if reqs.NumGPULayers > 0 {
			p.autoTuneGPULayers = reqs.NumGPULayers
		}
		// Heuristic warnings for low-VRAM fallbacks
		if reqs.Quantization == "q4_0" && reqs.ContextLength == 2048 && reqs.NumGPULayers == 16 {
			p.autoTuneWarning = "VRAM low: applied minimal context and partial GPU layers; quality may be reduced"
		}
		if reqs.RecommendedVRAM == 0 {
			if name, _ := det.GPUName(); name == "cpu" {
				p.autoTuneWarning = "No GPU VRAM detected; CPU-only fallback in effect"
			}
		}
	}
	return nil
}

// GetAutoTuneWarning returns the last auto-tune warning message, if any.
func (p *Provider) GetAutoTuneWarning() string {
	return p.autoTuneWarning
}

// Complete performs a synchronous completion request.
func (p *Provider) Complete(ctx context.Context, req llm.CompletionRequest) (*llm.CompletionResponse, error) {
	prompt := p.buildPrompt(req.Messages)

	genReq := generateRequest{
		Model:       p.getModel(req.Model),
		Prompt:      prompt,
		Stream:      false,
		Temperature: req.Temperature,
	}

	// Apply auto-tuned options if present
	if p.autoTuneCtxLen > 0 {
		if genReq.Options == nil {
			genReq.Options = make(map[string]interface{})
		}
		genReq.Options["num_ctx"] = p.autoTuneCtxLen
	}

	httpReq, err := p.newRequest(ctx, http.MethodPost, "/api/generate", genReq)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, p.handleError(resp)
	}

	var genResp generateResponse
	if err := json.NewDecoder(resp.Body).Decode(&genResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return p.convertResponse(&genResp), nil
}

// Stream performs a streaming completion request.
func (p *Provider) Stream(ctx context.Context, req llm.CompletionRequest) (<-chan llm.StreamChunk, error) {
	if len(req.Tools) > 0 {
		return p.streamChat(ctx, req)
	}

	prompt := p.buildPrompt(req.Messages)
	genReq := generateRequest{
		Model:       p.getModel(req.Model),
		Prompt:      prompt,
		Stream:      true,
		Temperature: req.Temperature,
	}

	if p.streamTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, p.streamTimeout)
		defer func() { _ = cancel }()
	}

	httpReq, err := p.newRequest(ctx, http.MethodPost, "/api/generate", genReq)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := p.streamClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, p.handleError(resp)
	}

	chunks := make(chan llm.StreamChunk, 10)

	go func() {
		defer close(chunks)
		defer resp.Body.Close()

		if err := p.streamResponse(ctx, resp.Body, chunks); err != nil {
			chunks <- llm.StreamChunk{Type: llm.ChunkTypeError, Error: err}
		}
	}()

	return chunks, nil
}

// Models returns the list of available models.
func (p *Provider) Models(ctx context.Context) ([]llm.Model, error) {
	req, err := p.newRequest(ctx, http.MethodGet, "/api/tags", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, p.handleError(resp)
	}

	var tagsResp tagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&tagsResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	models := make([]llm.Model, len(tagsResp.Models))
	for i, m := range tagsResp.Models {
		models[i] = llm.Model{ID: m.Name, Name: m.Name}
	}

	return models, nil
}

// Capabilities returns the provider's capabilities.
func (p *Provider) Capabilities() llm.Capabilities {
	return llm.Capabilities{
		Streaming:       true,
		FunctionCalling: true, // via /api/chat
		Vision:          false,
	}
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "ollama"
}

// Close closes the provider and releases resources.
func (p *Provider) Close() error {
	return nil
}

// buildPrompt converts structured messages to Ollama's text prompt format.
func (p *Provider) buildPrompt(messages []llm.Message) string {
	var parts []string

	for _, msg := range messages {
		role := "User"
		switch msg.Role {
		case "system":
			role = "System"
		case "user":
			role = "User"
		case "assistant":
			role = "Assistant"
		case "tool":
			role = "Tool"
		}
		if msg.Content != "" {
			parts = append(parts, fmt.Sprintf("%s: %s", role, msg.Content))
		}
	}

	prompt := strings.Join(parts, "\n\n")
	if prompt != "" {
		prompt += "\n\n"
	}
	prompt += "Assistant:"
	return prompt
}

// convertResponse converts Ollama response to common format.
func (p *Provider) convertResponse(resp *generateResponse) *llm.CompletionResponse {
	result := &llm.CompletionResponse{
		Model:   resp.Model,
		Content: resp.Response,
		Usage: llm.Usage{
			PromptTokens:     resp.PromptEvalCount,
			CompletionTokens: resp.EvalCount,
			TotalTokens:      resp.PromptEvalCount + resp.EvalCount,
		},
	}
	if resp.Done {
		result.FinishReason = "stop"
	}
	return result
}

// streamResponse processes streaming response for /api/generate.
func (p *Provider) streamResponse(ctx context.Context, r io.Reader, out chan<- llm.StreamChunk) error {
	// Reader avoids bufio.Scanner's 64 KiB token limit.
	br := bufio.NewReaderSize(r, 256*1024)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line, err := br.ReadBytes('\n')
		if len(bytes.TrimSpace(line)) > 0 {
			var chunk generateResponse
			if uErr := json.Unmarshal(line, &chunk); uErr == nil {
				if chunk.Response != "" {
					out <- llm.StreamChunk{Type: llm.ChunkTypeContentDelta, Content: chunk.Response}
				}
				if chunk.Done {
					out <- llm.StreamChunk{Type: llm.ChunkTypeDone, FinishReason: "stop"}
					return nil
				}
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
	}
}

// newRequest creates an HTTP request.
func (p *Provider) newRequest(ctx context.Context, method, path string, body interface{}) (*http.Request, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, p.baseURL+path, bodyReader)
	if err != nil {
		return nil, err
	}

	// Headers
	req.Header.Set("Content-Type", "application/json")
	// Be explicit for streaming endpoints; harmless for non-stream too.
	req.Header.Set("Accept", "application/x-ndjson")
	req.Header.Set("Connection", "keep-alive")

	return req, nil
}

// handleError parses and returns an error from HTTP response.
func (p *Provider) handleError(resp *http.Response) error {
	var errResp errorResponse

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("http %d: failed to read error response", resp.StatusCode)
	}

	if err := json.Unmarshal(body, &errResp); err != nil || errResp.Error == "" {
		return fmt.Errorf("http %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	switch resp.StatusCode {
	case http.StatusNotFound:
		return fmt.Errorf("%w: %s", llm.ErrModelNotFound, errResp.Error)
	default:
		return fmt.Errorf("http %d: %s", resp.StatusCode, errResp.Error)
	}
}

// getModel returns the model to use for the request.
func (p *Provider) getModel(model string) string {
	if model != "" {
		return model
	}
	return p.model
}

// streamChat performs a streaming chat request (with tool support).
func (p *Provider) streamChat(ctx context.Context, req llm.CompletionRequest) (<-chan llm.StreamChunk, error) {
	// Convert messages
	chatMessages := make([]chatMessage, 0, len(req.Messages))
	for _, msg := range req.Messages {
		chatMsg := chatMessage{
			Role:       msg.Role,
			Content:    msg.Content,
			ToolCallID: msg.ToolCallID,
		}

		if len(msg.ToolCalls) > 0 {
			chatMsg.ToolCalls = make([]chatToolCall, len(msg.ToolCalls))
			for j, tc := range msg.ToolCalls {
				var args interface{}
				if tc.Function.Arguments != "" {
					var m map[string]interface{}
					if err := json.Unmarshal([]byte(tc.Function.Arguments), &m); err == nil {
						args = m
					} else {
						args = tc.Function.Arguments
					}
				} else {
					args = map[string]interface{}{}
				}

				chatMsg.ToolCalls[j] = chatToolCall{
					ID:   tc.ID,
					Type: tc.Type,
					Function: chatToolCallFunction{
						Name:      tc.Function.Name,
						Arguments: args,
					},
				}
			}
		}

		chatMessages = append(chatMessages, chatMsg)
	}

	// Tools
	var chatTools []chatTool
	if len(req.Tools) > 0 {
		chatTools = make([]chatTool, len(req.Tools))
		for i, tool := range req.Tools {
			params := make(map[string]interface{})
			if tool.Function.Parameters != nil {
				if p, ok := tool.Function.Parameters.(map[string]interface{}); ok {
					params = p
				}
			}
			chatTools[i] = chatTool{
				Type: tool.Type,
				Function: chatToolFunction{
					Name:        tool.Function.Name,
					Description: tool.Function.Description,
					Parameters:  params,
				},
			}
		}
	}

	chatReq := chatRequest{
		Model:    p.getModel(req.Model),
		Messages: chatMessages,
		Tools:    chatTools,
		Stream:   true,
	}

	if req.Temperature > 0 {
		chatReq.Options = &chatOptions{Temperature: req.Temperature}
	}
	if req.MaxTokens > 0 {
		if chatReq.Options == nil {
			chatReq.Options = &chatOptions{}
		}
		chatReq.Options.NumPredict = req.MaxTokens
	}
	// Apply auto-tuned options if present
	if p.autoTuneCtxLen > 0 {
		if chatReq.Options == nil {
			chatReq.Options = &chatOptions{}
		}
		chatReq.Options.NumCtx = p.autoTuneCtxLen
	}
	if p.autoTuneGPULayers > 0 {
		if chatReq.Options == nil {
			chatReq.Options = &chatOptions{}
		}
		chatReq.Options.NumGPU = p.autoTuneGPULayers
	}

	slog.Debug("Ollama sending chat request", "messages", len(chatMessages), "tools", len(chatTools))
	for i, msg := range chatMessages {
		slog.Debug("Ollama message", "index", i, "role", msg.Role, "content_len", len(msg.Content), "tool_calls", len(msg.ToolCalls), "tool_call_id", msg.ToolCallID)
	}

	if p.streamTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, p.streamTimeout)
		defer func() { _ = cancel }()
	}

	httpReq, err := p.newRequest(ctx, http.MethodPost, "/api/chat", chatReq)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := p.streamClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, p.handleError(resp)
	}

	chunks := make(chan llm.StreamChunk, 10)

	go func() {
		defer close(chunks)
		defer resp.Body.Close()

		if err := p.streamChatResponse(ctx, resp.Body, chunks); err != nil {
			chunks <- llm.StreamChunk{Type: llm.ChunkTypeError, Error: err}
		}
	}()

	return chunks, nil
}

// streamChatResponse processes streaming chat response (with tool calls).
func (p *Provider) streamChatResponse(ctx context.Context, r io.Reader, out chan<- llm.StreamChunk) error {
	br := bufio.NewReaderSize(r, 256*1024)
	lineCount := 0

	slog.Debug("Ollama starting chat stream processing")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line, err := br.ReadBytes('\n')
		if len(bytes.TrimSpace(line)) > 0 {
			lineCount++
			var chunk chatResponse
			if uErr := json.Unmarshal(line, &chunk); uErr != nil {
				slog.Debug("Ollama failed to parse chunk", "line", lineCount, "error", uErr)
			} else {
				slog.Debug("Ollama chunk received", "line", lineCount, "content_len", len(chunk.Message.Content), "tool_calls", len(chunk.Message.ToolCalls), "done", chunk.Done)

				// Some builds put deltas in response (legacy), others in message.content (current)
				delta := chunk.Message.Content
				if delta == "" {
					delta = chunk.Response
				}
				if delta != "" {
					out <- llm.StreamChunk{Type: llm.ChunkTypeContentDelta, Content: delta}
				}

				if len(chunk.Message.ToolCalls) > 0 {
					slog.Debug("Ollama processing tool calls", "count", len(chunk.Message.ToolCalls))
					for i, tc := range chunk.Message.ToolCalls {
						id := tc.ID
						if id == "" {
							id = fmt.Sprintf("call_%d", i)
						}
						tType := tc.Type
						if tType == "" {
							tType = "function"
						}

						var argsStr string
						switch args := tc.Function.Arguments.(type) {
						case string:
							argsStr = args
						case map[string]interface{}, []interface{}:
							if b, err := json.Marshal(args); err == nil {
								argsStr = string(b)
							} else {
								argsStr = "{}"
							}
						default:
							argsStr = "{}"
						}

						out <- llm.StreamChunk{
							Type: llm.ChunkTypeToolCallStart,
							ToolCall: &llm.ToolCall{
								ID:   id,
								Type: tType,
								Function: llm.FunctionCall{
									Name:      tc.Function.Name,
									Arguments: argsStr,
								},
							},
						}
					}
				}

				if chunk.Done {
					slog.Debug("Ollama received done chunk")
					out <- llm.StreamChunk{Type: llm.ChunkTypeDone}
					return nil
				}
			}
		}

		if err != nil {
			if errors.Is(err, io.EOF) {
				slog.Debug("Ollama stream processing ended", "total_lines", lineCount)
				// Send done chunk if we haven't already
				out <- llm.StreamChunk{Type: llm.ChunkTypeDone}
				return nil
			}
			slog.Debug("Ollama stream error", "error", err)
			return err
		}
	}
}
