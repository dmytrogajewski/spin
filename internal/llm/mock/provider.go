package mock

import (
	"context"
	"strings"
	"time"

	"github.com/dmytrogajewski/spin/internal/llm"
)

// Provider is a mock LLM provider for testing.
//
// It returns canned responses instantly without calling any API.
// Useful for hermetic tests that don't depend on external services.
type Provider struct {
	// Responses is a queue of responses to return.
	// Each call to Complete/Stream pops the first response.
	Responses []string

	// Delay simulates LLM processing time (default: 0).
	Delay time.Duration

	// StreamChunkSize determines how responses are chunked when streaming (default: 10).
	StreamChunkSize int

	// CallHistory records all prompts received.
	CallHistory []string
}

// NewProvider creates a new mock provider.
func NewProvider(responses ...string) *Provider {
	return &Provider{
		Responses:       responses,
		StreamChunkSize: 10,
		CallHistory:     []string{},
	}
}

// Complete returns a mock response instantly.
func (p *Provider) Complete(ctx context.Context, req llm.CompletionRequest) (*llm.CompletionResponse, error) {
	// Get last user message for tracking
	var lastMsg string
	for _, msg := range req.Messages {
		if msg.Role == "user" {
			lastMsg = msg.Content
		}
	}
	p.CallHistory = append(p.CallHistory, lastMsg)

	// Simulate delay
	if p.Delay > 0 {
		select {
		case <-time.After(p.Delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	// Pop response from queue
	var content string
	if len(p.Responses) > 0 {
		content = p.Responses[0]
		p.Responses = p.Responses[1:]
	} else {
		content = "Mock response"
	}

	return &llm.CompletionResponse{
		Content:      content,
		Model:        "mock-model",
		FinishReason: "stop",
		Usage: llm.Usage{
			PromptTokens:     len(lastMsg) / 4, // Rough estimate
			CompletionTokens: len(content) / 4,
			TotalTokens:      (len(lastMsg) + len(content)) / 4,
		},
	}, nil
}

// Stream returns a mock streaming response.
func (p *Provider) Stream(ctx context.Context, req llm.CompletionRequest) (<-chan llm.StreamChunk, error) {
	// Get last user message for tracking
	var lastMsg string
	for _, msg := range req.Messages {
		if msg.Role == "user" {
			lastMsg = msg.Content
		}
	}
	p.CallHistory = append(p.CallHistory, lastMsg)

	chunks := make(chan llm.StreamChunk, 100)

	go func() {
		defer close(chunks)

		// Simulate delay
		if p.Delay > 0 {
			select {
			case <-time.After(p.Delay):
			case <-ctx.Done():
				return
			}
		}

		// Pop response from queue
		var content string
		if len(p.Responses) > 0 {
			content = p.Responses[0]
			p.Responses = p.Responses[1:]
		} else {
			content = "Mock streaming response"
		}

		// Send content in chunks
		chunkSize := p.StreamChunkSize
		if chunkSize <= 0 {
			chunkSize = 10
		}

		for i := 0; i < len(content); i += chunkSize {
			end := i + chunkSize
			if end > len(content) {
				end = len(content)
			}

			chunk := content[i:end]
			select {
			case chunks <- llm.StreamChunk{
				Type:    llm.ChunkTypeContentDelta,
				Content: chunk,
			}:
			case <-ctx.Done():
				return
			}

			// Small delay between chunks for realism
			if end < len(content) {
				time.Sleep(10 * time.Millisecond)
			}
		}

		// Send completion chunk
		chunks <- llm.StreamChunk{
			Type:         llm.ChunkTypeDone,
			FinishReason: "stop",
		}
	}()

	return chunks, nil
}

// Close is a no-op for mock provider.
func (p *Provider) Close() error {
	return nil
}

// Reset clears call history and reloads responses.
func (p *Provider) Reset(responses ...string) {
	p.Responses = responses
	p.CallHistory = []string{}
}

// LastPrompt returns the most recent prompt received, or empty string if none.
func (p *Provider) LastPrompt() string {
	if len(p.CallHistory) == 0 {
		return ""
	}
	return p.CallHistory[len(p.CallHistory)-1]
}

// PromptCount returns the number of prompts received.
func (p *Provider) PromptCount() int {
	return len(p.CallHistory)
}

// SetResponseForPrompt sets a response based on prompt matching.
//
// Example:
//
//	p.SetResponseForPrompt("what is 2+2", "The answer is 4")
//	p.SetResponseForPrompt("hello", "Hi there!")
type ResponseMap struct {
	Provider *Provider
	mapping  map[string]string
}

// NewResponseMap creates a mock provider with prompt-based routing.
func NewResponseMap() *ResponseMap {
	return &ResponseMap{
		Provider: NewProvider(),
		mapping:  make(map[string]string),
	}
}

// Set maps a prompt substring to a response.
func (rm *ResponseMap) Set(promptSubstring, response string) {
	rm.mapping[strings.ToLower(promptSubstring)] = response
}

// Get returns the response for a prompt (used internally).
func (rm *ResponseMap) Get(prompt string) string {
	promptLower := strings.ToLower(prompt)
	for substring, response := range rm.mapping {
		if strings.Contains(promptLower, substring) {
			return response
		}
	}
	return "No matching response configured for: " + prompt
}
