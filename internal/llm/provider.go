package llm

import "context"

// Provider represents an LLM backend.
//
// Implementations must be safe for concurrent use by multiple goroutines.
type Provider interface {
	// Complete performs a non-streaming completion request.
	//
	// The context can be used for cancellation and timeout control.
	// Returns an error if the request fails or times out.
	Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)

	// Stream performs a streaming completion request.
	//
	// Returns a channel that will receive chunks as they arrive.
	// The channel will be closed when the stream completes or an error occurs.
	// The final chunk will have Type ChunkTypeDone or ChunkTypeError.
	//
	// Callers must consume all chunks from the channel to avoid goroutine leaks.
	// Context cancellation will stop the stream and close the channel.
	Stream(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error)

	// Models returns the list of available models.
	//
	// Returns an empty slice if model listing is not supported.
	// Some providers may return an error if not authenticated.
	Models(ctx context.Context) ([]Model, error)

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
