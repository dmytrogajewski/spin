# JOURNEY-CTX-5 — Phase 5: Timeout and HTTP Safety

**Status**: Done
**Roadmap**: specs/ctx/ROADMAP.md -> 5.1, 5.2, 5.3

## 5.1 — HTTP Client Timeouts (CTX-023, CTX-030)

Added timeouts to HTTP clients that had none:
- `internal/ace/embedding/ollama_embedder.go` — 60s timeout for embedding requests
- `cmd/spin/mcp.go` — 30s timeout for Smithery API search

## 5.2 — Ollama Race Condition Fix (CTX-017)

Replaced unprotected `if p.detectedCtxLen == 0` check with `sync.Once` to ensure `detectContextLength` runs exactly once even under concurrent access.

## 5.3 — streamOutput Guarded Channel Send (CTX-016)

Replaced bare `chunks <- OutputChunk{...}` with `sendChunk(ctx, chunks, chunk)` helper that selects on `ctx.Done()`, preventing goroutine leak when consumer abandons the channel.

## Implementation

### Files Modified
- `internal/ace/embedding/ollama_embedder.go` — added `embeddingClientTimeout` constant (60s); HTTP client uses it.
- `cmd/spin/mcp.go` — added `smitheryClientTimeout` constant (30s); search HTTP client uses it.
- `internal/llm/ollama/provider.go` — added `ctxLenOnce sync.Once` field; `setContextOptions` uses `sync.Once.Do`.
- `internal/agent/executor.go` — extracted `sendChunk` helper; `streamOutput` uses it for context-safe sends.
