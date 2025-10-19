package compress

import (
	"context"
	"errors"
	"testing"
)

// Mock LLM provider for testing composite compressor
type mockLLMForComposite struct {
	completeFunc func(ctx context.Context, req interface{}) (string, error)
}

func (m *mockLLMForComposite) Complete(ctx context.Context, req interface{}) (string, error) {
	if m.completeFunc != nil {
		return m.completeFunc(ctx, req)
	}
	return "Mock summary", nil
}

func TestNewLLMWithHybridFallback(t *testing.T) {
	mockLLM := &mockLLMForComposite{}
	config := DefaultLLMSummarizerConfig()

	compressor := NewLLMWithHybridFallback(mockLLM, config)

	if compressor == nil {
		t.Fatal("NewLLMWithHybridFallback() returned nil")
	}

	if !contains(compressor.Name(), "composite") {
		t.Errorf("NewLLMWithHybridFallback().Name() = %v, want composite", compressor.Name())
	}
}

func TestNewDefaultLLMWithHybridFallback(t *testing.T) {
	mockLLM := &mockLLMForComposite{}

	compressor := NewDefaultLLMWithHybridFallback(mockLLM)

	if compressor == nil {
		t.Fatal("NewDefaultLLMWithHybridFallback() returned nil")
	}

	if !contains(compressor.Name(), "composite") {
		t.Errorf("NewDefaultLLMWithHybridFallback().Name() = %v, want composite", compressor.Name())
	}
}

func TestCompositeCompressor_Compress_LLMSuccess(t *testing.T) {
	mockLLM := &mockLLMForComposite{
		completeFunc: func(ctx context.Context, req interface{}) (string, error) {
			return "Summarized content", nil
		},
	}

	compressor := NewDefaultLLMWithHybridFallback(mockLLM)

	tokenizer := &mockTokenizer{
		counts: map[string]int{
			"Test message 1":  100,
			"Test response 1": 100,
		},
	}

	messages := []CompressibleMessage{
		{Role: "user", Content: "Test message 1", Tokens: 100},
		{Role: "assistant", Content: "Test response 1", Tokens: 100},
	}

	result, err := compressor.Compress(context.Background(), messages, 150, tokenizer)

	if err != nil {
		t.Errorf("Compress() unexpected error: %v", err)
	}

	if len(result) == 0 {
		t.Error("Compress() returned empty result")
	}
}

func TestCompositeCompressor_Compress_LLMFallsBackToHybrid(t *testing.T) {
	mockLLM := &mockLLMForComposite{
		completeFunc: func(ctx context.Context, req interface{}) (string, error) {
			return "", errors.New("LLM error")
		},
	}

	compressor := NewDefaultLLMWithHybridFallback(mockLLM)

	tokenizer := &mockTokenizer{
		counts: map[string]int{
			"Test message 1":  50,
			"Test response 1": 50,
		},
	}

	messages := []CompressibleMessage{
		{Role: "user", Content: "Test message 1", Tokens: 50},
		{Role: "assistant", Content: "Test response 1", Tokens: 50},
	}

	result, err := compressor.Compress(context.Background(), messages, 80, tokenizer)

	if err != nil {
		t.Errorf("Compress() unexpected error: %v", err)
	}

	// Should fall back to hybrid compressor
	if len(result) == 0 {
		t.Error("Compress() returned empty result after fallback")
	}
}

func TestCompositeCompressor_Name(t *testing.T) {
	mockLLM := &mockLLMForComposite{}

	compressor := NewDefaultLLMWithHybridFallback(mockLLM)

	name := compressor.Name()
	if !contains(name, "composite") {
		t.Errorf("Name() = %v, want to contain 'composite'", name)
	}
}

func TestCompositeCompressor_Compress_EmptyMessages(t *testing.T) {
	mockLLM := &mockLLMForComposite{}

	compressor := NewDefaultLLMWithHybridFallback(mockLLM)

	tokenizer := &mockTokenizer{
		counts: map[string]int{},
	}

	result, err := compressor.Compress(context.Background(), []CompressibleMessage{}, 100, tokenizer)

	if err != nil {
		t.Errorf("Compress() unexpected error: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("Compress() expected empty result, got %d messages", len(result))
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || hasSubstring(s, substr)))
}

func hasSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
