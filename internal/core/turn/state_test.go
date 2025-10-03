package turn

import (
	"testing"
)

func TestTurnState_String(t *testing.T) {
	tests := []struct {
		name  string
		state TurnState
		want  string
	}{
		{"Pending", StatePending, "pending"},
		{"Running", StateRunning, "running"},
		{"WaitingApproval", StateWaitingApproval, "waiting_approval"},
		{"Completed", StateCompleted, "completed"},
		{"Failed", StateFailed, "failed"},
		{"Cancelled", StateCancelled, "cancelled"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.state.String(); got != tt.want {
				t.Errorf("TurnState.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCanTransition(t *testing.T) {
	tests := []struct {
		name string
		from TurnState
		to   TurnState
		want bool
	}{
		// Valid transitions from Pending
		{"Pending to Running", StatePending, StateRunning, true},

		// Invalid transitions from Pending
		{"Pending to WaitingApproval", StatePending, StateWaitingApproval, false},
		{"Pending to Completed", StatePending, StateCompleted, false},
		{"Pending to Failed", StatePending, StateFailed, false},
		{"Pending to Cancelled", StatePending, StateCancelled, false},

		// Valid transitions from Running
		{"Running to WaitingApproval", StateRunning, StateWaitingApproval, true},
		{"Running to Completed", StateRunning, StateCompleted, true},
		{"Running to Failed", StateRunning, StateFailed, true},
		{"Running to Cancelled", StateRunning, StateCancelled, true},

		// Invalid transitions from Running
		{"Running to Pending", StateRunning, StatePending, false},

		// Valid transitions from WaitingApproval
		{"WaitingApproval to Running", StateWaitingApproval, StateRunning, true},
		{"WaitingApproval to Cancelled", StateWaitingApproval, StateCancelled, true},

		// Invalid transitions from WaitingApproval
		{"WaitingApproval to Pending", StateWaitingApproval, StatePending, false},
		{"WaitingApproval to Completed", StateWaitingApproval, StateCompleted, false},
		{"WaitingApproval to Failed", StateWaitingApproval, StateFailed, false},

		// Terminal states cannot transition
		{"Completed to Running", StateCompleted, StateRunning, false},
		{"Completed to Pending", StateCompleted, StatePending, false},
		{"Failed to Running", StateFailed, StateRunning, false},
		{"Failed to Pending", StateFailed, StatePending, false},
		{"Cancelled to Running", StateCancelled, StateRunning, false},
		{"Cancelled to Pending", StateCancelled, StatePending, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CanTransition(tt.from, tt.to); got != tt.want {
				t.Errorf("CanTransition(%v, %v) = %v, want %v", tt.from, tt.to, got, tt.want)
			}
		})
	}
}

func TestIsTerminal(t *testing.T) {
	tests := []struct {
		name  string
		state TurnState
		want  bool
	}{
		{"Pending not terminal", StatePending, false},
		{"Running not terminal", StateRunning, false},
		{"WaitingApproval not terminal", StateWaitingApproval, false},
		{"Completed is terminal", StateCompleted, true},
		{"Failed is terminal", StateFailed, true},
		{"Cancelled is terminal", StateCancelled, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsTerminal(tt.state); got != tt.want {
				t.Errorf("IsTerminal(%v) = %v, want %v", tt.state, got, tt.want)
			}
		})
	}
}
