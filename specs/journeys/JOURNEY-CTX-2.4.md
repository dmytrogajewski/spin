# JOURNEY-CTX-2.4 — LSP readLoop Error Propagation

**Status**: Done
**Roadmap**: specs/ctx/ROADMAP.md -> 2.4
**Spec**: specs/ctx/SPEC.md -> CTX-005

## User Journey

LSP language server crashes mid-session. Currently, all pending `Send` calls hang until their individual context timeouts fire because `readLoop` exits silently without closing the `done` channel. After this change, when `readLoop` encounters a read error it stores the error and closes `done`, immediately unblocking all pending callers with a meaningful error.

## Phases

### Phase 1: Store Read Error
- Add `readErr atomic.Pointer[error]` field to `StdioTransport`.
- When `readLoop` exits due to error (not clean close), store the error and close `done` via `sync.Once`.

### Phase 2: Propagate Error to Send
- In `Send`: when `done` is closed, check stored read error.
- If read error exists, return it wrapped. Otherwise return `ErrTransportClosed` (clean close).

### Phase 3: Safe done-Channel Close
- Use `sync.Once` for closing `done` to prevent double-close panic (both `readLoop` and `Close` may close it).
- `Close` also triggers `sync.Once` close of `done`.

## Friction Points

| Point | Risk | Mitigation |
|-------|------|------------|
| Double close of done channel | Panic | Use sync.Once |
| Race between readLoop exit and Close | Data race | atomic.Pointer for error, sync.Once for channel |
| Existing tests expect ErrTransportClosed | False regression | Clean Close still returns ErrTransportClosed |

## Design Decisions

1. **sync.Once for done channel**: Both `readLoop` (on error) and `Close()` may need to close `done`. Using `sync.Once` prevents double-close panic.
2. **atomic.Pointer[error]**: Thread-safe error storage without additional mutex.
3. **Preserve ErrTransportClosed for clean close**: Existing callers that check `ErrorIs(err, ErrTransportClosed)` still work.

## DoD

- [x] readLoop stores error and closes done on reader failure.
- [x] Send returns stored error when done is closed by readLoop.
- [x] Clean Close still returns ErrTransportClosed.
- [x] No double-close panic (sync.Once).
- [x] readLoop distinguishes clean close from unexpected crash via `closed` flag.
- [x] Test: server crash unblocks pending Send immediately with read error.
- [x] Test: clean Close unblocks pending Send with ErrTransportClosed.
- [x] `go vet ./...` clean.
- [x] `make lint` clean.
- [x] `go test ./...` full suite passing.

## Implementation

### Files Modified
- `internal/lsp/transport.go` — added `closeOnce sync.Once` and `readErr atomic.Pointer[error]` to `StdioTransport`; `readLoop` calls `handleReadError` on exit which stores error and closes done via `setReadErr`; `Close` uses `closeDone` (sync.Once) instead of direct close; `Send` calls `doneError()` which returns stored read error or `ErrTransportClosed`.
- `internal/lsp/transport_test.go` — 2 new tests: `TestStdioTransport_ServerCrash_UnblocksPendingSend`, `TestStdioTransport_CleanClose_ReturnsTransportClosed`.
