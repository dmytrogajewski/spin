# Journey R1.2 — Config Option Update Notification

**Roadmap**: [../acp-protocol-gaps/ROADMAP.md](../acp-protocol-gaps/ROADMAP.md)
**Actor**: ACP Client (IDE/editor)
**Goal**: Receive real-time `config_option_update` notification when a config option changes

## Context

SDK v0.6.4+ adds `SessionUpdate.ConfigOptionUpdate` field with `SessionConfigOptionUpdate` type.
After any config option change (via `SetSessionConfigOption` or legacy `SetSessionMode`), the agent
must emit this notification so clients can update their UI.

## Phases

### Phase 1 — Config Option Changes
- **Trigger**: `SetSessionConfigOption` or `SetSessionMode` succeeds
- **Action**: Agent constructs `SessionUpdate` with `ConfigOptionUpdate` discriminator
- **Payload**: `SessionConfigOptionUpdate{ConfigOptions: buildConfigOptions(currentMode)}`

### Phase 2 — Notification Delivered
- **Action**: `connection.SessionUpdate(ctx, notification)` sends to client
- **UX**: Client updates all config-related UI elements

## Test Plan

| # | Test | Type | Input | Expected |
|---|------|------|-------|----------|
| 1 | ConfigOptionUpdate sent after SetSessionConfigOption | Unit | set mode to review via config option | `ConfigOptionUpdate` notification with currentValue="review" |
| 2 | ConfigOptionUpdate sent after legacy SetSessionMode | Unit | set mode to review via set_mode | `ConfigOptionUpdate` notification with currentValue="review" |
| 3 | Both notifications sent | Unit | either method | Both `CurrentModeUpdate` AND `ConfigOptionUpdate` emitted |

## Implementation

- **Modified**: `internal/protocol/acp/agent.go` — `sendConfigOptionUpdate` helper, called from both `applyModeConfigOption` and `SetSessionMode`
- **Modified**: `internal/protocol/acp/session_mode_test.go` — tests for notification emission
