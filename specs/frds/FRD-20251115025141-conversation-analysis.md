# FRD-20251115025141: Analyze Conversation Type Usage Patterns

## Metadata
- **Status**: IN PROGRESS
- **Priority**: P0 (CRITICAL)
- **Effort**: S (1 day)
- **Dependencies**: None
- **Related**: [Codepath Duplication Roadmap](../../codepath-duplication-assessment/ROADMAP.md#feature-31-analyze-conversation-type-usage-patterns)

## Problem Statement

Three different conversation types exist in the codebase with overlapping responsibilities:

1. **`conversation.Conversation`** (`internal/conversation/conversation.go:17-141`):
   - Uses `history.History` wrapper for message management
   - Task mode management (`taskMode` field, `SetTaskMode()`)
   - Turn execution (`RunTurn()`)
   - Session tracking (`sessionID`)

2. **`appserver.Conversation`** (`internal/appserver/processor.go:52-59`):
   - Raw `[]message.Message` slice for history storage
   - Task mode tracking (`taskMode` field)
   - Protocol-level conversation tracking
   - Cancellation context management

3. **`session.Session`** (`internal/session/session.go:35-45`):
   - Turn-based structure (`Turns []*Turn` with embedded messages)
   - Persistent session state
   - Metadata management
   - State machine for session lifecycle

**Issues:**
- **History Management Duplication**: Three different models for conversation history
- **Task Mode Duplication**: Task mode validation and management logic duplicated
- **ID Generation Duplication**: Different ID types and generation methods
- **Unclear Separation of Concerns**: Overlapping responsibilities without clear boundaries
- **Maintenance Burden**: Changes must be made in multiple places
- **Inconsistency Risk**: Different implementations may diverge over time

## Goals

1. **Analyze all three conversation types** and document their responsibilities
2. **Identify overlap areas** (history, task mode, IDs)
3. **Map all usages** of each type across the codebase
4. **Determine unification strategy**:
   - Option A: Merge `appserver.Conversation` into `conversation.Conversation`
   - Option B: Clear separation of concerns with adapter pattern
   - Option C: Keep separate with shared interfaces
5. **Create analysis document** with recommendations
6. **Define next steps** for implementation

## Non-Goals

1. **NOT implementing unification** - this is an analysis task only
2. **NOT changing existing code** - only documenting and analyzing
3. **NOT breaking backward compatibility** - analysis phase only

## Design

### Analysis Framework

1. **Responsibilities Analysis**:
   - Document what each type does
   - Identify unique responsibilities
   - Identify shared responsibilities

2. **Usage Mapping**:
   - Find all usages of each type
   - Document how each type is used
   - Identify dependencies

3. **Overlap Analysis**:
   - History management duplication
   - Task mode duplication
   - ID generation duplication
   - Other shared concerns

4. **Strategy Evaluation**:
   - Evaluate each unification option
   - Consider trade-offs
   - Recommend best approach

## Implementation Plan

### Step 1: Document Responsibilities
1. Document `conversation.Conversation` responsibilities
2. Document `appserver.Conversation` responsibilities
3. Document `session.Session` responsibilities
4. Create responsibility matrix

### Step 2: Map Usages
1. Find all usages of `conversation.Conversation`
2. Find all usages of `appserver.Conversation`
3. Find all usages of `session.Session`
4. Document usage patterns

### Step 3: Identify Overlaps
1. Analyze history management duplication
2. Analyze task mode duplication
3. Analyze ID generation duplication
4. Identify other overlaps

### Step 4: Evaluate Strategies
1. Evaluate Option A (merge)
2. Evaluate Option B (adapter pattern)
3. Evaluate Option C (shared interfaces)
4. Document trade-offs

### Step 5: Create Analysis Document
1. Write analysis document with findings
2. Include recommendations
3. Define next steps
4. Review and approve

## Testing Strategy

Since this is an analysis task, no code tests are required. However, the analysis should:
1. Be thorough and accurate
2. Include code examples and references
3. Be reviewed for completeness
4. Include actionable recommendations

## Acceptance Criteria

1. ✅ All three types responsibilities documented
2. ✅ All usages of each type mapped
3. ✅ Overlap areas identified and documented
4. ✅ Unification strategy recommended with justification
5. ✅ Analysis document created with recommendations
6. ✅ Next steps defined for implementation

**Status**: ✅ **COMPLETE** (2025-11-15)

**Achievement**: Successfully analyzed three conversation types (`conversation.Conversation`, `appserver.Conversation`, `session.Session`). Documented responsibilities, mapped usages, identified overlaps (history management, task mode, ID generation). **Selected Option A (Merge `appserver.Conversation` into `conversation.Conversation`)** to eliminate duplication and unify conversation management. Created comprehensive analysis document with recommendations and next steps.

## Files to Analyze

- `internal/conversation/conversation.go` - Type 1: Uses `history.History` wrapper
- `internal/appserver/processor.go:52-59` - Type 2: Raw `[]message.Message` slice
- `internal/session/session.go:35-45` - Type 3: `Turns []*Turn` with embedded messages
- All files that use these types (to be discovered)

## Deliverables

1. **FRD** - This document
2. **Analysis Document** - `specs/analysis/conversation-unification-analysis.md`
3. **Usage Map** - List of all usages of each type
4. **Responsibility Matrix** - Comparison of responsibilities
5. **Recommendations** - Unification strategy with justification

## References

- [Codepath Duplication Assessment](../../codepath-duplication-assessment/assessment.md)
- [Roadmap Feature 3.1](../../codepath-duplication-assessment/ROADMAP.md#feature-31-analyze-conversation-type-usage-patterns)
- `internal/conversation/conversation.go` - Type 1 definition
- `internal/appserver/processor.go` - Type 2 definition
- `internal/session/session.go` - Type 3 definition

