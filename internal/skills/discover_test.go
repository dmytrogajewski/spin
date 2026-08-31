package skills_test

// Journey: specs/journeys/JOURNEY-002-discover-skill-catalog.md.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/skills"
)

const (
	catalogDirPerm    = 0o750
	catalogFilePerm   = 0o600
	fixtureSkillAlpha = "alpha"
	fixtureSkillBeta  = "beta"
	fixtureSkillGamma = "gamma"
	descProjectAlpha  = "Project alpha description."
	descUserAlpha     = "User alpha description."
	descUserBeta      = "User beta description."
	descBundledGamma  = "Bundled gamma description."
	bodyLeakHeading   = "## MustNotAppearInCatalog"
	bodyLeakParagraph = "Secret body that must stay out of the catalog."
)

func TestDiscover_EmptyCatalogIsValid(t *testing.T) {
	t.Parallel()

	work := t.TempDir()
	home := t.TempDir()
	bundled := t.TempDir()

	require.NoError(t, os.MkdirAll(filepath.Join(work, ".agents", "skills"), catalogDirPerm))
	require.NoError(t, os.MkdirAll(filepath.Join(home, ".spin", "skills"), catalogDirPerm))

	catalog := skills.Discover(skills.Options{
		WorkDir:    work,
		HomeDir:    home,
		BundledDir: bundled,
	})

	require.Empty(t, catalog)
}

func TestDiscover_MissingRootsIgnored(t *testing.T) {
	t.Parallel()

	catalog := skills.Discover(skills.Options{
		WorkDir:    filepath.Join(t.TempDir(), "missing-work"),
		HomeDir:    filepath.Join(t.TempDir(), "missing-home"),
		BundledDir: filepath.Join(t.TempDir(), "missing-bundled"),
	})

	require.Empty(t, catalog)
}

func TestDiscover_CollisionProjectWinsSourceTag(t *testing.T) {
	t.Parallel()

	work := t.TempDir()
	home := t.TempDir()

	projectBody := bodyLeakHeading + "\n\n" + bodyLeakParagraph
	projectDir := writeCatalogSkill(t, filepath.Join(work, ".agents", "skills"),
		fixtureSkillAlpha, descProjectAlpha, projectBody)
	writeCatalogSkill(t, filepath.Join(home, ".spin", "skills"), fixtureSkillAlpha, descUserAlpha, "user body")

	catalog := skills.Discover(skills.Options{
		WorkDir: work,
		HomeDir: home,
	})

	require.Len(t, catalog, 1)
	require.Equal(t, fixtureSkillAlpha, catalog[0].Name)
	require.Equal(t, skills.SourceProject, catalog[0].Source)
	require.Equal(t, descProjectAlpha, catalog[0].Description)
	require.Equal(t, projectDir, catalog[0].Location)
}

func TestDiscover_SourcesAndDeterministicOrder(t *testing.T) {
	t.Parallel()

	work := t.TempDir()
	home := t.TempDir()
	bundled := t.TempDir()

	writeCatalogSkill(t, filepath.Join(home, ".spin", "skills"), fixtureSkillBeta, descUserBeta, "beta body")
	writeCatalogSkill(t, filepath.Join(home, ".agents", "skills"), "zeta", "User agents zeta.", "zeta body")
	writeCatalogSkill(t, bundled, fixtureSkillGamma, descBundledGamma, "gamma body")
	writeCatalogSkill(t, filepath.Join(work, ".claude", "skills"), "delta", "Claude interop delta.", "delta body")

	catalog := skills.Discover(skills.Options{
		WorkDir:    work,
		HomeDir:    home,
		BundledDir: bundled,
	})

	require.Len(t, catalog, 4)
	require.Equal(t, fixtureSkillBeta, catalog[0].Name)
	require.Equal(t, skills.SourceUser, catalog[0].Source)
	require.Equal(t, "delta", catalog[1].Name)
	require.Equal(t, skills.SourceProject, catalog[1].Source)
	require.Equal(t, fixtureSkillGamma, catalog[2].Name)
	require.Equal(t, skills.SourceBundled, catalog[2].Source)
	require.Equal(t, "zeta", catalog[3].Name)
	require.Equal(t, skills.SourceUser, catalog[3].Source)
}

func TestDiscover_AgentsWinsOverClaude(t *testing.T) {
	t.Parallel()

	work := t.TempDir()
	agentsDir := writeCatalogSkill(t, filepath.Join(work, ".agents", "skills"), fixtureSkillAlpha, descProjectAlpha, "agents body")
	writeCatalogSkill(t, filepath.Join(work, ".claude", "skills"), fixtureSkillAlpha, "Claude alpha description.", "claude body")

	catalog := skills.Discover(skills.Options{WorkDir: work})

	require.Len(t, catalog, 1)
	require.Equal(t, skills.SourceProject, catalog[0].Source)
	require.Equal(t, descProjectAlpha, catalog[0].Description)
	require.Equal(t, agentsDir, catalog[0].Location)
}

func TestDiscover_SkipsInvalidAndNested(t *testing.T) {
	t.Parallel()

	work := t.TempDir()
	root := filepath.Join(work, ".agents", "skills")
	writeCatalogSkill(t, root, fixtureSkillAlpha, descProjectAlpha, "ok")

	require.NoError(t, os.WriteFile(filepath.Join(root, "readme.txt"), []byte("not a skill"), catalogFilePerm))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "broken"), catalogDirPerm))
	nested := filepath.Join(root, fixtureSkillAlpha, "nested")
	writeCatalogSkill(t, nested, "nested", "Nested must not be discovered.", "nested body")

	catalog := skills.Discover(skills.Options{WorkDir: work})

	require.Len(t, catalog, 1)
	require.Equal(t, fixtureSkillAlpha, catalog[0].Name)
}

func TestDiscover_PluginSkillsSourceTag(t *testing.T) {
	t.Parallel()

	work := t.TempDir()
	pluginLoc := filepath.Join(work, "plugin-skill")

	catalog := skills.Discover(skills.Options{
		WorkDir: work,
		PluginSkills: []skills.PluginSkill{
			{
				PluginName:  "sample-plugin",
				Name:        fixtureSkillAlpha,
				Description: descProjectAlpha,
				Location:    pluginLoc,
			},
		},
	})

	require.Len(t, catalog, 1)
	require.Equal(t, fixtureSkillAlpha, catalog[0].Name)
	require.Equal(t, skills.PluginSource("sample-plugin"), catalog[0].Source)
	require.Equal(t, pluginLoc, catalog[0].Location)
}

func TestDiscover_ProjectWinsOverPluginSkill(t *testing.T) {
	t.Parallel()

	work := t.TempDir()
	projectDir := writeCatalogSkill(t, filepath.Join(work, ".agents", "skills"),
		fixtureSkillAlpha, descProjectAlpha, "project body")

	catalog := skills.Discover(skills.Options{
		WorkDir: work,
		PluginSkills: []skills.PluginSkill{
			{
				PluginName:  "sample-plugin",
				Name:        fixtureSkillAlpha,
				Description: descUserAlpha,
				Location:    filepath.Join(work, "plugin-alpha"),
			},
		},
	})

	require.Len(t, catalog, 1)
	require.Equal(t, skills.SourceProject, catalog[0].Source)
	require.Equal(t, descProjectAlpha, catalog[0].Description)
	require.Equal(t, projectDir, catalog[0].Location)
}

func TestDiscover_PluginBeforeBundled(t *testing.T) {
	t.Parallel()

	work := t.TempDir()
	bundled := t.TempDir()
	writeCatalogSkill(t, bundled, fixtureSkillAlpha, descBundledGamma, "bundled body")

	pluginLoc := filepath.Join(work, "plugin-alpha")
	catalog := skills.Discover(skills.Options{
		WorkDir:    work,
		BundledDir: bundled,
		PluginSkills: []skills.PluginSkill{
			{
				PluginName:  "sample-plugin",
				Name:        fixtureSkillAlpha,
				Description: descProjectAlpha,
				Location:    pluginLoc,
			},
		},
	})

	require.Len(t, catalog, 1)
	require.Equal(t, skills.PluginSource("sample-plugin"), catalog[0].Source)
	require.Equal(t, pluginLoc, catalog[0].Location)
}

func TestFormat_OneLinePerSkill(t *testing.T) {
	t.Parallel()

	require.Equal(t, skills.EmptyCatalogMessage, skills.Format(nil))

	got := skills.Format(skills.Catalog{
		{Name: fixtureSkillAlpha, Source: skills.SourceProject, Description: descProjectAlpha},
	})
	require.Equal(t, fixtureSkillAlpha+"  "+string(skills.SourceProject)+"  "+descProjectAlpha, got)

	pluginLine := skills.Format(skills.Catalog{
		{Name: fixtureSkillAlpha, Source: skills.PluginSource("sample-plugin"), Description: descProjectAlpha},
	})
	require.Equal(t, fixtureSkillAlpha+"  plugin:sample-plugin  "+descProjectAlpha, pluginLine)
}

func writeCatalogSkill(t *testing.T, root, name, description, body string) string {
	t.Helper()

	dir := filepath.Join(root, name)
	require.NoError(t, os.MkdirAll(dir, catalogDirPerm))

	content := "---\nname: " + name + "\ndescription: " + description + "\n---\n\n" + body + "\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, skills.FileName), []byte(content), catalogFilePerm))

	return dir
}
