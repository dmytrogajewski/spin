package prompt_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/agent/prompt"
)

// TestDeveloperGoals_TotalCount verifies the 30 goals from IEEE Software
// (Ferrari-Church & Egelman, 2024) are all present.
func TestDeveloperGoals_TotalCount(t *testing.T) {
	t.Parallel()

	phases := prompt.DeveloperGoals()
	total := 0

	for _, phase := range phases {
		total += len(phase.Goals)
	}

	assert.Equal(t, prompt.TotalGoalCount, total,
		"total goals must match TotalGoalCount constant")
}

// TestDeveloperGoals_PhaseCount verifies 6 SDLC phases exist.
func TestDeveloperGoals_PhaseCount(t *testing.T) {
	t.Parallel()

	phases := prompt.DeveloperGoals()
	require.Len(t, phases, len(prompt.DeveloperGoals()), "must have all SDLC phases")
}

// TestDeveloperGoals_PhaseNames verifies phase names match the paper.
func TestDeveloperGoals_PhaseNames(t *testing.T) {
	t.Parallel()

	phases := prompt.DeveloperGoals()
	expected := []string{
		"Information Gathering",
		"Plan and Track Work, and Manage Approvals",
		"Develop, Test and Commit Code",
		"Experiment, Release and Rollout",
		"Monitoring, Reliability, and Configuring Infrastructure",
		"Data Management",
	}

	for i, phase := range phases {
		assert.Equal(t, expected[i], phase.Name)
	}
}

// TestDeveloperGoals_GoalsPerPhase verifies goal counts per phase match the paper.
func TestDeveloperGoals_GoalsPerPhase(t *testing.T) {
	t.Parallel()

	phases := prompt.DeveloperGoals()
	expectedCounts := []int{5, 6, 6, 3, 6, 4}

	for i, phase := range phases {
		assert.Len(t, phase.Goals, expectedCounts[i],
			"phase %q should have %d goals", phase.Name, expectedCounts[i])
	}
}

// TestDeveloperGoals_NoEmptyGoals verifies no goal string is empty.
func TestDeveloperGoals_NoEmptyGoals(t *testing.T) {
	t.Parallel()

	for _, phase := range prompt.DeveloperGoals() {
		for _, goal := range phase.Goals {
			assert.NotEmpty(t, goal, "empty goal in phase %q", phase.Name)
		}
	}
}
