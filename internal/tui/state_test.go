package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAppState_String(t *testing.T) {
	tests := []struct {
		state AppState
		want  string
	}{
		{StateIdle, "idle"},
		{StateWaitingResponse, "waiting_response"},
		{StateToolApproval, "tool_approval"},
		{StateFilePickerOpen, "file_picker_open"},
		{StateBacktrackMode, "backtrack_mode"},
		{StateExiting, "exiting"},
		{AppState(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.state.String()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAppState_CanTransitionTo(t *testing.T) {
	tests := []struct {
		name      string
		from      AppState
		to        AppState
		wantValid bool
	}{
		// From StateIdle
		{"idle to waiting", StateIdle, StateWaitingResponse, true},
		{"idle to file_picker", StateIdle, StateFilePickerOpen, true},
		{"idle to backtrack", StateIdle, StateBacktrackMode, true},
		{"idle to exiting", StateIdle, StateExiting, true},
		{"idle to approval", StateIdle, StateToolApproval, false}, // Invalid
		{"idle to idle", StateIdle, StateIdle, false},

		// From StateWaitingResponse
		{"waiting to idle", StateWaitingResponse, StateIdle, true},
		{"waiting to approval", StateWaitingResponse, StateToolApproval, true},
		{"waiting to exiting", StateWaitingResponse, StateExiting, true},
		{"waiting to file_picker", StateWaitingResponse, StateFilePickerOpen, false}, // Invalid
		{"waiting to backtrack", StateWaitingResponse, StateBacktrackMode, false},

		// From StateToolApproval
		{"approval to waiting", StateToolApproval, StateWaitingResponse, true},
		{"approval to idle", StateToolApproval, StateIdle, true}, // Valid (deny approval with Ctrl+C)
		{"approval to exiting", StateToolApproval, StateExiting, true},
		{"approval to file_picker", StateToolApproval, StateFilePickerOpen, false},

		// From StateFilePickerOpen
		{"file_picker to idle", StateFilePickerOpen, StateIdle, true},
		{"file_picker to exiting", StateFilePickerOpen, StateExiting, true},
		{"file_picker to waiting", StateFilePickerOpen, StateWaitingResponse, false}, // Invalid

		// From StateBacktrackMode
		{"backtrack to idle", StateBacktrackMode, StateIdle, true},
		{"backtrack to exiting", StateBacktrackMode, StateExiting, true},
		{"backtrack to waiting", StateBacktrackMode, StateWaitingResponse, false}, // Invalid

		// From StateExiting (terminal state)
		{"exiting to idle", StateExiting, StateIdle, false},
		{"exiting to waiting", StateExiting, StateWaitingResponse, false},
		{"exiting to approval", StateExiting, StateToolApproval, false},
		{"exiting to exiting", StateExiting, StateExiting, false},

		// Unknown states
		{"unknown to idle", AppState(999), StateIdle, false},
		{"idle to unknown", StateIdle, AppState(999), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.from.CanTransitionTo(tt.to)
			assert.Equal(t, tt.wantValid, got,
				"transition from %s to %s should be valid=%v",
				tt.from, tt.to, tt.wantValid)
		})
	}
}

// TestStateTransitionPaths verifies complete state transition paths
func TestStateTransitionPaths(t *testing.T) {
	// Test a typical flow: Idle → Waiting → Approval → Waiting → Idle
	assert.True(t, StateIdle.CanTransitionTo(StateWaitingResponse))
	assert.True(t, StateWaitingResponse.CanTransitionTo(StateToolApproval))
	assert.True(t, StateToolApproval.CanTransitionTo(StateWaitingResponse))
	assert.True(t, StateWaitingResponse.CanTransitionTo(StateIdle))

	// Test file picker flow: Idle → FilePicker → Idle → Waiting
	assert.True(t, StateIdle.CanTransitionTo(StateFilePickerOpen))
	assert.True(t, StateFilePickerOpen.CanTransitionTo(StateIdle))
	assert.True(t, StateIdle.CanTransitionTo(StateWaitingResponse))

	// Test backtrack flow: Idle → Backtrack → Idle
	assert.True(t, StateIdle.CanTransitionTo(StateBacktrackMode))
	assert.True(t, StateBacktrackMode.CanTransitionTo(StateIdle))

	// Any state can exit
	assert.True(t, StateIdle.CanTransitionTo(StateExiting))
	assert.True(t, StateWaitingResponse.CanTransitionTo(StateExiting))
	assert.True(t, StateToolApproval.CanTransitionTo(StateExiting))
	assert.True(t, StateFilePickerOpen.CanTransitionTo(StateExiting))
	assert.True(t, StateBacktrackMode.CanTransitionTo(StateExiting))
}
