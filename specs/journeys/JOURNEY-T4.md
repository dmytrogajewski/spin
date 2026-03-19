# JOURNEY-T4: Start Background Process Tool

**Roadmap**: specs/wiring/TOOL-TESTING-ROADMAP.md -> T-4
**Status**: Complete

## User Journey

As a developer using spin exec, I want the agent to start long-running processes (servers, watchers, build daemons) in the background so I can interact with them without blocking the conversation.

Previously, the agent had `list_processes`, `get_process_output`, and `kill_process` tools but no way to start a background process. The only option was `shell_command(execute)` which blocks until completion or timeout.

## Design Decisions

1. **Separate `TaskStarter` interface** — `Start` is on a new `TaskStarter` interface, not added to `TaskManager`. Starting a process requires shell parsing and env setup that read-only management tools don't need. This follows ISP (Interface Segregation Principle).
2. **Shell delegation** — The adapter uses `sh -c "<command>"` to parse the command string, matching how `shell_command` works.
3. **Same `TaskManagerAdapter`** — The adapter implements both `TaskManager` and `TaskStarter`, keeping registration simple in `builtin.go`.
4. **Initial output capture** — The tool returns initial stdout from process startup (e.g., "Server started on port 8080"), helping the agent verify the process launched.

## DoD

- [x] `TaskStarter` interface defined in `internal/tools/task_manager.go`.
- [x] `StartProcessTool` created in `internal/tools/start_process.go`.
- [x] `TaskManagerAdapter.Start()` added in `internal/agent/executor/adapters.go`.
- [x] Tool registered in `internal/agent/executor/builtin.go` (`builtinToolCount` = 9).
- [x] 8 unit tests passing in `process_tools_test.go`.
- [x] `go vet ./...` clean.
- [x] `make lint` clean.
- [x] `go test ./...` full suite passing (88 packages).

## Implementation

### Files Created
- `internal/tools/start_process.go` — `StartProcessTool` with `command` and `working_directory` parameters.

### Files Modified
- `internal/tools/task_manager.go` — Added `TaskStarter` interface with `Start(ctx, command, workDir)`.
- `internal/agent/executor/adapters.go` — Added `TaskManagerAdapter.Start()` delegating to `BackgroundTaskManager.Start()` via `sh -c`.
- `internal/agent/executor/builtin.go` — Registered `start_process` tool, incremented `builtinToolCount` to 9.
- `internal/tools/process_tools_test.go` — Added `mockTaskStarter` and 8 tests for `StartProcessTool`.
