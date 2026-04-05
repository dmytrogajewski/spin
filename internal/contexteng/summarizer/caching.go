package summarizer

import (
	"context"

	"github.com/dmytrogajewski/spin/internal/message"
)

// CachingSummarizer wraps a Summarizer with caching.
type CachingSummarizer struct {
	inner Summarizer
	cache *Cache
}

// NewCachingSummarizer creates a caching wrapper around a summarizer.
func NewCachingSummarizer(inner Summarizer, cache *Cache) *CachingSummarizer {
	return &CachingSummarizer{
		inner: inner,
		cache: cache,
	}
}

// Summarize implements Summarizer.Summarize with caching.
func (s *CachingSummarizer) Summarize(ctx context.Context, content string, opts Options) (*Result, error) {
	// Check cache first.
	if cached, ok := s.cache.Get(content); ok {
		return cached, nil
	}

	// Not cached, call inner summarizer.
	result, err := s.inner.Summarize(ctx, content, opts)
	if err != nil {
		return nil, err
	}

	// Cache the result.
	s.cache.Set(content, result)

	return result, nil
}

// SummarizeMessages implements Summarizer.SummarizeMessages.
// Note: Message summarization is not cached as message sequences vary.
func (s *CachingSummarizer) SummarizeMessages(ctx context.Context, messages []message.Message, opts Options) (*MessageResult, error) {
	return s.inner.SummarizeMessages(ctx, messages, opts)
}
