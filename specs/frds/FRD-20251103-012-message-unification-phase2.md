# FRD-20251103-012: Message Type Unification (Phase 2)

**Date**: 2025-11-03  
**Status**: Draft  
**Owner**: Rob Pike  
**Phase**: 2 of 7 (Refactoring Roadmap)

## Problem Statement

The codebase has **3 different Message types** across different packages, causing:
- **Type fragmentation**: Constant conversions between message.Message, agent.Message, llm.Message
- **Adapter proliferation**: 5+ files with conversion functions (llm_convert.go, ollama/convert.go, etc.)
- **Cognitive overhead**: Developers must remember which Message type to use where
- **Bug surface area**: Each conversion can fail or lose data
- **Performance overhead**: Allocations and copying in conversion hot paths

## Current State Analysis

### Message Types Found

1. **`internal/message/message.go`** (CANONICAL - most complete)
   - Fields: ID, Role, Content, ToolCalls, ToolCallID, Timestamp, Tokens, Name, Metadata
   - Fully featured with metadata support
   - Used in conversation history

2. **`internal/agent/request.go`**
   - Fields: Role, Content, ToolCalls (orchestration.ToolCall), ToolCallID, Timestamp
   - Missing: ID, Tokens, Name, Metadata
   - Used in agent requests/responses

3. **`internal/llm/completion.go`**
   - Fields: Role, Content, ToolCalls, ToolCallID
   - Minimal for LLM API interactions
   - Missing: ID, Timestamp, Tokens, Metadata

4. **`internal/protocol/protocol.go`** (KEEP - different purpose)
   - Wire protocol message (Type + JSON data)
   - NOT part of unification - serves different purpose

### Conversion Functions to Remove

Files with conversion code:
- `internal/agent/llm_convert.go` - Convert agent.Message ↔ llm.Message
- `internal/llm/ollama/convert.go` - Convert to Ollama API format
- `internal/agent/loop.go` - Inline conversions
- `internal/agent/agent.go` - Inline conversions

## Goals

### Primary Goal
**Single Message Type**: Use `message.Message` throughout codebase, except protocol wire format.

### Secondary Goals
1. **Remove all Message conversion functions** - No more ToMessage/FromMessage
2. **Keep tests green** - All existing tests must pass after migration
3. **No backward compatibility** - Breaking change is acceptable (as per instructions)
4. **Zero deadcode** - Remove all unused conversion code
5. **Performance neutral or better** - No allocations in hot paths

## Non-Goals

- Protocol message unification (protocol.Message serves different purpose - wire format)
- Changing external API contracts (focus on internal types)
- Adding new Message features (just unify existing ones)

## Proposed Solution

### Step 1: Enhance message.Message (if needed)

Current `message.Message` is already comprehensive. No changes needed.

### Step 2: Replace agent.Message → message.Message

Files to update:
- `internal/agent/request.go` - Remove Message type, use message.Message
- `internal/agent/agent.go` - Update all references
- `internal/agent/loop.go` - Update all references
- `internal/agent/executor.go` - Update if used
- `internal/agent/*_test.go` - Update test code

### Step 3: Replace llm.Message → message.Message

Files to update:
- `internal/llm/completion.go` - Remove Message type, use message.Message
- `internal/llm/provider.go` - Update interface
- `internal/llm/ollama/provider.go` - Update implementation
- `internal/llm/ollama/convert.go` - Convert message.Message → Ollama API directly
- `internal/llm/*_test.go` - Update test code

### Step 4: Remove Conversion Code

Delete/simplify:
- `internal/agent/llm_convert.go` - DELETE entirely
- `internal/llm/ollama/convert.go` - Keep only Ollama API conversion (message.Message → ollama.Message)
- All inline conversion functions in agent.go, loop.go

### Step 5: Update orchestration.ToolCall

Current: `agent.Message` uses `orchestration.ToolCall`  
After: Use `message.ToolCall` directly

Options:
- Replace orchestration.ToolCall with message.ToolCall (preferred)
- OR: Add conversion in orchestration boundary only

## Implementation Plan (Micro-TDD)

### Phase A: Preparation (No Breaking Changes Yet)
1. Add helper constructors to message.Message if needed
2. Ensure message.ToolCall is complete for all use cases
3. Write golden tests for message.Message serialization

### Phase B: Replace agent.Message
1. Update agent/request.go to import and use message.Message
2. Update agent/agent.go method signatures
3. Update agent/loop.go
4. Remove agent.Message type definition
5. Delete agent/llm_convert.go
6. Run tests, fix compilation errors

### Phase C: Replace llm.Message  
1. Update llm/completion.go to use message.Message
2. Update llm/provider.go interface
3. Update llm/ollama/provider.go implementation
4. Simplify llm/ollama/convert.go (only Ollama API conversion remains)
5. Run tests, fix compilation errors

### Phase D: Cleanup
1. Remove all dead conversion code
2. Run `make lint` and fix issues
3. Run `uast parse` and analyze
4. Verify zero deadcode

## Testing Strategy

### Unit Tests (90%+ coverage target)
- `message/message_test.go` - Comprehensive tests for Message type
- Update existing agent tests to use message.Message
- Update existing llm tests to use message.Message

### Integration Tests
- Verify agent loop works with unified Message
- Verify LLM calls work with unified Message
- Verify conversation history persistence works

### Property-Based Tests
- Round-trip JSON serialization
- ToolCall accumulation in streaming

### Backward Compatibility
NOT REQUIRED (as per instructions - no backward compat needed)

## Success Criteria

1. ✅ Single Message type used everywhere (except protocol wire format)
2. ✅ Zero conversion functions between internal Message types
3. ✅ All tests passing (make test)
4. ✅ Zero lint errors (make lint)
5. ✅ Zero deadcode (uast analysis)
6. ✅ Test coverage ≥ 90% for message package

## Risk Assessment

### Low Risk
- message.Message is already well-designed and complete
- Most conversions are mechanical (field mapping)
- Tests will catch any issues

### Medium Risk
- orchestration.ToolCall vs message.ToolCall - need to pick one
- Large number of files to update in one go

### Mitigation
- Follow micro-TDD: small changes, frequent test runs
- Use compiler to find all usage sites (grep + compiler errors)
- Keep Phase B and C separate (agent first, then llm)

## Performance Impact

**Expected**: Neutral or improved
- Remove conversion allocations in hot paths
- Fewer intermediate copies
- Direct field access instead of function calls

## Migration Impact

**Breaking Changes**: YES
- agent.Message removed
- llm.Message removed  
- Conversion functions removed

**User Impact**: None (internal types only)

## Dependencies

- Phase 1 (Config) must be complete ✅
- No external dependencies

## Timeline

- Day 1: FRD review and approval
- Day 2-3: Phase A (Preparation)
- Day 4-5: Phase B (Replace agent.Message)
- Day 6-7: Phase C (Replace llm.Message)
- Day 8: Phase D (Cleanup, lint, analyze)
- Day 9-10: Testing, verification, documentation

**Total**: 10 days (2 weeks)

## References

- ROADMAP.md: Phase 2 - Message Type Unification
- Issue #2: Message Type Fragmentation (HIGH PRIORITY)
- internal/message/message.go - Canonical Message type
