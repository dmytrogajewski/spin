// Journey: specs/journeys/JOURNEY-1.3.md.
package hooks_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/safety/hooks"
)

// writeHookScript creates a hook script file in dir with the given name and content.
func writeHookScript(t *testing.T, dir, name, content string) {
	t.Helper()

	err := os.WriteFile(filepath.Join(dir, name), []byte(content+"\n"), 0o600)
	require.NoError(t, err)
}

func TestHookRunner_SessionStart_ExecutesScript(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	marker := filepath.Join(dir, "session-started")

	writeHookScript(t, dir, "session-start", "touch "+marker)

	runner := hooks.NewRunner(hooks.Config{ProjectDir: dir})

	result := runner.Execute(
		context.Background(),
		hooks.EventSessionStart,
		hooks.EventContext{SessionID: "test-session"},
	)

	require.False(t, result.Blocked, "SESSION_START is non-blocking and should not block")

	// SESSION_START is async, so wait for the marker file.
	require.Eventually(t, func() bool {
		_, statErr := os.Stat(marker)

		return statErr == nil
	}, 2*time.Second, 50*time.Millisecond, "session-start hook should have created marker file")
}

func TestHookRunner_UserPromptSubmit_ExecutesScript(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	marker := filepath.Join(dir, "prompt-submitted")

	writeHookScript(t, dir, "user-prompt-submit", "touch "+marker+"\nexit 0")

	runner := hooks.NewRunner(hooks.Config{ProjectDir: dir})

	result := runner.Execute(
		context.Background(),
		hooks.EventUserPromptSubmit,
		hooks.EventContext{SessionID: "test-session"},
	)

	require.False(t, result.Blocked, "exit 0 should not block")

	_, err := os.Stat(marker)
	require.NoError(t, err, "user-prompt-submit hook should have created marker file")
}

func TestHookRunner_UserPromptSubmit_BlockingExitCode2(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	writeHookScript(t, dir, "user-prompt-submit",
		`echo '{"reason":"prompt rejected by policy","updated_input":"sanitized"}'
exit 2`)

	runner := hooks.NewRunner(hooks.Config{ProjectDir: dir})

	result := runner.Execute(
		context.Background(),
		hooks.EventUserPromptSubmit,
		hooks.EventContext{SessionID: "test-session"},
	)

	require.True(t, result.Blocked, "exit code 2 should block")
	require.Equal(t, "prompt rejected by policy", result.Reason)
	require.Equal(t, "sanitized", result.UpdatedInput)
}

func TestHookRunner_SessionStart_NonBlocking(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	// Script sleeps for 3 seconds — Execute must return almost immediately
	// because SESSION_START is non-blocking (async).
	writeHookScript(t, dir, "session-start", "sleep 3")

	runner := hooks.NewRunner(hooks.Config{ProjectDir: dir})

	start := time.Now()

	result := runner.Execute(
		context.Background(),
		hooks.EventSessionStart,
		hooks.EventContext{SessionID: "test-session"},
	)
	elapsed := time.Since(start)

	require.False(t, result.Blocked, "SESSION_START should never block")
	require.Less(t, elapsed, 500*time.Millisecond,
		"non-blocking SESSION_START should return immediately, not wait for slow hook")
}
