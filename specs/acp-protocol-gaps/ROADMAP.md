# ACP Protocol Compliance — Roadmap

**Spec**: [SPEC.md](./SPEC.md)
**Created**: 2026-03-20
**SDK**: `github.com/coder/acp-go-sdk v0.6.4-0.20260227160919-584abe6abe22` (upgraded from v0.6.3)

Items are ordered by protocol impact (clients break → clients degrade → cosmetic). Each item is independently valuable and testable.

---

## R1 — Config Options: `session/set_config_option` (High Priority)

**Why first**: This replaces a deprecated method. Newer ACP clients may *only* send `set_config_option`, making mode switching impossible without this.

### R1.1 — Implement `SetConfigOption` handler for `mode` category
- **Journey**: [JOURNEY-R1.1.md](../journeys/JOURNEY-R1.1-acp-config-option-mode.md)
- **Description**: Add `SetConfigOption(ctx, req) (resp, error)` to `SpinACPAgent` that handles `category: "mode"`. Delegates to existing `SetSessionMode` logic internally. Returns current config state.
- **DoR**:
  - [ ] SDK `v0.6.3` exposes `SetConfigOptionRequest` / `SetConfigOptionResponse` types
  - [ ] Existing `SetSessionMode` tests pass (baseline)
- **DoD**:
  - [ ] `SpinACPAgent.SetConfigOption()` implemented in `agent.go`
  - [ ] Handles `category: "mode"` — validates mode ID against `getAvailableModes()`
  - [ ] Returns error for unknown categories (forward-compatible)
  - [ ] Unit test: set valid mode via config option → mode updated
  - [ ] Unit test: set invalid mode → error returned
  - [ ] Unit test: unknown category → error returned
  - [ ] Existing `SetSessionMode` tests still pass (backward compat)

### R1.2 — Emit `config_option_update` notification
- **Journey**: [JOURNEY-R1.2.md](../journeys/JOURNEY-R1.2-acp-config-option-notify.md)
- **Description**: After a config option changes (via either `set_config_option` or legacy `set_mode`), emit a `session/update` notification with `config_option_update` discriminator containing the new option state.
- **DoR**:
  - [ ] R1.1 merged and passing
  - [ ] SDK exposes `SessionUpdate.ConfigOptionUpdate` field
- **DoD**:
  - [ ] `config_option_update` notification sent after `SetConfigOption` succeeds
  - [ ] `config_option_update` notification sent after legacy `SetSessionMode` succeeds (consistency)
  - [ ] Unit test: config option change → notification captured with correct payload
  - [ ] Unit test: legacy set_mode → also emits config_option_update
  - [ ] Integration test: full flow — set_config_option → notification received by mock client

### R1.3 — Return `configOptions` in `NewSessionResponse`
- **Journey**: [JOURNEY-R1.3.md](../journeys/JOURNEY-R1.3-acp-config-options-in-session.md)
- **Description**: When creating a new session, include `configOptions` array in the response describing available modes (and any future option categories). This tells clients what config UI to render.
- **DoR**:
  - [ ] R1.1 merged
  - [ ] SDK exposes `NewSessionResponse.ConfigOptions` field
- **DoD**:
  - [ ] `NewSession` response includes `configOptions` with mode options
  - [ ] Each mode option has `id`, `label`, `description` matching existing `getAvailableModes()`
  - [ ] Default option marked appropriately
  - [ ] Unit test: NewSession response contains expected config options
  - [ ] Backward compat: `modes` field still populated (until SDK deprecates it)

---

## R2 — Session Listing: `session/list` (Medium Priority)

**Why second**: Enables session discovery/resume UX. Building blocks (`SessionHandoff.ListSessions`, `storage.Store.List`) already exist — this is a wiring task.

### R2.1 — Implement `ListSessions` handler
- **Journey**: [JOURNEY-R2.1.md](../journeys/JOURNEY-R2.1-acp-session-list.md)
- **Description**: Add `ListSessions(ctx, req) (resp, error)` to `SpinACPAgent`. Reads persisted sessions from storage, returns `SessionInfo[]` with pagination. Optionally filters by `cwd`.
- **DoR**:
  - [ ] SDK exposes `ListSessionsRequest` / `ListSessionsResponse` / `SessionInfo` types
  - [ ] `session.Storage` (file-backed) supports `List()` for key enumeration
  - [ ] `SessionHandoff.ListSessions()` tested and working
- **DoD**:
  - [ ] `SpinACPAgent.ListSessions()` implemented in `agent.go`
  - [ ] Reads from `session.Storage` — loads session metadata (ID, cwd, creation time)
  - [ ] Supports `cwd` filter (optional, empty = all sessions)
  - [ ] Supports cursor-based pagination (cursor = offset or session ID)
  - [ ] Returns `[]SessionInfo` with at minimum: `sessionId`, `cwd`
  - [ ] Unit test: list with 0 sessions → empty array
  - [ ] Unit test: list with 3 sessions → returns all 3
  - [ ] Unit test: list with cwd filter → returns only matching
  - [ ] Unit test: pagination — cursor returns next page
  - [ ] Returns error if storage not configured

### R2.2 — Advertise `sessionCapabilities.list` in capabilities
- **Journey**: [JOURNEY-R2.2.md](../journeys/JOURNEY-R2.2-acp-session-list-capability.md)
- **Description**: Update `buildAgentCapabilities()` to include `sessionCapabilities.list` when session storage is available. This tells clients the `session/list` method is callable.
- **DoR**:
  - [ ] R2.1 merged
  - [ ] SDK exposes `AgentCapabilities.SessionCapabilities.List` field
- **DoD**:
  - [ ] `buildAgentCapabilities()` sets `SessionCapabilities.List` when `storage != nil`
  - [ ] Does NOT advertise when storage is nil
  - [ ] Unit test: with storage → capability present
  - [ ] Unit test: without storage → capability absent
  - [ ] Integration test: initialize → capabilities include session list

---

## R3 — Session Info Updates (Medium Priority)

**Why third**: Improves UX for clients that show session titles/metadata — but sessions work fine without it.

### R3.1 — Emit `session_info_update` notification
- **Journey**: [JOURNEY-R3.1.md](../journeys/JOURNEY-R3.1-acp-session-info-update.md)
- **Description**: When session metadata changes (title generated, description updated), emit `session/update` with `session_info_update` discriminator. The first natural trigger is auto-generating a session title from the first prompt turn.
- **DoR**:
  - [ ] SDK exposes `SessionUpdate.SessionInfoUpdate` and `SessionInfo` type
  - [ ] Spin has a mechanism to generate/store session titles (or this is created as part of this item)
- **DoD**:
  - [ ] Define when session info updates occur (at minimum: after first prompt turn completes, generate title from conversation)
  - [ ] `session_info_update` notification sent with `sessionId`, `title`
  - [ ] Title generation: extract summary from first agent response (simple heuristic — first sentence or first N chars)
  - [ ] EventTransformer or dedicated handler emits the notification
  - [ ] Unit test: first prompt completion → session_info_update emitted with non-empty title
  - [ ] Unit test: subsequent prompts → no duplicate title notification (unless title changes)
  - [ ] Session title persisted in session storage for later retrieval by `session/list`

---

## R4 — Tool Kind Mapping Completeness (Low Priority)

**Why last**: Cosmetic improvement. All tools work correctly — this only affects how clients render tool call UI.

### R4.1 — Expand `mapToolNameToKind` to cover all spec kinds
- **Journey**: [JOURNEY-R4.1.md](../journeys/JOURNEY-R4.1-acp-tool-kind-mapping.md)
- **Description**: Extend `mapToolNameToKind()` in `approval_handler.go` (and any duplicated mapping in `notifications.go`) to map all Spin tool names to the full set of ACP tool kinds. Add `other` as fallback instead of `nil`.
- **DoR**:
  - [ ] Inventory of all Spin tool names (from tool registry)
  - [ ] SDK has all `ToolKind*` constants
- **DoD**:
  - [ ] `delete_file`, `remove_file` → `ToolKindDelete`
  - [ ] `move_file`, `rename_file` → `ToolKindMove`
  - [ ] `think`, `plan` → `ToolKindThink`
  - [ ] `web_fetch`, `http_request` → `ToolKindFetch`
  - [ ] `switch_mode` → `ToolKindSwitchMode`
  - [ ] All unmapped tools → `ToolKindOther` (not nil)
  - [ ] Single source of truth: extract mapping to shared function used by both `approval_handler.go` and `notifications.go`
  - [ ] Unit test: every Spin tool name maps to a non-nil kind
  - [ ] Unit test: table-driven test covering all known tool names
  - [ ] No behavioral changes to existing mapped tools (read, edit, search, execute)

---

## Dependency Graph

```
R1.1 ──→ R1.2
  │
  └────→ R1.3

R2.1 ──→ R2.2

R3.1       (independent)

R4.1       (independent)
```

**Parallel tracks**: R1, R2, R3, R4 are independent of each other and can be worked in parallel.
Within R1: R1.2 and R1.3 depend on R1.1.
Within R2: R2.2 depends on R2.1.

## Summary

| Item | Priority | Effort | Status |
|------|----------|--------|--------|
| R1.1 — SetConfigOption for mode | High | S | [x] DONE — [Journey](../journeys/JOURNEY-R1.1-acp-config-option-mode.md), [impl](../../internal/protocol/acp/agent.go), [test](../../internal/protocol/acp/session_mode_test.go) |
| R1.2 — config_option_update notification | High | S | [x] DONE — [Journey](../journeys/JOURNEY-R1.2-acp-config-option-notify.md), [impl](../../internal/protocol/acp/agent.go), [test](../../internal/protocol/acp/session_mode_test.go) |
| R1.3 — configOptions in NewSession response | High | S | [x] DONE — [Journey](../journeys/JOURNEY-R1.3-acp-config-options-in-session.md), [impl](../../internal/protocol/acp/agent.go), [test](../../internal/protocol/acp/session_mode_test.go) |
| R2.1 — ListSessions handler | Medium | M | [x] DONE — [Journey](../journeys/JOURNEY-R2.1-acp-session-list.md), [impl](../../internal/protocol/acp/agent.go), [test](../../internal/protocol/acp/session_list_test.go) |
| R2.2 — Advertise list capability | Medium | S | [x] DONE — implemented in `buildAgentCapabilities()`, [test](../../internal/protocol/acp/session_list_test.go) |
| R3.1 — session_info_update notification | Medium | M | [x] DONE — [Journey](../journeys/JOURNEY-R3.1-acp-session-info-update.md), [impl](../../internal/protocol/acp/event_transformer.go), [title](../../internal/protocol/acp/title.go), [test](../../internal/protocol/acp/session_info_test.go) |
| R4.1 — Complete tool kind mapping | Low | S | [x] DONE — [Journey](../journeys/JOURNEY-R4.1-acp-tool-kind-mapping.md), [impl](../../internal/protocol/acp/approval_handler.go), [test](../../internal/protocol/acp/notifications_test.go) |

**Effort**: S = small (< 2h), M = medium (2-4h)

---

## Changelog

| Date | Change |
|------|--------|
| 2026-03-20 | Initial roadmap created from protocol compliance audit |
| 2026-03-20 | R1.1-R3.1 marked BLOCKED: SDK v0.6.3 does not expose the required types. Proceeding with R4.1 (tool kind mapping) which is fully supported. |
| 2026-03-20 | R4.1 DONE: expanded `mapToolNameToKind` from 5 to 23 tool names across 7 kinds, default changed from nil to `ToolKindOther`. |
| 2026-03-20 | SDK upgraded from v0.6.3 to v0.6.4-pre (main branch). All items unblocked. Breaking changes fixed: `RequestPermissionToolCall` → `ToolCallUpdate`, `NewRequestPermissionOutcomeSelected` API change, `McpServerHttp` → `McpServerHttpInline`. |
| 2026-03-20 | R1.1 DONE: `SetSessionConfigOption` implemented for mode category with `buildConfigOptions` helper, `sendCurrentModeUpdate` extracted from `SetSessionMode`. 7 new tests. |
| 2026-03-20 | R1.2 DONE: `sendConfigOptionUpdate` emits `config_option_update` notification from both `SetSessionConfigOption` and `SetSessionMode`. 2 new tests. |
| 2026-03-20 | R1.3 DONE: `NewSessionResponse.ConfigOptions` populated with mode selector. 1 new test. |
| 2026-03-20 | R2.1+R2.2 DONE: `UnstableListSessions` with cwd filter, cursor pagination (page size 50), `sessionCapabilities.list` advertised when storage available. 11 new tests. |
| 2026-03-20 | R3.1 DONE: `session_info_update` sent on first turn complete via EventTransformer. Title generated from agent response (first sentence ≤80 chars). `title.go` + 15 tests. |
