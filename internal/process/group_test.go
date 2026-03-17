// Journey: specs/journeys/JOURNEY-R1.2.md.

//go:build !windows

package process_test

import (
	"context"
	"os/exec"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/process"
)

const cleanupDelay = 100 * time.Millisecond

func TestSetGroup_SetsSetpgid(t *testing.T) {
	t.Parallel()

	// Mutant killed: "no SysProcAttr set".
	cmd := exec.CommandContext(context.Background(), "echo", "hello")

	process.SetGroup(cmd)

	require.NotNil(t, cmd.SysProcAttr)
	require.True(t, cmd.SysProcAttr.Setpgid)
}

func TestSetGroup_PreservesExistingSysProcAttr(t *testing.T) {
	t.Parallel()

	// Mutant killed: "overwrites existing SysProcAttr".
	cmd := exec.CommandContext(context.Background(), "echo", "hello")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Noctty: true,
	}

	process.SetGroup(cmd)

	require.True(t, cmd.SysProcAttr.Setpgid)
	require.True(t, cmd.SysProcAttr.Noctty)
}

func TestKillGroup_KillsChildren(t *testing.T) {
	t.Parallel()

	// Mutant killed: "only parent killed".
	// Spawn a shell that forks a child sleep process in the same group.
	cmd := exec.CommandContext(context.Background(), "/bin/sh", "-c", "sleep 60 & echo $!; wait")

	process.SetGroup(cmd)

	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err)

	err = cmd.Start()
	require.NoError(t, err)

	// Read child PID from stdout.
	buf := make([]byte, 64)
	nr, readErr := stdout.Read(buf)
	require.NoError(t, readErr)
	require.Positive(t, nr)

	// Kill the entire process group.
	killErr := process.KillGroup(cmd)
	require.NoError(t, killErr)

	// Wait for the command to finish (it was killed).
	_ = cmd.Wait()

	// Give OS a moment to clean up.
	time.Sleep(cleanupDelay)

	// Parse the child PID and verify it's dead.
	childPIDStr := string(buf[:nr-1]) // strip newline.
	childPID := parsePID(t, childPIDStr)

	// Sending signal 0 checks if process exists.
	procErr := syscall.Kill(childPID, 0)
	require.ErrorIs(t, procErr, syscall.ESRCH, "child process should be dead")
}

func TestKillGroup_NilProcess(t *testing.T) {
	t.Parallel()

	// Mutant killed: "nil panic".
	cmd := exec.CommandContext(context.Background(), "echo", "hello")
	// Don't start it — Process is nil.

	err := process.KillGroup(cmd)
	require.Error(t, err)
}

func parsePID(tb testing.TB, pidStr string) int {
	tb.Helper()

	pid := 0

	for _, ch := range pidStr {
		if ch < '0' || ch > '9' {
			break
		}

		pid = pid*10 + int(ch-'0')
	}

	require.Positive(tb, pid, "failed to parse PID from: %q", pidStr)

	return pid
}
