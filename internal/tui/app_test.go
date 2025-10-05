package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestNewModel(t *testing.T) {
	m := NewModel()

	assert.Equal(t, StateIdle, m.state, "initial state should be Idle")
	assert.Nil(t, m.err, "initial error should be nil")
	assert.False(t, m.quitting, "initial quitting should be false")
	assert.Equal(t, 0, m.width, "initial width should be 0")
	assert.Equal(t, 0, m.height, "initial height should be 0")
}

func TestModel_Init(t *testing.T) {
	m := NewModel()
	cmd := m.Init()

	// Init should return focus command for input (Phase 3.3)
	assert.NotNil(t, cmd)
}

func TestModel_Err(t *testing.T) {
	m := NewModel()
	assert.Nil(t, m.Err())

	// Set an error
	m.err = assert.AnError
	assert.Equal(t, assert.AnError, m.Err())
}

func TestModel_State(t *testing.T) {
	m := NewModel()
	assert.Equal(t, StateIdle, m.State())

	m.state = StateWaitingResponse
	assert.Equal(t, StateWaitingResponse, m.State())
}

func TestModel_Update_WindowResize(t *testing.T) {
	m := NewModel()

	// Send window resize message
	newModel, cmd := m.Update(tea.WindowSizeMsg{
		Width:  120,
		Height: 40,
	})

	assert.Nil(t, cmd, "resize should not return a command")

	// Check dimensions updated
	updatedModel := newModel.(Model)
	assert.Equal(t, 120, updatedModel.width)
	assert.Equal(t, 40, updatedModel.height)
}

func TestModel_Update_KeyPress_CtrlC(t *testing.T) {
	m := NewModel()

	// Send Ctrl+C
	newModel, cmd := m.Update(tea.KeyMsg{
		Type: tea.KeyCtrlC,
	})

	updatedModel := newModel.(Model)
	assert.Equal(t, StateExiting, updatedModel.state, "should transition to Exiting")
	assert.True(t, updatedModel.quitting, "quitting flag should be set")
	assert.NotNil(t, cmd, "should return quit command")
}

func TestModel_Update_KeyPress_CtrlD(t *testing.T) {
	m := NewModel()

	// Send Ctrl+D
	newModel, cmd := m.Update(tea.KeyMsg{
		Type: tea.KeyCtrlD,
	})

	updatedModel := newModel.(Model)
	assert.Equal(t, StateExiting, updatedModel.state, "should transition to Exiting")
	assert.True(t, updatedModel.quitting, "quitting flag should be set")
	assert.NotNil(t, cmd, "should return quit command")
}

func TestModel_Update_ErrorMsg(t *testing.T) {
	m := NewModel()

	// Send error message
	newModel, cmd := m.Update(ErrorMsg{
		Err: assert.AnError,
	})

	updatedModel := newModel.(Model)
	assert.Equal(t, assert.AnError, updatedModel.err, "error should be set")
	assert.NotNil(t, cmd, "should return quit command on error")
}

func TestModel_View_NotQuitting(t *testing.T) {
	m := NewModel()
	m.width = 80
	m.height = 24
	m.state = StateIdle

	view := m.View()

	// Should render without error (view contains status bar component now)
	assert.NotEmpty(t, view, "view should not be empty")
	// Status bar should show idle icon
	assert.Contains(t, view, "⏸", "view should show idle status icon")
}

func TestModel_View_Quitting(t *testing.T) {
	m := NewModel()
	m.quitting = true

	view := m.View()

	// Should return empty string when quitting
	assert.Empty(t, view, "view should be empty when quitting")
}

func TestModel_View_DifferentStates(t *testing.T) {
	states := []AppState{
		StateIdle,
		StateWaitingResponse,
		StateToolApproval,
		StateFilePickerOpen,
		StateBacktrackMode,
		StateExiting,
	}

	for _, state := range states {
		t.Run(state.String(), func(t *testing.T) {
			m := NewModel()
			m.state = state
			m.width = 80
			m.height = 24

			view := m.View()

			// View should render successfully for all states
			assert.NotEmpty(t, view, "view should not be empty for state "+state.String())
		})
	}
}

// Integration test: full message flow
func TestModel_MessageFlow(t *testing.T) {
	m := NewModel()

	// 1. Init
	cmd := m.Init()
	assert.NotNil(t, cmd) // Returns focus command (Phase 3.3)
	assert.Equal(t, StateIdle, m.state)

	// 2. Resize
	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = newModel.(Model)
	assert.Equal(t, 100, m.width)
	assert.Equal(t, 30, m.height)

	// 3. Exit with Ctrl+C
	newModel, cmd = m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = newModel.(Model)
	assert.Equal(t, StateExiting, m.state)
	assert.True(t, m.quitting)
	assert.NotNil(t, cmd)

	// 4. View when quitting
	view := m.View()
	assert.Empty(t, view)
}
