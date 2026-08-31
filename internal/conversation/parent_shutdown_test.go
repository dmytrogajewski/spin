package conversation

// Journey: specs/journeys/JOURNEY-025-parent-shutdown-cancels-children.md.

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/agent/tasks"
	"github.com/dmytrogajewski/spin/internal/safety/hooks"
)

type shutdownHandle struct {
	calls []string
}

func (h *shutdownHandle) Get(context.Context) (string, error) {
	return tasks.StateWorking, nil
}

func (h *shutdownHandle) Cancel(context.Context) error {
	h.calls = append(h.calls, "cancel")

	return nil
}

func (h *shutdownHandle) SignalTERM() error {
	h.calls = append(h.calls, "sigterm")

	return nil
}

func TestClose_CancelsRunningTasksThenSessionEnd(t *testing.T) {
	t.Parallel()

	h := &shutdownHandle{}
	rec := &recordHooks{}
	reg := tasks.New()
	reg.Register("t1", "explorer", tasks.StateWorking, h)
	conv := &Conversation{hookRunner: rec, taskRegistry: reg, id: "sess-shut", workDir: t.TempDir()}

	require.NoError(t, conv.Close(context.Background()))
	require.Equal(t, []string{"cancel", "sigterm"}, h.calls)
	require.Equal(t, []hooks.Event{hooks.EventStop, hooks.EventSessionEnd}, rec.events)
}

func TestClose_CanceledContextStillCancelsTasks(t *testing.T) {
	t.Parallel()

	h := &shutdownHandle{}
	reg := tasks.New()
	reg.Register("t1", "explorer", tasks.StateWorking, h)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	require.NoError(t, (&Conversation{taskRegistry: reg}).Close(ctx))
	require.Equal(t, []string{"cancel", "sigterm"}, h.calls)
}
