// Journey: specs/journeys/JOURNEY-1.2.md
package executor_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/agent/executor"
	"github.com/dmytrogajewski/spin/internal/safety"
	"github.com/dmytrogajewski/spin/internal/safety/blocklist"
)

func TestBlocklistStage_BlocksForbiddenCommand(t *testing.T) {
	t.Parallel()

	// Mutant killed: "blocklist stage does not halt pipeline".
	checker := blocklist.NewChecker(true)
	stage := executor.NewBlocklistStage(checker)

	pc := executor.NewPipelineContext(&safety.Command{
		Program: "rm",
		Args:    []string{"-rf", "/"},
		Raw:     "rm -rf /",
	})

	err := stage(context.Background(), pc)

	require.NoError(t, err)
	require.True(t, pc.Halted, "pipeline should be halted for dangerous command")
	require.NotNil(t, pc.HaltErr)
	require.ErrorIs(t, pc.HaltErr, executor.ErrCommandBlocklisted)
}

func TestBlocklistStage_AllowsSafeCommand(t *testing.T) {
	t.Parallel()

	// Mutant killed: "blocklist stage halts safe commands".
	checker := blocklist.NewChecker(true)
	stage := executor.NewBlocklistStage(checker)

	pc := executor.NewPipelineContext(&safety.Command{
		Program: "ls",
		Args:    []string{"-la"},
		Raw:     "ls -la",
	})

	err := stage(context.Background(), pc)

	require.NoError(t, err)
	require.False(t, pc.Halted, "pipeline should NOT be halted for safe command")
}

func TestBlocklistStage_NilCheckerNoOp(t *testing.T) {
	t.Parallel()

	// Mutant killed: "nil checker panics".
	stage := executor.NewBlocklistStage(nil)

	pc := executor.NewPipelineContext(&safety.Command{
		Program: "rm",
		Raw:     "rm -rf /",
	})

	err := stage(context.Background(), pc)

	require.NoError(t, err)
	require.False(t, pc.Halted, "nil checker should not halt")
}
