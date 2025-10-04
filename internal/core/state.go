package core

import "fmt"

// State represents the execution state of conversations, sessions, turns, and tasks.
//
// State is used throughout the core package to track execution status in a type-safe,
// consistent manner. It provides JSON marshaling/unmarshaling and string conversion.
//
// State Transitions:
//
//	Idle → Running → Completed
//	               → Failed
//	               → Cancelled
//	     → Paused → Running
//	     → Archived (for sessions)
//
// Thread Safety: State values are immutable and safe for concurrent use.
type State int

const (
	// StateIdle indicates no active execution (initial state)
	StateIdle State = iota

	// StateRunning indicates active execution in progress
	StateRunning

	// StatePaused indicates execution is temporarily paused
	StatePaused

	// StateCompleted indicates successful completion
	StateCompleted

	// StateFailed indicates execution failed with an error
	StateFailed

	// StateCancelled indicates execution was cancelled by user
	StateCancelled

	// StateArchived indicates archived/inactive session (sessions only)
	StateArchived
)

// String returns the string representation of State.
// This is used for logging, debugging, and serialization.
func (s State) String() string {
	switch s {
	case StateIdle:
		return "idle"
	case StateRunning:
		return "running"
	case StatePaused:
		return "paused"
	case StateCompleted:
		return "completed"
	case StateFailed:
		return "failed"
	case StateCancelled:
		return "cancelled"
	case StateArchived:
		return "archived"
	default:
		return "unknown"
	}
}

// MarshalText implements encoding.TextMarshaler for JSON encoding.
// This allows State to be marshaled as a string rather than an integer.
//
// Example JSON output: "state": "running"
func (s State) MarshalText() ([]byte, error) {
	return []byte(s.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler for JSON decoding.
// This allows State to be unmarshaled from string values in JSON.
//
// Example JSON input: "state": "running"
func (s *State) UnmarshalText(text []byte) error {
	str := string(text)
	switch str {
	case "idle":
		*s = StateIdle
	case "running":
		*s = StateRunning
	case "paused":
		*s = StatePaused
	case "completed":
		*s = StateCompleted
	case "failed":
		*s = StateFailed
	case "cancelled":
		*s = StateCancelled
	case "archived":
		*s = StateArchived
	default:
		return fmt.Errorf("invalid state: %s", str)
	}
	return nil
}

// IsTerminal returns true if the state is a terminal state.
// Terminal states indicate execution has finished and won't continue.
func (s State) IsTerminal() bool {
	return s == StateCompleted || s == StateFailed || s == StateCancelled || s == StateArchived
}

// IsActive returns true if the state indicates active or ongoing execution.
func (s State) IsActive() bool {
	return s == StateRunning || s == StatePaused
}

// CanTransitionTo validates if a transition to the target state is valid.
// This helps prevent invalid state transitions.
func (s State) CanTransitionTo(target State) bool {
	// Terminal states can't transition (except to archived for sessions)
	if s.IsTerminal() && target != StateArchived {
		return false
	}

	switch s {
	case StateIdle:
		// Idle can start running or be archived
		return target == StateRunning || target == StateArchived

	case StateRunning:
		// Running can pause, complete, fail, or be cancelled
		return target == StatePaused || target == StateCompleted ||
			target == StateFailed || target == StateCancelled

	case StatePaused:
		// Paused can resume running, be cancelled, or archived
		return target == StateRunning || target == StateCancelled || target == StateArchived

	case StateCompleted, StateFailed, StateCancelled:
		// Terminal states can only transition to archived (for sessions)
		return target == StateArchived

	case StateArchived:
		// Archived is a final state
		return false

	default:
		return false
	}
}

// ParseState converts a string to a State value.
// Returns an error if the string doesn't represent a valid state.
func ParseState(s string) (State, error) {
	var state State
	if err := state.UnmarshalText([]byte(s)); err != nil {
		return StateIdle, err
	}
	return state, nil
}
