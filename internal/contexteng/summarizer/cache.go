package summarizer

import (
	"sync"
	"time"

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

// CachedSummary contains a cached summary with metadata.
type CachedSummary struct {
	Key         string
	Summary     *Result
	ContentHash string
	CreatedAt   time.Time
	AccessCount int
}

// Cache provides caching for summaries.
type Cache struct {
	cache  map[string]*CachedSummary
	config CacheConfig
	mu     sync.RWMutex
}

// NewCache creates a new summary cache.
func NewCache(config CacheConfig) *Cache {
	return &Cache{
		cache:  make(map[string]*CachedSummary),
		config: config,
	}
}

// Size returns the number of cached entries.
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.cache)
}

// Clear removes all cached entries.
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache = make(map[string]*CachedSummary)
}

// Get retrieves a cached summary if available and not expired.
func (c *Cache) Get(content string) (*Result, bool) {
	hash := hashContent(content)

	c.mu.RLock()
	cached, ok := c.cache[hash]
	c.mu.RUnlock()

	if !ok {
		return nil, false
	}

	// Check TTL.
	if time.Since(cached.CreatedAt) > c.config.TTL {
		c.mu.Lock()
		delete(c.cache, hash)
		c.mu.Unlock()

		return nil, false
	}

	// Update access count.
	c.mu.Lock()
	cached.AccessCount++
	c.mu.Unlock()

	return cached.Summary, true
}

// Set stores a summary in the cache.
func (c *Cache) Set(content string, summary *Result) {
	hash := hashContent(content)

	c.mu.Lock()
	defer c.mu.Unlock()

	// Evict if at capacity.
	if len(c.cache) >= c.config.MaxSize {
		c.evictLRU()
	}

	c.cache[hash] = &CachedSummary{
		Key:         hash,
		Summary:     summary,
		ContentHash: hash,
		CreatedAt:   time.Now(),
		AccessCount: 0,
	}
}

// evictLRU removes the least recently used entry.
// Must be called with lock held.
func (c *Cache) evictLRU() {
	if len(c.cache) == 0 {
		return
	}

	var lruKey string

	lruCount := -1

	var lruTime time.Time

	for key, entry := range c.cache {
		if lruCount == -1 || entry.AccessCount < lruCount ||
			(entry.AccessCount == lruCount && entry.CreatedAt.Before(lruTime)) {
			lruKey = key
			lruCount = entry.AccessCount
			lruTime = entry.CreatedAt
		}
	}

	if lruKey != "" {
		delete(c.cache, lruKey)
	}
}

func hashContent(content string) string {
	return hashx.SHA256Hex([]byte(content))
}
