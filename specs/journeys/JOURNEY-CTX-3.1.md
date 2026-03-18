# JOURNEY-CTX-3.1 — ProviderCache Detaches Background Refresh Context

**Status**: Done
**Roadmap**: specs/ctx/ROADMAP.md -> 3.1
**Spec**: specs/ctx/SPEC.md -> CTX-003

## User Journey

LLM provider cache uses stale-while-revalidate: when a cache entry is stale, it returns the stale data immediately and triggers a background refresh. Currently the background goroutine receives the caller's context, which gets cancelled when the caller returns — defeating the purpose of background refresh. After this change, the background refresh uses a detached context with its own timeout.

## Design Decisions

1. **context.WithoutCancel**: Detaches from parent cancellation while preserving values (trace IDs, etc.).
2. **30s timeout**: Background refresh should not run forever. 30s is generous for a single API call.
3. **Constant for timeout**: `BackgroundRefreshTimeout` exported for testability.

## DoD

- [x] Background refresh uses `context.WithoutCancel(ctx)` + `context.WithTimeout` (30s).
- [x] Caller cancellation does not abort background refresh.
- [x] Test: cancel caller ctx, background refresh still completes with live context.
- [x] `BackgroundRefreshTimeout` constant exported for documentation.
- [x] `go vet ./...` clean.
- [x] `make lint` clean (zero errors in modified files).
- [x] `go test ./...` full suite passing.

## Implementation

### Files Modified
- `internal/llm/cache/model.go` — added `BackgroundRefreshTimeout` constant (30s).
- `internal/llm/cache/provider_cache.go` — `Get` stale path: replaced `go pc.refreshInBackground(ctx, ...)` with detached context using `context.WithoutCancel(ctx)` + `context.WithTimeout`.
- `internal/llm/cache/provider_cache_test.go` — added `TestProviderCache_BackgroundRefresh_SurvivesCallerCancellation` with `contextAwareFetcher` that verifies the background refresh context is alive even after caller cancellation.
