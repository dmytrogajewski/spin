package child

// Journey: specs/journeys/JOURNEY-020-agent-task-registry.md
// Journey: specs/journeys/JOURNEY-025-parent-shutdown-cancels-children.md.

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/agent/tasks"
	"github.com/dmytrogajewski/spin/internal/protocol/a2a"
)

func TestTaskHandle_GetCancelSIGTERM(t *testing.T) {
	t.Parallel()

	proc := startExplorer(t)
	task, err := proc.SendImmediate(testCtx(t), spawnQuery)
	require.NoError(t, err)

	h := NewTaskHandle(proc, task.ID)
	state, getErr := h.Get(testCtx(t))
	require.NoError(t, getErr)
	require.Equal(t, tasks.StateWorking, state)

	require.NoError(t, h.Cancel(testCtx(t)))

	canceled, cancelGet := proc.GetTask(testCtx(t), task.ID)
	require.NoError(t, cancelGet)
	require.Equal(t, a2a.TaskStateCanceled, canceled.Status.State)
	require.NoError(t, h.SignalTERM())
}

func TestIsNotCancelable(t *testing.T) {
	t.Parallel()

	err := &a2a.RPCError{Code: a2a.CodeTaskNotCancelable, Message: "Task not cancelable"}
	require.True(t, isNotCancelable(err))
	require.False(t, isNotCancelable(a2a.NewRPCError(a2a.CodeTaskNotFound, "missing")))
}

func TestTaskHandle_CancelAlreadyTerminal(t *testing.T) {
	t.Parallel()

	proc := startExplorer(t)
	task, err := proc.SendImmediate(testCtx(t), spawnQuery)
	require.NoError(t, err)

	h := NewTaskHandle(proc, task.ID)
	require.NoError(t, h.Cancel(testCtx(t)))
	require.NoError(t, h.Cancel(testCtx(t)))
}
