// Journey: specs/journeys/JOURNEY-1.1.md
package executor_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/agent/executor"
	"github.com/dmytrogajewski/spin/internal/safety"
)

// errForbidden is a test sentinel for pipeline halts.
var errForbidden = errors.New("forbidden command")

// stubCommandInfo implements tools.CommandInfo for testing.
type stubCommandInfo struct {
	program string
	args    []string
	raw     string
	workDir string
}

func (s *stubCommandInfo) GetProgram() string { return s.program }
func (s *stubCommandInfo) GetArgs() []string  { return s.args }
func (s *stubCommandInfo) GetRaw() string     { return s.raw }
func (s *stubCommandInfo) GetWorkDir() string { return s.workDir }

// recordingExecutor records whether Execute was called.
type recordingExecutor struct {
	called bool
}

func (r *recordingExecutor) Execute(_ context.Context, _ *safety.Command, _ any) (*executor.CommandResult, error) {
	r.called = true

	return &executor.CommandResult{Stdout: "ok", ExitCode: 0}, nil
}

func TestAdapter_RunsPipelineBeforeExecution(t *testing.T) {
	t.Parallel()

	// Mutant killed: "pipeline stages skipped when set".
	var stageRan bool

	stage := func(_ context.Context, _ *executor.PipelineContext) error {
		stageRan = true

		return nil
	}

	rec := &recordingExecutor{}
	pipeline := executor.NewPipeline(stage)
	adapter := executor.NewAdapterWithPipeline(rec, pipeline)

	cmd := &stubCommandInfo{program: "echo", raw: "echo hello"}
	_, err := adapter.Execute(context.Background(), cmd, nil)

	require.NoError(t, err)
	require.True(t, stageRan, "pipeline stage should have run")
	require.True(t, rec.called, "executor should have been called after pipeline")
}

func TestAdapter_HaltedPipelineSkipsExecution(t *testing.T) {
	t.Parallel()

	// Mutant killed: "halted pipeline still executes command".
	haltStage := func(_ context.Context, pc *executor.PipelineContext) error {
		pc.Halt(errForbidden)

		return nil
	}

	rec := &recordingExecutor{}
	pipeline := executor.NewPipeline(haltStage)
	adapter := executor.NewAdapterWithPipeline(rec, pipeline)

	cmd := &stubCommandInfo{program: "rm", raw: "rm -rf /"}
	_, err := adapter.Execute(context.Background(), cmd, nil)

	require.Error(t, err)
	require.ErrorIs(t, err, errForbidden)
	require.False(t, rec.called, "executor should NOT be called when pipeline halts")
}

func TestAdapter_NilPipelineExecutesDirectly(t *testing.T) {
	t.Parallel()

	// Mutant killed: "nil pipeline panics".
	rec := &recordingExecutor{}
	adapter := executor.NewAdapterWithPipeline(rec, nil)

	cmd := &stubCommandInfo{program: "ls", raw: "ls"}
	result, err := adapter.Execute(context.Background(), cmd, nil)

	require.NoError(t, err)
	require.True(t, rec.called)
	require.Equal(t, "ok", result.GetStdout())
}
