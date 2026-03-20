# Journey R3.1 — Session Info Update Notification

**Roadmap**: [../acp-protocol-gaps/ROADMAP.md](../acp-protocol-gaps/ROADMAP.md)
**Actor**: ACP Client (IDE/editor)
**Goal**: Receive session title updates for sidebar/tab display

## Context

SDK has `SessionUpdate.SessionInfoUpdate` with `SessionSessionInfoUpdate` type.
Contains `Title *string` and `UpdatedAt *string`.

## Design

- After the first turn completes, generate a title from accumulated agent response content
- Title = first sentence (up to 80 chars), trimmed at sentence boundary or word boundary
- Send `session_info_update` notification once per session (first turn only)
- Track via `titleSent bool` on `EventTransformer`

## Phases

### Phase 1 — First Turn Completes
- **Trigger**: `EventTurnComplete` received by EventTransformer
- **Condition**: `titleSent == false` AND `accumulatedContent != ""`

### Phase 2 — Title Generated
- **Action**: `generateSessionTitle(content)` extracts first sentence (≤80 chars)
- **Heuristic**: Find first `.` `!` `?` followed by space/end, or truncate at word boundary

### Phase 3 — Notification Sent
- **Action**: `session_info_update` with title and current timestamp
- **Post**: `titleSent = true` — no more title notifications for this session

## Test Plan

| # | Test | Type | Input | Expected |
|---|------|------|-------|----------|
| 1 | Title from short content | Unit | "Hello world." | "Hello world." |
| 2 | Title from long content | Unit | 200 char response | Truncated ≤80 chars at word boundary |
| 3 | Title from sentence | Unit | "Fix the bug. Then deploy." | "Fix the bug." |
| 4 | Empty content → no title | Unit | "" | No notification |
| 5 | Second turn → no duplicate | Unit | Two turns | Only one notification |
| 6 | EventTransformer sends notification | Unit | Turn complete event | SessionInfoUpdate in mock |

## Implementation

- **Modified**: `internal/protocol/acp/event_transformer.go` — `titleSent` field, `EventTurnComplete` handling
- **Created**: `internal/protocol/acp/title.go` — `generateSessionTitle()`
- **Created**: `internal/protocol/acp/title_test.go`
- **Modified**: `internal/protocol/acp/event_transformer_test.go` or new test file
