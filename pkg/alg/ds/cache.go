package ds

import (
	"sync"
	"time"
)

// EvictionPolicy controls how entries are evicted when the cache is full.
type EvictionPolicy int

const (
	// EvictFIFO evicts the oldest entry by creation time (default).
	EvictFIFO EvictionPolicy = iota
	// EvictLRU evicts the least-recently-used entry by access count,
	// breaking ties by oldest creation time.
	EvictLRU
)

// CacheConfig configures a [Cache] instance.
type CacheConfig[Key comparable, Val any] struct {
	// TTL is the time-to-live for cache entries.
	TTL time.Duration
	// MaxEntries is the maximum number of entries before eviction.
	MaxEntries int
	// Clock provides the current time. Use [time.Now] in production.
	Clock func() time.Time
	// Eviction controls the eviction policy. Defaults to [EvictFIFO].
	Eviction EvictionPolicy
}

// cacheEntry holds a value alongside its expiration and creation time.
type cacheEntry[Val any] struct {
	value       Val
	expiresAt   time.Time
	createdAt   time.Time
	accessCount int
}

// Cache is a generic TTL + count-based eviction cache.
// Thread-safe via [sync.RWMutex].
type Cache[Key comparable, Val any] struct {
	entries map[Key]*cacheEntry[Val]
	config  CacheConfig[Key, Val]
	mu      sync.RWMutex
}

// NewCache creates a new cache with the given configuration.
func NewCache[Key comparable, Val any](config CacheConfig[Key, Val]) *Cache[Key, Val] {
	if config.Clock == nil {
		config.Clock = time.Now
	}

	return &Cache[Key, Val]{
		entries: make(map[Key]*cacheEntry[Val]),
		config:  config,
	}
}

// Get returns the value for key if it exists and is not expired.
// Returns (zero, false) for missing or expired entries.
// Increments access count for LRU tracking.
func (c *Cache[Key, Val]) Get(key Key) (Val, bool) {
	c.mu.Lock()
	entry, ok := c.entries[key]

	if !ok {
		c.mu.Unlock()

		var zero Val

		return zero, false
	}

	// Check TTL expiry.
	if c.config.Clock().After(entry.expiresAt) {
		delete(c.entries, key)
		c.mu.Unlock()

		var zero Val

		return zero, false
	}

	entry.accessCount++
	c.mu.Unlock()

	return entry.value, true
}

// Set stores a value with TTL expiration. Evicts an entry
// if the cache exceeds MaxEntries (using the configured eviction policy).
func (c *Cache[Key, Val]) Set(key Key, value Val) {
	now := c.config.Clock()

	c.mu.Lock()
	defer c.mu.Unlock()

	// Overwrite existing: don't count as new entry for eviction.
	if _, exists := c.entries[key]; !exists {
		c.evictIfNeeded()
	}

	c.entries[key] = &cacheEntry[Val]{
		value:     value,
		expiresAt: now.Add(c.config.TTL),
		createdAt: now,
	}
}

// Delete removes a key from the cache.
func (c *Cache[Key, Val]) Delete(key Key) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.entries, key)
}

// Clear removes all entries from the cache.
func (c *Cache[Key, Val]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[Key]*cacheEntry[Val])
}

// Len returns the number of entries in the cache (including potentially expired ones).
func (c *Cache[Key, Val]) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.entries)
}

// evictIfNeeded removes one entry if at capacity.
// Must be called with write lock held.
func (c *Cache[Key, Val]) evictIfNeeded() {
	if c.config.MaxEntries <= 0 || len(c.entries) < c.config.MaxEntries {
		return
	}

	switch c.config.Eviction {
	case EvictLRU:
		c.evictLRU()
	case EvictFIFO:
		c.evictFIFO()
	default:
		c.evictFIFO()
	}
}

// evictFIFO removes the oldest entry by creation time.
func (c *Cache[Key, Val]) evictFIFO() {
	c.evictByComparator(func(candidate, victim *cacheEntry[Val]) bool {
		return candidate.createdAt.Before(victim.createdAt)
	})
}

// evictLRU removes the least-recently-used entry (lowest access count,
// ties broken by oldest creation time).
func (c *Cache[Key, Val]) evictLRU() {
	c.evictByComparator(func(candidate, victim *cacheEntry[Val]) bool {
		return candidate.accessCount < victim.accessCount ||
			(candidate.accessCount == victim.accessCount && candidate.createdAt.Before(victim.createdAt))
	})
}

// evictByComparator removes the entry selected by the comparator.
// The comparator returns true if candidate should replace the current victim.
func (c *Cache[Key, Val]) evictByComparator(shouldReplace func(candidate, victim *cacheEntry[Val]) bool) {
	var victimKey Key

	var victim *cacheEntry[Val]

	for key, entry := range c.entries {
		if victim == nil || shouldReplace(entry, victim) {
			victimKey = key
			victim = entry
		}
	}

	if victim != nil {
		delete(c.entries, victimKey)
	}
}
