package tasks

// Journey: specs/journeys/JOURNEY-020-agent-task-registry.md
// Journey: specs/journeys/JOURNEY-025-parent-shutdown-cancels-children.md.

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	testTaskID       = "task-1"
	testSpecExplorer = "explorer"
)

func TestRegistry_ListEmpty(t *testing.T) {
	t.Parallel()

	reg := New()
	require.Empty(t, reg.List())
}

func TestRegistry_RegisterListsIDSpecState(t *testing.T) {
	t.Parallel()

	reg := New()
	reg.Register(testTaskID, testSpecExplorer, StateWorking, nil)

	require.Equal(t, []Record{{
		ID:    testTaskID,
		Spec:  testSpecExplorer,
		State: StateWorking,
	}}, reg.List())
}

func TestRegistry_WaitCompletedReturnsImmediately(t *testing.T) {
	t.Parallel()

	reg := New()
	reg.Register(testTaskID, testSpecExplorer, StateCompleted, nil)

	got, err := reg.Wait(t.Context(), testTaskID)
	require.NoError(t, err)
	require.Equal(t, StateCompleted, got.State)
}

func TestRegistry_WaitUnknown(t *testing.T) {
	t.Parallel()

	_, err := New().Wait(t.Context(), testTaskID)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestRegistry_WaitContextCancel(t *testing.T) {
	t.Parallel()

	reg := New()
	reg.Register(testTaskID, testSpecExplorer, StateWorking, stuckHandle{})

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := reg.Wait(ctx, testTaskID)
	require.ErrorIs(t, err, context.Canceled)
}

func TestRegistry_WaitUntilCompleted(t *testing.T) {
	t.Parallel()

	reg := New()
	reg.Register(testTaskID, testSpecExplorer, StateWorking, &flipHandle{})

	got, err := reg.Wait(t.Context(), testTaskID)
	require.NoError(t, err)
	require.Equal(t, StateCompleted, got.State)
}

type flipHandle struct {
	seen bool
}

func (h *flipHandle) Get(context.Context) (string, error) {
	if !h.seen {
		h.seen = true

		return StateWorking, nil
	}

	return StateCompleted, nil
}

func (h *flipHandle) Cancel(context.Context) error { return nil }

func (h *flipHandle) SignalTERM() error { return nil }

func TestRegistry_CancelIgnoresTerminal(t *testing.T) {
	t.Parallel()

	h := &orderHandle{}
	reg := New()
	reg.Register(testTaskID, testSpecExplorer, StateCompleted, h)

	require.NoError(t, reg.Cancel(t.Context(), testTaskID))
	require.Empty(t, h.calls)
}

func TestRegistry_CancelAllWorking(t *testing.T) {
	t.Parallel()

	h := &orderHandle{}
	done := &orderHandle{}
	reg := New()
	reg.Register(testTaskID, testSpecExplorer, StateWorking, h)
	reg.Register("done", testSpecExplorer, StateCompleted, done)

	require.NoError(t, reg.CancelAll(t.Context()))
	require.Equal(t, []string{"cancel", "sigterm"}, h.calls)
	require.Empty(t, done.calls)
}

func TestRegistry_CancelThenSIGTERM(t *testing.T) {
	t.Parallel()

	h := &orderHandle{}
	reg := New()
	reg.Register(testTaskID, testSpecExplorer, StateWorking, h)

	require.NoError(t, reg.Cancel(t.Context(), testTaskID))
	require.Equal(t, []string{"cancel", "sigterm"}, h.calls)

	got, err := reg.Wait(t.Context(), testTaskID)
	require.NoError(t, err)
	require.Equal(t, StateCanceled, got.State)
}

type orderHandle struct {
	calls []string
}

func (h *orderHandle) Get(context.Context) (string, error) {
	return StateCanceled, nil
}

func (h *orderHandle) Cancel(context.Context) error {
	h.calls = append(h.calls, "cancel")

	return nil
}

func (h *orderHandle) SignalTERM() error {
	h.calls = append(h.calls, "sigterm")

	return nil
}

type stuckHandle struct{}

func (stuckHandle) Get(ctx context.Context) (string, error) {
	<-ctx.Done()

	return "", fmt.Errorf("stuck get: %w", ctx.Err())
}

func (stuckHandle) Cancel(context.Context) error { return nil }

func (stuckHandle) SignalTERM() error { return nil }
