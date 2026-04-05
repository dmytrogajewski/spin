package concurrency

// Journey: specs/journeys/JOURNEY-S13.md.

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type testState struct {
	steps  []string
	halted bool
}

var errPipelineBoom = errors.New("boom")

func TestPipeline_runs_all_stages(t *testing.T) {
	t.Parallel()

	pipe := NewPipeline(PipelineConfig[testState]{},
		func(_ context.Context, state *testState) error {
			state.steps = append(state.steps, "a")

			return nil
		},
		func(_ context.Context, state *testState) error {
			state.steps = append(state.steps, "b")

			return nil
		},
	)

	state := &testState{}

	err := pipe.Run(context.Background(), state)
	require.NoError(t, err)
	require.Equal(t, []string{"a", "b"}, state.steps)
}

func TestPipeline_stops_on_error(t *testing.T) {
	t.Parallel()

	pipe := NewPipeline(PipelineConfig[testState]{},
		func(_ context.Context, state *testState) error {
			state.steps = append(state.steps, "a")

			return errPipelineBoom
		},
		func(_ context.Context, state *testState) error {
			state.steps = append(state.steps, "b")

			return nil
		},
	)

	state := &testState{}

	err := pipe.Run(context.Background(), state)
	require.ErrorIs(t, err, errPipelineBoom)
	require.Equal(t, []string{"a"}, state.steps)
}

func TestPipeline_stops_on_halt(t *testing.T) {
	t.Parallel()

	pipe := NewPipeline(
		PipelineConfig[testState]{
			Halted: func(state *testState) bool { return state.halted },
		},
		func(_ context.Context, state *testState) error {
			state.steps = append(state.steps, "a")
			state.halted = true

			return nil
		},
		func(_ context.Context, state *testState) error {
			state.steps = append(state.steps, "b")

			return nil
		},
	)

	state := &testState{}

	err := pipe.Run(context.Background(), state)
	require.NoError(t, err)
	require.Equal(t, []string{"a"}, state.steps)
}

func TestPipeline_empty(t *testing.T) {
	t.Parallel()

	pipe := NewPipeline[testState](PipelineConfig[testState]{})

	state := &testState{}

	err := pipe.Run(context.Background(), state)
	require.NoError(t, err)
	require.Empty(t, state.steps)
}
