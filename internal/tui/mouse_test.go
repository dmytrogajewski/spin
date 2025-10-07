package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestHandleMouseEvents(t *testing.T) {
	m := NewTestModel()
	originalInputValue := m.input.GetValue()

	// Simulate mouse wheel scroll event
	mouseMsg := tea.MouseMsg{
		Type:   tea.MouseWheelUp,
		X:      10,
		Y:      10,
		Button: tea.MouseButtonWheelUp,
	}

	// Update model with mouse message
	updatedModel, _ := m.Update(mouseMsg)
	updated := updatedModel.(Model)

	// Input value should not change when scrolling
	assert.Equal(t, originalInputValue, updated.input.GetValue(),
		"Input value should not change on mouse scroll")
}

func TestHandleMouseWheelDown(t *testing.T) {
	m := NewTestModel()

	// Set some initial text in input
	m.input.SetValue("test message")
	originalValue := m.input.GetValue()

	// Simulate mouse wheel down event
	mouseMsg := tea.MouseMsg{
		Type:   tea.MouseWheelDown,
		X:      10,
		Y:      10,
		Button: tea.MouseButtonWheelDown,
	}

	// Update model with mouse message
	updatedModel, _ := m.Update(mouseMsg)
	updated := updatedModel.(Model)

	// Input value should remain unchanged
	assert.Equal(t, originalValue, updated.input.GetValue(),
		"Input value should not be corrupted by mouse wheel down")
}

func TestMouseEventNotPassedToInput(t *testing.T) {
	m := NewTestModel()
	m.state = StateIdle

	// Add some text to input
	m.input.SetValue("important text")

	// Test various mouse events
	mouseEvents := []tea.MouseMsg{
		{Type: tea.MouseWheelUp},
		{Type: tea.MouseWheelDown},
		{Type: tea.MouseLeft},
		{Type: tea.MouseRight},
		{Type: tea.MouseMiddle},
		{Type: tea.MouseRelease},
		{Type: tea.MouseMotion},
	}

	for _, mouseEvent := range mouseEvents {
		originalValue := m.input.GetValue()

		updatedModel, _ := m.Update(mouseEvent)
		updated := updatedModel.(Model)

		assert.Equal(t, originalValue, updated.input.GetValue(),
			"Input value should not change for mouse event type %v", mouseEvent.Type)
	}
}