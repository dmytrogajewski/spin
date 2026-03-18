package openai

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"

	"github.com/dmytrogajewski/spin/internal/llm"
)

var (
	// ErrBaseURLIsRequired is a sentinel error.
	ErrBaseURLIsRequired = errors.New("base URL is required")
	// ErrModelIsRequired is a sentinel error.
	ErrModelIsRequired = errors.New("model is required")
	// ErrTimeoutMustBe0 is a sentinel error.
	ErrTimeoutMustBe0 = errors.New("timeout must be > 0")
)

// Config configures the OpenAI provider.
type Config struct {
	BaseURL string
	APIKey  string
	Model   string
	Timeout time.Duration
}

// Validate validates the OpenAI configuration.
func (c *Config) Validate() error {
	if c.BaseURL == "" {
		return ErrBaseURLIsRequired
	}

	if c.Model == "" {
		return ErrModelIsRequired
	}

	if c.Timeout <= 0 {
		return fmt.Errorf("timeout must be > 0, got %v: %w", c.Timeout, ErrTimeoutMustBe0)
	}

	return nil
}

// Provider implements the OpenAI LLM provider using the official SDK.
//
// GOROUTINE LIFECYCLE:
// - Stream() spawns one goroutine per streaming request that:
//
//   - Reads from SDK stream
//
//   - Converts and sends chunks to the returned channel
//
//   - Lives until EOF, context cancellation, or error
//
//   - Automatically cleans up (closes channel)
//
//   - The goroutine terminates when the caller stops reading from the channel
//     or when the context is canceled
//
// CONCURRENCY:
// - All methods are safe to call concurrently
// - Each stream has its own independent goroutine and channel
// - No shared mutable state between concurrent operations.
type Provider struct {
	client  *openai.Client
	model   string
	timeout time.Duration
}

// NewProvider creates a new OpenAI provider using the official SDK.
func NewProvider(cfg Config) (*Provider, error) {
	// Validate configuration.
	err := cfg.Validate()
	if err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// IMPORTANT: Do NOT remove trailing slash from BaseURL
	// The OpenAI SDK uses url.ResolveReference which requires trailing slash
	// to preserve the path component (e.g., /v1/)
	// Without trailing slash: http://host/v1 + "chat/completions" = http://host/chat/completions (WRONG)
	// With trailing slash: http://host/v1/ + "chat/completions" = http://host/v1/chat/completions (CORRECT).
	baseURL := cfg.BaseURL

	// Use timeout from config or default.
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = llm.DefaultTimeout
	}

	// Create SDK client.
	client := openai.NewClient(
		option.WithAPIKey(cfg.APIKey),
		option.WithBaseURL(baseURL),
		option.WithRequestTimeout(timeout),
	)

	return &Provider{
		client:  client,
		model:   cfg.Model,
		timeout: timeout,
	}, nil
}

// Complete performs a synchronous completion request.
func (p *Provider) Complete(ctx context.Context, params openai.ChatCompletionNewParams) (*openai.ChatCompletion, error) {
	// Set model if not specified in request.
	if !params.Model.Present {
		params.Model = openai.F(openai.ChatModel(p.model))
	}

	// Make completion request - returns *ChatCompletion directly.
	resp, err := p.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, mapError(err)
	}

	return resp, nil
}

// Stream performs a streaming completion request.
func (p *Provider) Stream(ctx context.Context, params openai.ChatCompletionNewParams) (<-chan openai.ChatCompletionChunk, error) {
	// Set model if not specified in request.
	if !params.Model.Present {
		params.Model = openai.F(openai.ChatModel(p.model))
	}

	// Create streaming request.
	stream := p.client.Chat.Completions.NewStreaming(ctx, params)

	// Create channel for chunks (buffered to avoid blocking).
	const streamChunkBuffer = 10

	chunks := make(chan openai.ChatCompletionChunk, streamChunkBuffer)

	// Spawn goroutine to read stream and send chunks.
	go func() {
		defer close(chunks)
		defer stream.Close()

		// Read chunks from stream.
		for stream.Next() {
			chunk := stream.Current()

			// Send chunk directly (no conversion needed!)
			select {
			case chunks <- chunk:
			case <-ctx.Done():
				// Context canceled, just exit (channel is closed by defer).
				return
			}
		}

		// Check for stream error
		// Note: With OpenAI SDK, errors should be returned from Stream() call itself,
		// but we check here for completeness. Consumers should check for channel close.
		err := stream.Err()
		if err != nil {
			// Stream errors cannot be sent to consumers via the chunk channel.
			// Log for diagnostics and close the channel.
			slog.Warn("OpenAI stream error", "error", mapError(err))

			return
		}
	}()

	return chunks, nil
}

// Models returns the list of available models.
func (p *Provider) Models(ctx context.Context) ([]openai.Model, error) {
	// Use SDK to list models.
	resp, err := p.client.Models.List(ctx)
	if err != nil {
		return nil, mapError(err)
	}

	var models []openai.Model
	// Add models from first page.
	models = append(models, resp.Data...)

	// Iterate through remaining pages, respecting context cancellation.
	for {
		if err := ctx.Err(); err != nil {
			return models, fmt.Errorf("list models pagination: %w", err)
		}

		nextPage, pageErr := resp.GetNextPage()
		if pageErr != nil {
			return models, fmt.Errorf("get next page: %w", mapError(pageErr))
		}

		if nextPage == nil {
			break
		}

		resp = nextPage
		models = append(models, resp.Data...)
	}

	return models, nil
}

// Capabilities returns the provider's capabilities.
func (p *Provider) Capabilities() llm.Capabilities {
	return llm.Capabilities{
		Streaming:       true,
		FunctionCalling: true,
		Vision:          false, // Not implemented yet.
		ContextWindow:   getModelContextWindow(p.model),
	}
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "openai-compatible"
}

// Close cleans up provider resources.
func (p *Provider) Close() error {
	// SDK client doesn't require explicit cleanup.
	return nil
}

// Context window size constants for known LLM models.
const (
	ctxWindow4K   = 4096
	ctxWindow8K   = 8192
	ctxWindow16K  = 16385
	ctxWindow32K  = 32768
	ctxWindow64K  = 64000
	ctxWindow65K  = 65536
	ctxWindow128K = 128000
	ctxWindow131K = 131072
	ctxWindow200K = 200000
	ctxWindow1M   = 1000000
)

// knownContextWindows maps model names to their context window sizes.
// This is a best-effort lookup - not all models are listed.
var knownContextWindows = map[string]int{
	// OpenAI GPT-4 models.
	"gpt-4":                  ctxWindow8K,
	"gpt-4-32k":              ctxWindow32K,
	"gpt-4-turbo":            ctxWindow128K,
	"gpt-4-turbo-preview":    ctxWindow128K,
	"gpt-4-0125-preview":     ctxWindow128K,
	"gpt-4-1106-preview":     ctxWindow128K,
	"gpt-4o":                 ctxWindow128K,
	"gpt-4o-mini":            ctxWindow128K,
	"gpt-4o-2024-05-13":      ctxWindow128K,
	"gpt-4o-2024-08-06":      ctxWindow128K,
	"gpt-4o-2024-11-20":      ctxWindow128K,
	"gpt-4o-mini-2024-07-18": ctxWindow128K,
	"gpt-4.1":                ctxWindow1M,
	"gpt-4.1-mini":           ctxWindow1M,
	"gpt-4.1-nano":           ctxWindow1M,
	"o1":                     ctxWindow200K,
	"o1-mini":                ctxWindow128K,
	"o1-preview":             ctxWindow128K,
	"o3":                     ctxWindow200K,
	"o3-mini":                ctxWindow200K,
	"o4-mini":                ctxWindow200K,

	// OpenAI GPT-3.5 models.
	"gpt-3.5-turbo":          ctxWindow16K,
	"gpt-3.5-turbo-16k":      ctxWindow16K,
	"gpt-3.5-turbo-0125":     ctxWindow16K,
	"gpt-3.5-turbo-1106":     ctxWindow16K,
	"gpt-3.5-turbo-instruct": ctxWindow4K,

	// Anthropic Claude models (for OpenAI-compatible endpoints).
	"claude-3-opus":            ctxWindow200K,
	"claude-3-opus-20240229":   ctxWindow200K,
	"claude-3-sonnet":          ctxWindow200K,
	"claude-3-sonnet-20240229": ctxWindow200K,
	"claude-3-haiku":           ctxWindow200K,
	"claude-3-haiku-20240307":  ctxWindow200K,
	"claude-3.5-sonnet":        ctxWindow200K,
	"claude-3-5-sonnet":        ctxWindow200K,
	"claude-3.5-haiku":         ctxWindow200K,
	"claude-3-5-haiku":         ctxWindow200K,
	"claude-sonnet-4":          ctxWindow200K,
	"claude-opus-4":            ctxWindow200K,

	// DeepSeek models.
	"deepseek-chat":     ctxWindow64K,
	"deepseek-coder":    ctxWindow64K,
	"deepseek-r1":       ctxWindow64K,
	"deepseek-v3":       ctxWindow64K,
	"deepseek-v2":       ctxWindow128K,
	"deepseek-v2.5":     ctxWindow128K,
	"deepseek-reasoner": ctxWindow64K,

	// Google Gemini models (for OpenAI-compatible endpoints).
	"gemini-pro":       ctxWindow32K,
	"gemini-1.5-pro":   ctxWindow1M,
	"gemini-1.5-flash": ctxWindow1M,
	"gemini-2.0-flash": ctxWindow1M,
	"gemini-2.0-pro":   ctxWindow1M,
	"gemini-2.5-pro":   ctxWindow1M,
	"gemini-2.5-flash": ctxWindow1M,

	// Mistral models.
	"mistral-tiny":       ctxWindow32K,
	"mistral-small":      ctxWindow32K,
	"mistral-medium":     ctxWindow32K,
	"mistral-large":      ctxWindow128K,
	"mistral-nemo":       ctxWindow128K,
	"codestral":          ctxWindow32K,
	"codestral-latest":   ctxWindow32K,
	"open-mistral-7b":    ctxWindow32K,
	"open-mixtral-8x7b":  ctxWindow32K,
	"open-mixtral-8x22b": ctxWindow65K,

	// Groq-hosted models.
	"llama3-8b-8192":     ctxWindow8K,
	"llama3-70b-8192":    ctxWindow8K,
	"llama-3.1-8b":       ctxWindow131K,
	"llama-3.1-70b":      ctxWindow131K,
	"llama-3.1-405b":     ctxWindow131K,
	"llama-3.2-1b":       ctxWindow131K,
	"llama-3.2-3b":       ctxWindow131K,
	"llama-3.2-11b":      ctxWindow131K,
	"llama-3.2-90b":      ctxWindow131K,
	"llama-3.3-70b":      ctxWindow131K,
	"mixtral-8x7b-32768": ctxWindow32K,
	"gemma-7b-it":        ctxWindow8K,
	"gemma2-9b-it":       ctxWindow8K,

	// Qwen models.
	"qwen-turbo":    ctxWindow8K,
	"qwen-plus":     ctxWindow32K,
	"qwen-max":      ctxWindow32K,
	"qwen2-72b":     ctxWindow131K,
	"qwen2.5-72b":   ctxWindow131K,
	"qwen2.5-coder": ctxWindow131K,
	"qwq-32b":       ctxWindow131K,
}

// getModelContextWindow returns the context window size for a model.
// Returns 0 if unknown (callers should use a sensible default).
func getModelContextWindow(model string) int {
	if ctxLen, ok := knownContextWindows[model]; ok {
		return ctxLen
	}

	return 0
}
