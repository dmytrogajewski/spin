package core

import (
	"encoding/json"
	"testing"

	"github.com/dmytrogajewski/spin/internal/state"
)

func TestStateString(t *testing.T) {
	tests := []struct {
		name  string
		state State
		want  string
	}{
		{"idle", state.StateIdle, "idle"},
		{"running", state.StateRunning, "running"},
		{"paused", state.StatePaused, "paused"},
		{"completed", state.StateCompleted, "completed"},
		{"failed", state.StateFailed, "failed"},
		{"cancelled", state.StateCancelled, "cancelled"},
		{"archived", state.StateArchived, "archived"},
		{"unknown", state.State(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.state.String(); got != tt.want {
				t.Errorf("State.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestState_MarshalJSON(t *testing.T) {
	type wrapper struct {
		State State `json:"state"`
	}

	tests := []struct {
		name  string
		state State
		want  string
	}{
		{"idle", state.StateIdle, `{"state":"idle"}`},
		{"running", state.StateRunning, `{"state":"running"}`},
		{"completed", state.StateCompleted, `{"state":"completed"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := wrapper{State: tt.state}
			got, err := json.Marshal(w)
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("json.Marshal() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestState_UnmarshalJSON(t *testing.T) {
	type wrapper struct {
		State State `json:"state"`
	}

	tests := []struct {
		name    string
		json    string
		want    State
		wantErr bool
	}{
		{"idle", `{"state":"idle"}`, state.StateIdle, false},
		{"running", `{"state":"running"}`, state.StateRunning, false},
		{"paused", `{"state":"paused"}`, state.StatePaused, false},
		{"completed", `{"state":"completed"}`, state.StateCompleted, false},
		{"failed", `{"state":"failed"}`, state.StateFailed, false},
		{"cancelled", `{"state":"cancelled"}`, state.StateCancelled, false},
		{"archived", `{"state":"archived"}`, state.StateArchived, false},
		{"invalid", `{"state":"invalid"}`, state.StateIdle, true},
		{"empty", `{"state":""}`, state.StateIdle, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var w wrapper
			err := json.Unmarshal([]byte(tt.json), &w)
			if (err != nil) != tt.wantErr {
				t.Errorf("json.Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && w.State != tt.want {
				t.Errorf("json.Unmarshal() state = %v, want %v", w.State, tt.want)
			}
		})
	}
}

func TestState_IsTerminal(t *testing.T) {
	tests := []struct {
		name  string
		state State
		want  bool
	}{
		{"idle is not terminal", state.StateIdle, false},
		{"running is not terminal", state.StateRunning, false},
		{"paused is not terminal", state.StatePaused, false},
		{"completed is terminal", state.StateCompleted, true},
		{"failed is terminal", state.StateFailed, true},
		{"cancelled is terminal", state.StateCancelled, true},
		{"archived is terminal", state.StateArchived, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.state.IsTerminal(); got != tt.want {
				t.Errorf("State.IsTerminal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestState_IsActive(t *testing.T) {
	tests := []struct {
		name  string
		state State
		want  bool
	}{
		{"idle is not active", state.StateIdle, false},
		{"running is active", state.StateRunning, true},
		{"paused is active", state.StatePaused, true},
		{"completed is not active", state.StateCompleted, false},
		{"failed is not active", state.StateFailed, false},
		{"cancelled is not active", state.StateCancelled, false},
		{"archived is not active", state.StateArchived, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.state.IsActive(); got != tt.want {
				t.Errorf("State.IsActive() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestState_CanTransitionTo(t *testing.T) {
	tests := []struct {
		name   string
		from   State
		to     State
		want   bool
		reason string
	}{
		// From Idle
		{"idle to running", state.StateIdle, state.StateRunning, true, "can start execution"},
		{"idle to archived", state.StateIdle, state.StateArchived, true, "can archive without running"},
		{"idle to completed", state.StateIdle, state.StateCompleted, false, "can't complete without running"},
		{"idle to paused", state.StateIdle, state.StatePaused, false, "can't pause without running"},

		// From Running
		{"running to paused", state.StateRunning, state.StatePaused, true, "can pause execution"},
		{"running to completed", state.StateRunning, state.StateCompleted, true, "can complete"},
		{"running to failed", state.StateRunning, state.StateFailed, true, "can fail"},
		{"running to cancelled", state.StateRunning, state.StateCancelled, true, "can be cancelled"},
		{"running to idle", state.StateRunning, state.StateIdle, false, "can't go back to idle"},
		{"running to archived", state.StateRunning, state.StateArchived, false, "can't archive while running"},

		// From Paused
		{"paused to running", state.StatePaused, state.StateRunning, true, "can resume"},
		{"paused to cancelled", state.StatePaused, state.StateCancelled, true, "can cancel"},
		{"paused to archived", state.StatePaused, state.StateArchived, true, "can archive"},
		{"paused to completed", state.StatePaused, state.StateCompleted, false, "can't complete while paused"},
		{"paused to idle", state.StatePaused, state.StateIdle, false, "can't go back to idle"},

		// From Completed
		{"completed to archived", state.StateCompleted, state.StateArchived, true, "can archive"},
		{"completed to running", state.StateCompleted, state.StateRunning, false, "terminal state"},
		{"completed to idle", state.StateCompleted, state.StateIdle, false, "terminal state"},
		{"completed to failed", state.StateCompleted, state.StateFailed, false, "terminal state"},

		// From Failed
		{"failed to archived", state.StateFailed, state.StateArchived, true, "can archive"},
		{"failed to running", state.StateFailed, state.StateRunning, false, "terminal state"},
		{"failed to completed", state.StateFailed, state.StateCompleted, false, "terminal state"},

		// From Cancelled
		{"cancelled to archived", state.StateCancelled, state.StateArchived, true, "can archive"},
		{"cancelled to running", state.StateCancelled, state.StateRunning, false, "terminal state"},
		{"cancelled to idle", state.StateCancelled, state.StateIdle, false, "terminal state"},

		// From Archived
		{"archived to running", state.StateArchived, state.StateRunning, false, "archived is final"},
		{"archived to idle", state.StateArchived, state.StateIdle, false, "archived is final"},
		{"archived to completed", state.StateArchived, state.StateCompleted, false, "archived is final"},
		{"archived to archived", state.StateArchived, state.StateArchived, false, "already archived"},

		// Self-transitions (generally invalid)
		{"idle to idle", state.StateIdle, state.StateIdle, false, "no self-transition"},
		{"running to running", state.StateRunning, state.StateRunning, false, "no self-transition"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.from.CanTransitionTo(tt.to); got != tt.want {
				t.Errorf("State.CanTransitionTo() = %v, want %v (reason: %s)",
					got, tt.want, tt.reason)
			}
		})
	}
}

func TestState_JSONRoundTrip(t *testing.T) {
	// Test that JSON marshaling/unmarshaling works correctly
	type wrapper struct {
		State State `json:"state"`
	}

	states := []state.State{
		state.StateIdle,
		state.StateRunning,
		state.StatePaused,
		state.StateCompleted,
		state.StateFailed,
		state.StateCancelled,
		state.StateArchived,
	}

	for _, original := range states {
		t.Run(original.String(), func(t *testing.T) {
			w := wrapper{State: original}
			data, err := json.Marshal(w)
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}

			var decoded wrapper
			err = json.Unmarshal(data, &decoded)
			if err != nil {
				t.Fatalf("json.Unmarshal() error = %v", err)
			}

			if decoded.State != original {
				t.Errorf("JSON round trip failed: %v -> %s -> %v",
					original, string(data), decoded.State)
			}
		})
	}
}
