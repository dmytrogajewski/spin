// Journey: specs/journeys/JOURNEY-R2.1.md.
package executor_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/agent/executor"
	"github.com/dmytrogajewski/spin/internal/safety"
)

var errStageFailure = errors.New("stage failed")

func TestPipeline_RunsAllStages(t *testing.T) {
	t.Parallel()

	// Mutant killed: "stages skipped".
	var order []int

	stageA := func(_ context.Context, _ *executor.PipelineContext) error {
		order = append(order, 1)

		return nil
	}
	stageB := func(_ context.Context, _ *executor.PipelineContext) error {
		order = append(order, 2)

		return nil
	}

	pp := executor.NewPipeline(stageA, stageB)
	pc := executor.NewPipelineContext(&safety.Command{Program: "echo"})

	err := pp.Run(context.Background(), pc)

	require.NoError(t, err)
	require.Equal(t, []int{1, 2}, order)
}

func TestPipeline_ShortCircuitsOnHalt(t *testing.T) {
	t.Parallel()

	// Mutant killed: "halt ignored".
	called := false

	stageHalt := func(_ context.Context, pc *executor.PipelineContext) error {
		pc.Halt(errStageFailure)

		return nil
	}
	stageAfter := func(_ context.Context, _ *executor.PipelineContext) error {
		called = true

		return nil
	}

	pp := executor.NewPipeline(stageHalt, stageAfter)
	pc := executor.NewPipelineContext(&safety.Command{Program: "echo"})

	err := pp.Run(context.Background(), pc)

	require.NoError(t, err)
	require.False(t, called, "stage after halt should not run")
	require.True(t, pc.Halted)
	require.ErrorIs(t, pc.HaltErr, errStageFailure)
}

func TestPipeline_StopsOnError(t *testing.T) {
	t.Parallel()

	// Mutant killed: "error swallowed".
	called := false

	stageErr := func(_ context.Context, _ *executor.PipelineContext) error {
		return errStageFailure
	}
	stageAfter := func(_ context.Context, _ *executor.PipelineContext) error {
		called = true

		return nil
	}

	pp := executor.NewPipeline(stageErr, stageAfter)
	pc := executor.NewPipelineContext(&safety.Command{Program: "echo"})

	err := pp.Run(context.Background(), pc)

	require.ErrorIs(t, err, errStageFailure)
	require.False(t, called, "stage after error should not run")
}

func TestPipeline_EmptyPipeline(t *testing.T) {
	t.Parallel()

	// Mutant killed: "nil panic on empty".
	pp := executor.NewPipeline()
	pc := executor.NewPipelineContext(&safety.Command{Program: "echo"})

	err := pp.Run(context.Background(), pc)

	require.NoError(t, err)
}

func TestPipelineContext_SetAndGetValue(t *testing.T) {
	t.Parallel()

	// Mutant killed: "values not stored".
	pc := executor.NewPipelineContext(&safety.Command{Program: "echo"})

	pc.SetValue("key", "val")
	got, ok := pc.GetValue("key")

	require.True(t, ok)
	require.Equal(t, "val", got)
}

func TestPipelineContext_GetValueMissing(t *testing.T) {
	t.Parallel()

	// Mutant killed: "missing returns non-nil".
	pc := executor.NewPipelineContext(&safety.Command{Program: "echo"})

	_, ok := pc.GetValue("missing")

	require.False(t, ok)
}
