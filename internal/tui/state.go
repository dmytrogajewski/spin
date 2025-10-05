package tui

// AppState represents the different states of the TUI application.
type AppState int

const (
	// StateIdle indicates the TUI is waiting for user input
	StateIdle AppState = iota

	// StateWaitingResponse indicates the AI is generating a response
	StateWaitingResponse

	// StateToolApproval indicates waiting for user to approve/deny a tool call
	StateToolApproval

	// StateFilePickerOpen indicates the @ file picker is active
	StateFilePickerOpen

	// StateBacktrackMode indicates Esc-Esc backtrack mode is active
	StateBacktrackMode

	// StateHelp indicates the help modal is displayed
	StateHelp

	// StateExiting indicates the application is shutting down
	StateExiting
)

// String returns the string representation of the state.
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
	case StateHelp:
		return "help"
	case StateExiting:
		return "exiting"
	default:
		return "unknown"
	}
}

// CanTransitionTo checks if a transition from the current state to a new state is valid.
// This enforces the state machine rules and prevents invalid state transitions.
func (s AppState) CanTransitionTo(new AppState) bool {
	// No self-transitions allowed (except implicitly staying in same state)
	if s == new {
		return false
	}

	// Terminal state - no transitions allowed
	if s == StateExiting {
		return false
	}

	// Check if transition is in the allowed list for current state
	return s.isTransitionAllowed(new)
}

// isTransitionAllowed checks if a specific transition is allowed for the current state.
// This helper function reduces complexity of CanTransitionTo.
func (s AppState) isTransitionAllowed(new AppState) bool {
	// Most states can transition to Help or Exiting
	if s.canAlwaysTransitionTo(new) {
		return true
	}

	// State-specific transitions
	switch s {
	case StateIdle:
		return new == StateWaitingResponse || new == StateFilePickerOpen || new == StateBacktrackMode

	case StateWaitingResponse:
		return new == StateIdle || new == StateToolApproval

	case StateToolApproval:
		return new == StateIdle || new == StateWaitingResponse

	case StateFilePickerOpen:
		return new == StateIdle

	case StateBacktrackMode:
		return new == StateIdle

	case StateHelp:
		return new == StateIdle

	default:
		return false
	}
}

// canAlwaysTransitionTo checks if a state can always transition to certain common states.
// Reduces cyclomatic complexity of isTransitionAllowed.
func (s AppState) canAlwaysTransitionTo(new AppState) bool {
	// Can't transition from terminal state
	if s == StateExiting {
		return false
	}

	// Most states can go to Help or Exiting
	return new == StateHelp || new == StateExiting
}
