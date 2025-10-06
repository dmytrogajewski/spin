package llm

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MockProvider implements Provider for testing.
//
// MockProvider allows configuring responses, tool calls, errors, and streaming
// behavior for predictable testing of LLM-dependent code.
type MockProvider struct {
	mu sync.RWMutex

	name         string
	response     string
	toolCalls    []ToolCall
	streamChunks []string
	err          error
	capabilities Capabilities
	models       []Model
	delay        time.Duration
}

// NewMockProvider creates a new mock provider with the given name.
//
// By default, the mock provider:
//   - Returns "ok" as the response
//   - Has no tool calls
//   - Supports streaming and function calling
//   - Has no delay
//
// Use MockOption functions to configure behavior.
func NewMockProvider(name string, opts ...MockOption) *MockProvider {
	p := &MockProvider{
		name:     name,
		response: "ok",
		capabilities: Capabilities{
			Streaming:       true,
			FunctionCalling: true,
			Vision:          false,
		},
		models: []Model{
			{
				ID:          "mock-model",
				Name:        "Mock Model",
				Description: "A mock model for testing",
				ContextSize: 4096,
			},
		},
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

// Complete implements Provider.Complete.
func (p *MockProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Simulate delay if configured
	if p.delay > 0 {
		select {
		case <-time.After(p.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	// Return error if configured
	if p.err != nil {
		return nil, p.err
	}

	// Build response
	resp := &CompletionResponse{
		ID:        fmt.Sprintf("mock-%d", time.Now().UnixNano()),
		Model:     req.Model,
		Content:   p.response,
		ToolCalls: p.toolCalls,
		Usage: Usage{
			PromptTokens:     len(req.Messages) * 10,
			CompletionTokens: len(p.response) / 4,
			TotalTokens:      len(req.Messages)*10 + len(p.response)/4,
		},
		FinishReason: "stop",
	}

	if len(p.toolCalls) > 0 {
		resp.FinishReason = "tool_calls"
	}

	return resp, nil
}

// Stream implements Provider.Stream.
func (p *MockProvider) Stream(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Return error immediately if configured
	if p.err != nil {
		return nil, p.err
	}

	chunks := make(chan StreamChunk, 10)

	go func() {
		defer close(chunks)

		// Send content chunks or tool calls
		if len(p.streamChunks) > 0 {
			// Stream configured chunks
			for _, content := range p.streamChunks {
				select {
				case <-ctx.Done():
					chunks <- StreamChunk{Type: ChunkTypeError, Error: ctx.Err()}
					return
				case chunks <- StreamChunk{Type: ChunkTypeContentDelta, Content: content}:
				}

				// Simulate delay between chunks
				if p.delay > 0 {
					select {
					case <-time.After(p.delay):
					case <-ctx.Done():
						chunks <- StreamChunk{Type: ChunkTypeError, Error: ctx.Err()}
						return
					}
				}
			}
		} else if len(p.toolCalls) > 0 {
			// Stream tool calls
			for _, tc := range p.toolCalls {
				select {
				case <-ctx.Done():
					chunks <- StreamChunk{Type: ChunkTypeError, Error: ctx.Err()}
					return
				case chunks <- StreamChunk{Type: ChunkTypeToolCallStart, ToolCall: &tc}:
				}
			}
		} else {
			// Stream response as single chunk
			// Apply delay before sending if configured
			if p.delay > 0 {
				select {
				case <-time.After(p.delay):
				case <-ctx.Done():
					chunks <- StreamChunk{Type: ChunkTypeError, Error: ctx.Err()}
					return
				}
			}

			select {
			case <-ctx.Done():
				chunks <- StreamChunk{Type: ChunkTypeError, Error: ctx.Err()}
				return
			case chunks <- StreamChunk{Type: ChunkTypeContentDelta, Content: p.response}:
			}
		}

		// Send done chunk
		finishReason := "stop"
		if len(p.toolCalls) > 0 {
			finishReason = "tool_calls"
		}

		select {
		case <-ctx.Done():
			chunks <- StreamChunk{Type: ChunkTypeError, Error: ctx.Err()}
		case chunks <- StreamChunk{Type: ChunkTypeDone, FinishReason: finishReason}:
		}
	}()

	return chunks, nil
}

// Models implements Provider.Models.
func (p *MockProvider) Models(ctx context.Context) ([]Model, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.err != nil {
		return nil, p.err
	}

	return p.models, nil
}

// Capabilities implements Provider.Capabilities.
func (p *MockProvider) Capabilities() Capabilities {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.capabilities
}

// Name implements Provider.Name.
func (p *MockProvider) Name() string {
	return p.name
}

// Close implements Provider.Close.
func (p *MockProvider) Close() error {
	// Mock provider has no resources to clean up
	return nil
}

// SetResponse updates the mock response.
// Thread-safe and can be called during provider usage.
func (p *MockProvider) SetResponse(response string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.response = response
}

// SetError updates the mock error.
// Thread-safe and can be called during provider usage.
func (p *MockProvider) SetError(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.err = err
}

// SetToolCalls updates the mock tool calls.
// Thread-safe and can be called during provider usage.
func (p *MockProvider) SetToolCalls(calls []ToolCall) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.toolCalls = calls
}

// MockOption configures a MockProvider.
type MockOption func(*MockProvider)

// WithResponse sets the response content.
func WithResponse(response string) MockOption {
	return func(p *MockProvider) {
		p.response = response
	}
}

// WithToolCalls sets the tool calls to return.
func WithToolCalls(calls []ToolCall) MockOption {
	return func(p *MockProvider) {
		p.toolCalls = calls
	}
}

// WithError sets an error to return from all operations.
func WithError(err error) MockOption {
	return func(p *MockProvider) {
		p.err = err
	}
}

// WithStreaming configures streaming chunks.
// If chunks are provided, Complete() will return their concatenation,
// and Stream() will emit each chunk separately.
func WithStreaming(chunks []string) MockOption {
	return func(p *MockProvider) {
		p.streamChunks = chunks
		// Update response to be concatenation of chunks
		result := ""
		for _, chunk := range chunks {
			result += chunk
		}
		p.response = result
	}
}

// WithCapabilities sets the provider capabilities.
func WithCapabilities(caps Capabilities) MockOption {
	return func(p *MockProvider) {
		p.capabilities = caps
	}
}

// WithModels sets the available models.
func WithModels(models []Model) MockOption {
	return func(p *MockProvider) {
		p.models = models
	}
}

// WithDelay sets a delay for each operation.
// Useful for testing timeout and cancellation behavior.
func WithDelay(delay time.Duration) MockOption {
	return func(p *MockProvider) {
		p.delay = delay
	}
}
