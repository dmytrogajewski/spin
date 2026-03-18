# JOURNEY-CTX-2.2 — TranscriptWriter Gains Context

**Status**: Done
**Roadmap**: specs/ctx/ROADMAP.md -> 2.2
**Spec**: specs/ctx/SPEC.md -> CTX-010
**Depends on**: JOURNEY-CTX-2.1 (flock helper)

## User Journey

During a conversation turn, `TranscriptWriter.Append` acquires an exclusive flock. If another process holds the lock, the current implementation blocks forever. After this change, both `Append` and `ReadAll` accept `context.Context` and use the context-aware flock helper, respecting cancellation and timeouts.

## Phases

### Phase 1: Append Gains Context
- Add `ctx context.Context` as first parameter.
- Replace `syscall.Flock(fd, LOCK_EX)` with `storage.FlockExclusiveWithContext(ctx, fd)`.
- Replace `syscall.Flock(fd, LOCK_UN)` with `storage.FlockUnlock(fd)`.
- Use `storage.SafeFlockFd` instead of local `safeFlockFd`.

### Phase 2: ReadAll Gains Context
- Add `ctx context.Context` as first parameter.
- Replace `syscall.Flock(fd, LOCK_SH)` with `storage.FlockSharedWithContext(ctx, fd)`.
- Replace `syscall.Flock(fd, LOCK_UN)` with `storage.FlockUnlock(fd)`.

### Phase 3: Remove Duplicated Code
- Remove local `safeFlockFd` — use `storage.SafeFlockFd` instead.
- Remove `syscall` import (no longer directly used).

## Design Decisions

1. **No production callers**: TranscriptWriter is only used in tests currently. Adding ctx is safe.
2. **Reuse flock helper**: Uses `storage.FlockExclusiveWithContext`/`FlockSharedWithContext` from step 2.1.
3. **Remove duplication**: `safeFlockFd` was duplicated between session and safety packages.

## DoD

- [x] `Append(ctx, msg)` uses context-aware exclusive flock.
- [x] `ReadAll(ctx)` uses context-aware shared flock.
- [x] Local `safeFlockFd` removed, uses `storage.SafeFlockFd`.
- [x] `syscall` import removed from transcript.go.
- [x] All 11 existing tests pass with ctx.
- [x] 2 new cancellation tests: `Append_CanceledContext`, `ReadAll_CanceledContext`.
- [x] `go vet ./...` clean.
- [x] `make lint` clean (zero errors in modified files).
- [x] `go test ./...` full suite passing.

## Implementation

### Files Modified
- `internal/session/transcript.go` — `Append` and `ReadAll` accept `ctx context.Context`; replaced `syscall.Flock` with `storage.FlockExclusiveWithContext`/`FlockSharedWithContext`/`FlockUnlock`; replaced local `safeFlockFd` with `storage.SafeFlockFd`; removed `syscall` import.
- `internal/session/transcript_test.go` — all `Append`/`ReadAll` calls pass ctx; 2 new cancellation tests added.
