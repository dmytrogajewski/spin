package state

import "fmt"

// State represents the current state of the system.
type State string

// State constants
const (
	StateIdle            State = "idle"
	StateRunning         State = "running"
	StateWaitingApproval State = "waiting_approval"
	StateCompleted       State = "completed"
	StateFailed          State = "failed"
	StateCancelled       State = "cancelled"
	StatePaused          State = "paused"
	StateArchived        State = "archived"
	StateActive          State = "active"
)

// validStates is the set of all valid states for validation.
var validStates = map[State]bool{
	StateIdle:            true,
	StateRunning:         true,
	StateWaitingApproval: true,
	StateCompleted:       true,
	StateFailed:          true,
	StateCancelled:       true,
	StatePaused:          true,
	StateArchived:        true,
	StateActive:          true,
}

// CanTransitionTo returns true if transition to the target state is valid.
func (s State) CanTransitionTo(target State) bool {
	// Cannot transition from unknown states
	if !validStates[s] {
		return false
	}

	// Cannot transition to same state
	if s == target {
		return false
	}

	// State-specific transition rules
	switch s {
	case StateIdle:
		// Idle can only go to running or archived
		return target == StateRunning || target == StateArchived

	case StateRunning:
		// Running can go to paused, waiting_approval, completed, failed, or cancelled
		return target == StatePaused || target == StateWaitingApproval ||
			target == StateCompleted || target == StateFailed || target == StateCancelled

	case StatePaused:
		// Paused can go to running, cancelled, or archived
		return target == StateRunning || target == StateCancelled || target == StateArchived

	case StateWaitingApproval:
		// WaitingApproval can go to running or cancelled
		return target == StateRunning || target == StateCancelled

	case StateCompleted, StateFailed, StateCancelled:
		// Terminal states can only go to archived
		return target == StateArchived

	case StateArchived:
		// Archived is final - cannot transition anywhere
		return false

	default:
		return false
	}
}

// String returns the string representation of the state.
// Returns "unknown" for unrecognized states.
func (s State) String() string {
	if validStates[s] {
		return string(s)
	}
	return "unknown"
}

// MarshalText implements encoding.TextMarshaler.
func (s State) MarshalText() ([]byte, error) {
	return []byte(s), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
// Returns an error for invalid or empty states.
func (s *State) UnmarshalText(text []byte) error {
	state := State(text)
	if len(text) == 0 {
		return fmt.Errorf("state cannot be empty")
	}
	if !validStates[state] {
		return fmt.Errorf("invalid state: %s", text)
	}
	*s = state
	return nil
}

// IsTerminal returns true if the state is terminal (completed, failed, cancelled, archived).
func (s State) IsTerminal() bool {
	switch s {
	case StateCompleted, StateFailed, StateCancelled, StateArchived:
		return true
	default:
		return false
	}
}

// IsActive returns true if the state is actively working (running, paused, or waiting for approval).
func (s State) IsActive() bool {
	switch s {
	case StateRunning, StatePaused, StateWaitingApproval:
		return true
	default:
		return false
	}
}
