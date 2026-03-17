# JOURNEY-R7.1 — Stale-While-Revalidate Provider Cache

**Status**: Done
**Roadmap**: specs/refactoring/opendev-gaps/ROADMAP.md → R-7.1

## User Journey

Developer starts Spin on an airplane. Today, provider initialization fails or uses hardcoded defaults. After this change, the cache from the last online session serves capabilities instantly, and a background refresh runs when connectivity returns.

## Phases

### Phase 1: Types
- `ModelCapabilities` struct: model ID, provider, context length, vision, thinking, pricing
- `CacheEntry` wraps `ModelCapabilities` with `FetchedAt` timestamp

### Phase 2: ProviderCache Core
- `NewProviderCache(cacheDir, fetcher)` creates cache with directory and fetcher interface
- `Get(provider, model)` returns cached capabilities with three-path strategy:
  - Fresh (<24h): return immediately
  - Stale (>24h): return + background refresh
  - Missing: synchronous fetch + fallback on error
- `ContextLength(provider, model)` convenience method

### Phase 3: Persistence
- Cache dir: `~/.spin/cache/<provider>.json` (one file per provider)
- Atomic writes via `storage.AtomicWriteFile()`
- JSON serialization with `FetchedAt` for TTL computation

### Phase 4: Graceful Degradation
- If all paths fail (no cache, fetch error), return safe defaults (128K context)
- `CapabilityFetcher` interface for testability
- Environment overrides: `SPIN_MODELS_DEV_PATH`, `SPIN_DISABLE_REMOTE_MODELS`

## Friction Points

| Point | Risk | Mitigation |
|-------|------|------------|
| Offline startup | No cache, no network | Return safe defaults (128K context) |
| Stale cache | Model capabilities changed | Background revalidation on next access |
| Concurrent access | Data race | sync.RWMutex |
| Disk corruption | Bad JSON | Re-fetch and rewrite |

## Design Decisions

1. **Three-path strategy**: Fresh → return, Stale → return + async refresh, Missing → sync fetch.
2. **One file per provider**: Keeps files small, avoids lock contention across providers.
3. **CapabilityFetcher interface**: Decouples cache from actual network calls.
4. **Safe defaults**: 128K context window, streaming on, function calling on — matches modern model baselines.
5. **Environment overrides**: `SPIN_MODELS_DEV_PATH` loads from local file, `SPIN_DISABLE_REMOTE_MODELS` skips network.

## DoD

- [x] `internal/llm/cache/types.go` — `ModelCapabilities`, `CacheEntry`, `ProviderData`
- [x] `internal/llm/cache/provider_cache.go` — `ProviderCache` with `Get()`, `ContextLength()`
- [x] Three-path strategy: fresh/stale/missing
- [x] Atomic writes via `storage.AtomicWriteFile()`
- [x] Graceful degradation with safe defaults (128K context)
- [x] `CapabilityFetcher` interface for testability
- [x] Unit tests (≥90% coverage)
- [x] `go vet` and `make lint` clean

## Implementation

### Files Created
- `internal/llm/cache/types.go` — Types and constants
- `internal/llm/cache/provider_cache.go` — Cache implementation
- `internal/llm/cache/provider_cache_test.go` — Unit tests

### Files Modified
- `specs/refactoring/opendev-gaps/ROADMAP.md` — R-7.1 marked Done
