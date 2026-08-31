package hooks

// Journey: specs/journeys/JOURNEY-007-finish-the-hook-runner-contract.md.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultGlobalDir_ExpandsHome(t *testing.T) {
	t.Parallel()

	home, err := os.UserHomeDir()
	require.NoError(t, err)

	got := DefaultGlobalDir()
	require.Equal(t, filepath.Join(home, ".spin", "hooks"), got)
	require.NotContains(t, got, "~", "expanded global dir must not contain ~")
}

func TestNewRunner_ExpandsTildeGlobalDir(t *testing.T) {
	t.Parallel()

	home, err := os.UserHomeDir()
	require.NoError(t, err)

	runner := NewRunner(Config{GlobalDir: defaultGlobalHooksRel})
	require.Equal(t, filepath.Join(home, ".spin", "hooks"), runner.globalDir)
}

func TestRunner_Exit0JSONUpdatedInput(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeScript(t, dir, EventPreToolUse.ScriptName(),
		`echo '{"updated_input":"{\"path\":\"safe\"}"}'
exit 0`)

	result := NewRunner(Config{ProjectDir: dir}).Execute(
		t.Context(),
		EventPreToolUse,
		EventContext{SessionID: "rewrite"},
	)

	require.False(t, result.Blocked)
	require.JSONEq(t, `{"path":"safe"}`, result.UpdatedInput)
}

func TestRunner_LastUpdatedInputWins(t *testing.T) {
	t.Parallel()

	global := t.TempDir()
	project := t.TempDir()
	writeScript(t, global, EventPreToolUse.ScriptName(),
		`echo '{"updated_input":"{\"path\":\"from-global\"}"}'
exit 0`)
	writeScript(t, project, EventPreToolUse.ScriptName(),
		`echo '{"updated_input":"{\"path\":\"from-project\"}"}'
exit 0`)

	result := NewRunner(Config{GlobalDir: global, ProjectDir: project}).Execute(
		t.Context(),
		EventPreToolUse,
		EventContext{SessionID: "chain"},
	)

	require.False(t, result.Blocked)
	require.JSONEq(t, `{"path":"from-project"}`, result.UpdatedInput)
}

func TestRunner_UpdatedInputJSONObject(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeScript(t, dir, EventPreToolUse.ScriptName(),
		`echo '{"updated_input":{"path":"safe"}}'
exit 0`)

	result := NewRunner(Config{ProjectDir: dir}).Execute(
		t.Context(),
		EventPreToolUse,
		EventContext{SessionID: "object"},
	)

	require.False(t, result.Blocked)
	require.JSONEq(t, `{"path":"safe"}`, result.UpdatedInput)
}
