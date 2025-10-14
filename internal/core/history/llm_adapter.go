package history

import (
	"context"

	"github.com/dmytrogajewski/spin/internal/llm"
)

// LLMProviderAdapter adapts llm.Provider to compress.LLMProvider interface.
//
// This adapter resolves the circular import between core and compress packages
// by implementing the minimal interface needed for summarization.
type LLMProviderAdapter struct {
	provider llm.Provider
}

// NewLLMProviderAdapter creates an adapter for an LLM provider.
func NewLLMProviderAdapter(provider llm.Provider) *LLMProviderAdapter {
	return &LLMProviderAdapter{
		provider: provider,
	}
}

// Complete implements compress.LLMProvider.Complete.
//
// It takes a string prompt (from summarization) and returns the LLM's response.
// The prompt parameter should be a string, not llm.CompletionRequest.
func (a *LLMProviderAdapter) Complete(ctx context.Context, prompt interface{}) (string, error) {
	// Extract prompt string
	promptStr, ok := prompt.(string)
	if !ok {
		promptStr = "Invalid prompt type"
	}

	// Build LLM request
	req := llm.CompletionRequest{
		Messages: []llm.Message{
			{
				Role:    "user",
				Content: promptStr,
			},
		},
		Temperature: 0.3, // Factual summarization
		MaxTokens:   200, // Compact summaries
	}

	// Call real LLM provider
	resp, err := a.provider.Complete(ctx, req)
	if err != nil {
		return "", err
	}

	return resp.Content, nil
}
