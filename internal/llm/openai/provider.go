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
	"github.com/dmytrogajewski/spin/pkg/llmutil"
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
	client  openai.Client
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
	if params.Model == "" {
		params.Model = openai.ChatModel(p.model)
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
	if params.Model == "" {
		params.Model = openai.ChatModel(p.model)
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
		ContextWindow:   llmutil.ModelContextWindow(p.model),
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
