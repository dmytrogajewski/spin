# Journey R-3.1: Background Task Manager Core

**Roadmap Item**: R-3.1
**Spec**: [SPEC.md](../refactoring/opendev-gaps/SPEC.md) Section 3
**Status**: Done

## Context

When the agent runs long-running server commands (e.g., `npm run dev`, `flask run`), execution blocks for the full timeout period (typically 5 minutes) before returning. The agent cannot inspect output or manage the process. A background task manager enables starting, monitoring, and killing long-running processes independently of the agent loop.

## User Journey

### Persona
Developer using Spin who asks the agent to start a dev server, then wants to verify it is running and read its startup logs.

### Phases

| Phase | Action | Current Experience | Target Experience |
|-------|--------|--------------------|-------------------|
| Start | Agent runs `npm run dev` | Blocks for 5min timeout | Command starts in background, returns task ID |
| Monitor | Agent checks if server is running | No way to check | `List()` returns task state (Running/Completed/Failed/Killed) |
| Output | Agent reads server logs | No access to output | `GetOutput()` returns captured stdout/stderr |
| Kill | Agent stops the server | No graceful stop | `Kill()` sends SIGTERM, escalates to SIGKILL after 5s |
| Startup | Agent waits for initial output | N/A | `Start()` waits up to 20s for first output before returning |
| Capacity | 11th concurrent task started | N/A | Error: max 10 concurrent tasks |

### Friction Points (Current)
1. Server commands block the agent loop for the entire timeout.
2. No way to read server output after launch.
3. No graceful shutdown — only context cancellation.
4. No task state tracking.

### Success Criteria
- `TaskState` enum: Running, Completed, Failed, Killed.
- `BackgroundTask` struct with state machine, output capture, process management.
- `BackgroundTaskManager` with `Start()`, `List()`, `GetOutput()`, `Kill()`.
- Task IDs: 7-char hex string.
- Output captured to temp file at `/tmp/spin/<sanitized-workdir>/tasks/<id>.output`.
- Graceful kill: SIGTERM → 5s wait → SIGKILL (process group).
- `Start()` waits up to 20s for initial startup output.
- Max 10 concurrent tasks.
- Thread-safe via `sync.Mutex`.
- Process group isolation via `process.SetGroup()`.

## Technical Design

### Package Location
- `internal/agent/executor/` — extends existing executor package.

### Constants
```go
const (
    MaxConcurrentTasks    = 10
    TaskIDLength          = 7
    StartupWaitTimeout    = 20 * time.Second
    GracefulKillTimeout   = 5 * time.Second
    OutputBufferSize      = 4096
)
```

### Types
```go
type TaskState int

const (
    TaskRunning   TaskState = iota
    TaskCompleted
    TaskFailed
    TaskKilled
)

type BackgroundTask struct {
    ID        string
    Command   string
    State     TaskState
    StartedAt time.Time
    // ... process, output file, mutex
}

type BackgroundTaskManager struct {
    mu       sync.Mutex
    tasks    map[string]*BackgroundTask
    workDir  string
    outputDir string
}
```

### Key Methods
- `NewBackgroundTaskManager(workDir string) *BackgroundTaskManager` — creates manager, ensures output dir.
- `Start(ctx, program, args, env, workDir) (taskID, initialOutput, error)` — launches process in background.
- `List() []TaskInfo` — returns snapshot of all tasks.
- `GetOutput(taskID, maxLines) (string, error)` — reads output from file.
- `Kill(taskID) error` — graceful kill with escalation.
- `Cleanup()` — kills all running tasks on shutdown.

### Integration Point
- `BuiltinRuntime` creates `BackgroundTaskManager` and makes it available.
- Pipeline `IsServer` flag drives background promotion in R-3.2.

## Test Plan

| Test | Mutant Killed | Description |
|------|---------------|-------------|
| `TestTaskState_String` | "state name wrong" | String representation of each state |
| `TestBackgroundTask_StartsAndReportsRunning` | "state not set" | Task starts in Running state |
| `TestBackgroundTask_CapturesOutput` | "output not captured" | Output written to file and readable |
| `TestBackgroundTask_CompletedState` | "exit not tracked" | Task transitions to Completed on exit 0 |
| `TestBackgroundTask_FailedState` | "failure not tracked" | Task transitions to Failed on non-zero exit |
| `TestBackgroundTask_GracefulKill` | "SIGTERM not sent" | Kill sends SIGTERM, task transitions to Killed |
| `TestBackgroundTask_KillEscalation` | "SIGKILL not sent" | If SIGTERM ignored, SIGKILL after 5s |
| `TestBackgroundTask_WaitStartup` | "startup not waited" | Start waits for initial output |
| `TestBackgroundTask_WaitStartupTimeout` | "timeout not enforced" | Start returns after 20s if no output |
| `TestBackgroundTaskManager_MaxConcurrent` | "cap not enforced" | 11th task rejected |
| `TestBackgroundTaskManager_ListTasks` | "list incomplete" | All tasks returned with correct state |
| `TestBackgroundTaskManager_GetOutput` | "output not returned" | Output readable by task ID |
| `TestBackgroundTaskManager_GetOutput_InvalidID` | "invalid ID accepted" | Error on unknown task ID |
| `TestBackgroundTaskManager_Kill_InvalidID` | "invalid kill accepted" | Error on unknown task ID |
| `TestBackgroundTaskManager_ConcurrentAccess` | "race condition" | Concurrent Start/List/Kill safe |
| `TestBackgroundTaskManager_Cleanup` | "cleanup incomplete" | All running tasks killed |

## Implementation

**Status**: Done

### Design Decisions
- **No PTY dependency**: Used pipe-based output capture (`io.MultiWriter` to file + startup reader) instead of `github.com/creack/pty`. PTY adds complexity and a dependency for minimal benefit — pipe capture suffices for output collection.
- **Configurable startup timeout**: `SetStartupTimeout()` allows tests to use short timeouts while production uses the 20s default.
- **First-line startup**: `waitStartup()` returns after the first line of output (not all output), so servers that print a "listening on :3000" line return immediately.
- **Build-tagged**: Unix implementation in `background.go` (`!windows`), Windows stub in `background_windows.go`.

### Files Created
- `internal/agent/executor/task_state.go` — `TaskState` enum (Running/Completed/Failed/Killed), `TaskInfo` snapshot struct.
- `internal/agent/executor/background.go` — `BackgroundTaskManager` with `Start()`, `List()`, `GetOutput()`, `Kill()`, `Cleanup()`, `SetStartupTimeout()`. Build-tagged `!windows`.
- `internal/agent/executor/background_windows.go` — Windows stub returning `ErrUnsupportedPlatform`.
- `internal/agent/executor/background_test.go` — 15 tests covering all state transitions, output capture, kill, cleanup, concurrency, max tasks.

### Files Modified
- `internal/events/event.go` — Added `EventBackgroundTaskStarted` and `EventBackgroundTaskStopped`.
- `internal/events/event_test.go` — Added test cases for new events.

### Roadmap
- [ROADMAP.md](../refactoring/opendev-gaps/ROADMAP.md) — R-3.1 marked Done.
