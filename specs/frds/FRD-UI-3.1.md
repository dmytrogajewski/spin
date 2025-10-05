# FRD-UI-3.1: Bubble Tea Application Setup

**Feature:** Interactive TUI Foundation
**Phase:** 3.1
**Priority:** High
**Status:** In Progress
**Created:** 2025-10-05

---

## 1. Overview

Implement the foundational Bubble Tea application infrastructure for Spin's interactive TUI mode. This establishes the core UI framework, state management, and rendering pipeline that all subsequent TUI features will build upon.

**Goals:**
- Create a working Bubble Tea application structure
- Implement The Elm Architecture pattern (Model-Update-View)
- Set up state machine for TUI modes
- Enable basic TUI launch via `spin` or `spin tui`
- Handle terminal resize events
- Achieve 60 FPS rendering performance

---

## 2. Technical Design

### 2.1 Architecture

**The Elm Architecture Pattern:**

```
┌─────────────────────────────────────────┐
│         User Input (tea.Msg)            │
└────────────────┬────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────┐
│    Update(msg) → (Model, tea.Cmd)       │
│  - Process message                      │
│  - Update model state                   │
│  - Return commands for side effects     │
└────────────────┬────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────┐
│        View() → string                  │
│  - Render current model to string       │
│  - No side effects                      │
└────────────────┬────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────┐
│         Terminal Display                │
└─────────────────────────────────────────┘
```

### 2.2 Package Structure

```
cmd/spin/
├── tui.go              # Cobra command for TUI mode

internal/tui/
├── app.go              # Main application model
├── state.go            # State machine definitions
├── init.go             # Initialization logic
├── update.go           # Message routing and state updates
├── view.go             # Rendering pipeline
└── app_test.go         # Comprehensive tests
```

### 2.3 State Machine

**Application States:**

```go
type AppState int

const (
    StateIdle AppState = iota          // Waiting for user input
    StateWaitingResponse               // AI is generating response
    StateToolApproval                  // Waiting for tool approval
    StateFilePickerOpen                // @ file search active
    StateBacktrackMode                 // Esc-Esc mode
    StateExiting                       // Shutting down
)
```

**State Transitions:**

```
     ┌─────────────┐
     │    Idle     │ ◄──────────────────────┐
     └─────────────┘                        │
            │                               │
     (User sends message)            (Turn complete)
            │                               │
            ▼                               │
  ┌──────────────────┐                      │
  │ WaitingResponse  │──────────────────────┘
  └──────────────────┘
            │
     (Tool proposed)
            │
            ▼
  ┌──────────────────┐
  │  ToolApproval    │
  └──────────────────┘
            │
     (Approved/Denied)
            │
            └──────► (back to WaitingResponse)
```

### 2.4 Core Model

```go
// Model represents the entire TUI application state
type Model struct {
    // State management
    state       AppState
    err         error

    // UI components (will be added in later phases)
    width       int
    height      int

    // Core communication (will be integrated in Phase 3.11)
    // coreChan    chan core.Event

    // Exit flag
    quitting    bool
}
```

### 2.5 Message Types

**Built-in Bubble Tea Messages:**
- `tea.KeyMsg` - Keyboard input
- `tea.WindowSizeMsg` - Terminal resize
- `tea.MouseMsg` - Mouse events (future)

**Custom Messages (for Phase 3.11):**
```go
// CoreEventMsg wraps events from internal/core
type CoreEventMsg struct {
    Event core.Event
}

// ErrorMsg represents an error condition
type ErrorMsg struct {
    Err error
}
```

---

## 3. Implementation Details

### 3.1 Cobra Command (`cmd/spin/tui.go`)

```go
package main

import (
    "fmt"
    "os"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/spf13/cobra"

    "github.com/dmytrogajewski/spin/internal/tui"
)

// tuiCmd represents the TUI mode command
var tuiCmd = &cobra.Command{
    Use:   "tui",
    Short: "Start interactive terminal UI (default)",
    Long:  `Launch Spin in interactive TUI mode with full Bubble Tea interface.`,
    RunE:  runTUI,
}

func init() {
    rootCmd.AddCommand(tuiCmd)

    // TUI-specific flags (future)
    // tuiCmd.Flags().Bool("no-color", false, "Disable colors")
}

func runTUI(cmd *cobra.Command, args []string) error {
    // Create TUI model
    m := tui.NewModel()

    // Start Bubble Tea program
    p := tea.NewProgram(
        m,
        tea.WithAltScreen(),       // Use alternate screen buffer
        tea.WithMouseCellMotion(), // Enable mouse support
    )

    // Run the program
    finalModel, err := p.Run()
    if err != nil {
        return fmt.Errorf("TUI error: %w", err)
    }

    // Check for errors in final model
    if m, ok := finalModel.(tui.Model); ok {
        if m.Err() != nil {
            return m.Err()
        }
    }

    return nil
}
```

### 3.2 Application Model (`internal/tui/app.go`)

```go
package tui

import (
    tea "github.com/charmbracelet/bubbletea"
)

// Model represents the TUI application state
type Model struct {
    state    AppState
    err      error
    width    int
    height   int
    quitting bool
}

// NewModel creates a new TUI model
func NewModel() Model {
    return Model{
        state: StateIdle,
    }
}

// Err returns any error that occurred
func (m Model) Err() error {
    return m.err
}

// State returns the current application state
func (m Model) State() AppState {
    return m.state
}
```

### 3.3 State Machine (`internal/tui/state.go`)

```go
package tui

// AppState represents the different states of the TUI
type AppState int

const (
    StateIdle AppState = iota
    StateWaitingResponse
    StateToolApproval
    StateFilePickerOpen
    StateBacktrackMode
    StateExiting
)

// String returns the string representation of the state
func (s AppState) String() string {
    switch s {
    case StateIdle:
        return "idle"
    case StateWaitingResponse:
        return "waiting_response"
    case StateToolApproval:
        return "tool_approval"
    case StateFilePickerOpen:
        return "file_picker_open"
    case StateBacktrackMode:
        return "backtrack_mode"
    case StateExiting:
        return "exiting"
    default:
        return "unknown"
    }
}

// CanTransitionTo checks if transition to new state is valid
func (s AppState) CanTransitionTo(new AppState) bool {
    switch s {
    case StateIdle:
        return new == StateWaitingResponse ||
               new == StateFilePickerOpen ||
               new == StateBacktrackMode ||
               new == StateExiting

    case StateWaitingResponse:
        return new == StateIdle ||
               new == StateToolApproval ||
               new == StateExiting

    case StateToolApproval:
        return new == StateWaitingResponse ||
               new == StateExiting

    case StateFilePickerOpen:
        return new == StateIdle ||
               new == StateExiting

    case StateBacktrackMode:
        return new == StateIdle ||
               new == StateExiting

    case StateExiting:
        return false // Terminal state

    default:
        return false
    }
}
```

### 3.4 Initialization (`internal/tui/init.go`)

```go
package tui

import tea "github.com/charmbracelet/bubbletea"

// Init is the first function called when the program starts
func (m Model) Init() tea.Cmd {
    // Return any initial commands
    // For now, just wait for window size
    return nil
}
```

### 3.5 Update Logic (`internal/tui/update.go`)

```go
package tui

import tea "github.com/charmbracelet/bubbletea"

// Update handles incoming messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {

    case tea.KeyMsg:
        return m.handleKeyPress(msg)

    case tea.WindowSizeMsg:
        return m.handleResize(msg)

    case ErrorMsg:
        m.err = msg.Err
        return m, tea.Quit
    }

    return m, nil
}

// handleKeyPress processes keyboard input
func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    // Handle global shortcuts
    switch msg.String() {
    case "ctrl+c", "ctrl+d":
        m.state = StateExiting
        m.quitting = true
        return m, tea.Quit
    }

    return m, nil
}

// handleResize processes terminal resize events
func (m Model) handleResize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
    m.width = msg.Width
    m.height = msg.Height
    return m, nil
}
```

### 3.6 View Rendering (`internal/tui/view.go`)

```go
package tui

import "fmt"

// View renders the UI
func (m Model) View() string {
    if m.quitting {
        return ""
    }

    // Placeholder view
    return fmt.Sprintf(
        "Spin TUI (State: %s)\n"+
        "Terminal: %dx%d\n\n"+
        "Press Ctrl+C or Ctrl+D to exit\n",
        m.state.String(),
        m.width,
        m.height,
    )
}
```

---

## 4. Testing Strategy

### 4.1 State Machine Tests

```go
func TestStateTransitions(t *testing.T) {
    tests := []struct {
        name      string
        from      AppState
        to        AppState
        wantValid bool
    }{
        {"idle to waiting", StateIdle, StateWaitingResponse, true},
        {"idle to exiting", StateIdle, StateExiting, true},
        {"waiting to idle", StateWaitingResponse, StateIdle, true},
        {"waiting to approval", StateWaitingResponse, StateToolApproval, true},
        {"approval to idle", StateToolApproval, StateIdle, false}, // Invalid!
        {"exiting to idle", StateExiting, StateIdle, false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := tt.from.CanTransitionTo(tt.to)
            assert.Equal(t, tt.wantValid, got)
        })
    }
}
```

### 4.2 Update Function Tests

```go
func TestUpdate_KeyPress(t *testing.T) {
    m := NewModel()

    // Test Ctrl+C
    newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
    assert.Equal(t, StateExiting, newModel.(Model).state)
    assert.True(t, newModel.(Model).quitting)
    assert.NotNil(t, cmd) // Should return quit command
}

func TestUpdate_WindowResize(t *testing.T) {
    m := NewModel()

    newModel, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
    assert.Equal(t, 120, newModel.(Model).width)
    assert.Equal(t, 40, newModel.(Model).height)
}
```

### 4.3 Integration Tests

```go
func TestTUI_Launch(t *testing.T) {
    m := NewModel()

    // Verify initial state
    assert.Equal(t, StateIdle, m.State())
    assert.Nil(t, m.Err())

    // Initialize
    cmd := m.Init()
    assert.Nil(t, cmd) // No initial commands yet
}
```

---

## 5. Performance Requirements

### 5.1 Rendering Performance

**Target: 60 FPS (16ms frame time)**

- View() function must complete in <16ms
- Minimal string allocations in hot path
- Use strings.Builder for efficient concatenation

**Measurement:**
```go
func BenchmarkView(b *testing.B) {
    m := NewModel()
    m.width = 120
    m.height = 40

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = m.View()
    }
}
```

### 5.2 Memory Usage

**Target: <5MB for basic TUI structure**

- Avoid unnecessary allocations in Update()
- Reuse buffers where possible
- Profile with `go test -memprofile=mem.prof`

---

## 6. Dependencies

### 6.1 Required Packages

```go
// go.mod additions
require (
    github.com/charmbracelet/bubbletea v0.25.0
    github.com/stretchr/testify v1.9.0 // for tests
)
```

### 6.2 Bubble Tea Installation

```bash
go get github.com/charmbracelet/bubbletea@latest
```

---

## 7. Integration Points

### 7.1 With Core (Phase 3.11)

**Future integration:**
```go
// Model will include:
type Model struct {
    // ...
    coreChan    chan core.Event

    // Core communication helpers
    agent       *core.Agent
    conv        *core.Conversation
}

// waitForCoreEvent will be a Bubble Tea command
func waitForCoreEvent(ch chan core.Event) tea.Cmd {
    return func() tea.Msg {
        return CoreEventMsg{Event: <-ch}
    }
}
```

### 7.2 With Config (Already Available)

```go
// Load configuration in runTUI
cfg := config.NewLoader()
cfg.Load("")

// Pass to model if needed
m := tui.NewModel(tui.WithConfig(cfg))
```

---

## 8. Quality Checklist

### 8.1 Definition of Ready (DoR)

- [x] Bubble Tea framework studied
- [x] Dependencies identified (bubbletea)
- [x] TUI state machine designed
- [x] Architecture documented

### 8.2 Definition of Done (DoD)

- [ ] Tests for state transitions (≥85% coverage)
- [ ] Tests for message routing
- [ ] Basic TUI launches successfully via `spin` or `spin tui`
- [ ] Window resize works correctly
- [ ] Render latency <16ms (60 FPS)
- [ ] All tests passing with race detector
- [ ] Linter clean (make lint)
- [ ] Complexity ≤15 for all functions
- [ ] Godoc on all exports
- [ ] ROADMAP updated

---

## 9. Risks and Mitigations

### 9.1 Risks

1. **Performance degradation** - Complex views may slow rendering
   - **Mitigation:** Benchmark early, optimize hot paths, use viewport for large content

2. **State machine complexity** - Invalid transitions could cause bugs
   - **Mitigation:** Comprehensive state transition tests, validation in CanTransitionTo()

3. **Terminal compatibility** - Different terminals may behave differently
   - **Mitigation:** Test on major terminals (xterm, iTerm2, Windows Terminal)

### 9.2 Fallback Plan

If Bubble Tea proves problematic:
- Fall back to simpler TUI library (tview, termui)
- Reduce to basic line-based interface
- Focus on `spin exec` mode as primary interface

---

## 10. Success Criteria

Phase 3.1 is complete when:

1. ✅ `spin` or `spin tui` launches a Bubble Tea application
2. ✅ Application displays basic placeholder UI
3. ✅ Ctrl+C and Ctrl+D exit gracefully
4. ✅ Terminal resize updates dimensions correctly
5. ✅ State machine enforces valid transitions
6. ✅ All tests passing with ≥85% coverage
7. ✅ Performance targets met (<16ms render)
8. ✅ No lint errors, complexity ≤15

---

## 11. Future Enhancements (Out of Scope)

These will be implemented in later phases:

- Chat interface components (Phase 3.2)
- Input widget (Phase 3.3)
- File picker (Phase 3.4)
- Tool approval UI (Phase 3.5)
- Status bar (Phase 3.6)
- Transcript view (Phase 3.7)
- Backtrack mode (Phase 3.8)
- Keyboard shortcuts (Phase 3.9)
- Styling (Phase 3.10)
- Core integration (Phase 3.11)
- Error handling (Phase 3.12)

---

## 12. References

- [Bubble Tea Documentation](https://github.com/charmbracelet/bubbletea)
- [The Elm Architecture](https://guide.elm-lang.org/architecture/)
- [specs/ui-modules/spec.md](../ui-modules/spec.md)
- [specs/ui-modules/ROADMAP.md](../ui-modules/ROADMAP.md)
- [AGENTS.md](../../AGENTS.md) - Quality standards

---

**Document Version:** 1.0
**Last Updated:** 2025-10-05
**Author:** Spin Development Team
