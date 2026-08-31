package prompt_test

// Journey: specs/journeys/JOURNEY-002-discover-skill-catalog.md.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/agent/prompt"
	"github.com/dmytrogajewski/spin/internal/skills"
)

const (
	catalogSkillName        = "leak-body"
	catalogSkillDescription = "Metadata-only catalog fixture."
	catalogBodyHeading      = "## MustNotAppearInCatalog"
)

func TestSkillCatalogSection_MetadataOnly(t *testing.T) {
	t.Parallel()

	section := prompt.SkillCatalogSection(skills.Catalog{
		{
			Name:        catalogSkillName,
			Description: catalogSkillDescription,
			Location:    "/tmp/leak-body",
			Source:      skills.SourceProject,
		},
	})

	require.Equal(t, prompt.SectionSkillCatalog, section.Name)
	require.Contains(t, section.Template, catalogSkillName)
	require.Contains(t, section.Template, catalogSkillDescription)
	require.NotContains(t, section.Template, catalogBodyHeading)
	require.NotContains(t, section.Template, "/tmp/leak-body")
	require.NotContains(t, section.Template, string(skills.SourceProject))
}

func TestApplyCatalog_EmptyOmitsSection(t *testing.T) {
	t.Parallel()

	composer := prompt.NewComposer()
	prompt.ApplyCatalog(composer, nil)

	require.Empty(t, composer.Compose(nonGitEnv()))
}

func TestApplyCatalog_RendersNames(t *testing.T) {
	t.Parallel()

	composer := prompt.NewComposer()
	prompt.ApplyCatalog(composer, skills.Catalog{
		{Name: catalogSkillName, Description: catalogSkillDescription},
	})

	composed := composer.Compose(nonGitEnv())
	require.Contains(t, composed, catalogSkillName)
	require.Contains(t, composed, catalogSkillDescription)
	require.NotContains(t, composed, catalogBodyHeading)
}

func TestApplyCatalog_DiscoverFixtureOmitsBodyHeading(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	skillDir := filepath.Join(workDir, ".agents", "skills", catalogSkillName)
	require.NoError(t, os.MkdirAll(skillDir, 0o750))

	content := "---\nname: " + catalogSkillName + "\ndescription: " + catalogSkillDescription +
		"\n---\n\n" + catalogBodyHeading + "\n\nSecret body.\n"
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, skills.FileName), []byte(content), 0o600))

	catalog := skills.Discover(skills.Options{WorkDir: workDir})
	require.Len(t, catalog, 1)

	composer := prompt.NewComposer()
	prompt.ApplyCatalog(composer, catalog)

	composed := composer.Compose(nonGitEnv())
	require.Contains(t, composed, catalogSkillName)
	require.Contains(t, composed, catalogSkillDescription)
	require.NotContains(t, composed, catalogBodyHeading)
	require.NotContains(t, composed, "Secret body.")
}
