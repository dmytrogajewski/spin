package child

// Journey: specs/journeys/JOURNEY-020-agent-task-registry.md.

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/agent/subagent"
	"github.com/dmytrogajewski/spin/internal/agent/tasks"
)

func TestImmediateStarter_ReturnsWorkingID(t *testing.T) {
	t.Parallel()

	spec, err := subagent.Lookup(subagent.NameExplorer)
	require.NoError(t, err)

	id, handle, startErr := ImmediateStarter(requireSpin(t), "", nil)(testCtx(t), spec, spawnQuery)
	require.NoError(t, startErr)
	require.NotEmpty(t, id)
	require.NotNil(t, handle)

	state, getErr := handle.Get(testCtx(t))
	require.NoError(t, getErr)
	require.Equal(t, tasks.StateWorking, state)
	require.NoError(t, handle.SignalTERM())
}
