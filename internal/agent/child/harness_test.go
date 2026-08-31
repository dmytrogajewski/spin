package child

// Journey: specs/journeys/JOURNEY-017-local-a2a-server-process.md.

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/agent/subagent"
	"github.com/dmytrogajewski/spin/internal/protocol/a2a"
)

const parentHistorySentinel = "PARENT_TRANSCRIPT_SENTINEL"

func TestNewHarness_EmptyHistoryAndFrameTools(t *testing.T) {
	t.Parallel()

	spec, err := subagent.Lookup(subagent.NameExplorer)
	require.NoError(t, err)

	childHarness := NewHarness(spec)
	require.Empty(t, childHarness.History())
	require.Equal(t, spec.AllowedTools, childHarness.Frame().Tools)
	require.NotContains(t, childHarness.Frame().Render(), parentHistorySentinel)
}

func TestNewHarness_DoesNotAcceptParentHistory(t *testing.T) {
	t.Parallel()

	spec, err := subagent.Lookup(subagent.NameExplorer)
	require.NoError(t, err)

	_ = []a2a.Message{{MessageID: "parent", Parts: []a2a.Part{{Text: parentHistorySentinel}}}}
	childHarness := NewHarness(spec)
	require.Empty(t, childHarness.History())
}
