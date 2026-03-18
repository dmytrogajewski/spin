# JOURNEY-RT5 — Consolidate ToolResult Construction Patterns

**Status**: Done
**Roadmap**: specs/refactoring/tools-cleanup/ROADMAP.md -> R-T5

## User Journey

Tool implementations use inconsistent patterns for constructing ToolResult values. Some use the standard constructors (`NewToolResult`, `NewToolError`, `ErrToResultf`), while others use inline `ToolResult{...}` struct literals. After this refactoring, all tool code uses the standard constructors for consistency and maintainability.

## Design Decisions

1. **Error results**: `ToolResult{Success: false, Error: "msg"}` replaced with `NewToolError(errors.New("msg"))` for static errors, or `NewToolError(err)` for dynamic errors.
2. **Success results**: `ToolResult{Success: true, Output: "msg"}` replaced with `NewToolResult("msg")`.
3. **Formatted errors**: `ToolResult{Success: false, Error: fmt.Sprintf(...)}` replaced with `ErrToResultf(format, err)` where appropriate.
4. **Complex results**: Results with additional fields (Output + metadata) stay as literals since no constructor covers them.
5. **Shell command**: Special cases with `Output` field populated alongside success stay as-is (constructors don't support Output+metadata combo).

## DoD

- [x] All `ToolResult{Success: false, Error: "static"}` replaced with `NewToolError` + sentinel errors.
- [x] Simple `ToolResult{Success: true, Output: "msg"}` replaced with `NewToolResult`.
- [x] All inline `errors.New("...")` extracted to package-level sentinel vars.
- [x] `fmt.Errorf("...: %v", err)` fixed to `%w` for proper error wrapping.
- [x] Capitalized error strings lowercased (staticcheck ST1005).
- [x] All tests pass (3 test expectations updated for new error message format).
- [x] `go vet` clean, `make lint` 0 issues, 0 dead code.

## Implementation

### Files Modified
- `internal/tools/errors.go` — added 5 sentinel errors: `ErrExecutorNotConfigured`, `ErrCommandParameterRequired`, `ErrValidatorNotConfigured`, `ErrQueryParameterRequired`, `ErrTaskIDParameterRequired`.
- `internal/tools/shell_command.go` — replaced 7 inline error literals with sentinels + constructors; replaced 4 success literals with `NewToolResult`.
- `internal/tools/file_search.go` — replaced 1 inline error with sentinel; fixed 2 `%v` to `%w`; replaced success literal.
- `internal/tools/get_process_output.go` — replaced inline error with sentinel; fixed `%s` to `%w`.
- `internal/tools/kill_process.go` — replaced inline error with sentinel; fixed `%s` to `%w`.
- `internal/tools/git_context.go` — lowercased error string; fixed `%v` to `%w`; replaced literals with constructors.
- `internal/tools/shell_command_test.go` — updated 4 test expectations for new error message format.
