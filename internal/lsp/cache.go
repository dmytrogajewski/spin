package lsp

import (
	"crypto/md5" //nolint:gosec // MD5 is used for content hashing, not security.
	"encoding/hex"
	"encoding/json"
	"sync"
)

// CacheEntry holds a cached response alongside its content hash.
type CacheEntry struct {
	ContentHash string          `json:"content_hash"`
	RawResponse json.RawMessage `json:"raw_response"`
	Symbols     []Symbol        `json:"symbols,omitempty"`
}

// Cache provides a two-level response cache for LSP results.
//   - L1: raw JSON-RPC responses keyed by (method, URI, content hash).
//   - L2: processed symbols keyed by (URI, content hash).
//
// Content hash invalidation: if the file content changes, the hash changes,
// and the old entry is no longer returned.
type Cache struct {
	mu      sync.RWMutex
	entries map[string]*CacheEntry
}

// NewCache creates an empty response cache.
func NewCache() *Cache {
	return &Cache{
		entries: make(map[string]*CacheEntry),
	}
}

// GetRaw returns the cached raw response for the given method, URI, and content hash.
// Returns nil if not found or if the content hash doesn't match.
func (c *Cache) GetRaw(method, uri, contentHash string) json.RawMessage {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := rawKey(method, uri)
	entry, ok := c.entries[key]

	if !ok || entry.ContentHash != contentHash {
		return nil
	}

	return entry.RawResponse
}

// PutRaw stores a raw response in the L1 cache.
func (c *Cache) PutRaw(method, uri, contentHash string, response json.RawMessage) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := rawKey(method, uri)
	c.entries[key] = &CacheEntry{
		ContentHash: contentHash,
		RawResponse: response,
	}
}

// GetSymbols returns cached symbols for the given URI and content hash.
// Returns nil if not found or if the content hash doesn't match.
func (c *Cache) GetSymbols(uri, contentHash string) []Symbol {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := symbolKey(uri)
	entry, ok := c.entries[key]

	if !ok || entry.ContentHash != contentHash {
		return nil
	}

	return entry.Symbols
}

// PutSymbols stores processed symbols in the L2 cache.
func (c *Cache) PutSymbols(uri, contentHash string, symbols []Symbol) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := symbolKey(uri)
	c.entries[key] = &CacheEntry{
		ContentHash: contentHash,
		Symbols:     symbols,
	}
}

// Invalidate removes all cache entries for the given URI.
func (c *Cache) Invalidate(uri string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for key := range c.entries {
		if containsURI(key, uri) {
			delete(c.entries, key)
		}
	}
}

// Size returns the number of entries in the cache.
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.entries)
}

// ContentHash computes an MD5 hash of the given content for cache keying.
func ContentHash(content []byte) string {
	sum := md5.Sum(content) //nolint:gosec // MD5 is used for content hashing, not security.

	return hex.EncodeToString(sum[:])
}

// rawKey builds the cache key for L1 entries.
func rawKey(method, uri string) string {
	return "raw:" + method + ":" + uri
}

// symbolKey builds the cache key for L2 entries.
func symbolKey(uri string) string {
	return "sym:" + uri
}

// containsURI checks if a cache key references the given URI.
func containsURI(key, uri string) bool {
	// Keys end with ":uri", so suffix match works.
	return len(key) > len(uri) && key[len(key)-len(uri):] == uri
}
