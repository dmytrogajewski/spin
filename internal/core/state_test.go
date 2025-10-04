package core

import (
	"encoding/json"
	"testing"
)

func TestStateString(t *testing.T) {
	tests := []struct {
		name  string
		state State
		want  string
	}{
		{"idle", StateIdle, "idle"},
		{"running", StateRunning, "running"},
		{"paused", StatePaused, "paused"},
		{"completed", StateCompleted, "completed"},
		{"failed", StateFailed, "failed"},
		{"cancelled", StateCancelled, "cancelled"},
		{"archived", StateArchived, "archived"},
		{"unknown", State(999), "unknown"},
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
		{"idle", StateIdle, `{"state":"idle"}`},
		{"running", StateRunning, `{"state":"running"}`},
		{"completed", StateCompleted, `{"state":"completed"}`},
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
		{"idle", `{"state":"idle"}`, StateIdle, false},
		{"running", `{"state":"running"}`, StateRunning, false},
		{"paused", `{"state":"paused"}`, StatePaused, false},
		{"completed", `{"state":"completed"}`, StateCompleted, false},
		{"failed", `{"state":"failed"}`, StateFailed, false},
		{"cancelled", `{"state":"cancelled"}`, StateCancelled, false},
		{"archived", `{"state":"archived"}`, StateArchived, false},
		{"invalid", `{"state":"invalid"}`, StateIdle, true},
		{"empty", `{"state":""}`, StateIdle, true},
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
		{"idle is not terminal", StateIdle, false},
		{"running is not terminal", StateRunning, false},
		{"paused is not terminal", StatePaused, false},
		{"completed is terminal", StateCompleted, true},
		{"failed is terminal", StateFailed, true},
		{"cancelled is terminal", StateCancelled, true},
		{"archived is terminal", StateArchived, true},
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
		{"idle is not active", StateIdle, false},
		{"running is active", StateRunning, true},
		{"paused is active", StatePaused, true},
		{"completed is not active", StateCompleted, false},
		{"failed is not active", StateFailed, false},
		{"cancelled is not active", StateCancelled, false},
		{"archived is not active", StateArchived, false},
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
		{"idle to running", StateIdle, StateRunning, true, "can start execution"},
		{"idle to archived", StateIdle, StateArchived, true, "can archive without running"},
		{"idle to completed", StateIdle, StateCompleted, false, "can't complete without running"},
		{"idle to paused", StateIdle, StatePaused, false, "can't pause without running"},

		// From Running
		{"running to paused", StateRunning, StatePaused, true, "can pause execution"},
		{"running to completed", StateRunning, StateCompleted, true, "can complete"},
		{"running to failed", StateRunning, StateFailed, true, "can fail"},
		{"running to cancelled", StateRunning, StateCancelled, true, "can be cancelled"},
		{"running to idle", StateRunning, StateIdle, false, "can't go back to idle"},
		{"running to archived", StateRunning, StateArchived, false, "can't archive while running"},

		// From Paused
		{"paused to running", StatePaused, StateRunning, true, "can resume"},
		{"paused to cancelled", StatePaused, StateCancelled, true, "can cancel"},
		{"paused to archived", StatePaused, StateArchived, true, "can archive"},
		{"paused to completed", StatePaused, StateCompleted, false, "can't complete while paused"},
		{"paused to idle", StatePaused, StateIdle, false, "can't go back to idle"},

		// From Completed
		{"completed to archived", StateCompleted, StateArchived, true, "can archive"},
		{"completed to running", StateCompleted, StateRunning, false, "terminal state"},
		{"completed to idle", StateCompleted, StateIdle, false, "terminal state"},
		{"completed to failed", StateCompleted, StateFailed, false, "terminal state"},

		// From Failed
		{"failed to archived", StateFailed, StateArchived, true, "can archive"},
		{"failed to running", StateFailed, StateRunning, false, "terminal state"},
		{"failed to completed", StateFailed, StateCompleted, false, "terminal state"},

		// From Cancelled
		{"cancelled to archived", StateCancelled, StateArchived, true, "can archive"},
		{"cancelled to running", StateCancelled, StateRunning, false, "terminal state"},
		{"cancelled to idle", StateCancelled, StateIdle, false, "terminal state"},

		// From Archived
		{"archived to running", StateArchived, StateRunning, false, "archived is final"},
		{"archived to idle", StateArchived, StateIdle, false, "archived is final"},
		{"archived to completed", StateArchived, StateCompleted, false, "archived is final"},
		{"archived to archived", StateArchived, StateArchived, false, "already archived"},

		// Self-transitions (generally invalid)
		{"idle to idle", StateIdle, StateIdle, false, "no self-transition"},
		{"running to running", StateRunning, StateRunning, false, "no self-transition"},
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

func TestParseState(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    State
		wantErr bool
	}{
		{"idle", "idle", StateIdle, false},
		{"running", "running", StateRunning, false},
		{"paused", "paused", StatePaused, false},
		{"completed", "completed", StateCompleted, false},
		{"failed", "failed", StateFailed, false},
		{"cancelled", "cancelled", StateCancelled, false},
		{"archived", "archived", StateArchived, false},
		{"invalid", "invalid", StateIdle, true},
		{"empty", "", StateIdle, true},
		{"uppercase", "RUNNING", StateIdle, true},
		{"mixed case", "Running", StateIdle, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseState(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseState() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ParseState() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestState_RoundTrip(t *testing.T) {
	// Test that String() -> ParseState() works correctly
	states := []State{
		StateIdle,
		StateRunning,
		StatePaused,
		StateCompleted,
		StateFailed,
		StateCancelled,
		StateArchived,
	}

	for _, original := range states {
		t.Run(original.String(), func(t *testing.T) {
			str := original.String()
			parsed, err := ParseState(str)
			if err != nil {
				t.Errorf("ParseState() error = %v", err)
				return
			}
			if parsed != original {
				t.Errorf("round trip failed: %v -> %s -> %v",
					original, str, parsed)
			}
		})
	}
}

func TestState_JSONRoundTrip(t *testing.T) {
	// Test that JSON marshaling/unmarshaling works correctly
	type wrapper struct {
		State State `json:"state"`
	}

	states := []State{
		StateIdle,
		StateRunning,
		StatePaused,
		StateCompleted,
		StateFailed,
		StateCancelled,
		StateArchived,
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
