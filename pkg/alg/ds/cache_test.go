package ds

// Journey: specs/journeys/JOURNEY-R19.md.

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testClock provides a controllable clock for cache testing.
type testClock struct {
	now time.Time
}

func (tc *testClock) Now() time.Time { return tc.now }

func (tc *testClock) Advance(dur time.Duration) { tc.now = tc.now.Add(dur) }

// testTTL is the default TTL for test caches.
const testTTL = time.Minute

// testMaxEntries is the default max entries for test caches.
const testMaxEntries = 3

func TestCache_set_and_get(t *testing.T) {
	t.Parallel()

	cache := newTestCache(t)

	cache.Set("key1", "val1")

	got, ok := cache.Get("key1")
	require.True(t, ok)
	require.Equal(t, "val1", got)
}

func TestCache_get_missing_key(t *testing.T) {
	t.Parallel()

	cache := newTestCache(t)

	_, ok := cache.Get("nonexistent")
	require.False(t, ok)
}

func TestCache_ttl_expiry(t *testing.T) {
	t.Parallel()

	clock := &testClock{now: time.Now()}

	cache := NewCache[string, string](CacheConfig[string, string]{
		TTL:        testTTL,
		MaxEntries: testMaxEntries,
		Clock:      clock.Now,
	})

	cache.Set("key1", "val1")

	// Before expiry.
	got, ok := cache.Get("key1")
	require.True(t, ok)
	require.Equal(t, "val1", got)

	// After expiry.
	clock.Advance(testTTL + time.Second)

	_, ok = cache.Get("key1")
	require.False(t, ok)
}

func TestCache_count_eviction(t *testing.T) {
	t.Parallel()

	clock := &testClock{now: time.Now()}

	cache := NewCache[string, string](CacheConfig[string, string]{
		TTL:        testTTL,
		MaxEntries: testMaxEntries,
		Clock:      clock.Now,
	})

	cache.Set("a", "1")

	clock.Advance(time.Second)

	cache.Set("b", "2")

	clock.Advance(time.Second)

	cache.Set("c", "3")

	clock.Advance(time.Second)

	// Fourth entry should evict "a" (oldest).
	cache.Set("d", "4")

	_, ok := cache.Get("a")
	require.False(t, ok, "oldest entry should be evicted")

	got, ok := cache.Get("d")
	require.True(t, ok)
	require.Equal(t, "4", got)

	require.Equal(t, testMaxEntries, cache.Len())
}

func TestCache_overwrite_existing_key(t *testing.T) {
	t.Parallel()

	cache := newTestCache(t)

	cache.Set("key1", "old")
	cache.Set("key1", "new")

	got, ok := cache.Get("key1")
	require.True(t, ok)
	require.Equal(t, "new", got)
	require.Equal(t, 1, cache.Len())
}

func TestCache_delete(t *testing.T) {
	t.Parallel()

	cache := newTestCache(t)

	cache.Set("key1", "val1")
	cache.Delete("key1")

	_, ok := cache.Get("key1")
	require.False(t, ok)
	require.Equal(t, 0, cache.Len())
}

func TestCache_clear(t *testing.T) {
	t.Parallel()

	cache := newTestCache(t)

	cache.Set("a", "1")
	cache.Set("b", "2")
	cache.Clear()

	require.Equal(t, 0, cache.Len())
}

func TestCache_concurrent_access(t *testing.T) {
	t.Parallel()

	cache := newTestCache(t)
	concurrency := 50

	var wg sync.WaitGroup

	wg.Add(concurrency)

	for idx := range concurrency {
		go func() {
			defer wg.Done()

			key := string(rune('a' + idx%26))
			cache.Set(key, key)

			_, _ = cache.Get(key)
		}()
	}

	wg.Wait()

	// Just verify no panic or race; exact count depends on timing.
	assert.GreaterOrEqual(t, cache.Len(), 0)
}

func TestCache_int_keys(t *testing.T) {
	t.Parallel()

	cache := NewCache[int, string](CacheConfig[int, string]{
		TTL:        testTTL,
		MaxEntries: testMaxEntries,
		Clock:      time.Now,
	})

	cache.Set(42, "answer")

	got, ok := cache.Get(42)
	require.True(t, ok)
	require.Equal(t, "answer", got)
}

// newTestCache creates a string→string cache with default test config.
func newTestCache(t *testing.T) *Cache[string, string] {
	t.Helper()

	return NewCache[string, string](CacheConfig[string, string]{
		TTL:        testTTL,
		MaxEntries: testMaxEntries,
		Clock:      time.Now,
	})
}

// Journey: specs/journeys/JOURNEY-S7.md.

func TestCache_LRU_eviction(t *testing.T) {
	t.Parallel()

	clock := &testClock{now: time.Now()}

	cache := NewCache[string, string](CacheConfig[string, string]{
		TTL:        testTTL,
		MaxEntries: testMaxEntries,
		Eviction:   EvictLRU,
		Clock:      clock.Now,
	})

	cache.Set("a", "1")
	clock.Advance(time.Second)
	cache.Set("b", "2")
	clock.Advance(time.Second)
	cache.Set("c", "3")

	// Access "a" twice to boost its count; "b" stays at 0.
	cache.Get("a")
	cache.Get("a")

	clock.Advance(time.Second)

	// Fourth entry: should evict "b" (lowest access count = 0).
	cache.Set("d", "4")

	_, okB := cache.Get("b")
	require.False(t, okB, "LRU should evict least-accessed entry 'b'")

	_, okA := cache.Get("a")
	require.True(t, okA, "'a' should survive (highest access count)")

	_, okD := cache.Get("d")
	require.True(t, okD, "'d' should exist (just added)")
}

func TestCache_LRU_ties_broken_by_age(t *testing.T) {
	t.Parallel()

	clock := &testClock{now: time.Now()}

	cache := NewCache[string, string](CacheConfig[string, string]{
		TTL:        testTTL,
		MaxEntries: testMaxEntries,
		Eviction:   EvictLRU,
		Clock:      clock.Now,
	})

	cache.Set("old", "1")
	clock.Advance(time.Second)
	cache.Set("mid", "2")
	clock.Advance(time.Second)
	cache.Set("new", "3")
	clock.Advance(time.Second)

	// All access count 0; "old" should be evicted (oldest).
	cache.Set("extra", "4")

	_, okOld := cache.Get("old")
	require.False(t, okOld, "LRU tie-break should evict oldest")

	_, okNew := cache.Get("new")
	require.True(t, okNew)
}

func TestCache_FIFO_ignores_access_count(t *testing.T) {
	t.Parallel()

	clock := &testClock{now: time.Now()}

	cache := NewCache[string, string](CacheConfig[string, string]{
		TTL:        testTTL,
		MaxEntries: testMaxEntries,
		Eviction:   EvictFIFO,
		Clock:      clock.Now,
	})

	cache.Set("a", "1")
	clock.Advance(time.Second)
	cache.Set("b", "2")
	clock.Advance(time.Second)
	cache.Set("c", "3")

	// Access "a" many times (shouldn't matter for FIFO).
	cache.Get("a")
	cache.Get("a")
	cache.Get("a")

	clock.Advance(time.Second)

	// Eviction: FIFO evicts "a" (oldest), regardless of access count.
	cache.Set("d", "4")

	_, okA := cache.Get("a")
	require.False(t, okA, "FIFO should evict oldest 'a'")
}
