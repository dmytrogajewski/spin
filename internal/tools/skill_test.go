package tools

// Journey: specs/journeys/JOURNEY-003-activate-skill-body.md.

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/skills"
)

const (
	skillTestName        = "alpha"
	skillTestDesc        = "Alpha skill for tool tests."
	skillTestBody        = "# Alpha body heading\n\nSee [extra](references/extra.md).\n"
	skillTestExtra       = "UNIQUE_TOOL_EXTRA_SENTINEL"
	skillTestNested      = "UNIQUE_TOOL_NESTED_SENTINEL"
	skillTestAllowed     = "read_file grep"
	skillTestDirPerm     = 0o750
	skillTestFilePerm    = 0o600
	skillTestUnknownName = "missing"
	skillTestBetaName    = "beta"
	skillTestBetaBody    = "# Beta body must not leak.\n"
)

func TestSkillTool_ActivateReturnsBodyAndRoot(t *testing.T) {
	t.Parallel()

	catalog, root := testSkillCatalog(t)
	tool := NewSkillTool(catalog)
	require.Equal(t, skillToolName, tool.Name())

	params, err := FromMap(map[string]any{skillParamName: skillTestName})
	require.NoError(t, err)

	result, execErr := tool.Execute(context.Background(), params)
	require.NoError(t, execErr)
	require.True(t, result.Success)
	require.Contains(t, result.Output, skillTestBody)
	require.Contains(t, result.Output, root)
	require.Contains(t, result.Output, "allowed-tools: "+skillTestAllowed)
	require.NotContains(t, result.Output, skillTestExtra)
	require.Equal(t, skillTestName, result.Metadata["name"])
	require.Equal(t, string(skills.SourceProject), result.Metadata["source"])
	require.Equal(t, root, result.Metadata["root"])
	require.Equal(t, skillTestAllowed, result.Metadata["allowed-tools"])
}

func TestLoadSkillTool_SameActivation(t *testing.T) {
	t.Parallel()

	catalog, root := testSkillCatalog(t)
	params, err := FromMap(map[string]any{skillParamName: skillTestName})
	require.NoError(t, err)

	skillResult, skillErr := NewSkillTool(catalog).Execute(context.Background(), params)
	require.NoError(t, skillErr)

	alias := NewLoadSkillTool(catalog)
	require.Equal(t, loadSkillToolName, alias.Name())

	aliasResult, aliasErr := alias.Execute(context.Background(), params)
	require.NoError(t, aliasErr)
	require.Equal(t, skillResult.Output, aliasResult.Output)
	require.Contains(t, aliasResult.Output, root)
}

func TestSkillTool_UnknownNameTypedError(t *testing.T) {
	t.Parallel()

	catalog, _ := testSkillCatalog(t)
	params, err := FromMap(map[string]any{skillParamName: skillTestUnknownName})
	require.NoError(t, err)

	result, execErr := NewSkillTool(catalog).Execute(context.Background(), params)
	require.NoError(t, execErr)
	require.False(t, result.Success)
	require.ErrorIs(t, result.Err, skills.ErrUnknownSkill)

	var unknown *skills.UnknownSkillError
	require.ErrorAs(t, result.Err, &unknown)
	require.Contains(t, unknown.Catalog, skillTestName)
	require.Contains(t, unknown.Catalog, skillTestBetaName)
	require.NotContains(t, result.Error, skillTestBody)
	require.NotContains(t, result.Error, skillTestBetaBody)
}

func TestSkillTool_PathReadOneHop(t *testing.T) {
	t.Parallel()

	catalog, _ := testSkillCatalog(t)
	params, err := FromMap(map[string]any{
		skillParamName: skillTestName,
		skillParamPath: "references/extra.md",
	})
	require.NoError(t, err)

	result, execErr := NewSkillTool(catalog).Execute(context.Background(), params)
	require.NoError(t, execErr)
	require.True(t, result.Success)
	require.Contains(t, result.Output, skillTestExtra)
	require.NotContains(t, result.Output, skillTestNested)
}

func TestSkillTool_RejectsDotDot(t *testing.T) {
	t.Parallel()

	catalog, _ := testSkillCatalog(t)
	params, err := FromMap(map[string]any{
		skillParamName: skillTestName,
		skillParamPath: "../secrets.md",
	})
	require.NoError(t, err)

	result, execErr := NewSkillTool(catalog).Execute(context.Background(), params)
	require.NoError(t, execErr)
	require.False(t, result.Success)
	require.ErrorIs(t, result.Err, skills.ErrPathEscape)
}

func TestSkillTool_EmptyName(t *testing.T) {
	t.Parallel()

	params, err := FromMap(map[string]any{skillParamName: ""})
	require.NoError(t, err)

	result, execErr := NewSkillTool(nil).Execute(context.Background(), params)
	require.NoError(t, execErr)
	require.False(t, result.Success)
	require.ErrorIs(t, result.Err, skills.ErrEmptyName)
}

func TestRegisterSkillTools(t *testing.T) {
	t.Parallel()

	catalog, _ := testSkillCatalog(t)
	reg := NewRegistry()
	RegisterSkillTools(reg, catalog)

	skill, err := reg.Get(skillToolName)
	require.NoError(t, err)
	require.Equal(t, skillToolName, skill.Name())

	alias, err := reg.Get(loadSkillToolName)
	require.NoError(t, err)
	require.Equal(t, loadSkillToolName, alias.Name())
}

func testSkillCatalog(t *testing.T) (catalog skills.Catalog, root string) {
	t.Helper()

	work := t.TempDir()
	alpha := writeSkillTree(t, work, skillTestName, skillTestDesc, skillTestBody, skillTestAllowed)
	beta := writeSkillTree(t, work, skillTestBetaName, "Beta description.", skillTestBetaBody, "")

	return skills.Catalog{
		{Name: skillTestName, Description: skillTestDesc, Location: alpha, Source: skills.SourceProject},
		{Name: skillTestBetaName, Description: "Beta description.", Location: beta, Source: skills.SourceUser},
	}, alpha
}

func writeSkillTree(t *testing.T, parent, name, description, body, allowedTools string) string {
	t.Helper()

	dir := filepath.Join(parent, name)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "references"), skillTestDirPerm))

	front := "---\nname: " + name + "\ndescription: " + description + "\n"
	if allowedTools != "" {
		front += "allowed-tools: " + allowedTools + "\n"
	}

	front += "---\n\n" + body
	require.NoError(t, os.WriteFile(filepath.Join(dir, skills.FileName), []byte(front), skillTestFilePerm))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "references", "extra.md"),
		[]byte(skillTestExtra+"\n\nSee [nested](nested.md).\n"), skillTestFilePerm))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "references", "nested.md"),
		[]byte(skillTestNested+"\n"), skillTestFilePerm))

	return dir
}
