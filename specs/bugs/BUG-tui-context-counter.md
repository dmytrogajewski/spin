# BUG-tui-context-counter: Context counter always tiny; YOLO marker uncolored

## Summary
The status bar context counter shows implausibly small numbers (e.g. `0% (13/262.1K)`) no matter how large the conversation gets, and the YOLO approval marker renders in plain white, making bypass mode easy to miss.

## Reproduction
- Method: test
- Test: `internal/llm/ollama/convert_test.go:TestConvertOllamaChunkToOpenAI_DoneChunkCarriesUsage`
- Test: `internal/agent/caller/caller_test.go:TestCall_EmitsRealTokenUsage`
- Test: `cmd/spin/tui_tokens_test.go:TestTokenCounter_RealUsageWinsOverEstimate`
- Test: `internal/ui/status/renderer_test.go:TestRenderer_Render_HighlightsYolo`
- Evidence: status stuck at `(13/262.1K)` on `ornith-1.5:35b-262k` after multi-turn sessions

## Expected Behavior
- The counter reflects the context the provider actually processed (system prompt, tool schemas, history), taken from the LLM-reported usage of the latest call.
- `YOLO` in the status bar is highlighted yellow.

## Actual Behavior
- `convertOllamaChunkToOpenAI` drops `PromptEvalCount`/`EvalCount`, so streamed completions (the only path the TUI uses) carry zero `Usage`.
- Nothing emits `EventTurnProgress` with `TokensUsed`; the TUI consumer for it is dead code.
- On completion events the TUI overwrites the counter with `history.TokenCount()` — a content-only estimate that excludes system prompt and tool schemas.
- `Renderer.Render` strips all ANSI and paints the whole line bright white.

## Root Cause Analysis
Real usage was lost at the source: the ollama streaming converter never mapped eval counts into chunk usage, so the openai-go accumulator produced `Usage{0,0,0}` and every layer above fell back to the history estimate.

## Fix
Map eval counts into the final done chunk's `Usage` (accumulator sums chunk usage). Emit `EventTurnProgress{TokensUsed: TotalTokens}` from `LLMCaller.Call` after each successful call. In the TUI, prefer real usage once seen (`tokenCounter`), keeping the history estimate only as a fallback. Highlight `YOLO` yellow in `Renderer.Render` after width measurement so centering math is unaffected.

## Traceability
- Failing tests: see Reproduction
- Fixed in: `internal/llm/ollama/convert.go`, `internal/llm/mock.go`, `internal/agent/caller/caller.go`, `cmd/spin/tui.go`, `internal/ui/status/renderer.go`
