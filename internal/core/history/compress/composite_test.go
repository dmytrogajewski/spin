package compress

import (
	"context"
	"fmt"
	"testing"
)

func TestCompositeCompressor_PrimarySucceeds(t *testing.T) {
	primary := &mockLLMProvider{response: "Summary"}

	config := LLMSummarizerConfig{
		ChunkSize:    5,
		RecentWindow: 5,
		Temperature:  0.3,
		MaxTokens:    200,
	}

	llmSummarizer := NewLLMSummarizer(primary, config)
	hybrid := NewDefaultHybridCompressor()

	composite := NewCompositeCompressor(llmSummarizer, hybrid)

	tokenizer := &simpleTokenizer{}
	ctx := context.Background()

	// Create enough messages to trigger summarization
	messages := make([]CompressibleMessage, 20)
	for i := 0; i < 20; i++ {
		messages[i] = CompressibleMessage{
			Role:    RoleAssistant,
			Content: fmt.Sprintf("Long message %d", i),
			Tokens:  150,
		}
	}

	// Target: 500 tokens (current: 3000, so needs compression)
	compressed, err := composite.Compress(ctx, messages, 500, tokenizer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(compressed) == 0 {
		t.Errorf("expected compressed messages")
	}

	// Verify primary (LLM) was called
	if primary.calls == 0 {
		t.Errorf("expected primary (LLM) to be called")
	}
}

func TestCompositeCompressor_PrimaryFailsFallbackSucceeds(t *testing.T) {
	primary := &mockLLMProvider{
		err: fmt.Errorf("LLM API unavailable"),
	}
	llmSummarizer := NewDefaultLLMSummarizer(primary)
	hybrid := NewDefaultHybridCompressor()

	composite := NewCompositeCompressor(llmSummarizer, hybrid)

	tokenizer := &simpleTokenizer{}
	ctx := context.Background()

	messages := []CompressibleMessage{
		{Role: RoleUser, Content: "Critical", Tokens: 50},
		{Role: RoleAssistant, Content: "Response 1", Tokens: 100},
		{Role: RoleAssistant, Content: "Response 2", Tokens: 100},
	}

	compressed, err := composite.Compress(ctx, messages, 150, tokenizer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have fallen back to hybrid compression
	if len(compressed) == 0 {
		t.Errorf("expected fallback to produce results")
	}

	// Verify critical message preserved (hybrid behavior)
	hasCritical := false
	for _, msg := range compressed {
		if msg.Role == RoleUser {
			hasCritical = true
			break
		}
	}

	if !hasCritical {
		t.Errorf("expected critical message preserved by fallback")
	}
}

func TestCompositeCompressor_BothFail(t *testing.T) {
	// Create failing compressors
	primary := &failingCompressor{name: "primary"}
	fallback := &failingCompressor{name: "fallback"}

	composite := NewCompositeCompressor(primary, fallback)

	tokenizer := &simpleTokenizer{}
	ctx := context.Background()

	messages := []CompressibleMessage{
		{Role: RoleAssistant, Content: "Message", Tokens: 100},
	}

	_, err := composite.Compress(ctx, messages, 50, tokenizer)
	if err == nil {
		t.Errorf("expected error when both strategies fail")
	}

	// Error should mention both strategies
	if !contains(err.Error(), "primary") || !contains(err.Error(), "fallback") {
		t.Errorf("expected error to mention both strategies, got: %v", err)
	}
}

func TestCompositeCompressor_Name(t *testing.T) {
	primary := NewDefaultLLMSummarizer(&mockLLMProvider{response: "ok"})
	fallback := NewDefaultHybridCompressor()

	composite := NewCompositeCompressor(primary, fallback)

	name := composite.Name()
	if !contains(name, "llm-summary") || !contains(name, "hybrid") {
		t.Errorf("expected name to include both strategies, got: %s", name)
	}
}

func TestNewLLMWithHybridFallback(t *testing.T) {
	mock := &mockLLMProvider{response: "Summary"}
	composite := NewDefaultLLMWithHybridFallback(mock)

	if composite.primary == nil {
		t.Errorf("expected primary strategy to be set")
	}

	if composite.fallback == nil {
		t.Errorf("expected fallback strategy to be set")
	}

	// Verify it compresses
	tokenizer := &simpleTokenizer{}
	ctx := context.Background()

	messages := generateTestMessages(100)

	// Current: 100 messages * ~110 tokens avg = 11K tokens
	// Target: 2000 tokens - needs compression
	compressed, err := composite.Compress(ctx, messages, 2000, tokenizer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(compressed) >= len(messages) {
		t.Errorf("expected compression, got %d messages (original: %d)", len(compressed), len(messages))
	}
}

// Helper types

type failingCompressor struct {
	name string
}

func (f *failingCompressor) Compress(ctx context.Context, messages []CompressibleMessage, targetTokens int, tokenizer Tokenizer) ([]CompressibleMessage, error) {
	return nil, fmt.Errorf("%s compressor failed", f.name)
}

func (f *failingCompressor) Name() string {
	return f.name
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
