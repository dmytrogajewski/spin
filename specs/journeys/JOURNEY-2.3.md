# JOURNEY-2.3: Session Persistence

## Overview

| Field | Value |
|-------|-------|
| Journey ID | 2.3 |
| Title | Wire Session Index and Transcript Writer |
| User Story | As a developer, sessions are indexed for fast listing and conversation transcripts are persisted as JSONL for recovery. |
| Paper Section | Persistence layer — session index, transcript writer |
| Roadmap Item | JOURNEY-2.3: Session Persistence (12 functions) |

## Phases

### Phase 1: Discovery
- `session.Index` with CRUD + rebuild fully implemented, unit tested
- `session.TranscriptWriter` with append/read/close fully implemented, unit tested
- **Friction**: Never instantiated from production code

### Phase 2: Integration
- Create `SessionIndex` in `builder.go::Build()` after session creation
- Update index with new session entry
- Create `TranscriptWriter` in `Build()` for the session
- Add `transcriptWriter` to `Conversation` struct
- Append messages in `RunTurn()` after harness execution
- Close transcript in `Close()`

## Implementation

### Files Modified
- `internal/conversation/builder.go` — Create `SessionIndex` with `WithRebuildCallback`, update on session creation; create `TranscriptWriter` for JSONL persistence; pass both to Conversation
- `internal/conversation/conversation.go` — Added `transcriptWriter` and `sessionIndex` fields; append messages in `RunTurn()`; close transcript in `Close()`; added `GetSessionIndex()` getter
