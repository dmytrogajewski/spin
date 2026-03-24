package summarizer

import (
	"time"

	"github.com/dmytrogajewski/spin/pkg/alg/ds"
	"github.com/dmytrogajewski/spin/pkg/alg/hashx"
)

const defaultCacheMaxSize = 100

// CacheConfig configures the summary cache.
type CacheConfig struct {
	// MaxSize is the maximum number of cached summaries.
	MaxSize int

	// TTL is the time-to-live for cached entries.
	TTL time.Duration
}

// DefaultCacheConfig returns sensible default configuration.
func DefaultCacheConfig() CacheConfig {
	return CacheConfig{
		MaxSize: defaultCacheMaxSize,
		TTL:     time.Hour,
	}
}

// Cache provides caching for summaries.
// Delegates storage and eviction to [ds.Cache] with LRU policy.
type Cache struct {
	inner *ds.Cache[string, *Result]
}

// NewCache creates a new summary cache.
func NewCache(config CacheConfig) *Cache {
	return &Cache{
		inner: ds.NewCache(ds.CacheConfig[string, *Result]{
			TTL:        config.TTL,
			MaxEntries: config.MaxSize,
			Eviction:   ds.EvictLRU,
		}),
	}
}

// Size returns the number of cached entries.
func (c *Cache) Size() int {
	return c.inner.Len()
}

// Clear removes all cached entries.
func (c *Cache) Clear() {
	c.inner.Clear()
}

// Get retrieves a cached summary if available and not expired.
func (c *Cache) Get(content string) (*Result, bool) {
	hash := hashContent(content)

	return c.inner.Get(hash)
}

// Set stores a summary in the cache.
func (c *Cache) Set(content string, summary *Result) {
	hash := hashContent(content)
	c.inner.Set(hash, summary)
}

func hashContent(content string) string {
	return hashx.SHA256Hex([]byte(content))
}
