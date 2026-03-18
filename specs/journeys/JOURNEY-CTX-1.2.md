# JOURNEY-CTX-1.2 — Store[T] Interface Gains Context

**Status**: Done
**Roadmap**: specs/ctx/ROADMAP.md -> 1.2
**Spec**: specs/ctx/SPEC.md -> CTX-008
**Depends on**: JOURNEY-CTX-1.1 (AtomicWriteFile has ctx)

## User Journey

Any code using `storage.Store` (session persistence, conversation history, config) currently cannot cancel I/O. A hung filesystem blocks callers forever. After this change, the generic Store contract supports `context.Context` on every method, and `FileStore` respects cancellation at each I/O boundary.

## Phases

### Phase 1: Interface Change
- Add `ctx context.Context` as first parameter to all `Store[T]` methods: `Save`, `Load`, `Delete`, `Exists`, `List`.
- Update `FileStore[T]` implementation:
  - `Save`: pass ctx to `AtomicWriteFile` (enabled by CTX-1.1).
  - `Load`: check `ctx.Err()` before `os.ReadFile`.
  - `Delete`: check `ctx.Err()` before `os.Remove`.
  - `Exists`: check `ctx.Err()` before `os.Stat`.
  - `List`: check `ctx.Err()` before `os.ReadDir`.

### Phase 2: Type Alias Propagation
- `session.Storage` = `storage.Store[Session]` — automatically picks up new signatures.
- `history.Storage` = `storage.Store[Data]` — automatically picks up new signatures.

### Phase 3: Caller Migration
- `contexteng/history/storage.go` — `History.Save` and `History.Load` gain ctx, pass to store.
- `conversation/manager.go` — passes ctx from `Manager.Load`/`Manager.Save` through to store.
- `protocol/acp/agent.go` — passes ctx from `LoadSession`/`ExecutePrompt` through to store.
- All test files updated to pass `context.Background()` or `t.Context()`.

## Friction Points

| Point | Risk | Mitigation |
|-------|------|------------|
| Interface change breaks all callers | Compilation failure | Update all callers atomically |
| Type aliases auto-propagate | No extra work needed | Verify compilation |
| Test files are numerous | Tedious but mechanical | Table-driven where possible |

## Design Decisions

1. **ctx.Err() guard pattern**: Same as CTX-1.1. Check before each I/O call.
2. **History.Save/Load gain ctx**: These are the bridge between domain logic and Store. They must thread ctx.
3. **context.Background() not needed**: All production callers already have ctx available from their parent.

## DoD

- [x] `Store[T]` interface has `ctx` on all 5 methods.
- [x] `FileStore[T]` checks `ctx.Err()` and passes ctx to `AtomicWriteFile`.
- [x] `History.Save` and `History.Load` accept and propagate ctx.
- [x] All production callers pass their available ctx.
- [x] All test files compile and pass.
- [x] New cancellation tests for FileStore with canceled context (5 tests: Save, Load, Delete, Exists, List).
- [x] `go vet ./...` clean.
- [x] `make lint` clean (zero errors in modified files).
- [x] `go test ./...` full suite passing.

## Implementation

### Files Modified
- `internal/storage/store.go` — added `ctx context.Context` to all 5 `Store[T]` interface methods and `FileStore[T]` implementation.
- `internal/storage/store_test.go` — updated all existing tests to pass ctx, added 5 cancellation tests.
- `internal/contexteng/history/storage.go` — `History.Save` and `History.Load` now accept `ctx context.Context`.
- `internal/contexteng/history/storage_test.go` — updated all calls to pass ctx.
- `internal/conversation/manager.go` — `Manager.Load`/`Manager.Save` now pass ctx through to store.
- `internal/protocol/acp/agent.go` — `LoadSession`, `promptWithConversation`, `replayConversationHistory` pass ctx.
- `internal/protocol/acp/load_session_test.go` — updated Save calls with ctx.
- `internal/protocol/acp/user_message_test.go` — updated Save call with ctx.
- `internal/session/storage_test.go` — updated all calls to pass ctx.
