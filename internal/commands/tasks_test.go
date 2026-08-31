package commands

// Journey: specs/journeys/JOURNEY-020-agent-task-registry.md.
// Journey: specs/journeys/JOURNEY-021-unified-task-view.md.

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/agent/tasks"
	"github.com/dmytrogajewski/spin/internal/tools"
)

type mockTaskSource struct {
	mockCommandContext

	reg    *tasks.Registry
	shells *tools.ShellAdapter
}

func (m *mockTaskSource) AgentTasks() *tasks.Registry {
	return m.reg
}

func (m *mockTaskSource) ShellTasks() *tools.ShellAdapter {
	return m.shells
}

type stubTaskManager struct {
	snaps  []tools.TaskSnapshot
	killed []string
}

func (s *stubTaskManager) List(context.Context) []tools.TaskSnapshot { return s.snaps }

func (s *stubTaskManager) GetOutput(context.Context, string, int) (string, error) {
	return "", nil
}

func (s *stubTaskManager) Kill(_ context.Context, id string) error {
	s.killed = append(s.killed, id)

	return nil
}

func TestTasksCommand_ListsIDSpecState(t *testing.T) {
	t.Parallel()

	reg := tasks.New()
	reg.Register("task-1", "explorer", tasks.StateWorking, nil)

	out, err := (&TasksCommand{}).Execute(t.Context(), nil, &mockTaskSource{reg: reg})
	require.NoError(t, err)
	require.Contains(t, out, "task-1")
	require.Contains(t, out, "explorer")
	require.Contains(t, out, tasks.StateWorking)
	require.Contains(t, out, "kind=agent")
}

func TestTasksCommand_MixedListShowsBothKinds(t *testing.T) {
	t.Parallel()

	reg := tasks.New()
	reg.Register("abc1234", "explorer", tasks.StateWorking, nil)

	mgr := &stubTaskManager{snaps: []tools.TaskSnapshot{{
		ID: "abc1234", Command: "sleep 300", Status: tools.TaskStatusRunning,
	}}}

	out, err := (&TasksCommand{}).Execute(t.Context(), nil, &mockTaskSource{
		reg: reg, shells: tools.AsShellSource(mgr),
	})
	require.NoError(t, err)
	require.Contains(t, out, "kind=agent")
	require.Contains(t, out, "kind=shell")
	require.Contains(t, out, tasks.TypedID(tasks.KindAgent, "abc1234"))
	require.Contains(t, out, tasks.TypedID(tasks.KindShell, "abc1234"))
}

func TestTaskCommand_CancelShellRowKills(t *testing.T) {
	t.Parallel()

	reg := tasks.New()
	mgr := &stubTaskManager{snaps: []tools.TaskSnapshot{{
		ID: "abc1234", Command: "sleep 300", Status: tools.TaskStatusRunning,
	}}}

	out, err := (&TaskCommand{}).Execute(
		t.Context(),
		[]string{"cancel", tasks.TypedID(tasks.KindShell, "abc1234")},
		&mockTaskSource{reg: reg, shells: tools.AsShellSource(mgr)},
	)
	require.NoError(t, err)
	require.Contains(t, out, "abc1234")
	require.Equal(t, []string{"abc1234"}, mgr.killed)
}

func TestTaskCommand_WaitAndCancel(t *testing.T) {
	t.Parallel()

	reg := tasks.New()
	reg.Register("task-1", "explorer", tasks.StateCompleted, nil)

	waited, err := (&TaskCommand{}).Execute(t.Context(), []string{"wait", "task-1"}, &mockTaskSource{reg: reg})
	require.NoError(t, err)
	require.Contains(t, waited, "task-1")
	require.Contains(t, waited, tasks.StateCompleted)

	typed, typedErr := (&TaskCommand{}).Execute(
		t.Context(), []string{"wait", tasks.TypedID(tasks.KindAgent, "task-1")}, &mockTaskSource{reg: reg},
	)
	require.NoError(t, typedErr)
	require.Contains(t, typed, tasks.StateCompleted)

	reg.Register("task-2", "explorer", tasks.StateWorking, nil)
	canceled, cancelErr := (&TaskCommand{}).Execute(t.Context(), []string{"cancel", "task-2"}, &mockTaskSource{reg: reg})
	require.NoError(t, cancelErr)
	require.Contains(t, canceled, "task-2")
}

func TestTaskCommand_RequiresRegistry(t *testing.T) {
	t.Parallel()

	_, err := (&TasksCommand{}).Execute(t.Context(), nil, &mockCommandContext{})
	require.ErrorIs(t, err, ErrTaskRegistryUnavailable)
}

func TestTaskCommand_WaitContextCancel(t *testing.T) {
	t.Parallel()

	reg := tasks.New()
	reg.Register("task-1", "explorer", tasks.StateWorking, nil)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := (&TaskCommand{}).Execute(ctx, []string{"wait", "task-1"}, &mockTaskSource{reg: reg})
	require.ErrorIs(t, err, context.Canceled)
}
