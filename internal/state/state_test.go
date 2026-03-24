package state

import (
	"encoding/json"
	"testing"
)

func TestState_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		state    State
		expected string
	}{
		{"StateIdle", StateIdle, "idle"},
		{"StateRunning", StateRunning, "running"},
		{"StatePaused", StatePaused, "paused"},
		{"StateWaitingApproval", StateWaitingApproval, "waiting_approval"},
		{"StateCompleted", StateCompleted, "completed"},
		{"StateFailed", StateFailed, "failed"},
		{"StateCancelled", StateCancelled, "canceled"},
		{"StateArchived", StateArchived, "archived"},
		{"Unknown", "invalid-state", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.state.String()
			if result != tt.expected {
				t.Errorf("State.String() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestState_MarshalText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		state    State
		expected string
	}{
		{"StateIdle", StateIdle, "idle"},
		{"StateRunning", StateRunning, "running"},
		{"StatePaused", StatePaused, "paused"},
		{"StateWaitingApproval", StateWaitingApproval, "waiting_approval"},
		{"StateCompleted", StateCompleted, "completed"},
		{"StateFailed", StateFailed, "failed"},
		{"StateCancelled", StateCancelled, "canceled"},
		{"StateArchived", StateArchived, "archived"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := tt.state.MarshalText()
			if err != nil {
				t.Errorf("State.MarshalText() unexpected error: %v", err)
			}

			if string(result) != tt.expected {
				t.Errorf("State.MarshalText() = %v, want %v", string(result), tt.expected)
			}
		})
	}
}

func TestState_UnmarshalText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		expected  State
		wantError bool
	}{
		{"idle", "idle", StateIdle, false},
		{"running", "running", StateRunning, false},
		{"paused", "paused", StatePaused, false},
		{"waiting_approval", "waiting_approval", StateWaitingApproval, false},
		{"completed", "completed", StateCompleted, false},
		{"failed", "failed", StateFailed, false},
		{"canceled", "canceled", StateCancelled, false},
		{"archived", "archived", StateArchived, false},
		{"invalid", "invalid", "", true},
		{"empty", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var state State

			err := state.UnmarshalText([]byte(tt.input))
			if (err != nil) != tt.wantError {
				t.Errorf("State.UnmarshalText() error = %v, wantError %v", err, tt.wantError)
			}

			if !tt.wantError && state != tt.expected {
				t.Errorf("State.UnmarshalText() = %v, want %v", state, tt.expected)
			}
		})
	}
}

// statePredicateCase describes a test case for a State predicate method.
type statePredicateCase struct {
	name     string
	state    State
	expected bool
}

func runStatePredicateTests(t *testing.T, cases []statePredicateCase, opName string, op func(*State) bool) {
	t.Helper()

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			state := tt.state
			result := op(&state)

			if result != tt.expected {
				t.Errorf("State.%s() = %v, want %v", opName, result, tt.expected)
			}
		})
	}
}

func TestState_IsTerminal(t *testing.T) {
	t.Parallel()
	runStatePredicateTests(t, []statePredicateCase{
		{"StateIdle", StateIdle, false},
		{"StateRunning", StateRunning, false},
		{"StatePaused", StatePaused, false},
		{"StateWaitingApproval", StateWaitingApproval, false},
		{"StateCompleted", StateCompleted, true},
		{"StateFailed", StateFailed, true},
		{"StateCancelled", StateCancelled, true},
		{"StateArchived", StateArchived, true},
		{"Unknown", "invalid-state", false},
	}, "IsTerminal", (*State).IsTerminal)
}

func TestState_IsActive(t *testing.T) {
	t.Parallel()
	runStatePredicateTests(t, []statePredicateCase{
		{"StateIdle", StateIdle, false},
		{"StateRunning", StateRunning, true},
		{"StatePaused", StatePaused, true},
		{"StateWaitingApproval", StateWaitingApproval, true},
		{"StateCompleted", StateCompleted, false},
		{"StateFailed", StateFailed, false},
		{"StateCancelled", StateCancelled, false},
		{"StateArchived", StateArchived, false},
		{"Unknown", "invalid-state", false},
	}, "IsActive", (*State).IsActive)
}

func TestState_CanTransitionTo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		from     State
		to       State
		expected bool
	}{
		// Idle transitions.
		{"idle to running", StateIdle, StateRunning, true},
		{"idle to archived", StateIdle, StateArchived, true},
		{"idle to paused", StateIdle, StatePaused, false},
		{"idle to completed", StateIdle, StateCompleted, false},

		// Running transitions.
		{"running to paused", StateRunning, StatePaused, true},
		{"running to waiting_approval", StateRunning, StateWaitingApproval, true},
		{"running to completed", StateRunning, StateCompleted, true},
		{"running to failed", StateRunning, StateFailed, true},
		{"running to canceled", StateRunning, StateCancelled, true},
		{"running to idle", StateRunning, StateIdle, false},

		// Paused transitions.
		{"paused to running", StatePaused, StateRunning, true},
		{"paused to canceled", StatePaused, StateCancelled, true},
		{"paused to archived", StatePaused, StateArchived, true},
		{"paused to idle", StatePaused, StateIdle, false},

		// WaitingApproval transitions.
		{"waiting_approval to running", StateWaitingApproval, StateRunning, true},
		{"waiting_approval to canceled", StateWaitingApproval, StateCancelled, true},
		{"waiting_approval to paused", StateWaitingApproval, StatePaused, false},

		// Terminal state transitions.
		{"completed to archived", StateCompleted, StateArchived, true},
		{"failed to archived", StateFailed, StateArchived, true},
		{"canceled to archived", StateCancelled, StateArchived, true},
		{"completed to running", StateCompleted, StateRunning, false},
		{"failed to running", StateFailed, StateRunning, false},

		// Archived transitions.
		{"archived to running", StateArchived, StateRunning, false},
		{"archived to idle", StateArchived, StateIdle, false},
		{"archived to archived", StateArchived, StateArchived, false},

		// Same state transitions.
		{"idle to idle", StateIdle, StateIdle, false},
		{"running to running", StateRunning, StateRunning, false},

		// Terminal to terminal (other than archived).
		{"completed to failed", StateCompleted, StateFailed, false},
		{"failed to completed", StateFailed, StateCompleted, false},
		{"canceled to failed", StateCancelled, StateFailed, false},

		// Unknown state.
		{"unknown to running", "invalid-state", StateRunning, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.from.CanTransitionTo(tt.to)
			if result != tt.expected {
				t.Errorf("State.CanTransitionTo(%v, %v) = %v, want %v", tt.from, tt.to, result, tt.expected)
			}
		})
	}
}

func TestState_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		state State
	}{
		{"StateIdle", StateIdle},
		{"StateRunning", StateRunning},
		{"StatePaused", StatePaused},
		{"StateWaitingApproval", StateWaitingApproval},
		{"StateCompleted", StateCompleted},
		{"StateFailed", StateFailed},
		{"StateCancelled", StateCancelled},
		{"StateArchived", StateArchived},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel(
			// Marshal to JSON.
			)

			data, err := json.Marshal(tt.state)
			if err != nil {
				t.Errorf("json.Marshal() unexpected error: %v", err)
			}

			// Unmarshal from JSON.
			var result State

			err = json.Unmarshal(data, &result)
			if err != nil {
				t.Errorf("json.Unmarshal() unexpected error: %v", err)
			}

			// Verify round trip.
			if result != tt.state {
				t.Errorf("JSON round trip failed: original = %v, result = %v", tt.state, result)
			}
		})
	}
}

func TestState_Concurrency(t *testing.T) {
	t.Parallel()
	// Test concurrent access to state methods.

	done := make(chan bool, 10)

	for range 10 {
		go func() {
			state := StateRunning
			_ = state.String()
			_ = state.IsTerminal()
			_ = state.IsActive()
			_ = state.CanTransitionTo(StateCompleted)
			_, _ = state.MarshalText()

			done <- true
		}()
	}

	// Wait for all goroutines to complete.
	for range 10 {
		<-done
	}
}
