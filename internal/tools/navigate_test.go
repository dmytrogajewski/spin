package tools

// Journey: specs/journeys/JOURNEY-022-structured-navigation-index.md.

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/nav"
	"github.com/dmytrogajewski/spin/internal/skills"
)

func TestNavigateTool_ReturnsRecordShape(t *testing.T) {
	t.Parallel()

	index := nav.New(nav.Sources{Skills: skills.Catalog{{
		Name:        "nav-probe",
		Description: "Probe skill for navigation index tests.",
		Location:    "/tmp/nav-probe",
		Source:      skills.SourceProject,
	}}})

	params, err := FromMap(map[string]any{navigateKindParam: string(nav.KindSkill)})
	require.NoError(t, err)

	result, execErr := NewNavigateTool(index).Execute(t.Context(), params)
	require.NoError(t, execErr)
	require.True(t, result.Success)

	var payload nav.Result
	require.NoError(t, json.Unmarshal([]byte(result.Output), &payload))
	require.Len(t, payload.Records, 1)
	require.Equal(t, nav.KindSkill, payload.Records[0].Kind)
	require.Equal(t, "nav-probe", payload.Records[0].ID)
	require.Equal(t, "/tmp/nav-probe", payload.Records[0].Open)
	require.NotContains(t, result.Output, "SECRET_SKILL_BODY_MUST_NOT_LEAK")
}

func TestNavigateTool_UnknownKind(t *testing.T) {
	t.Parallel()

	params, err := FromMap(map[string]any{navigateKindParam: "nope"})
	require.NoError(t, err)

	result, execErr := NewNavigateTool(nav.New(nav.Sources{})).Execute(t.Context(), params)
	require.NoError(t, execErr)
	require.False(t, result.Success)
	require.Contains(t, result.Error, nav.ValidKinds)
}

func TestNavigateTool_PathListingCompressed(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	params, err := FromMap(map[string]any{
		navigateKindParam: string(nav.KindPath),
		navigatePathParam: dir,
	})
	require.NoError(t, err)

	result, execErr := NewNavigateTool(nav.New(nav.Sources{})).Execute(t.Context(), params)
	require.NoError(t, execErr)
	require.True(t, result.Success)

	var payload nav.Result
	require.NoError(t, json.Unmarshal([]byte(result.Output), &payload))
	require.Len(t, payload.Records, 1)
	require.Equal(t, dir, payload.Records[0].Open)
}

func TestRegisterNavigateTool(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	RegisterNavigateTool(reg, nav.New(nav.Sources{}))

	tool, err := reg.Get(navigateToolName)
	require.NoError(t, err)
	require.Equal(t, navigateToolName, tool.Name())
}
