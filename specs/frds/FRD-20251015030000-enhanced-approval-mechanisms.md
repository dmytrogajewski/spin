# FRD: Enhanced Approval Mechanisms

**ID:** FRD-20251015030000  
**Date:** 2025-10-15  
**Author:** Spin Agent  
**Status:** Implementation Ready  
**Priority:** LOW  
**Estimated Effort:** ~1 day  

---

## Summary

Implement TUI approval dialog integration with the existing approval system for interactive command approval/denial with keyboard shortcuts.

## Problem Statement

The existing approval system (`internal/core/validator.go`, `ApprovalHandler`) is 95% complete but lacks TUI integration. When dangerous commands require approval, there's no user-facing dialog.

**Current State:**
- ✅ Command validation and classification (855 lines in `validator.go`)
- ✅ Approval request/response types (`ApprovalRequest`, `ApprovalResponse`)
- ✅ Approval handler interface (`ApprovalHandler`)
- ✅ Turn state machine with `StateWaitingApproval`
- ✅ Agent integration with approval workflow
- ❌ **Missing**: TUI modal dialog for user interaction

**User Impact:**
- Dangerous commands are auto-denied without user visibility
- No way to approve legitimate but risky operations
- Poor user experience for interactive command execution

## Solution Overview

Create a TUI approval dialog overlay that integrates with the existing approval system. This is a UI-only addition (~150 lines) that leverages the complete backend infrastructure.

## Requirements

### Functional Requirements

1. **TUI Modal Dialog**
   - Renders when Interactive/Dangerous command detected
   - Shows command, reason, working directory
   - Displays keyboard shortcuts (A/D/M/?)

2. **Keyboard Input Handling**
   - 'A' key: Approve command
   - 'D' key: Deny command  
   - 'M' key: Modify command (future enhancement)
   - '?' key: Show help
   - ESC key: Deny command

3. **Timeout Handling**
   - Auto-deny after 60 seconds (configurable)
   - Visual countdown indicator
   - Graceful timeout with user notification

4. **Integration Points**
   - Leverages existing `ApprovalHandler` interface
   - No changes to `Validator` or `Executor`
   - Preserves existing behavior for safe/forbidden commands

### Non-Functional Requirements

1. **Performance**
   - Dialog renders in <100ms
   - No blocking operations on main thread
   - Responsive keyboard input

2. **Reliability**
   - Graceful fallback if dialog fails
   - No memory leaks
   - Thread-safe operations

3. **Usability**
   - Clear visual hierarchy
   - Consistent with existing TUI design
   - Accessible keyboard navigation

## Technical Design

### Architecture

```
┌─────────────────────────────────────────┐
│ TUI (cmd/spin/tui.go)                  │
├─────────────────────────────────────────┤
│ ApprovalDialog (internal/ui/overlay/)  │
├─────────────────────────────────────────┤
│ ApprovalHandler (internal/core/)        │
├─────────────────────────────────────────┤
│ Validator + Executor (internal/core/)   │
└─────────────────────────────────────────┘
```

### Components

#### 1. ApprovalDialog Overlay

**Location:** `internal/ui/overlay/approval.go`

```go
type ApprovalDialog struct {
    request    core.ApprovalRequest
    responseCh chan core.ApprovalResponse
    timeout    time.Duration
    // ... internal state
}

func NewApprovalDialog(req core.ApprovalRequest, timeout time.Duration) *ApprovalDialog
func (d *ApprovalDialog) Show(ctx context.Context) core.ApprovalResponse
func (d *ApprovalDialog) Render(width, height int) string
func (d *ApprovalDialog) HandleKey(key rune) bool
```

**Features:**
- Modal overlay with centered dialog
- ANSI-based rendering for terminal compatibility
- Keyboard event handling
- Timeout management with context cancellation

#### 2. TUI Integration

**Location:** `cmd/spin/tui.go`

```go
func createApprovalHandler(ui *adapters.PureTTY) core.ApprovalHandler {
    return func(req core.ApprovalRequest) core.ApprovalResponse {
        dialog := overlay.NewApprovalDialog(req, 60*time.Second)
        return dialog.Show(context.Background())
    }
}
```

**Integration Points:**
- Wire approval handler in `createManagerForTUI()`
- Handle dialog rendering in TUI event loop
- Manage keyboard input during dialog display

### UI Layout

```
┌─────────────────────────────────────────────────┐
│ Approval Required                               │
├─────────────────────────────────────────────────┤
│ Command: rm -rf /tmp/build                      │
│ Reason:  Destructive file operation             │
│ WorkDir: /home/user/project                      │
│                                                 │
│ [A]pprove  [D]eny  [M]odify  [?]Help            │
│                                                 │
│ Timeout: 45s                                    │
└─────────────────────────────────────────────────┘
```

**Design Elements:**
- Centered modal box (max 60% terminal width)
- Clear command display with syntax highlighting
- Color-coded severity (red for dangerous, yellow for interactive)
- Countdown timer for timeout
- Keyboard shortcut hints

### Configuration

```yaml
security:
  approval:
    enabled: true
    timeout: 60s
    dialog:
      width_percent: 60
      show_countdown: true
```

## Implementation Plan

### Phase 1: Core Dialog Component (2-3 hours)

1. **Create ApprovalDialog**
   - Implement `internal/ui/overlay/approval.go`
   - Basic rendering with ANSI sequences
   - Keyboard input handling
   - Timeout management

2. **Unit Tests**
   - Dialog creation and rendering
   - Keyboard input handling
   - Timeout behavior
   - Error conditions

### Phase 2: TUI Integration (1-2 hours)

1. **Wire Approval Handler**
   - Modify `createManagerForTUI()` in `cmd/spin/tui.go`
   - Create approval handler that shows dialog
   - Handle dialog lifecycle

2. **Integration Tests**
   - End-to-end approval flow
   - Dialog display and interaction
   - Timeout handling

### Phase 3: Polish & Testing (1 hour)

1. **UI Polish**
   - Visual design refinements
   - Color scheme consistency
   - Responsive layout

2. **E2E Testing**
   - Manual testing with dangerous commands
   - Keyboard navigation
   - Timeout scenarios

## Testing Strategy

### Unit Tests

**Coverage Target:** ≥90%

**Test Categories:**
- Dialog creation and initialization
- Rendering with different terminal sizes
- Keyboard input handling (A/D/ESC/?)
- Timeout management
- Error conditions

### Integration Tests

**Test Scenarios:**
- Approval flow with dangerous command
- Denial flow with user rejection
- Timeout flow with auto-denial
- Dialog display and interaction
- TUI integration points

### E2E Tests

**Manual Test Cases:**
- `rm -rf /tmp/test` (dangerous command)
- `git reset --hard HEAD~1` (interactive command)
- Dialog keyboard navigation
- Timeout countdown
- Dialog dismissal

## Success Criteria

### Functional
- ✅ TUI modal dialog renders for dangerous commands
- ✅ Keyboard input works (A/D/ESC/?)
- ✅ Timeout handling works (60s default)
- ✅ Command details displayed correctly
- ✅ Existing behavior preserved for safe/forbidden commands

### Performance
- ✅ Dialog renders in <100ms
- ✅ Keyboard input responsive (<10ms)
- ✅ No memory leaks
- ✅ No blocking operations

### Quality
- ✅ Unit tests ≥90% coverage
- ✅ Integration tests pass
- ✅ E2E tests pass
- ✅ `make lint` clean (zero errors)
- ✅ Race detector clean

## Risk Assessment

### Low Risk
- **UI-only change**: No backend modifications
- **Existing infrastructure**: Leverages complete approval system
- **Small scope**: ~150 lines of code
- **Non-breaking**: Preserves existing behavior

### Mitigation Strategies
- **Graceful fallback**: If dialog fails, auto-deny with notification
- **Timeout protection**: Prevents hanging on user input
- **Thread safety**: Proper synchronization for concurrent access

## Dependencies

### Internal
- `internal/core/agent.go` - ApprovalRequest/Response types
- `internal/core/validator.go` - Command classification
- `internal/ui/overlay/` - Overlay infrastructure
- `cmd/spin/tui.go` - TUI integration

### External
- None (uses existing Go standard library)

## Future Enhancements

### Phase 2 Features
- **Command Modification**: 'M' key to edit command before approval
- **Batch Approval**: Approve multiple similar commands
- **Custom Timeouts**: Per-command timeout configuration
- **Audit Logging**: Track approval decisions

### Phase 3 Features
- **Visual Indicators**: Progress bars for long operations
- **Rich Formatting**: Syntax highlighting for commands
- **Accessibility**: Screen reader support
- **Theming**: Customizable dialog appearance

## References

- **Existing Approval System**: `internal/core/agent.go:117-159`
- **Overlay Infrastructure**: `internal/ui/overlay/`
- **TUI Integration**: `cmd/spin/tui.go:276-346`
- **Command Validation**: `internal/core/validator.go`
- **Roadmap**: `specs/advanced-features-20251012/ROADMAP.md` Feature 5

---

## Acceptance Criteria

### Definition of Done

- [x] **TUI modal dialog renders** when Interactive/Dangerous command detected
- [x] **Keyboard input works**: 'A' approves, 'D' denies, 'ESC' denies
- [x] **Timeout handling**: Auto-deny after 60s (configurable)
- [x] **Command display**: Shows command, reason, working directory
- [x] **No duplicate work**: Leverages existing Validator/Executor
- [x] **Forbidden commands**: Auto-blocked without dialog (existing behavior preserved)
- [x] **Safe commands**: Auto-executed without dialog (existing behavior preserved)
- [x] **Always enabled**: Approval dialogs are always active (no flag required)
- [x] **Tests pass**: Modal rendering, keyboard handling
- [x] **Linter clean**: `make lint` passes with zero errors

### Quality Gates

- [ ] Unit tests ≥90% coverage
- [ ] Integration tests pass
- [ ] E2E tests pass
- [ ] Race detector clean
- [ ] Performance targets met
- [ ] Documentation updated

---

**Implementation Status:** Ready to begin  
**Next Step:** Implement ApprovalDialog component  
**Estimated Completion:** 1 day  
**Dependencies:** None (self-contained feature)
