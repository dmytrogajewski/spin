package executor_test

// Journey: specs/journeys/JOURNEY-R2.3.md.

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/agent/executor"
	"github.com/dmytrogajewski/spin/internal/safety"
)

func TestDetectStage_NpmRunDev(t *testing.T) {
	t.Parallel()

	// Mutant killed: "npm dev missed".
	stage := executor.NewDetectStage()
	pc := executor.NewPipelineContext(&safety.Command{
		Program: "npm",
		Args:    []string{"run", "dev"},
		Raw:     "npm run dev",
	})

	err := stage(context.Background(), pc)

	require.NoError(t, err)
	require.True(t, pc.IsServer)
}

func TestDetectStage_YarnStart(t *testing.T) {
	t.Parallel()

	// Mutant killed: "yarn missed".
	stage := executor.NewDetectStage()
	pc := executor.NewPipelineContext(&safety.Command{
		Program: "yarn",
		Args:    []string{"start"},
		Raw:     "yarn start",
	})

	err := stage(context.Background(), pc)

	require.NoError(t, err)
	require.True(t, pc.IsServer)
}

func TestDetectStage_FlaskRun(t *testing.T) {
	t.Parallel()

	// Mutant killed: "flask missed".
	stage := executor.NewDetectStage()
	pc := executor.NewPipelineContext(&safety.Command{
		Program: "flask",
		Args:    []string{"run"},
		Raw:     "flask run",
	})

	err := stage(context.Background(), pc)

	require.NoError(t, err)
	require.True(t, pc.IsServer)
}

func TestDetectStage_UvicornMain(t *testing.T) {
	t.Parallel()

	// Mutant killed: "uvicorn missed".
	stage := executor.NewDetectStage()
	pc := executor.NewPipelineContext(&safety.Command{
		Program: "uvicorn",
		Args:    []string{"main:app"},
		Raw:     "uvicorn main:app",
	})

	err := stage(context.Background(), pc)

	require.NoError(t, err)
	require.True(t, pc.IsServer)
}

func TestDetectStage_GoRun(t *testing.T) {
	t.Parallel()

	// Mutant killed: "go run missed".
	stage := executor.NewDetectStage()
	pc := executor.NewPipelineContext(&safety.Command{
		Program: "go",
		Args:    []string{"run", "."},
		Raw:     "go run .",
	})

	err := stage(context.Background(), pc)

	require.NoError(t, err)
	require.True(t, pc.IsServer)
}

func TestDetectStage_DockerComposeUp(t *testing.T) {
	t.Parallel()

	// Mutant killed: "docker missed".
	stage := executor.NewDetectStage()
	pc := executor.NewPipelineContext(&safety.Command{
		Program: "docker",
		Args:    []string{"compose", "up"},
		Raw:     "docker compose up",
	})

	err := stage(context.Background(), pc)

	require.NoError(t, err)
	require.True(t, pc.IsServer)
}

func TestDetectStage_GoTestNotDetected(t *testing.T) {
	t.Parallel()

	// Mutant killed: "false positive".
	stage := executor.NewDetectStage()
	pc := executor.NewPipelineContext(&safety.Command{
		Program: "go",
		Args:    []string{"test", "./..."},
		Raw:     "go test ./...",
	})

	err := stage(context.Background(), pc)

	require.NoError(t, err)
	require.False(t, pc.IsServer)
}

func TestDetectStage_LsNotDetected(t *testing.T) {
	t.Parallel()

	// Mutant killed: "false positive".
	stage := executor.NewDetectStage()
	pc := executor.NewPipelineContext(&safety.Command{
		Program: "ls",
		Args:    []string{"-la"},
		Raw:     "ls -la",
	})

	err := stage(context.Background(), pc)

	require.NoError(t, err)
	require.False(t, pc.IsServer)
}

func TestDetectStage_AllPatternsCompile(t *testing.T) {
	t.Parallel()

	// Mutant killed: "regex panic".
	require.NotEmpty(t, executor.ServerPatterns)
	require.GreaterOrEqual(t, len(executor.ServerPatterns), executor.ServerPatternCount)
}

func TestDetectStage_PnpmServe(t *testing.T) {
	t.Parallel()

	// Mutant killed: "pnpm missed".
	stage := executor.NewDetectStage()
	pc := executor.NewPipelineContext(&safety.Command{
		Program: "pnpm",
		Args:    []string{"serve"},
		Raw:     "pnpm serve",
	})

	err := stage(context.Background(), pc)

	require.NoError(t, err)
	require.True(t, pc.IsServer)
}
