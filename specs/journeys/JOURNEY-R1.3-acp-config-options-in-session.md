# Journey R1.3 — Config Options in NewSession Response

**Roadmap**: [../acp-protocol-gaps/ROADMAP.md](../acp-protocol-gaps/ROADMAP.md)
**Actor**: ACP Client (IDE/editor)
**Goal**: Know what configuration options are available when creating a session

## Context

The `NewSessionResponse` now includes a `ConfigOptions` field alongside the legacy `Modes` field.
This tells clients what configuration UI to render (mode selector, future model/thought-level selectors).

## Phases

### Phase 1 — Client Creates Session
- **Action**: Client sends `session/new`

### Phase 2 — Agent Returns Config Options
- **Action**: Response includes `configOptions` array with mode selector
- **Payload**: `SessionConfigOptionSelect` with `id: "mode"`, 4 options, `currentValue: "regular"`
- **Backward compat**: `modes` field still populated for older clients

### Phase 3 — Client Renders UI
- **Action**: Client reads `configOptions` and renders mode dropdown
- **UX**: User sees mode selector with 4 options

## Implementation

- **Modified**: `internal/protocol/acp/agent.go` — added `ConfigOptions: buildConfigOptions(defaultMode)` to `NewSessionResponse`
- **Modified**: `internal/protocol/acp/session_mode_test.go` — `TestNewSession_IncludesConfigOptions`
