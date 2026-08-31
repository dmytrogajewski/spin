package tools

// Journey: specs/journeys/JOURNEY-020-agent-task-registry.md.

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/agent/tasks"
)

func TestListAgentTasksTool_ListsIDSpecState(t *testing.T) {
	t.Parallel()

	reg := tasks.New()
	reg.Register("task-1", "explorer", tasks.StateWorking, nil)

	result, err := NewListAgentTasksTool(reg).Execute(t.Context(), ToolParameters{})
	require.NoError(t, err)
	require.Contains(t, result.Output, "task-1")
	require.Contains(t, result.Output, "explorer")
	require.Contains(t, result.Output, tasks.StateWorking)
	require.NotContains(t, result.Output, "start_process")
}

func TestWaitAgentTaskTool_Completed(t *testing.T) {
	t.Parallel()

	reg := tasks.New()
	reg.Register("task-1", "explorer", tasks.StateCompleted, nil)

	params, err := FromMap(map[string]any{agentTaskIDParam: "task-1"})
	require.NoError(t, err)

	result, execErr := NewWaitAgentTaskTool(reg).Execute(t.Context(), params)
	require.NoError(t, execErr)
	require.Contains(t, result.Output, tasks.StateCompleted)
}

func TestCancelAgentTaskTool_Cancels(t *testing.T) {
	t.Parallel()

	reg := tasks.New()
	reg.Register("task-1", "explorer", tasks.StateWorking, nil)

	params, err := FromMap(map[string]any{agentTaskIDParam: "task-1"})
	require.NoError(t, err)

	result, execErr := NewCancelAgentTaskTool(reg).Execute(t.Context(), params)
	require.NoError(t, execErr)
	require.Contains(t, result.Output, "task-1")

	got, waitErr := reg.Wait(t.Context(), "task-1")
	require.NoError(t, waitErr)
	require.Equal(t, tasks.StateCanceled, got.State)
}

func TestRegisterAgentTaskTools(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	RegisterAgentTaskTools(reg, tasks.New())

	for _, name := range []string{listAgentTasksName, waitAgentTaskName, cancelAgentTaskName} {
		tool, err := reg.Get(name)
		require.NoError(t, err)
		require.Equal(t, name, tool.Name())
	}
}
