package child

// Journey: specs/journeys/JOURNEY-019-subagent-hooks-and-no-silent-drop.md.

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/agent/subagent"
	"github.com/dmytrogajewski/spin/internal/safety/hooks"
)

func TestHarness_InheritsParentPluginHookMarker(t *testing.T) {
	t.Parallel()

	pluginRoot := t.TempDir()
	parentMarker := filepath.Join(pluginRoot, "parent-fired")
	childMarker := filepath.Join(pluginRoot, "child-fired")
	script := filepath.Join(pluginRoot, hooks.EventPreToolUse.ScriptName())
	require.NoError(t, os.WriteFile(script, []byte(
		"touch parent-fired\ntouch child-fired\nexit 0\n",
	), 0o600))

	extras := []hooks.PluginScript{{
		Name: hooks.EventPreToolUse.ScriptName(),
		Path: script,
		Cwd:  pluginRoot,
	}}
	parent := hooks.NewRunner(hooks.Config{PluginScripts: extras})
	parent.Execute(context.Background(), hooks.EventPreToolUse, hooks.EventContext{SessionID: "p"})

	_, err := os.Stat(parentMarker)
	require.NoError(t, err)

	spec, lookupErr := subagent.Lookup(subagent.NameExplorer)
	require.NoError(t, lookupErr)

	childHarness := NewHarness(spec)
	childHarness.InheritHookScripts(parent.PluginScripts())
	require.NotEmpty(t, childHarness.HookScripts(), "missing child hooks is a test failure")
	require.Equal(t, extras, childHarness.HookScripts())

	_ = os.Remove(childMarker)
	childRunner := hooks.NewRunner(hooks.Config{PluginScripts: childHarness.HookScripts()})
	childRunner.Execute(context.Background(), hooks.EventPreToolUse, hooks.EventContext{SessionID: "c"})

	_, childErr := os.Stat(childMarker)
	require.NoError(t, childErr)
}
