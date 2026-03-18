# JOURNEY-CTX-2.3 — FilePolicyStore Uses Context for Flock

**Status**: Done
**Roadmap**: specs/ctx/ROADMAP.md -> 2.3
**Spec**: specs/ctx/SPEC.md -> CTX-007, CTX-020
**Depends on**: JOURNEY-CTX-2.1 (flock helper)

## User Journey

The approval/policy system persists global-scope policies to disk with advisory file locking. If another process holds the lock, `persistGlobalLocked` and `loadFromDisk` block forever. After this change, all flock calls use the context-aware helpers from the storage package, respecting cancellation and timeouts.

## Phases

### Phase 1: persistGlobalLocked Gains Context
- Accept `ctx context.Context` parameter.
- Replace `syscall.Flock(fd, LOCK_EX)` with `storage.FlockExclusiveWithContext(ctx, fd)`.
- Replace `syscall.Flock(fd, LOCK_UN)` with `storage.FlockUnlock(fd)`.
- Use `storage.SafeFlockFd` instead of local `safeFlockFd`.

### Phase 2: loadFromDisk Gains Context
- Accept `ctx context.Context` parameter.
- Replace `syscall.Flock(fd, LOCK_SH)` with `storage.FlockSharedWithContext(ctx, fd)`.

### Phase 3: Public Methods Use Their Context
- `Save`, `Delete`, `Clear`: rename `_ context.Context` to `ctx`, pass to `persistGlobalLocked`.
- `Get`, `List`: rename `_` to `ctx`, check `ctx.Err()` at entry.

### Phase 4: Constructor and Callers
- `NewFilePolicyStore` accepts `ctx` for initial `loadFromDisk`.
- Update 4 callers: `cmd/spin/approval.go`, `cmd/spin/approval_test.go`, `internal/safety/policy_file_store_test.go`, `internal/agent/builder.go`.

### Phase 5: Remove Duplicated Code
- Remove local `safeFlockFd` — use `storage.SafeFlockFd`.
- Remove `syscall` import.

## DoD

- [x] All flock calls use `storage.FlockExclusiveWithContext`/`FlockSharedWithContext`/`FlockUnlock`.
- [x] `persistGlobalLocked(ctx)` and `loadFromDisk(ctx)` accept ctx.
- [x] `Save`, `Delete`, `Clear` pass ctx to `persistGlobalLocked`.
- [x] `Get`, `List` check `ctx.Err()` at entry.
- [x] `NewFilePolicyStore(ctx, path, interval)` accepts ctx for initial load.
- [x] Local `safeFlockFd` removed — uses `storage.SafeFlockFd`.
- [x] `syscall` import removed.
- [x] All 4 callers updated.
- [x] 2 new cancellation tests: `Save_CanceledContext`, `Get_CanceledContext`.
- [x] Janitor uses `context.WithoutCancel(ctx)` to satisfy `contextcheck` linter.
- [x] `go vet ./...` clean.
- [x] `make lint` clean.
- [x] `go test ./...` full suite passing.

## Implementation

### Files Modified
- `internal/safety/policy_file_store.go` — all flock calls use storage helpers; `persistGlobalLocked` and `loadFromDisk` accept ctx; public methods use their ctx parameter; constructor accepts ctx; local `safeFlockFd` removed; `syscall` import removed.
- `internal/safety/policy_file_store_test.go` — all `NewFilePolicyStore` calls pass `t.Context()`; 2 new cancellation tests.
- `cmd/spin/approval.go` — `NewFilePolicyStore` call passes `context.Background()`.
- `cmd/spin/approval_test.go` — `NewFilePolicyStore` call passes `context.Background()`.
- `internal/agent/builder.go` — `NewFilePolicyStore` call passes `context.Background()`.
