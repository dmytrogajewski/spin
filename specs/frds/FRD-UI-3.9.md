# FRD-UI-3.9: Keyboard Shortcuts & Events

## Metadata
- **FRD ID**: FRD-UI-3.9
- **Title**: Comprehensive Keyboard Shortcuts for TUI
- **Component**: UI Modules (TUI)
- **Status**: Draft
- **Created**: 2025-10-05
- **Roadmap**: Phase 3.9

## Overview

Implements comprehensive keyboard shortcut handling for the Spin TUI, including command cancellation, screen clear, graceful exit, help display, and transcript navigation. This FRD consolidates all keyboard interactions into a cohesive, well-tested system.

## Background

The TUI currently has basic keyboard handling (Enter, Esc-Esc for backtrack, file picker navigation). Phase 3.9 completes the keyboard interaction layer by adding:

- **Ctrl+C**: Cancel current AI turn (stop generation)
- **Ctrl+D**: Graceful exit
- **Ctrl+L**: Clear screen (reset transcript view)
- **Ctrl+H** or **?**: Show help overlay
- **PgUp/PgDn**: Scroll transcript (already in viewport)
- **Home/End**: Jump to top/bottom of transcript

This provides a complete, terminal-native keyboard experience.

## Requirements

### Functional Requirements

#### FR-3.9.1: Command Cancellation (Ctrl+C)
**Priority**: P0 (Critical)

When user presses Ctrl+C during an active AI turn:
1. Send cancellation signal to core conversation
2. Transition state from `StateWaitingResponse` to `StateIdle`
3. Display cancellation message in transcript
4. Clear any pending approval dialogs
5. Re-enable input field

**Acceptance Criteria:**
- Cancellation is immediate (<100ms)
- State transitions correctly: `StateWaitingResponse` → `StateIdle`
- No errors logged on cancellation (expected behavior)
- Input field becomes active after cancellation

**Edge Cases:**
- Ctrl+C when idle (state already `StateIdle`): Exit application (defer to Ctrl+D behavior)
- Ctrl+C during tool approval: Cancel approval, return to idle
- Ctrl+C during backtrack mode: Exit backtrack, return to idle

#### FR-3.9.2: Graceful Exit (Ctrl+D)
**Priority**: P0 (Critical)

When user presses Ctrl+D:
1. Transition to `StateExiting`
2. Send shutdown signal to core
3. Wait for core cleanup (max 2s timeout)
4. Save session state (if applicable)
5. Exit cleanly with code 0

**Acceptance Criteria:**
- Exit is graceful (no panic, no data loss)
- Core resources cleaned up (goroutines stopped, connections closed)
- Works from any state
- Exit code is 0 on normal exit

#### FR-3.9.3: Screen Clear (Ctrl+L)
**Priority**: P1 (High)

When user presses Ctrl+L:
1. Clear viewport (scroll to bottom, reset scroll position)
2. Optionally: Clear transcript (config option, default: false)
3. Redraw UI
4. Preserve conversation history in memory

**Acceptance Criteria:**
- Screen clears immediately (<50ms)
- Conversation history preserved (can scroll back up)
- Works from `StateIdle` only (ignored in other states)

**Configuration:**
```toml
[tui]
ctrl_l_clears_transcript = false  # Default: just resets view, keeps history
```

#### FR-3.9.4: Help Display (Ctrl+H or ?)
**Priority**: P1 (High)

When user presses Ctrl+H or `?`:
1. Enter `StateHelp` (new state)
2. Show modal overlay with keyboard shortcuts
3. Press any key or Esc to dismiss
4. Return to previous state

**Acceptance Criteria:**
- Help modal is readable and complete
- Lists all keyboard shortcuts with descriptions
- Dismissible with any key
- State restoration after dismiss

**Help Content:**
```
╭─────────────────────────────────────────────╮
│          Keyboard Shortcuts                 │
├─────────────────────────────────────────────┤
│                                             │
│ Enter       - Send message                  │
│ Ctrl+C      - Cancel current turn           │
│ Ctrl+D      - Exit Spin                     │
│ Ctrl+L      - Clear screen                  │
│ Ctrl+H / ?  - Show this help                │
│                                             │
│ Esc-Esc     - Enter backtrack mode          │
│ @           - Open file picker              │
│                                             │
│ PgUp/PgDn   - Scroll transcript             │
│ Home/End    - Jump to top/bottom            │
│                                             │
│ Tool Approval (during approval):            │
│   A         - Approve                       │
│   D         - Deny                          │
│   M         - Modify command                │
│                                             │
│ Press any key to close                      │
╰─────────────────────────────────────────────╯
```

#### FR-3.9.5: Transcript Navigation
**Priority**: P1 (High)

Keyboard shortcuts for transcript scrolling (integrated with viewport):
- **PgUp**: Scroll up one page
- **PgDn**: Scroll down one page
- **Home**: Jump to top
- **End**: Jump to bottom

**Acceptance Criteria:**
- Scroll navigation works in any state except input editing
- Auto-scroll to bottom disabled when user scrolls up
- Auto-scroll re-enabled when user sends new message

**Note:** Most of this is already handled by bubbles/viewport. This FR ensures proper integration with TUI state machine.

### Non-Functional Requirements

#### NFR-3.9.1: Performance
- Keyboard event handling: <5ms latency
- Help modal render: <16ms
- Cancellation response: <100ms

#### NFR-3.9.2: Usability
- All shortcuts follow terminal conventions
- No conflicts between shortcuts
- Help is always accessible
- Shortcuts work consistently across states

#### NFR-3.9.3: Test Coverage
- ≥90% code coverage for keyboard handling
- All state transitions tested
- All shortcuts tested in isolation and combination

## Design

### State Machine Updates

```go
// New state for help display
const (
    StateIdle AppState = iota
    StateWaitingResponse
    StateToolApproval
    StateFilePickerOpen
    StateBacktrackMode
    StateHelp          // NEW
    StateExiting
)
```

### Keyboard Event Router

```go
// In internal/tui/app.go

// handleKeyPress processes all keyboard input
func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    // Global shortcuts (work in any state)
    switch msg.String() {
    case "ctrl+c":
        return m.handleCtrlC()
    case "ctrl+d":
        return m.handleCtrlD()
    case "ctrl+h", "?":
        return m.handleHelp()
    }

    // State-specific shortcuts
    switch m.state {
    case StateIdle:
        return m.handleIdleKeys(msg)
    case StateWaitingResponse:
        return m.handleWaitingKeys(msg)
    case StateToolApproval:
        return m.handleApprovalKeys(msg)
    case StateBacktrackMode:
        return m.handleBacktrackKeys(msg)
    case StateFilePickerOpen:
        return m.handleFilePickerKeys(msg)
    case StateHelp:
        return m.handleHelpKeys(msg)
    case StateExiting:
        // Ignore all input during exit
        return m, nil
    }

    return m, nil
}
```

### Cancellation Flow

```go
// handleCtrlC handles Ctrl+C for command cancellation
func (m Model) handleCtrlC() (tea.Model, tea.Cmd) {
    switch m.state {
    case StateWaitingResponse:
        // Cancel AI generation
        return m.cancelTurn()

    case StateToolApproval:
        // Deny approval and return to idle
        return m.denyApproval()

    case StateBacktrackMode:
        // Exit backtrack mode
        return m.exitBacktrack()

    case StateIdle:
        // Exit application (same as Ctrl+D)
        return m.handleCtrlD()

    default:
        // In other states, do nothing or exit
        return m, nil
    }
}

// cancelTurn sends cancellation to core (Phase 3.11 integration)
func (m Model) cancelTurn() (tea.Model, tea.Cmd) {
    // TODO: Send cancellation via core channel
    // For now, just transition state

    m.chat.AddMessage(ui.Message{
        Role:    ui.RoleSystem,
        Content: "Turn cancelled by user",
    })

    m.state = StateIdle
    return m, nil
}
```

### Help Modal

```go
// In internal/tui/ui/help.go (NEW)

// Help represents the help modal overlay
type Help struct {
    width  int
    height int
}

func NewHelp(width, height int) Help {
    return Help{width: width, height: height}
}

func (h Help) View() string {
    // Render help modal with lipgloss
    // Center modal on screen
    // Use border style with title
    // List all shortcuts
    return helpContent
}
```

## Implementation Plan

### Phase 1: Write Tests (TDD)
1. Create `internal/tui/keyboard_test.go`
2. Write test cases for each shortcut
3. Write state transition tests
4. Write edge case tests

### Phase 2: Implement Handlers
1. Update `handleKeyPress()` in `internal/tui/app.go`
2. Implement `handleCtrlC()`, `handleCtrlD()`, `handleCtrlL()`
3. Implement `handleHelp()` and help state management
4. Add help modal component in `internal/tui/ui/help.go`

### Phase 3: Core Integration Prep
1. Add cancellation channel to Model
2. Add core event types for cancellation
3. Document integration points for Phase 3.11

### Phase 4: Testing & Polish
1. Run all tests with race detector
2. Test keyboard shortcuts in real TUI
3. Fix any state transition bugs
4. Update documentation

## Testing Strategy

### Unit Tests

```go
// internal/tui/keyboard_test.go

func TestCtrlC_CancelTurn(t *testing.T) {
    // Setup model in StateWaitingResponse
    // Send Ctrl+C
    // Assert: state is StateIdle
    // Assert: cancellation message in transcript
}

func TestCtrlC_ExitFromIdle(t *testing.T) {
    // Setup model in StateIdle
    // Send Ctrl+C
    // Assert: state is StateExiting
    // Assert: quitting flag is true
}

func TestCtrlD_GracefulExit(t *testing.T) {
    // Setup model in any state
    // Send Ctrl+D
    // Assert: state is StateExiting
    // Assert: quitting flag is true
}

func TestCtrlL_ClearScreen(t *testing.T) {
    // Setup model with transcript
    // Send Ctrl+L
    // Assert: viewport reset to bottom
    // Assert: transcript preserved
}

func TestCtrlH_ShowHelp(t *testing.T) {
    // Setup model in StateIdle
    // Send Ctrl+H
    // Assert: state is StateHelp
    // Assert: help content rendered
}

func TestQuestionMark_ShowHelp(t *testing.T) {
    // Same as TestCtrlH but with '?'
}

func TestHelp_Dismiss(t *testing.T) {
    // Setup model in StateHelp
    // Send any key
    // Assert: state returns to StateIdle
}

func TestKeyboardShortcutConflicts(t *testing.T) {
    // Ensure no shortcuts conflict
    // Test all combinations
}
```

### Integration Tests

```go
// internal/tui/app_keyboard_integration_test.go

func TestFullKeyboardFlow(t *testing.T) {
    // Test realistic user flow:
    // 1. Type message, press Enter
    // 2. Press Ctrl+C to cancel
    // 3. Press Ctrl+H to show help
    // 4. Dismiss help
    // 5. Press Ctrl+D to exit
}

func TestCancellationDuringApproval(t *testing.T) {
    // Setup tool approval state
    // Press Ctrl+C
    // Assert approval denied, state is Idle
}
```

## Success Criteria

✅ **Complete when:**
1. All FR requirements implemented
2. All NFR requirements met
3. Test coverage ≥90% for keyboard handling code
4. All tests passing with `-race`
5. Linter clean (golangci-lint)
6. Complexity ≤15 for all functions
7. Godoc on all exports
8. Integration with existing TUI complete
9. Help modal implemented and tested
10. FRD approved and merged

## Dependencies

### External Libraries
- `github.com/charmbracelet/bubbletea` - Keyboard event handling
- `github.com/charmbracelet/lipgloss` - Help modal styling

### Internal Modules
- `internal/tui/state.go` - State machine (add `StateHelp`)
- `internal/tui/app.go` - Main keyboard router
- `internal/tui/ui/help.go` - NEW: Help modal component
- `internal/core/conversation.go` - Cancellation support (Phase 3.11)

## Risks & Mitigations

### Risk 1: Keyboard Shortcut Conflicts
**Likelihood:** Medium
**Impact:** High
**Mitigation:**
- Document all shortcuts in one place
- Test for conflicts programmatically
- Follow terminal conventions (Ctrl+C = cancel, Ctrl+D = exit)

### Risk 2: State Transition Bugs
**Likelihood:** Medium
**Impact:** Medium
**Mitigation:**
- Comprehensive state transition tests
- Use state machine validation (CanTransitionTo)
- Log all state transitions in debug mode

### Risk 3: Cancellation Race Conditions
**Likelihood:** Low
**Impact:** High
**Mitigation:**
- Proper channel-based cancellation (context.Context)
- Mutex protection for state changes
- Race detector in all tests

## Future Enhancements

### Phase 2 (Future)
- Customizable keyboard shortcuts (config file)
- Vim-style keybindings mode
- Emacs-style keybindings mode
- Mouse support (click to scroll, select text)
- Incremental search (Ctrl+R, Ctrl+S)
- Copy to clipboard (Ctrl+Shift+C)
- Paste from clipboard (Ctrl+Shift+V)

## References

- **AGENTS.md**: [AGENTS.md](../../AGENTS.md) - Implementation workflow
- **Roadmap**: [specs/ui-modules/ROADMAP.md](../ui-modules/ROADMAP.md) - Phase 3.9
- **Spec**: [specs/ui-modules/spec.md](../ui-modules/spec.md) - Keyboard shortcuts section
- **State Machine**: [internal/tui/state.go](../../internal/tui/state.go)
- **FRD-UI-3.7**: [FRD-UI-3.7.md](FRD-UI-3.7.md) - Transcript navigation
- **FRD-UI-3.8**: [FRD-UI-3.8.md](FRD-UI-3.8.md) - Backtrack mode (Esc-Esc)

## Appendix A: Complete Keyboard Shortcut Reference

### Global Shortcuts (work in any state)

| Key | Action | Description |
|-----|--------|-------------|
| Ctrl+C | Cancel / Exit | Cancel turn (if active), or exit (if idle) |
| Ctrl+D | Exit | Graceful exit from any state |
| Ctrl+H | Help | Show keyboard shortcuts |
| ? | Help | Show keyboard shortcuts (alternative) |

### State-Specific Shortcuts

#### StateIdle
| Key | Action | Description |
|-----|--------|-------------|
| Enter | Submit | Send message to AI |
| Esc | Clear | Clear input field |
| Esc-Esc | Backtrack | Enter backtrack mode |
| @ | File Picker | Open file search |
| Ctrl+L | Clear Screen | Reset viewport to bottom |

#### StateWaitingResponse
| Key | Action | Description |
|-----|--------|-------------|
| Ctrl+C | Cancel | Stop AI generation |

#### StateToolApproval
| Key | Action | Description |
|-----|--------|-------------|
| A | Approve | Execute tool call |
| D | Deny | Reject tool call |
| M | Modify | Edit command before execution |
| Ctrl+C | Cancel | Deny and return to idle |

#### StateBacktrackMode
| Key | Action | Description |
|-----|--------|-------------|
| Esc | Navigate | Move to previous user message |
| Enter | Select | Load message into input |
| Ctrl+C | Cancel | Exit backtrack mode |

#### StateFilePickerOpen
| Key | Action | Description |
|-----|--------|-------------|
| ↑/↓ | Navigate | Move selection |
| Enter | Select | Choose file |
| Tab | Select | Choose file (alternative) |
| Esc | Cancel | Close file picker |

#### StateHelp
| Key | Action | Description |
|-----|--------|-------------|
| Any | Dismiss | Close help modal |

### Navigation Shortcuts (transcript)

| Key | Action | Description |
|-----|--------|-------------|
| PgUp | Scroll Up | Move up one page |
| PgDn | Scroll Down | Move down one page |
| Home | Jump to Top | Go to first message |
| End | Jump to Bottom | Go to last message |

## Appendix B: Implementation Checklist

### Code Changes
- [ ] Add `StateHelp` to state machine
- [ ] Update `handleKeyPress()` in app.go
- [ ] Implement `handleCtrlC()`, `handleCtrlD()`, `handleCtrlL()`, `handleHelp()`
- [ ] Create `internal/tui/ui/help.go` with Help component
- [ ] Add cancellation channel to Model
- [ ] Update View() to render help modal

### Tests
- [ ] Create `internal/tui/keyboard_test.go`
- [ ] Test Ctrl+C in all states
- [ ] Test Ctrl+D in all states
- [ ] Test Ctrl+L behavior
- [ ] Test Ctrl+H / ? help display
- [ ] Test help dismissal
- [ ] Test no shortcut conflicts
- [ ] Integration tests for keyboard flows

### Documentation
- [ ] Update README with keyboard shortcuts
- [ ] Add godoc to all new functions
- [ ] Document state transitions
- [ ] Update ROADMAP.md to mark 3.9 complete

### Quality Gates
- [ ] All tests passing
- [ ] Race detector clean
- [ ] Linter clean
- [ ] Coverage ≥90%
- [ ] Complexity ≤15
- [ ] Manual testing complete

---

**End of FRD-UI-3.9**
