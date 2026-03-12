package summarizer

import (
	"testing"
	"time"
)

func TestDefaultCacheConfig(t *testing.T) {
	config := DefaultCacheConfig()

	const (
		expectedMaxSize = 100
		expectedTTL     = time.Hour
	)

	if config.MaxSize != expectedMaxSize {
		t.Errorf("MaxSize = %d, want %d", config.MaxSize, expectedMaxSize)
	}

	if config.TTL != expectedTTL {
		t.Errorf("TTL = %v, want %v", config.TTL, expectedTTL)
	}
}

func TestNewCache(t *testing.T) {
	config := DefaultCacheConfig()
	cache := NewCache(config)

	if cache == nil {
		t.Fatal("NewCache returned nil")
	}

	if cache.Size() != 0 {
		t.Errorf("Size() = %d, want 0", cache.Size())
	}
}

func TestCache_Get_Miss(t *testing.T) {
	cache := NewCache(DefaultCacheConfig())

	result, ok := cache.Get("uncached content")

	if ok {
		t.Error("Get returned ok=true for uncached content")
	}

	if result != nil {
		t.Errorf("Get returned non-nil result: %v", result)
	}
}

func TestCache_SetAndGet_Hit(t *testing.T) {
	cache := NewCache(DefaultCacheConfig())
	content := "test content to cache"
	summary := &Result{
		Summary:          "cached summary",
		OriginalTokens:   100,
		SummaryTokens:    20,
		CompressionRatio: 0.2,
	}

	cache.Set(content, summary)

	if cache.Size() != 1 {
		t.Errorf("Size() = %d, want 1", cache.Size())
	}

	result, ok := cache.Get(content)

	if !ok {
		t.Error("Get returned ok=false for cached content")
	}

	if result == nil {
		t.Fatal("Get returned nil result for cached content")
	}

	if result.Summary != summary.Summary {
		t.Errorf("result.Summary = %q, want %q", result.Summary, summary.Summary)
	}
}

func TestCache_TTLExpiration(t *testing.T) {
	// Use very short TTL.
	config := CacheConfig{
		MaxSize: 100,
		TTL:     1 * time.Millisecond,
	}
	cache := NewCache(config)
	content := "test content"
	summary := &Result{Summary: "summary"}

	cache.Set(content, summary)

	// Wait for TTL to expire.
	time.Sleep(5 * time.Millisecond)

	result, ok := cache.Get(content)

	if ok {
		t.Error("Get returned ok=true for expired content")
	}

	if result != nil {
		t.Errorf("Get returned non-nil result for expired content: %v", result)
	}
}

func TestCache_LRUEviction(t *testing.T) {
	// Use small cache.
	const maxSize = 2

	config := CacheConfig{
		MaxSize: maxSize,
		TTL:     time.Hour,
	}
	cache := NewCache(config)

	// Add first entry.
	cache.Set("content1", &Result{Summary: "summary1"})

	// Add second entry.
	cache.Set("content2", &Result{Summary: "summary2"})

	// Access first entry (increases its access count).
	_, _ = cache.Get("content1")

	// Add third entry - should evict content2 (lower access count).
	cache.Set("content3", &Result{Summary: "summary3"})

	// Verify size is at max.
	if cache.Size() != maxSize {
		t.Errorf("Size() = %d, want %d", cache.Size(), maxSize)
	}

	// content1 should still be cached (had access).
	if _, ok := cache.Get("content1"); !ok {
		t.Error("content1 was evicted but had higher access count")
	}

	// content2 should be evicted (no access).
	if _, ok := cache.Get("content2"); ok {
		t.Error("content2 was not evicted but had lowest access count")
	}

	// content3 should be cached (just added).
	if _, ok := cache.Get("content3"); !ok {
		t.Error("content3 was evicted but was just added")
	}
}

func TestCache_Clear(t *testing.T) {
	cache := NewCache(DefaultCacheConfig())

	cache.Set("content1", &Result{Summary: "summary1"})
	cache.Set("content2", &Result{Summary: "summary2"})

	if cache.Size() != 2 {
		t.Errorf("Size() = %d, want 2", cache.Size())
	}

	cache.Clear()

	if cache.Size() != 0 {
		t.Errorf("Size() after Clear = %d, want 0", cache.Size())
	}

	// Verify entries are gone.
	if _, ok := cache.Get("content1"); ok {
		t.Error("content1 still exists after Clear")
	}
}
