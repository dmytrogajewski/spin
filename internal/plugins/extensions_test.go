package plugins_test

// Journey: specs/journeys/JOURNEY-006-load-com-spin-agent-extension-hooks.md.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/plugins"
	"github.com/dmytrogajewski/spin/internal/safety/hooks"
)

func TestDiscoverAgentHooks_ExtensionsHooksPath(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	name := hooks.EventPreToolUse.ScriptName()
	scriptRel := "./scripts/" + name
	require.NoError(t, os.WriteFile(filepath.Join(root, plugins.ManifestFile),
		spinAgentHooksManifest("ext-hooks", `{"`+name+`":"`+scriptRel+`"}`), filePerm))
	require.NoError(t, os.Mkdir(filepath.Join(root, "scripts"), dirPerm))
	require.NoError(t, os.WriteFile(filepath.Join(root, "scripts", name), []byte("exit 0\n"), filePerm))

	plugin, err := plugins.Load(root)
	require.NoError(t, err)

	found := plugins.DiscoverAgentHooks(plugin)
	require.Len(t, found, 1)
	require.Equal(t, name, found[0].ScriptName)
	require.Equal(t, plugin.Root, found[0].Cwd)
}

func TestDiscoverAgentHooks_ExtensionsHooksDir(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	name := hooks.EventPreToolUse.ScriptName()

	require.NoError(t, os.WriteFile(filepath.Join(root, plugins.ManifestFile),
		spinAgentHooksManifest("ext-dir", `"./alt-hooks"`), filePerm))
	require.NoError(t, os.Mkdir(filepath.Join(root, "alt-hooks"), dirPerm))
	require.NoError(t, os.WriteFile(filepath.Join(root, "alt-hooks", name), []byte("exit 0\n"), filePerm))

	plugin, err := plugins.Load(root)
	require.NoError(t, err)

	found := plugins.DiscoverAgentHooks(plugin)
	require.Len(t, found, 1)
	require.Equal(t, name, found[0].ScriptName)
}

func TestDiscoverAgentHooks_ExtensionsEscapeSkipped(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	name := hooks.EventPreToolUse.ScriptName()

	require.NoError(t, os.WriteFile(filepath.Join(root, plugins.ManifestFile),
		spinAgentHooksManifest("escape-hooks", `{"`+name+`":"./../secret"}`), filePerm))

	plugin, err := plugins.Load(root)
	require.NoError(t, err)
	require.Empty(t, plugins.DiscoverAgentHooks(plugin))
}

func TestDiscoverAgentHooks_SymlinkEscapeSkipped(t *testing.T) {
	t.Parallel()

	outside := t.TempDir()
	outsideScript := filepath.Join(outside, hooks.EventPreToolUse.ScriptName())
	require.NoError(t, os.WriteFile(outsideScript, []byte("exit 0\n"), filePerm))

	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, plugins.ManifestFile), minimalManifest("symlink-hook"), filePerm))
	dir := filepath.Join(root, filepath.FromSlash(plugins.SpinAgentHooksDir))
	require.NoError(t, os.MkdirAll(dir, dirPerm))
	require.NoError(t, os.Symlink(outsideScript, filepath.Join(dir, hooks.EventPreToolUse.ScriptName())))

	plugin, err := plugins.Load(root)
	require.NoError(t, err)
	require.Empty(t, plugins.DiscoverAgentHooks(plugin))
}

func TestDiscoverAgentHooks_TopLevelHooksKeyIgnored(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	payload := `{"$schema":"` + plugins.SchemaV1 + `","name":"top-hooks","hooks":{"pre-tool-use":"./hooks/pre-tool-use"}}`
	require.NoError(t, os.WriteFile(filepath.Join(root, plugins.ManifestFile), []byte(payload), filePerm))

	plugin, err := plugins.Load(root)
	require.NoError(t, err)
	require.Contains(t, plugin.Manifest.UnknownFields, "hooks")
	require.Empty(t, plugins.DiscoverAgentHooks(plugin))
}

func spinAgentHooksManifest(name, hooksJSON string) []byte {
	return []byte(`{"$schema":"` + plugins.SchemaV1 + `","name":"` + name +
		`","extensions":{"` + plugins.SpinAgentExtension + `":{"hooks":` + hooksJSON + `}}}`)
}

func TestDiscoverAgentHooks_FiresOnPreToolUseRunner(t *testing.T) {
	t.Parallel()

	root := writeHookPlugin(t, "hook-fire", map[string]string{
		hooks.EventPreToolUse.ScriptName(): "touch fired\nexit 0",
	}, nil)
	plugin, err := plugins.Load(root)
	require.NoError(t, err)

	runner := hooks.NewRunner(hooks.Config{
		PluginScripts: plugins.HookScripts([]plugins.Plugin{plugin}),
	})
	result := runner.Execute(t.Context(), hooks.EventPreToolUse, hooks.EventContext{SessionID: "plugin"})
	require.False(t, result.Blocked)

	_, statErr := os.Stat(filepath.Join(plugin.Root, "fired"))
	require.NoError(t, statErr)
}

func TestDiscoverAgentHooks_FindsPreToolUse(t *testing.T) {
	t.Parallel()

	root := writeHookPlugin(t, "hook-pre-tool", map[string]string{
		hooks.EventPreToolUse.ScriptName(): "exit 0",
	}, nil)
	plugin, err := plugins.Load(root)
	require.NoError(t, err)

	found := plugins.DiscoverAgentHooks(plugin)
	require.Len(t, found, 1)
	require.Equal(t, hooks.EventPreToolUse.ScriptName(), found[0].ScriptName)
	require.Equal(t, plugin.Root, found[0].Cwd)
	require.Equal(t, filepath.Join(plugin.Root, plugins.SpinAgentHooksDir, hooks.EventPreToolUse.ScriptName()), found[0].Path)
}

func TestDiscoverAgentHooks_IgnoresUnknownFilename(t *testing.T) {
	t.Parallel()

	root := writeHookPlugin(t, "hook-suffix", map[string]string{
		hooks.EventPreToolUse.ScriptName() + ".sh": "exit 0",
	}, nil)
	plugin, err := plugins.Load(root)
	require.NoError(t, err)
	require.Empty(t, plugins.DiscoverAgentHooks(plugin))
}

func TestDiscoverAgentHooks_AllScriptNames(t *testing.T) {
	t.Parallel()

	scripts := make(map[string]string, len(hooks.AllEvents()))
	for _, event := range hooks.AllEvents() {
		scripts[event.ScriptName()] = "exit 0"
	}

	root := writeHookPlugin(t, "hook-all", scripts, nil)
	plugin, err := plugins.Load(root)
	require.NoError(t, err)

	found := plugins.DiscoverAgentHooks(plugin)
	require.Len(t, found, len(hooks.AllEvents()))

	for i, event := range hooks.AllEvents() {
		require.Equal(t, event.ScriptName(), found[i].ScriptName)
	}
}

func TestDiscoverAgentHooks_IgnoresForeignExtensionDir(t *testing.T) {
	t.Parallel()

	root := writeHookPlugin(t, "foreign-hooks", nil, map[string]string{
		hooks.EventPreToolUse.ScriptName(): "exit 0",
	})
	plugin, err := plugins.Load(root)
	require.NoError(t, err)
	require.Empty(t, plugins.DiscoverAgentHooks(plugin))
}

func writeHookPlugin(t *testing.T, name string, spinHooks, foreignHooks map[string]string) string {
	t.Helper()

	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, plugins.ManifestFile), minimalManifest(name), filePerm))
	writeHookDir(t, root, plugins.SpinAgentHooksDir, spinHooks)
	writeHookDir(t, root, "com.example.client/hooks", foreignHooks)

	return root
}

func writeHookDir(t *testing.T, root, rel string, scripts map[string]string) {
	t.Helper()

	if len(scripts) == 0 {
		return
	}

	dir := filepath.Join(root, filepath.FromSlash(rel))
	require.NoError(t, os.MkdirAll(dir, dirPerm))

	for name, body := range scripts {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(body+"\n"), filePerm))
	}
}
