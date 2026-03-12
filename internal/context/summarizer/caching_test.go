package summarizer

import (
	"context"
	"testing"

	"github.com/dmytrogajewski/spin/internal/message"
)

func TestNewCachingSummarizer(t *testing.T) {
	inner := &mockSummarizer{}
	cache := NewCache(DefaultCacheConfig())

	cs := NewCachingSummarizer(inner, cache)

	if cs == nil {
		t.Fatal("NewCachingSummarizer returned nil")
	}

	// Verify it implements Summarizer.
	var _ Summarizer = cs
}

func TestCachingSummarizer_CacheHit(t *testing.T) {
	callCount := 0
	inner := &mockSummarizer{
		summarizeFunc: func(ctx context.Context, content string, opts Options) (*Result, error) {
			callCount++

			return &Result{Summary: "inner summary"}, nil
		},
	}
	cache := NewCache(DefaultCacheConfig())
	cs := NewCachingSummarizer(inner, cache)

	ctx := context.Background()
	content := "test content"

	// First call - should call inner.
	result1, err := cs.Summarize(ctx, content, Options{})
	if err != nil {
		t.Fatalf("First Summarize error: %v", err)
	}

	if result1.Summary != "inner summary" {
		t.Errorf("result1.Summary = %q, want %q", result1.Summary, "inner summary")
	}

	if callCount != 1 {
		t.Errorf("callCount after first call = %d, want 1", callCount)
	}

	// Second call - should use cache.
	result2, err := cs.Summarize(ctx, content, Options{})
	if err != nil {
		t.Fatalf("Second Summarize error: %v", err)
	}

	if result2.Summary != "inner summary" {
		t.Errorf("result2.Summary = %q, want %q", result2.Summary, "inner summary")
	}

	if callCount != 1 {
		t.Errorf("callCount after second call = %d, want 1 (cache hit)", callCount)
	}
}

func TestCachingSummarizer_MessagesNotCached(t *testing.T) {
	callCount := 0
	inner := &mockSummarizer{
		summarizeMessagesFunc: func(ctx context.Context, messages []message.Message, opts Options) (*MessageResult, error) {
			callCount++

			return &MessageResult{
				Summary: message.Message{Role: message.RoleAssistant, Content: "summary"},
			}, nil
		},
	}
	cache := NewCache(DefaultCacheConfig())
	cs := NewCachingSummarizer(inner, cache)

	ctx := context.Background()
	messages := []message.Message{{Role: message.RoleUser, Content: "test"}}

	// First call.
	_, _ = cs.SummarizeMessages(ctx, messages, Options{})

	if callCount != 1 {
		t.Errorf("callCount after first call = %d, want 1", callCount)
	}

	// Second call - should still call inner (messages not cached).
	_, _ = cs.SummarizeMessages(ctx, messages, Options{})

	if callCount != 2 {
		t.Errorf("callCount after second call = %d, want 2 (no caching)", callCount)
	}
}

// mockSummarizer defined in summarizer_test.go.
