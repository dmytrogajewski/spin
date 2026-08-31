package prompt_test

// Journey: specs/journeys/JOURNEY-014-taskframe-on-every-parent-turn.md.

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/agent"
	"github.com/dmytrogajewski/spin/internal/agent/frame"
	"github.com/dmytrogajewski/spin/internal/agent/prompt"
)

func TestTaskFrameSection_NotCacheable(t *testing.T) {
	t.Parallel()

	section := prompt.TaskFrameSection(frame.FromMode(agent.ModeReview))
	require.Equal(t, prompt.SectionTaskFrame, section.Name)
	require.False(t, section.Cacheable)
	require.Contains(t, section.Template, `"phase":"review"`)
}

func TestApplyTaskFrame_DynamicOnly(t *testing.T) {
	t.Parallel()

	composer := prompt.NewComposer()
	for _, section := range prompt.DefaultRegularSections() {
		composer.AddSection(section)
	}

	prompt.ApplyTaskFrame(composer, frame.FromMode(agent.ModeReview))

	stable, dynamic := composer.ComposeTwoPart(nonGitEnv())
	require.Contains(t, dynamic, taskFrameMarker)
	require.Contains(t, dynamic, `"phase":"review"`)
	require.NotContains(t, stable, taskFrameMarker)
	require.NotContains(t, stable, `"phase":"review"`)
}

func TestApplyTaskFrame_NilComposer(t *testing.T) {
	t.Parallel()

	require.NotPanics(t, func() {
		prompt.ApplyTaskFrame(nil, frame.FromMode(agent.ModeRegular))
	})
}

const taskFrameMarker = "# Task Frame"
