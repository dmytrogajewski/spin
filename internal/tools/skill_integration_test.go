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

func TestSkillActivation_CatalogPromptThenBodyThenReference(t *testing.T) {
	t.Parallel()

	work := t.TempDir()
	skillDir := filepath.Join(work, ".agents", "skills", skillTestName)
	require.NoError(t, os.MkdirAll(filepath.Join(skillDir, "references"), skillTestDirPerm))

	front := "---\nname: " + skillTestName + "\ndescription: " + skillTestDesc + "\n---\n\n" + skillTestBody
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, skills.FileName), []byte(front), skillTestFilePerm))
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "references", "extra.md"),
		[]byte(skillTestExtra+"\n\nSee [nested](nested.md).\n"), skillTestFilePerm))
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "references", "nested.md"),
		[]byte(skillTestNested+"\n"), skillTestFilePerm))

	catalog := skills.Discover(skills.Options{WorkDir: work})
	require.Len(t, catalog, 1)
	require.Equal(t, skillTestName, catalog[0].Name)
	require.Equal(t, skillTestDesc, catalog[0].Description)

	tool := NewSkillTool(catalog)
	activateParams, err := FromMap(map[string]any{skillParamName: skillTestName})
	require.NoError(t, err)

	activated, execErr := tool.Execute(context.Background(), activateParams)
	require.NoError(t, execErr)
	require.True(t, activated.Success)
	require.Contains(t, activated.Output, "# Alpha body heading")
	require.Contains(t, activated.Output, skillDir)
	require.NotContains(t, activated.Output, skillTestExtra)

	readParams, err := FromMap(map[string]any{
		skillParamName: skillTestName,
		skillParamPath: "references/extra.md",
	})
	require.NoError(t, err)

	read, readErr := tool.Execute(context.Background(), readParams)
	require.NoError(t, readErr)
	require.True(t, read.Success)
	require.Contains(t, read.Output, skillTestExtra)
	require.NotContains(t, read.Output, skillTestNested)
}
