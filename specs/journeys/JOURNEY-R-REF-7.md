# JOURNEY-R-REF-7: Migrate Lazy Init to syncmap.GetOrCreate

**Roadmap Item:** R-REF-7

## Summary

Added `syncmap.GetOrCreateErr(key K, create func() (V, error)) (V, error)` to support error-returning factory functions. Migrated `conversation/manager.go` to use it, removing the manual `createMu` mutex and double-checked locking.

`lsp/manager.go` not migrated — its pattern is get-or-replace-if-dead (health check on existing value), not create-if-absent.

## Acceptance Criteria

- [x] `GetOrCreateErr` added to `syncmap.Map`
- [x] 3 tests added (miss-success, miss-error, hit)
- [x] `conversation/manager.go` migrated, `createMu` removed
- [x] All tests pass

## Implementation

- **Modified:** `pkg/alg/ds/syncmap/syncmap_map.go` — added `GetOrCreateErr`
- **Modified:** `pkg/alg/ds/syncmap/syncmap_map_test.go` — added 3 tests
- **Modified:** `internal/conversation/manager.go` — uses `GetOrCreateErr`, removed `createMu` + `sync` import
