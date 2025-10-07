package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestNewInput(t *testing.T) {
	i := NewInput(80, 3)

	assert.NotNil(t, i.textarea)
	assert.NotNil(t, i.history)
	assert.Equal(t, 80, i.width)
	assert.Equal(t, 3, i.height)
	assert.True(t, i.focused)
	assert.True(t, i.textarea.Focused())
	assert.Equal(t, -1, i.triggerPos)
}

func TestInput_SetSize(t *testing.T) {
	i := NewInput(80, 3)

	i.SetSize(100, 5)

	assert.Equal(t, 100, i.width)
	assert.Equal(t, 5, i.height)
	// Textarea width/height are set (internal padding may differ)
	assert.True(t, i.textarea.Width() > 0)
	assert.Equal(t, 5, i.textarea.Height())
}

func TestInput_Focus(t *testing.T) {
	i := NewInput(80, 3)

	i.Focus()

	assert.True(t, i.focused)
	assert.True(t, i.textarea.Focused())
}

func TestInput_Blur(t *testing.T) {
	i := NewInput(80, 3)
	i.Focus()

	i.Blur()

	assert.False(t, i.focused)
	assert.False(t, i.textarea.Focused())
}

func TestInput_SetValue(t *testing.T) {
	i := NewInput(80, 3)

	i.SetValue("Hello world")

	assert.Equal(t, "Hello world", i.GetValue())
}

func TestInput_GetValue(t *testing.T) {
	i := NewInput(80, 3)
	i.textarea.SetValue("Test value")

	value := i.GetValue()

	assert.Equal(t, "Test value", value)
}

func TestInput_Clear(t *testing.T) {
	i := NewInput(80, 3)
	i.SetValue("Test")

	i.Clear()

	assert.Equal(t, "", i.GetValue())
	assert.Equal(t, -1, i.triggerPos)
}

func TestInput_TriggerDetection_AtStart(t *testing.T) {
	i := NewInput(80, 3)
	triggered := false
	i.SetTriggerCallback(func() { triggered = true })

	i.SetValue("@")

	assert.True(t, triggered)
	assert.Equal(t, 0, i.triggerPos)
}

func TestInput_TriggerDetection_AfterSpace(t *testing.T) {
	i := NewInput(80, 3)
	triggered := false
	i.SetTriggerCallback(func() { triggered = true })

	// Simulate typing "hello @"
	i.textarea.SetValue("hello @")
	i.textarea.CursorEnd()
	i.detectTrigger()

	assert.True(t, triggered)
	assert.Equal(t, 6, i.triggerPos)
}

func TestInput_TriggerDetection_NotInMiddleOfWord(t *testing.T) {
	i := NewInput(80, 3)
	triggered := false
	i.SetTriggerCallback(func() { triggered = true })

	// @ in middle of word (like email)
	i.textarea.SetValue("test@")
	i.textarea.CursorEnd()
	i.detectTrigger()

	assert.False(t, triggered)
	assert.Equal(t, -1, i.triggerPos)
}

func TestInput_TriggerDetection_EmailAddress(t *testing.T) {
	i := NewInput(80, 3)
	triggered := false
	i.SetTriggerCallback(func() { triggered = true })

	// Email address should not trigger
	i.textarea.SetValue("user@example.com")
	i.textarea.CursorEnd()
	i.detectTrigger()

	assert.False(t, triggered)
}

func TestInput_SetTriggerCallback(t *testing.T) {
	i := NewInput(80, 3)
	called := false

	i.SetTriggerCallback(func() { called = true })
	i.SetValue("@")

	assert.True(t, called)
}

func TestInput_AddToHistory(t *testing.T) {
	i := NewInput(80, 3)
	i.SetValue("First message")

	i.AddToHistory()

	history := i.history.GetAll()
	assert.Len(t, history, 1)
	assert.Equal(t, "First message", history[0])
}

func TestInput_AddToHistory_EmptyIgnored(t *testing.T) {
	i := NewInput(80, 3)
	i.SetValue("   ") // Only whitespace

	i.AddToHistory()

	history := i.history.GetAll()
	assert.Len(t, history, 0)
}

func TestInput_Update_Enter(t *testing.T) {
	i := NewInput(80, 3)
	i.SetValue("Test message")

	// Press Enter (without modifiers)
	newInput, cmd := i.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Should return nil command (parent handles send)
	assert.Nil(t, cmd)

	// Input should still have value (parent clears it)
	assert.Equal(t, "Test message", newInput.GetValue())
}

func TestInput_Update_ShiftEnter(t *testing.T) {
	i := NewInput(80, 3)
	i.Focus()
	i.SetValue("Line 1")

	// Press Enter - textarea handles newlines internally
	// Just verify update doesn't panic
	newInput, _ := i.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Check passes if no panic
	_ = newInput
}

func TestInput_Update_Up_HistoryNavigation(t *testing.T) {
	i := NewInput(80, 3)
	i.Focus()

	// Add history
	i.history.Add("First")
	i.history.Add("Second")

	// Press Up (navigate to previous)
	newInput, _ := i.Update(tea.KeyMsg{Type: tea.KeyUp})

	assert.Equal(t, "Second", newInput.GetValue())
}

func TestInput_Update_Down_HistoryNavigation(t *testing.T) {
	i := NewInput(80, 3)
	i.Focus()

	i.history.Add("First")
	i.history.Add("Second")

	// Navigate backward
	i.Update(tea.KeyMsg{Type: tea.KeyUp})

	// Navigate forward
	newInput, _ := i.Update(tea.KeyMsg{Type: tea.KeyDown})

	// Should go back to temp buffer (empty)
	assert.Equal(t, "", newInput.GetValue())
}

func TestInput_View(t *testing.T) {
	i := NewInput(80, 3)

	view := i.View()

	// Should render something
	assert.NotEmpty(t, view)
}

func TestInput_View_WithTrigger(t *testing.T) {
	i := NewInput(80, 3)
	i.SetValue("@")

	view := i.View()

	// Should contain hint about file picker
	assert.Contains(t, view, "file picker")
}

func TestInput_View_Focused(t *testing.T) {
	i := NewInput(80, 3)
	i.Focus()

	view := i.View()

	// Should render with focus styling
	assert.NotEmpty(t, view)
}

func TestInput_Integration_HistoryWithNavigation(t *testing.T) {
	i := NewInput(80, 3)
	i.Focus()

	// Send messages
	i.SetValue("First message")
	i.AddToHistory()
	i.Clear()

	i.SetValue("Second message")
	i.AddToHistory()
	i.Clear()

	// Navigate history with Up - should go to most recent first
	prev, ok := i.history.Previous()
	assert.True(t, ok)
	assert.Equal(t, "Second message", prev)

	prev, ok = i.history.Previous()
	assert.True(t, ok)
	assert.Equal(t, "First message", prev)

	next, ok := i.history.Next()
	assert.True(t, ok)
	assert.Equal(t, "Second message", next)
}

func TestInput_Integration_TriggerAfterTyping(t *testing.T) {
	i := NewInput(80, 3)
	triggered := false
	i.SetTriggerCallback(func() { triggered = true })

	// Type gradually
	i.textarea.SetValue("Hello ")
	i.detectTrigger()
	assert.False(t, triggered)

	i.textarea.SetValue("Hello @")
	i.textarea.CursorEnd()
	i.detectTrigger()
	assert.True(t, triggered)
}
