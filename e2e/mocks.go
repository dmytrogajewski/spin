package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/dmytrogajewski/spin/internal/llm"
)

// MockLLM is a mock LLM provider for testing.
type MockLLM struct {
	mu        sync.RWMutex
	responses []MockResponse
	callIndex int
	calls     []llm.CompletionRequest
}

// MockResponse represents a mocked LLM response.
type MockResponse struct {
	// Content is the text response
	Content string

	// ToolCalls are tool calls to return
	ToolCalls []llm.ToolCall

	// Error to return (if any)
	Error error

	// Delay before returning response
	Delay time.Duration

	// Stream indicates whether to stream the response
	Stream bool
}

// NewMockLLM creates a new mock LLM provider.
func NewMockLLM(responses []MockResponse) *MockLLM {
	return &MockLLM{
		responses: responses,
		calls:     make([]llm.CompletionRequest, 0),
	}
}

// Name returns the provider name.
func (m *MockLLM) Name() string {
	return "mock"
}

// Models returns available models (empty for mock).
func (m *MockLLM) Models(ctx context.Context) ([]llm.Model, error) {
	return []llm.Model{
		{
			ID:          "mock-model",
			Name:        "Mock Model",
			Description: "Mock LLM for testing",
			ContextSize: 8192,
		},
	}, nil
}

// Capabilities returns mock capabilities.
func (m *MockLLM) Capabilities() llm.Capabilities {
	return llm.Capabilities{
		Streaming:       true,
		FunctionCalling: true,
		Vision:          false,
	}
}

// Close closes the mock provider (no-op).
func (m *MockLLM) Close() error {
	return nil
}

// Complete generates a completion (non-streaming).
func (m *MockLLM) Complete(ctx context.Context, req llm.CompletionRequest) (*llm.CompletionResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Record the call
	m.calls = append(m.calls, req)

	// Check if we have responses left
	if m.callIndex >= len(m.responses) {
		return nil, fmt.Errorf("mock LLM: no more responses (call index %d, total %d)", m.callIndex, len(m.responses))
	}

	resp := m.responses[m.callIndex]
	m.callIndex++

	// Simulate delay
	if resp.Delay > 0 {
		time.Sleep(resp.Delay)
	}

	// Return error if configured
	if resp.Error != nil {
		return nil, resp.Error
	}

	// Build response
	llmResp := &llm.CompletionResponse{
		ID:        fmt.Sprintf("mock-%d", time.Now().UnixNano()),
		Content:   resp.Content,
		ToolCalls: resp.ToolCalls,
		Model:     "mock-model",
		Usage: llm.Usage{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
		},
		FinishReason: "stop",
	}

	return llmResp, nil
}

// Stream generates a streaming completion.
func (m *MockLLM) Stream(ctx context.Context, req llm.CompletionRequest) (<-chan llm.StreamChunk, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Record the call
	m.calls = append(m.calls, req)

	// Check if we have responses left
	if m.callIndex >= len(m.responses) {
		return nil, fmt.Errorf("mock LLM: no more responses (call index %d, total %d)", m.callIndex, len(m.responses))
	}

	resp := m.responses[m.callIndex]
	m.callIndex++

	chunks := make(chan llm.StreamChunk, 100)

	go func() {
		defer close(chunks)

		// Simulate delay
		if resp.Delay > 0 {
			time.Sleep(resp.Delay)
		}

		// Return error if configured
		if resp.Error != nil {
			chunks <- llm.StreamChunk{
				Error: resp.Error,
			}
			return
		}

		// Stream content in chunks
		if resp.Content != "" {
			// Split content into words for realistic streaming
			words := splitWords(resp.Content)
			for _, word := range words {
				select {
				case <-ctx.Done():
					return
				default:
					chunks <- llm.StreamChunk{
						Type:    llm.ChunkTypeContentDelta,
						Content: word,
					}

					// Small delay between chunks for realism
					time.Sleep(1 * time.Millisecond)
				}
			}
		}

		// Send tool calls
		if len(resp.ToolCalls) > 0 {
			for _, toolCall := range resp.ToolCalls {
				chunks <- llm.StreamChunk{
					Type:     llm.ChunkTypeToolCallComplete,
					ToolCall: &toolCall,
				}
			}
		}

		// Send final chunk
		chunks <- llm.StreamChunk{
			Type:         llm.ChunkTypeDone,
			FinishReason: "stop",
		}
	}()

	return chunks, nil
}

// GetCalls returns all completion requests made to the mock.
func (m *MockLLM) GetCalls() []llm.CompletionRequest {
	m.mu.RLock()
	defer m.mu.RUnlock()

	calls := make([]llm.CompletionRequest, len(m.calls))
	copy(calls, m.calls)
	return calls
}

// GetCallCount returns the number of calls made.
func (m *MockLLM) GetCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.calls)
}

// Reset resets the mock state.
func (m *MockLLM) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.callIndex = 0
	m.calls = make([]llm.CompletionRequest, 0)
}

// AddResponse adds a response to the mock.
func (m *MockLLM) AddResponse(resp MockResponse) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.responses = append(m.responses, resp)
}

// SetResponses replaces all responses.
func (m *MockLLM) SetResponses(responses []MockResponse) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.responses = responses
	m.callIndex = 0
}

// splitWords splits text into words for streaming simulation.
func splitWords(text string) []string {
	if text == "" {
		return nil
	}

	var words []string
	var current string

	for _, ch := range text {
		current += string(ch)
		if ch == ' ' || ch == '\n' || ch == '\t' {
			if current != "" {
				words = append(words, current)
				current = ""
			}
		}
	}

	if current != "" {
		words = append(words, current)
	}

	return words
}

// MockToolCall creates a mock tool call.
func MockToolCall(name string, args map[string]interface{}) llm.ToolCall {
	// Encode arguments as JSON string
	argsJSON := "{}"
	if len(args) > 0 {
		data, _ := json.Marshal(args)
		argsJSON = string(data)
	}

	return llm.ToolCall{
		ID:   fmt.Sprintf("call_%s_%d", name, time.Now().UnixNano()),
		Type: "function",
		Function: llm.FunctionCall{
			Name:      name,
			Arguments: argsJSON,
		},
	}
}

// MockResponseWithContent creates a simple content response.
func MockResponseWithContent(content string) MockResponse {
	return MockResponse{
		Content: content,
		Stream:  true,
	}
}

// MockResponseWithToolCalls creates a response with tool calls.
func MockResponseWithToolCalls(toolCalls ...llm.ToolCall) MockResponse {
	return MockResponse{
		ToolCalls: toolCalls,
		Stream:    true,
	}
}

// MockResponseWithError creates an error response.
func MockResponseWithError(err error) MockResponse {
	return MockResponse{
		Error: err,
	}
}

// MockResponseWithDelay creates a delayed response.
func MockResponseWithDelay(content string, delay time.Duration) MockResponse {
	return MockResponse{
		Content: content,
		Delay:   delay,
		Stream:  true,
	}
}
