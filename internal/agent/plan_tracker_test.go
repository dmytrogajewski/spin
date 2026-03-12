package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/planning"
)

func TestPlanTracker_OnToolCallComplete_MatchesAndUpdates(t *testing.T) {
	t.Parallel()
	emitter := events.NewEventEmitter(10)
	plan := &planning.Plan{
		Steps: []planning.Step{
			{
				ID:          "step1",
				Description: "Read configuration file",
				Status:      planning.StepStatusPending,
			},
		},
	}

	tracker := NewPlanTracker(plan, emitter)

	// Simulate tool call complete.
	event := events.Event{
		Type: events.EventToolCallComplete,
		Data: events.ToolCallCompleteData{
			ToolName: "read_file",
			Success:  true,
		},
	}

	tracker.OnToolCallComplete(event)

	assert.Equal(t, planning.StepStatusCompleted, plan.Steps[0].Status)
	assert.NotNil(t, plan.Steps[0].CompletedAt)
}

func TestPlanTracker_UpdatePlanStatus(t *testing.T) {
	t.Parallel()
	emitter := events.NewEventEmitter(10)
	plan := &planning.Plan{
		Status: planning.PlanStatusPending,
		Steps: []planning.Step{
			{ID: "s1", Status: planning.StepStatusCompleted},
			{ID: "s2", Status: planning.StepStatusCompleted},
		},
	}

	tracker := NewPlanTracker(plan, emitter)
	tracker.UpdatePlanStatus()

	assert.Equal(t, planning.PlanStatusCompleted, plan.Status)
}

func TestPlanTracker_FuzzyMatching(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		stepDesc    string
		stepAction  string
		toolName    string
		shouldMatch bool
	}{
		{
			name:        "Exact match",
			stepDesc:    "Use read_file",
			stepAction:  "",
			toolName:    "read_file",
			shouldMatch: true,
		},
		{
			name:        "Action match",
			stepDesc:    "Check file",
			stepAction:  "read_file config.json",
			toolName:    "read_file",
			shouldMatch: true,
		},
		{
			name:        "Semantic match read",
			stepDesc:    "Read configuration",
			stepAction:  "",
			toolName:    "read_file",
			shouldMatch: true,
		},
		{
			name:        "Semantic match list",
			stepDesc:    "List files in directory",
			stepAction:  "",
			toolName:    "list_directory",
			shouldMatch: true,
		},
		{
			name:        "No match",
			stepDesc:    "Do something else",
			stepAction:  "",
			toolName:    "read_file",
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			step := &planning.Step{
				Description: tt.stepDesc,
				Action:      tt.stepAction,
			}
			tracker := &PlanTracker{}
			match := tracker.matchesStep(step, tt.toolName)
			assert.Equal(t, tt.shouldMatch, match)
		})
	}
}
