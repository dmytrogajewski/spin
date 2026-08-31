package hooks

// Journey: specs/journeys/JOURNEY-006-load-com-spin-agent-extension-hooks.md.

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunner_PluginHookFiresOnPreToolUse(t *testing.T) {
	t.Parallel()

	pluginRoot := t.TempDir()
	marker := filepath.Join(pluginRoot, "fired")
	script := filepath.Join(pluginRoot, EventPreToolUse.ScriptName())
	require.NoError(t, os.WriteFile(script, []byte("touch fired\nexit 0\n"), 0o600))

	runner := NewRunner(Config{
		PluginScripts: []PluginScript{{
			Name: EventPreToolUse.ScriptName(),
			Path: script,
			Cwd:  pluginRoot,
		}},
	})

	result := runner.Execute(context.Background(), EventPreToolUse, EventContext{SessionID: "plugin"})
	require.False(t, result.Blocked)

	_, err := os.Stat(marker)
	require.NoError(t, err)
}

func TestRunner_PluginHookCwdIsPluginRoot(t *testing.T) {
	t.Parallel()

	pluginRoot := t.TempDir()
	script := filepath.Join(pluginRoot, EventPreToolUse.ScriptName())
	require.NoError(t, os.WriteFile(script, []byte("pwd > pwd.out\nexit 0\n"), 0o600))

	runner := NewRunner(Config{
		PluginScripts: []PluginScript{{
			Name: EventPreToolUse.ScriptName(),
			Path: script,
			Cwd:  pluginRoot,
		}},
	})
	runner.Execute(context.Background(), EventPreToolUse, EventContext{SessionID: "cwd"})

	got, err := os.ReadFile(filepath.Join(pluginRoot, "pwd.out"))
	require.NoError(t, err)

	want, err := filepath.EvalSymlinks(pluginRoot)
	require.NoError(t, err)
	require.Equal(t, want, strings.TrimSpace(string(got)))
}

func TestRunner_PluginHookInheritsEnv(t *testing.T) {
	t.Setenv(pluginHookProbeEnv, pluginHookProbeVal)

	pluginRoot := t.TempDir()
	script := filepath.Join(pluginRoot, EventPreToolUse.ScriptName())
	body := "printf '%s' \"$" + pluginHookProbeEnv + "\" > env.out\nexit 0\n"
	require.NoError(t, os.WriteFile(script, []byte(body), 0o600))

	runner := NewRunner(Config{
		PluginScripts: []PluginScript{{
			Name: EventPreToolUse.ScriptName(),
			Path: script,
			Cwd:  pluginRoot,
		}},
	})
	runner.Execute(context.Background(), EventPreToolUse, EventContext{SessionID: "env"})

	got, err := os.ReadFile(filepath.Join(pluginRoot, "env.out"))
	require.NoError(t, err)
	require.Equal(t, pluginHookProbeVal, string(got))
}

func TestRunner_PluginHookUnknownNameIgnored(t *testing.T) {
	t.Parallel()

	pluginRoot := t.TempDir()
	marker := filepath.Join(pluginRoot, "fired")
	script := filepath.Join(pluginRoot, "not-a-script-name")
	require.NoError(t, os.WriteFile(script, []byte("touch fired\nexit 0\n"), 0o600))

	runner := NewRunner(Config{
		PluginScripts: []PluginScript{{
			Name: "not-a-script-name",
			Path: script,
			Cwd:  pluginRoot,
		}},
	})
	runner.Execute(context.Background(), EventPreToolUse, EventContext{SessionID: "unknown"})

	_, err := os.Stat(marker)
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestRunner_PluginHookExit2Blocks(t *testing.T) {
	t.Parallel()

	pluginRoot := t.TempDir()
	script := filepath.Join(pluginRoot, EventPreToolUse.ScriptName())
	require.NoError(t, os.WriteFile(script, []byte("echo plugin-policy\nexit 2\n"), 0o600))

	runner := NewRunner(Config{
		PluginScripts: []PluginScript{{
			Name: EventPreToolUse.ScriptName(),
			Path: script,
			Cwd:  pluginRoot,
		}},
	})
	result := runner.Execute(context.Background(), EventPreToolUse, EventContext{SessionID: "block"})
	require.True(t, result.Blocked)
	require.Equal(t, "plugin-policy", result.Reason)
}

const (
	pluginHookProbeEnv = "SPIN_HOOK_PROBE"
	pluginHookProbeVal = "visible"
)
