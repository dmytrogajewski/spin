# FRD-UI-3.5: Tool Approval UI

**Feature ID:** UI-3.5
**Priority:** High (Phase 3 - TUI)
**Status:** In Progress
**Created:** 2025-10-05
**Roadmap:** [Phase 3.5](../ui-modules/ROADMAP.md#35-tool-approval-ui)

---

## Overview

Implement a modal overlay UI component in the TUI that displays command approval requests from the core module and allows users to approve, deny, or modify dangerous commands before execution. This integrates with the approval response mechanism implemented in Phase 0.1 (FRD-CORE-0.1).

## Problem Statement

### Current State

- ✅ Core module has approval request/response mechanism (Phase 0.1)
- ✅ Core emits `EventCommandApproval` when dangerous command detected
- ✅ Core has `ApprovalHandler` callback pattern
- ❌ TUI has no UI component to display approval requests
- ❌ TUI cannot respond to approval requests
- ❌ Users cannot see what commands require approval

### What's Needed

The TUI needs to:
1. Subscribe to `EventCommandApproval` events from core
2. Display a modal overlay showing command details
3. Provide keyboard shortcuts for [A]pprove, [D]eny, [M]odify
4. Allow users to edit commands before approval
5. Send `ApprovalResponse` back to core
6. Transition back to waiting state

## Requirements

### Functional Requirements

1. **Approval Modal Component**
   - Modal overlay appears on top of chat interface
   - Shows command details (raw command, args, working directory)
   - Shows safety classification reason
   - Provides clear action buttons: [A]pprove, [D]eny, [M]odify
   - Dismissible only via action selection (no escape/close)

2. **Keyboard Shortcuts**
   - `a` or `A` - Approve command as-is
   - `d` or `D` - Deny command
   - `m` or `M` - Modify command (enter edit mode)
   - `Ctrl+C` - Cancel turn and deny (global shortcut)

3. **Command Modification Flow**
   - Pressing `M` opens text input with current command
   - User can edit command text
   - `Enter` to confirm modification and request approval
   - `Esc` to cancel modification and return to approval dialog
   - Modified command is re-validated by core

4. **State Management**
   - `StateToolApproval` during approval dialog
   - Store `pendingApproval` with request details
   - Transition to `StateWaitingResponse` after approval/denial
   - Clear `pendingApproval` when state changes

5. **Integration with Core**
   - Use `core.WithApprovalHandler` when creating agent
   - Handler blocks and waits for user input via channel
   - Handler returns `core.ApprovalResponse` with decision

### Non-Functional Requirements

1. **Responsiveness**: Modal renders immediately (<50ms)
2. **Clarity**: Command details clearly visible
3. **Safety**: Default action is deny (not approve)
4. **Accessibility**: Keyboard-only navigation
5. **Thread Safety**: Approval channel properly synchronized
6. **Test Coverage**: ≥90% coverage

## Design

### Types

```go
// ApprovalMsg is sent from core when approval is needed
type ApprovalMsg struct {
    Request core.ApprovalRequest
}

// ApprovalDecisionMsg is sent when user makes a decision
type ApprovalDecisionMsg struct {
    Response core.ApprovalResponse
}

// ApprovalModalModel represents the approval UI component
type ApprovalModalModel struct {
    request       core.ApprovalRequest
    editing       bool
    modifiedCmd   string
    width         int
    height        int
}
```

### Model Updates

```go
// Add to internal/tui/app.go Model struct:
type Model struct {
    // ... existing fields ...

    // Approval state
    approvalModal   *ApprovalModalModel
    approvalChan    chan core.ApprovalResponse
}
```

### Component Structure

**Files to create:**
- `internal/tui/ui/approval.go` - Approval modal component
- `internal/tui/ui/approval_test.go` - Tests

**Approval Modal Component:**
```go
package ui

import (
    "github.com/charmbracelet/lipgloss"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/dmytrogajewski/spin/internal/core"
)

// ApprovalModal is a modal overlay for command approval.
type ApprovalModal struct {
    request     core.ApprovalRequest
    editing     bool
    editValue   string
    width       int
    height      int
}

// NewApprovalModal creates a new approval modal.
func NewApprovalModal(req core.ApprovalRequest, width, height int) ApprovalModal

// Update handles Bubble Tea messages.
func (m ApprovalModal) Update(msg tea.Msg) (ApprovalModal, tea.Cmd)

// View renders the approval modal.
func (m ApprovalModal) View() string
```

### State Machine Update

**Existing states:**
- `StateIdle` → `StateWaitingResponse` (send message)
- `StateWaitingResponse` → `StateToolApproval` (approval needed)
- `StateToolApproval` → `StateWaitingResponse` (after approval/denial)

**State transitions already defined in state.go** ✅

### Approval Handler Implementation

```go
// In cmd/spin/tui.go or internal/tui/integration.go

func createApprovalHandler(m *Model) core.ApprovalHandler {
    return func(req core.ApprovalRequest) core.ApprovalResponse {
        // Send request to TUI via Bubble Tea message
        m.program.Send(ApprovalMsg{Request: req})

        // Block waiting for user decision
        resp := <-m.approvalChan

        return resp
    }
}
```

### Update Flow

```go
// In internal/tui/app.go Update():

case ApprovalMsg:
    // Core needs approval
    m.state = StateToolApproval
    m.approvalModal = NewApprovalModal(msg.Request, m.width, m.height)
    return m, nil

case tea.KeyMsg:
    if m.state == StateToolApproval {
        switch msg.String() {
        case "a", "A":
            // Approve
            resp := core.ApprovalResponse{
                RequestID: m.approvalModal.request.ID,
                Approved:  true,
                Reason:    "user approved via TUI",
                Timestamp: time.Now(),
            }
            m.approvalChan <- resp
            m.state = StateWaitingResponse
            m.approvalModal = nil
            return m, nil

        case "d", "D":
            // Deny
            resp := core.ApprovalResponse{
                RequestID: m.approvalModal.request.ID,
                Approved:  false,
                Reason:    "user denied via TUI",
                Timestamp: time.Now(),
            }
            m.approvalChan <- resp
            m.state = StateWaitingResponse
            m.approvalModal = nil
            return m, nil

        case "m", "M":
            // Modify
            m.approvalModal.editing = true
            m.approvalModal.editValue = m.approvalModal.request.Command.Raw
            return m, nil
        }
    }
```

### View Rendering

```go
// In internal/tui/view.go

func (m Model) View() string {
    if m.quitting {
        return ""
    }

    // Base view (chat + input + status bar)
    base := lipgloss.JoinVertical(
        lipgloss.Top,
        m.chat.View(),
        m.input.View(),
        m.renderStatusBar(),
    )

    // Overlay approval modal if active
    if m.state == StateToolApproval && m.approvalModal != nil {
        return lipgloss.Place(
            m.width, m.height,
            lipgloss.Center, lipgloss.Middle,
            m.approvalModal.View(),
            lipgloss.WithWhitespaceChars(" "),
            lipgloss.WithWhitespaceForeground(lipgloss.Color("240")),
        )
    }

    return base
}
```

### Modal UI Design

```
┌─────────────────────────────────────────────────┐
│              🔒 Command Approval                │
├─────────────────────────────────────────────────┤
│                                                 │
│  Command:  rm -rf /tmp/cache                    │
│  Directory: /home/user/project                  │
│                                                 │
│  Reason: Destructive operation detected         │
│                                                 │
│  This command will permanently delete files.    │
│                                                 │
├─────────────────────────────────────────────────┤
│  [A]pprove   [D]eny   [M]odify                  │
└─────────────────────────────────────────────────┘
```

**When editing:**
```
┌─────────────────────────────────────────────────┐
│            Modify Command                       │
├─────────────────────────────────────────────────┤
│                                                 │
│  > rm -rf /tmp/cache/_                          │
│                                                 │
│  Press Enter to approve, Esc to cancel          │
│                                                 │
└─────────────────────────────────────────────────┘
```

## Implementation Plan

### Step 1: Create Approval Modal Component
- [ ] Create `internal/tui/ui/approval.go`
- [ ] Implement `ApprovalModal` struct
- [ ] Implement `NewApprovalModal()` constructor
- [ ] Implement `Update()` message handler
- [ ] Implement `View()` rendering
- [ ] Handle edit mode (text input for command modification)

### Step 2: Integrate with TUI Model
- [ ] Add `approvalModal` field to `Model` in `app.go`
- [ ] Add `approvalChan` field to `Model`
- [ ] Handle `ApprovalMsg` in `Update()`
- [ ] Handle approval keyboard shortcuts (A/D/M)
- [ ] Update `View()` to overlay modal when active

### Step 3: Connect to Core
- [ ] Create approval handler function
- [ ] Pass handler to core via `WithApprovalHandler`
- [ ] Handle blocking/waiting for user decision
- [ ] Send response through channel

### Step 4: Write Tests
- [ ] Test modal creation
- [ ] Test approval (A key)
- [ ] Test denial (D key)
- [ ] Test modification flow (M key)
- [ ] Test state transitions
- [ ] Test rendering
- [ ] Coverage ≥90%

### Step 5: Styling
- [ ] Use lipgloss for modal styling
- [ ] Add border and padding
- [ ] Color-code actions (green=approve, red=deny, yellow=modify)
- [ ] Ensure contrast and readability

### Step 6: Integration Testing
- [ ] Test with real core approval requests
- [ ] Test command modification and re-validation
- [ ] Test timeout scenarios
- [ ] Test cancel during approval

## Testing Strategy

### Unit Tests

```go
// internal/tui/ui/approval_test.go

func TestApprovalModal_Create(t *testing.T) {
    req := core.ApprovalRequest{
        ID: "test-123",
        Command: &core.Command{Raw: "rm -rf /tmp"},
        Reason: "destructive",
        WorkDir: "/home/user",
        Timestamp: time.Now(),
    }

    modal := NewApprovalModal(req, 80, 24)
    assert.Equal(t, req.ID, modal.request.ID)
    assert.False(t, modal.editing)
}

func TestApprovalModal_Approve(t *testing.T) {
    modal := NewApprovalModal(testReq, 80, 24)

    modal, cmd := modal.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

    // Should emit ApprovalDecisionMsg
    assert.NotNil(t, cmd)
}

func TestApprovalModal_Modify(t *testing.T) {
    modal := NewApprovalModal(testReq, 80, 24)

    modal, _ := modal.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})

    assert.True(t, modal.editing)
    assert.Equal(t, testReq.Command.Raw, modal.editValue)
}
```

### Integration Tests

```go
// internal/tui/app_test.go

func TestTUI_ApprovalFlow(t *testing.T) {
    m := NewModel()

    // Simulate approval request from core
    approvalReq := core.ApprovalRequest{
        ID: uuid.New().String(),
        Command: &core.Command{Raw: "rm -rf /tmp"},
        Reason: "destructive",
        WorkDir: "/home/user",
        Timestamp: time.Now(),
    }

    m, _ = m.Update(ApprovalMsg{Request: approvalReq})
    assert.Equal(t, StateToolApproval, m.state)
    assert.NotNil(t, m.approvalModal)

    // Simulate user pressing 'a' to approve
    m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
    assert.Equal(t, StateWaitingResponse, m.state)
    assert.Nil(t, m.approvalModal)

    // Check response sent through channel
    select {
    case resp := <-m.approvalChan:
        assert.Equal(t, approvalReq.ID, resp.RequestID)
        assert.True(t, resp.Approved)
    case <-time.After(100 * time.Millisecond):
        t.Fatal("no response received")
    }
}
```

## Success Criteria

### Definition of Done (DoD)

- [ ] `ApprovalModal` component created in `internal/tui/ui/approval.go`
- [ ] Modal displays command, reason, and actions
- [ ] Keyboard shortcuts work (A/D/M)
- [ ] Command modification flow complete
- [ ] Integration with core approval handler
- [ ] State transitions correct (Idle ↔ ToolApproval ↔ WaitingResponse)
- [ ] Tests passing with ≥90% coverage
- [ ] Race detector clean (`go test -race`)
- [ ] Linter passing (`make lint`)
- [ ] Complexity ≤15 for all functions
- [ ] Godoc on all exports
- [ ] Roadmap Phase 3.5 marked complete

### Quality Gates

- [ ] All tests pass with `-race` flag
- [ ] Coverage: ≥90% for approval modal code
- [ ] `make lint` passes (zero errors)
- [ ] Complexity ≤15 (verified with `gocyclo`)
- [ ] Modal renders correctly (visual test)
- [ ] No deadlocks in approval flow
- [ ] Approval handler doesn't block UI

## Risks and Mitigations

### Risk 1: Approval Handler Blocks UI
**Mitigation**: Handler runs in separate goroutine, communicates via channel

### Risk 2: Modal Rendering Performance
**Mitigation**: Use lipgloss efficiently, pre-calculate layout, test render time <16ms

### Risk 3: State Synchronization Issues
**Mitigation**: Single-threaded Bubble Tea model, channel for cross-goroutine communication

### Risk 4: Command Modification Bugs
**Mitigation**: Core re-validates modified commands, comprehensive tests for edge cases

## Dependencies

- **Go 1.24+**: For context, channels, time
- **Bubble Tea**: For TUI framework and message passing
- **Lipgloss**: For modal styling
- **internal/core**: For `ApprovalRequest`, `ApprovalResponse`, `ApprovalHandler`
- **internal/tui/ui**: For UI component patterns

## Related Documents

- [FRD-CORE-0.1](FRD-CORE-0.1.md) - Approval response mechanism
- [ROADMAP.md](../ui-modules/ROADMAP.md) - Phase 3.5 roadmap item
- [core.md](../../docs/packages/core.md) - Core package documentation
- [spec.md](../ui-modules/spec.md) - UI modules technical spec

---

**Created by:** AI Agent (Claude)
**Reviewed by:** Pending
**Approved by:** Pending
