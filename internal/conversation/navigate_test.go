package conversation

// Journey: specs/journeys/JOURNEY-022-structured-navigation-index.md.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/nav"
	"github.com/dmytrogajewski/spin/internal/plugins"
	"github.com/dmytrogajewski/spin/internal/tools"
)

const (
	navLivePluginName   = "nav-plug"
	navLivePluginSecret = "SECRET_PLUGIN_README_MUST_NOT_LEAK"
)

func TestRegisterNavigate_LivePluginCatalog(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	root := filepath.Join(workDir, plugins.RelPluginsDir, navLivePluginName)
	require.NoError(t, os.MkdirAll(root, 0o750))

	manifest := `{
  "$schema": "https://agent-plugins.org/schemas/1.0.0/plugin.schema.json",
  "name": "nav-plug",
  "description": "Live plugin for navigate tests."
}`
	require.NoError(t, os.WriteFile(filepath.Join(root, plugins.ManifestFile), []byte(manifest), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "README.md"), []byte(navLivePluginSecret), 0o600))

	builder := &Builder{workDir: workDir, cfg: testConfig()}
	registry := tools.NewRegistry()
	builder.registerNavigate(registry, nil)

	tool, err := registry.Get("navigate")
	require.NoError(t, err)

	params, paramErr := tools.FromMap(map[string]any{"kind": string(nav.KindPlugin)})
	require.NoError(t, paramErr)

	result, execErr := tool.Execute(t.Context(), params)
	require.NoError(t, execErr)
	require.True(t, result.Success)
	require.NotContains(t, result.Output, navLivePluginSecret)

	var payload nav.Result
	require.NoError(t, json.Unmarshal([]byte(result.Output), &payload))

	found := false

	for _, record := range payload.Records {
		if record.ID != navLivePluginName {
			continue
		}

		found = true

		require.Equal(t, root, record.Open)
		require.Equal(t, nav.KindPlugin, record.Kind)
	}

	require.True(t, found, "live plugin %s missing from navigate", navLivePluginName)
}
