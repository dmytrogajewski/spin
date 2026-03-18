# JOURNEY-CTX-3.2 — OpenAI Pagination Respects Context

**Status**: Done
**Roadmap**: specs/ctx/ROADMAP.md -> 3.2
**Spec**: specs/ctx/SPEC.md -> CTX-004

## User Journey

Listing OpenAI models pages through results. If the user cancels mid-pagination, HTTP calls continue because the pagination loop ignores context. After this change, the loop checks `ctx.Err()` each iteration and handles `GetNextPage()` errors.

## Design Decisions

1. **ctx.Err() check at loop start**: Fast exit before making another HTTP call.
2. **Handle GetNextPage error**: Previously discarded with `_`. Now returns the error to the caller.

## DoD

- [x] Pagination loop checks `ctx.Err()` at start of each iteration.
- [x] `GetNextPage()` errors are handled and returned (no longer discarded with `_`).
- [x] `go vet ./...` clean.
- [x] `make lint` clean.
- [x] `go test ./...` full suite passing.

## Implementation

### Files Modified
- `internal/llm/openai/provider.go` — `Models` pagination loop: added `ctx.Err()` check, handled `GetNextPage()` error.
