package tasks

// Journey: specs/journeys/JOURNEY-020-agent-task-registry.md.

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/session"
)

func TestRegistry_SurvivesNewParentTurn(t *testing.T) {
	t.Parallel()

	sess := session.NewSession(t.TempDir())
	reg := New()
	reg.Bind(sess)
	reg.Register(testTaskID, testSpecExplorer, StateCompleted, nil)

	sess.IncrementTurnCount(1)

	got, err := reg.Wait(t.Context(), testTaskID)
	require.NoError(t, err)
	require.Equal(t, StateCompleted, got.State)
	require.Equal(t, testSpecExplorer, got.Spec)
}

func TestRestore_ListsPersistedRows(t *testing.T) {
	t.Parallel()

	sess := session.NewSession(t.TempDir())
	src := New()
	src.Bind(sess)
	src.Register(testTaskID, testSpecExplorer, StateWorking, nil)

	got := Restore(sess).List()
	require.Equal(t, []Record{{
		ID:    testTaskID,
		Spec:  testSpecExplorer,
		State: StateWorking,
	}}, got)
}

func TestRestore_NilAgentTasksEmpty(t *testing.T) {
	t.Parallel()

	require.Empty(t, Restore(session.NewSession(t.TempDir())).List())
}
