# JOURNEY-CTX-1.1 — AtomicWriteFile Gains Context

**Status**: Done
**Roadmap**: specs/ctx/ROADMAP.md -> 1.1
**Spec**: specs/ctx/SPEC.md -> CTX-009

## User Journey

Developer calls `storage.AtomicWriteFile` from a request-scoped path (conversation turn, session save, playbook persist). If the filesystem hangs (NFS, FUSE, or any slow block device), the goroutine blocks forever with no way to cancel. After this change, `AtomicWriteFile` accepts `context.Context` and respects caller cancellation at each I/O boundary.

## Phases

### Phase 1: Signature Change
- Add `ctx context.Context` as first parameter to `AtomicWriteFile`.
- Insert `ctx.Err()` guard checks before `os.CreateTemp`, `tmpFile.Write`, and `os.Rename`.
- Ensure temp file cleanup on cancellation (no leaked `.atomic-*` files).

### Phase 2: Caller Migration
- Update all 6 callers to pass context:
  - `internal/storage/store.go` (FileStore.Save) — no ctx available yet, use `context.Background()`
  - `internal/llm/cache/provider_cache.go` (ProviderCache.persist) — no ctx in method, use `context.Background()`
  - `internal/config/mcp_manager.go` (MCPConfigStore.writeConfig) — no ctx in method, use `context.Background()`
  - `internal/session/index.go` (Index.save) — no ctx in method, use `context.Background()`
  - `internal/memory/persistent.go` (PersistentStore.Put) — has `_ context.Context`, use `context.Background()`
  - `internal/ace/playbook/storage.go` (Playbook.Save) — no ctx in method, use `context.Background()`
- Each `context.Background()` becomes a future fix target in subsequent roadmap items.

### Phase 3: Test Coverage
- Update existing tests to pass `context.Background()`.
- Add cancellation tests: cancelled context before write, after temp create.
- Verify temp file cleanup on cancellation.

## Friction Points

| Point | Risk | Mitigation |
|-------|------|------------|
| Signature change breaks all callers | Compilation failure | Update all callers in the same change |
| Temp file leak on cancellation | Disk space waste | `defer os.Remove(tmpPath)` on error paths; test verifies cleanup |
| Performance overhead of ctx.Err() | Negligible | ctx.Err() is a single atomic read, ~1ns |
| NFS/FUSE mid-write cancellation | Partial temp file | Temp file cleaned up; target file untouched |

## Design Decisions

1. **Guard-check pattern**: Use `ctx.Err()` checks before each I/O step rather than wrapping I/O in goroutines. Simpler, no goroutine overhead, sufficient for the use case (we cannot cancel a blocking syscall, but we can refuse to start the next step).
2. **context.Background() for callers without ctx**: Future roadmap items (1.2, 3.4, 4.3, 4.4) will thread proper context through these callers. Using `context.Background()` preserves existing behavior.
3. **No new dependencies**: Pure stdlib context usage.

## DoD

- [x] `AtomicWriteFile` signature: `func AtomicWriteFile(ctx context.Context, path string, data []byte, perm os.FileMode) error`
- [x] `ctx.Err()` checked before CreateTemp, Write, and Rename
- [x] All 6 callers updated and compiling
- [x] Existing tests updated to pass `context.Background()`
- [x] New test: canceled context before write returns `context.Canceled`
- [x] New test: no temp file leak on cancellation
- [x] `go vet ./...` clean
- [x] `make lint` clean (zero errors in modified files)
- [x] `go test ./internal/storage/...` passing
- [x] `go test ./...` full suite passing

## Implementation

### Files Modified
- `internal/storage/atomic.go` — added `ctx context.Context` parameter, 3 guard checks
- `internal/storage/atomic_test.go` — updated existing tests, added 3 new context tests
- `internal/storage/store.go` — updated `FileStore.Save` caller to pass `context.Background()`
- `internal/llm/cache/provider_cache.go` — updated `persist` caller to pass `context.Background()`
- `internal/config/mcp_manager.go` — updated `writeConfig` caller to pass `context.Background()`
- `internal/session/index.go` — updated `save` caller to pass `context.Background()`
- `internal/memory/persistent.go` — updated `Put` caller to pass `context.Background()`
- `internal/ace/playbook/storage.go` — updated `Save` caller to pass `context.Background()`
