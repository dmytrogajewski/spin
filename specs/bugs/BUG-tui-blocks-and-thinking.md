# BUG-tui-blocks-and-thinking: TUI accent bar and thinking display

## Summary
Tool blocks render an empty colored line instead of a 1-cell accent bar, and thinking is invisible or mis-attributed: the timer spans tool execution, token counts ignore streamed chunks, exec mode stays blank, and the status bar ignores thinking events.

## Reproduction
- Method: test
- Test: `internal/ui/blocks/renderer_test.go:TestRenderHeader_AccentBarOnBadgeLine`
- Test: `internal/tui/mapper_thinking_test.go`
- Test: `internal/ui/status/aggregator_test.go:TestAggregator_ProcessEvent_AllTypes/ThinkingDelta`
- Test: `internal/ui/adapters/puretty_test.go:TestShouldUpdateStatusBar_IncludesThinkingDelta`
- Evidence: live `spin exec` against ornith-1.5:35b-262k printed a color-only blank line before WRITE/READ, then `[thought for 19.99s, ~43 tokens]` after the tools.

## Expected Behavior
- Each tool badge sits on the same line as a 1-cell accent glyph.
- A thinking phase prints a start marker, then a duration/token summary when that phase ends (content or tool start).
- A new thinking phase after tools starts a fresh timer.
- Streamed thinking chunks contribute to the token estimate (chars/4, same as status).
- Interactive status bar switches to Thinking on `EventThinkingDelta`.

## Actual Behavior
- Header writes tag color and reset with no glyph, then an extra newline.
- Thinking stays open across tool calls; summary appears only on the first later content delta.
- Token estimate counts whitespace only, so 1–8 character Ollama chunks count as ~0.
- Exec mode has no status bar and no thinking transcript line.
- Aggregator and `shouldUpdateStatusBar` ignore `EventThinkingDelta`.

## Root Cause Analysis
`RenderHeader` reserved a 1-cell accent bar but emitted only SGR codes. `Mapper` treated all thinking deltas in a turn as one open block and counted tokens by whitespace. `Aggregator.ProcessEvent` and `PureTTY.shouldUpdateStatusBar` had no `EventThinkingDelta` case, so exec (no status bar) and interactive TUI both hid live thinking.

## Fix
Draw `AccentBarGlyph` on the badge line. Close thinking on tool start and emit `[thinking]` / `[thought for …]` per phase. Estimate tokens as `max(1, runes/CharsPerToken)`. Map thinking deltas to status state `Thinking` and redraw the status bar.

## Traceability
- Failing test: `internal/ui/blocks/renderer_test.go`, `internal/tui/mapper_thinking_test.go`, `internal/ui/status/aggregator_test.go`, `internal/ui/adapters/puretty_test.go`
- Fixed in: `internal/ui/blocks/renderer.go`, `internal/ui/blocks/tokens.go`, `internal/tui/mapper.go`, `internal/ui/status/aggregator.go`, `internal/ui/adapters/puretty.go`
