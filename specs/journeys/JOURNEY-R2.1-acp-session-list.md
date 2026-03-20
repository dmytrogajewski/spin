# Journey R2.1 — List Sessions

**Roadmap**: [../acp-protocol-gaps/ROADMAP.md](../acp-protocol-gaps/ROADMAP.md)
**Actor**: ACP Client (IDE/editor)
**Goal**: Discover existing sessions for resume/history UX

## Context

SDK v0.6.4+ has `UnstableListSessions` dispatched via ad-hoc interface assertion.
The agent only needs to implement `UnstableListSessions(ctx, req) (resp, error)`.
Session data lives in `session.Storage` (file-backed `Store[Session]`).

## Phases

### Phase 1 — Client Requests Session List
- **Trigger**: User opens session history panel in IDE
- **Action**: Client sends `session/list` with optional `cwd` filter and `cursor`
- **Precondition**: Agent advertised `sessionCapabilities.list` in initialize response

### Phase 2 — Agent Queries Storage
- **Action**: `SpinACPAgent.UnstableListSessions()`:
  1. Validates storage is configured
  2. Calls `storage.List(ctx)` to get all session keys
  3. Loads each session to extract metadata
  4. Applies `cwd` filter if provided
  5. Applies cursor-based pagination
  6. Maps `session.Session` → `acp.UnstableSessionInfo`
- **Error path**: No storage configured → error

### Phase 3 — Client Receives Session List
- **Action**: Response contains `[]UnstableSessionInfo` with session IDs, cwd, title, updatedAt
- **UX**: Client renders session list in sidebar/picker

## Field Mapping

| Session field | UnstableSessionInfo field |
|---|---|
| `ID` | `SessionId` |
| `WorkDir` | `Cwd` |
| `Metadata.Title` | `Title` (nil if empty) |
| `UpdatedAt` (RFC3339) | `UpdatedAt` (nil if zero) |

## Pagination

Simple offset-based: cursor = string(offset). Page size = 50 (constant).
If more items remain after page, set `NextCursor`.

## Test Plan

| # | Test | Type | Input | Expected |
|---|------|------|-------|----------|
| 1 | No storage → error | Unit | storage=nil | Error wraps ErrSessionPersistenceNotAvailable |
| 2 | Empty storage → empty list | Unit | 0 sessions | Sessions=[], NextCursor=nil |
| 3 | Multiple sessions → all returned | Unit | 3 sessions saved | 3 SessionInfo items |
| 4 | Cwd filter → matching only | Unit | 3 sessions, filter by cwd | Only matching returned |
| 5 | Session fields mapped correctly | Unit | 1 session with metadata | All fields populated |
| 6 | Pagination cursor | Unit | 60 sessions, no cursor | First 50, NextCursor set |
| 7 | Pagination second page | Unit | 60 sessions, cursor="50" | Last 10, NextCursor=nil |

## Implementation

- **Modified**: `internal/protocol/acp/agent.go` — `UnstableListSessions`, `buildAgentCapabilities`
- **Created**: `internal/protocol/acp/session_list_test.go`
