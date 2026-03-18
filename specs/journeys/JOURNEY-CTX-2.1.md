# JOURNEY-CTX-2.1 — Context-Aware Flock Helper

**Status**: Done
**Roadmap**: specs/ctx/ROADMAP.md -> 2.1
**Spec**: specs/ctx/SPEC.md -> Enables CTX-007, CTX-010, CTX-020

## User Journey

TranscriptWriter and FilePolicyStore both call `syscall.Flock` which blocks forever if another process holds the lock. After this change, a shared `FlockWithContext` utility allows all flock callers to respect context cancellation and timeouts, eliminating indefinite hangs.

## Phases

### Phase 1: Core FlockWithContext
- `FlockWithContext(ctx, fd, how)` uses `LOCK_NB` for non-blocking attempt.
- Retry loop with short sleep between attempts.
- Checks `ctx.Done()` each iteration.
- Returns wrapped `ctx.Err()` on cancellation.

### Phase 2: Convenience Wrappers
- `FlockExclusiveWithContext(ctx, fd)` — wraps `LOCK_EX|LOCK_NB`.
- `FlockSharedWithContext(ctx, fd)` — wraps `LOCK_SH|LOCK_NB`.
- `FlockUnlock(fd)` — wraps `LOCK_UN`.

### Phase 3: SafeFlockFd Consolidation
- Export `SafeFlockFd` from storage package.
- Both `session.safeFlockFd` and `safety.safeFlockFd` can be replaced in future steps.

## Friction Points

| Point | Risk | Mitigation |
|-------|------|------------|
| LOCK_NB returns EWOULDBLOCK | Normal contention signal | Retry loop handles this |
| Retry polling burns CPU | Excessive wake-ups | 10ms sleep between attempts |
| Platform support | syscall.Flock is Unix-only | Build tag `!windows` |

## Design Decisions

1. **Non-blocking + retry**: `LOCK_NB` with polling is the only way to make flock cancellable. Blocking flock cannot be interrupted.
2. **10ms poll interval**: Balances latency (fast lock acquisition) with CPU usage. Flock contention is rare in practice.
3. **Build tag `!windows`**: Matches existing codebase pattern. Windows uses different locking APIs.
4. **Wrap ctx.Err()**: Satisfies `wrapcheck` linter. Error message includes "flock" for context.

## DoD

- [x] `FlockWithContext` uses LOCK_NB + retry loop (10ms poll interval).
- [x] `FlockExclusiveWithContext` and `FlockSharedWithContext` convenience wrappers.
- [x] `FlockUnlock` convenience wrapper.
- [x] `SafeFlockFd` exported.
- [x] Test: lock acquired immediately (exclusive and shared).
- [x] Test: canceled context returns context.Canceled (pre-canceled and during contention).
- [x] Test: contended lock, then released, acquired after retry.
- [x] Test: multiple shared readers concurrently.
- [x] Test: SafeFlockFd edge cases.
- [x] `go vet ./...` clean.
- [x] `make lint` clean (zero errors in new files).
- [x] `go test ./...` full suite passing.

## Implementation

### Files Created
- `internal/storage/flock.go` — `FlockWithContext`, `FlockExclusiveWithContext`, `FlockSharedWithContext`, `FlockUnlock`, `SafeFlockFd`. Build tag `!windows`.
- `internal/storage/flock_test.go` — 10 tests covering immediate acquisition, contention, cancellation, shared locks, unlock, SafeFlockFd.
