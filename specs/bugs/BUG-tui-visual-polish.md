# BUG: TUI does not separate user prompts from agent work

## Summary

The transcript mixes `> ` echoes, colored badges, and raw tool bodies. The input line is a bare prefix with no bar. Agent work should read as one-line activity (`Read`, `Grepped`, `Edited`), large previews should sit in a truncated green box, and the user's line should use `→` on a full-width bar.

## Reproduction

- Method: test
- Tests:
  - `internal/ui/prompt/renderer_test.go:TestRenderer_InputBar_FullWidthGrey`
  - `internal/ui/prompt/echo_test.go:TestFormatUserEcho_UsesArrowAndCyan`
  - `internal/ui/adapters/puretty_test.go:TestHandleSubmittedLine_SeparatesUserFromAgent`
  - `internal/ui/blocks/activity_test.go`
  - `internal/ui/blocks/box_test.go`

## Expected Behavior

- Prompt prefix is `→ ` on a dim 3-line grey input box (pad / text / pad), inset by three cells, with one blank row above so the status line is not flush.
- Submitted lines echo as cyan `→ …` with blank lines around them.
- Read / grep / write blocks start with activity sentences; diffs render in a dark-green box and truncate with `... truncated (N more lines)`.
- Execute blocks keep the existing badge header.

## Actual Behavior

- Prefix is `> ` with no bar.
- Echo is `> line` with no separation from tool output.
- Blocks use `▌ WRITE` / `pattern: …` badges and dump full bodies.

## Root Cause Analysis

`TermRenderer` only writes prefix + buffer. `handleSubmittedLine` reprints the same prefix. `Renderer.Render` always emits a badge header and unboxed `renderDiff` / `renderCode`.

## Fix

- Opt-in input bar + `prompt.DefaultPrefix`.
- `FormatUserEcho` + blank-line echo.
- `FormatActivity` for read / grep / write; boxed truncated preview for diffs.

## Non-goals

- `ctrl+r` review pager (that key is not bound).
- Replacing ACP or changing tool execution.
- Agent-harness roadmap steps.

## Traceability

- Journey: this file
- Implementation: `internal/ui/prompt/`, `internal/ui/blocks/`, `internal/ui/adapters/puretty.go`
