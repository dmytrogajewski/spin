# FRD-20251012140000: Conversation Task Mode Integration

**Status**: Draft
**Created**: 2025-10-12
**Author**: Spin Agent
**Related Roadmap**: specs/task-modes/ROADMAP.md - Phase 2
**Related Spec**: specs/task-modes/specification.md

## Overview

Add task mode tracking and switching capability to the Conversation type to enable runtime mode changes and persistence of mode selection across turns.

## Background

Phase 1 (P1.1-P1.5) successfully implemented task mode support in the Agent layer:
- ✅ TaskRegistry added to Agent with 4 built-in modes
- ✅ Task resolution logic (explicit task → task name → default)
- ✅ Tool filtering based on task.AllowedTools()
- ✅ Token budget application from task.MaxTokens()
- ✅ Full test coverage (85%+)

However, the Conversation layer currently has no awareness of task modes. Each turn execution uses the agent's default mode (regular) with no way to:
1. Track the current task mode for a conversation
2. Switch modes mid-conversation
3. Persist mode selection across turns
4. Query the current mode for UI display

## Goals

1. **Track Current Mode**: Conversation maintains current task mode state
2. **Mode Switching**: Support runtime mode switching via SetTaskMode()
3. **Mode Queries**: Provide GetTaskMode() for UI/protocol layers
4. **Thread Safety**: Protect mode state with existing mutex patterns
5. **Persistence**: Mode passes correctly through AgentRequest to Agent
6. **Backward Compatibility**: Default behavior unchanged (regular mode)

## Non-Goals

- Custom mode registration (handled by Agent.taskRegistry)
- Mode auto-selection based on input (future enhancement)
- Per-tool permissions (future enhancement)
- Mode composition or inheritance (future enhancement)

## Requirements

### Functional Requirements

**FR1**: Conversation shall track the current task mode
**FR2**: Conversation shall provide SetTaskMode(taskName string) method
**FR3**: Conversation shall provide GetTaskMode() string method
**FR4**: SetTaskMode shall validate mode name against agent's task registry
**FR5**: SetTaskMode shall return error for invalid mode names
**FR6**: RunTurn/sendMessageInternal shall pass current mode to AgentRequest
**FR7**: Default mode shall be "regular" when not explicitly set
**FR8**: Mode switching shall emit EventTypeSystemInfo event

### Non-Functional Requirements

**NFR1**: Thread safety - all mode access protected by sync.RWMutex
**NFR2**: Performance - GetTaskMode() < 100ns (read-only mutex)
**NFR3**: Performance - SetTaskMode() < 1μs (validation + mutex write)
**NFR4**: Test coverage ≥ 90% for new code
**NFR5**: No breaking changes to existing Conversation API

## Design

### Data Structures

#### Conversation Struct Changes

```go
// File: internal/core/conversation.go

type Conversation struct {
    // ... existing fields ...
    agent   *Agent
    history *History
    emitter *EventEmitter

    // ... existing state fields ...
    mu          sync.RWMutex
    state       State
    turnCancel  context.CancelFunc

    // NEW: Task mode tracking
    currentTask Task   // Current task object (resolved)
    taskName    string // Current task name (for queries)
}
```

**Rationale**:
- Store both Task object (for execution) and string name (for queries/UI)
- Protected by existing `mu sync.RWMutex` (follows existing pattern)
- Task object is resolved once during SetTaskMode to avoid repeated lookups

### API Design

#### SetTaskMode Method

```go
// SetTaskMode switches the conversation to a different task mode.
// Returns an error if the task mode is not registered in the agent's task registry.
//
// This method is thread-safe and can be called concurrently with other operations.
// Mode switching takes effect on the next turn execution.
//
// Example:
//   if err := conv.SetTaskMode("review"); err != nil {
//       return fmt.Errorf("failed to switch mode: %w", err)
//   }
func (c *Conversation) SetTaskMode(taskName string) error {
    // Validate mode exists in agent's registry
    task, err := c.agent.GetTaskRegistry().Get(taskName)
    if err != nil {
        return fmt.Errorf("invalid task mode %q: %w", taskName, err)
    }

    // Validate task
    if err := task.Validate(); err != nil {
        return fmt.Errorf("task %q validation failed: %w", taskName, err)
    }

    // Update state (thread-safe)
    c.mu.Lock()
    c.currentTask = task
    c.taskName = taskName
    c.mu.Unlock()

    // Emit system event for UI/logging
    c.emitter.Emit(Event{
        Type: EventTypeSystemInfo,
        ConversationID: c.ID(),
        Data: SystemEventData{
            Message: fmt.Sprintf("Switched to %s mode", taskName),
        },
    })

    return nil
}
```

#### GetTaskMode Method

```go
// GetTaskMode returns the name of the current task mode.
// Returns "regular" if no mode has been explicitly set.
//
// This method is thread-safe and can be called concurrently with other operations.
//
// Example:
//   currentMode := conv.GetTaskMode()
//   fmt.Printf("Current mode: %s\n", currentMode)
func (c *Conversation) GetTaskMode() string {
    c.mu.RLock()
    defer c.mu.RUnlock()

    // Return explicitly set mode, or default to "regular"
    if c.taskName != "" {
        return c.taskName
    }
    return "regular"
}
```

#### GetCurrentTask Method (Internal)

```go
// getCurrentTask returns the current task object for execution.
// Returns nil if no task is set (agent will use default).
//
// This is an internal helper for sendMessageInternal.
func (c *Conversation) getCurrentTask() Task {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return c.currentTask
}
```

### Integration Points

#### Update sendMessageInternal

```go
// File: internal/core/conversation.go, line ~200

func (c *Conversation) sendMessageInternal(
    turnCtx context.Context,
    userInput string,
    controlChan <-chan ControlSignal,
) (<-chan Event, error) {
    // ... existing validation and setup ...

    var historyMsgs []Message
    if c.history != nil {
        historyMsgs = c.history.MessagesForLLM()
    }

    // Get working directory
    workDir := c.agent.context.WorkDir
    if c.workDir != "" {
        workDir = c.workDir
    }

    // NEW: Get current task mode for this turn
    task := c.getCurrentTask()
    taskName := c.GetTaskMode()

    // Build agent request with task mode
    req := &AgentRequest{
        Input:    userInput,
        History:  historyMsgs,
        Context:  c.agent.context,
        WorkDir:  workDir,
        Task:     task,     // NEW: Pass task object
        TaskName: taskName, // NEW: Pass task name
    }

    // Execute turn with control signal checking
    return c.runTurnWithControl(turnCtx, req, controlChan)
}
```

### Initialization

#### Update NewConversation (No Changes Required)

```go
// File: internal/core/conversation.go

func NewConversation(agent *Agent, history *History, emitter *EventEmitter) *Conversation {
    // ... existing code ...

    conv := &Conversation{
        agent:   agent,
        history: history,
        emitter: emitter,
        // ... other fields ...

        // currentTask and taskName are zero-values (nil, "")
        // This causes GetTaskMode() to return "regular"
        // and agent.resolveTask() to use the default task
    }

    // ... rest of initialization ...
    return conv
}
```

**Rationale**: Zero-value initialization is intentional. Empty string → "regular" default.

### Error Handling

#### Error Cases

1. **Invalid Task Name**
   ```go
   err := conv.SetTaskMode("invalid")
   // Returns: invalid task mode "invalid": task not found
   ```

2. **Task Validation Failure**
   ```go
   // If task.Validate() fails (edge case)
   // Returns: task "mode-name" validation failed: <reason>
   ```

3. **Nil Agent Registry** (defensive)
   ```go
   // Should never happen, but defensive check
   if c.agent.GetTaskRegistry() == nil {
       return errors.New("agent task registry not initialized")
   }
   ```

### Thread Safety Analysis

#### Mutex Usage

```
Read Operations (RLock):
- GetTaskMode()           ✅ Uses c.mu.RLock()
- getCurrentTask()        ✅ Uses c.mu.RLock()

Write Operations (Lock):
- SetTaskMode()           ✅ Uses c.mu.Lock()

No Mutex Required:
- agent.GetTaskRegistry() ✅ TaskRegistry has internal RWMutex
```

#### Race Condition Prevention

**Scenario 1: Mode switch during turn execution**
```
Thread A: Executing turn (reading c.currentTask)
Thread B: SetTaskMode() (writing c.currentTask)

Protection: c.mu.RLock() in getCurrentTask() blocks until SetTaskMode() completes
Result: Turn uses either old or new mode consistently
```

**Scenario 2: Concurrent GetTaskMode() calls**
```
Thread A: GetTaskMode() (c.mu.RLock())
Thread B: GetTaskMode() (c.mu.RLock())

Protection: Multiple readers allowed (RWMutex)
Result: Both threads read safely
```

**Scenario 3: Concurrent SetTaskMode() calls**
```
Thread A: SetTaskMode("review") (c.mu.Lock())
Thread B: SetTaskMode("compact") (c.mu.Lock())

Protection: Mutex serializes writes
Result: Last write wins (expected behavior)
```

## Testing Strategy

### Unit Tests

#### Test 1: SetTaskMode Success
```go
func TestConversation_SetTaskMode(t *testing.T) {
    conv := setupTestConversation(t)

    // Should default to "regular"
    assert.Equal(t, "regular", conv.GetTaskMode())

    // Switch to review mode
    err := conv.SetTaskMode("review")
    assert.NoError(t, err)
    assert.Equal(t, "review", conv.GetTaskMode())

    // Switch to compact mode
    err = conv.SetTaskMode("compact")
    assert.NoError(t, err)
    assert.Equal(t, "compact", conv.GetTaskMode())
}
```

#### Test 2: SetTaskMode Invalid Mode
```go
func TestConversation_SetTaskMode_Invalid(t *testing.T) {
    conv := setupTestConversation(t)

    err := conv.SetTaskMode("invalid-mode")
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "invalid task mode")

    // Should remain in default mode
    assert.Equal(t, "regular", conv.GetTaskMode())
}
```

#### Test 3: GetTaskMode Default
```go
func TestConversation_GetTaskMode_Default(t *testing.T) {
    conv := setupTestConversation(t)

    // Should default to "regular" without explicit SetTaskMode
    assert.Equal(t, "regular", conv.GetTaskMode())
}
```

#### Test 4: Concurrent Mode Access
```go
func TestConversation_TaskMode_Concurrent(t *testing.T) {
    conv := setupTestConversation(t)

    var wg sync.WaitGroup

    // 50 concurrent readers
    for i := 0; i < 50; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            _ = conv.GetTaskMode()
        }()
    }

    // 10 concurrent writers
    modes := []string{"regular", "review", "compact", "planning"}
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func(i int) {
            defer wg.Done()
            mode := modes[i%len(modes)]
            _ = conv.SetTaskMode(mode)
        }(i)
    }

    wg.Wait()

    // Should not race (verified with go test -race)
}
```

#### Test 5: Mode Persists Across Turns
```go
func TestConversation_TaskMode_PersistsAcrossTurns(t *testing.T) {
    conv := setupTestConversation(t)

    // Set mode to compact
    err := conv.SetTaskMode("compact")
    assert.NoError(t, err)

    // Execute turn 1
    err = conv.RunTurn(context.Background(), "Turn 1")
    assert.NoError(t, err)

    // Mode should still be compact
    assert.Equal(t, "compact", conv.GetTaskMode())

    // Execute turn 2
    err = conv.RunTurn(context.Background(), "Turn 2")
    assert.NoError(t, err)

    // Mode should still be compact
    assert.Equal(t, "compact", conv.GetTaskMode())
}
```

### Integration Tests

#### Test 6: Mode Affects Tool Availability
```go
func TestConversation_TaskMode_ToolFiltering(t *testing.T) {
    conv := setupTestConversation(t)

    // Start in regular mode (all tools)
    // (Would need to inspect events or mock LLM to verify)

    // Switch to review mode (read-only tools)
    err := conv.SetTaskMode("review")
    assert.NoError(t, err)

    // Execute turn - verify only read tools available
    // (Check via EventTypeToolCallStart events)
}
```

#### Test 7: Mode Affects Token Budget
```go
func TestConversation_TaskMode_TokenBudget(t *testing.T) {
    conv := setupTestConversation(t)

    // Compact mode has 4096 token limit
    err := conv.SetTaskMode("compact")
    assert.NoError(t, err)

    // Execute turn and verify token budget applied
    // (Would need to mock LLM and inspect request)
}
```

### Test Coverage Target

- **New Methods**: 100% coverage (SetTaskMode, GetTaskMode, getCurrentTask)
- **Modified Methods**: Maintain existing coverage (sendMessageInternal)
- **Overall Conversation Package**: ≥ 85%

## Performance Considerations

### Benchmarks

```go
func BenchmarkConversation_GetTaskMode(b *testing.B) {
    conv := setupTestConversation(b)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = conv.GetTaskMode()
    }
}
// Target: < 100ns/op

func BenchmarkConversation_SetTaskMode(b *testing.B) {
    conv := setupTestConversation(b)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = conv.SetTaskMode("review")
    }
}
// Target: < 1000ns/op (1μs)
```

### Memory Impact

- **Per Conversation**: +16 bytes (Task interface) + ~16 bytes (string)
- **Total Overhead**: ~32 bytes per conversation
- **Acceptable**: ✅ Well under 10KB target

## Migration & Compatibility

### Backward Compatibility

✅ **No Breaking Changes**:
- Existing Conversation API unchanged
- New methods are additive
- Default behavior preserved ("regular" mode)
- Zero-value initialization works correctly

### Deprecations

None.

### Migration Path

Existing code continues to work without changes:
```go
// Before: Uses default "regular" mode
conv := manager.NewConversation(ctx, workDir)
err := conv.RunTurn(ctx, "Fix the bug")

// After: Still uses default "regular" mode
conv := manager.NewConversation(ctx, workDir)
err := conv.RunTurn(ctx, "Fix the bug")

// New capability: Explicit mode switching
err = conv.SetTaskMode("review")
err = conv.RunTurn(ctx, "Review this code")
```

## Security Considerations

### Threat: Unauthorized Mode Switching

**Risk**: Malicious client switches to "regular" mode to gain write access
**Mitigation**:
- Mode validation against registered modes only
- All mode switches logged via system events
- Future: Add approval requirement for mode switches

### Threat: Mode Name Injection

**Risk**: Invalid characters in mode name
**Mitigation**:
- TaskRegistry.Get() validates names
- Error returned for invalid names
- No string interpolation in critical paths

## Open Questions

1. **Q**: Should mode switching during active turn execution be blocked?
   **A**: No, not in P2.1. Mode takes effect on next turn. Future: Add state check.

2. **Q**: Should we emit a dedicated EventTypeTaskModeChanged event?
   **A**: Not in P2.1. Using EventTypeSystemInfo is sufficient. Future enhancement.

3. **Q**: Should GetTaskMode() return the Task object or just the name?
   **A**: Just the name. UI/protocol layers need string, not Task interface.

## Acceptance Criteria

- [ ] Conversation has currentTask and taskName fields
- [ ] SetTaskMode() validates mode and updates state
- [ ] GetTaskMode() returns current mode name or "regular" default
- [ ] sendMessageInternal() passes task to AgentRequest
- [ ] All unit tests pass (6 tests minimum)
- [ ] Race detector clean (go test -race)
- [ ] Test coverage ≥ 90% for new code
- [ ] make lint passes with zero errors
- [ ] Godoc complete for all new methods
- [ ] Benchmark shows GetTaskMode() < 100ns

## Dependencies

### Blocked By
- None (Phase 1 complete)

### Blocks
- P2.2: Update Manager for Task Support
- P2.3: Integration Tests for Conversation

### Related
- P1.1: Add Task Registry to Agent (complete)
- P1.2: Implement Task Resolution Logic (complete)

## References

- [specs/task-modes/ROADMAP.md](../../task-modes/ROADMAP.md) - Phase 2 roadmap
- [specs/task-modes/specification.md](../../task-modes/specification.md) - Full spec
- [internal/core/conversation.go](../../../internal/core/conversation.go) - Implementation file
- [internal/core/agent.go](../../../internal/core/agent.go) - Agent task registry
- [AGENTS.md](../../../AGENTS.md) - Development standards

## Approval

**Author**: Spin Agent
**Reviewed**: TBD
**Approved**: TBD
**Status**: Draft → Pending Review

---

**Last Updated**: 2025-10-12
