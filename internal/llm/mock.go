package llm

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/openai/openai-go"
)

const (
	mockTokensPerMessage = 10
	mockCharsPerToken    = 4
)

// MockProvider implements Provider for testing.
//
// MockProvider allows configuring responses, tool calls, errors, and streaming
// behavior for predictable testing of LLM-dependent code.
type MockProvider struct {
	mu sync.RWMutex

	name         string
	response     string
	toolCalls    []openai.ChatCompletionMessageToolCall
	streamChunks []string
	err          error
	capabilities Capabilities
	models       []openai.Model
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
		models: []openai.Model{
			{
				ID:      "mock-model",
				Created: time.Now().Unix(),
				Object:  "model",
			},
		},
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

// Complete implements Provider.Complete.
func (p *MockProvider) Complete(ctx context.Context, params openai.ChatCompletionNewParams) (*openai.ChatCompletion, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Simulate delay if configured.
	if p.delay > 0 {
		select {
		case <-time.After(p.delay):
		case <-ctx.Done():
			return nil, fmt.Errorf("mock complete: %w", ctx.Err())
		}
	}

	// Return error if configured.
	if p.err != nil {
		return nil, p.err
	}

	// Determine finish reason.
	finishReason := FinishReasonStop
	if len(p.toolCalls) > 0 {
		finishReason = FinishReasonToolCalls
	}

	// Count messages for usage calculation.
	messageCount := len(params.Messages)

	// Build response.
	resp := &openai.ChatCompletion{
		ID:      fmt.Sprintf("mock-%d", time.Now().UnixNano()),
		Created: time.Now().Unix(),
		Model:   "mock-model",
		Object:  "chat.completion",
		Choices: []openai.ChatCompletionChoice{
			{
				Index: 0,
				Message: openai.ChatCompletionMessage{
					Role:      "assistant",
					Content:   p.response,
					ToolCalls: p.toolCalls,
				},
				FinishReason: finishReason,
			},
		},
		Usage: openai.CompletionUsage{
			PromptTokens:     int64(messageCount * mockTokensPerMessage),
			CompletionTokens: int64(len(p.response) / mockCharsPerToken),
			TotalTokens:      int64(messageCount*10 + len(p.response)/4),
		},
	}

	return resp, nil
}

// Stream implements Provider.Stream.
func (p *MockProvider) Stream(ctx context.Context, _ openai.ChatCompletionNewParams) (<-chan openai.ChatCompletionChunk, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Return error immediately if configured.
	if p.err != nil {
		return nil, p.err
	}

	chunks := make(chan openai.ChatCompletionChunk, 10)
	chunkID := fmt.Sprintf("mock-chunk-%d", time.Now().UnixNano())

	go func() {
		defer close(chunks)

		// Send content chunks or tool calls.
		switch {
		case len(p.streamChunks) > 0:
			if !p.streamContentChunks(ctx, chunks, chunkID) {
				return
			}
		case len(p.toolCalls) > 0:
			if !p.streamToolCallChunks(ctx, chunks, chunkID) {
				return
			}
		default:
			if !p.streamSingleChunk(ctx, chunks, chunkID) {
				return
			}
		}

		// Send done chunk with finish reason.
		p.sendDoneChunk(ctx, chunks, chunkID)
	}()

	return chunks, nil
}

// streamContentChunks streams configured content chunks. Returns false if context canceled.
func (p *MockProvider) streamContentChunks(ctx context.Context, chunks chan<- openai.ChatCompletionChunk, chunkID string) bool {
	for _, content := range p.streamChunks {
		chunk := newMockChunk(chunkID, content, "assistant", nil)

		if !sendChunk(ctx, chunks, chunk) {
			return false
		}

		if p.delay > 0 && !waitDelay(ctx, p.delay) {
			return false
		}
	}

	return true
}

// streamToolCallChunks streams tool call chunks. Returns false if context canceled.
func (p *MockProvider) streamToolCallChunks(ctx context.Context, chunks chan<- openai.ChatCompletionChunk, chunkID string) bool {
	toolCallChunks := make([]openai.ChatCompletionChunkChoiceDeltaToolCall, len(p.toolCalls))
	for i, tc := range p.toolCalls {
		toolCallChunks[i] = openai.ChatCompletionChunkChoiceDeltaToolCall{
			Index: int64(i),
			ID:    tc.ID,
			Type:  string(tc.Type),
			Function: openai.ChatCompletionChunkChoiceDeltaToolCallFunction{
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			},
		}
	}

	chunk := newMockChunk(chunkID, "", "assistant", toolCallChunks)

	return sendChunk(ctx, chunks, chunk)
}

// streamSingleChunk streams a single response chunk. Returns false if context canceled.
func (p *MockProvider) streamSingleChunk(ctx context.Context, chunks chan<- openai.ChatCompletionChunk, chunkID string) bool {
	if p.delay > 0 && !waitDelay(ctx, p.delay) {
		return false
	}

	chunk := newMockChunk(chunkID, p.response, "assistant", nil)

	return sendChunk(ctx, chunks, chunk)
}

// sendDoneChunk sends the final done chunk with the appropriate finish reason.
func (p *MockProvider) sendDoneChunk(ctx context.Context, chunks chan<- openai.ChatCompletionChunk, chunkID string) {
	finishReason := FinishReasonStop
	if len(p.toolCalls) > 0 {
		finishReason = FinishReasonToolCalls
	}

	select {
	case <-ctx.Done():
		return
	case chunks <- openai.ChatCompletionChunk{
		ID:      chunkID,
		Created: time.Now().Unix(),
		Model:   "mock-model",
		Object:  "chat.completion.chunk",
		Choices: []openai.ChatCompletionChunkChoice{
			{
				Index:        0,
				FinishReason: finishReason,
			},
		},
	}:
	}
}

// newMockChunk creates a new mock ChatCompletionChunk.
func newMockChunk(
	chunkID, content string,
	role string,
	toolCalls []openai.ChatCompletionChunkChoiceDeltaToolCall,
) openai.ChatCompletionChunk {
	return openai.ChatCompletionChunk{
		ID:      chunkID,
		Created: time.Now().Unix(),
		Model:   "mock-model",
		Object:  "chat.completion.chunk",
		Choices: []openai.ChatCompletionChunkChoice{
			{
				Index: 0,
				Delta: openai.ChatCompletionChunkChoiceDelta{
					Content:   content,
					Role:      role,
					ToolCalls: toolCalls,
				},
			},
		},
	}
}

// sendChunk sends a chunk to the channel, returning false if context is canceled.
func sendChunk(ctx context.Context, chunks chan<- openai.ChatCompletionChunk, chunk openai.ChatCompletionChunk) bool {
	select {
	case <-ctx.Done():
		return false
	case chunks <- chunk:
		return true
	}
}

// waitDelay waits for the specified delay, returning false if context is canceled.
func waitDelay(ctx context.Context, delay time.Duration) bool {
	select {
	case <-time.After(delay):
		return true
	case <-ctx.Done():
		return false
	}
}

// Models implements Provider.Models.
func (p *MockProvider) Models(_ context.Context) ([]openai.Model, error) {
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
	// Mock provider has no resources to clean up.
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
func (p *MockProvider) SetToolCalls(calls []openai.ChatCompletionMessageToolCall) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.toolCalls = calls
}

// MockOption configures a MockProvider.
type MockOption func(*MockProvider)

// WithResponse sets a custom response for the mock provider.
func WithResponse(response string) MockOption {
	return func(p *MockProvider) {
		p.response = response
	}
}

// WithError sets an error to be returned by the mock provider.
func WithError(err error) MockOption {
	return func(p *MockProvider) {
		p.err = err
	}
}

// WithToolCalls sets tool calls to be returned by the mock provider.
func WithToolCalls(calls []openai.ChatCompletionMessageToolCall) MockOption {
	return func(p *MockProvider) {
		p.toolCalls = calls
	}
}

// WithStreaming sets chunks to be streamed by the mock provider.
func WithStreaming(chunks []string) MockOption {
	return func(p *MockProvider) {
		p.streamChunks = chunks
		// Build response from chunks.
		var (
			combined      string
			combinedSb351 strings.Builder
		)

		for _, chunk := range chunks {
			combinedSb351.WriteString(chunk)
		}

		combined += combinedSb351.String()

		p.response = combined
	}
}

// WithCapabilities sets the capabilities of the mock provider.
func WithCapabilities(caps Capabilities) MockOption {
	return func(p *MockProvider) {
		p.capabilities = caps
	}
}

// WithModels sets the models available from the mock provider.
func WithModels(models []openai.Model) MockOption {
	return func(p *MockProvider) {
		p.models = models
	}
}

// WithDelay sets a delay to simulate latency in the mock provider.
func WithDelay(delay time.Duration) MockOption {
	return func(p *MockProvider) {
		p.delay = delay
	}
}
