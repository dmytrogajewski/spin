# JOURNEY-CTX-7 — Phase 7: Leaf-Level Polish

**Status**: Done
**Roadmap**: specs/ctx/ROADMAP.md -> 7.2, 7.3

## 7.2 — Remaining context.Background() Cleanup

Addressed remaining items from the ctx audit:
- `memory/scratchpad.go` (CTX-049): All 5 methods renamed `_` to `ctx`, added `ctx.Err()` checks.
- `tools/git_operation_tool.go` (CTX-018): `handleGitStatus` renamed `_` to `ctx`.
- `agent/environment.go` (CTX-037): `scanProjectFiles` accepts ctx, checks `ctx.Err()` in walk callback.

Items deferred (low risk, recommendation-level):
- CTX-043 (`filesearch.Scan()`): Only used in tests. `ScanWithContext` exists for production use.
- CTX-048 (`config/mcp_manager.go`): `writeConfig` uses fast local I/O. Adding ctx changes public API for minimal benefit.

## 7.3 — LSP/OpenAI/Ollama Polish

- `openai/provider.go` (CTX-050): Stream errors now logged via `slog.Warn` instead of silently discarded.
- CTX-035 (LSP transport ctx): Deferred. `Close()` is the shutdown mechanism; readLoop error propagation (step 2.4) already handles the crash case.
- CTX-036 (Ollama client timeout): Decision: **keep** the global `http.Client.Timeout` as a safety backstop. Context-based cancellation works via the SDK. No code change needed.

## Implementation

### Files Modified
- `internal/memory/scratchpad.go` — all 5 methods use ctx with `ctx.Err()` checks.
- `internal/tools/git_operation_tool.go` — `handleGitStatus` accepts ctx.
- `internal/agent/environment.go` — `scanProjectFiles(ctx, ...)` checks ctx in walk callback.
- `internal/llm/openai/provider.go` — stream errors logged via `slog.Warn`.
