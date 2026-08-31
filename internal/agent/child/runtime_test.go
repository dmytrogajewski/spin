package child

// Journey: specs/journeys/JOURNEY-025-parent-shutdown-cancels-children.md.

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRuntimeDir_UsesXDG(t *testing.T) {
	base := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", base)

	require.Equal(t, filepath.Join(base, runtimeSubdir), RuntimeDir())
}

func TestRuntimeDir_FallbackTemp(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", "")

	require.True(t, strings.HasSuffix(RuntimeDir(), runtimeSubdir))
}

func TestWritePidFile_CreatesFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, WritePidFile(dir, 42))

	got, err := os.ReadFile(PidPath(dir, 42))
	require.NoError(t, err)
	require.Equal(t, "42\n", string(got))
}

func TestReapStale_RemovesDeadPid(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	const dead = 2147483646

	require.NoError(t, WritePidFile(dir, dead))
	require.NoError(t, ReapStale(dir))

	_, err := os.Stat(PidPath(dir, dead))
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestReapStale_RemovesSiblingSocket(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	const dead = 2147483646

	require.NoError(t, WritePidFile(dir, dead))
	require.NoError(t, os.WriteFile(SockPath(dir, dead), nil, 0o600))
	require.NoError(t, ReapStale(dir))

	_, err := os.Stat(SockPath(dir, dead))
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestReapStale_SignalsLiveOrphan(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cmd := exec.CommandContext(t.Context(), "sleep", "60")
	require.NoError(t, cmd.Start())
	t.Cleanup(func() { _ = cmd.Process.Kill(); _ = cmd.Wait() })

	require.NoError(t, WritePidFile(dir, cmd.Process.Pid))
	require.NoError(t, ReapStale(dir))

	_, err := os.Stat(PidPath(dir, cmd.Process.Pid))
	require.ErrorIs(t, err, os.ErrNotExist)

	done := make(chan error, 1)

	go func() { done <- cmd.Wait() }()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("orphan must exit after reap SIGTERM")
	}
}

func TestReapOnStart_UsesRuntimeDir(t *testing.T) {
	base := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", base)

	const dead = 2147483646

	require.NoError(t, WritePidFile(RuntimeDir(), dead))
	require.NoError(t, ReapOnStart())

	_, err := os.Stat(PidPath(RuntimeDir(), dead))
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestReapStale_MissingDir(t *testing.T) {
	t.Parallel()

	require.NoError(t, ReapStale(filepath.Join(t.TempDir(), "missing")))
}
