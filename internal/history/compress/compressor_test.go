package compress

import (
	"context"
	"testing"
)

func TestCompressibleMessage(t *testing.T) {
	msg := CompressibleMessage{
		ID:            "msg-1",
		Role:          "user",
		Content:       "test content",
		ToolCallCount: 2,
		Tokens:        100,
	}

	if msg.GetRole() != "user" {
		t.Errorf("CompressibleMessage.GetRole() = %s, want 'user'", msg.GetRole())
	}

	if msg.GetContent() != "test content" {
		t.Errorf("CompressibleMessage.GetContent() = %s, want 'test content'", msg.GetContent())
	}

	if msg.GetToolCallCount() != 2 {
		t.Errorf("CompressibleMessage.GetToolCallCount() = %d, want 2", msg.GetToolCallCount())
	}
}

// mockTokenizer is a test implementation of Tokenizer
type mockTokenizer struct {
	counts map[string]int
}

func (m *mockTokenizer) Count(text string) int {
	if count, exists := m.counts[text]; exists {
		return count
	}
	return len(text) // Simple fallback
}

// mockCompressor is a test implementation of Compressor
type mockCompressor struct {
	name     string
	compress func(ctx context.Context, messages []CompressibleMessage, targetTokens int, tokenizer Tokenizer) ([]CompressibleMessage, error)
}

func (m *mockCompressor) Compress(ctx context.Context, messages []CompressibleMessage, targetTokens int, tokenizer Tokenizer) ([]CompressibleMessage, error) {
	return m.compress(ctx, messages, targetTokens, tokenizer)
}

func (m *mockCompressor) Name() string {
	return m.name
}

func TestMockCompressor(t *testing.T) {
	compressor := &mockCompressor{
		name: "test-compressor",
		compress: func(ctx context.Context, messages []CompressibleMessage, targetTokens int, tokenizer Tokenizer) ([]CompressibleMessage, error) {
			// Simple compression: return first message if it fits
			if len(messages) > 0 && tokenizer.Count(messages[0].Content) <= targetTokens {
				return messages[:1], nil
			}
			return []CompressibleMessage{}, nil
		},
	}

	if compressor.Name() != "test-compressor" {
		t.Errorf("mockCompressor.Name() = %s, want 'test-compressor'", compressor.Name())
	}

	messages := []CompressibleMessage{
		{ID: "1", Role: "user", Content: "short", Tokens: 10},
		{ID: "2", Role: "assistant", Content: "long message", Tokens: 100},
	}

	tokenizer := &mockTokenizer{counts: map[string]int{"short": 5}}

	result, err := compressor.Compress(context.Background(), messages, 10, tokenizer)
	if err != nil {
		t.Errorf("mockCompressor.Compress() unexpected error: %v", err)
	}

	if len(result) != 1 {
		t.Errorf("mockCompressor.Compress() result length = %d, want 1", len(result))
	}

	if result[0].ID != "1" {
		t.Errorf("mockCompressor.Compress() result[0].ID = %s, want '1'", result[0].ID)
	}
}

func TestMockTokenizer(t *testing.T) {
	tokenizer := &mockTokenizer{
		counts: map[string]int{
			"short": 5,
			"long":  100,
		},
	}

	if tokenizer.Count("short") != 5 {
		t.Errorf("mockTokenizer.Count('short') = %d, want 5", tokenizer.Count("short"))
	}

	if tokenizer.Count("long") != 100 {
		t.Errorf("mockTokenizer.Count('long') = %d, want 100", tokenizer.Count("long"))
	}

	if tokenizer.Count("unknown") != 7 {
		t.Errorf("mockTokenizer.Count('unknown') = %d, want 7", tokenizer.Count("unknown"))
	}
}
