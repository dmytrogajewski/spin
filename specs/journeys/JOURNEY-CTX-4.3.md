# JOURNEY-CTX-4.3 — Session Index and History Gain Context

**Status**: Done
**Roadmap**: specs/ctx/ROADMAP.md -> 4.3
**Spec**: specs/ctx/SPEC.md -> CTX-024, CTX-025, CTX-028
**Note**: History.Save/Load already gained ctx in step 1.2.

## User Journey

Session index `Update`, `Remove`, and `Rebuild` perform file I/O without context. The `save()` method calls `AtomicWriteFile` with `context.Background()`. After this change, all index methods accept ctx, `save`/`load` check `ctx.Err()`, `MetadataScanner.ScanSessions` accepts ctx, and `AtomicWriteFile` receives the caller's ctx.

## Design Decisions

1. **Public methods get ctx**: `Update`, `Remove`, `Rebuild` accept ctx as first parameter.
2. **Internal methods get ctx**: `save(ctx)`, `load(ctx)` thread ctx through.
3. **MetadataScanner.ScanSessions(ctx)**: Interface gains ctx for filesystem scanning.
4. **Constructor**: `NewSessionIndex` gains ctx for initial `load`/`Rebuild`.
5. **No production callers**: Index is only used from tests — safe to change.

## DoD

- [x] `Update(ctx, ...)`, `Remove(ctx, ...)`, `Rebuild(ctx, ...)` accept ctx.
- [x] `save(ctx)` passes ctx to `AtomicWriteFile`; `load(ctx)` checks `ctx.Err()` before `os.ReadFile`.
- [x] `MetadataScanner.ScanSessions(ctx)` accepts ctx.
- [x] `NewSessionIndex(ctx, ...)` accepts ctx for initial load/rebuild.
- [x] All tests updated (15 NewSessionIndex, 14 Update, 3 Remove, 2 Rebuild calls, mock scanner).
- [x] `go vet ./...` clean. `make lint` clean. `go test ./...` full suite passing.

## Implementation

### Files Modified
- `internal/session/index.go` — `NewSessionIndex`, `Update`, `Remove`, `Rebuild` accept ctx; `save`/`load` thread ctx; `MetadataScanner.ScanSessions` accepts ctx; `AtomicWriteFile` receives caller's ctx.
- `internal/session/index_test.go` — all calls pass `t.Context()`; mock scanner accepts ctx.
