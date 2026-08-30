# BUG-banner-scroll-ghost: welcome cat ghosts after TUI start

## Summary

The welcome mascot paints cleanly, then about a second later the ears
duplicate, a second "spin" appears, and scrolling makes it worse.

## Reproduction

- Method: test
- Test: `internal/ui/status/renderer_test.go:TestRenderer_ScrollRegionStartsAtLineOne`
- Test: `internal/ui/banner/cat_test.go:TestEyeOverlay_AnchorsAtBaseRowAndRestoresCursor`
- Test: `internal/ui/banner/cat_test.go:TestRenderFrame_WordmarkHugsCat`
- Evidence (before fix): wordmark gap was 17 spaces; eye overlay reprinted
  "spin"; scroll region CSI was always `\x1b[1;Hr`.

## Expected Behavior

The cat starts on the first rows with help directly under it and the
wordmark next to the head. Blink only redraws eyes and stops before any
transcript output. Once the session appends text, the banner scrolls away
naturally like regular transcript content — no ghosts are left behind.

## Actual Behavior

Help jumped to the bottom of the screen (huge gap). After the first status
update the cat scrolled up; blink then stamped eye rows at absolute row 1,
ghosting ears and a second wordmark.

## Root Cause Analysis

`status.NewRenderer` set DECSTBM to lines 1..(height-2), so the mascot
lived *inside* the scroll region. Welcome `PrintLine` then called
`MoveToScrollRegion` and wrote the help text at the bottom of that region
— the gigantic gap. Later status/prompt writes issued newlines at the
region bottom, scrolling the cat up. `Blink` kept painting eye rows (and
the wordmark) at absolute `BaseRow=1`, which was no longer the cat.

Trailing empty cells in the 44-wide grid also parked "spin" 17 spaces
after the last fur pixel.

## Fix

- Welcome help is written to `Out()` directly under the cat (no
  `MoveToScrollRegion` jump), removing the gigantic gap.
- The blink animation stops synchronously on the first transcript output
  of any kind — submitted line, printed line, streamed chunks, or block
  (`SetTranscriptStartHook`) — so absolute-row eye overlays can never
  paint a moved cat.
- The scroll region stays `1..height-2`: the banner scrolls away with the
  transcript like normal content (pinning it on top was rejected as UX).
- Startup purges the terminal scrollback (`ESC[3J`, like clear(1)).
  xterm.js-based terminals push the screen into scrollback on `ESC[2J`
  and never expire it via margin scrolls, so partial frames from killed
  runs would otherwise sit at the top of scrollback forever.
- `writeHalfRow` trims trailing empty cells so the wordmark hugs the head.
- `EyeOverlay` no longer reprints the wordmark.

## Known Limitation

With DECSTBM margins active, xterm.js discards lines scrolled out of the
transcript region instead of adding them to scrollback, so past transcript
content is not scrollable. Preserving it requires replacing the margin
approach with sticky-footer repainting (separate FRD).

## Traceability

- Failing test: `internal/ui/status/renderer_test.go`, `internal/ui/banner/cat_test.go`
- Fixed in:
  - `internal/ui/status/renderer.go`
  - `internal/ui/banner/cat.go`
  - `internal/ui/adapters/puretty.go`
  - `cmd/spin/tui.go`
