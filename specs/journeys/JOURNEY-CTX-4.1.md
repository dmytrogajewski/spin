# JOURNEY-CTX-4.1 — TaskManager Interface Gains Context

**Status**: Done
**Roadmap**: specs/ctx/ROADMAP.md -> 4.1
**Spec**: specs/ctx/SPEC.md -> CTX-019

## User Journey

Process management tools (list, get output, kill) cannot be canceled because the `TaskManager` interface lacks context. After this change, all three methods accept `context.Context` and all tool implementations propagate their ctx through.

## Design Decisions

1. **Interface change**: Add `ctx context.Context` to `List`, `GetOutput`, `Kill`.
2. **Adapter passthrough**: `TaskManagerAdapter` passes ctx to tools layer but the underlying `BackgroundTaskManager` methods don't use ctx yet (they operate on local state/files).
3. **Tool Execute methods**: Rename `_ context.Context` to `ctx`, pass through to manager.

## DoD

- [x] TaskManager interface has ctx on all 3 methods.
- [x] TaskManagerAdapter updated (passes through, underlying methods don't use ctx yet).
- [x] All 3 tool Execute methods renamed `_` to `ctx`, pass through.
- [x] Test mock updated.
- [x] `go vet ./...` clean.
- [x] `make lint` clean.
- [x] `go test ./...` full suite passing.

## Implementation

### Files Modified
- `internal/tools/task_manager.go` — `TaskManager` interface: `List`, `GetOutput`, `Kill` all accept `ctx context.Context`.
- `internal/agent/executor/adapters.go` — `TaskManagerAdapter`: `List`, `GetOutput`, `Kill` accept `_ context.Context`.
- `internal/tools/list_processes.go` — `Execute` passes ctx to `manager.List(ctx)`.
- `internal/tools/get_process_output.go` — `Execute` passes ctx to `manager.GetOutput(ctx, ...)`.
- `internal/tools/kill_process.go` — `Execute` passes ctx to `manager.Kill(ctx, ...)`.
- `internal/tools/process_tools_test.go` — `mockTaskManager` methods accept `_ context.Context`.
