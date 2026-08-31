package commands

// Journey: specs/journeys/JOURNEY-023-tui-and-acp-surfaces.md.

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/agent/subagent"
)

func TestAgentsCommand_ListsBuiltins(t *testing.T) {
	t.Parallel()

	cmd := &AgentsCommand{}
	require.Equal(t, "/agents", cmd.Name())

	out, err := cmd.Execute(context.Background(), nil, &mockCommandContext{})
	require.NoError(t, err)
	require.Contains(t, out, subagent.NameExplorer)
}
