// Package llm provides vendor-agnostic LLM provider interfaces and types.
//
// This package defines the core abstractions for integrating with various
// Large Language Model (LLM) providers such as OpenAI, Ollama, LMStudio, and
// other compatible backends.
//
// # Provider Interface
//
// The Provider interface defines the contract for LLM backends:
//
//	type Provider interface {
//	    Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
//	    Stream(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error)
//	    Models(ctx context.Context) ([]Model, error)
//	    Capabilities() Capabilities
//	    Name() string
//	    Close() error
//	}
//
// # Usage Example
//
//	// Create a provider
//	provider := llm.NewMockProvider("test")
//
//	// Make a completion request
//	req := llm.CompletionRequest{
//	    Messages: []llm.Message{
//	        {Role: "user", Content: "Hello!"},
//	    },
//	    Model: "gpt-4",
//	    MaxTokens: 100,
//	}
//
//	resp, err := provider.Complete(ctx, req)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	fmt.Println(resp.Content)
//
// # Streaming Example
//
//	// Stream a response
//	chunks, err := provider.Stream(ctx, req)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	for chunk := range chunks {
//	    if chunk.Error != nil {
//	        log.Printf("Error: %v", chunk.Error)
//	        continue
//	    }
//	    fmt.Print(chunk.Content)
//	}
//
// # Mock Provider
//
// The package includes a MockProvider for testing:
//
//	mock := llm.NewMockProvider("test",
//	    llm.WithResponse("Hello, World!"),
//	    llm.WithToolCalls([]llm.ToolCall{...}),
//	)
//
// # Design Philosophy
//
// This package follows these principles:
//
//   - Zero vendor lock-in: Works with any OpenAI-compatible API
//   - Simple interfaces: Minimal abstraction over provider APIs
//   - Context-aware: All operations support context cancellation
//   - Testable: Mock implementation for easy testing
//   - Type-safe: Strong typing for all request/response types
package llm
