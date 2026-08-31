package subagent

// Journey: specs/journeys/JOURNEY-020-agent-task-registry.md.

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/agent/tasks"
)

const testBackgroundID = "task-bg-1"

func TestSpawnBackground_ReturnsIDWithoutWaiting(t *testing.T) {
	t.Parallel()

	gate := make(chan struct{})

	t.Cleanup(func() { close(gate) })

	mgr := NewManager(echoExecutor, 1)
	mgr.SetBackgroundStarter(func(_ context.Context, _ *Spec, _ string) (string, tasks.Handle, error) {
		return testBackgroundID, &holdHandle{gate: gate}, nil
	})

	id, err := mgr.SpawnBackground(t.Context(), NameExplorer, testQuery, tasks.New())
	require.NoError(t, err)
	require.Equal(t, testBackgroundID, id)
}

type holdHandle struct {
	gate <-chan struct{}
}

func (h *holdHandle) Get(ctx context.Context) (string, error) {
	select {
	case <-h.gate:
		return tasks.StateCompleted, nil
	case <-ctx.Done():
		return "", fmt.Errorf("hold get: %w", ctx.Err())
	}
}

func (h *holdHandle) Cancel(context.Context) error { return nil }

func (h *holdHandle) SignalTERM() error { return nil }
