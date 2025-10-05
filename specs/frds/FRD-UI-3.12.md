# FRD-UI-3.12: TUI Error Handling & Display

**Status:** Draft
**Created:** 2025-10-05
**Phase:** 3.12 - TUI Error Handling & Display

---

## Overview

This FRD defines comprehensive error handling and display mechanisms for the Spin TUI. It implements inline error display in the transcript, transient status bar notifications, modal overlays for critical errors, graceful recovery mechanisms, and dismissible error notifications.

---

## Context

The TUI currently handles `EventError` from core by calling `handleError()`, but lacks a complete error display and recovery system. Users need clear, actionable error messages with appropriate severity levels and recovery options.

**Related:**
- [specs/ui-modules/spec.md](../ui-modules/spec.md) - Section "TUI Error Handling"
- [specs/ui-modules/ROADMAP.md](../ui-modules/ROADMAP.md) - Phase 3.12
- [internal/core/event.go](../../internal/core/event.go) - EventError, ErrorData
- [internal/core/error.go](../../internal/core/error.go) - Error types, ErrorCode
- [internal/tui/event_handler.go](../../internal/tui/event_handler.go) - handleError() stub

---

## Requirements

### Functional Requirements

#### FR-1: Error Severity Classification
- **FR-1.1**: Classify errors into severity levels: Info, Warning, Error, Critical
- **FR-1.2**: Map core.ErrorCode to severity levels
- **FR-1.3**: Support custom severity override from ErrorData

#### FR-2: Inline Error Display
- **FR-2.1**: Display non-critical errors inline in transcript
- **FR-2.2**: Format errors with severity icon, timestamp, and message
- **FR-2.3**: Syntax highlight error details/stack traces
- **FR-2.4**: Auto-scroll to new errors

#### FR-3: Status Bar Notifications
- **FR-3.1**: Display transient errors in status bar (warnings, info)
- **FR-3.2**: Auto-dismiss after 5 seconds
- **FR-3.3**: Color-code by severity
- **FR-3.4**: Show countdown indicator for auto-dismiss

#### FR-4: Modal Error Overlays
- **FR-4.1**: Display critical errors in centered modal overlay
- **FR-4.2**: Block input until user acknowledges
- **FR-4.3**: Show full error details with scrolling
- **FR-4.4**: Provide "Dismiss" and "View Logs" actions

#### FR-5: Error Recovery
- **FR-5.1**: Auto-reconnect on network failures (max 3 retries)
- **FR-5.2**: Resume conversation after transient errors
- **FR-5.3**: Preserve state during recovery
- **FR-5.4**: Show recovery progress

#### FR-6: User Actions
- **FR-6.1**: Allow error dismissal (Esc or Enter)
- **FR-6.2**: Copy error details to clipboard (Ctrl+C in modal)
- **FR-6.3**: Navigate error history (up/down in modal)
- **FR-6.4**: Toggle error details expansion

### Non-Functional Requirements

#### NFR-1: Performance
- **NFR-1.1**: Error display latency < 10ms
- **NFR-1.2**: Modal render time < 16ms (60 FPS)
- **NFR-1.3**: No blocking on error emission

#### NFR-2: UX
- **NFR-2.1**: Errors never cause TUI crash
- **NFR-2.2**: All error messages actionable
- **NFR-2.3**: Clear visual distinction by severity
- **NFR-2.4**: Keyboard-accessible error actions

#### NFR-3: Testing
- **NFR-3.1**: ≥90% test coverage for error handling code
- **NFR-3.2**: All error scenarios tested
- **NFR-3.3**: Recovery mechanisms tested

---

## Design

### Error Severity Levels

```go
// ErrorSeverity represents the severity level of an error.
type ErrorSeverity int

const (
    SeverityInfo ErrorSeverity = iota    // Informational messages
    SeverityWarning                      // Warnings (recoverable issues)
    SeverityError                        // Errors (operation failed)
    SeverityCritical                     // Critical (system failure)
)
```

**Severity Mapping:**

| core.ErrorCode | Severity | Display | Auto-Dismiss |
|---|---|---|---|
| ErrCodeUnknown | Error | Inline | No |
| ErrCodeInvalidInput | Warning | Status Bar | Yes (5s) |
| ErrCodeNotFound | Warning | Status Bar | Yes (5s) |
| ErrCodeAlreadyExists | Warning | Status Bar | Yes (5s) |
| ErrCodePermissionDenied | Error | Inline | No |
| ErrCodeTimeout | Warning | Status Bar | Yes (5s) |
| ErrCodeCancelled | Info | Status Bar | Yes (3s) |
| ErrCodeInternal | Critical | Modal | No |
| ErrCodeExternal | Error | Inline | No |

### Error Display Strategy

```
┌──────────────────────────────────────────────┐
│ Event: EventError (from core)               │
│ Data: ErrorData { Message, Code, Details }  │
└──────────────────────────────────────────────┘
                    │
                    ▼
      ┌─────────────────────────────┐
      │ ClassifySeverity(ErrorData) │
      └─────────────────────────────┘
                    │
        ┌───────────┴───────────┐
        │                       │
    Info/Warning            Error/Critical
        │                       │
        ▼                       ▼
  ┌────────────┐        ┌──────────────┐
  │ Status Bar │        │   Inline     │
  │ (5s auto)  │        │  Transcript  │
  └────────────┘        └──────────────┘
                              │
                      (if Critical)
                              ▼
                        ┌──────────┐
                        │  Modal   │
                        │ Overlay  │
                        └──────────┘
```

### Component Structure

```
internal/tui/
├── error.go                 # Error handling logic
├── error_test.go            # Error handling tests
└── ui/
    ├── error_modal.go       # Modal error overlay
    ├── error_modal_test.go  # Modal tests
    └── statusbar.go         # Updated for error display
```

### Error Display Components

#### 1. Inline Transcript Errors

```
┌────────────────────────────────────────────────┐
│ User: Read config.yaml                         │
│ Assistant: I'll read the configuration file... │
│                                                │
│ ⚠️  Error: Permission Denied (12:34:56)        │
│ ├─ Operation: Executor.Execute                 │
│ ├─ Code: permission_denied                     │
│ └─ Details: Cannot read /etc/config.yaml       │
│            Try running with sudo or check      │
│            file permissions                    │
│                                                │
│ User: _                                        │
└────────────────────────────────────────────────┘
```

#### 2. Status Bar Transient Errors

```
┌────────────────────────────────────────────────┐
│ llama3.1 | 🔒 | ~/project | 1.2K tokens        │
│                                                │
│ ⚠️  Warning: File not found (dismiss in 4s)   │
└────────────────────────────────────────────────┘
```

#### 3. Critical Error Modal

```
┌────────────────────────────────────────────────┐
│                                                │
│  ╔════════════════════════════════════════╗   │
│  ║   ❌ Critical Error                     ║   │
│  ╠════════════════════════════════════════╣   │
│  ║                                        ║   │
│  ║  LLM Provider Connection Failed        ║   │
│  ║                                        ║   │
│  ║  Operation: Agent.RunTurn              ║   │
│  ║  Code: external                        ║   │
│  ║  Time: 2025-10-05 12:34:56             ║   │
│  ║                                        ║   │
│  ║  Details:                              ║   │
│  ║  Failed to connect to http://...      ║   │
│  ║  Connection refused (ECONNREFUSED)     ║   │
│  ║                                        ║   │
│  ║  Retrying in 2 seconds... (1/3)        ║   │
│  ║                                        ║   │
│  ║  [Esc] Dismiss  [V] View Logs          ║   │
│  ║  [C] Copy Error                        ║   │
│  ╚════════════════════════════════════════╝   │
│                                                │
└────────────────────────────────────────────────┘
```

### Data Structures

```go
// ErrorDisplay represents an error for display in the TUI.
type ErrorDisplay struct {
    Message     string        // User-friendly message
    Code        string        // Error code (from core.ErrorCode)
    Details     string        // Technical details
    Operation   string        // Operation that failed
    Severity    ErrorSeverity // Severity level
    Timestamp   time.Time     // When error occurred
    Dismissible bool          // Can user dismiss?
    Dismissed   bool          // Has user dismissed?
    AutoDismiss time.Duration // Auto-dismiss after (0 = never)
}

// ErrorModal represents the critical error modal overlay.
type ErrorModal struct {
    Errors      []ErrorDisplay // Error history
    CurrentIdx  int            // Currently displayed error
    Width       int            // Modal width
    Height      int            // Modal height
    Visible     bool           // Is modal showing?
}
```

### Recovery Mechanisms

```go
// RecoveryStrategy defines error recovery behavior.
type RecoveryStrategy struct {
    ErrorCode     core.ErrorCode
    MaxRetries    int
    RetryDelay    time.Duration
    RecoverFunc   func(error) error
    ShouldRetry   func(error) bool
}

// Default recovery strategies
var defaultStrategies = []RecoveryStrategy{
    {
        ErrorCode:   core.ErrCodeExternal,
        MaxRetries:  3,
        RetryDelay:  2 * time.Second,
        ShouldRetry: isNetworkError,
        RecoverFunc: reconnectLLM,
    },
    {
        ErrorCode:   core.ErrCodeTimeout,
        MaxRetries:  1,
        RetryDelay:  5 * time.Second,
        ShouldRetry: isTemporaryTimeout,
        RecoverFunc: retryOperation,
    },
}
```

---

## Implementation Plan

### Step 1: Error Classification & Display Types (TDD)

**Test First:**
```go
// internal/tui/error_test.go
func TestClassifySeverity(t *testing.T) {
    tests := []struct {
        name     string
        errData  core.ErrorData
        expected ErrorSeverity
    }{
        {
            name: "internal error is critical",
            errData: core.ErrorData{
                Code: core.ErrCodeInternal.String(),
            },
            expected: SeverityCritical,
        },
        {
            name: "timeout is warning",
            errData: core.ErrorData{
                Code: core.ErrCodeTimeout.String(),
            },
            expected: SeverityWarning,
        },
        // ... more test cases
    }
}

func TestErrorDisplay_ShouldShowInModal(t *testing.T) {
    // Test critical errors show in modal
    // Test errors show inline
    // Test warnings show in status bar
}

func TestErrorDisplay_AutoDismiss(t *testing.T) {
    // Test auto-dismiss timing
    // Test dismissible vs non-dismissible
}
```

**Implementation:**
```go
// internal/tui/error.go

// ErrorSeverity represents error severity levels.
type ErrorSeverity int

const (
    SeverityInfo ErrorSeverity = iota
    SeverityWarning
    SeverityError
    SeverityCritical
)

// ClassifySeverity maps core error codes to severity levels.
func ClassifySeverity(errData core.ErrorData) ErrorSeverity {
    // Implementation based on mapping table
}

// ErrorDisplay represents an error for TUI display.
type ErrorDisplay struct {
    Message     string
    Code        string
    Details     string
    Operation   string
    Severity    ErrorSeverity
    Timestamp   time.Time
    Dismissible bool
    Dismissed   bool
    AutoDismiss time.Duration
}

// NewErrorDisplay creates an ErrorDisplay from core.ErrorData.
func NewErrorDisplay(errData core.ErrorData) ErrorDisplay {
    // Create display error with severity, auto-dismiss, etc.
}

// ShouldShowInModal returns true if error should display in modal.
func (e ErrorDisplay) ShouldShowInModal() bool {
    return e.Severity == SeverityCritical
}

// ShouldShowInStatusBar returns true if error should display in status bar.
func (e ErrorDisplay) ShouldShowInStatusBar() bool {
    return e.Severity == SeverityInfo || e.Severity == SeverityWarning
}

// ShouldShowInTranscript returns true if error should display inline.
func (e ErrorDisplay) ShouldShowInTranscript() bool {
    return e.Severity == SeverityError || e.Severity == SeverityCritical
}
```

### Step 2: Error Modal Component (TDD)

**Test First:**
```go
// internal/tui/ui/error_modal_test.go
func TestErrorModal_Render(t *testing.T) {
    // Test modal rendering
    // Test centering
    // Test scrolling for long errors
}

func TestErrorModal_Navigation(t *testing.T) {
    // Test up/down navigation
    // Test dismiss action
    // Test copy action
}

func TestErrorModal_Update(t *testing.T) {
    // Test keyboard handling
    // Test state transitions
}
```

**Implementation:**
```go
// internal/tui/ui/error_modal.go

// ErrorModal displays critical errors in a modal overlay.
type ErrorModal struct {
    Errors     []ErrorDisplay
    CurrentIdx int
    Width      int
    Height     int
    Visible    bool
}

// NewErrorModal creates a new ErrorModal.
func NewErrorModal() ErrorModal {
    return ErrorModal{
        Errors:  make([]ErrorDisplay, 0),
        Width:   60,
        Height:  20,
        Visible: false,
    }
}

// Show displays the modal with the given error.
func (m *ErrorModal) Show(err ErrorDisplay) {
    m.Errors = append(m.Errors, err)
    m.CurrentIdx = len(m.Errors) - 1
    m.Visible = true
}

// Hide dismisses the modal.
func (m *ErrorModal) Hide() {
    m.Visible = false
}

// Update handles keyboard input (Bubble Tea pattern).
func (m ErrorModal) Update(msg tea.Msg) (ErrorModal, tea.Cmd) {
    // Handle Esc, Enter, Up, Down, C (copy)
}

// View renders the modal (Bubble Tea pattern).
func (m ErrorModal) View() string {
    if !m.Visible || len(m.Errors) == 0 {
        return ""
    }
    // Render centered modal with current error
}
```

### Step 3: Status Bar Error Display (TDD)

**Test First:**
```go
// internal/tui/ui/statusbar_test.go (extend existing)
func TestStatusBar_ShowError(t *testing.T) {
    // Test error display
    // Test auto-dismiss
    // Test color coding
}

func TestStatusBar_AutoDismissCountdown(t *testing.T) {
    // Test countdown ticker
}
```

**Implementation:**
```go
// internal/tui/ui/statusbar.go (extend)

// ShowError displays a transient error in the status bar.
func (s *StatusBar) ShowError(err ErrorDisplay) {
    s.errorMsg = err
    s.errorShownAt = time.Now()
}

// ClearError clears the error message.
func (s *StatusBar) ClearError() {
    s.errorMsg = nil
}

// Update the View() method to render error if present and not expired
```

### Step 4: Inline Transcript Error Display (TDD)

**Test First:**
```go
// internal/tui/ui/chat_test.go (extend existing)
func TestChat_AddError(t *testing.T) {
    // Test error message addition
    // Test error formatting
    // Test auto-scroll
}

func TestChat_RenderError(t *testing.T) {
    // Test error rendering with icons
    // Test syntax highlighting of details
}
```

**Implementation:**
```go
// internal/tui/ui/chat.go (extend)

// AddError adds an error message to the transcript.
func (c *Chat) AddError(err ErrorDisplay) {
    msg := Message{
        Role:      "system",
        Content:   formatErrorMessage(err),
        Timestamp: err.Timestamp,
        IsError:   true,
    }
    c.AddMessage(msg)
    c.ScrollToBottom()
}

// formatErrorMessage formats an error for display.
func formatErrorMessage(err ErrorDisplay) string {
    // Format with icon, operation, code, details
}
```

### Step 5: Error Recovery (TDD)

**Test First:**
```go
// internal/tui/error_test.go
func TestErrorRecovery_NetworkError(t *testing.T) {
    // Test auto-reconnect
    // Test max retries
    // Test retry delay
}

func TestErrorRecovery_TransientError(t *testing.T) {
    // Test recovery on transient errors
}

func TestErrorRecovery_StatePreservation(t *testing.T) {
    // Test state preserved during recovery
}
```

**Implementation:**
```go
// internal/tui/error.go

// RecoveryManager handles error recovery strategies.
type RecoveryManager struct {
    strategies map[core.ErrorCode]RecoveryStrategy
    retries    map[string]int // Track retry count per error type
}

// NewRecoveryManager creates a recovery manager with default strategies.
func NewRecoveryManager() *RecoveryManager {
    // Initialize with default recovery strategies
}

// TryRecover attempts to recover from an error.
func (r *RecoveryManager) TryRecover(err error) (recovered bool, cmd tea.Cmd) {
    // Determine strategy based on error code
    // Check retry count
    // Execute recovery function
    // Return command for delayed retry if needed
}
```

### Step 6: Integration with TUI Model (TDD)

**Test First:**
```go
// internal/tui/app_test.go (extend)
func TestModel_HandleError_Critical(t *testing.T) {
    // Test critical error shows modal
}

func TestModel_HandleError_Warning(t *testing.T) {
    // Test warning shows in status bar
}

func TestModel_HandleError_Inline(t *testing.T) {
    // Test error shows in transcript
}

func TestModel_ErrorRecovery(t *testing.T) {
    // Test recovery flow
}
```

**Implementation:**
```go
// internal/tui/app.go (extend)

// Add error state to Model
type Model struct {
    // ... existing fields
    errorModal   ui.ErrorModal
    recovery     *RecoveryManager
}

// internal/tui/event_handler.go (complete handleError)

func (m Model) handleError(event core.Event) Model {
    if errData, ok := event.Data.(*core.ErrorData); ok {
        // Create ErrorDisplay
        errDisplay := NewErrorDisplay(*errData)

        // Route based on severity
        if errDisplay.ShouldShowInModal() {
            m.errorModal.Show(errDisplay)
            m.state = StateError
        } else if errDisplay.ShouldShowInStatusBar() {
            m.statusBar.ShowError(errDisplay)
        } else if errDisplay.ShouldShowInTranscript() {
            m.chat.AddError(errDisplay)
        }

        // Attempt recovery if applicable
        // (returns Bubble Tea command for retry)
    }
    return m
}
```

---

## Testing Strategy

### Unit Tests

```go
// internal/tui/error_test.go
- TestClassifySeverity (all error codes)
- TestNewErrorDisplay (error creation)
- TestErrorDisplay_ShouldShowInModal
- TestErrorDisplay_ShouldShowInStatusBar
- TestErrorDisplay_ShouldShowInTranscript
- TestErrorDisplay_AutoDismiss
- TestRecoveryManager_TryRecover
- TestRecoveryManager_MaxRetries
- TestRecoveryManager_RetryDelay

// internal/tui/ui/error_modal_test.go
- TestErrorModal_Show
- TestErrorModal_Hide
- TestErrorModal_Navigation
- TestErrorModal_Render
- TestErrorModal_Copy
- TestErrorModal_Keyboard

// internal/tui/ui/statusbar_test.go (extend)
- TestStatusBar_ShowError
- TestStatusBar_ClearError
- TestStatusBar_AutoDismiss
- TestStatusBar_ColorCoding

// internal/tui/ui/chat_test.go (extend)
- TestChat_AddError
- TestChat_RenderError
- TestChat_ErrorFormatting
```

### Integration Tests

```go
// internal/tui/app_test.go (extend)
- TestModel_HandleError_CriticalModal
- TestModel_HandleError_WarningStatusBar
- TestModel_HandleError_InlineTranscript
- TestModel_ErrorRecovery_Network
- TestModel_ErrorRecovery_Transient
- TestModel_ErrorDismissal
- TestModel_ErrorNavigation
```

### Manual Testing Checklist

- [ ] Critical error displays modal overlay
- [ ] Modal is centered and sized correctly
- [ ] Error details are readable and formatted
- [ ] Esc/Enter dismisses modal
- [ ] Warnings display in status bar with countdown
- [ ] Status bar auto-dismisses after timeout
- [ ] Inline errors appear in transcript
- [ ] Error icons display correctly (⚠️ ❌ ℹ️)
- [ ] Error colors match severity
- [ ] Network error triggers auto-reconnect
- [ ] Retry progress displays
- [ ] Max retries respected
- [ ] State preserved during recovery
- [ ] No TUI crash on any error type
- [ ] Keyboard navigation works in error modal
- [ ] Copy error to clipboard works

---

## Quality Gates

### Code Quality
- [x] Tests written first (TDD)
- [ ] All tests passing
- [ ] ≥90% coverage (error handling code)
- [ ] Race detector clean
- [ ] Linter clean (golangci-lint)
- [ ] Complexity ≤15 for all functions
- [ ] Godoc on all exports

### Performance
- [ ] Error display latency < 10ms
- [ ] Modal render time < 16ms
- [ ] No blocking on error emission
- [ ] Auto-dismiss timer accurate (±100ms)

### UX
- [ ] All error messages actionable
- [ ] Clear severity visual distinction
- [ ] Keyboard-accessible actions
- [ ] No TUI crashes on any error
- [ ] Error recovery transparent to user

---

## Success Criteria

Phase 3.12 is complete when:

1. **Functionality:**
   - [x] Error severity classification implemented
   - [ ] Inline transcript errors display correctly
   - [ ] Status bar transient errors work with auto-dismiss
   - [ ] Critical error modal displays and is dismissible
   - [ ] Error recovery mechanisms functional
   - [ ] All user actions work (dismiss, copy, navigate)

2. **Quality:**
   - [ ] All tests passing (≥90% coverage)
   - [ ] All quality gates passed
   - [ ] No crashes on any error scenario
   - [ ] Performance targets met

3. **Documentation:**
   - [x] FRD created (this document)
   - [ ] Godoc complete
   - [ ] Error handling patterns documented
   - [ ] ROADMAP updated

---

## Future Enhancements

- Error history view (Ctrl+E to view all errors)
- Export errors to file
- Custom error handlers per error code
- Error rate limiting (prevent spam)
- Error aggregation (group similar errors)
- Persistent error log across sessions
- Error reporting to telemetry (opt-in)

---

## References

- [specs/ui-modules/spec.md](../ui-modules/spec.md) - TUI Error Handling section
- [specs/ui-modules/ROADMAP.md](../ui-modules/ROADMAP.md) - Phase 3.12
- [internal/core/event.go](../../internal/core/event.go) - Event types
- [internal/core/error.go](../../internal/core/error.go) - Error types
- [AGENTS.md](../../AGENTS.md) - Implementation workflow
