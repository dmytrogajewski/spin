# FRD-CORE-0.1: Approval Response Mechanism

**Feature ID:** CORE-0.1
**Priority:** Critical (BLOCKING Phase 3)
**Status:** Pending
**Created:** 2025-10-05

---

## Overview

Implement a public approval response mechanism in `internal/core` that allows UI modules (TUI, exec, IDE extensions) to intercept command approval requests, present them to users, and send back approval/denial responses. Currently, the approval system is only usable in tests via a private field.

## Problem Statement

### Current State

The `Agent` struct in `internal/core/agent.go` has a private `approvalHandler` field (line 49):

```go
type Agent struct {
    // ... other fields ...
    approvalHandler func(*Command, string) bool // Approval handler for testing
}
```

When a dangerous command needs approval, `requestApproval()` is called (lines 827-846):

```go
func (a *Agent) requestApproval(ctx context.Context, cmd *Command, reason string) bool {
    // Emit approval request event
    a.emitter.Emit(Event{
        Type:      EventCommandApproval,
        Timestamp: time.Now(),
        Data: map[string]interface{}{
            "command": cmd.Raw,
            "reason":  reason,
        },
    })

    // Use approval handler if set (for testing)
    if a.approvalHandler != nil {
        return a.approvalHandler(cmd, reason)
    }

    // Default to deny if no handler
    return false
}
```

**Issues:**
1. ❌ `approvalHandler` is **private** - only accessible in tests
2. ❌ Emits `EventCommandApproval` but has **no way to receive response**
3. ❌ Always **denies** in production (returns `false`)
4. ❌ UI modules **cannot intercept** approval requests
5. ❌ No **request/response correlation** (no request ID)
6. ❌ No **result events** (`EventCommandApproved`, `EventCommandDenied`)

### What's Needed

UI modules need to:
1. Receive approval requests via events
2. Display approval dialog to user
3. Send approval/denial response back to core
4. Receive confirmation that response was processed

## Requirements

### Functional Requirements

1. **Public Approval Handler API**
   - `WithApprovalHandler` option for Agent creation
   - Handler callback signature: `func(ApprovalRequest) ApprovalResponse`
   - Handler is optional (defaults to auto-deny)

2. **Request/Response Types**
   - `ApprovalRequest` with:
     - Request ID (for correlation)
     - Command details (raw, parsed, args)
     - Safety classification reason
     - Working directory
     - Timestamp
   - `ApprovalResponse` with:
     - Request ID (must match request)
     - Approved (boolean)
     - Reason (user-provided or auto-generated)
     - Modified command (optional, for "approve with modifications")
     - Timestamp

3. **Event Flow**
   - **Request**: Emit `EventCommandApproval` with `ApprovalRequest`
   - **Handler**: Invoke callback (blocking or with timeout)
   - **Response**: Handler returns `ApprovalResponse`
   - **Result**: Emit `EventCommandApproved` or `EventCommandDenied`

4. **Timeout Handling**
   - Default approval timeout: 60 seconds
   - Configurable via `Config.ApprovalTimeout`
   - Timeout results in denial with reason "approval timeout"

5. **Command Modification**
   - UI can modify command before approval
   - Modified command must be re-validated
   - If re-validation fails, denial is automatic

### Non-Functional Requirements

1. **Thread Safety**: Handler can be called from any goroutine
2. **Context Cancellation**: Respect context cancellation during approval wait
3. **No Deadlocks**: Approval flow must not block indefinitely
4. **Backward Compatibility**: Existing code using events continues to work
5. **Test Coverage**: ≥95% coverage for approval flow

## Design

### Types

```go
// ApprovalRequest represents a command approval request.
type ApprovalRequest struct {
    ID         string    // Unique request ID (UUID)
    Command    *Command  // Command requiring approval
    Reason     string    // Why approval is needed (from Validator)
    WorkDir    string    // Working directory
    Timestamp  time.Time // When request was created
}

// ApprovalResponse represents the user's approval decision.
type ApprovalResponse struct {
    RequestID       string    // Must match ApprovalRequest.ID
    Approved        bool      // True = approve, False = deny
    Reason          string    // User-provided reason (optional)
    ModifiedCommand string    // Modified command (optional, empty = no modification)
    Timestamp       time.Time // When response was created
}

// ApprovalHandler is a callback for handling approval requests.
// It receives an ApprovalRequest and must return an ApprovalResponse.
// The handler should block until the user makes a decision or timeout occurs.
type ApprovalHandler func(ApprovalRequest) ApprovalResponse
```

### Configuration

```go
// Add to Config struct
type Config struct {
    // ... existing fields ...

    // ApprovalTimeout is the maximum time to wait for approval response.
    // Default: 60 seconds. 0 = no timeout.
    ApprovalTimeout time.Duration
}
```

### Agent Option

```go
// WithApprovalHandler sets the approval handler for the agent.
// The handler is called when a command requires user approval.
// If no handler is set, commands requiring approval are automatically denied.
func WithApprovalHandler(handler ApprovalHandler) AgentOption {
    return func(a *Agent) error {
        a.approvalHandler = handler
        return nil
    }
}
```

### Updated requestApproval

```go
// requestApproval requests user approval for a command.
// It emits an EventCommandApproval event, invokes the approval handler (if set),
// and emits the result (EventCommandApproved or EventCommandDenied).
func (a *Agent) requestApproval(ctx context.Context, cmd *Command, reason string) bool {
    // Generate request ID
    reqID := uuid.New().String()

    // Create request
    req := ApprovalRequest{
        ID:        reqID,
        Command:   cmd,
        Reason:    reason,
        WorkDir:   a.config.WorkDir,
        Timestamp: time.Now(),
    }

    // Emit request event
    a.emitter.Emit(Event{
        Type:      EventCommandApproval,
        Timestamp: req.Timestamp,
        Data:      req,
    })

    // If no handler, auto-deny
    if a.approvalHandler == nil {
        a.emitter.Emit(Event{
            Type:      EventCommandDenied,
            Timestamp: time.Now(),
            Data: map[string]interface{}{
                "request_id": reqID,
                "reason":     "no approval handler configured",
            },
        })
        return false
    }

    // Create timeout context
    timeout := a.config.ApprovalTimeout
    if timeout == 0 {
        timeout = 60 * time.Second // default
    }
    approvalCtx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()

    // Invoke handler in goroutine (with timeout)
    respChan := make(chan ApprovalResponse, 1)
    go func() {
        resp := a.approvalHandler(req)
        respChan <- resp
    }()

    // Wait for response or timeout
    var resp ApprovalResponse
    select {
    case resp = <-respChan:
        // Got response
    case <-approvalCtx.Done():
        // Timeout or cancellation
        a.emitter.Emit(Event{
            Type:      EventCommandDenied,
            Timestamp: time.Now(),
            Data: map[string]interface{}{
                "request_id": reqID,
                "reason":     "approval timeout or context cancelled",
            },
        })
        return false
    }

    // Validate response
    if resp.RequestID != reqID {
        a.emitter.Emit(Event{
            Type:      EventCommandDenied,
            Timestamp: time.Now(),
            Data: map[string]interface{}{
                "request_id": reqID,
                "reason":     "response request ID mismatch",
            },
        })
        return false
    }

    // Handle command modification
    if resp.Approved && resp.ModifiedCommand != "" {
        // Re-parse modified command
        modCmd, err := ParseCommand(resp.ModifiedCommand)
        if err != nil {
            a.emitter.Emit(Event{
                Type:      EventCommandDenied,
                Timestamp: time.Now(),
                Data: map[string]interface{}{
                    "request_id": reqID,
                    "reason":     "modified command parse error: " + err.Error(),
                },
            })
            return false
        }

        // Re-validate modified command
        safety, modReason := a.validator.Validate(modCmd)
        if safety != SafetySafe {
            a.emitter.Emit(Event{
                Type:      EventCommandDenied,
                Timestamp: time.Now(),
                Data: map[string]interface{}{
                    "request_id": reqID,
                    "reason":     "modified command failed validation: " + modReason,
                },
            })
            return false
        }

        // Update command
        *cmd = *modCmd
    }

    // Emit result event
    if resp.Approved {
        a.emitter.Emit(Event{
            Type:      EventCommandApproved,
            Timestamp: time.Now(),
            Data: map[string]interface{}{
                "request_id": reqID,
                "command":    cmd.Raw,
                "reason":     resp.Reason,
            },
        })
        return true
    } else {
        a.emitter.Emit(Event{
            Type:      EventCommandDenied,
            Timestamp: time.Now(),
            Data: map[string]interface{}{
                "request_id": reqID,
                "reason":     resp.Reason,
            },
        })
        return false
    }
}
```

### New Event Types

```go
const (
    // ... existing event types ...

    // EventCommandApproval is emitted when a command requires approval
    EventCommandApproval EventType = "command_approval"

    // EventCommandApproved is emitted when a command is approved
    EventCommandApproved EventType = "command_approved"

    // EventCommandDenied is emitted when a command is denied
    EventCommandDenied EventType = "command_denied"
)
```

## Usage Examples

### TUI Usage

```go
// Create approval handler for TUI
handler := func(req core.ApprovalRequest) core.ApprovalResponse {
    // Display modal dialog to user
    approved, modifiedCmd := showApprovalDialog(req)

    return core.ApprovalResponse{
        RequestID:       req.ID,
        Approved:        approved,
        Reason:          getUserReason(),
        ModifiedCommand: modifiedCmd,
        Timestamp:       time.Now(),
    }
}

// Create agent with handler
agent := core.NewAgent(cfg, core.WithApprovalHandler(handler))
```

### Exec Mode Usage

```go
// Auto-deny all approvals in exec mode (unless --auto-approve)
if !autoApprove {
    handler := func(req core.ApprovalRequest) core.ApprovalResponse {
        // Log to audit trail
        auditLogger.Info("approval request denied",
            "request_id", req.ID,
            "command", req.Command.Raw,
            "reason", "exec mode requires --auto-approve")

        return core.ApprovalResponse{
            RequestID: req.ID,
            Approved:  false,
            Reason:    "exec mode requires --auto-approve flag",
            Timestamp: time.Now(),
        }
    }
    agent = core.NewAgent(cfg, core.WithApprovalHandler(handler))
}
```

### IDE Extension Usage

```go
// Forward approval request to IDE UI
handler := func(req core.ApprovalRequest) core.ApprovalResponse {
    // Send request to IDE over JSON-RPC
    result := sendToIDE("approval_request", req)

    return core.ApprovalResponse{
        RequestID:       req.ID,
        Approved:        result.Approved,
        Reason:          result.Reason,
        ModifiedCommand: result.ModifiedCommand,
        Timestamp:       time.Now(),
    }
}

agent := core.NewAgent(cfg, core.WithApprovalHandler(handler))
```

## Implementation Plan

### Step 1: Define Types (agent.go)
- [x] Add `ApprovalRequest` struct
- [x] Add `ApprovalResponse` struct
- [x] Add `ApprovalHandler` type
- [x] Add `ApprovalTimeout` to `Config`

### Step 2: Update Event Types (event.go)
- [x] Add `EventCommandApproved` constant
- [x] Add `EventCommandDenied` constant
- [x] Update event type tests

### Step 3: Add Agent Option
- [x] Implement `WithApprovalHandler` function
- [x] Update `Agent` struct to use `ApprovalHandler` type (change from private func)

### Step 4: Update requestApproval
- [x] Generate request ID (UUID)
- [x] Create `ApprovalRequest`
- [x] Emit `EventCommandApproval` with request
- [x] Invoke handler with timeout
- [x] Handle command modification
- [x] Emit result events (`EventCommandApproved` or `EventCommandDenied`)

### Step 5: Write Tests
- [ ] Test approval flow (approved)
- [ ] Test approval flow (denied)
- [ ] Test approval with command modification
- [ ] Test approval timeout
- [ ] Test context cancellation
- [ ] Test no handler (auto-deny)
- [ ] Test request ID mismatch
- [ ] Test modified command validation failure
- [ ] Coverage ≥95%

### Step 6: Update Documentation
- [ ] Add godoc to all new types and functions
- [ ] Update `docs/packages/core.md` with approval examples
- [ ] Update roadmap Phase 0.1 status

## Testing Strategy

### Unit Tests (agent_test.go)

```go
func TestApprovalFlow_Approved(t *testing.T) {
    handler := func(req ApprovalRequest) ApprovalResponse {
        return ApprovalResponse{
            RequestID: req.ID,
            Approved:  true,
            Reason:    "user approved",
            Timestamp: time.Now(),
        }
    }

    agent := NewAgent(cfg, WithApprovalHandler(handler))
    approved := agent.requestApproval(ctx, cmd, "test reason")
    assert.True(t, approved)
}

func TestApprovalFlow_Denied(t *testing.T) {
    handler := func(req ApprovalRequest) ApprovalResponse {
        return ApprovalResponse{
            RequestID: req.ID,
            Approved:  false,
            Reason:    "user denied",
            Timestamp: time.Now(),
        }
    }

    agent := NewAgent(cfg, WithApprovalHandler(handler))
    approved := agent.requestApproval(ctx, cmd, "test reason")
    assert.False(t, approved)
}

func TestApprovalFlow_Timeout(t *testing.T) {
    handler := func(req ApprovalRequest) ApprovalResponse {
        time.Sleep(2 * time.Second) // Simulate slow response
        return ApprovalResponse{RequestID: req.ID, Approved: true}
    }

    cfg.ApprovalTimeout = 100 * time.Millisecond
    agent := NewAgent(cfg, WithApprovalHandler(handler))
    approved := agent.requestApproval(ctx, cmd, "test reason")
    assert.False(t, approved) // Should timeout and deny
}
```

### Integration Tests

Test with real UI modules:
1. TUI approval dialog flow
2. Exec mode auto-denial
3. IDE extension approval forwarding

## Success Criteria

### DoD (Definition of Done)

- [x] Types defined: `ApprovalRequest`, `ApprovalResponse`, `ApprovalHandler`
- [x] Event types added: `EventCommandApproved`, `EventCommandDenied`
- [x] Agent option: `WithApprovalHandler` implemented
- [ ] `requestApproval` updated with full flow
- [ ] Tests passing with ≥95% coverage
- [ ] Godoc complete on all exports
- [ ] Documentation updated (`docs/packages/core.md`)
- [ ] No lint errors (`make lint` passes)
- [ ] Complexity ≤15 for all functions
- [ ] Roadmap Phase 0.1 marked complete

### Quality Gates

- [ ] All tests pass with `-race` flag
- [ ] Coverage: ≥95% for approval code paths
- [ ] `make lint` passes (zero errors)
- [ ] Complexity ≤15 (verified with `gocyclo`)
- [ ] No deadlocks in approval flow
- [ ] Timeout handling works correctly

## Risks and Mitigations

### Risk 1: Deadlocks in Handler
**Mitigation**: Use timeout context and goroutine for handler invocation

### Risk 2: Handler Panics
**Mitigation**: Recover from panics in goroutine, treat as denial

### Risk 3: Request ID Collision
**Mitigation**: Use UUID v4 (collision probability negligible)

### Risk 4: Backward Compatibility
**Mitigation**: Make handler optional, default behavior is auto-deny

## Dependencies

- **Go 1.24+**: For `context`, `time`, `sync`
- **google/uuid**: For request ID generation (add to `go.mod`)
- **internal/core**: Existing validator, executor, event system

## References

- [AGENTS.md](../../AGENTS.md) - Implementation workflow
- [architecture-overview.md](../architecture-overview.md) - Core architecture
- [ROADMAP.md](../ui-modules/ROADMAP.md) - Phase 0.1 requirements
- [core.md](../../docs/packages/core.md) - Core package documentation

---

**Created by:** Rob Pike (AI Agent)
**Reviewed by:** Pending
**Approved by:** Pending
