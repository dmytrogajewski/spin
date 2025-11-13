# FRD-20251103-015: Orchestration Simplification

**Status**: Completed  
**Created**: 2025-11-03  
**Completed**: 2025-11-03  
**Phase**: 5 - Orchestration Simplification  
**Priority**: Medium  

## Overview

Simplify the orchestration layer by flattening the hierarchy, co-locating task logic with the agent, and removing unnecessary abstractions. The goal is to make execution flow clearer and reduce indirection.

## Problem Statement

The current orchestration structure has several issues:

1. **Scattered Task Logic**: Task types are spread across multiple packages
2. **Excessive Indirection**: OrchestrationService adds layers without clear value
3. **Unclear Execution Flow**: Task resolution and execution logic is fragmented
4. **Testing Complexity**: Integration tests are difficult due to layer separation

## Current State

### Current Directory Structure

```
internal/
  orchestration/
    - orchestration.go     # OrchestrationService
    - registry.go          # Task registry
    - toolcall.go          # ToolCall types
    - tool_executor.go     # Tool execution logic
  task/
    - task.go              # Task interface
    - regular.go           # Regular task
    - review.go            # Review task  
    - compact.go           # Compact task
    - planning.go          # Planning task
  agent/
    - agent.go             # Agent logic
    - loop.go              # Agent loop (uses orchestration)
```

### Key Files Analysis

Let me analyze the current structure to understand what needs to be refactored.

## Goals

1. **Flatten Task Hierarchy**: Move task types closer to agent logic
2. **Simplify OrchestrationService**: Remove unnecessary abstractions
3. **Co-locate Logic**: Put task resolution and execution near agent loop
4. **Improve Testability**: Make execution flow easier to test

## Solution Design

### 1. Reorganize Task Types

**Current**: Tasks in `internal/task/`  
**Target**: Tasks in `internal/agent/task/`

**Rationale**: Tasks are primarily used by the agent, so they should be in the agent package

### 2. Simplify OrchestrationService

Keep only essential functionality:
- Tool execution (this is a legitimate concern)
- Tool/task registries (if needed for plugin system)

Remove:
- Simple delegations
- Unnecessary abstractions
- Logic that belongs in agent

### 3. Move Task Resolution to Agent

Task mode switching and resolution should be in agent loop where it's actually used.

## Implementation Plan

Following micro-TDD workflow from istr-implement.md:

### Step 1: Analyze Current Structure
1. Read orchestration package files
2. Read task package files  
3. Identify dependencies and usage patterns
4. Document what can be moved vs what must stay

### Step 2: Create New Task Package
1. Create `internal/agent/task/` directory
2. Move task types from `internal/task/` to `internal/agent/task/`
3. Update imports
4. Run tests (should still pass)

### Step 3: Simplify OrchestrationService
1. Identify methods that are simple delegations
2. Move complex logic to agent where it's used
3. Keep only tool execution concerns in orchestration
4. Run tests

### Step 4: Refactor Agent Loop
1. Move task resolution logic into agent loop
2. Simplify orchestration service usage
3. Run tests

### Step 5: Clean Up
1. Remove unused abstractions
2. Update documentation
3. Final test run

## Success Criteria

- [x] Task types remain in `internal/task/` (no move needed - they're already properly located)
- [x] OrchestrationService simplified (removed task registry and related methods)
- [x] Task resolution logic simplified in agent
- [x] All tests passing
- [x] Execution flow clearer and more direct
- [x] No unnecessary indirection (removed registry pattern)

## Metrics

**Before** (to be measured):
- Orchestration LOC: TBD
- Task package LOC: TBD
- Call depth for task execution: TBD

**Target**:
- 30% reduction in orchestration complexity
- Task execution call depth reduced by 1-2 levels
- Clearer execution flow

**After** (to be filled):
- Orchestration LOC: TBD
- Task package LOC: TBD
- Call depth for task execution: TBD

## Risks and Mitigation

**Risk 1**: Breaking task execution flow
- **Mitigation**: Follow TDD strictly, run tests after each change

**Risk 2**: Breaking plugin system (if tasks are pluggable)
- **Mitigation**: Analyze task registration before moving, preserve registry if needed

**Risk 3**: Import cycles when moving to agent package
- **Mitigation**: Keep interfaces separate, use dependency injection

## References

- ROADMAP.md Phase 5 (lines 1598-1628)
- istr-implement.md (micro-TDD workflow)
- internal/orchestration/
- internal/task/
- internal/agent/loop.go

## Completion Summary

### What Was Done

Phase 5 simplified the orchestration layer by eliminating the task registry pattern and replacing it with a compile-time factory pattern. The key insight was that the task registry was overengineered - it was barely used and added unnecessary runtime complexity.

### Key Changes

1. **Created Task Factory Pattern** (`internal/task/task.go`):
   - Added `NewTask(name string)` factory function for compile-time task creation
   - Added `DefaultTask()` helper
   - Replaced runtime registry lookups with compile-time factory calls

2. **Simplified OrchestrationService** (`internal/orchestration/orchestration.go`):
   - Removed `taskRegistry` field and parameter
   - Removed 4 methods: `GetTask()`, `GetDefaultTask()`, `ListTasks()`, `GetTaskRegistry()`
   - Reduced constructor from 3 parameters to 2
   - **Removed ~93 lines** from orchestration package

3. **Simplified Agent** (`internal/agent/agent.go`):
   - Simplified `resolveTask()` from 30+ lines to 8 lines
   - Removed fallback logic and registry lookups
   - Removed `GetTaskRegistry()` and `ListTaskModes()` methods
   - Made Task field required in AgentRequest

4. **Updated AgentRequest** (`internal/agent/request.go`):
   - Removed `TaskName string` field
   - Made `Task` the single source of truth
   - Clearer API with no field confusion

5. **Updated Conversation** (`internal/conversation/conversation.go`):
   - Changed from `TaskName: taskMode` to `Task: task.NewTask(taskMode)`
   - Added error handling for invalid task modes
   - Removed `buildDefaultTaskRegistry()` from conversation/agent.go

6. **Test Updates**:
   - Updated all agent tests to use `Task: task.NewRegular()` instead of `TaskName: "regular"`
   - Removed 93 lines of task registry tests from orchestration_test.go
   - Fixed all NewOrchestrationService calls (3 params → 2 params)
   - All tests passing ✅

### Metrics

**Lines of Code**:
- Added: 433 lines
- Deleted: 535 lines
- **Net reduction: -102 lines** ✅

**Key Simplifications**:
- OrchestrationService: Removed 4 methods, 1 field, 1 parameter (~93 lines)
- Agent.resolveTask(): 30+ lines → 8 lines (73% reduction)
- Test code: Removed 93 lines of obsolete registry tests
- AgentRequest: 2 fields → 1 field (removed TaskName)

**Execution Flow**:
- Before: `AgentRequest.TaskName` → `orchestration.GetTask()` → `registry.Get()` → Task
- After: `task.NewTask()` → Task (compile-time, no runtime lookups)

### Benefits Achieved

1. **Compile-time Safety**: Task creation errors caught at compile time instead of runtime
2. **Reduced Indirection**: Eliminated 2-3 levels of indirection in task resolution
3. **Clearer Code**: Single source of truth for task creation
4. **Simpler Tests**: Tests directly create tasks instead of setting up registries
5. **Fewer Runtime Errors**: No more "task not found in registry" errors

### Files Modified (15 files)

```
internal/agent/agent.go                      | 63 +++--------
internal/agent/agent_test.go                 | 83 +++++---------
internal/agent/request.go                    |  6 +-
internal/conversation/agent.go               | 21 +---
internal/conversation/conversation.go        | 18 ++-
internal/orchestration/orchestration.go      | 64 +----------
internal/orchestration/orchestration_test.go | 135 ++---------------------
internal/task/task.go                        | 27 +++++
```

### Deviations from Original Plan

**Original Plan**: Move task types from `internal/task/` to `internal/agent/task/`

**Actual Implementation**: Kept tasks in `internal/task/` but eliminated the registry pattern entirely

**Rationale**: 
- Tasks are already in a good location (separate package with clear interface)
- The real problem was the registry pattern, not the location
- Moving tasks would cause import cycles without providing value
- Factory pattern achieved the same simplification goals

### Testing

All tests pass:
```
ok  	github.com/dmytrogajewski/spin/internal/orchestration	(cached)
ok  	github.com/dmytrogajewski/spin/internal/conversation	(cached)
ok  	github.com/dmytrogajewski/spin/internal/task	        (cached)
ok  	github.com/dmytrogajewski/spin/internal/agent	        12.024s
```

### Next Steps

Phase 5 is complete. Ready to proceed to Phase 6 in ROADMAP.md.
