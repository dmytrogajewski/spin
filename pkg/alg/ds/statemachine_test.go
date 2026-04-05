package ds

// Journey: specs/journeys/JOURNEY-R-REF-25.md.

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStateMachine_CanTransition(t *testing.T) {
	t.Parallel()

	sm := NewStateMachine(map[string]map[string]bool{
		"idle":    {"running": true},
		"running": {"done": true, "failed": true},
	})

	require.True(t, sm.CanTransition("idle", "running"))
	require.True(t, sm.CanTransition("running", "done"))
	require.False(t, sm.CanTransition("idle", "done"))       // Not allowed.
	require.False(t, sm.CanTransition("idle", "idle"))       // Same state.
	require.False(t, sm.CanTransition("unknown", "running")) // Unknown source.
}

func TestStateMachine_Validate(t *testing.T) {
	t.Parallel()

	sm := NewStateMachine(map[string]map[string]bool{
		"a": {"b": true},
	})

	require.NoError(t, sm.Validate("a", "b"))
	require.ErrorIs(t, sm.Validate("a", "c"), ErrInvalidTransition)
}
