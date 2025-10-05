package ui

import (
	"strings"

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
	ta.Focus() // Focus by default

	// Key bindings
	ta.KeyMap.InsertNewline.SetEnabled(true)

	return Input{
		textarea:   ta,
		history:    NewHistory(DefaultMaxHistory),
		width:      width,
		height:     height,
		multiline:  true,
		focused:    true, // Start focused
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
			// Submit (Enter key without modifiers)
			// Shift+Enter is handled by textarea for newline
			// Will be handled by parent (send message)
			return i, nil

		case "up":
			// Navigate history if at beginning and single line
			lineInfo := i.textarea.LineInfo()
			if i.textarea.Line() == 0 && lineInfo.ColumnOffset == 0 {
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

	// Calculate cursor position from line info
	lineInfo := i.textarea.LineInfo()
	currentLine := i.textarea.Line()

	// Count characters up to current line
	lines := strings.Split(value, "\n")
	cursorPos := 0
	for idx := 0; idx < currentLine && idx < len(lines); idx++ {
		cursorPos += len(lines[idx]) + 1 // +1 for newline
	}
	cursorPos += lineInfo.CharOffset

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
