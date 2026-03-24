package ds

import (
	"errors"
	"fmt"
)

// ErrInvalidTransition indicates an invalid state transition.
var ErrInvalidTransition = errors.New("invalid state transition")

// StateMachine validates state transitions using a configurable transition table.
type StateMachine[State comparable] struct {
	transitions map[State]map[State]bool
}

// NewStateMachine creates a state machine with the given allowed transitions.
// Each key maps to a set of states it can transition to.
func NewStateMachine[State comparable](transitions map[State]map[State]bool) *StateMachine[State] {
	return &StateMachine[State]{transitions: transitions}
}

// CanTransition returns true if transitioning from current to target is allowed.
func (sm *StateMachine[State]) CanTransition(current, target State) bool {
	if current == target {
		return false
	}

	allowed, ok := sm.transitions[current]
	if !ok {
		return false
	}

	return allowed[target]
}

// Validate returns an error if the transition is not allowed.
func (sm *StateMachine[State]) Validate(current, target State) error {
	if !sm.CanTransition(current, target) {
		return fmt.Errorf("%w: %v -> %v", ErrInvalidTransition, current, target)
	}

	return nil
}
