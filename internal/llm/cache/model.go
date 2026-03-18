// Package cache provides a stale-while-revalidate disk-backed cache
// for LLM provider model capabilities.
package cache

import "time"

const (
	// DefaultContextWindow is the safe default context window size (128K tokens).
	DefaultContextWindow = 128_000
	// CacheTTL is the duration a cache entry is considered fresh.
	CacheTTL = 24 * time.Hour
	// BackgroundRefreshTimeout is the maximum duration for a background refresh operation.
	BackgroundRefreshTimeout = 30 * time.Second
	// cacheFilePerm is the file permission for cache files.
	cacheFilePerm = 0o600
	// cacheDirPerm is the directory permission for the cache directory.
	cacheDirPerm = 0o750
)

// ModelCapabilities describes a single model's capabilities.
type ModelCapabilities struct {
	ModelID         string  `json:"model_id"`
	Provider        string  `json:"provider"`
	ContextLength   int     `json:"context_length"`
	Vision          bool    `json:"vision"`
	Thinking        bool    `json:"thinking"`
	Streaming       bool    `json:"streaming"`
	FunctionCalling bool    `json:"function_calling"`
	InputPriceUSD   float64 `json:"input_price_usd,omitempty"`
	OutputPriceUSD  float64 `json:"output_price_usd,omitempty"`
}

// Entry wraps ModelCapabilities with a fetch timestamp for TTL computation.
type Entry struct {
	Capabilities ModelCapabilities `json:"capabilities"`
	FetchedAt    time.Time         `json:"fetched_at"`
}

// IsFresh returns true if the entry was fetched within the TTL window.
func (e Entry) IsFresh() bool {
	return time.Since(e.FetchedAt) < CacheTTL
}

// ProviderData holds all cached entries for a single provider.
type ProviderData struct {
	Version int              `json:"version"`
	Models  map[string]Entry `json:"models"`
}

// providerDataVersion is the current schema version for provider cache files.
const providerDataVersion = 1
