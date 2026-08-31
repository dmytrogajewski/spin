# BUG: TUI leaves the previous frame after Ctrl+C / does not exit on one Ctrl+C

## Summary

Ctrl+C must both leave a clean terminal and exit the process. Clearing the screen without canceling the conversation event loop made the hang obvious: one Ctrl+C wiped the TUI, then the process stayed alive until a second SIGINT.

## Reproduction

- Method: test
- Test: `cmd/spin/tui_quit_test.go:TestStopTUILoop_UnblocksEventLoopWithoutSecondSignal`
- Evidence (before fix): `single quit must cancel the event loop; hung waiting for events like Ctrl+C without SIGINT`
- Also: `internal/ui/adapters/puretty_e2e_test.go:TestE2E_ShutdownCtrlC_ExitsCleanly`

## Expected Behavior

One Ctrl+C (or `/exit`) exits the process and the terminal is cleared (`ESC[H ESC[2J ESC[3J]`).

## Actual Behavior

`PureTTY.Run` returned, restored cooked mode, and cleared the screen. `runTUI` then blocked on `eventDone`. That channel only closes when `ctx` is canceled or the conversation stream ends. `ctx` was canceled only by SIGINT, which raw mode does not generate. A second Ctrl+C after cooked-mode restore was required to leave.

Welcome blink and the status spinner could also redraw after the clear because they share the outer context, not `Run`'s child context.

## Root Cause Analysis

`runTUI` treated input-channel close (Ctrl+C / Ctrl+D) and `/exit` as "wait for the event loop" without canceling `ctx`. `startEventLoop` only returns on `ctx.Done()` or a closed event stream. Conversation.Stream() stays open until `conv.Close`, which runs in a defer after that wait — deadlock-shaped hang.

## Fix

- `stopTUILoop` cancels `ctx` then waits for `eventDone`. Used on signal cancel, input close, and `/exit`.
- `Run` teardown stops the welcome-blink hook and spinner before writing `ClearHome`.

## Non-goals

- `spin exec` (does not call `Run`)
- Changing Ctrl+C from "quit" to "cancel in-flight turn"

## Traceability

- Failing test: `cmd/spin/tui_quit_test.go:TestStopTUILoop_UnblocksEventLoopWithoutSecondSignal`
- Fixed in: `cmd/spin/tui.go`, `internal/ui/adapters/puretty.go`, `internal/ui/term/ansi.go`
