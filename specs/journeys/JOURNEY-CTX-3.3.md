# JOURNEY-CTX-3.3 — ACP Server Startup Context

**Status**: Done
**Roadmap**: specs/ctx/ROADMAP.md -> 3.3
**Spec**: specs/ctx/SPEC.md -> CTX-013, CTX-014

## User Journey

ACP server creates infrastructure with `context.Background()` before setting up cancellation. If a signal arrives during initialization, operations cannot be canceled. Additionally, `acpCommandContext.SetMode` uses `context.Background()` when the caller's context is available. After this change, context is cancellable from the start and SetMode uses the caller's context.

## Design Decisions

1. **Move WithCancel to top**: `ctx, cancel := context.WithCancel(context.Background())` created before `createACPInfra`.
2. **Store ctx in acpCommandContext**: Add `ctx` field set during `executeCommand`, avoiding interface changes.

## DoD

- [x] `runACPServer` creates cancellable context before infrastructure creation.
- [x] `CommandContext.SetMode` interface accepts `ctx context.Context`.
- [x] `acpCommandContext.SetMode` passes caller ctx to `SetSessionMode` (no more `context.Background()`).
- [x] `tuiCommandContext.SetMode` and test mock updated for new signature.
- [x] `go vet ./...` clean.
- [x] `make lint` clean (zero errors in modified files).
- [x] `go test ./...` full suite passing.

## Implementation

### Files Modified
- `cmd/spin/acp.go` — moved `context.WithCancel` before `createACPInfra`; removed redundant second `WithCancel`.
- `internal/commands/commands.go` — `CommandContext.SetMode` interface now accepts `ctx context.Context`; `ModeCommand.Execute` passes ctx.
- `internal/protocol/acp/commands.go` — `acpCommandContext.SetMode` accepts ctx, passes to `SetSessionMode`.
- `cmd/spin/tui_command_context.go` — `tuiCommandContext.SetMode` accepts `_ context.Context`.
- `internal/commands/commands_test.go` — mock `SetMode` accepts `_ context.Context`.
