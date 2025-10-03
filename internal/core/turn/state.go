package turn

// TurnState represents turn execution state.
// It defines the lifecycle states a turn can be in, from creation through completion.
type TurnState int

const (
	// StatePending indicates turn is pending execution
	StatePending TurnState = iota
	// StateRunning indicates turn is currently executing
	StateRunning
	// StateWaitingApproval indicates turn is paused waiting for user approval
	StateWaitingApproval
	// StateCompleted indicates turn completed successfully
	StateCompleted
	// StateFailed indicates turn failed with an error
	StateFailed
	// StateCancelled indicates turn was cancelled by user
	StateCancelled
)

// String returns the string representation of TurnState.
func (s TurnState) String() string {
	switch s {
	case StatePending:
		return "pending"
	case StateRunning:
		return "running"
	case StateWaitingApproval:
		return "waiting_approval"
	case StateCompleted:
		return "completed"
	case StateFailed:
		return "failed"
	case StateCancelled:
		return "cancelled"
	default:
		return "unknown"
	}
}

// CanTransition returns true if transition from 'from' state to 'to' state is valid.
func CanTransition(from, to TurnState) bool {
	validTransitions := map[TurnState]map[TurnState]bool{
		StatePending: {
			StateRunning: true,
		},
		StateRunning: {
			StateWaitingApproval: true,
			StateCompleted:       true,
			StateFailed:          true,
			StateCancelled:       true,
		},
		StateWaitingApproval: {
			StateRunning:   true,
			StateCancelled: true,
		},
		// Terminal states cannot transition
		StateCompleted: {},
		StateFailed:    {},
		StateCancelled: {},
	}

	if targets, ok := validTransitions[from]; ok {
		return targets[to]
	}
	return false
}

// IsTerminal returns true if the state is a terminal state (cannot transition further).
func IsTerminal(state TurnState) bool {
	return state == StateCompleted || state == StateFailed || state == StateCancelled
}
