package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/dmytrogajewski/spin/internal/storage"
)

// CapabilityFetcher retrieves model capabilities from a remote source.
type CapabilityFetcher interface {
	FetchCapabilities(ctx context.Context, provider, model string) (ModelCapabilities, error)
}

// Option configures a ProviderCache.
type Option func(*ProviderCache)

// WithTimeFunc sets a custom time function for testing.
func WithTimeFunc(fn func() time.Time) Option {
	return func(pc *ProviderCache) {
		pc.now = fn
	}
}

// ProviderCache implements a stale-while-revalidate disk-backed cache for model capabilities.
// Thread-safe via [sync.RWMutex].
type ProviderCache struct {
	mu       sync.RWMutex
	cacheDir string
	fetcher  CapabilityFetcher
	data     map[string]*ProviderData // provider name → data.
	now      func() time.Time
}

// NewProviderCache creates a new provider cache backed by the given directory.
// The fetcher is used for synchronous and background capability lookups.
// A nil fetcher disables remote fetching (cache-only mode).
func NewProviderCache(cacheDir string, fetcher CapabilityFetcher, opts ...Option) (*ProviderCache, error) {
	if err := os.MkdirAll(cacheDir, cacheDirPerm); err != nil {
		return nil, fmt.Errorf("create cache directory: %w", err)
	}

	pc := &ProviderCache{
		cacheDir: cacheDir,
		fetcher:  fetcher,
		data:     make(map[string]*ProviderData),
		now:      time.Now,
	}

	for _, opt := range opts {
		opt(pc)
	}

	return pc, nil
}

// Get returns cached model capabilities using a three-path strategy:
//   - Fresh (<24h): return immediately.
//   - Stale (>24h): return stale data + trigger background refresh.
//   - Missing: synchronous fetch; on error, return safe defaults.
func (pc *ProviderCache) Get(ctx context.Context, provider, model string) ModelCapabilities {
	pc.mu.RLock()
	entry, found := pc.lookupEntry(provider, model)
	pc.mu.RUnlock()

	if found && pc.isFresh(entry) {
		return entry.Capabilities
	}

	if found {
		// Stale — return immediately and refresh in background.
		go pc.refreshInBackground(ctx, provider, model)

		return entry.Capabilities
	}

	// Missing — synchronous fetch.
	return pc.fetchOrDefault(ctx, provider, model)
}

// ContextLength returns the context window size for the given model.
// Returns DefaultContextWindow if not cached.
func (pc *ProviderCache) ContextLength(ctx context.Context, provider, model string) int {
	caps := pc.Get(ctx, provider, model)
	if caps.ContextLength <= 0 {
		return DefaultContextWindow
	}

	return caps.ContextLength
}

// Put stores a cache entry for the given provider and model.
func (pc *ProviderCache) Put(provider, model string, caps ModelCapabilities) error {
	pc.mu.Lock()

	pd := pc.ensureProviderData(provider)
	pd.Models[model] = Entry{
		Capabilities: caps,
		FetchedAt:    pc.now(),
	}

	pc.mu.Unlock()

	return pc.persistProvider(provider)
}

// Load reads cached data for a provider from disk.
func (pc *ProviderCache) Load(provider string) error {
	path := pc.providerPath(provider)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		return fmt.Errorf("read cache file: %w", err)
	}

	var pd ProviderData
	if jsonErr := json.Unmarshal(data, &pd); jsonErr != nil {
		return fmt.Errorf("unmarshal cache data: %w", jsonErr)
	}

	pc.mu.Lock()
	pc.data[provider] = &pd
	pc.mu.Unlock()

	return nil
}

// isFresh checks if a cache entry is within the TTL window.
func (pc *ProviderCache) isFresh(entry Entry) bool {
	return pc.now().Sub(entry.FetchedAt) < CacheTTL
}

// lookupEntry finds a cache entry. Caller must hold at least RLock.
func (pc *ProviderCache) lookupEntry(provider, model string) (Entry, bool) {
	pd, ok := pc.data[provider]
	if !ok {
		// Try loading from disk.
		pc.mu.RUnlock()

		if loadErr := pc.Load(provider); loadErr != nil {
			pc.mu.RLock()

			return Entry{}, false
		}

		pc.mu.RLock()

		pd, ok = pc.data[provider]
		if !ok {
			return Entry{}, false
		}
	}

	entry, found := pd.Models[model]

	return entry, found
}

// ensureProviderData returns or creates the ProviderData for a provider.
// Caller must hold write lock.
func (pc *ProviderCache) ensureProviderData(provider string) *ProviderData {
	pd, ok := pc.data[provider]
	if !ok {
		pd = &ProviderData{
			Version: providerDataVersion,
			Models:  make(map[string]Entry),
		}

		pc.data[provider] = pd
	}

	return pd
}

// persistProvider writes the provider data to disk atomically.
func (pc *ProviderCache) persistProvider(provider string) error {
	pc.mu.RLock()

	pd, ok := pc.data[provider]
	if !ok {
		pc.mu.RUnlock()

		return nil
	}

	data, err := json.MarshalIndent(pd, "", "  ")

	pc.mu.RUnlock()

	if err != nil {
		return fmt.Errorf("marshal cache data: %w", err)
	}

	path := pc.providerPath(provider)

	if writeErr := storage.AtomicWriteFile(path, data, cacheFilePerm); writeErr != nil {
		return fmt.Errorf("write cache file: %w", writeErr)
	}

	return nil
}

// providerPath returns the file path for a provider's cache file.
func (pc *ProviderCache) providerPath(provider string) string {
	return filepath.Join(pc.cacheDir, provider+".json")
}

// refreshInBackground fetches fresh capabilities and updates the cache.
func (pc *ProviderCache) refreshInBackground(ctx context.Context, provider, model string) {
	if pc.fetcher == nil {
		return
	}

	caps, err := pc.fetcher.FetchCapabilities(ctx, provider, model)
	if err != nil {
		return
	}

	_ = pc.Put(provider, model, caps)
}

// fetchOrDefault attempts a synchronous fetch; returns safe defaults on failure.
func (pc *ProviderCache) fetchOrDefault(ctx context.Context, provider, model string) ModelCapabilities {
	if pc.fetcher == nil {
		return safeDefaults(provider, model)
	}

	caps, err := pc.fetcher.FetchCapabilities(ctx, provider, model)
	if err != nil {
		return safeDefaults(provider, model)
	}

	_ = pc.Put(provider, model, caps)

	return caps
}

// safeDefaults returns conservative defaults when no data is available.
func safeDefaults(provider, model string) ModelCapabilities {
	return ModelCapabilities{
		ModelID:         model,
		Provider:        provider,
		ContextLength:   DefaultContextWindow,
		Streaming:       true,
		FunctionCalling: true,
	}
}
