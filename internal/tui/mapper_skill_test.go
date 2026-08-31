package tui

// Journey: specs/journeys/JOURNEY-003-activate-skill-body.md.

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/tools"
	"github.com/dmytrogajewski/spin/internal/ui/blocks"
)

func TestMapper_SkillBlockNameAndSource(t *testing.T) {
	t.Parallel()

	ui := newFakeUI()
	mapper := NewMapper(ui)

	args, err := tools.FromMap(map[string]any{"name": "alpha"})
	require.NoError(t, err)

	start := events.Event{
		Type: events.EventToolCallStart,
		Data: events.ToolCallStartData{
			ToolName:   "skill",
			ToolID:     "tool_skill_1",
			Parameters: args,
		},
	}
	require.NoError(t, mapper.MapEvent(context.Background(), start))

	block := ui.blocks["tool_skill_1"]
	require.NotNil(t, block)
	require.Equal(t, blocks.BlockTypeSkill, block.Type)
	require.Equal(t, "alpha", block.Title)

	meta, metaErr := blocks.ParseSkillMeta(block)
	require.NoError(t, metaErr)
	require.Equal(t, "alpha", meta.Name)

	complete := events.Event{
		Type: events.EventToolCallComplete,
		Data: events.ToolCallCompleteData{
			ToolID:   "tool_skill_1",
			ToolName: "skill",
			Success:  true,
			Output:   "name: alpha\nsource: project\nroot: /tmp/alpha\n",
			Metadata: map[string]any{
				"name":   "alpha",
				"source": "project",
				"root":   "/tmp/alpha",
			},
		},
	}
	require.NoError(t, mapper.MapEvent(context.Background(), complete))

	updated := ui.blocks["tool_skill_1"]
	require.NotNil(t, updated)
	require.Equal(t, blocks.BlockTypeSkill, updated.Type)
	require.Contains(t, updated.Title, "alpha")
	require.Contains(t, updated.Title, "project")

	updatedMeta, updatedErr := blocks.ParseSkillMeta(updated)
	require.NoError(t, updatedErr)
	require.Equal(t, "alpha", updatedMeta.Name)
	require.Equal(t, "project", updatedMeta.Source)
}

func TestMapper_LoadSkillAliasBlock(t *testing.T) {
	t.Parallel()

	ui := newFakeUI()
	mapper := NewMapper(ui)

	args, err := tools.FromMap(map[string]any{"name": "beta"})
	require.NoError(t, err)

	start := events.Event{
		Type: events.EventToolCallStart,
		Data: events.ToolCallStartData{
			ToolName:   "load_skill",
			ToolID:     "tool_load_skill_1",
			Parameters: args,
		},
	}
	require.NoError(t, mapper.MapEvent(context.Background(), start))

	block := ui.blocks["tool_load_skill_1"]
	require.NotNil(t, block)
	require.Equal(t, blocks.BlockTypeSkill, block.Type)
	require.Equal(t, "beta", block.Title)
}
