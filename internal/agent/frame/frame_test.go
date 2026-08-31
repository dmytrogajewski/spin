package frame_test

// Journey: specs/journeys/JOURNEY-014-taskframe-on-every-parent-turn.md.

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/agent"
	"github.com/dmytrogajewski/spin/internal/agent/frame"
)

func TestPhaseForMode_Review(t *testing.T) {
	t.Parallel()

	require.Equal(t, agent.ModeReview, frame.PhaseForMode(agent.ModeReview))
}

func TestPhaseForMode_AllModes(t *testing.T) {
	t.Parallel()

	modes := []string{agent.ModeRegular, agent.ModeReview, agent.ModeCompact, agent.ModePlanning}
	for _, mode := range modes {
		t.Run(mode, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, mode, frame.PhaseForMode(mode))
		})
	}
}

func TestPhaseForMode_FallbackRegular(t *testing.T) {
	t.Parallel()

	require.Equal(t, agent.ModeRegular, frame.PhaseForMode(""))
	require.Equal(t, agent.ModeRegular, frame.PhaseForMode("nope"))
}

func TestFromMode_ShortDefaults(t *testing.T) {
	t.Parallel()

	got := frame.FromMode(agent.ModeReview)
	require.Equal(t, agent.ModeReview, got.Phase)
	require.NotEmpty(t, got.Objective)
	require.LessOrEqual(t, len(got.Objective), frame.MaxFieldBytes)
	require.NotEmpty(t, got.OutputFormat)
}

func TestTaskFrame_MarshalStable(t *testing.T) {
	t.Parallel()

	got := frame.FromMode(agent.ModeReview)
	first, err := got.MarshalStable()
	require.NoError(t, err)

	second, err := got.MarshalStable()
	require.NoError(t, err)
	require.Equal(t, first, second)
	require.Contains(t, string(first), `"phase":"review"`)

	keys := []string{
		`"objective"`, `"phase"`, `"output_format"`, `"tools"`,
		`"sources"`, `"boundaries"`, `"success_criteria"`,
	}
	for _, key := range keys {
		require.Contains(t, string(first), key)
	}
}

const agentsBodySentinel = "UNIQUE_AGENTS_BODY_SENTINEL"

func TestTaskFrame_SourcesOmitBodies(t *testing.T) {
	t.Parallel()

	body := agentsBodySentinel + "\nfull file contents here"
	got := frame.FromMode(agent.ModeRegular).WithSources("AGENTS.md", body)
	raw, err := got.MarshalStable()
	require.NoError(t, err)
	require.Contains(t, string(raw), "AGENTS.md")
	require.NotContains(t, string(raw), agentsBodySentinel)
}

func TestTaskFrame_RenderedSizeUnderCap(t *testing.T) {
	t.Parallel()

	got := frame.FromMode(agent.ModeRegular).WithSources("AGENTS.md", "internal/agent/frame")
	rendered := got.Render()
	require.NotEmpty(t, rendered)
	require.LessOrEqual(t, len(rendered), frame.MaxRenderedBytes)
	require.NotContains(t, rendered, agentsBodySentinel)
}
