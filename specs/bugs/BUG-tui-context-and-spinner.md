# BUG-tui-context-and-spinner: Status bar shows 128K and no spinner on submit

## Summary
The TUI status bar hard-codes a 128K context cap and stays Idle after Enter until the first stream delta, so the model window (262144) is wrong and the activity spinner never starts on prompt submit.

## Reproduction
- Method: test
- Test: `cmd/spin/tui_status_test.go:TestResolveUIContextWindow_UsesConfig`
- Test: `internal/ui/adapters/puretty_test.go:TestProcessEvent_TurnStartStartsSpinner`
- Evidence: status `(31/128.0K) Idle` on `ornith-1.5:35b-262k`; no spinner after Enter

## Expected Behavior
- Status max tokens come from `llm.context_window`, then provider capabilities, then the 128K fallback.
- Enter starts the status spinner (Starting/Thinking) immediately.

## Actual Behavior
- `configureMaxTokens` always sets 128000.
- `EventTurnStart` is never emitted; mapper also skips status updates when `Data` is nil. Spinner waits for the first thinking/content delta.

## Root Cause Analysis
TUI ignored configured/detected context and used `defaultMaxTokens`. The harness never emitted `EventTurnStart`, and `MapEvent` returned before `ProcessEvent` when `Data` was nil.

## Fix
Resolve context window from config then provider. Emit `EventTurnStart` at the start of `RunTurn`, process status before the nil-data return, and mark the TUI Starting as soon as a line is submitted.

## Traceability
- Failing test: `cmd/spin/tui_status_test.go`, `internal/tui/mapper_thinking_test.go`, `internal/ui/adapters/puretty_test.go`
- Fixed in: `cmd/spin/tui.go`, `internal/tui/mapper.go`, `internal/conversation/conversation.go`
