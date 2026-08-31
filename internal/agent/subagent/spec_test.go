package subagent

// Journey: specs/journeys/JOURNEY-019-subagent-hooks-and-no-silent-drop.md.

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHasTool_SpawnDenyByDefault(t *testing.T) {
	t.Parallel()

	open := &Spec{Name: "open"}
	require.True(t, open.HasTool("read_file"))
	require.False(t, open.HasTool(ToolSpawn))

	for _, spec := range Builtins() {
		require.False(t, spec.HasTool(ToolSpawn), spec.Name)
		require.NotContains(t, spec.AllowedTools, ToolSpawn)
	}

	allowed := &Spec{Name: "nested", AllowedTools: []string{ToolSpawn}}
	require.True(t, allowed.HasTool(ToolSpawn))
}
