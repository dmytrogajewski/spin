# JOURNEY-CTX-4.2 — Keystore Interface Gains Context

**Status**: Done
**Roadmap**: specs/ctx/ROADMAP.md -> 4.2
**Spec**: specs/ctx/SPEC.md -> CTX-027

## User Journey

Auth operations call platform keystores (D-Bus on Linux, Keychain on macOS) without context. A hung keystore blocks forever. After this change, all Keystore methods accept context and Manager passes ctx through instead of only checking it once at entry.

## Design Decisions

1. **ctx.Err() guard check**: For memory and Linux keystores, check before the operation. The underlying `go-keyring` library doesn't support context natively, but a guard check catches already-canceled contexts.
2. **Manager passes ctx**: Instead of checking `ctx.Err()` then calling keystore without ctx, Manager now passes ctx to keystore. The keystore does the check.

## DoD

- [x] Keystore interface has ctx on all 4 methods.
- [x] memoryKeystore checks `ctx.Err()` before operations.
- [x] linuxKeystore checks `ctx.Err()` before keyring calls.
- [x] Manager passes ctx through to keystore (removed redundant ctx.Err() checks).
- [x] All test mocks updated (mockKeystore, errorKeystore).
- [x] `go vet ./...` clean.
- [x] `make lint` clean (zero errors in modified files).
- [x] `go test ./...` full suite passing.

## Implementation

### Files Modified
- `internal/auth/keystore.go` — `Keystore` interface: all 4 methods accept `ctx context.Context`.
- `internal/auth/keystore_memory.go` — `memoryKeystore` methods check `ctx.Err()` before operations.
- `internal/auth/keystore_linux.go` — `linuxKeystore` methods check `ctx.Err()` before keyring calls.
- `internal/auth/auth.go` — `Manager` methods pass ctx to keystore; removed redundant ctx.Err() checks.
- `internal/auth/auth_test.go` — `mockKeystore` and `errorKeystore` methods accept ctx; mock checks ctx.
- `internal/auth/keystore_linux_test.go` — all direct keystore calls pass `t.Context()`.
- `internal/llm/factory/factory_test.go` — `keystore.Set` call passes `t.Context()`.
