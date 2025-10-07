package tui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/dmytrogajewski/spin/internal/core"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/tui/ui"
)

// Config contains TUI-specific configuration extracted from core.Config.
type Config struct {
	Model       string `mapstructure:"model"`
	Provider    string `mapstructure:"provider"`
	SandboxMode string `mapstructure:"sandbox_mode"`
	WorkDir     string `mapstructure:"work_dir"`
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
	spinner    ui.Spinner       // Loading spinner

	// Backtrack mode (Phase 3.8)
	backtrackIdx  int // Index of selected message in backtrack mode (-1 = not in backtrack)
	escPressCount int // Count consecutive Esc presses for Esc-Esc detection

	// Display preferences
	showThinking bool // Whether to show full thinking or collapse it

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
		spinner:       ui.NewSpinner(),        // Loading spinner
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

// NewModelWithLLM creates a new TUI model with LLM integration.
func NewModelWithLLM(tuiCfg *Config, coreCfg *core.Config, provider llm.Provider) (Model, error) {
	m := NewModelWithConfig(tuiCfg)

	Info("Creating TUI model with LLM", "provider", coreCfg.Provider, "model", coreCfg.Model)

	// Create core manager
	coreManager, err := NewCoreManager(coreCfg, provider)
	if err != nil {
		Error("Failed to create core manager", "error", err)
		return m, fmt.Errorf("create core manager: %w", err)
	}

	// Start conversation and get event stream
	events, err := coreManager.StartConversation()
	if err != nil {
		Error("Failed to start conversation", "error", err)
		coreManager.Close()
		return m, fmt.Errorf("start conversation: %w", err)
	}

	Info("Conversation started successfully", "has_events", events != nil)

	m.coreManager = coreManager
	m.events = events

	return m, nil
}

// Close cleans up TUI resources.
func (m Model) Close() error {
	if m.coreManager != nil {
		return m.coreManager.Close()
	}
	return nil
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
	Info("Init called", "has_core_manager", m.coreManager != nil, "has_events", m.events != nil)

	// Start listening to core events if manager exists
	if m.coreManager != nil && m.events != nil {
		Info("Starting event listener and ticker")
		return tea.Batch(
			m.input.Focus(),
			waitForBatchedEvents(m.events, time.Millisecond*8),
			tickCmd(), // Start periodic ticks for rendering
		)
	}

	// Focus input initially
	Info("No core manager, just focusing input")
	return m.input.Focus()
}

// tickCmd creates a command that sends periodic tick messages for UI updates.
func tickCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*16, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// tickMsg represents a periodic tick for UI updates.
type tickMsg time.Time

// llmErrorMsg represents an error from the LLM.
type llmErrorMsg struct {
	err error
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

	case tea.MouseMsg:
		// Handle mouse events - only pass to chat for scrolling
		// DO NOT pass to input component to prevent symbols appearing
		m.chat, cmd = m.chat.Update(msg)
		return m, cmd

	case CoreEventMsg:
		// Handle single core event (backward compatibility)
		return m.handleCoreEvent(msg)

	case BatchedEventsMsg:
		// Handle batched core events for better streaming performance
		return m.handleBatchedEvents(msg)

	case ui.ApprovalDecisionMsg:
		// Handle approval decision from modal
		if m.coreManager != nil && m.coreManager.ApprovalBridge() != nil {
			// Send the response to the approval bridge
			m.coreManager.ApprovalBridge().SendResponse(msg.Response)
		}
		// Clear the approval modal and return to waiting state
		m.approval.Clear()
		m.state = StateWaitingResponse
		return m, nil

	case ErrorMsg:
		m.err = msg.Err
		return m, tea.Quit

	case tickMsg:
		// Periodic tick for UI updates (streaming, spinner animation, etc.)
		// Animate spinner
		m.spinner.Tick()

		// Update chat to trigger re-render if dirty
		m.chat, cmd = m.chat.Update(msg)
		cmds = append(cmds, cmd)

		// Schedule next tick
		return m, tea.Batch(cmd, tickCmd())

	default:
		// For all other message types, update both components
		// Update chat component
		m.chat, cmd = m.chat.Update(msg)
		cmds = append(cmds, cmd)

		// Update input component (Phase 3.3)
		m.input, cmd = m.input.Update(msg)
		cmds = append(cmds, cmd)

		return m, tea.Batch(cmds...)
	}
}

// handleKeyPress processes keyboard input.
func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// State-specific key handling first
	switch m.state {
	case StateHelp:
		// Any key dismisses help
		return m.dismissHelp()

	case StateToolApproval:
		// Handle approval modal keys
		updatedApproval, cmd := m.approval.Update(msg)
		m.approval = updatedApproval
		return m, cmd

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

	case "ctrl+h":
		// Only Ctrl+H for help, not '?' (which is a regular character)
		return m.handleHelp()

	case "ctrl+l":
		return m.handleCtrlL()

	case "ctrl+t":
		return m.handleCtrlT()

	case "esc":
		return m.handleEscPress()

	case "enter":
		return m.handleEnterPress()
	}

	// Reset Esc counter on any other key
	m.escPressCount = 0

	// Handle scrolling keys - pass to chat viewport
	switch msg.String() {
	case "pgup", "pgdown", "home", "end":
		var cmd tea.Cmd
		m.chat, cmd = m.chat.Update(msg)
		return m, cmd
	}

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
		// Handle message submission
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

			// Send to LLM
			if m.coreManager != nil {
				Info("Sending message to LLM", "message", message)

				// Start spinner
				m.spinner.Start()

				// Transition to waiting state immediately
				m.state = StateWaitingResponse
				Info("State changed to WaitingResponse")

				go func() {
					Info("SendMessage goroutine started")
					if err := m.coreManager.SendMessage(message); err != nil {
						Error("SendMessage failed", "error", err)
					} else {
						Info("SendMessage completed successfully")
					}
				}()
			} else {
				Warn("No core manager available")
			}

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

	// Force chat to re-render
	m.chat.Update(msg)

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

// handleCtrlT toggles thinking display (expanded/collapsed).
func (m Model) handleCtrlT() (tea.Model, tea.Cmd) {
	m.escPressCount = 0

	// Toggle thinking display
	m.chat.ToggleThinking()

	return m, nil
}

func (m Model) handleHelp() (tea.Model, tea.Cmd) {
	// Can show help from most states
	if !m.state.CanTransitionTo(StateHelp) {
		return m, nil
	}

	m.state = StateHelp
	m.escPressCount = 0

	return m, nil
}

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

	if m.state == StateHelp {
		return m.help.View()
	}

	return m.renderChat()
}
