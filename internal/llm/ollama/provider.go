// Package ollama provides an Ollama LLM provider implementation.
// Ollama is a tool for running large language models locally with OpenAI-compatible API.
package ollama

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/llm/vram"
	"github.com/ollama/ollama/api"
	openaisdk "github.com/openai/openai-go"
)

// vramNewDetector is a test seam for VRAM detector.
var vramNewDetector = vram.NewDetector

// newRequirementsCalculator is a test seam for VRAM calculator.
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

	// Timeout is the request timeout (defaults to 5 minutes)
	Timeout time.Duration

	// StreamTimeout optionally bounds streaming calls (default: 30m)
	StreamTimeout time.Duration
}

// Provider implements the Ollama LLM provider using the native Ollama API client.
type Provider struct {
	// Ollama SDK client
	client *api.Client

	model   string
	baseURL string
	timeout time.Duration

	// Auto-tune fields
	autoTuneCtxLen    int
	autoTuneGPULayers int
	autoTuneWarning   string
}

// NewProvider creates a new Ollama provider.
func NewProvider(cfg Config) (*Provider, error) {
	if cfg.Model == "" {
		return nil, fmt.Errorf("model is required")
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	// Validate base URL early
	baseURLParsed, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse base URL: %w", err)
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = llm.DefaultTimeout
	}

	// Create Ollama SDK client
	ollamaClient := api.NewClient(baseURLParsed, &http.Client{
		Timeout: timeout,
	})

	return &Provider{
		client:  ollamaClient,
		model:   cfg.Model,
		baseURL: baseURL,
		timeout: timeout,
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
		FunctionCalling: true,  // Ollama supports function calling
		Vision:          false, // Vision not supported yet
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
			Created: 0, // Ollama doesn't provide creation time
			Object:  "model",
		})
	}

	return models, nil
}

// AutoTune automatically configures model parameters based on available VRAM.
// This uses VRAM detection to set optimal num_ctx and num_gpu values.
func (p *Provider) AutoTune(ctx context.Context, headroomBytes int64) error {
	// Get model size using Ollama SDK
	resp, err := p.client.List(ctx)
	if err != nil {
		return fmt.Errorf("list models for auto-tune: %w", err)
	}

	var modelSize int64
	for _, m := range resp.Models {
		if m.Name == p.model {
			modelSize = int64(m.Size)
			break
		}
	}

	if modelSize == 0 {
		// Model not found or size unavailable, skip auto-tune
		return nil
	}

	// Use VRAM calculator to determine optimal settings
	det := vramNewDetector(nil)
	calc := newRequirementsCalculator(det, headroomBytes)

	// Calculate requirements for 4096 context (reasonable default)
	reqs, err := calc.Calculate(modelSize, 4096)
	if err != nil {
		return fmt.Errorf("calculate VRAM requirements: %w", err)
	}

	if reqs != nil {
		if reqs.ContextLength > 0 {
			p.autoTuneCtxLen = reqs.ContextLength
		}
		if reqs.NumGPULayers > 0 {
			p.autoTuneGPULayers = reqs.NumGPULayers
		}

		// Set warning for low-VRAM scenarios
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

// Complete performs a non-streaming completion request using Ollama's native API.
func (p *Provider) Complete(ctx context.Context, params openaisdk.ChatCompletionNewParams) (*openaisdk.ChatCompletion, error) {
	// Convert OpenAI params to Ollama ChatRequest
	req := &api.ChatRequest{
		Model: p.model,
	}

	// Convert messages
	if params.Messages.Present {
		req.Messages = make([]api.Message, len(params.Messages.Value))
		for i, msg := range params.Messages.Value {
			req.Messages[i] = convertMessageToOllama(msg)
		}
	}

	// Convert tools if present
	if params.Tools.Present && len(params.Tools.Value) > 0 {
		req.Tools = make([]api.Tool, len(params.Tools.Value))
		for i, tool := range params.Tools.Value {
			req.Tools[i] = convertToolToOllama(tool)
		}
	}

	// Set options
	if params.Temperature.Present {
		if req.Options == nil {
			req.Options = make(map[string]interface{})
		}
		req.Options["temperature"] = params.Temperature.Value
	}
	if params.MaxTokens.Present {
		if req.Options == nil {
			req.Options = make(map[string]interface{})
		}
		req.Options["num_predict"] = params.MaxTokens.Value
	}

	// Call Ollama API
	// Note: Ollama sends multiple callbacks even for non-streaming requests
	// We need to accumulate the content from all callbacks
	var resp api.ChatResponse
	var fullContent strings.Builder
	callbackCount := 0

	err := p.client.Chat(ctx, req, func(r api.ChatResponse) error {
		callbackCount++
		resp = r // Keep the last response for metadata
		// Accumulate content from all callbacks
		if r.Message.Content != "" {
			fullContent.WriteString(r.Message.Content)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("ollama chat: %w", err)
	}

	// Use accumulated content
	resp.Message.Content = fullContent.String()

	// Debug: Log the Ollama response
	slog.Debug("Ollama Complete", "callbacks", callbackCount, "content_length", len(resp.Message.Content))
	if len(resp.Message.Content) > 0 {
		preview := resp.Message.Content
		if len(preview) > 100 {
			preview = preview[:100]
		}
		slog.Debug("Ollama Complete response preview", "preview", preview)
	}

	// Convert response to OpenAI format
	return convertOllamaResponseToOpenAI(resp, p.model), nil
}

// Stream performs a streaming completion request using Ollama's native API.
func (p *Provider) Stream(ctx context.Context, params openaisdk.ChatCompletionNewParams) (<-chan openaisdk.ChatCompletionChunk, error) {
	// Convert OpenAI params to Ollama ChatRequest
	req := &api.ChatRequest{
		Model:  p.model,
		Stream: new(bool),
	}
	*req.Stream = true

	// Convert messages
	if params.Messages.Present {
		req.Messages = make([]api.Message, len(params.Messages.Value))
		for i, msg := range params.Messages.Value {
			req.Messages[i] = convertMessageToOllama(msg)
		}
	}

	// Convert tools if present
	if params.Tools.Present && len(params.Tools.Value) > 0 {
		req.Tools = make([]api.Tool, len(params.Tools.Value))
		for i, tool := range params.Tools.Value {
			req.Tools[i] = convertToolToOllama(tool)
		}
	}

	// Set options
	if params.Temperature.Present {
		if req.Options == nil {
			req.Options = make(map[string]interface{})
		}
		req.Options["temperature"] = params.Temperature.Value
	}
	if params.MaxTokens.Present {
		if req.Options == nil {
			req.Options = make(map[string]interface{})
		}
		req.Options["num_predict"] = params.MaxTokens.Value
	}

	// Create channel for chunks
	chunks := make(chan openaisdk.ChatCompletionChunk, 10)

	// Start streaming in background
	go func() {
		defer close(chunks)

		chunkID := fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano())
		chunkIndex := 0
		var lastDoneReason string

		err := p.client.Chat(ctx, req, func(resp api.ChatResponse) error {
			// Check context cancellation
			if ctx.Err() != nil {
				return ctx.Err()
			}

			// Debug: Log raw Ollama response for diagnostics
			slog.Debug("ollama stream chunk",
				"index", chunkIndex,
				"content_len", len(resp.Message.Content),
				"tool_calls", len(resp.Message.ToolCalls),
				"done", resp.Done,
				"done_reason", resp.DoneReason)

			// Track done reason for final chunk handling
			if resp.Done && resp.DoneReason != "" {
				lastDoneReason = resp.DoneReason
			}

			// Convert to OpenAI chunk and send
			chunk := convertOllamaChunkToOpenAI(resp, chunkID, p.model)

			select {
			case chunks <- chunk:
				chunkIndex++
			case <-ctx.Done():
				return ctx.Err()
			}

			return nil
		})

		// Handle error - send an error indicator chunk if possible
		if err != nil && ctx.Err() == nil {
			slog.Error("ollama stream error", "error", err, "chunks_sent", chunkIndex, "done_reason", lastDoneReason)

			// If we got zero chunks, this is a connection/API error
			// Send an error chunk so the caller knows something went wrong
			if chunkIndex == 0 {
				errorChunk := openaisdk.ChatCompletionChunk{
					ID:      chunkID,
					Created: time.Now().Unix(),
					Model:   p.model,
					Object:  "chat.completion.chunk",
					Choices: []openaisdk.ChatCompletionChunkChoice{
						{
							Index: 0,
							Delta: openaisdk.ChatCompletionChunkChoicesDelta{
								Role:    openaisdk.ChatCompletionChunkChoicesDeltaRoleAssistant,
								Content: fmt.Sprintf("[Error: %v]", err),
							},
							FinishReason: openaisdk.ChatCompletionChunkChoicesFinishReasonStop,
						},
					},
				}
				select {
				case chunks <- errorChunk:
				default:
					// Channel full or closed, can't send error
				}
			}
		}

		slog.Debug("ollama stream finished", "total_chunks", chunkIndex, "done_reason", lastDoneReason)
	}()

	return chunks, nil
}

// Close closes the provider and releases resources.
func (p *Provider) Close() error {
	// Ollama client doesn't require explicit cleanup
	return nil
}
