package llm

import (
	"context"

	"github.com/openai/openai-go"
)

// Provider represents an LLM backend.
//
// Implementations must be safe for concurrent use by multiple goroutines.
//
// All providers use OpenAI SDK types directly to eliminate unnecessary abstraction layers.
// This applies even to non-OpenAI providers (Ollama, LMStudio) since they implement
// OpenAI-compatible APIs.
type Provider interface {
	// Complete performs a non-streaming completion request.
	//
	// The context can be used for cancellation and timeout control.
	// Returns an error if the request fails or times out.
	Complete(ctx context.Context, params openai.ChatCompletionNewParams) (*openai.ChatCompletion, error)

	// Stream performs a streaming completion request.
	//
	// Returns a channel that receives chunks as they arrive.
	// The channel is closed when the stream completes or an error occurs.
	//
	// Callers must consume all chunks from the channel to avoid goroutine leaks.
	// Context cancellation will stop the stream and close the channel.
	//
	// Note: Errors are sent as the last chunk before closing. Check chunk for errors.
	Stream(ctx context.Context, params openai.ChatCompletionNewParams) (<-chan openai.ChatCompletionChunk, error)

	// Models returns the list of available models.
	//
	// Returns an empty slice if model listing is not supported.
	// Some providers may return an error if not authenticated.
	Models(ctx context.Context) ([]openai.Model, error)

	// Capabilities returns the capabilities of this provider.
	//
	// This indicates what features the provider supports (streaming,
	// function calling, vision, etc.).
	Capabilities() Capabilities

	// Name returns the provider name (e.g., "openai", "ollama", "mock").
	//
	// This is used for logging, diagnostics, and provider selection.
	Name() string

	// Close closes the provider and releases any resources.
	//
	// After Close is called, the provider should not be used.
	// Implementations should be idempotent (safe to call multiple times).
	Close() error
}
