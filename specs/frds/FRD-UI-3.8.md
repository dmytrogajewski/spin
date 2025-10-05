# FRD-UI-3.8: Backtrack Mode (Esc-Esc)

## Metadata
- **FRD ID**: FRD-UI-3.8
- **Title**: Backtrack Mode for Message Editing
- **Component**: UI Modules (TUI)
- **Status**: Draft
- **Created**: 2025-10-05
- **Roadmap**: Phase 3.8

## Overview

Implements backtrack mode, allowing users to navigate backward through their message history, select a previous message, edit it, and resubmit to create a conversation fork.

## Background

Users often want to:
- Fix typos in previous messages
- Rephrase questions for better results
- Try different approaches from a specific point in conversation

Traditional chat interfaces make this difficult. Backtrack mode provides an elegant keyboard-driven solution inspired by terminal history navigation.

## Requirements

### Functional Requirements

#### FR-3.8.1: Esc-Esc Trigger
**Priority**: P0 (Critical)

When the user presses Esc twice with an empty input field:
1. Enter `StateBacktrackMode`
2. Highlight the most recent user message in transcript
3. Set `backtrackIdx` to point to that message
4. Disable normal input editing

**Acceptance Criteria:**
- Esc-Esc only triggers when input is empty
- If input has text, first Esc should clear it (existing behavior)
- State transition: `StateIdle` → `StateBacktrackMode`

#### FR-3.8.2: Navigation with Esc
**Priority**: P0 (Critical)

When in `StateBacktrackMode`, pressing Esc steps backward to the previous user message:
1. Decrement `backtrackIdx`
2. Skip non-user messages (assistant, system, tool)
3. Update highlight in transcript
4. Stop at the first user message (no wrap-around)

**Acceptance Criteria:**
- Only user messages are selectable
- Visual highlight moves to older messages with each Esc press
- Cannot go before the first user message

#### FR-3.8.3: Message Selection with Enter
**Priority**: P0 (Critical)

When in `StateBacktrackMode`, pressing Enter:
1. Load the selected message into input field
2. Exit backtrack mode (`StateBacktrackMode` → `StateIdle`)
3. Focus input for editing
4. Clear `backtrackIdx`

**Acceptance Criteria:**
- Selected message content appears in input
- User can edit before resubmitting
- Backtrack mode exits cleanly

#### FR-3.8.4: Conversation Forking
**Priority**: P1 (High)

When user edits and resubmits a backtracked message:
1. Truncate conversation at the selected message
2. Replace old message with edited version
3. Continue conversation from that point
4. Old continuation is discarded (simple fork, no branch storage)

**Acceptance Criteria:**
- Messages after the selected point are removed
- Edited message appears in transcript
- Conversation continues naturally from edited point

#### FR-3.8.5: Cancel Backtrack
**Priority**: P0 (Critical)

User can cancel backtrack mode by:
- Pressing `Ctrl+C`: Exit backtrack, return to idle
- Pressing `Ctrl+D`: Exit application

**Acceptance Criteria:**
- Backtrack mode can be cancelled without selection
- No changes to transcript when cancelled
- Input remains empty after cancel

### Non-Functional Requirements

#### NFR-3.8.1: Performance
- Highlight update: <5ms
- Esc navigation response: <10ms
- No lag when navigating through 1000+ messages

#### NFR-3.8.2: Usability
- Visual indicator shows backtrack mode is active
- Selected message clearly highlighted (distinct from normal styling)
- Status bar shows "Backtrack Mode" state

#### NFR-3.8.3: Test Coverage
- ≥90% code coverage for backtrack logic
- All state transitions tested
- Edge cases covered (empty transcript, single message, etc.)

## Design

### State Machine

```
StateIdle (input empty) + Esc → (escPressCount = 1)
  + Esc again → StateBacktrackMode

StateBacktrackMode + Esc → navigate to previous user message
StateBacktrackMode + Enter → load message, transition to StateIdle
StateBacktrackMode + Ctrl+C → cancel, transition to StateIdle
```

### Data Structures

```go
// In Model (internal/tui/app.go)
type Model struct {
    // ... existing fields

    backtrackIdx   int  // Index of selected message in chat.messages
    escPressCount  int  // Track consecutive Esc presses
}
```

### Message Highlighting

```go
// In Message (internal/tui/ui/message.go)
type Message struct {
    // ... existing fields

    Highlighted bool // True if this message is selected in backtrack
}

// In Chat (internal/tui/ui/chat.go)
func (c *Chat) SetHighlight(idx int)
func (c *Chat) ClearHighlight()
func (c Chat) GetUserMessageIndices() []int // Returns indices of all user messages
```

### Conversation Forking

```go
// In Chat (internal/tui/ui/chat.go)
func (c *Chat) TruncateAfter(idx int) {
    // Remove all messages after idx
    if idx >= 0 && idx < len(c.messages) {
        c.messages = c.messages[:idx+1]
        c.contentDirty = true
    }
}

func (c *Chat) ReplaceMessage(idx int, content string) {
    // Replace message at idx with new content
    if idx >= 0 && idx < len(c.messages) {
        c.messages[idx].Content = content
        c.contentDirty = true
    }
}
```

## Implementation Plan

### Phase 1: Core Logic (TDD)
1. Write tests for Esc-Esc detection
2. Write tests for backtrack navigation
3. Write tests for message selection
4. Implement logic to pass tests

### Phase 2: UI Integration
1. Add highlight support to chat.go
2. Implement visual feedback in view.go
3. Update status bar to show "Backtrack Mode"

### Phase 3: Conversation Forking
1. Implement truncate logic
2. Test forking behavior
3. Verify conversation continuity

## Testing Strategy

### Unit Tests

```go
// internal/tui/app_backtrack_test.go

func TestBacktrackModeEntry(t *testing.T)
func TestBacktrackNavigation(t *testing.T)
func TestBacktrackSelection(t *testing.T)
func TestBacktrackCancel(t *testing.T)
func TestBacktrackWithEmptyTranscript(t *testing.T)
func TestBacktrackWithSingleMessage(t *testing.T)
func TestConversationForking(t *testing.T)
```

### Integration Tests
- End-to-end backtrack flow
- State transitions with real chat component
- Message highlighting rendering

## Success Criteria

✅ **Complete when:**
1. All FR requirements implemented
2. All NFR requirements met
3. Test coverage ≥90%
4. All tests passing with `-race`
5. Linter clean
6. Complexity ≤15 for all functions
7. Godoc on all exports
8. FRD approved and merged

## References

- **Roadmap**: [specs/ui-modules/ROADMAP.md](../ui-modules/ROADMAP.md) - Phase 3.8
- **Spec**: [specs/ui-modules/spec.md](../ui-modules/spec.md) - Section 6
- **State Machine**: [internal/tui/state.go](../../internal/tui/state.go)
- **Chat Component**: [internal/tui/ui/chat.go](../../internal/tui/ui/chat.go)

## Appendix

### Example User Flow

```
1. User: "Fix the bug"
   Assistant: "I'll check the code..."

2. User: "Run tests"
   Assistant: "Running tests..."

3. [User presses Esc-Esc]
   → Highlights "Run tests" ← currently selected

4. [User presses Esc]
   → Highlights "Fix the bug" ← currently selected

5. [User presses Enter]
   → Input shows: "Fix the bug"

6. [User edits to: "Fix the authentication bug"]
   [User presses Enter to submit]

7. Transcript now shows:
   User: "Fix the authentication bug"  ← replaced
   (Old "Run tests" conversation removed)
   Assistant: [new response...]
```

### Visual Design

**Normal Mode:**
```
You │ 14:30:00
Fix the bug

Assistant │ 14:30:05
I'll check the code...
```

**Backtrack Mode (selected message):**
```
┃ You │ 14:30:00          ← highlight border
┃ Fix the bug
┗━━━━━━━━━━━━━━━━━━━━━━━

Assistant │ 14:30:05
I'll check the code...

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
⏪ Backtrack Mode - Esc: older | Enter: edit
```
