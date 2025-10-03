// Package testing provides test utilities for the core package.
package testing

import (
	"context"
	"errors"
)

// MockProvider is a mock LLM provider for testing.
// It returns predefined responses or errors for testing purposes.
type MockProvider struct {
	// Response is the mock response content
	Response string

	// Error is the mock error to return
	Error error

	// Calls tracks the number of times Complete was called
	Calls int

	// RequestHistory stores all requests made
	RequestHistory []MockRequest
}

// MockRequest stores a mock request for inspection
type MockRequest struct {
	Messages    []MockMessage
	Temperature float64
	MaxTokens   int
}

// MockMessage represents a conversation message in mock requests
type MockMessage struct {
	Role    string
	Content string
}

// Complete implements the LLMProvider interface for testing
func (m *MockProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	m.Calls++

	// Store request for inspection
	mockReq := MockRequest{
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}
	for _, msg := range req.Messages {
		mockReq.Messages = append(mockReq.Messages, MockMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}
	m.RequestHistory = append(m.RequestHistory, mockReq)

	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Return error if set
	if m.Error != nil {
		return nil, m.Error
	}

	// Return mock response
	return &CompletionResponse{
		Content: m.Response,
		Error:   nil,
	}, nil
}

// Reset clears the mock state
func (m *MockProvider) Reset() {
	m.Calls = 0
	m.RequestHistory = nil
}

// NewMockProvider creates a new mock provider with the given response
func NewMockProvider(response string) *MockProvider {
	return &MockProvider{
		Response: response,
	}
}

// NewMockProviderWithError creates a mock provider that returns an error
func NewMockProviderWithError(err error) *MockProvider {
	return &MockProvider{
		Error: err,
	}
}

// Common errors for testing
var (
	ErrMockLLMFailed    = errors.New("mock LLM failed")
	ErrMockLLMTimeout   = errors.New("mock LLM timeout")
	ErrMockInvalidInput = errors.New("mock invalid input")
)

// CompletionRequest is a minimal version of the LLM completion request
// This will be replaced by the full internal/llm package in Phase 8.1
type CompletionRequest struct {
	Messages    []Message
	Temperature float64
	MaxTokens   int
}

// CompletionResponse is a minimal version of the LLM completion response
// This will be replaced by the full internal/llm package in Phase 8.1
type CompletionResponse struct {
	Content string
	Error   error
}

// Message represents a conversation message
// This will be replaced by the full internal/llm package in Phase 8.1
type Message struct {
	Role    string // system, user, assistant
	Content string
}

// LLMProvider defines the minimal interface needed for planning
// This will be replaced by the full internal/llm package in Phase 8.1
type LLMProvider interface {
	Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
}
