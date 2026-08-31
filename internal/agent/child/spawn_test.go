package child

// Journey: specs/journeys/JOURNEY-018-spawn-process-children-from-the-parent.md.

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/agent/subagent"
	"github.com/dmytrogajewski/spin/internal/protocol/a2a"
)

const spawnQuery = "explore-please"

func TestStartSpec_PIDPositive(t *testing.T) {
	t.Parallel()

	proc := startExplorer(t)
	require.Positive(t, proc.PID())
	require.NotEqual(t, os.Getpid(), proc.PID())
}

func TestProcess_SendReturnsArtifact(t *testing.T) {
	t.Parallel()

	proc := startExplorer(t)
	summary, err := proc.Send(testCtx(t), spawnQuery)
	require.NoError(t, err)
	require.Equal(t, spawnQuery, summary)
	require.Equal(t, a2a.TaskStateCompleted, proc.Task().Status.State)
}

func TestProcess_CrashFailedStderr(t *testing.T) {
	t.Parallel()

	proc, err := Start(testCtx(t), "/bin/sh", []string{"-c", "echo crash-stderr >&2; exit 1"}, "")
	require.NotNil(t, proc)
	t.Cleanup(func() { _ = proc.Close() })
	require.Error(t, err)
	require.Positive(t, proc.PID())
	require.Equal(t, a2a.TaskStateFailed, proc.Task().Status.State)
	require.Contains(t, firstArtifactText(proc.Task()), "crash-stderr")

	_, secondErr := Start(testCtx(t), "/bin/sh", []string{"-c", "echo crash-stderr >&2; exit 1"}, "")
	require.Error(t, secondErr)
}

func TestProcess_StdoutIsNotTUI(t *testing.T) {
	t.Parallel()

	proc := startExplorer(t)
	require.NotEqual(t, os.Stdout, proc.cmd.Stdout)
}

func TestProcess_SendImmediateReturnsWorkingID(t *testing.T) {
	t.Parallel()

	proc := startExplorer(t)
	task, err := proc.SendImmediate(testCtx(t), spawnQuery)
	require.NoError(t, err)
	require.NotEmpty(t, task.ID)
	require.Equal(t, a2a.TaskStateWorking, task.Status.State)
}

func startExplorer(t *testing.T) *Process {
	t.Helper()

	spec, err := subagent.Lookup(subagent.NameExplorer)
	require.NoError(t, err)

	proc, startErr := StartSpec(testCtx(t), requireSpin(t), spec.Name, "")
	require.NoError(t, startErr)
	require.NotNil(t, proc)
	t.Cleanup(func() { _ = proc.Close() })

	return proc
}

func testCtx(t *testing.T) context.Context {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	return ctx
}

func requireSpin(t *testing.T) string {
	t.Helper()

	if bin := os.Getenv(envSpinBin); bin != "" && fileExists(bin) {
		return bin
	}

	if found, ok := FindRepoBinary(); ok {
		return found
	}

	t.Fatal("build/bin/spin not found")

	return ""
}
