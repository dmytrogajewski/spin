package state

import (
	"fmt"
)

// State represents the execution state for all components in the system.
// This replaces core.State, session.State, and turn.TurnState for consistency.
//
// State Transitions:
//
//	Idle → Running → Completed
//	               → Failed
//	               → Cancelled
//	     → Paused → Running
//	     → WaitingApproval → Running (turns only)
//	     → Archived (sessions only)
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

	// StateWaitingApproval indicates execution is paused waiting for user approval (turns only)
	StateWaitingApproval

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
func (s State) String() string {
	switch s {
	case StateIdle:
		return "idle"
	case StateRunning:
		return "running"
	case StatePaused:
		return "paused"
	case StateWaitingApproval:
		return "waiting_approval"
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
func (s State) MarshalText() ([]byte, error) {
	return []byte(s.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler for JSON decoding.
func (s *State) UnmarshalText(text []byte) error {
	str := string(text)
	switch str {
	case "idle":
		*s = StateIdle
	case "running":
		*s = StateRunning
	case "paused":
		*s = StatePaused
	case "waiting_approval":
		*s = StateWaitingApproval
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
func (s State) IsTerminal() bool {
	return s == StateCompleted || s == StateFailed || s == StateCancelled || s == StateArchived
}

// IsActive returns true if the state indicates active or ongoing execution.
func (s State) IsActive() bool {
	return s == StateRunning || s == StatePaused || s == StateWaitingApproval
}

// CanTransitionTo validates if a transition to the target state is valid.
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
		// Running can pause, wait for approval, complete, fail, or be cancelled
		return target == StatePaused || target == StateWaitingApproval ||
			target == StateCompleted || target == StateFailed || target == StateCancelled

	case StatePaused:
		// Paused can resume running, be cancelled, or archived
		return target == StateRunning || target == StateCancelled || target == StateArchived

	case StateWaitingApproval:
		// Waiting approval can resume running or be cancelled
		return target == StateRunning || target == StateCancelled

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
