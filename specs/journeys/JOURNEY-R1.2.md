# Journey R-1.2: Process-Group Isolation

**Roadmap Item**: R-1.2
**Spec**: [SPEC.md](../refactoring/opendev-gaps/SPEC.md) Section 1, Stage 5
**Status**: In Progress

## Context

When the agent spawns a command that forks child processes (e.g., `webpack-dev-server` with file watchers), killing the parent on timeout leaves orphan children. The OS default is to place child processes in the parent's process group, but `exec.Command` only kills the direct process on context cancellation — not the entire group.

## User Journey

### Persona
Developer using Spin to run dev servers, watchers, or build tools that spawn child processes.

### Phases

| Phase | Action | Current Experience | Target Experience |
|-------|--------|--------------------|-------------------|
| Execute | Run `npm run dev` (spawns watchers) | Agent starts command normally | Agent starts with `Setpgid: true` |
| Timeout | Command exceeds timeout | Parent killed, children orphaned | Entire process group killed |
| Cancel | User cancels running command | Parent killed, children orphaned | Entire process group killed |
| Windows | Same command on Windows | N/A | Graceful no-op, same behavior as today |

### Friction Points (Current)
1. **Orphan processes**: File watchers, dev servers leave zombie children consuming resources.
2. **Port conflicts**: Orphaned servers hold ports, blocking subsequent runs.
3. **Resource leaks**: Orphaned processes accumulate over multiple agent iterations.

### Success Criteria
- All foreground commands get `SysProcAttr.Setpgid = true` on Unix.
- On context cancellation/timeout, the entire process group is killed via `syscall.Kill(-pid, SIGKILL)`.
- Windows builds compile without errors (no-op stub).
- Pure functions with build tags, no conditional compilation in main code.
- Both `Executor.prepareExecCmd` and `shell.Context.ExecuteShellCommand` use process-group isolation.

## Technical Design

### Package Location
`internal/process/` — small shared package with build-tagged files.

### Functions
```go
// SetGroup configures cmd to run in its own process group.
// On Unix: sets SysProcAttr.Setpgid = true.
// On Windows: no-op.
func SetGroup(cmd *exec.Cmd)

// KillGroup kills the entire process group of cmd.
// On Unix: sends SIGKILL to -pid (process group).
// On Windows: kills just the process (fallback).
func KillGroup(cmd *exec.Cmd) error
```

### Files
- `internal/process/group_unix.go` — `//go:build !windows`
- `internal/process/group_windows.go` — `//go:build windows`
- `internal/process/group_test.go` — `//go:build !windows` (integration test)

### Integration Points
1. `internal/agent/executor.go` `prepareExecCmd()` — call `process.SetGroup(execCmd)` after creating the command.
2. `internal/agent/executor.go` `executeAndCapture()` — use `cmd.Cancel` callback to call `process.KillGroup`.
3. `internal/shell/context.go` `ExecuteShellCommand()` — call `process.SetGroup(cmd)` after creating the command.

## Test Plan

| Test | Mutant Killed | Description |
|------|---------------|-------------|
| `TestSetGroup_SetsSetpgid` | "no SysProcAttr set" | Verifies SysProcAttr.Setpgid is true after SetGroup |
| `TestKillGroup_KillsChildren` | "only parent killed" | Spawns process that forks child, kills group, verifies child dead |
| `TestKillGroup_NilProcess` | "nil panic" | KillGroup on cmd with nil Process returns error |
| `TestKillGroup_NotStarted` | "nil panic" | KillGroup on unstarted cmd returns error |

## Implementation

**Status**: Complete

### Files Created
- `internal/process/group_unix.go` — `SetGroup` and `KillGroup` with `//go:build !windows`.
- `internal/process/group_windows.go` — No-op stubs with `//go:build windows`.
- `internal/process/group_test.go` — 4 tests including integration test for child process kill.

### Files Modified
- `internal/agent/executor.go` — `prepareExecCmd()` calls `process.SetGroup()` and sets `cmd.Cancel` to `process.KillGroup()`.
- `internal/shell/context.go` — `ExecuteShellCommand()` calls `process.SetGroup()` and sets `cmd.Cancel`.

### Roadmap
- [ROADMAP.md](../refactoring/opendev-gaps/ROADMAP.md) — R-1.2 marked Done.
