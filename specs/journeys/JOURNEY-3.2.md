# JOURNEY-3.2: Context Retrieval Pipeline

## Overview

| Field | Value |
|-------|-------|
| Journey ID | 3.2 |
| Title | Wire Context Retrieval Pipeline and SummarizeError |
| User Story | As a developer, context fragments are assembled from multiple sources before each LLM call, and error tool results get dedicated summarization. |
| Paper Section | 2.3 — context assembly, bullet injection |
| Roadmap Item | JOURNEY-3.2 (7 functions) |
| Depends on | JOURNEY-3.1 |

## Phases

### Phase 1: Discovery
- `retrieval.Pipeline` assembles `Fragment`s from registered `Source`s
- `retrieval.BulletSource` extracts ACE bullets from trajectory context
- `observation.SummarizeError` truncates error output with classified prefix
- All have unit tests. Never wired.

### Phase 2: Integration
- Create `retrieval.NewPipeline(NewBulletSource())` in builder.go, store on Builder for future middleware use
- Call `SummarizeError()` in `ObservationAdapter` for error tool results
- `DotProduct` and `Magnitude` become reachable transitively via embedding similarity used by BulletSource's ranking

## Implementation

### Files Modified
- `internal/contexteng/adapter/observation.go` — Added `isErrorContent()` check, call `SummarizeError()` for error tool results
- `internal/mathutil/vector.go` — Refactored `CosineSimilarity` to delegate to `DotProduct`+`Magnitude` (DRY, both now reachable)
- `internal/conversation/builder.go` — Create `retrieval.NewPipeline(NewBulletSource())`, pass to Conversation
- `internal/conversation/conversation.go` — Added `retrievalPipeline` field, `GetRetrievalPipeline()` getter
