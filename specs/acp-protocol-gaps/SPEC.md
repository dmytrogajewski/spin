# ACP Protocol Compliance Gaps

## Context

Spin implements the Agent Client Protocol (ACP) using `github.com/coder/acp-go-sdk v0.6.3`.
The official spec lives at https://agentclientprotocol.com (protocol version 1, JSON-RPC 2.0 over stdio).

A compatibility audit (2026-03-20) identified the following gaps between Spin's implementation and the stable ACP specification.

## Gap 1 — `session/set_config_option` not implemented

**Spec reference**: `session/set_config_option` request (Client → Agent)

The spec introduces a generalized config option system that supersedes `session/set_mode` (now deprecated).
Config option categories: `mode`, `model`, `thought_level`, plus custom `_`-prefixed categories.

Spin only implements `session/set_mode`. Newer ACP clients that target the current spec will send `session/set_config_option` with `category: "mode"` instead.

**Impact**: Clients using only `set_config_option` cannot change session mode.

### Related: `config_option_update` notification

When config options change, the agent must emit a `session/update` notification with `config_option_update` discriminator. Spin does not send this notification type.

### Related: `configOptions` in `session/new` response

The `NewSessionResponse` should include a `configOptions` array describing available configuration options (modes, models, thought levels). Spin only returns `modes`.

## Gap 2 — `session/list` not implemented

**Spec reference**: `session/list` request (Client → Agent)

Allows clients to discover existing sessions with optional `cwd` filtering and cursor-based pagination.
Requires agent to advertise `sessionCapabilities.list` in `AgentCapabilities`.

Spin has `SessionHandoff.ListSessions()` in `internal/memory/handoff.go` and `session.Storage` (file-backed `Store[Session]`) that supports listing — the building blocks exist but are not wired to the ACP protocol layer.

**Impact**: Clients cannot browse/resume previous sessions without knowing the session ID.

## Gap 3 — `session_info_update` notification not sent

**Spec reference**: `session/update` notification with `session_info_update` discriminator

Sent when session metadata changes — primarily for auto-generated session titles, descriptions, or other metadata.

Spin does not emit this notification.

**Impact**: Clients that render session titles (tab names, sidebar, history) will show blank or stale titles.

## Gap 4 — Tool kind mapping is incomplete

**Spec reference**: `ToolKind` enum values

The spec defines: `read`, `edit`, `delete`, `move`, `search`, `execute`, `think`, `fetch`, `switch_mode`, `other`.

Spin only maps: `read`, `edit`, `search`, `execute` (in `mapToolNameToKind`). Unknown tools return `nil`.

The SDK already has all kind constants (`ToolKindDelete`, `ToolKindMove`, etc.) — used in e2e tests.

**Impact**: Clients that render tool calls differently per kind will show generic UI for unmapped tools.

## Existing Implementation Assets

| Asset | Location | Reusable for |
|-------|----------|-------------|
| `SessionHandoff.ListSessions()` | `internal/memory/handoff.go:122` | Gap 2 |
| `session.Storage` (file-backed) | `internal/session/file_storage.go` | Gap 2 |
| `storage.Store[T].List()` | `internal/storage/store.go` | Gap 2 |
| `SetSessionMode()` | `internal/protocol/acp/agent.go:1131` | Gap 1 (refactor target) |
| `mapToolNameToKind()` | `internal/protocol/acp/approval_handler.go:163` | Gap 4 |
| `getAvailableModes()` | `internal/protocol/acp/agent.go` | Gap 1 |
| `EventTransformer` | `internal/protocol/acp/event_transformer.go` | Gap 3 |
