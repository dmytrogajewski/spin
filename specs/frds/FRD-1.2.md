# FRD-1.2: Turn State Machine

**Feature ID:** 1.2  
**Feature Name:** Turn State Machine  
**Phase:** Phase 1 - State Management  
**Priority:** P0 (Blocker)  
**Estimated Effort:** 10 hours  
**Status:** Ready for Implementation

---

## Overview

Implement comprehensive turn state management with state transitions, turn execution tracking, and result handling. This feature builds upon the minimal Turn implementation from Feature 1.1 and provides a robust state machine for managing the lifecycle of individual user-AI interactions.

## Context

A **Turn** represents a single user-AI interaction cycle within a conversation. Each turn has a well-defined lifecycle with state transitions, execution tracking, and result capture. This feature is critical for:

- Tracking turn execution progress
- Enabling turn resumption after interruptions
- Supporting approval workflows (user confirmation for dangerous commands)
- Providing detailed execution history
- Implementing proper error handling and recovery

## Definition of Ready (DoR)

- [x] Feature 1.1 (Session Management) completed
- [ ] Turn state machine diagram defined
- [ ] State transition rules documented
- [ ] Token usage tracking requirements clarified

## Definition of Done (DoD)

- [ ] `turn/turn.go` fully implemented with Turn struct
- [ ] `turn/state.go` with TurnState enum and transition validation
- [ ] `turn/result.go` with comprehensive turn execution results
- [ ] All TurnState constants defined and documented
- [ ] State transition validation implemented
- [ ] Turn ID generation (UUIDs)
- [ ] Token usage tracking (prompt, completion, total)
- [ ] Timestamp tracking (start/complete)
- [ ] Unit tests for state machine (>90% coverage)
- [ ] State transition tests (all valid/invalid paths)
- [ ] Turn serialization/deserialization tests
- [ ] Godoc comments for all exported symbols
- [ ] Code analyzed with uast/herr (complexity <15)
- [ ] All linters passing

---

## Requirements

### Functional Requirements

#### FR-1.2.1: Turn State Enum

**Description:** Define comprehensive turn states covering the complete lifecycle.

**States:**

1. **Pending** - Turn created but not yet started
2. **Running** - Turn is currently executing
3. **WaitingApproval** - Paused, waiting for user approval of a command
4. **Completed** - Turn completed successfully
5. **Failed** - Turn failed with an error
6. **Cancelled** - Turn was cancelled by user

**Acceptance Criteria:**
- All states defined as constants
- String representation for each state
- State description documentation

---

#### FR-1.2.2: State Transitions

**Description:** Implement and validate state transition rules.

**Valid Transitions:**

```
Pending → Running
Running → WaitingApproval
Running → Completed
Running → Failed
Running → Cancelled
WaitingApproval → Running (after approval)
WaitingApproval → Cancelled (if denied)
```

**Invalid Transitions:**
- Any transition from Completed, Failed, or Cancelled (terminal states)
- Direct transition from Pending to WaitingApproval
- Direct transition from Pending to Completed/Failed/Cancelled

**Acceptance Criteria:**
- `CanTransition(from, to TurnState) bool` function
- `Transition(to TurnState) error` method on Turn
- Error returned for invalid transitions
- All valid transitions allowed
- All invalid transitions rejected

---

#### FR-1.2.3: Turn Struct

**Description:** Complete Turn struct with all required fields.

**Fields:**

```go
type Turn struct {
    // Identity
    ID          string    // UUID v4
    SessionID   string    // Parent session ID
    
    // Content
    UserInput   string    // User's input message
    AIResponse  string    // AI's accumulated response
    
    // Tool Execution
    ToolCalls   []ToolCall   // Tools invoked during turn
    ToolResults []ToolResult // Results from tool execution
    
    // State
    State       TurnState // Current state
    Error       error     // Error if State == Failed
    
    // Timing
    StartedAt   time.Time // When turn started
    CompletedAt time.Time // When turn completed/failed/cancelled
    
    // Metrics
    Tokens      TokenUsage // Token consumption tracking
    
    // Metadata
    Metadata    map[string]interface{} // Extensible metadata
}
```

**Acceptance Criteria:**
- All fields properly typed
- Godoc comments for each field
- JSON struct tags for serialization
- Proper zero values

---

#### FR-1.2.4: Tool Call Tracking

**Description:** Track tool invocations and results within a turn.

**Types:**

```go
type ToolCall struct {
    ID       string                 // Tool call ID from LLM
    Name     string                 // Tool name
    Args     map[string]interface{} // Tool arguments
    CallTime time.Time              // When tool was called
}

type ToolResult struct {
    ToolCallID string      // Matching ToolCall.ID
    Result     interface{} // Tool execution result
    Error      error       // Error if tool failed
    Duration   time.Duration // Execution time
}
```

**Acceptance Criteria:**
- ToolCall captures LLM tool invocation
- ToolResult captures execution result
- AddToolCall() method
- AddToolResult() method
- Results matched to calls by ID

---

#### FR-1.2.5: Token Usage Tracking

**Description:** Track token consumption for cost monitoring and context management.

**Type:**

```go
type TokenUsage struct {
    PromptTokens     int // Tokens in prompt
    CompletionTokens int // Tokens in completion
    TotalTokens      int // Total tokens used
}
```

**Methods:**
- `UpdateTokens(usage TokenUsage)` - Update token counts
- `GetTotalTokens() int` - Retrieve total token usage

**Acceptance Criteria:**
- Accurate token tracking
- Proper accumulation for multi-turn LLM calls
- Thread-safe token updates (if concurrent)

---

#### FR-1.2.6: Turn Execution Results

**Description:** Comprehensive result structure for turn execution.

**Type:**

```go
type Result struct {
    // Outcome
    Success      bool   // Whether turn succeeded
    FinalState   TurnState // Final turn state
    Error        error  // Error if failed
    
    // Response
    Response     string // Final AI response
    
    // Metrics
    Duration     time.Duration // Total execution time
    Tokens       TokenUsage    // Token usage
    ToolCount    int           // Number of tools called
    
    // Context
    ContextSize  int // Size of context used (bytes)
    Truncated    bool // Whether context was truncated
}
```

**Methods:**
- `NewResult(turn *Turn) *Result` - Create result from turn
- `IsSuccess() bool` - Check if successful
- `GetError() error` - Retrieve error

**Acceptance Criteria:**
- Complete result information
- Easy result inspection
- Serializable to JSON

---

#### FR-1.2.7: Turn Lifecycle Methods

**Description:** Methods for managing turn lifecycle.

**Methods:**

```go
// NewTurn creates a new turn
func NewTurn(sessionID, userInput string) *Turn

// Start transitions turn to Running state
func (t *Turn) Start() error

// Complete marks turn as successfully completed
func (t *Turn) Complete(response string, tokens TokenUsage) error

// Fail marks turn as failed with error
func (t *Turn) Fail(err error) error

// Cancel marks turn as cancelled
func (t *Turn) Cancel() error

// RequestApproval transitions to WaitingApproval
func (t *Turn) RequestApproval() error

// Approve transitions from WaitingApproval back to Running
func (t *Turn) Approve() error

// Deny transitions from WaitingApproval to Cancelled
func (t *Turn) Deny() error
```

**Acceptance Criteria:**
- All lifecycle methods implemented
- Proper state transitions
- Timestamp updates
- Error handling for invalid transitions

---

### Non-Functional Requirements

#### NFR-1.2.1: Performance
- Turn creation: <1ms
- State transition: <100μs
- Serialization: <10ms for typical turn

#### NFR-1.2.2: Thread Safety
- Turn state transitions must be thread-safe
- Use mutex for state changes
- Concurrent reads allowed

#### NFR-1.2.3: Memory
- Typical turn: <10KB in memory
- Large turn (with tool calls): <100KB

#### NFR-1.2.4: Testability
- >90% test coverage
- All state transitions tested
- All error paths tested
- Race detector clean

---

## Design

### State Machine Diagram

```
                    Start()
    Pending ────────────────────────► Running
                                         │
                                         │ RequestApproval()
                                         ├──────────────────► WaitingApproval
                                         │                         │
                                         │                         │ Approve()
                                         │                         └────────► Running
                                         │                         
                                         │                         │ Deny()
                                         │                         └────────► Cancelled
                                         │
                                         │ Complete()
                                         ├──────────────────────────────────► Completed
                                         │
                                         │ Fail()
                                         ├──────────────────────────────────► Failed
                                         │
                                         │ Cancel()
                                         └──────────────────────────────────► Cancelled
```

### File Structure

```
internal/core/turn/
├── turn.go           # Turn struct and lifecycle methods
├── state.go          # TurnState enum and transitions
├── result.go         # Result struct and methods
├── turn_test.go      # Turn tests
├── state_test.go     # State transition tests
└── result_test.go    # Result tests
```

### Key Algorithms

#### State Transition Validation

```go
func (t *Turn) canTransition(to TurnState) bool {
    transitions := map[TurnState][]TurnState{
        StatePending: {StateRunning},
        StateRunning: {StateWaitingApproval, StateCompleted, StateFailed, StateCancelled},
        StateWaitingApproval: {StateRunning, StateCancelled},
        // Terminal states cannot transition
        StateCompleted: {},
        StateFailed: {},
        StateCancelled: {},
    }
    
    validTargets := transitions[t.State]
    for _, valid := range validTargets {
        if valid == to {
            return true
        }
    }
    return false
}
```

---

## Implementation Plan

### Task Breakdown

#### Task 1: Update state.go (1 hour)
- [ ] Add StateWaitingApproval constant
- [ ] Implement String() method for TurnState
- [ ] Implement CanTransition() function
- [ ] Add state transition validation logic
- [ ] Write state_test.go with transition tests

#### Task 2: Complete turn.go (3 hours)
- [ ] Expand Turn struct with all fields
- [ ] Add ToolCall and ToolResult types
- [ ] Add TokenUsage type
- [ ] Implement NewTurn() constructor
- [ ] Add mutex for thread safety
- [ ] Implement all lifecycle methods
- [ ] Add tool tracking methods
- [ ] Add token tracking methods
- [ ] Write comprehensive tests

#### Task 3: Implement result.go (2 hours)
- [ ] Define Result struct
- [ ] Implement NewResult()
- [ ] Add helper methods
- [ ] Write result tests

#### Task 4: Testing (3 hours)
- [ ] Write unit tests for all methods
- [ ] Write state transition tests (all paths)
- [ ] Write concurrent access tests
- [ ] Write serialization tests
- [ ] Achieve >90% coverage
- [ ] Run race detector tests

#### Task 5: Documentation & Polish (1 hour)
- [ ] Add godoc comments
- [ ] Create usage examples
- [ ] Run linters
- [ ] Analyze with uast/herr
- [ ] Fix any issues

---

## Testing Strategy

### Unit Tests

#### State Transition Tests
```go
func TestTurnState_Transitions(t *testing.T) {
    tests := []struct {
        name    string
        from    TurnState
        to      TurnState
        wantErr bool
    }{
        {"Pending to Running", StatePending, StateRunning, false},
        {"Pending to Completed", StatePending, StateCompleted, true},
        {"Running to WaitingApproval", StateRunning, StateWaitingApproval, false},
        {"WaitingApproval to Running", StateWaitingApproval, StateRunning, false},
        {"Completed to Running", StateCompleted, StateRunning, true},
        // ... more cases
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            turn := &Turn{State: tt.from}
            err := turn.Transition(tt.to)
            if (err != nil) != tt.wantErr {
                t.Errorf("Transition() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

#### Lifecycle Tests
```go
func TestTurn_Lifecycle(t *testing.T) {
    turn := NewTurn("session-123", "test input")
    
    // Should start in Pending
    assert.Equal(t, StatePending, turn.State)
    
    // Start turn
    err := turn.Start()
    assert.NoError(t, err)
    assert.Equal(t, StateRunning, turn.State)
    assert.False(t, turn.StartedAt.IsZero())
    
    // Complete turn
    err = turn.Complete("response", TokenUsage{PromptTokens: 10, CompletionTokens: 20})
    assert.NoError(t, err)
    assert.Equal(t, StateCompleted, turn.State)
    assert.False(t, turn.CompletedAt.IsZero())
}
```

#### Concurrent Access Tests
```go
func TestTurn_ConcurrentAccess(t *testing.T) {
    turn := NewTurn("session-123", "test")
    turn.Start()
    
    var wg sync.WaitGroup
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            turn.AddToolCall(ToolCall{ID: uuid.New().String()})
        }()
    }
    wg.Wait()
    
    assert.Equal(t, 100, len(turn.ToolCalls))
}
```

### Integration Tests

Test turn usage within session context:

```go
func TestSession_TurnIntegration(t *testing.T) {
    session := session.NewSession("/tmp/test")
    
    turn := NewTurn(session.ID, "create a file")
    turn.Start()
    turn.Complete("File created", TokenUsage{TotalTokens: 50})
    
    session.AddTurn(turn)
    assert.Equal(t, 1, len(session.Turns))
    assert.Equal(t, StateCompleted, session.Turns[0].State)
}
```

---

## Error Handling

### Error Types

```go
var (
    ErrInvalidTransition = errors.New("invalid state transition")
    ErrTurnNotStarted    = errors.New("turn not started")
    ErrTurnAlreadyDone   = errors.New("turn already completed/failed/cancelled")
)
```

### Error Patterns

```go
func (t *Turn) Complete(response string, tokens TokenUsage) error {
    if t.State != StateRunning {
        return fmt.Errorf("%w: cannot complete from %s", ErrInvalidTransition, t.State)
    }
    
    t.State = StateCompleted
    t.AIResponse = response
    t.Tokens = tokens
    t.CompletedAt = time.Now()
    return nil
}
```

---

## Dependencies

### Internal Dependencies
- `internal/core/error.go` - Error types
- `internal/core/session` - Parent session

### External Dependencies
- `github.com/google/uuid` - UUID generation (already in go.mod)
- Standard library: `time`, `sync`, `encoding/json`

---

## Migration Notes

### Compatibility

This feature extends the minimal Turn implementation from Feature 1.1. Changes:

**Added fields:**
- ToolCalls []ToolCall
- ToolResults []ToolResult
- Error error
- Tokens TokenUsage
- Metadata map[string]interface{}

**Added state:**
- StateWaitingApproval

**Backwards compatibility:**
- Existing sessions with minimal turns will deserialize correctly
- New fields will have zero values
- No breaking changes to existing API

---

## Examples

### Basic Turn Lifecycle

```go
// Create turn
turn := turn.NewTurn("session-123", "List files in current directory")

// Start execution
if err := turn.Start(); err != nil {
    return err
}

// Add tool calls
turn.AddToolCall(turn.ToolCall{
    ID:   "call-1",
    Name: "shell",
    Args: map[string]interface{}{"command": "ls -la"},
})

// Add tool result
turn.AddToolResult(turn.ToolResult{
    ToolCallID: "call-1",
    Result:     "file1.txt\nfile2.txt",
    Duration:   10 * time.Millisecond,
})

// Complete turn
tokens := turn.TokenUsage{
    PromptTokens:     50,
    CompletionTokens: 30,
    TotalTokens:      80,
}
if err := turn.Complete("I listed the files.", tokens); err != nil {
    return err
}
```

### Approval Workflow

```go
turn := turn.NewTurn("session-123", "Delete all log files")
turn.Start()

// AI wants to run dangerous command
if needsApproval(command) {
    if err := turn.RequestApproval(); err != nil {
        return err
    }
    
    // Wait for user decision...
    approved := getUserApproval()
    
    if approved {
        turn.Approve()
        // Continue execution
    } else {
        turn.Deny()
        return nil
    }
}
```

### Error Handling

```go
turn := turn.NewTurn("session-123", "Complex task")
turn.Start()

// Execution fails
if err := executeTask(); err != nil {
    turn.Fail(fmt.Errorf("task failed: %w", err))
    return nil
}
```

---

## Acceptance Tests

### Test Case 1: Complete Turn Lifecycle

**Given:** A new turn is created  
**When:** Start() → Complete() is called  
**Then:** Turn state is Completed, timestamps are set, response is stored

### Test Case 2: Approval Workflow

**Given:** A running turn  
**When:** RequestApproval() → Approve() → Complete() is called  
**Then:** Turn transitions through WaitingApproval back to Running, then Completed

### Test Case 3: Turn Failure

**Given:** A running turn  
**When:** Fail() is called with an error  
**Then:** Turn state is Failed, error is stored, CompletedAt is set

### Test Case 4: Invalid Transition

**Given:** A completed turn  
**When:** Start() is called  
**Then:** Error is returned, state remains Completed

### Test Case 5: Tool Tracking

**Given:** A running turn  
**When:** Multiple tool calls and results are added  
**Then:** All calls and results are tracked, matched by ID

---

## Performance Requirements

### Benchmarks

Create benchmarks for critical operations:

```go
func BenchmarkTurn_Create(b *testing.B) {
    for i := 0; i < b.N; i++ {
        _ = turn.NewTurn("session-123", "input")
    }
}

func BenchmarkTurn_StateTransition(b *testing.B) {
    t := turn.NewTurn("session-123", "input")
    t.Start()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        // Measure transition time
    }
}
```

### Performance Targets
- Turn creation: <1ms (p99)
- State transition: <100μs (p99)
- Tool call addition: <50μs (p99)

---

## Security Considerations

### Input Validation
- Validate session ID is non-empty
- Validate user input length (prevent DoS)
- Sanitize metadata values

### Data Exposure
- Do not log sensitive tool arguments
- Sanitize error messages
- Filter credentials from tool results

---

## Documentation Requirements

### Godoc Comments

```go
// Turn represents a single user-AI interaction cycle within a conversation.
//
// A Turn tracks the complete lifecycle of processing user input, including
// LLM interactions, tool executions, and state transitions. Turns support
// approval workflows for dangerous operations and comprehensive execution
// tracking.
//
// State Machine:
//   Pending → Running → WaitingApproval → Running → Completed
//                    → Completed
//                    → Failed
//                    → Cancelled
//
// Thread Safety:
//   Turn methods are thread-safe and can be called concurrently.
//
// Example:
//   turn := turn.NewTurn(sessionID, userInput)
//   turn.Start()
//   // ... execute ...
//   turn.Complete(response, tokens)
type Turn struct { ... }
```

### Package Documentation

Add comprehensive package-level documentation to `doc.go`:

```go
// Package turn provides turn state management for conversations.
//
// A turn represents a single user-AI interaction cycle, tracking
// execution state, tool calls, token usage, and results.
package turn
```

---

## Success Criteria

- [ ] All DoD items checked off
- [ ] Test coverage >90%
- [ ] All state transitions tested
- [ ] Race detector clean
- [ ] Linters passing
- [ ] Code complexity <15 (verified with uast/herr)
- [ ] Documentation complete
- [ ] Can be used by Feature 1.3 (History Management)
- [ ] Can be used by Feature 7.1 (Conversation Implementation)

---

## References

- [Core Module Spec](../core-module/spec.md)
- [ROADMAP](../core-module/ROADMAP.md)
- [Feature 1.1 - Session Management](./FRD-1.1.md)
- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)

---

**Created:** 2025-10-03  
**Author:** Development Team  
**Status:** Ready for Implementation
