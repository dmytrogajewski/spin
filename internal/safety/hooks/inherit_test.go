package hooks

// Journey: specs/journeys/JOURNEY-019-subagent-hooks-and-no-silent-drop.md.

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunner_PluginScriptsExported(t *testing.T) {
	t.Parallel()

	extras := []PluginScript{{
		Name: EventPreToolUse.ScriptName(),
		Path: "/plugin/pre-tool-use",
		Cwd:  "/plugin",
	}}
	runner := NewRunner(Config{PluginScripts: extras})
	require.Equal(t, extras, runner.PluginScripts())
}

func TestCopyScripts_PreservesPluginAndSkillExtras(t *testing.T) {
	t.Parallel()

	src := []PluginScript{
		{Name: EventPreToolUse.ScriptName(), Path: "/plugin/pre-tool-use", Cwd: "/plugin"},
		{Name: EventSessionStart.ScriptName(), Path: "/skill/session-start", Cwd: "/skill"},
	}

	got := CopyScripts(src)
	require.Equal(t, src, got)
	got[0].Cwd = "/mutated"
	require.Equal(t, "/plugin", src[0].Cwd)
}
