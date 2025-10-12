# Feature Requirements Document: Implement Task Resolution Logic

**FRD-20251012132252**

**Status**: Draft
**Created**: 2025-10-12
**Author**: Spin Agent
**Related**: [P1.2] Task Mode System Integration - Phase 1
**Priority**: HIGH - Needed for P1.3
**Estimated Effort**: 1-2 hours
**Depends On**: P1.1 (Task Registry to Agent) - ✅ COMPLETE

## Overview

This FRD describes the implementation of P1.2 from the task-modes roadmap: adding logic to resolve which task mode to use based on explicit task object, task name, or default.

## Context

### Current State

After P1.1 completion:
- ✅ Agent has `taskRegistry` field initialized with 4 modes
- ✅ `AgentRequest` has `Task` field
- ❌ No `TaskName` field in `AgentRequest`
- ❌ No resolution logic to determine which task to use
- ❌ Agent always uses whatever task is passed explicitly or none

### Problem Statement

Currently, there's no flexible way to specify which task mode to use:
- Must pass explicit `Task` object (cumbersome for API users)
- No way to specify task by name string (ergonomic)
- No automatic fallback to default task (safety)
- No validation of task names (error-prone)

This makes task modes hard to use from CLI, REPL, and WebSocket protocol where passing string names is natural.

### Goals

1. Add `TaskName` field to `AgentRequest` struct
2. Implement `resolveTask(*AgentRequest) (Task, error)` method
3. Establish clear precedence: explicit Task > TaskName > default
4. Validate task names against registry
5. Return clear errors for invalid configurations
6. Maintain backward compatibility
7. Achieve ≥90% test coverage

### Non-Goals

- Tool filtering implementation (P1.3)
- Token budget application (P1.4)
- Conversation-level task tracking (Phase 2)
- CLI/REPL integration (Phase 3)

## Requirements

### Functional Requirements

#### FR1: Add TaskName Field to AgentRequest

**Priority**: CRITICAL

**Description**: Add a `TaskName` string field to `AgentRequest` to allow specifying task mode by name.

**Acceptance Criteria**:
```go
type AgentRequest struct {
	// Input is the user's request
	Input string

	// History is the conversation history
	History []Message

	// Context is the environment context (optional, will use agent's context if nil)
	Context *Environment

	// Task is the task mode (optional, uses regular mode if nil)
	Task Task

	// TaskName is the name of the task mode to use (optional, resolved from registry)
	// Takes precedence over default but is overridden by explicit Task field.
	// If both Task and TaskName are provided, Task takes precedence.
	TaskName string  // NEW: Task name for resolution

	// WorkDir is the working directory
	WorkDir string
}
```

**Validation**:
- Field must be string type (task names are strings)
- Field must be optional (allows fallback to default)
- Field must be after Task field (logical ordering)
- Godoc must explain precedence rules clearly

---

#### FR2: Implement resolveTask() Method

**Priority**: CRITICAL

**Description**: Implement core resolution logic with 3-tier precedence system.

**Acceptance Criteria**:
```go
// resolveTask determines which task to use for this request.
// Precedence order:
//   1. Explicit req.Task object (if non-nil)
//   2. Task by name req.TaskName (if non-empty, looked up in registry)
//   3. Default task from registry
//
// Returns an error if:
//   - TaskName is provided but not found in registry
//   - No default task is configured in registry
func (a *Agent) resolveTask(req *AgentRequest) (Task, error) {
	// Priority 1: Explicit task object provided
	if req.Task != nil {
		return req.Task, nil
	}

	// Priority 2: Task name provided - look up in registry
	if req.TaskName != "" {
		task, err := a.taskRegistry.Get(req.TaskName)
		if err != nil {
			return nil, fmt.Errorf("task resolution failed: task '%s' not found in registry", req.TaskName)
		}
		return task, nil
	}

	// Priority 3: Use default task from registry
	task, err := a.taskRegistry.GetDefault()
	if err != nil {
		return nil, fmt.Errorf("task resolution failed: no default task configured: %w", err)
	}

	return task, nil
}
```

**Validation**:
- Must return explicit Task if provided (priority 1)
- Must look up TaskName in registry if provided (priority 2)
- Must fall back to default if neither provided (priority 3)
- Must return clear error for unknown task name
- Must return clear error if no default configured
- Must not modify the request

---

#### FR3: Error Handling for Invalid Task Names

**Priority**: HIGH

**Description**: Provide clear, actionable error messages when task resolution fails.

**Error Scenarios**:

1. **Unknown Task Name**
   ```go
   req := &AgentRequest{TaskName: "nonexistent"}
   task, err := agent.resolveTask(req)
   // err: "task resolution failed: task 'nonexistent' not found in registry"
   ```

2. **No Default Configured**
   ```go
   // Edge case: registry has no default set (should never happen with default init)
   req := &AgentRequest{}
   task, err := agent.resolveTask(req)
   // err: "task resolution failed: no default task configured: <registry error>"
   ```

**Validation**:
- Error messages must include the task name for unknown tasks
- Error messages must wrap underlying registry errors
- Error messages must start with "task resolution failed:" for consistency
- Error messages must help user understand what went wrong

---

### Non-Functional Requirements

#### NFR1: Backward Compatibility

**Description**: Existing code must continue to work without modification.

**Acceptance Criteria**:
- Requests with only explicit `Task` set continue to work
- Requests with neither `Task` nor `TaskName` get default (regular mode)
- All existing tests pass without modification
- No breaking changes to public API

**Validation**:
```bash
go test ./internal/core/
# All existing tests must pass
```

---

#### NFR2: Performance

**Description**: Task resolution must be extremely fast (hot path).

**Acceptance Criteria**:
- Resolution time: < 1μs for explicit task (single nil check)
- Resolution time: < 5μs for name lookup (single map lookup)
- Resolution time: < 5μs for default (single map lookup)
- Zero allocations for explicit task path

**Validation**:
```bash
go test -bench=BenchmarkResolveTask ./internal/core/
```

---

#### NFR3: Code Quality

**Description**: Code must meet project quality standards.

**Acceptance Criteria**:
- `make lint` passes (zero errors)
- Cyclomatic complexity ≤ 5 (simple if-else chain)
- Test coverage ≥ 90%
- Godoc on resolveTask() method
- No dead code

**Validation**:
```bash
make lint
go test -cover ./internal/core/
gocyclo -over 5 internal/core/agent.go
```

---

## Technical Design

### Algorithm

```
FUNCTION resolveTask(req *AgentRequest) -> (Task, error):
    // Path 1: Explicit task object (fastest - single nil check)
    IF req.Task != nil:
        RETURN req.Task, nil

    // Path 2: Task by name (fast - single map lookup)
    IF req.TaskName != "":
        task = taskRegistry.Get(req.TaskName)
        IF task == nil:
            RETURN nil, error("task resolution failed: task '%s' not found in registry", req.TaskName)
        RETURN task, nil

    // Path 3: Default task (fast - single map lookup)
    task = taskRegistry.GetDefault()
    IF task == nil:
        RETURN nil, error("task resolution failed: no default task configured")
    RETURN task, nil
```

**Time Complexity**: O(1) for all paths (nil check or single map lookup)
**Space Complexity**: O(1) (no allocations in explicit task path)

---

### Data Flow

```
┌─────────────────────────────────────────────┐
│         AgentRequest                        │
│  - Task: Task?                              │
│  - TaskName: string?                        │
└───────────────┬─────────────────────────────┘
                │
                v
┌─────────────────────────────────────────────┐
│         resolveTask()                       │
│  1. Check Task != nil?                      │
│     YES → return Task                       │
│     NO  → continue                          │
│                                             │
│  2. Check TaskName != ""?                   │
│     YES → registry.Get(TaskName)            │
│            Found? → return Task             │
│            Not Found? → error               │
│     NO  → continue                          │
│                                             │
│  3. registry.GetDefault()                   │
│     Found? → return Task                    │
│     Not Found? → error (should never happen)│
└───────────────┬─────────────────────────────┘
                │
                v
┌─────────────────────────────────────────────┐
│         Resolved Task                       │
│  - Used in P1.3 (tool filtering)            │
│  - Used in P1.4 (token budget)              │
└─────────────────────────────────────────────┘
```

---

### Edge Cases

| Case | Input | Expected Output | Notes |
|------|-------|----------------|-------|
| Both Task and TaskName set | `Task: compact, TaskName: "review"` | Returns `compact` task | Task takes precedence |
| Only Task set | `Task: review, TaskName: ""` | Returns `review` task | Direct return |
| Only TaskName set | `Task: nil, TaskName: "compact"` | Returns `compact` task from registry | Name lookup |
| Neither set | `Task: nil, TaskName: ""` | Returns default task (`regular`) | Fallback to default |
| Invalid TaskName | `Task: nil, TaskName: "invalid"` | Error: "task 'invalid' not found" | Validation error |
| No default (edge case) | `Task: nil, TaskName: "", no default` | Error: "no default task configured" | Should never happen |

---

## Testing Strategy

### Unit Tests

#### Test 1: Explicit Task Takes Precedence
```go
func TestAgent_ResolveTask_ExplicitTask(t *testing.T) {
	agent := newTestAgent(t)

	explicitTask := task.NewReview()
	req := &AgentRequest{
		Task:     explicitTask,
		TaskName: "compact", // Should be ignored
	}

	resolved, err := agent.resolveTask(req)
	require.NoError(t, err)
	require.Equal(t, explicitTask, resolved)
	require.Equal(t, "review", resolved.Name())
}
```

#### Test 2: Task Name Lookup
```go
func TestAgent_ResolveTask_ByName(t *testing.T) {
	agent := newTestAgent(t)

	req := &AgentRequest{
		TaskName: "compact",
	}

	resolved, err := agent.resolveTask(req)
	require.NoError(t, err)
	require.NotNil(t, resolved)
	require.Equal(t, "compact", resolved.Name())
}
```

#### Test 3: Default Task Fallback
```go
func TestAgent_ResolveTask_Default(t *testing.T) {
	agent := newTestAgent(t)

	req := &AgentRequest{
		// Neither Task nor TaskName set
	}

	resolved, err := agent.resolveTask(req)
	require.NoError(t, err)
	require.NotNil(t, resolved)
	require.Equal(t, "regular", resolved.Name()) // Default is regular
}
```

#### Test 4: Invalid Task Name Error
```go
func TestAgent_ResolveTask_InvalidName(t *testing.T) {
	agent := newTestAgent(t)

	req := &AgentRequest{
		TaskName: "nonexistent",
	}

	resolved, err := agent.resolveTask(req)
	require.Error(t, err)
	require.Nil(t, resolved)
	require.Contains(t, err.Error(), "task resolution failed")
	require.Contains(t, err.Error(), "nonexistent")
	require.Contains(t, err.Error(), "not found in registry")
}
```

#### Test 5: All Four Built-in Modes Resolve
```go
func TestAgent_ResolveTask_AllBuiltinModes(t *testing.T) {
	agent := newTestAgent(t)

	modes := []string{"regular", "review", "compact", "planning"}
	for _, mode := range modes {
		t.Run(mode, func(t *testing.T) {
			req := &AgentRequest{TaskName: mode}
			resolved, err := agent.resolveTask(req)
			require.NoError(t, err)
			require.Equal(t, mode, resolved.Name())
		})
	}
}
```

#### Test 6: Empty String TaskName Uses Default
```go
func TestAgent_ResolveTask_EmptyTaskName(t *testing.T) {
	agent := newTestAgent(t)

	req := &AgentRequest{
		TaskName: "", // Explicitly empty
	}

	resolved, err := agent.resolveTask(req)
	require.NoError(t, err)
	require.Equal(t, "regular", resolved.Name()) // Gets default
}
```

### Benchmarks

#### Benchmark 1: Explicit Task Path (Fastest)
```go
func BenchmarkResolveTask_ExplicitTask(b *testing.B) {
	agent := newTestAgent(b)
	req := &AgentRequest{
		Task: task.NewRegular(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		task, err := agent.resolveTask(req)
		if err != nil {
			b.Fatal(err)
		}
		_ = task
	}
}
// Expected: < 1μs per operation, 0 allocs
```

#### Benchmark 2: Task Name Lookup
```go
func BenchmarkResolveTask_ByName(b *testing.B) {
	agent := newTestAgent(b)
	req := &AgentRequest{
		TaskName: "regular",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		task, err := agent.resolveTask(req)
		if err != nil {
			b.Fatal(err)
		}
		_ = task
	}
}
// Expected: < 5μs per operation
```

#### Benchmark 3: Default Fallback
```go
func BenchmarkResolveTask_Default(b *testing.B) {
	agent := newTestAgent(b)
	req := &AgentRequest{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		task, err := agent.resolveTask(req)
		if err != nil {
			b.Fatal(err)
		}
		_ = task
	}
}
// Expected: < 5μs per operation
```

---

## Implementation Plan

### Step 1: Add TaskName Field
- Modify `internal/core/agent.go`
- Add `TaskName string` to `AgentRequest` struct (line ~156)
- Add godoc comment explaining precedence

**Estimate**: 5 minutes

### Step 2: Implement resolveTask() Method
- Add `resolveTask(*AgentRequest) (Task, error)` method
- Implement 3-tier precedence logic
- Add clear error messages
- Add godoc with precedence explanation

**Estimate**: 20 minutes

### Step 3: Write Unit Tests
- Create test file `internal/core/agent_test.go` (or add to existing)
- Write all 6 unit tests (Tests 1-6)
- Ensure ≥90% coverage

**Estimate**: 30 minutes

### Step 4: Write Benchmarks
- Add 3 benchmarks (Benchmarks 1-3)
- Verify performance targets met

**Estimate**: 15 minutes

### Step 5: Quality Checks
- Run `go test ./internal/core/`
- Run `go test -race ./internal/core/`
- Run `make lint`
- Run `go test -bench=. ./internal/core/`
- Fix any issues

**Estimate**: 10 minutes

### Step 6: Documentation
- Update godoc comments
- Update `docs/packages/core.md` with resolution logic explanation
- Add usage example

**Estimate**: 10 minutes

**Total Estimate**: ~1.5 hours

---

## Risks and Mitigations

### Risk 1: Precedence Rules Confusion
**Likelihood**: Medium
**Impact**: Medium (API misuse)
**Mitigation**: Clear godoc, comprehensive tests, examples in docs
**Contingency**: Add FAQ section to documentation

### Risk 2: Registry Lookup Failure
**Likelihood**: Low (well-tested registry)
**Impact**: High (request fails)
**Mitigation**: Clear error messages, validate task names early
**Contingency**: Log failed lookups for debugging

### Risk 3: Nil Registry Edge Case
**Likelihood**: Very Low (constructor ensures non-nil)
**Impact**: High (panic)
**Mitigation**: Defensive nil check in resolveTask if needed
**Contingency**: Return error instead of panic

---

## Success Criteria

### Definition of Done

- [ ] `TaskName` field added to `AgentRequest` struct
- [ ] `resolveTask()` method implemented with 3-tier precedence
- [ ] All 6 unit tests written and passing
- [ ] All 3 benchmarks written and meeting targets
- [ ] Test coverage ≥ 90%
- [ ] `make lint` passes (zero errors)
- [ ] Race detector clean (`go test -race`)
- [ ] Cyclomatic complexity ≤ 5
- [ ] Godoc complete on resolveTask()
- [ ] Documentation updated in `docs/packages/core.md`
- [ ] No breaking changes to existing API

### Acceptance Testing

1. **Smoke Test**: All 4 built-in modes resolve by name
2. **Precedence Test**: Explicit task overrides task name
3. **Default Test**: Empty request gets regular mode
4. **Error Test**: Invalid name returns clear error
5. **Performance Test**: < 5μs for name lookup
6. **Compatibility Test**: All existing tests pass

---

## Dependencies

### Upstream Dependencies (Must Be Complete)
- ✅ P1.1: Task registry in Agent (COMPLETE)
- ✅ `internal/core/task/` - Task interface and registry (EXISTS)

### Downstream Dependencies (Blocked By This)
- P1.3: Tool filtering (needs resolved task)
- P1.4: Token budget application (needs resolved task)
- Phase 2: Conversation integration (will use TaskName)
- Phase 3: CLI integration (will use TaskName)

---

## Open Questions

1. **Q**: Should we log task resolution for debugging?
   **A**: YES - Add debug log at resolution time: `slog.Debug("resolved task", "name", task.Name())`

2. **Q**: Should resolveTask validate the returned task?
   **A**: NO - Assume registry only contains valid tasks (validated at registration)

3. **Q**: Should we cache resolved tasks?
   **A**: NO - Resolution is already O(1) and < 5μs, caching adds complexity

4. **Q**: Should empty string TaskName be treated as "use default" or error?
   **A**: Use default (same as nil) - this is more ergonomic

---

## Revision History

| Date       | Version | Author     | Changes              |
|------------|---------|------------|----------------------|
| 2025-10-12 | 1.0     | Spin Agent | Initial draft        |

---

## Approval

**Author**: Spin Agent
**Reviewers**: [To be assigned]
**Status**: Draft → Ready for Implementation

---

## References

1. [Task Modes Specification](../task-modes/specification.md)
2. [Task Modes Roadmap](../task-modes/ROADMAP.md) - P1.2
3. [AGENTS.md](../../AGENTS.md)
4. [Core Package Docs](../../docs/packages/core.md)
5. [P1.1 FRD](./FRD-20251012130940-task-registry-agent.md)
