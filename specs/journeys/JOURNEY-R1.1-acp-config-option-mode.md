# Journey R1.1 — Set Config Option (Mode Category)

**Roadmap**: [../acp-protocol-gaps/ROADMAP.md](../acp-protocol-gaps/ROADMAP.md)
**Actor**: ACP Client (IDE/editor)
**Goal**: Change session mode via the non-deprecated `session/set_config_option` method

## Context

SDK v0.6.4+ adds `SetSessionConfigOption` to the stable `Agent` interface.
The request uses `ConfigId` (identifies which option) and `Value` (the selected value).
The response returns the full `ConfigOptions` array reflecting current state.

## Phases

### Phase 1 — Client Sends Config Option Request
- **Trigger**: User selects a mode from client UI (e.g., dropdown in IDE sidebar)
- **Action**: Client sends `session/set_config_option` with `{ sessionId, configId: "mode", value: "review" }`
- **Friction**: None — standard JSON-RPC request
- **UX**: Client shows loading/pending state on mode selector

### Phase 2 — Agent Validates and Applies
- **Action**: `SpinACPAgent.SetSessionConfigOption()` validates:
  1. Session exists
  2. `configId` matches a known config option ID (currently only `"mode"`)
  3. `value` is valid for the config option (valid mode ID from `getAvailableModes()`)
- **Action**: Updates `sessionModes[sessionId]` — same state as `SetSessionMode`
- **Friction**: Unknown configId → error. Invalid value → error.
- **Error path**: Returns descriptive error with wrapping sentinel

### Phase 3 — Agent Responds
- **Action**: Returns `SetSessionConfigOptionResponse` with `ConfigOptions` array showing current state
- **UX**: Client updates mode selector to reflect new mode
- **Friction**: None if successful

### Phase 4 — Agent Emits Notification (see R1.2)
- **Action**: `config_option_update` notification sent (separate journey)

## Implementation Notes

- Add `SetSessionConfigOption(ctx, req) (resp, error)` to `SpinACPAgent` in `agent.go`
- Extract `applyModeChange(sessionId, modeId) error` helper shared by both `SetSessionMode` and `SetSessionConfigOption`
- Add `buildConfigOptions(currentModeId) []acp.SessionConfigOption` to construct the response
- Keep `SetSessionMode` working for backward compatibility
- Add `ErrUnknownConfigOption` sentinel error
- Forward-compatible: unknown configIds return error now, can be extended later

## Test Plan

| # | Test | Type | Input | Expected |
|---|------|------|-------|----------|
| 1 | Set mode to "review" via config option | Unit | `{sessionId: valid, configId: "mode", value: "review"}` | Success, mode updated, response has configOptions |
| 2 | Set mode to "planning" via config option | Unit | `{sessionId: valid, configId: "mode", value: "planning"}` | Success, mode updated |
| 3 | Invalid mode value | Unit | `{sessionId: valid, configId: "mode", value: "invalid"}` | Error wraps `ErrInvalidMode` |
| 4 | Unknown configId | Unit | `{sessionId: valid, configId: "model", value: "gpt-4"}` | Error wraps `ErrUnknownConfigOption` |
| 5 | Session not found | Unit | `{sessionId: "nonexistent", configId: "mode", value: "review"}` | Error wraps `ErrSessionNotFound` |
| 6 | Response contains correct config options | Unit | After successful set | `ConfigOptions` has mode selector with updated `currentValue` |
| 7 | Legacy set_mode still works | Unit | `SetSessionMode({sessionId: valid, modeId: "review"})` | Success (regression) |

## Implementation

- **Modified**: `internal/protocol/acp/agent.go` — added `SetSessionConfigOption`, `buildConfigOptions`, `applyModeChange`
- **Modified**: `internal/protocol/acp/agent.go` — refactored `SetSessionMode` to use `applyModeChange`
- **Modified**: `internal/protocol/acp/agent_test.go` — added `TestSetSessionConfigOption_*` tests
- **Added**: `ErrUnknownConfigOption` sentinel in `agent.go`
