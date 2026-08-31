package plugins_test

// Journey: specs/journeys/JOURNEY-005-load-plugins-merge-skills.md.

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/commands"
	"github.com/dmytrogajewski/spin/internal/mcp"
	"github.com/dmytrogajewski/spin/internal/plugins"
	"github.com/dmytrogajewski/spin/internal/skills"
)

const (
	fixtureSample     = "sample-plugin"
	fixtureFailingMCP = "failing-mcp"
	skillGreet        = "greet"
	skillStillHere    = "still-here"
	pluginSampleName  = "sample-plugin"
	pluginFailingName = "failing-mcp"
)

func TestDiscover_SamplePluginSkillsInCatalog(t *testing.T) {
	t.Parallel()

	work := t.TempDir()
	installFixturePlugin(t, work, fixtureSample)

	catalog := isolatedCatalog(t, work, nil)
	entry := requirePluginSkill(t, catalog, skillGreet)
	require.Equal(t, skills.PluginSource(pluginSampleName), entry.Source)
	require.Contains(t, entry.Description, "sample plugin")
}

func TestDiscover_MissingRootsIgnored(t *testing.T) {
	t.Parallel()

	result := plugins.Discover(plugins.DiscoverOptions{
		WorkDir: filepath.Join(t.TempDir(), "missing-work"),
		HomeDir: filepath.Join(t.TempDir(), "missing-home"),
	})
	require.Empty(t, result.Plugins)
	require.Empty(t, result.Skipped)
}

func TestDiscover_FatalPluginJSONSkipsOnlyThatPlugin(t *testing.T) {
	t.Parallel()

	work := t.TempDir()
	installFixturePlugin(t, work, fixtureSample)

	broken := filepath.Join(work, ".spin", "plugins", "broken-plugin")
	require.NoError(t, os.MkdirAll(broken, dirPerm))
	require.NoError(t, os.WriteFile(filepath.Join(broken, plugins.ManifestFile),
		[]byte(`{"name":"broken-plugin"}`), filePerm))

	result := plugins.Discover(plugins.DiscoverOptions{WorkDir: work})
	require.Len(t, result.Plugins, 1)
	require.Equal(t, pluginSampleName, result.Plugins[0].Manifest.Name)
	require.Len(t, result.Skipped, 1)
	require.ErrorIs(t, result.Skipped[0].Err, plugins.ErrInvalidSchema)

	catalog := isolatedCatalog(t, work, nil)
	requirePluginSkill(t, catalog, skillGreet)
	require.False(t, catalogHas(catalog, "broken-plugin"))
}

func TestDiscover_ExtraPaths(t *testing.T) {
	t.Parallel()

	work := t.TempDir()
	extra := filepath.Join("testdata", fixtureSample)

	catalog := isolatedCatalog(t, work, []string{extra})
	entry := requirePluginSkill(t, catalog, skillGreet)
	require.Equal(t, skills.PluginSource(pluginSampleName), entry.Source)
}

func TestDiscover_FailingMCPStillListsSkills(t *testing.T) {
	t.Parallel()

	work := t.TempDir()
	installFixturePlugin(t, work, fixtureFailingMCP)

	result := plugins.Discover(plugins.DiscoverOptions{WorkDir: work})
	require.Len(t, result.Plugins, 1)
	require.True(t, result.Plugins[0].MCPValid)
	require.Len(t, result.Plugins[0].Skills, 1)

	svc := mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default()))

	t.Cleanup(func() { _ = svc.Close() })

	warnings := plugins.AttachMCP(context.Background(), svc, result.Plugins)
	require.NotEmpty(t, warnings)

	catalog := isolatedCatalog(t, work, nil)
	entry := requirePluginSkill(t, catalog, skillStillHere)
	require.Equal(t, skills.PluginSource(pluginFailingName), entry.Source)
}

func TestSkillsCommand_ListsPluginSource(t *testing.T) {
	t.Parallel()

	work := t.TempDir()
	installFixturePlugin(t, work, fixtureSample)

	cmd := &commands.SkillsCommand{}
	result, err := cmd.Execute(context.Background(), nil, skillsCommandCtx{workDir: work})
	require.NoError(t, err)
	require.Contains(t, result, skillGreet)
	require.Contains(t, result, "plugin:"+pluginSampleName)
}

func isolatedCatalog(t *testing.T, work string, extra []string) skills.Catalog {
	t.Helper()

	result := plugins.Discover(plugins.DiscoverOptions{
		WorkDir:    work,
		HomeDir:    t.TempDir(),
		ExtraPaths: extra,
	})

	return skills.Discover(skills.Options{
		WorkDir:      work,
		HomeDir:      t.TempDir(),
		PluginSkills: plugins.SkillContributions(result.Plugins),
	})
}

func requirePluginSkill(t *testing.T, catalog skills.Catalog, name string) skills.Entry {
	t.Helper()

	for _, entry := range catalog {
		if entry.Name == name {
			return entry
		}
	}

	t.Fatalf("skill %q not in catalog: %v", name, catalog)

	return skills.Entry{}
}

func catalogHas(catalog skills.Catalog, name string) bool {
	for _, entry := range catalog {
		if entry.Name == name {
			return true
		}
	}

	return false
}

func installFixturePlugin(t *testing.T, work, fixture string) {
	t.Helper()

	src := filepath.Join(fixtureRoot, fixture)
	dst := filepath.Join(work, ".spin", "plugins", fixture)
	require.NoError(t, os.CopyFS(dst, os.DirFS(src)))
}

type skillsCommandCtx struct {
	workDir string
}

func (c skillsCommandCtx) GetCurrentMode() string { return "regular" }

func (c skillsCommandCtx) SetMode(context.Context, string) error { return nil }

func (c skillsCommandCtx) GetWorkDir() string { return c.workDir }
