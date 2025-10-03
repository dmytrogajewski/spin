package turn

// TurnState represents turn execution state.
// This is a minimal implementation for Feature 1.1 (Session Management).
// Full implementation will be done in Feature 1.2.
type TurnState int

const (
	// StatePending indicates turn is pending execution
	StatePending TurnState = iota
	// StateRunning indicates turn is currently executing
	StateRunning
	// StateCompleted indicates turn completed successfully
	StateCompleted
	// StateFailed indicates turn failed
	StateFailed
	// StateCancelled indicates turn was cancelled
	StateCancelled
)
