package overlay

// Journey: specs/journeys/JOURNEY-023-tui-and-acp-surfaces.md.

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegisterHarnessCommands_ListsSkillsTasksAgents(t *testing.T) {
	t.Parallel()

	registry := NewCommandRegistry()
	RegisterHarnessCommands(registry)

	palette := NewPalette(registry)
	palette.Open()

	names := map[string]bool{}
	for _, cmd := range palette.FilteredCommands() {
		names[cmd.Name()] = true
	}

	require.True(t, names["Skills"], "palette must list Skills")
	require.True(t, names["Tasks"], "palette must list Tasks")
	require.True(t, names["Agents"], "palette must list Agents")
}
