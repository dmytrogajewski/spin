# FRD-CORE-0.2: Pause/Resume Turn Execution

**Feature ID:** CORE-0.2
**Priority:** Critical (BLOCKING Phase 3 - TUI)
**Status:** Pending
**Created:** 2025-10-05

---

## Overview

Implement pause/resume functionality for conversation turn execution, allowing UI modules (especially TUI) to pause a running turn for user review or interaction, then resume execution without cancelling the entire conversation.

## Problem Statement

### Current State

The `Conversation` type in `internal/core/conversation.go` only supports:
- **Running** turns via `RunTurn()`
- **Stopping** the entire conversation via `Stop()` (which cancels active turn and prevents further execution)

**What's Missing:**
- ❌ No way to **pause** a running turn temporarily
- ❌ No way to **resume** a paused turn
- ❌ `RunTurn()` doesn't check for control signals during execution
- ❌ State machine doesn't properly support `StatePaused` transitions

### Use Cases

1. **Approval Dialogs (TUI):**
   - Turn is running, Agent requests approval
   - UI calls `Pause()` to freeze execution
   - User reviews approval request in modal dialog
   - User approves → UI calls `Resume()` → execution continues
   - User denies → UI calls `Stop()` → turn cancelled

2. **User Interrupt (TUI):**
   - Turn is running, generating long response
   - User presses Ctrl+P to pause and review partial output
   - User decides to let it continue → `Resume()`
   - Or user cancels → `Stop()`

3. **Rate Limiting (Background):**
   - Turn hits API rate limit
   - Automatically `Pause()` for backoff period
   - Auto-`Resume()` after delay

## Requirements

### Functional Requirements

1. **Pause API**
   - `Pause()` method on `Conversation`
   - Can only pause when `State == StateRunning`
   - Transitions to `State == StatePaused`
   - Does NOT cancel the turn context
   - Emits `EventTurnPaused`

2. **Resume API**
   - `Resume()` method on `Conversation`
   - Can only resume when `State == StatePaused`
   - Transitions back to `State == StateRunning`
   - Continues turn execution from where it paused
   - Emits `EventTurnResumed`

3. **Control Channel**
   - Internal `controlChan chan ControlSignal` for pause/resume/cancel signals
   - Buffered channel (size 1) to prevent blocking
   - Created when turn starts, closed when turn ends
   - Checked at key points in `RunTurn()` execution loop

4. **Execution Loop Integration**
   - `RunTurn()` checks `controlChan` periodically
   - When paused, blocks until resumed or cancelled
   - When cancelled (via `Stop()`), returns immediately
   - Agent's `Execute()` method also checks control signals

5. **State Transitions**
   ```
   StateIdle → StateRunning → StatePaused → StateRunning → StateCompleted
                           ↓                             ↓
                        StateIdle (via Stop)      StateIdle (via Stop)
   ```

### Non-Functional Requirements

1. **Thread Safety**: Pause/Resume must be safe for concurrent calls
2. **No Deadlocks**: Control channel must not block indefinitely
3. **Fast Pause**: Pause should take effect within 100ms
4. **State Consistency**: State transitions must be atomic
5. **Test Coverage**: ≥90% coverage for pause/resume paths

## Design

### Types

```go
// ControlSignal represents a control signal sent to a running turn.
type ControlSignal int

const (
	// SignalPause requests the turn to pause execution
	SignalPause ControlSignal = iota
	// SignalResume requests the turn to resume from paused state
	SignalResume
	// SignalCancel requests the turn to cancel immediately
	SignalCancel
)

// String returns the string representation of ControlSignal.
func (s ControlSignal) String() string {
	switch s {
	case SignalPause:
		return "pause"
	case SignalResume:
		return "resume"
	case SignalCancel:
		return "cancel"
	default:
		return "unknown"
	}
}
```

### Updated Conversation Struct

```go
type Conversation struct {
	// ... existing fields ...

	// Control channel for pause/resume/cancel
	controlChan chan ControlSignal
	controlMu   sync.Mutex // Protects controlChan creation/access
}
```

### Pause Method

```go
// Pause pauses the currently running turn.
// Returns an error if no turn is running or conversation is stopped.
func (c *Conversation) Pause() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Validate state
	if c.state != StateRunning {
		return fmt.Errorf("cannot pause: conversation is %s", c.state.String())
	}

	// Send pause signal (non-blocking)
	c.controlMu.Lock()
	if c.controlChan != nil {
		select {
		case c.controlChan <- SignalPause:
			// Signal sent
		default:
			// Channel full, pause already requested
		}
	}
	c.controlMu.Unlock()

	// Transition to paused state
	c.state = StatePaused

	// Emit event
	if c.emitter != nil {
		c.emitter.Emit(Event{
			Type:      EventTurnPaused,
			Timestamp: time.Now(),
			Data:      map[string]interface{}{"reason": "user requested"},
		})
	}

	return nil
}
```

### Resume Method

```go
// Resume resumes a paused turn.
// Returns an error if no turn is paused.
func (c *Conversation) Resume() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Validate state
	if c.state != StatePaused {
		return fmt.Errorf("cannot resume: conversation is %s", c.state.String())
	}

	// Send resume signal (non-blocking)
	c.controlMu.Lock()
	if c.controlChan != nil {
		select {
		case c.controlChan <- SignalResume:
			// Signal sent
		default:
			// Channel full, resume already requested
		}
	}
	c.controlMu.Unlock()

	// Transition back to running
	c.state = StateRunning

	// Emit event
	if c.emitter != nil {
		c.emitter.Emit(Event{
			Type:      EventTurnResumed,
			Timestamp: time.Now(),
			Data:      map[string]interface{}{"reason": "user requested"},
		})
	}

	return nil
}
```

### Updated RunTurn Method

```go
func (c *Conversation) RunTurn(ctx context.Context, userInput string) error {
	// ... existing validation and setup ...

	// Create control channel for this turn
	c.controlMu.Lock()
	c.controlChan = make(chan ControlSignal, 1)
	controlChan := c.controlChan
	c.controlMu.Unlock()

	// Ensure control channel cleanup
	defer func() {
		c.controlMu.Lock()
		if c.controlChan != nil {
			close(c.controlChan)
			c.controlChan = nil
		}
		c.controlMu.Unlock()
	}()

	// Execute turn with control signal checking
	return c.runTurnWithControl(turnCtx, req, controlChan)
}

func (c *Conversation) runTurnWithControl(ctx context.Context, req *AgentRequest, controlChan <-chan ControlSignal) error {
	// Wrap agent execution with control signal checking
	done := make(chan error, 1)
	go func() {
		resp, err := c.agent.Execute(ctx, req)
		if err != nil {
			done <- err
			return
		}

		// Append assistant response
		if resp != nil && resp.Content != "" && c.history != nil {
			_ = c.history.AddAssistantMessage(resp.Content)
		}

		done <- nil
	}()

	// Monitor for completion or control signals
	for {
		select {
		case err := <-done:
			// Turn completed
			return err

		case signal := <-controlChan:
			switch signal {
			case SignalPause:
				// Enter paused state, wait for resume or cancel
				if err := c.waitForResume(ctx, controlChan); err != nil {
					return err
				}

			case SignalCancel:
				// Cancel requested, propagate cancellation
				return context.Canceled

			case SignalResume:
				// Already running, ignore
				continue
			}

		case <-ctx.Done():
			// Context cancelled (e.g., by Stop())
			return ctx.Err()
		}
	}
}

func (c *Conversation) waitForResume(ctx context.Context, controlChan <-chan ControlSignal) error {
	for {
		select {
		case signal := <-controlChan:
			switch signal {
			case SignalResume:
				// Resume execution
				return nil

			case SignalCancel:
				// Cancel while paused
				return context.Canceled

			case SignalPause:
				// Already paused, ignore
				continue
			}

		case <-ctx.Done():
			// Context cancelled while paused
			return ctx.Err()
		}
	}
}
```

### New Event Types

```go
const (
	// ... existing event types ...

	// EventTurnPaused is emitted when a turn is paused
	EventTurnPaused EventType = "turn_paused"

	// EventTurnResumed is emitted when a turn resumes from paused state
	EventTurnResumed EventType = "turn_resumed"
)
```

Update `String()` method in `event.go`:

```go
func (e EventType) String() string {
	names := []string{
		// ... existing names ...
		"turn_paused",
		"turn_resumed",
	}
	// ... rest of implementation ...
}
```

## Implementation Plan

### Step 1: Add ControlSignal Type
- [x] Define `ControlSignal` type and constants
- [x] Add `String()` method

### Step 2: Update Event Types
- [x] Add `EventTurnPaused` constant
- [x] Add `EventTurnResumed` constant
- [x] Update `String()` method

### Step 3: Update Conversation Struct
- [x] Add `controlChan chan ControlSignal` field
- [x] Add `controlMu sync.Mutex` field

### Step 4: Implement Pause()
- [ ] Add `Pause()` method
- [ ] Validate state (must be Running)
- [ ] Send pause signal to control channel
- [ ] Transition state to Paused
- [ ] Emit EventTurnPaused

### Step 5: Implement Resume()
- [ ] Add `Resume()` method
- [ ] Validate state (must be Paused)
- [ ] Send resume signal to control channel
- [ ] Transition state to Running
- [ ] Emit EventTurnResumed

### Step 6: Update RunTurn()
- [ ] Create control channel at turn start
- [ ] Call `runTurnWithControl()` instead of direct agent execute
- [ ] Ensure control channel cleanup in defer

### Step 7: Implement runTurnWithControl()
- [ ] Execute agent in goroutine
- [ ] Monitor for completion, control signals, and context cancellation
- [ ] Handle pause signal by calling `waitForResume()`
- [ ] Handle cancel signal by returning immediately
- [ ] Handle context cancellation

### Step 8: Implement waitForResume()
- [ ] Block until resume or cancel signal received
- [ ] Handle context cancellation while paused

### Step 9: Write Tests
- [ ] Test Pause() when running
- [ ] Test Pause() when not running (error)
- [ ] Test Resume() when paused
- [ ] Test Resume() when not paused (error)
- [ ] Test pause/resume cycle during turn
- [ ] Test cancel while paused
- [ ] Test context cancellation while paused
- [ ] Test concurrent Pause/Resume calls
- [ ] Coverage ≥90%

### Step 10: Update Documentation
- [ ] Update `docs/packages/core.md` with pause/resume examples
- [ ] Add godoc to all new methods and types

## Testing Strategy

### Unit Tests (conversation_test.go)

```go
func TestConversation_Pause_WhenRunning(t *testing.T) {
	// Start a long-running turn
	// Call Pause() while running
	// Verify state transitions to Paused
	// Verify EventTurnPaused emitted
}

func TestConversation_Pause_WhenNotRunning(t *testing.T) {
	// Call Pause() when Idle
	// Verify error returned
	// Verify state unchanged
}

func TestConversation_Resume_WhenPaused(t *testing.T) {
	// Pause a running turn
	// Call Resume()
	// Verify state transitions back to Running
	// Verify EventTurnResumed emitted
	// Verify turn continues executing
}

func TestConversation_Resume_WhenNotPaused(t *testing.T) {
	// Call Resume() when Running
	// Verify error returned
}

func TestConversation_PauseResumeCycle(t *testing.T) {
	// Start turn
	// Pause → verify paused
	// Resume → verify running
	// Verify turn completes successfully
}

func TestConversation_CancelWhilePaused(t *testing.T) {
	// Start turn
	// Pause
	// Call Stop()
	// Verify turn cancelled
	// Verify state transitions to Cancelled
}

func TestConversation_ContextCancellationWhilePaused(t *testing.T) {
	// Start turn with cancellable context
	// Pause
	// Cancel context
	// Verify turn returns with context.Canceled error
}
```

## Success Criteria

### DoD (Definition of Done)

- [x] ControlSignal type defined
- [x] Event types added: EventTurnPaused, EventTurnResumed
- [ ] Pause() method implemented
- [ ] Resume() method implemented
- [ ] RunTurn() updated with control channel
- [ ] runTurnWithControl() implemented
- [ ] waitForResume() implemented
- [ ] Tests passing with ≥90% coverage
- [ ] Godoc complete on all exports
- [ ] Documentation updated (docs/packages/core.md)
- [ ] Roadmap Phase 0.2 marked complete

### Quality Gates

- [ ] All tests pass with `-race` flag
- [ ] Coverage: ≥90% for pause/resume code paths
- [ ] `make lint` passes (zero errors)
- [ ] Complexity ≤15 (verified with `gocyclo`)
- [ ] No deadlocks in pause/resume flow
- [ ] Fast pause response (<100ms)

## Risks and Mitigations

### Risk 1: Deadlocks in Control Channel
**Mitigation**: Use buffered channel (size 1) and non-blocking sends

### Risk 2: Race Conditions in State Transitions
**Mitigation**: Use mutex locks for all state changes

### Risk 3: Pause Not Taking Effect Quickly
**Mitigation**: Check control signals at multiple points in execution loop

### Risk 4: Agent Execute Not Respecting Pause
**Mitigation**: This FRD only handles Conversation-level pause. Agent-level pause (checking during tool execution) is out of scope for 0.2

## Dependencies

- **Go 1.24+**: For context, channels, sync primitives
- **internal/core**: Existing Conversation, State, Event types

## References

- [AGENTS.md](../../AGENTS.md) - Implementation workflow
- [architecture-overview.md](../architecture-overview.md) - Core architecture
- [ROADMAP.md](../ui-modules/ROADMAP.md) - Phase 0.2 requirements
- [conversation.go](../../internal/core/conversation.go) - Conversation implementation
- [state.go](../../internal/core/state.go) - State type and transitions

---

**Created by:** Claude (AI Agent)
**Reviewed by:** Pending
**Approved by:** Pending
