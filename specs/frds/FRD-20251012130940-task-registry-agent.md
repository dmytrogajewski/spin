# Feature Requirements Document: Add Task Registry to Agent

**FRD-20251012130940**

**Status**: Draft
**Created**: 2025-10-12
**Author**: Spin Agent
**Related**: [P1.1] Task Mode System Integration - Phase 1
**Priority**: CRITICAL - Blocks all other task mode work
**Estimated Effort**: 2-3 hours

## Overview

This FRD describes the implementation of P1.1 from the task-modes roadmap: adding TaskRegistry to the Agent struct and wiring it through the constructor with default task registration.

## Context

### Current State

The Spin agent has a fully implemented task mode system in `internal/core/task/`:
- Task interface with 4 built-in modes (regular, review, compact, planning)
- Registry implementation with thread-safe operations
- Complete test coverage

**However**: The task system is never used because:
1. Agent struct has no `taskRegistry` field
2. Agent constructor never initializes tasks
3. No way to specify which task mode to use

### Problem Statement

Without TaskRegistry integration, the agent:
- Always uses default behavior (no mode switching)
- Cannot restrict tools based on task mode
- Cannot apply task-specific token budgets
- Cannot use task-specific system prompts
- Wastes tokens and costs in scenarios where compact mode would suffice

### Goals

1. Add `taskRegistry` field to Agent struct
2. Initialize registry with 4 built-in modes in `NewAgent()`
3. Provide `WithTaskRegistry()` option for custom registries
4. Maintain backward compatibility (no breaking changes)
5. Follow existing patterns (ToolRegistry precedent)
6. Ensure thread safety
7. Achieve ≥90% test coverage

### Non-Goals

- Task resolution logic (P1.2)
- Tool filtering (P1.3)
- Token budget application (P1.4)
- CLI integration (Phase 3)
- AppServer integration (Phase 4)

## Requirements

### Functional Requirements

#### FR1: Add taskRegistry Field to Agent Struct

**Priority**: CRITICAL

**Description**: Add a `taskRegistry` field to the Agent struct to store and manage available task modes.

**Acceptance Criteria**:
```go
type Agent struct {
    llm             llm.Provider
    executor        *Executor
    validator       *Validator
    context         *Environment
    emitter         *EventEmitter
    config          *AgentConfig
    toolRegistry    *tools.Registry
    taskRegistry    *task.Registry  // NEW: Task registry
    approvalHandler ApprovalHandler
}
```

**Validation**:
- Field must be exported (capitalized) for testing: NO - follow existing pattern (lowercase)
- Field must be after toolRegistry for consistency: YES
- Field type must be pointer to allow nil checking: YES

---

#### FR2: Initialize Default Task Registry in NewAgent()

**Priority**: CRITICAL

**Description**: Create and initialize a task registry with all 4 built-in modes when constructing a new Agent.

**Acceptance Criteria**:
```go
func NewAgent(
    llm llm.Provider,
    executor *Executor,
    validator *Validator,
    context *Environment,
    emitter *EventEmitter,
    opts ...AgentOption,
) (*Agent, error) {
    // ... existing validation ...

    // NEW: Create default task registry
    taskRegistry := task.NewRegistry()
    if err := taskRegistry.Register("regular", task.NewRegular()); err != nil {
        return nil, fmt.Errorf("failed to register regular task: %w", err)
    }
    if err := taskRegistry.Register("review", task.NewReview()); err != nil {
        return nil, fmt.Errorf("failed to register review task: %w", err)
    }
    if err := taskRegistry.Register("compact", task.NewCompact()); err != nil {
        return nil, fmt.Errorf("failed to register compact task: %w", err)
    }
    if err := taskRegistry.Register("planning", task.NewPlanning()); err != nil {
        return nil, fmt.Errorf("failed to register planning task: %w", err)
    }
    if err := taskRegistry.SetDefault("regular"); err != nil {
        return nil, fmt.Errorf("failed to set default task: %w", err)
    }

    agent := &Agent{
        llm:          llm,
        executor:     executor,
        validator:    validator,
        context:      context,
        emitter:      emitter,
        toolRegistry: tools.NewRegistry(),
        taskRegistry: taskRegistry,  // NEW: Initialize task registry
        config: &AgentConfig{
            MaxTurns:        DefaultMaxTurns,
            Timeout:         DefaultAgentTimeout,
            Temperature:     DefaultTemperature,
            MaxTokens:       DefaultMaxTokens,
            RequireApproval: false,
            ApprovalTimeout: DefaultApprovalTimeout,
        },
    }

    // Apply options
    for _, opt := range opts {
        if err := opt(agent); err != nil {
            return nil, err
        }
    }

    return agent, nil
}
```

**Validation**:
- All 4 modes must be registered: regular, review, compact, planning
- Default must be set to "regular" (maintains current behavior)
- Registration errors must be handled and returned
- Must happen before applying options (so custom registry can override)

---

#### FR3: Add WithTaskRegistry() AgentOption

**Priority**: HIGH

**Description**: Provide a functional option to allow users to supply custom task registries.

**Acceptance Criteria**:
```go
// WithTaskRegistry sets a custom task registry for the agent.
// If nil, returns an error.
// Example:
//   customRegistry := task.NewRegistry()
//   customRegistry.Register("custom", myCustomTask)
//   agent := NewAgent(llm, exec, val, ctx, emitter, WithTaskRegistry(customRegistry))
func WithTaskRegistry(registry *task.Registry) AgentOption {
    return func(a *Agent) error {
        if registry == nil {
            return fmt.Errorf("task registry cannot be nil")
        }
        a.taskRegistry = registry
        return nil
    }
}
```

**Validation**:
- Must follow AgentOption signature: `func(*Agent) error`
- Must validate input (reject nil)
- Must override default registry when applied
- Must include godoc comment with example
- Must be consistent with existing options (WithApprovalHandler, etc.)

---

#### FR4: Registry Access Methods (Optional)

**Priority**: LOW (Nice to have)

**Description**: Add helper methods to access the task registry for testing/debugging.

**Acceptance Criteria**:
```go
// GetTaskRegistry returns the agent's task registry.
// Useful for testing and introspection.
func (a *Agent) GetTaskRegistry() *task.Registry {
    return a.taskRegistry
}

// ListTaskModes returns all registered task mode names.
func (a *Agent) ListTaskModes() []string {
    if a.taskRegistry == nil {
        return nil
    }
    return a.taskRegistry.List()
}
```

**Validation**:
- Thread-safe (no mutex needed as Registry itself is thread-safe)
- Returns nil for nil registry (defensive)
- Godoc comments explain usage

---

### Non-Functional Requirements

#### NFR1: Thread Safety

**Description**: TaskRegistry access must be thread-safe.

**Acceptance Criteria**:
- Use `task.Registry` which already has `sync.RWMutex` internally
- No additional locking needed in Agent struct
- Must pass `-race` detector tests

**Validation**:
```bash
go test -race ./internal/core/
```

---

#### NFR2: Backward Compatibility

**Description**: Changes must not break existing code.

**Acceptance Criteria**:
- No changes to existing Agent method signatures
- Default behavior unchanged (regular mode = current behavior)
- All existing tests must pass without modification

**Validation**:
```bash
go test ./internal/core/
```

---

#### NFR3: Performance

**Description**: Minimal overhead from TaskRegistry addition.

**Acceptance Criteria**:
- Agent construction overhead: < 10μs
- Memory overhead: < 5KB per agent
- No measurable latency in execution path

**Validation**:
```bash
go test -bench=BenchmarkNewAgent ./internal/core/
```

---

#### NFR4: Code Quality

**Description**: Code must meet project quality standards.

**Acceptance Criteria**:
- `make lint` passes (zero errors)
- Cyclomatic complexity ≤ 15
- Test coverage ≥ 90%
- All public functions have godoc comments
- No dead code
- uast/herr analysis at least YELLOW

**Validation**:
```bash
make lint
go test -cover ./internal/core/
uast parse internal/core/agent.go | herr analyze
```

---

## Technical Design

### Architecture

```
┌─────────────────────────────────────────────────┐
│                 NewAgent()                      │
│  1. Validate inputs                             │
│  2. Create task.Registry                        │
│  3. Register 4 built-in modes                   │
│  4. Set "regular" as default                    │
│  5. Create Agent with taskRegistry              │
│  6. Apply functional options                    │
└─────────────────────────────────────────────────┘
                        │
                        v
┌─────────────────────────────────────────────────┐
│              Agent Struct                       │
│  - taskRegistry *task.Registry                  │
│    - Thread-safe via internal RWMutex           │
│    - Contains 4 built-in modes by default       │
│    - "regular" is default mode                  │
└─────────────────────────────────────────────────┘
                        │
                        v
┌─────────────────────────────────────────────────┐
│        WithTaskRegistry(registry)               │
│  - Replaces default registry                    │
│  - Allows custom task modes                     │
│  - Validates registry not nil                   │
└─────────────────────────────────────────────────┘
```

### Data Structures

```go
// Agent struct (existing + new field)
type Agent struct {
    llm             llm.Provider
    executor        *Executor
    validator       *Validator
    context         *Environment
    emitter         *EventEmitter
    config          *AgentConfig
    toolRegistry    *tools.Registry
    taskRegistry    *task.Registry  // NEW
    approvalHandler ApprovalHandler
}

// task.Registry (existing, already implemented)
type Registry struct {
    tasks       map[string]Task
    defaultTask string
    mu          sync.RWMutex
}

// Methods already exist:
// - Register(name string, task Task) error
// - Get(name string) (Task, error)
// - List() []string
// - SetDefault(name string) error
// - GetDefault() (Task, error)
```

### Key Algorithms

#### Algorithm 1: Agent Initialization

```
FUNCTION NewAgent(llm, executor, validator, context, emitter, opts...):
    1. Validate all required inputs (existing)

    2. Create task registry:
       registry = task.NewRegistry()

    3. Register built-in modes:
       registry.Register("regular", task.NewRegular())
       registry.Register("review", task.NewReview())
       registry.Register("compact", task.NewCompact())
       registry.Register("planning", task.NewPlanning())

    4. Set default:
       registry.SetDefault("regular")

    5. Create agent:
       agent = Agent{
           llm: llm,
           executor: executor,
           validator: validator,
           context: context,
           emitter: emitter,
           toolRegistry: tools.NewRegistry(),
           taskRegistry: registry,  // NEW
           config: defaultConfig,
       }

    6. Apply options:
       FOR EACH opt IN opts:
           err = opt(agent)
           IF err != nil:
               RETURN nil, err

    7. RETURN agent, nil
```

**Complexity**: O(1) - constant number of registrations

---

#### Algorithm 2: Custom Registry Option

```
FUNCTION WithTaskRegistry(registry):
    RETURN func(agent):
        IF registry == nil:
            RETURN error("task registry cannot be nil")
        agent.taskRegistry = registry
        RETURN nil
```

**Complexity**: O(1)

---

### Error Handling

#### Error Scenarios

1. **Registration Failure**
   - Cause: Built-in mode fails to register (should never happen)
   - Handling: Return error from NewAgent(), prevent agent creation
   - Recovery: Fix code bug (this is a developer error)

2. **Nil Registry in Option**
   - Cause: User passes nil to WithTaskRegistry()
   - Handling: Return error from option application
   - Recovery: User provides valid registry

3. **SetDefault Failure**
   - Cause: Default task name not registered (should never happen)
   - Handling: Return error from NewAgent()
   - Recovery: Fix code bug

#### Error Messages

```go
// Clear, actionable error messages
"failed to register regular task: %w"
"failed to register review task: %w"
"failed to register compact task: %w"
"failed to register planning task: %w"
"failed to set default task: %w"
"task registry cannot be nil"
```

---

### Testing Strategy

#### Unit Tests

**Test 1: Agent Initialization**
```go
func TestNewAgent_InitializesTaskRegistry(t *testing.T) {
    agent, err := NewAgent(mockLLM, mockExec, mockVal, mockCtx, mockEmitter)
    require.NoError(t, err)
    require.NotNil(t, agent.taskRegistry)

    // Verify all 4 modes registered
    modes := agent.taskRegistry.List()
    require.Len(t, modes, 4)
    require.Contains(t, modes, "regular")
    require.Contains(t, modes, "review")
    require.Contains(t, modes, "compact")
    require.Contains(t, modes, "planning")

    // Verify default is "regular"
    defaultTask, err := agent.taskRegistry.GetDefault()
    require.NoError(t, err)
    require.Equal(t, "regular", defaultTask.Name())
}
```

**Test 2: Custom Registry Option**
```go
func TestWithTaskRegistry_CustomRegistry(t *testing.T) {
    customRegistry := task.NewRegistry()
    customTask := &mockTask{name: "custom"}
    err := customRegistry.Register("custom", customTask)
    require.NoError(t, err)

    agent, err := NewAgent(
        mockLLM, mockExec, mockVal, mockCtx, mockEmitter,
        WithTaskRegistry(customRegistry),
    )
    require.NoError(t, err)

    // Verify custom registry used
    modes := agent.taskRegistry.List()
    require.Contains(t, modes, "custom")
}
```

**Test 3: Nil Registry Rejected**
```go
func TestWithTaskRegistry_RejectsNil(t *testing.T) {
    agent, err := NewAgent(
        mockLLM, mockExec, mockVal, mockCtx, mockEmitter,
        WithTaskRegistry(nil),
    )
    require.Error(t, err)
    require.Nil(t, agent)
    require.Contains(t, err.Error(), "task registry cannot be nil")
}
```

**Test 4: Thread Safety**
```go
func TestAgent_TaskRegistry_ThreadSafe(t *testing.T) {
    agent := newTestAgent(t)

    var wg sync.WaitGroup
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            // Concurrent reads
            modes := agent.taskRegistry.List()
            require.NotEmpty(t, modes)
        }()
    }
    wg.Wait()
}
```

**Test 5: Helper Methods**
```go
func TestAgent_GetTaskRegistry(t *testing.T) {
    agent := newTestAgent(t)
    registry := agent.GetTaskRegistry()
    require.NotNil(t, registry)
    require.Equal(t, agent.taskRegistry, registry)
}

func TestAgent_ListTaskModes(t *testing.T) {
    agent := newTestAgent(t)
    modes := agent.ListTaskModes()
    require.Len(t, modes, 4)
    require.Equal(t, []string{"compact", "planning", "regular", "review"}, modes)
}
```

#### Integration Tests

**Test 6: End-to-End Agent Creation**
```go
func TestAgent_E2E_WithTaskRegistry(t *testing.T) {
    // Full agent creation with all options
    agent, err := NewAgent(
        realLLMProvider,
        NewExecutor(),
        NewValidator(),
        NewEnvironment(),
        NewEventEmitter(),
        WithTaskRegistry(task.NewRegistry()),
    )
    require.NoError(t, err)
    require.NotNil(t, agent)

    // Verify agent is fully functional
    // (This will be tested more in P1.2 when task resolution is added)
}
```

#### Benchmarks

**Benchmark 1: Agent Construction**
```go
func BenchmarkNewAgent_WithDefaultTaskRegistry(b *testing.B) {
    for i := 0; i < b.N; i++ {
        agent, err := NewAgent(
            mockLLM, mockExec, mockVal, mockCtx, mockEmitter,
        )
        if err != nil {
            b.Fatal(err)
        }
        _ = agent
    }
}
```

**Expected**: < 10μs per agent creation

---

## Implementation Plan

### Step 1: Add Field to Agent Struct
- Modify `internal/core/agent.go`
- Add `taskRegistry *task.Registry` field
- Position after `toolRegistry` for consistency

### Step 2: Update NewAgent() Constructor
- Create default task registry
- Register 4 built-in modes
- Set "regular" as default
- Add error handling for registration failures
- Initialize agent.taskRegistry field

### Step 3: Add WithTaskRegistry() Option
- Create functional option following existing pattern
- Validate input (reject nil)
- Add godoc with example

### Step 4: Add Helper Methods (Optional)
- Add GetTaskRegistry()
- Add ListTaskModes()
- Add godoc comments

### Step 5: Write Tests
- Write all unit tests (Tests 1-5)
- Write integration test (Test 6)
- Write benchmark (Benchmark 1)
- Ensure ≥90% coverage

### Step 6: Run Quality Checks
- Run `go test -race ./internal/core/`
- Run `make lint`
- Run `uast parse internal/core/agent.go | herr analyze`
- Fix any issues

### Step 7: Update Documentation
- Update godoc comments
- Update `docs/packages/core.md` with task registry info
- Add example usage

---

## Risks and Mitigations

### Risk 1: Breaking Existing Tests
**Likelihood**: Medium
**Impact**: High
**Mitigation**: Run all existing tests before and after changes
**Contingency**: Revert changes, analyze failures, fix

### Risk 2: Registration Errors
**Likelihood**: Low (built-in modes are well-tested)
**Impact**: High (agent won't initialize)
**Mitigation**: Handle errors explicitly, add unit tests
**Contingency**: Fix registration logic

### Risk 3: Performance Regression
**Likelihood**: Low
**Impact**: Medium
**Mitigation**: Benchmark before/after, registry creation is O(1)
**Contingency**: Optimize registry initialization if needed

### Risk 4: Thread Safety Issues
**Likelihood**: Low (using existing thread-safe Registry)
**Impact**: High (race conditions)
**Mitigation**: Run with `-race` detector, use existing RWMutex
**Contingency**: Add synchronization if needed

---

## Success Criteria

### Definition of Done

- [x] taskRegistry field added to Agent struct
- [x] NewAgent() initializes registry with 4 modes
- [x] WithTaskRegistry() option implemented
- [x] Helper methods implemented (optional)
- [x] All unit tests written and passing
- [x] Test coverage ≥90%
- [x] `make lint` passes (zero errors)
- [x] Race detector clean (`go test -race`)
- [x] Complexity ≤15 for all functions
- [x] Godoc on all exports
- [x] No dead code
- [x] Documentation updated

### Acceptance Testing

1. **Smoke Test**: Agent creates successfully with default registry
2. **Functional Test**: All 4 modes are registered
3. **Default Test**: "regular" is the default mode
4. **Custom Test**: WithTaskRegistry() overrides default
5. **Error Test**: Nil registry is rejected
6. **Thread Test**: Concurrent access works correctly
7. **Performance Test**: No measurable overhead

---

## Dependencies

### Upstream Dependencies
- `internal/core/task/` - Task interface and built-in modes (✅ exists)
- `internal/core/task/` - Registry implementation (✅ exists)
- Agent struct and NewAgent() (✅ exists)

### Downstream Dependencies (Blocked By This)
- P1.2: Task resolution logic
- P1.3: Tool filtering
- P1.4: Token budget application
- All Phase 2-5 work

---

## Open Questions

1. **Q**: Should GetTaskRegistry() be exported?
   **A**: Yes, needed for testing and future extensibility

2. **Q**: Should we validate that "regular" mode exists before setting it as default?
   **A**: Yes, defensive programming - add check after registration

3. **Q**: What if a built-in mode fails to register?
   **A**: Return error from NewAgent() - this should never happen, but handle defensively

4. **Q**: Should we cache the default task reference?
   **A**: No, GetDefault() is fast enough and keeps logic simple

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
2. [Task Modes Roadmap](../task-modes/ROADMAP.md)
3. [AGENTS.md](../../AGENTS.md)
4. [Core Package Docs](../../docs/packages/core.md)
5. [Existing Task Implementation](../../internal/core/task/)
