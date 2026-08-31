package tui

// Journey: specs/journeys/JOURNEY-023-tui-and-acp-surfaces.md.

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/ui/blocks"
)

func TestMapper_TaskBlockFromBackgroundStart(t *testing.T) {
	t.Parallel()

	ui := newFakeUI()
	mapper := NewMapper(ui)

	err := mapper.MapEvent(context.Background(), events.Event{
		Type: events.EventBackgroundTaskStarted,
		Data: events.TaskStateData{TaskID: "abc1234", Spec: "explorer", State: "working", Kind: "a2a"},
	})
	require.NoError(t, err)

	block := firstBlockOfType(t, ui, blocks.BlockTypeTask)
	require.Contains(t, block.Title, "abc1234")
}

func TestMapper_SubagentBlockFromSpawn(t *testing.T) {
	t.Parallel()

	ui := newFakeUI()
	mapper := NewMapper(ui)

	err := mapper.MapEvent(context.Background(), events.Event{
		Type: events.EventSubagentSpawn,
		Data: events.SubagentSpawnData{AgentType: "explorer", Query: "find auth"},
	})
	require.NoError(t, err)

	block := firstBlockOfType(t, ui, blocks.BlockTypeSubagent)
	require.Equal(t, "explorer", block.Title)
}

func TestMapper_HookVetoBlockShowsReason(t *testing.T) {
	t.Parallel()

	ui := newFakeUI()
	mapper := NewMapper(ui)

	err := mapper.MapEvent(context.Background(), events.Event{
		Type: events.EventHookVeto,
		Data: events.HookVetoData{Event: "SUBAGENT_START", Reason: "veto", Spec: "explorer"},
	})
	require.NoError(t, err)

	block := firstBlockOfType(t, ui, blocks.BlockTypeHook)
	require.Contains(t, block.Body, "veto")
	require.Equal(t, blocks.SeverityError, block.Severity)
}

func TestMapper_CompactBlockFromCompaction(t *testing.T) {
	t.Parallel()

	ui := newFakeUI()
	mapper := NewMapper(ui)

	err := mapper.MapEvent(context.Background(), events.Event{
		Type: events.EventCompactionTriggered,
		Data: events.CompactionTriggeredData{Turn: 3, Stage: "history"},
	})
	require.NoError(t, err)

	block := firstBlockOfType(t, ui, blocks.BlockTypeCompact)
	require.Equal(t, "history", block.Title)
}

func firstBlockOfType(t *testing.T, ui *fakeUI, want blocks.BlockType) *blocks.Block {
	t.Helper()

	for _, block := range ui.blocks {
		if block.Type == want {
			return block
		}
	}

	t.Fatalf("no block of type %s", want)

	return nil
}
