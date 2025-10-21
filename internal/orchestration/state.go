package orchestration

// TurnState represents the state of a turn.
type TurnState string

// Turn states
const (
	StatePending         TurnState = "pending"          // Turn is pending execution
	StateRunning         TurnState = "running"          // Turn is currently executing
	StateWaitingApproval TurnState = "waiting_approval" // Turn is paused waiting for user approval
	StateCompleted       TurnState = "completed"        // Turn completed successfully
	StateFailed          TurnState = "failed"           // Turn failed with an error
	StateCancelled       TurnState = "cancelled"        // Turn was cancelled by user
)

// CanTransitionTo returns true if transition to the target state is valid.
func (s TurnState) CanTransitionTo(target TurnState) bool {
	// Simple transition logic - can transition to any state
	return true
}

// CanTransition returns true if transition from 'from' state to 'to' state is valid.
func CanTransition(from, to TurnState) bool {
	return from.CanTransitionTo(to)
}
