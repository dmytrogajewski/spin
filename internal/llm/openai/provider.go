// Package openai provides an OpenAI LLM provider implementation using the official openai-go SDK.
package openai

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"

	"github.com/dmytrogajewski/spin/internal/llm"
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
		return errors.New("base URL is required")
	}

	if c.Model == "" {
		return errors.New("model is required")
	}

	if c.Timeout <= 0 {
		return fmt.Errorf("timeout must be > 0, got %v", c.Timeout)
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
	chunks := make(chan openai.ChatCompletionChunk, 10)

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
			// We can't send errors in chunks anymore since we removed the abstraction
			// Errors must be handled differently by consumers
			// Just log and close the channel.
			_ = mapError(err)

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
	for _, sdkModel := range resp.Data {
		models = append(models, sdkModel)
	}

	// Iterate through remaining pages.
	for {
		nextPage, err := resp.GetNextPage()
		if err != nil || nextPage == nil {
			break
		}

		resp = nextPage
		for _, sdkModel := range resp.Data {
			models = append(models, sdkModel)
		}
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

// getModelContextWindow returns the context window size for a model.
// Returns 0 if unknown (callers should use a sensible default).
func getModelContextWindow(model string) int {
	// Known context windows for popular models.
	// This is a best-effort lookup - not all models are listed.
	contextWindows := map[string]int{
		// OpenAI GPT-4 models.
		"gpt-4":                  8192,
		"gpt-4-32k":              32768,
		"gpt-4-turbo":            128000,
		"gpt-4-turbo-preview":    128000,
		"gpt-4-0125-preview":     128000,
		"gpt-4-1106-preview":     128000,
		"gpt-4o":                 128000,
		"gpt-4o-mini":            128000,
		"gpt-4o-2024-05-13":      128000,
		"gpt-4o-2024-08-06":      128000,
		"gpt-4o-2024-11-20":      128000,
		"gpt-4o-mini-2024-07-18": 128000,
		"gpt-4.1":                1000000,
		"gpt-4.1-mini":           1000000,
		"gpt-4.1-nano":           1000000,
		"o1":                     200000,
		"o1-mini":                128000,
		"o1-preview":             128000,
		"o3":                     200000,
		"o3-mini":                200000,
		"o4-mini":                200000,

		// OpenAI GPT-3.5 models.
		"gpt-3.5-turbo":          16385,
		"gpt-3.5-turbo-16k":      16385,
		"gpt-3.5-turbo-0125":     16385,
		"gpt-3.5-turbo-1106":     16385,
		"gpt-3.5-turbo-instruct": 4096,

		// Anthropic Claude models (for OpenAI-compatible endpoints).
		"claude-3-opus":            200000,
		"claude-3-opus-20240229":   200000,
		"claude-3-sonnet":          200000,
		"claude-3-sonnet-20240229": 200000,
		"claude-3-haiku":           200000,
		"claude-3-haiku-20240307":  200000,
		"claude-3.5-sonnet":        200000,
		"claude-3-5-sonnet":        200000,
		"claude-3.5-haiku":         200000,
		"claude-3-5-haiku":         200000,
		"claude-sonnet-4":          200000,
		"claude-opus-4":            200000,

		// DeepSeek models.
		"deepseek-chat":     64000,
		"deepseek-coder":    64000,
		"deepseek-r1":       64000,
		"deepseek-v3":       64000,
		"deepseek-v2":       128000,
		"deepseek-v2.5":     128000,
		"deepseek-reasoner": 64000,

		// Google Gemini models (for OpenAI-compatible endpoints).
		"gemini-pro":       32768,
		"gemini-1.5-pro":   1000000,
		"gemini-1.5-flash": 1000000,
		"gemini-2.0-flash": 1000000,
		"gemini-2.0-pro":   1000000,
		"gemini-2.5-pro":   1000000,
		"gemini-2.5-flash": 1000000,

		// Mistral models.
		"mistral-tiny":       32768,
		"mistral-small":      32768,
		"mistral-medium":     32768,
		"mistral-large":      128000,
		"mistral-nemo":       128000,
		"codestral":          32768,
		"codestral-latest":   32768,
		"open-mistral-7b":    32768,
		"open-mixtral-8x7b":  32768,
		"open-mixtral-8x22b": 65536,

		// Groq-hosted models.
		"llama3-8b-8192":     8192,
		"llama3-70b-8192":    8192,
		"llama-3.1-8b":       131072,
		"llama-3.1-70b":      131072,
		"llama-3.1-405b":     131072,
		"llama-3.2-1b":       131072,
		"llama-3.2-3b":       131072,
		"llama-3.2-11b":      131072,
		"llama-3.2-90b":      131072,
		"llama-3.3-70b":      131072,
		"mixtral-8x7b-32768": 32768,
		"gemma-7b-it":        8192,
		"gemma2-9b-it":       8192,

		// Qwen models.
		"qwen-turbo":    8192,
		"qwen-plus":     32768,
		"qwen-max":      32768,
		"qwen2-72b":     131072,
		"qwen2.5-72b":   131072,
		"qwen2.5-coder": 131072,
		"qwq-32b":       131072,
	}

	if ctx, ok := contextWindows[model]; ok {
		return ctx
	}

	// Return 0 for unknown models - callers should use a sensible default.
	return 0
}
