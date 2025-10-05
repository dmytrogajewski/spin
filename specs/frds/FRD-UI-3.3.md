# FRD-UI-3.3: Input Widget & Multi-line Support

**Feature:** Multi-line Text Input with Enhanced Features
**Phase:** 3.3
**Priority:** High
**Status:** In Progress
**Created:** 2025-10-05

---

## 1. Overview

Implement a sophisticated input widget for Spin's TUI that supports multi-line text entry, large paste operations, input history, and @ file picker triggers. The input widget must be performant, user-friendly, and integrate seamlessly with the chat interface.

**Goals:**
- Multi-line text input with word wrapping
- Handle large paste operations without freezing
- Input history navigation (up/down arrows)
- @ trigger detection for file picker
- Submit with Enter (Shift+Enter for newline)
- Visual feedback for multi-line mode
- <5ms input latency
- Memory efficient (<5MB for history)

---

## 2. Technical Design

### 2.1 Architecture

```
┌─────────────────────────────────────────┐
│       Input Component (input.go)        │
│                                         │
│  ┌───────────────────────────────────┐ │
│  │   Textarea (Bubble Tea Component) │ │
│  │                                   │ │
│  │  - Multi-line editing             │ │
│  │  - Word wrapping                  │ │
│  │  - Cursor management              │ │
│  │  - Clipboard integration          │ │
│  └───────────────────────────────────┘ │
│                                         │
│  ┌───────────────────────────────────┐ │
│  │      History Manager              │ │
│  │                                   │ │
│  │  - Ring buffer (max 100 items)    │ │
│  │  - Up/Down navigation             │ │
│  │  - Position tracking              │ │
│  └───────────────────────────────────┘ │
│                                         │
│  ┌───────────────────────────────────┐ │
│  │     Trigger Detector              │ │
│  │                                   │ │
│  │  - @ character detection          │ │
│  │  - Position tracking              │ │
│  │  - Trigger callback               │ │
│  └───────────────────────────────────┘ │
└─────────────────────────────────────────┘
```

### 2.2 Package Structure

```
internal/tui/ui/
├── input.go           # Main input component
├── input_test.go      # Input tests
├── history.go         # Input history manager
├── history_test.go    # History tests
```

### 2.3 Input Component

```go
// Input represents the text input component.
type Input struct {
    textarea    textarea.Model
    history     *History
    width       int
    height      int

    // State
    multiline   bool   // Multi-line mode enabled
    focused     bool   // Component has focus

    // Trigger detection
    triggerPos  int    // Position of @ trigger (-1 if none)
    onTrigger   func() // Callback when @ is pressed
}

// NewInput creates a new input component.
func NewInput(width, height int) Input

// SetSize updates the input dimensions.
func (i *Input) SetSize(width, height int)

// Focus gives the input focus.
func (i *Input) Focus() tea.Cmd

// Blur removes focus from the input.
func (i *Input) Blur()

// SetValue sets the input value.
func (i *Input) SetValue(value string)

// GetValue returns the current input value.
func (i Input) GetValue() string

// Clear clears the input.
func (i *Input) Clear()

// SetTriggerCallback sets the @ trigger callback.
func (i *Input) SetTriggerCallback(callback func())

// Update handles Bubble Tea messages.
func (i Input) Update(msg tea.Msg) (Input, tea.Cmd)

// View renders the input.
func (i Input) View() string
```

### 2.4 History Manager

```go
// History manages input history with a ring buffer.
type History struct {
    items    []string // Ring buffer of history items
    maxSize  int      // Maximum history size
    position int      // Current position in history
    tempBuf  string   // Temporary buffer for current input
}

const DefaultMaxHistory = 100

// NewHistory creates a new history manager.
func NewHistory(maxSize int) *History

// Add adds an item to history.
func (h *History) Add(item string)

// Previous returns the previous history item.
func (h *History) Previous() (string, bool)

// Next returns the next history item.
func (h *History) Next() (string, bool)

// Reset resets the history position.
func (h *History) Reset()

// SetTempBuffer stores the current input before navigation.
func (h *History) SetTempBuffer(value string)
```

---

## 3. Implementation Details

### 3.1 Dependencies

**Required Packages:**
```go
// Bubble Tea ecosystem
"github.com/charmbracelet/bubbles/textarea"
"github.com/charmbracelet/bubbles/key"
"github.com/charmbracelet/lipgloss"
```

**Note:** `bubbles/textarea` is already in go.mod via `github.com/charmbracelet/bubbles v0.21.0`

### 3.2 Input Component Implementation

```go
package ui

import (
    "strings"

    "github.com/charmbracelet/bubbles/key"
    "github.com/charmbracelet/bubbles/textarea"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
)

// Input represents the text input component.
type Input struct {
    textarea   textarea.Model
    history    *History
    width      int
    height     int
    multiline  bool
    focused    bool
    triggerPos int
    onTrigger  func()
}

// NewInput creates a new input component.
func NewInput(width, height int) Input {
    ta := textarea.New()
    ta.Placeholder = "Type your message (Enter to send, Shift+Enter for newline)..."
    ta.ShowLineNumbers = false
    ta.SetWidth(width)
    ta.SetHeight(height)

    // Key bindings
    ta.KeyMap.InsertNewline.SetEnabled(true)

    return Input{
        textarea:   ta,
        history:    NewHistory(DefaultMaxHistory),
        width:      width,
        height:     height,
        multiline:  true,
        focused:    false,
        triggerPos: -1,
        onTrigger:  nil,
    }
}

// SetSize updates the input dimensions.
func (i *Input) SetSize(width, height int) {
    i.width = width
    i.height = height
    i.textarea.SetWidth(width)
    i.textarea.SetHeight(height)
}

// Focus gives the input focus.
func (i *Input) Focus() tea.Cmd {
    i.focused = true
    return i.textarea.Focus()
}

// Blur removes focus from the input.
func (i *Input) Blur() {
    i.focused = false
    i.textarea.Blur()
}

// SetValue sets the input value.
func (i *Input) SetValue(value string) {
    i.textarea.SetValue(value)
    i.detectTrigger()
}

// GetValue returns the current input value.
func (i Input) GetValue() string {
    return i.textarea.Value()
}

// Clear clears the input.
func (i *Input) Clear() {
    i.textarea.Reset()
    i.triggerPos = -1
}

// SetTriggerCallback sets the @ trigger callback.
func (i *Input) SetTriggerCallback(callback func()) {
    i.onTrigger = callback
}

// Update handles Bubble Tea messages.
func (i Input) Update(msg tea.Msg) (Input, tea.Cmd) {
    var cmd tea.Cmd

    switch msg := msg.(type) {
    case tea.KeyMsg:
        // Handle special keys before textarea
        switch msg.String() {
        case "enter":
            // Submit if not Shift+Enter
            if !msg.Alt && !msg.Shift && !msg.Ctrl {
                // Will be handled by parent (send message)
                return i, nil
            }
            // Otherwise, insert newline (let textarea handle)

        case "up":
            // Navigate history if at beginning and single line
            if i.textarea.Line() == 0 && i.textarea.Position() == 0 {
                if prev, ok := i.history.Previous(); ok {
                    i.textarea.SetValue(prev)
                    // Move cursor to end
                    i.textarea.CursorEnd()
                    return i, nil
                }
            }

        case "down":
            // Navigate history
            if i.textarea.Line() == i.textarea.LineCount()-1 {
                if next, ok := i.history.Next(); ok {
                    i.textarea.SetValue(next)
                    i.textarea.CursorEnd()
                    return i, nil
                }
            }
        }
    }

    // Update textarea
    i.textarea, cmd = i.textarea.Update(msg)

    // Detect @ trigger after update
    i.detectTrigger()

    return i, cmd
}

// View renders the input.
func (i Input) View() string {
    style := lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(lipgloss.Color("240"))

    if i.focused {
        style = style.BorderForeground(lipgloss.Color("12")) // Blue when focused
    }

    // Show hint if @ trigger is active
    view := i.textarea.View()
    if i.triggerPos >= 0 {
        hint := lipgloss.NewStyle().
            Foreground(lipgloss.Color("240")).
            Render("💡 Press Tab to open file picker")
        view += "\n" + hint
    }

    return style.Render(view)
}

// detectTrigger detects @ character for file picker.
func (i *Input) detectTrigger() {
    value := i.textarea.Value()
    cursorPos := i.textarea.Position()

    // Look for @ before cursor
    if cursorPos > 0 && cursorPos <= len(value) {
        beforeCursor := value[:cursorPos]
        // Check if last character is @
        if strings.HasSuffix(beforeCursor, "@") {
            // Check if it's at word boundary (start or after space)
            if len(beforeCursor) == 1 || beforeCursor[len(beforeCursor)-2] == ' ' {
                i.triggerPos = cursorPos - 1
                // Trigger callback
                if i.onTrigger != nil {
                    i.onTrigger()
                }
                return
            }
        }
    }

    // No trigger found
    i.triggerPos = -1
}

// AddToHistory adds the current input to history.
func (i *Input) AddToHistory() {
    value := i.GetValue()
    if strings.TrimSpace(value) != "" {
        i.history.Add(value)
        i.history.Reset()
    }
}
```

### 3.3 History Manager Implementation

```go
package ui

// History manages input history with a ring buffer.
type History struct {
    items    []string
    maxSize  int
    position int      // -1 = at current input, 0+ = in history
    tempBuf  string   // Temporary buffer for current input
}

// DefaultMaxHistory is the default maximum history size.
const DefaultMaxHistory = 100

// NewHistory creates a new history manager.
func NewHistory(maxSize int) *History {
    if maxSize <= 0 {
        maxSize = DefaultMaxHistory
    }

    return &History{
        items:    make([]string, 0, maxSize),
        maxSize:  maxSize,
        position: -1,
        tempBuf:  "",
    }
}

// Add adds an item to history (deduplicated).
func (h *History) Add(item string) {
    // Don't add empty strings
    if item == "" {
        return
    }

    // Don't add duplicates (remove old occurrence)
    for i, existing := range h.items {
        if existing == item {
            h.items = append(h.items[:i], h.items[i+1:]...)
            break
        }
    }

    // Add to end
    h.items = append(h.items, item)

    // Trim if exceeds max size
    if len(h.items) > h.maxSize {
        h.items = h.items[len(h.items)-h.maxSize:]
    }

    h.position = -1
}

// Previous returns the previous history item.
func (h *History) Previous() (string, bool) {
    if len(h.items) == 0 {
        return "", false
    }

    // First time: move from current to most recent
    if h.position == -1 {
        h.position = len(h.items) - 1
        return h.items[h.position], true
    }

    // Already in history: move backward
    if h.position > 0 {
        h.position--
        return h.items[h.position], true
    }

    // At oldest item
    return h.items[h.position], false
}

// Next returns the next history item.
func (h *History) Next() (string, bool) {
    if len(h.items) == 0 || h.position == -1 {
        return "", false
    }

    // Move forward
    if h.position < len(h.items)-1 {
        h.position++
        return h.items[h.position], true
    }

    // Back to current input (temp buffer)
    h.position = -1
    return h.tempBuf, true
}

// Reset resets the history position.
func (h *History) Reset() {
    h.position = -1
    h.tempBuf = ""
}

// SetTempBuffer stores the current input before navigation.
func (h *History) SetTempBuffer(value string) {
    h.tempBuf = value
}

// GetAll returns all history items.
func (h *History) GetAll() []string {
    return h.items
}

// Clear clears the history.
func (h *History) Clear() {
    h.items = make([]string, 0, h.maxSize)
    h.position = -1
    h.tempBuf = ""
}
```

### 3.4 Integration with TUI Model

```go
// In internal/tui/app.go

type Model struct {
    // ... existing fields ...
    input Input  // Add input component
}

func NewModel() Model {
    return Model{
        state:    StateIdle,
        chat:     ui.NewChat(0, 0),
        input:    ui.NewInput(0, 3), // 3 lines height
    }
}

func (m Model) Init() tea.Cmd {
    // Focus input initially
    return m.input.Focus()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmds []tea.Cmd

    switch msg := msg.(type) {
    case tea.KeyMsg:
        // Handle Enter to send message
        if msg.String() == "enter" && m.input.GetValue() != "" {
            // Get message
            message := m.input.GetValue()

            // Add to history
            m.input.AddToHistory()

            // Clear input
            m.input.Clear()

            // Send message (placeholder for Phase 3.11)
            // Will integrate with core module later
            m.chat.AddMessage(ui.Message{
                Role:    ui.RoleUser,
                Content: message,
            })

            return m, nil
        }

    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height

        // Size components
        chatHeight := m.height - 5  // Leave room for input + status
        m.chat.SetSize(m.width, chatHeight)
        m.input.SetSize(m.width-2, 3) // 3 lines, with padding

        return m, nil
    }

    // Update input
    var cmd tea.Cmd
    m.input, cmd = m.input.Update(msg)
    cmds = append(cmds, cmd)

    // Update chat
    m.chat, cmd = m.chat.Update(msg)
    cmds = append(cmds, cmd)

    return m, tea.Batch(cmds...)
}

func (m Model) View() string {
    // ... existing rendering ...

    // Render input
    inputView := m.input.View()

    return lipgloss.JoinVertical(
        lipgloss.Top,
        chatView,
        inputView,
        statusView,
    )
}
```

---

## 4. Testing Strategy

### 4.1 Unit Tests

```go
func TestInput_SetValue(t *testing.T) {
    i := NewInput(80, 3)

    i.SetValue("Hello world")

    assert.Equal(t, "Hello world", i.GetValue())
}

func TestInput_Clear(t *testing.T) {
    i := NewInput(80, 3)
    i.SetValue("Test")

    i.Clear()

    assert.Equal(t, "", i.GetValue())
}

func TestInput_TriggerDetection(t *testing.T) {
    i := NewInput(80, 3)
    triggered := false
    i.SetTriggerCallback(func() { triggered = true })

    // Type @ at word boundary
    i.SetValue("@")

    assert.True(t, triggered)
    assert.Equal(t, 0, i.triggerPos)
}

func TestInput_TriggerDetection_NotInMiddleOfWord(t *testing.T) {
    i := NewInput(80, 3)
    triggered := false
    i.SetTriggerCallback(func() { triggered = true })

    // Type @ in middle of word
    i.SetValue("test@")

    assert.False(t, triggered)
}

func TestHistory_Add(t *testing.T) {
    h := NewHistory(10)

    h.Add("first")
    h.Add("second")

    assert.Len(t, h.items, 2)
    assert.Equal(t, "second", h.items[1])
}

func TestHistory_Previous(t *testing.T) {
    h := NewHistory(10)
    h.Add("first")
    h.Add("second")

    prev, ok := h.Previous()

    assert.True(t, ok)
    assert.Equal(t, "second", prev)
}

func TestHistory_Navigation(t *testing.T) {
    h := NewHistory(10)
    h.Add("first")
    h.Add("second")
    h.Add("third")

    // Navigate backward
    prev, ok := h.Previous()
    assert.True(t, ok)
    assert.Equal(t, "third", prev)

    prev, ok = h.Previous()
    assert.True(t, ok)
    assert.Equal(t, "second", prev)

    // Navigate forward
    next, ok := h.Next()
    assert.True(t, ok)
    assert.Equal(t, "third", next)
}

func TestHistory_MaxSize(t *testing.T) {
    h := NewHistory(3)

    h.Add("first")
    h.Add("second")
    h.Add("third")
    h.Add("fourth")  // Should evict "first"

    assert.Len(t, h.items, 3)
    assert.Equal(t, "second", h.items[0])
}

func TestHistory_Deduplication(t *testing.T) {
    h := NewHistory(10)

    h.Add("duplicate")
    h.Add("other")
    h.Add("duplicate")  // Should remove old "duplicate"

    assert.Len(t, h.items, 2)
    assert.Equal(t, "duplicate", h.items[1]) // At end
}
```

### 4.2 Integration Tests

```go
func TestInput_Integration_SendMessage(t *testing.T) {
    m := NewModel()

    // Type message
    m.input.SetValue("Hello, world!")

    // Press Enter
    newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
    m = newModel.(Model)

    // Input should be cleared
    assert.Equal(t, "", m.input.GetValue())

    // Message should be added to chat
    messages := m.chat.GetMessages()
    assert.Len(t, messages, 1)
    assert.Equal(t, "Hello, world!", messages[0].Content)
}

func TestInput_Integration_History(t *testing.T) {
    m := NewModel()

    // Send two messages
    m.input.SetValue("First message")
    m.Update(tea.KeyMsg{Type: tea.KeyEnter})

    m.input.SetValue("Second message")
    m.Update(tea.KeyMsg{Type: tea.KeyEnter})

    // Navigate history with Up arrow
    m.Update(tea.KeyMsg{Type: tea.KeyUp})
    assert.Equal(t, "Second message", m.input.GetValue())

    m.Update(tea.KeyMsg{Type: tea.KeyUp})
    assert.Equal(t, "First message", m.input.GetValue())
}
```

### 4.3 Performance Tests

```go
func BenchmarkInput_LargePaste(b *testing.B) {
    i := NewInput(80, 3)

    // Simulate 10KB paste
    largeText := strings.Repeat("Hello world! ", 1000)

    b.ResetTimer()
    for n := 0; n < b.N; n++ {
        i.SetValue(largeText)
    }
}

func BenchmarkHistory_Add(b *testing.B) {
    h := NewHistory(100)

    b.ResetTimer()
    for n := 0; n < b.N; n++ {
        h.Add(fmt.Sprintf("message %d", n))
    }
}
```

---

## 5. Performance Requirements

### 5.1 Input Latency

**Targets:**
- Key press response: <5ms
- Large paste (10KB): <50ms
- History navigation: <10ms
- Rendering: <16ms (60 FPS)

**Optimizations:**
- Use textarea's built-in optimizations
- Debounce trigger detection
- Efficient history ring buffer
- Minimal allocations in Update()

### 5.2 Memory Management

```go
// Limit history to prevent unbounded growth
const DefaultMaxHistory = 100

// Each history item ~100 bytes average
// Max memory: 100 * 100 bytes = ~10KB
// Well within <5MB target
```

---

## 6. Integration Points

### 6.1 With File Picker (Phase 3.4)

```go
// In internal/tui/app.go

func NewModel() Model {
    m := Model{
        // ...
        input: ui.NewInput(0, 3),
    }

    // Set @ trigger callback
    m.input.SetTriggerCallback(func() {
        // Transition to file picker state
        m.state = StateFilePickerOpen
        // Open file picker (Phase 3.4)
    })

    return m
}
```

### 6.2 With Core Module (Phase 3.11)

```go
// Send message to core agent
if msg.String() == "enter" {
    message := m.input.GetValue()
    m.input.AddToHistory()
    m.input.Clear()

    // Send to core (Phase 3.11)
    return m, sendMessageToCore(message)
}
```

---

## 7. Quality Checklist

### 7.1 Definition of Ready (DoR)

- [x] Input requirements reviewed (multi-line, paste, history)
- [x] Keyboard shortcuts defined (Enter, Shift+Enter, Up/Down)
- [x] @ file picker trigger understood
- [x] Dependencies identified (bubbles/textarea)

### 7.2 Definition of Done (DoD)

- [ ] Tests for input handling (≥90% coverage)
- [ ] Tests for history navigation
- [ ] Multi-line works correctly
- [ ] Large paste doesn't freeze UI (<50ms)
- [ ] @ triggers detected correctly
- [ ] Input history works (Up/Down)
- [ ] Input latency <5ms
- [ ] All tests passing with race detector
- [ ] Linter clean (make lint)
- [ ] Complexity ≤15 for all functions
- [ ] Godoc on all exports
- [ ] ROADMAP updated

---

## 8. Risks and Mitigations

### 8.1 Risks

1. **Large paste freezing UI** - 10MB paste could block rendering
   - **Mitigation:** Async paste handling, size limits, truncation warnings

2. **History memory growth** - Unbounded history could consume memory
   - **Mitigation:** Ring buffer with max size (100 items), ~10KB total

3. **@ trigger false positives** - Email addresses contain @
   - **Mitigation:** Word boundary detection, only trigger at start or after space

4. **Textarea performance** - bubbles/textarea may have limitations
   - **Mitigation:** Use built-in optimizations, test with large inputs, fallback to simpler input

---

## 9. Success Criteria

Phase 3.3 is complete when:

1. ✅ Input widget accepts multi-line text
2. ✅ Enter sends message, Shift+Enter adds newline
3. ✅ Up/Down arrows navigate history
4. ✅ @ character triggers callback for file picker
5. ✅ Large paste (10KB) completes in <50ms
6. ✅ Input latency <5ms for key presses
7. ✅ History limited to 100 items (~10KB memory)
8. ✅ All tests passing with ≥90% coverage
9. ✅ Visual feedback shows focus state
10. ✅ Integration with TUI model complete

---

## 10. Future Enhancements (Out of Scope)

- Autocomplete suggestions (Phase 3.4 - File Picker)
- Rich text formatting
- Inline image preview
- Voice input
- Multi-cursor editing
- Custom key bindings
- Input macros/snippets
- Tab completion for commands

---

## 11. References

- [Bubble Tea Textarea](https://github.com/charmbracelet/bubbles/tree/master/textarea)
- [Lipgloss Documentation](https://github.com/charmbracelet/lipgloss)
- [specs/ui-modules/spec.md](../ui-modules/spec.md)
- [specs/ui-modules/ROADMAP.md](../ui-modules/ROADMAP.md)
- [FRD-UI-3.1.md](FRD-UI-3.1.md) - TUI Foundation
- [FRD-UI-3.2.md](FRD-UI-3.2.md) - Chat Interface
- [AGENTS.md](../../AGENTS.md) - Quality standards

---

**Document Version:** 1.0
**Last Updated:** 2025-10-05
**Author:** Spin Development Team
