package child

// Journey: specs/journeys/JOURNEY-019-subagent-hooks-and-no-silent-drop.md.

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/agent/subagent"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/safety/hooks"
)

func TestStartIfAllowed_StartExit2NoPID(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, hooks.EventSubagentStart.ScriptName()),
		[]byte("echo veto\nexit 2\n"),
		0o600,
	))

	proc, err := StartIfAllowed(
		context.Background(),
		hooks.NewRunner(hooks.Config{ProjectDir: dir}),
		"/bin/true",
		"explorer",
		dir,
	)
	require.ErrorIs(t, err, ErrStartBlocked)
	require.Zero(t, processPID(proc))
}

func TestNewExecutor_StopOnSuccess(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	marker := filepath.Join(dir, "stop-success")
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, hooks.EventSubagentStop.ScriptName()),
		[]byte("touch "+marker+"\n"),
		0o600,
	))

	spec, err := lookupExplorer(t)
	require.NoError(t, err)

	_, runErr := NewExecutor(requireSpin(t), dir, nil, hooks.NewRunner(hooks.Config{ProjectDir: dir}))(
		testCtx(t), spec, spawnQuery,
	)
	require.NoError(t, runErr)
	requireMarker(t, marker)
}

func TestNewExecutor_StopOnFailure(t *testing.T) {
	t.Parallel()

	dir, marker := stopHookDir(t, "stop-fail")
	spec, err := lookupExplorer(t)
	require.NoError(t, err)

	_, runErr := NewExecutor("/bin/false", dir, nil, hooks.NewRunner(hooks.Config{ProjectDir: dir}))(
		context.Background(), spec, spawnQuery,
	)
	require.Error(t, runErr)
	requireMarker(t, marker)
}

func TestNewExecutor_StopOnCrash(t *testing.T) {
	t.Parallel()

	dir, marker := stopHookDir(t, "stop-crash")
	spec, err := lookupExplorer(t)
	require.NoError(t, err)

	_, runErr := NewExecutor("/bin/sh", dir, nil, hooks.NewRunner(hooks.Config{ProjectDir: dir}))(
		context.Background(), spec, spawnQuery,
	)
	require.Error(t, runErr)
	requireMarker(t, marker)
}

func TestNewExecutor_StopNotOnVeto(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	marker := filepath.Join(dir, "stop-veto")
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, hooks.EventSubagentStart.ScriptName()),
		[]byte("echo veto\nexit 2\n"),
		0o600,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, hooks.EventSubagentStop.ScriptName()),
		[]byte("touch "+marker+"\n"),
		0o600,
	))

	spec, err := lookupExplorer(t)
	require.NoError(t, err)

	_, runErr := NewExecutor("/bin/true", dir, nil, hooks.NewRunner(hooks.Config{ProjectDir: dir}))(
		context.Background(), spec, spawnQuery,
	)
	require.ErrorIs(t, runErr, ErrStartBlocked)

	_, statErr := os.Stat(marker)
	require.ErrorIs(t, statErr, os.ErrNotExist)
}

func TestNewExecutor_VetoEmitsHookVetoEvent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, hooks.EventSubagentStart.ScriptName()),
		[]byte("echo veto\nexit 2\n"),
		0o600,
	))

	spec, err := lookupExplorer(t)
	require.NoError(t, err)

	emitter := events.NewEventEmitter(8)
	_, evts, subErr := emitter.Subscribe()
	require.NoError(t, subErr)

	_, runErr := NewExecutor("/bin/true", dir, emitter, hooks.NewRunner(hooks.Config{ProjectDir: dir}))(
		context.Background(), spec, "q",
	)
	require.ErrorIs(t, runErr, ErrStartBlocked)

	got := findEvent(t, drainEvents(evts), events.EventHookVeto)
	data, ok := got.HookVetoData()
	require.True(t, ok)
	require.Contains(t, data.Reason, "veto")
}

func TestNewExecutor_StartVetoNoProcess(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, hooks.EventSubagentStart.ScriptName()),
		[]byte("echo veto\nexit 2\n"),
		0o600,
	))

	spec, err := lookupExplorer(t)
	require.NoError(t, err)

	_, runErr := NewExecutor("/bin/true", dir, nil, hooks.NewRunner(hooks.Config{ProjectDir: dir}))(
		context.Background(), spec, "q",
	)
	require.ErrorIs(t, runErr, ErrStartBlocked)
}

func lookupExplorer(t *testing.T) (*subagent.Spec, error) {
	t.Helper()

	return subagent.Lookup(subagent.NameExplorer)
}

func stopHookDir(t *testing.T, markerName string) (dir, marker string) {
	t.Helper()

	dir = t.TempDir()
	marker = filepath.Join(dir, markerName)
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, hooks.EventSubagentStop.ScriptName()),
		[]byte("touch "+marker+"\n"),
		0o600,
	))

	return dir, marker
}

func requireMarker(t *testing.T, path string) {
	t.Helper()

	require.Eventually(t, func() bool {
		_, err := os.Stat(path)

		return err == nil
	}, 2*time.Second, 20*time.Millisecond, "hook marker %s", path)
}

func processPID(proc *Process) int {
	if proc == nil {
		return 0
	}

	return proc.PID()
}
