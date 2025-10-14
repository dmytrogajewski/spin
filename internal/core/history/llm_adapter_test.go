package history

import (
	"context"
	"testing"

	"github.com/dmytrogajewski/spin/internal/llm"
)

func TestLLMProviderAdapter(t *testing.T) {
	// Create a mock LLM provider
	mock := llm.NewMockProvider("test",
		llm.WithResponse("This is a test summary"),
	)

	// Create adapter
	adapter := NewLLMProviderAdapter(mock)

	// Test completion with string prompt
	ctx := context.Background()
	prompt := "Summarize this conversation"

	result, err := adapter.Complete(ctx, prompt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == "" {
		t.Errorf("expected non-empty response")
	}

	if result != "This is a test summary" {
		t.Errorf("expected mock response, got: %s", result)
	}
}

func TestLLMProviderAdapter_InvalidPromptType(t *testing.T) {
	mock := llm.NewMockProvider("test",
		llm.WithResponse("Response"),
	)

	adapter := NewLLMProviderAdapter(mock)

	ctx := context.Background()

	// Pass invalid type (int instead of string)
	result, err := adapter.Complete(ctx, 12345)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should handle gracefully and still call LLM
	if result == "" {
		t.Errorf("expected response even with invalid prompt type")
	}
}
