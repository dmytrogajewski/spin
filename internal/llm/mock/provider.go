package mock

import (
	"time"
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
