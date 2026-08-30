# BUG-tui-startup-and-double-reply: TUI startup dump, broken logo, double hello

## Summary
Interactive TUI can show leftover transcript above the SPIN banner, a garbled shade-block logo, leaked chain-of-thought, and two full assistant replies for one prompt.

## Reproduction
- Method: manual + test
- Command: `spin` (or `spin tui`) with `ollama/ornith-1.5:35b-262k`, type `hello`
- Test: `internal/llm/ollama/stream_replay_test.go:TestIsFullStreamReplay`
- Evidence: screenshot with capabilities list + "I'm Claude" above the banner, then two `[thinking]` / greeting pairs after one `> hello`

## Expected Behavior
- Welcome logo and help text appear first, before any `> ` echo or model output.
- One user line produces one thinking phase and one assistant reply.
- Model thinking stays in `[thinking]` / `[thought for …]` markers, not in the transcript body.

## Actual Behavior
- Prior session text (or a prompt echo during slow startup) sits above the banner.
- Shade-block logo looks misaligned.
- Thinking prose and a "Claude" identity leak into the transcript.
- One `hello` yields two thinking summaries and two greetings.

## Root Cause Analysis
`runTUI` starts `ui.Run` (prompt accepts input) before printing the logo, so a submitted line is echoed first. Ollama `Stream` wraps every callback with `mergeThinkingContent` and forwards the final `Done` callback even when it repeats the full thinking+content body; the sanitizer then opens a second thinking phase and reprints the greeting. Per-chunk `<think>` wrap plus unsanitized model content can also leak CoT as `EventContentDelta`.

## Fix
Print the welcome block before starting the prompt loop. Drop Ollama `Done` chunks whose visible text already streamed. Emit thinking deltas before content deltas for a combined chunk.

## Traceability
- Failing test: `internal/llm/ollama/stream_replay_test.go`
- Fixed in: `internal/llm/ollama/provider.go`, `internal/agent/caller/caller.go`, `cmd/spin/tui.go`
