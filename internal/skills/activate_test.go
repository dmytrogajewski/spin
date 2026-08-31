package skills_test

// Journey: specs/journeys/JOURNEY-003-activate-skill-body.md.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/skills"
)

const (
	activateSkillAlpha     = "alpha"
	activateSkillBeta      = "beta"
	activateDescAlpha      = "Activate alpha description."
	activateDescBeta       = "Activate beta description."
	activateBodyAlpha      = "# Alpha body\n\nSee [extra](references/extra.md).\n"
	activateBodyBeta       = "# Beta secret body must not leak.\n"
	activateExtraSentinel  = "UNIQUE_EXTRA_SENTINEL_NOT_IN_BODY"
	activateNestedSentinel = "UNIQUE_NESTED_SENTINEL_NOT_IN_EXTRA"
	activateAllowedTools   = "read_file grep"
	activateDirPerm        = 0o750
	activateFilePerm       = 0o600
)

func TestActivate_ReturnsBodyAndRoot(t *testing.T) {
	t.Parallel()

	root := writeActivateSkill(t, t.TempDir(), activateSkillAlpha, activateDescAlpha, activateBodyAlpha, "")
	catalog := skills.Catalog{{
		Name:     activateSkillAlpha,
		Location: root,
		Source:   skills.SourceProject,
	}}

	got, err := skills.Activate(catalog, activateSkillAlpha)
	require.NoError(t, err)
	require.Equal(t, activateSkillAlpha, got.Name)
	require.Equal(t, root, got.Root)
	require.Equal(t, skills.SourceProject, got.Source)
	require.Contains(t, got.Body, "# Alpha body")
	require.Empty(t, got.AllowedTools)
}

func TestActivate_UnknownNameListsCatalogNoBodyLeak(t *testing.T) {
	t.Parallel()

	work := t.TempDir()
	alphaRoot := writeActivateSkill(t, work, activateSkillAlpha, activateDescAlpha, activateBodyAlpha, "")
	betaRoot := writeActivateSkill(t, work, activateSkillBeta, activateDescBeta, activateBodyBeta, "")

	catalog := skills.Catalog{
		{Name: activateSkillAlpha, Location: alphaRoot, Source: skills.SourceProject},
		{Name: activateSkillBeta, Location: betaRoot, Source: skills.SourceUser},
	}

	_, err := skills.Activate(catalog, "missing")
	require.Error(t, err)
	require.ErrorIs(t, err, skills.ErrUnknownSkill)

	var unknown *skills.UnknownSkillError
	require.ErrorAs(t, err, &unknown)
	require.Equal(t, "missing", unknown.Name)
	require.Equal(t, []string{activateSkillAlpha, activateSkillBeta}, unknown.Catalog)
	require.NotContains(t, err.Error(), activateBodyAlpha)
	require.NotContains(t, err.Error(), activateBodyBeta)
	require.NotContains(t, err.Error(), "# Beta secret")
}

func TestActivate_EmptyName(t *testing.T) {
	t.Parallel()

	_, err := skills.Activate(nil, "")
	require.ErrorIs(t, err, skills.ErrEmptyName)
}

func TestActivate_DoesNotReadReferences(t *testing.T) {
	t.Parallel()

	root := writeActivateSkillWithRefs(t)
	catalog := skills.Catalog{{
		Name:     activateSkillAlpha,
		Location: root,
		Source:   skills.SourceProject,
	}}

	got, err := skills.Activate(catalog, activateSkillAlpha)
	require.NoError(t, err)
	require.Contains(t, got.Body, "# Alpha body")
	require.NotContains(t, got.Body, activateExtraSentinel)
	require.NotContains(t, got.Body, activateNestedSentinel)
}

func TestActivate_RecordsAllowedTools(t *testing.T) {
	t.Parallel()

	root := writeActivateSkill(t, t.TempDir(), activateSkillAlpha, activateDescAlpha, activateBodyAlpha, activateAllowedTools)
	catalog := skills.Catalog{{
		Name:     activateSkillAlpha,
		Location: root,
		Source:   skills.SourceBundled,
	}}

	got, err := skills.Activate(catalog, activateSkillAlpha)
	require.NoError(t, err)
	require.Equal(t, activateAllowedTools, got.AllowedTools)
	require.Equal(t, skills.SourceBundled, got.Source)
}

func writeActivateSkill(t *testing.T, parent, name, description, body, allowedTools string) string {
	t.Helper()

	dir := filepath.Join(parent, name)
	require.NoError(t, os.MkdirAll(dir, activateDirPerm))

	front := "---\nname: " + name + "\ndescription: " + description + "\n"
	if allowedTools != "" {
		front += "allowed-tools: " + allowedTools + "\n"
	}

	front += "---\n\n" + body
	require.NoError(t, os.WriteFile(filepath.Join(dir, skills.FileName), []byte(front), activateFilePerm))

	return dir
}

func writeActivateSkillWithRefs(t *testing.T) string {
	t.Helper()

	root := writeActivateSkill(t, t.TempDir(), activateSkillAlpha, activateDescAlpha, activateBodyAlpha, activateAllowedTools)
	refs := filepath.Join(root, "references")
	require.NoError(t, os.MkdirAll(refs, activateDirPerm))
	require.NoError(t, os.WriteFile(filepath.Join(refs, "extra.md"),
		[]byte(activateExtraSentinel+"\n\nSee [nested](nested.md).\n"), activateFilePerm))
	require.NoError(t, os.WriteFile(filepath.Join(refs, "nested.md"),
		[]byte(activateNestedSentinel+"\n"), activateFilePerm))

	return root
}
