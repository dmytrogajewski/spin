package executor_test

// Journey: specs/journeys/JOURNEY-R2.2.md.

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/agent/executor"
	"github.com/dmytrogajewski/spin/internal/safety"
)

func TestPrepareStage_NpmInstall(t *testing.T) {
	t.Parallel()

	// Mutant killed: "npm not rewritten".
	stage := executor.NewPrepareStage()
	pc := executor.NewPipelineContext(&safety.Command{
		Program: "npm",
		Args:    []string{"install"},
		Raw:     "npm install",
	})

	err := stage(context.Background(), pc)

	require.NoError(t, err)
	require.Contains(t, pc.Command.Args, "--yes")
}

func TestPrepareStage_PipInstall(t *testing.T) {
	t.Parallel()

	// Mutant killed: "pip not rewritten".
	stage := executor.NewPrepareStage()
	pc := executor.NewPipelineContext(&safety.Command{
		Program: "pip",
		Args:    []string{"install", "flask"},
		Raw:     "pip install flask",
	})

	err := stage(context.Background(), pc)

	require.NoError(t, err)
	require.Contains(t, pc.Command.Args, "--no-input")
}

func TestPrepareStage_Pip3Install(t *testing.T) {
	t.Parallel()

	// Mutant killed: "pip3 missed".
	stage := executor.NewPrepareStage()
	pc := executor.NewPipelineContext(&safety.Command{
		Program: "pip3",
		Args:    []string{"install", "requests"},
		Raw:     "pip3 install requests",
	})

	err := stage(context.Background(), pc)

	require.NoError(t, err)
	require.Contains(t, pc.Command.Args, "--no-input")
}

func TestPrepareStage_AptInstall(t *testing.T) {
	t.Parallel()

	// Mutant killed: "apt not rewritten".
	stage := executor.NewPrepareStage()
	pc := executor.NewPipelineContext(&safety.Command{
		Program: "apt",
		Args:    []string{"install", "curl"},
		Raw:     "apt install curl",
	})

	err := stage(context.Background(), pc)

	require.NoError(t, err)
	require.Contains(t, pc.Command.Args, "-y")
}

func TestPrepareStage_AptGetInstall(t *testing.T) {
	t.Parallel()

	// Mutant killed: "apt-get missed".
	stage := executor.NewPrepareStage()
	pc := executor.NewPipelineContext(&safety.Command{
		Program: "apt-get",
		Args:    []string{"install", "vim"},
		Raw:     "apt-get install vim",
	})

	err := stage(context.Background(), pc)

	require.NoError(t, err)
	require.Contains(t, pc.Command.Args, "-y")
}

func TestPrepareStage_PythonUnbuffered(t *testing.T) {
	t.Parallel()

	// Mutant killed: "env not set".
	stage := executor.NewPrepareStage()
	pc := executor.NewPipelineContext(&safety.Command{
		Program: "python",
		Args:    []string{"script.py"},
		Raw:     "python script.py",
	})

	err := stage(context.Background(), pc)

	require.NoError(t, err)
	require.Equal(t, "1", pc.Command.Env["PYTHONUNBUFFERED"])
}

func TestPrepareStage_Python3Unbuffered(t *testing.T) {
	t.Parallel()

	// Mutant killed: "python3 missed".
	stage := executor.NewPrepareStage()
	pc := executor.NewPipelineContext(&safety.Command{
		Program: "python3",
		Args:    []string{"-m", "pytest"},
		Raw:     "python3 -m pytest",
	})

	err := stage(context.Background(), pc)

	require.NoError(t, err)
	require.Equal(t, "1", pc.Command.Env["PYTHONUNBUFFERED"])
}

func TestPrepareStage_PytestUnbuffered(t *testing.T) {
	t.Parallel()

	// Mutant killed: "pytest missed".
	stage := executor.NewPrepareStage()
	pc := executor.NewPipelineContext(&safety.Command{
		Program: "pytest",
		Args:    []string{"tests/"},
		Raw:     "pytest tests/",
	})

	err := stage(context.Background(), pc)

	require.NoError(t, err)
	require.Equal(t, "1", pc.Command.Env["PYTHONUNBUFFERED"])
}

func TestPrepareStage_UnknownUnchanged(t *testing.T) {
	t.Parallel()

	// Mutant killed: "rewrites everything".
	stage := executor.NewPrepareStage()
	pc := executor.NewPipelineContext(&safety.Command{
		Program: "ls",
		Args:    []string{"-la"},
		Raw:     "ls -la",
	})

	origArgs := make([]string, len(pc.Command.Args))
	copy(origArgs, pc.Command.Args)

	err := stage(context.Background(), pc)

	require.NoError(t, err)
	require.Equal(t, origArgs, pc.Command.Args)
}

func TestPrepareStage_NpmRunNotRewritten(t *testing.T) {
	t.Parallel()

	// Mutant killed: "non-install rewritten".
	stage := executor.NewPrepareStage()
	pc := executor.NewPipelineContext(&safety.Command{
		Program: "npm",
		Args:    []string{"run", "dev"},
		Raw:     "npm run dev",
	})

	err := stage(context.Background(), pc)

	require.NoError(t, err)
	require.NotContains(t, pc.Command.Args, "--yes")
}

func TestPrepareStage_AlreadyHasFlag(t *testing.T) {
	t.Parallel()

	// Mutant killed: "double flag".
	stage := executor.NewPrepareStage()
	pc := executor.NewPipelineContext(&safety.Command{
		Program: "npm",
		Args:    []string{"install", "--yes"},
		Raw:     "npm install --yes",
	})

	err := stage(context.Background(), pc)

	require.NoError(t, err)

	count := 0

	for _, arg := range pc.Command.Args {
		if arg == "--yes" {
			count++
		}
	}

	require.Equal(t, 1, count, "flag should not be duplicated")
}
