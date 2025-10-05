package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/dmytrogajewski/spin/internal/core"
	"github.com/dmytrogajewski/spin/internal/tui/ui"
)

// Config contains TUI-specific configuration extracted from core.Config.
type Config struct {
	Model        string `mapstructure:"model"`
	Provider     string `mapstructure:"provider"`
	SandboxMode  string `mapstructure:"sandbox_mode"`
	WorkDir      string `mapstructure:"work_dir"`
}

// Model represents the TUI application state following The Elm Architecture pattern.
type Model struct {
	state    AppState // Current application state
	err      error    // Any error that occurred
	width    int      // Terminal width
	height   int      // Terminal height
	quitting bool     // Exit flag

	// UI components
	chat       ui.Chat          // Chat interface (Phase 3.2)
	input      ui.Input         // Input widget (Phase 3.3)
	statusBar  ui.StatusBar     // Status bar (Phase 3.6)
	help       ui.Help          // Help modal (Phase 3.9)
	approval   ui.ApprovalModal // Approval modal (Phase 3.5)
	errorModal ui.ErrorModal    // Error modal (Phase 3.12)

	// Backtrack mode (Phase 3.8)
	backtrackIdx  int // Index of selected message in backtrack mode (-1 = not in backtrack)
	escPressCount int // Count consecutive Esc presses for Esc-Esc detection

	// Core integration (Phase 3.11)
	coreManager *CoreManager      // Core manager
	events      <-chan core.Event // Event stream from core
}

// ErrorMsg represents an error message for the Update function.
type ErrorMsg struct {
	Err error
}

// NewModel creates a new TUI model with initial state.
func NewModel() Model {
	return Model{
		state:         StateIdle,
		err:           nil,
		width:         0,
		height:        0,
		quitting:      false,
		chat:          ui.NewChat(0, 0),       // Will be sized on first resize
		input:         ui.NewInput(0, 3),      // 3 lines height (Phase 3.3)
		statusBar:     ui.NewStatusBar(0),     // Will be sized on first resize (Phase 3.6)
		help:          ui.NewHelp(0, 0),       // Will be sized on first resize (Phase 3.9)
		approval:      ui.NewApproval(),       // Approval modal (Phase 3.5)
		errorModal:    ui.NewErrorModal(0, 0), // Will be sized on first resize (Phase 3.12)
		backtrackIdx:  -1,                     // Not in backtrack mode
		escPressCount: 0,
	}
}

// NewModelWithConfig creates a new TUI model with configuration.
func NewModelWithConfig(cfg *Config) Model {
	m := NewModel()

	// Initialize status bar with config info
	statusInfo := ui.StatusInfo{
		Model:         cfg.Model,
		Provider:      cfg.Provider,
		SandboxMode:   cfg.SandboxMode,
		WorkingDir:    cfg.WorkDir,
		Status:        ui.StatusIdle,
		TurnTokens:    0,
		SessionTokens: 0,
	}
	m.statusBar.SetInfo(statusInfo)

	return m
}

// Err returns any error that occurred during TUI operation.
func (m Model) Err() error {
	return m.err
}

// State returns the current application state.
func (m Model) State() AppState {
	return m.state
}

// Init is called when the program starts.
// It returns any initial commands to run.
func (m Model) Init() tea.Cmd {
	// Focus input initially (Phase 3.3)
	// Note: The command returned here will be executed by Bubble Tea
	return m.input.Focus()
}

// Update handles incoming messages and returns an updated model.
// This is the core of The Elm Architecture's update cycle.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case tea.WindowSizeMsg:
		return m.handleResize(msg)

	case CoreEventMsg:
		// Handle core events (Phase 3.11)
		return m.handleCoreEvent(msg)

	case ErrorMsg:
		m.err = msg.Err
		return m, tea.Quit
	}

	// Update chat component
	m.chat, cmd = m.chat.Update(msg)
	cmds = append(cmds, cmd)

	// Update input component (Phase 3.3)
	m.input, cmd = m.input.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// handleKeyPress processes keyboard input.
func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// State-specific key handling first
	switch m.state {
	case StateHelp:
		// Any key dismisses help
		return m.dismissHelp()

	case StateExiting:
		// Ignore all input during exit
		return m, nil
	}

	// Handle global shortcuts that work in any state
	switch msg.String() {
	case "ctrl+c":
		return m.handleCtrlC()

	case "ctrl+d":
		return m.handleCtrlD()

	case "ctrl+h", "?":
		return m.handleHelp()

	case "ctrl+l":
		return m.handleCtrlL()

	case "esc":
		return m.handleEscPress()

	case "enter":
		return m.handleEnterPress()
	}

	// Reset Esc counter on any other key
	m.escPressCount = 0

	// Pass all other keys to input component (typing, arrows, etc.)
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// handleEscPress handles Esc key logic for backtrack mode.
func (m Model) handleEscPress() (tea.Model, tea.Cmd) {
	switch m.state {
	case StateIdle:
		// Check if input is empty
		if m.input.GetValue() == "" {
			m.escPressCount++

			// Esc-Esc detection
			if m.escPressCount >= 2 {
				// Try to enter backtrack mode
				return m.enterBacktrackMode()
			}
		} else {
			// Input not empty - clear it (delegate to input component)
			m.escPressCount = 0
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(tea.KeyMsg{Type: tea.KeyEsc})
			return m, cmd
		}

	case StateBacktrackMode:
		// Navigate to previous user message
		m.escPressCount = 0 // Reset counter
		return m.navigateBacktrack()

	case StateHelp:
		// Esc dismisses help
		return m.dismissHelp()

	default:
		m.escPressCount = 0
	}

	return m, nil
}

// handleEnterPress handles Enter key logic.
func (m Model) handleEnterPress() (tea.Model, tea.Cmd) {
	m.escPressCount = 0 // Reset Esc counter

	switch m.state {
	case StateIdle:
		// Handle message submission (Phase 3.3)
		if m.input.GetValue() != "" {
			message := m.input.GetValue()

			// Check if this is a forked message (backtrackIdx was set)
			if m.backtrackIdx >= 0 {
				// Fork conversation: truncate after selected message
				m.chat.TruncateAfter(m.backtrackIdx - 1)
			}

			// Add to history
			m.input.AddToHistory()

			// Add to chat as user message
			m.chat.AddMessage(ui.Message{
				Role:    ui.RoleUser,
				Content: message,
			})

			// Clear input
			m.input.Clear()

			// Reset backtrack state
			m.backtrackIdx = -1

			// TODO: Send to core in Phase 3.11
			return m, nil
		}

	case StateBacktrackMode:
		// Load selected message into input
		return m.selectBacktrackMessage()
	}

	return m, nil
}

// enterBacktrackMode enters backtrack mode if there are user messages.
func (m Model) enterBacktrackMode() (tea.Model, tea.Cmd) {
	// Find user message indices
	userIndices := m.chat.GetUserMessageIndices()

	if len(userIndices) == 0 {
		// No user messages - stay in Idle
		m.escPressCount = 0
		return m, nil
	}

	// Transition to backtrack mode
	if !m.state.CanTransitionTo(StateBacktrackMode) {
		m.escPressCount = 0
		return m, nil
	}

	m.state = StateBacktrackMode
	m.escPressCount = 0

	// Start at last user message
	m.backtrackIdx = userIndices[len(userIndices)-1]

	// Highlight the selected message
	m.chat.SetHighlight(m.backtrackIdx)

	return m, nil
}

// navigateBacktrack moves to the previous user message.
func (m Model) navigateBacktrack() (tea.Model, tea.Cmd) {
	userIndices := m.chat.GetUserMessageIndices()

	if len(userIndices) == 0 {
		return m, nil
	}

	// Find current position in userIndices
	currentPos := -1
	for i, idx := range userIndices {
		if idx == m.backtrackIdx {
			currentPos = i
			break
		}
	}

	if currentPos < 0 {
		// Invalid state - reset to last message
		m.backtrackIdx = userIndices[len(userIndices)-1]
		m.chat.SetHighlight(m.backtrackIdx)
		return m, nil
	}

	// Move to previous user message (if not at first)
	if currentPos > 0 {
		m.backtrackIdx = userIndices[currentPos-1]
		m.chat.SetHighlight(m.backtrackIdx)
	}
	// If at first message, stay there (no wrap-around)

	return m, nil
}

// selectBacktrackMessage loads the selected message into input and exits backtrack.
func (m Model) selectBacktrackMessage() (tea.Model, tea.Cmd) {
	// Get the selected message
	messages := m.chat.GetMessages()
	if m.backtrackIdx < 0 || m.backtrackIdx >= len(messages) {
		// Invalid index - just exit backtrack
		m.state = StateIdle
		m.backtrackIdx = -1
		m.chat.ClearHighlight()
		return m, nil
	}

	selectedMsg := messages[m.backtrackIdx]

	// Validate state transition
	if !m.state.CanTransitionTo(StateIdle) {
		return m, nil
	}

	// Load message into input
	m.input.SetValue(selectedMsg.Content)

	// Clear highlight
	m.chat.ClearHighlight()

	// Transition to Idle (keep backtrackIdx for conversation forking)
	m.state = StateIdle

	return m, nil
}

// handleResize processes terminal resize events.
func (m Model) handleResize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height

	// Size components (Phase 3.6 update)
	// Reserve space: 3 lines for input, 1 line for status bar, rest for chat
	inputHeight := 3
	statusHeight := 1
	reservedHeight := inputHeight + statusHeight + 1 // +1 for margins

	chatHeight := m.height - reservedHeight
	if chatHeight < 5 {
		chatHeight = 5 // Minimum height
	}

	m.chat.SetSize(m.width, chatHeight)
	m.input.SetSize(m.width-2, inputHeight) // -2 for border padding
	m.statusBar.SetWidth(m.width)           // Phase 3.6
	m.help.SetSize(m.width, m.height)       // Phase 3.9
	m.errorModal.Resize(m.width, m.height)  // Phase 3.12

	return m, nil
}

// handleCtrlC handles Ctrl+C for command cancellation.
// Phase 3.9 implementation.
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
		// In other states, do nothing (or could exit)
		return m, nil
	}
}

// cancelTurn cancels the current AI turn.
// Phase 3.9 - Stub implementation (will integrate with core in Phase 3.11).
func (m Model) cancelTurn() (tea.Model, tea.Cmd) {
	// TODO: Send cancellation signal to core in Phase 3.11

	// Add cancellation message to transcript
	m.chat.AddMessage(ui.Message{
		Role:    ui.RoleSystem,
		Content: "Turn cancelled by user",
	})

	// Transition to Idle
	m.state = StateIdle
	m.escPressCount = 0

	return m, nil
}

// denyApproval denies the pending tool approval.
// Phase 3.9 - Stub implementation.
func (m Model) denyApproval() (tea.Model, tea.Cmd) {
	// Add denial message
	m.chat.AddMessage(ui.Message{
		Role:    ui.RoleSystem,
		Content: "Tool call denied by user",
	})

	// Transition to Idle
	if m.state.CanTransitionTo(StateIdle) {
		m.state = StateIdle
	}
	m.escPressCount = 0

	return m, nil
}

// exitBacktrack exits backtrack mode without selection.
// Phase 3.9 implementation.
func (m Model) exitBacktrack() (tea.Model, tea.Cmd) {
	// Clear highlight
	m.chat.ClearHighlight()

	// Reset backtrack state
	m.backtrackIdx = -1
	m.escPressCount = 0

	// Transition to Idle
	if m.state.CanTransitionTo(StateIdle) {
		m.state = StateIdle
	}

	return m, nil
}

// handleCtrlD handles Ctrl+D for graceful exit.
// Phase 3.9 implementation.
func (m Model) handleCtrlD() (tea.Model, tea.Cmd) {
	// Transition to Exiting
	m.state = StateExiting
	m.quitting = true
	m.escPressCount = 0

	// TODO: Send shutdown signal to core in Phase 3.11
	// TODO: Wait for core cleanup (max 2s timeout)

	return m, tea.Quit
}

// handleCtrlL handles Ctrl+L for screen clear.
// Phase 3.9 implementation.
func (m Model) handleCtrlL() (tea.Model, tea.Cmd) {
	// Only works in Idle state
	if m.state != StateIdle {
		return m, nil
	}

	m.escPressCount = 0

	// Scroll chat to bottom (clear screen effect)
	m.chat.ScrollToBottom()

	return m, nil
}

// handleHelp shows the help modal.
// Phase 3.9 implementation.
func (m Model) handleHelp() (tea.Model, tea.Cmd) {
	// Can show help from most states
	if !m.state.CanTransitionTo(StateHelp) {
		return m, nil
	}

	m.state = StateHelp
	m.escPressCount = 0

	return m, nil
}

// dismissHelp dismisses the help modal and returns to previous state.
// Phase 3.9 implementation.
func (m Model) dismissHelp() (tea.Model, tea.Cmd) {
	// Return to Idle (simplification - could track previous state)
	if m.state.CanTransitionTo(StateIdle) {
		m.state = StateIdle
	}
	m.escPressCount = 0

	return m, nil
}

// View renders the UI to a string.
// This is called after every Update and should be fast (<16ms for 60 FPS).
func (m Model) View() string {
	if m.quitting {
		return ""
	}

	// Show help modal overlay if in help state
	if m.state == StateHelp {
		return m.help.View()
	}

	// Placeholder view for Phase 3.1
	// Real UI components will be added in later phases (3.2-3.10)
	return m.renderPlaceholder()
}
