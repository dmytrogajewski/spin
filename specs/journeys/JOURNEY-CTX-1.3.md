# JOURNEY-CTX-1.3 — BackgroundTaskManager Gains Context

**Status**: Done
**Roadmap**: specs/ctx/ROADMAP.md -> 1.3
**Spec**: specs/ctx/SPEC.md -> CTX-001, CTX-002, CTX-015

## User Journey

Agent spawns a background shell command via `BackgroundTaskManager.Start`. User cancels the conversation. Currently the background process runs forever because `Start` uses `context.Background()`. After this change, cancellation propagates: the context drives `exec.CommandContext`, the monitor goroutine reacts to `ctx.Done()` with a graceful kill sequence (SIGTERM then SIGKILL), and `waitStartup` aborts early on cancellation.

## Phases

### Phase 1: Start Gains Context
- Add `ctx context.Context` as first parameter to `Start`.
- Replace `context.Background()` with the caller-provided ctx in `exec.CommandContext`.
- The cmd context is *derived* from the caller ctx so that both `Kill()` and parent cancellation work.

### Phase 2: Monitor Becomes Context-Aware
- `monitor` receives ctx.
- On `ctx.Done()`, run the graceful kill sequence (SIGTERM -> GracefulKillTimeout -> SIGKILL).
- Process natural exit still works as before.

### Phase 3: WaitStartup Becomes Context-Aware
- `waitStartup` receives ctx.
- Select on `ctx.Done()` alongside the first-line channel and timeout.
- On cancellation, return empty string immediately.

### Phase 4: Windows Stub and Tests
- Update Windows stub `Start` signature to accept `ctx`.
- Update all test callers to pass `t.Context()`.
- Add cancellation tests.

## Friction Points

| Point | Risk | Mitigation |
|-------|------|------------|
| exec.CommandContext cancels on ctx done | Process gets SIGKILL from Go runtime | We derive our own context so we control the kill sequence |
| Monitor race with Kill | Both try to kill | Check state before kill; already-killed is no-op |
| waitStartup goroutine leak on cancel | Scanner blocks on pipe read | Pipe writer closes when monitor exits, unblocking scanner |

## Design Decisions

1. **Derive context for cmd**: `cmdCtx, cmdCancel := context.WithCancel(ctx)`. Store `cmdCancel` so `killProcess` can call it. This prevents Go's `exec.CommandContext` from sending its own SIGKILL — we handle the kill sequence ourselves.
2. **Monitor selects on ctx**: After cmd.Wait() returns OR ctx is cancelled, update state and clean up.
3. **No new dependencies**: Pure stdlib context usage.

## DoD

- [x] `Start` accepts `ctx context.Context`.
- [x] `exec.CommandContext` uses derived context with `cmd.Cancel` (SIGTERM) and `WaitDelay` (SIGKILL escalation).
- [x] `monitor` watches parent ctx; marks task as killed on cancellation.
- [x] `waitStartup` aborts on context cancellation.
- [x] Windows stub updated.
- [x] All existing tests pass with ctx.
- [x] New test: cancel context -> process killed (`TestBackgroundTask_ContextCancellation_KillsProcess`).
- [x] New test: cancel during startup wait -> returns early (`TestBackgroundTask_ContextCancellation_DuringStartup`).
- [x] New test: normal completion with live context (`TestBackgroundTask_NormalCompletion_WithLiveContext`).
- [x] `go vet ./...` clean.
- [x] `make lint` clean (zero errors in modified files).
- [x] `go test ./...` full suite passing.

## Implementation

### Files Modified
- `internal/agent/executor/background.go` — `Start` accepts ctx; `newCommand` helper creates [exec.Cmd] with `cmd.Cancel` (SIGTERM) and `WaitDelay` (SIGKILL); `monitor` tracks parent ctx; `waitStartup` checks `ctx.Done()`; `killProcess` uses `cmdCancel` for graceful kill.
- `internal/agent/executor/background_windows.go` — `Start` stub updated with ctx parameter.
- `internal/agent/executor/background_test.go` — all `Start` calls pass `t.Context()`; 3 new cancellation tests added.

### Design
Used Go 1.20+ `exec.Cmd.Cancel` and `WaitDelay` fields instead of manual goroutine-based kill management. `cmd.Cancel` sends SIGTERM to the process group; after `WaitDelay` (5s), Go escalates to SIGKILL. This is simpler and more correct than the original `context.Background()` approach.
