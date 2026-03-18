# JOURNEY-CTX-3.4 — PersistentStore Honors Its Context Parameter

**Status**: Done
**Roadmap**: specs/ctx/ROADMAP.md -> 3.4
**Spec**: specs/ctx/SPEC.md -> CTX-011, CTX-012, CTX-026

## User Journey

PersistentStore methods accept `context.Context` but ignore it (parameter named `_`). A large memory store means hundreds of file reads with no cancellation. After this change, all methods check `ctx.Err()` before I/O, `Search` checks inside its loop, and the constructor accepts ctx for startup index rebuild.

## Design Decisions

1. **ctx.Err() guard pattern**: Check before each I/O step, consistent with the rest of the codebase.
2. **Search loop check**: Check `ctx.Err()` every iteration since each iteration may read a file.
3. **Constructor ctx**: `NewPersistentStore(ctx, basePath)` passes ctx to `rebuildIndex` which checks it in the walk callback.
4. **Pass ctx to AtomicWriteFile**: `Put` now passes its ctx instead of `context.Background()`.

## DoD

- [x] All methods use their ctx parameter (`Put`, `Get`, `Delete`, `List`, `Search`).
- [x] `NewPersistentStore(ctx, basePath)` accepts ctx.
- [x] `rebuildIndex(ctx)` checks ctx in walk callback.
- [x] `Put` passes ctx to `AtomicWriteFile`.
- [x] `Search` checks `ctx.Err()` inside the loop.
- [x] 2 cancellation tests pass (`Put_CanceledContext`, `Search_CanceledContext`).
- [x] All 8 callers updated.
- [x] `go vet ./...` clean.
- [x] `make lint` clean (zero errors in modified files).
- [x] `go test ./...` full suite passing.

## Implementation

### Files Modified
- `internal/memory/persistent.go` — all `_ context.Context` renamed to `ctx`; `ctx.Err()` checks added to Put, Get, Delete, List, Search; `rebuildIndex` accepts ctx with check in walk callback; `Put` passes ctx to `AtomicWriteFile`; `NewPersistentStore` accepts ctx.
- `internal/memory/persistent_test.go` — all `NewPersistentStore` calls pass `t.Context()`; 2 new cancellation tests.
- `internal/conversation/memory.go` — `NewPersistentStore` call passes `context.Background()`.
- `internal/conversation/memory_test.go` — `NewPersistentStore` calls pass `t.Context()`.
- `internal/memory/handoff_test.go` — `NewPersistentStore` calls pass `t.Context()`.
- `internal/memory/offloader_test.go` — `NewPersistentStore` calls pass `t.Context()`.
- `internal/tools/memory_tool_test.go` — `NewPersistentStore` calls pass `t.Context()`.
