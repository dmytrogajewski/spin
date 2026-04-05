# JOURNEY-R-REF-14: Migrate summarizer.Cache to ds.Cache

**Roadmap Item:** R-REF-14
**Spec:** [specs/ref/SPEC.md](../ref/SPEC.md) — Cluster 4: LRU/TTL Cache Consolidation

## Summary

Replaced the custom cache implementation in `internal/contexteng/summarizer/cache.go` with composition over `pkg/alg/ds.Cache[string, *Result]`. Removed:
- Custom `evictLRU()` method (82 lines of LRU eviction logic)
- `CachedSummary` type (metadata struct)
- Manual `sync.RWMutex` locking (delegated to `ds.Cache`)

The public API (`Get`, `Set`, `Size`, `Clear`, `DefaultCacheConfig`, `CacheConfig`) is unchanged. All 7 existing tests pass without modification.

## Acceptance Criteria

- [x] `cache.go` composes `ds.Cache[string, *Result]` with LRU eviction
- [x] Custom LRU eviction code removed
- [x] `CachedSummary` type removed (no external users)
- [x] All 7 existing tests pass unchanged
- [x] No new lint issues

## Implementation

- **Modified:** `internal/contexteng/summarizer/cache.go` — rewritten to compose `ds.Cache`
