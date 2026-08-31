package tasks

// Journey: specs/journeys/JOURNEY-020-agent-task-registry.md.

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFormat_IncludesIDSpecState(t *testing.T) {
	t.Parallel()

	got := Format([]Record{{
		ID:    testTaskID,
		Spec:  testSpecExplorer,
		State: StateWorking,
	}})
	require.Contains(t, got, testTaskID)
	require.Contains(t, got, testSpecExplorer)
	require.Contains(t, got, StateWorking)
}

func TestFormat_Empty(t *testing.T) {
	t.Parallel()

	require.Equal(t, "No agent tasks.", Format(nil))
}

func TestFormatView_MixedKinds(t *testing.T) {
	t.Parallel()

	got := FormatView(Merge(
		[]Record{{ID: "abc1234", Spec: testSpecExplorer, State: StateWorking}},
		[]ShellSnapshot{{ID: "abc1234", Command: "sleep 300", State: "running"}},
	))
	require.Contains(t, got, "kind=agent")
	require.Contains(t, got, "kind=shell")
	require.Contains(t, got, TypedID(KindAgent, "abc1234"))
	require.Contains(t, got, TypedID(KindShell, "abc1234"))
}
