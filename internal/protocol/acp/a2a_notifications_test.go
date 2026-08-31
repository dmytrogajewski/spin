package acp

// Journey: specs/journeys/JOURNEY-023-tui-and-acp-surfaces.md.

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/events"
)

func TestConvertEventToSessionUpdate_TaskStateKindA2A(t *testing.T) {
	t.Parallel()

	update, ok := convertEventToSessionUpdate(events.Event{
		Type: events.EventBackgroundTaskStarted,
		Data: events.TaskStateData{TaskID: "abc1234", State: "working", Kind: "a2a"},
	}, nil, "")

	require.True(t, ok)
	require.NotNil(t, update.AgentThoughtChunk)
	require.NotNil(t, update.AgentThoughtChunk.Content.Text)
	require.Contains(t, update.AgentThoughtChunk.Content.Text.Text, kindA2A)
	require.Contains(t, update.AgentThoughtChunk.Content.Text.Text, "abc1234")
}

func TestConvertEventToSessionUpdate_SubagentSpawnKindA2A(t *testing.T) {
	t.Parallel()

	update, ok := convertEventToSessionUpdate(events.Event{
		Type: events.EventSubagentSpawn,
		Data: events.SubagentSpawnData{AgentType: "explorer", Query: "find auth"},
	}, nil, "")

	require.True(t, ok)
	require.NotNil(t, update.AgentThoughtChunk)
	require.Contains(t, update.AgentThoughtChunk.Content.Text.Text, kindA2A)
	require.Contains(t, update.AgentThoughtChunk.Content.Text.Text, "explorer")
}
