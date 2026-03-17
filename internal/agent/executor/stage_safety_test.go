package executor_test

// Journey: specs/journeys/JOURNEY-R2.1.md.

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/agent/executor"
	"github.com/dmytrogajewski/spin/internal/safety"
)

func TestSafetyStage_ForbiddenHalts(t *testing.T) {
	t.Parallel()

	// Mutant killed: "forbidden passes".
	validator := safety.NewValidator()
	stage := executor.NewSafetyStage(validator)
	pc := executor.NewPipelineContext(&safety.Command{
		Program: "rm",
		Args:    []string{"-rf", "/"},
		Raw:     "rm -rf /",
	})

	err := stage(context.Background(), pc)

	// Safety stage should halt on forbidden commands.
	require.NoError(t, err)
	require.True(t, pc.Halted)
}

func TestSafetyStage_SafePasses(t *testing.T) {
	t.Parallel()

	// Mutant killed: "safe blocked".
	validator := safety.NewValidator()
	stage := executor.NewSafetyStage(validator)
	pc := executor.NewPipelineContext(&safety.Command{
		Program: "echo",
		Args:    []string{"hello"},
		Raw:     "echo hello",
	})

	err := stage(context.Background(), pc)

	require.NoError(t, err)
	require.False(t, pc.Halted)
}

func TestSafetyStage_NilValidator(t *testing.T) {
	t.Parallel()

	// Mutant killed: "nil panic".
	stage := executor.NewSafetyStage(nil)
	pc := executor.NewPipelineContext(&safety.Command{
		Program: "echo",
		Raw:     "echo hello",
	})

	err := stage(context.Background(), pc)

	require.NoError(t, err)
	require.False(t, pc.Halted)
}
