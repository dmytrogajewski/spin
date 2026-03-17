# JOURNEY-R3.2 — Background Task Tool Handlers

**Status**: Done
**Roadmap**: specs/refactoring/opendev-gaps/ROADMAP.md → R-3.2

## User Journey

Agent spawns a server via `ShellCommandTool` (auto-promoted by pipeline). Agent calls `list_processes` to see running background tasks. Reads logs with `get_process_output`. Stops server with `kill_process` before exiting.

## Phases

### Phase 1: Discovery
- Agent runs a command that gets backgrounded by the pipeline
- Agent needs to check what's running

### Phase 2: Monitoring
- Agent calls `list_processes` — gets table of tasks (ID, command, state, started)
- Agent calls `get_process_output` with task ID — reads stdout/stderr

### Phase 3: Lifecycle Management
- Agent calls `kill_process` with task ID — task transitions to Killed state
- Agent re-lists to confirm task is stopped

## Friction Points

| Point | Risk | Mitigation |
|-------|------|------------|
| No TaskManager injected | Tool returns clear error | Check nil before call |
| Invalid task ID | User-friendly error | Wrap sentinel errors |
| Task already dead | Kill returns ErrTaskNotRunning | Surface as tool error, not function error |

## Design Decisions

1. **TaskManager interface**: Define in `internal/tools/` to decouple tools from concrete `executor.BackgroundTaskManager`. Three methods: `List()`, `GetOutput()`, `Kill()`.
2. **Runtime-only registration**: Tools require runtime dependency (TaskManager), so registered only in `BuiltinRuntime.RegisterTools()`, not in static `BuiltinTools` slice.
3. **Tool output format**: `list_processes` returns human-readable table. `get_process_output` returns raw output. `kill_process` returns confirmation message.
4. **TaskManagerAdapter**: Bridge in `executor/adapters.go` converts `executor.TaskInfo`/`TaskState` to `tools.TaskSnapshot`/`TaskStatus` to avoid import cycles.

## Test Plan

```
TestListProcessesTool_Name
TestListProcessesTool_Schema
TestListProcessesTool_ReturnsRunningTasks
TestListProcessesTool_EmptyList
TestListProcessesTool_NilManager
TestListProcessesTool_MultipleTasks
TestGetProcessOutputTool_Name
TestGetProcessOutputTool_Schema
TestGetProcessOutputTool_ReturnsOutput
TestGetProcessOutputTool_DefaultMaxLines
TestGetProcessOutputTool_EmptyOutput
TestGetProcessOutputTool_InvalidID
TestGetProcessOutputTool_MissingTaskID
TestGetProcessOutputTool_NilManager
TestKillProcessTool_Name
TestKillProcessTool_Schema
TestKillProcessTool_KillsTask
TestKillProcessTool_InvalidID
TestKillProcessTool_NotRunning
TestKillProcessTool_MissingTaskID
TestKillProcessTool_NilManager
TestTaskStatus_String
```

## DoD

- [x] `TaskManager` interface in `internal/tools/task_manager.go`
- [x] `internal/tools/list_processes.go` — implements `Tool`, calls `TaskManager.List()`
- [x] `internal/tools/get_process_output.go` — implements `Tool`, takes `task_id` + optional `max_lines`
- [x] `internal/tools/kill_process.go` — implements `Tool`, takes `task_id`, calls `TaskManager.Kill()`
- [x] All three registered in `BuiltinRuntime.RegisterTools()`
- [x] `builtinToolCount` updated from 5 to 8
- [x] Unit tests with mock TaskManager (22 tests)
- [x] `go vet` and `make lint` clean (0 issues)

## Implementation

### Files Created
- `internal/tools/task_manager.go` — `TaskManager` interface, `TaskStatus` enum, `TaskSnapshot` struct
- `internal/tools/list_processes.go` — `ListProcessesTool` with table formatting
- `internal/tools/get_process_output.go` — `GetProcessOutputTool` with default max lines
- `internal/tools/kill_process.go` — `KillProcessTool` with error surfacing
- `internal/tools/process_tools_test.go` — 22 unit tests with mock `TaskManager`

### Files Modified
- `internal/agent/executor/adapters.go` — added `TaskManagerAdapter`
- `internal/agent/executor/builtin.go` — registered three tools, updated `builtinToolCount` to 8
- `internal/tools/tool.go` — added `unknownStatus` constant
- `internal/tools/approval.go` — use `unknownStatus` constant
- `internal/tools/shell_command.go` — use `unknownStatus` constant
