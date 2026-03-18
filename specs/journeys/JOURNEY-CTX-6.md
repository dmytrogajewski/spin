# JOURNEY-CTX-6 — Phase 6: Entrypoint and CLI Context

**Status**: Done
**Roadmap**: specs/ctx/ROADMAP.md -> 6.1, 6.2

## 6.1 — CLI Commands Use cmd.Context() (CTX-041, CTX-042)

Replaced `context.Background()` with `cmd.Context()` in CLI commands so Ctrl-C cancels in-progress operations:
- `cmd/spin/approval.go` — 3 `context.WithTimeout` calls now derive from `cmd.Context()`
- `cmd/spin/auth.go` — 3 functions now use `cmd.Context()` directly; removed unused `context` import

## 6.2 — EventEmitter.Emit Timeout Safety (CTX-031)

Added 5-second timeout to `emitBlock` to prevent deadlock from stuck subscribers:
- `internal/events/event.go` — `emitBlock` uses `select` with `time.After(emitBlockTimeout)` instead of bare channel send

## Implementation

### Files Modified
- `cmd/spin/approval.go` — 3 `context.Background()` replaced with `cmd.Context()`.
- `cmd/spin/auth.go` — 3 `context.Background()` replaced with `cmd.Context()`; removed `context` import.
- `internal/events/event.go` — `emitBlock` uses select with 5s timeout; added `emitBlockTimeout` constant.
