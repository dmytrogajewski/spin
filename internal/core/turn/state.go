package turn

import "github.com/dmytrogajewski/spin/internal/state"

// TurnState is now unified with state.State for consistency.
// Use state.State instead of this type.
type TurnState = state.State

// Turn states - now using unified state constants.
const (
	StatePending         = state.StateIdle            // Turn is pending execution
	StateRunning         = state.StateRunning         // Turn is currently executing
	StateWaitingApproval = state.StateWaitingApproval // Turn is paused waiting for user approval
	StateCompleted       = state.StateCompleted       // Turn completed successfully
	StateFailed          = state.StateFailed          // Turn failed with an error
	StateCancelled       = state.StateCancelled       // Turn was cancelled by user
)

// CanTransition returns true if transition from 'from' state to 'to' state is valid.
func CanTransition(from, to TurnState) bool {
	return from.CanTransitionTo(to)
}
