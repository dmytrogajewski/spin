// Package state provides application state management.
package state

import (
	"errors"
	"fmt"
)

var (
	// ErrStateCannotBeEmpty is a sentinel error.
	ErrStateCannotBeEmpty = errors.New("state cannot be empty")
	// ErrInvalidState is a sentinel error.
	ErrInvalidState = errors.New("invalid state")
)

// State represents the current state of the system.
type State string

// State constants.
const (
	StateIdle            State = "idle"
	StateRunning         State = "running"
	StateWaitingApproval State = "waiting_approval"
	StateCompleted       State = "completed"
	StateFailed          State = "failed"
	StateCancelled       State = "canceled"
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

// allowedTransitions maps each state to the set of states it can transition to.
var allowedTransitions = map[State]map[State]bool{
	StateIdle:            {StateRunning: true, StateArchived: true},
	StateRunning:         {StatePaused: true, StateWaitingApproval: true, StateCompleted: true, StateFailed: true, StateCancelled: true},
	StatePaused:          {StateRunning: true, StateCancelled: true, StateArchived: true},
	StateWaitingApproval: {StateRunning: true, StateCancelled: true},
	StateCompleted:       {StateArchived: true},
	StateFailed:          {StateArchived: true},
	StateCancelled:       {StateArchived: true},
	StateArchived:        {},
	StateActive:          {},
}

// CanTransitionTo returns true if transition to the target state is valid.
func (s *State) CanTransitionTo(target State) bool {
	if !validStates[*s] || *s == target {
		return false
	}

	return allowedTransitions[*s][target]
}

// String returns the string representation of the state.
// Returns "unknown" for unrecognized states.
func (s *State) String() string {
	if validStates[*s] {
		return string(*s)
	}

	return "unknown"
}

// MarshalText implements [encoding.TextMarshaler].
func (s *State) MarshalText() ([]byte, error) {
	return []byte(*s), nil
}

// UnmarshalText implements [encoding.TextUnmarshaler].
// Returns an error for invalid or empty states.
func (s *State) UnmarshalText(text []byte) error {
	state := State(text)
	if len(text) == 0 {
		return ErrStateCannotBeEmpty
	}

	if !validStates[state] {
		return fmt.Errorf("invalid state: %s: %w", text, ErrInvalidState)
	}

	*s = state

	return nil
}

// IsTerminal returns true if the state is terminal (completed, failed, canceled, archived).
func (s *State) IsTerminal() bool {
	switch *s {
	case StateCompleted, StateFailed, StateCancelled, StateArchived:
		return true
	default:
		return false
	}
}

// IsActive returns true if the state is actively working (running, paused, or waiting for approval).
func (s *State) IsActive() bool {
	switch *s {
	case StateRunning, StatePaused, StateWaitingApproval:
		return true
	default:
		return false
	}
}
