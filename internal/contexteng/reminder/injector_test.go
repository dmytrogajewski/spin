package reminder_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/contexteng/reminder"
	"github.com/dmytrogajewski/spin/internal/message"
)

// Journey: specs/journeys/JOURNEY-2.4.md.

// TestInject_ToolFailure verifies reminder injection on tool failure.
// Kills mutant: not detecting tool failure would miss recovery opportunity.
func TestInject_ToolFailure(t *testing.T) {
	t.Parallel()

	inj := reminder.NewInjector(reminder.DefaultDetectors(), reminder.DefaultTemplates())

	ctx := reminder.CheckContext{LastToolFailed: true}
	msgs := inj.Inject(ctx)

	require.Len(t, msgs, 1)
	assert.Equal(t, message.RoleUser, msgs[0].Role)
	assert.Contains(t, msgs[0].Content, "failed")
}

// TestInject_ExplorationSpiral verifies reminder on consecutive reads.
// Kills mutant: ignoring exploration spirals would waste tokens.
func TestInject_ExplorationSpiral(t *testing.T) {
	t.Parallel()

	inj := reminder.NewInjector(reminder.DefaultDetectors(), reminder.DefaultTemplates())

	ctx := reminder.CheckContext{ConsecutiveReads: 5}
	msgs := inj.Inject(ctx)

	require.Len(t, msgs, 1)
	assert.Contains(t, msgs[0].Content, "reading files")
}

// TestInject_BelowSpiralThreshold verifies no reminder below read threshold.
// Kills mutant: triggering too early would annoy the agent.
func TestInject_BelowSpiralThreshold(t *testing.T) {
	t.Parallel()

	inj := reminder.NewInjector(reminder.DefaultDetectors(), reminder.DefaultTemplates())

	ctx := reminder.CheckContext{ConsecutiveReads: 4}
	msgs := inj.Inject(ctx)

	assert.Empty(t, msgs)
}

// TestInject_PrematureCompletion verifies reminder on incomplete todos.
// Kills mutant: letting premature completion pass would leave work undone.
func TestInject_PrematureCompletion(t *testing.T) {
	t.Parallel()

	inj := reminder.NewInjector(reminder.DefaultDetectors(), reminder.DefaultTemplates())

	ctx := reminder.CheckContext{HasIncompleteTodos: true}
	msgs := inj.Inject(ctx)

	require.Len(t, msgs, 1)
	assert.Contains(t, msgs[0].Content, "incomplete")
}

// TestInject_EmptyCompletion verifies reminder on empty assistant message.
// Kills mutant: accepting empty completions would confuse users.
func TestInject_EmptyCompletion(t *testing.T) {
	t.Parallel()

	inj := reminder.NewInjector(reminder.DefaultDetectors(), reminder.DefaultTemplates())

	ctx := reminder.CheckContext{LastAssistantEmpty: true}
	msgs := inj.Inject(ctx)

	require.Len(t, msgs, 1)
	assert.Contains(t, msgs[0].Content, "empty")
}

// TestInject_NoConditionMet verifies no reminders when everything is normal.
// Kills mutant: false positives would inject unwanted reminders.
func TestInject_NoConditionMet(t *testing.T) {
	t.Parallel()

	inj := reminder.NewInjector(reminder.DefaultDetectors(), reminder.DefaultTemplates())

	ctx := reminder.CheckContext{}
	msgs := inj.Inject(ctx)

	assert.Empty(t, msgs)
}

// TestInject_MaxFiresCap verifies detector stops after reaching fire cap.
// Kills mutant: unlimited firing would cause reminder fatigue.
func TestInject_MaxFiresCap(t *testing.T) {
	t.Parallel()

	det := reminder.NewToolFailureDetector()
	inj := reminder.NewInjector(
		[]reminder.Detector{det},
		reminder.DefaultTemplates(),
	)

	ctx := reminder.CheckContext{LastToolFailed: true}

	// Fire up to MaxFires.
	for range det.MaxFires() {
		msgs := inj.Inject(ctx)
		require.Len(t, msgs, 1)
	}

	// One more should produce nothing.
	msgs := inj.Inject(ctx)
	assert.Empty(t, msgs)
}

// TestInject_MultipleDetectorsFire verifies independent detector firing.
// Kills mutant: one detector blocking another would miss patterns.
func TestInject_MultipleDetectorsFire(t *testing.T) {
	t.Parallel()

	inj := reminder.NewInjector(reminder.DefaultDetectors(), reminder.DefaultTemplates())

	ctx := reminder.CheckContext{
		LastToolFailed:     true,
		LastAssistantEmpty: true,
	}

	msgs := inj.Inject(ctx)

	assert.Len(t, msgs, 2)
}

// TestInject_Reset verifies counter reset.
// Kills mutant: stale counters would prevent firing in new queries.
func TestInject_Reset(t *testing.T) {
	t.Parallel()

	det := reminder.NewToolFailureDetector()
	inj := reminder.NewInjector(
		[]reminder.Detector{det},
		reminder.DefaultTemplates(),
	)

	ctx := reminder.CheckContext{LastToolFailed: true}

	// Exhaust fires.
	for range det.MaxFires() {
		inj.Inject(ctx)
	}

	assert.Empty(t, inj.Inject(ctx))

	// Reset and fire again.
	inj.Reset()

	msgs := inj.Inject(ctx)
	require.Len(t, msgs, 1)
}

// TestInject_DeniedToolRetry verifies reminder when agent retries denied tool.
// Kills mutant: ignoring denied tool retries would waste tokens on futile attempts.
func TestInject_DeniedToolRetry(t *testing.T) {
	t.Parallel()

	inj := reminder.NewInjector(reminder.DefaultDetectors(), reminder.DefaultTemplates())

	ctx := reminder.CheckContext{LastToolDenied: true}
	msgs := inj.Inject(ctx)

	require.Len(t, msgs, 1)
	assert.Contains(t, msgs[0].Content, "denied")
}

// TestInject_DeniedToolRetry_NotTriggered verifies no reminder when tool not denied.
// Kills mutant: false positive on denied tool would inject unwanted reminders.
func TestInject_DeniedToolRetry_NotTriggered(t *testing.T) {
	t.Parallel()

	det := reminder.NewDeniedToolRetryDetector()
	result := det.Check(reminder.CheckContext{LastToolDenied: false})

	assert.False(t, result)
}

// TestInject_CompletedTodos verifies reminder when agent continues after all todos done.
// Kills mutant: not detecting completed todos would waste cycles on unnecessary work.
func TestInject_CompletedTodos(t *testing.T) {
	t.Parallel()

	inj := reminder.NewInjector(reminder.DefaultDetectors(), reminder.DefaultTemplates())

	ctx := reminder.CheckContext{AllTodosComplete: true}
	msgs := inj.Inject(ctx)

	require.Len(t, msgs, 1)
	assert.Contains(t, msgs[0].Content, "completed")
}

// TestInject_CompletedTodos_NotTriggered verifies no reminder when todos incomplete.
// Kills mutant: false positive would prematurely stop work.
func TestInject_CompletedTodos_NotTriggered(t *testing.T) {
	t.Parallel()

	det := reminder.NewCompletedTodosDetector()
	result := det.Check(reminder.CheckContext{AllTodosComplete: false})

	assert.False(t, result)
}

// TestInject_PlanNotExecuted verifies reminder when plan approved but not executed.
// Kills mutant: ignoring unapplied plans would leave approved work undone.
func TestInject_PlanNotExecuted(t *testing.T) {
	t.Parallel()

	inj := reminder.NewInjector(reminder.DefaultDetectors(), reminder.DefaultTemplates())

	ctx := reminder.CheckContext{PlanApprovedNotExecuted: true}
	msgs := inj.Inject(ctx)

	require.Len(t, msgs, 1)
	assert.Contains(t, msgs[0].Content, "plan")
}

// TestInject_PlanNotExecuted_NotTriggered verifies no reminder when no pending plan.
// Kills mutant: false positive would inject confusing plan reminders.
func TestInject_PlanNotExecuted_NotTriggered(t *testing.T) {
	t.Parallel()

	det := reminder.NewPlanNotExecutedDetector()
	result := det.Check(reminder.CheckContext{PlanApprovedNotExecuted: false})

	assert.False(t, result)
}

// TestInject_UnprocessedSubagent verifies reminder when subagent results ignored.
// Kills mutant: ignoring subagent results would lose valuable work output.
func TestInject_UnprocessedSubagent(t *testing.T) {
	t.Parallel()

	inj := reminder.NewInjector(reminder.DefaultDetectors(), reminder.DefaultTemplates())

	ctx := reminder.CheckContext{HasUnprocessedSubagentResults: true}
	msgs := inj.Inject(ctx)

	require.Len(t, msgs, 1)
	assert.Contains(t, msgs[0].Content, "subagent")
}

// TestInject_UnprocessedSubagent_NotTriggered verifies no reminder when results processed.
// Kills mutant: false positive would annoy with unnecessary subagent reminders.
func TestInject_UnprocessedSubagent_NotTriggered(t *testing.T) {
	t.Parallel()

	det := reminder.NewUnprocessedSubagentDetector()
	result := det.Check(reminder.CheckContext{HasUnprocessedSubagentResults: false})

	assert.False(t, result)
}

// TestNewDetectors_TableDriven verifies all 8 detectors via table-driven tests.
// Kills mutant: ensures each detector fires only on its designated field.
func TestNewDetectors_TableDriven(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		detector reminder.Detector
		ctx      reminder.CheckContext
		expected bool
	}{
		{
			name:     "denied_tool_retry_fires",
			detector: reminder.NewDeniedToolRetryDetector(),
			ctx:      reminder.CheckContext{LastToolDenied: true},
			expected: true,
		},
		{
			name:     "denied_tool_retry_silent",
			detector: reminder.NewDeniedToolRetryDetector(),
			ctx:      reminder.CheckContext{},
			expected: false,
		},
		{
			name:     "completed_todos_fires",
			detector: reminder.NewCompletedTodosDetector(),
			ctx:      reminder.CheckContext{AllTodosComplete: true},
			expected: true,
		},
		{
			name:     "completed_todos_silent",
			detector: reminder.NewCompletedTodosDetector(),
			ctx:      reminder.CheckContext{},
			expected: false,
		},
		{
			name:     "plan_not_executed_fires",
			detector: reminder.NewPlanNotExecutedDetector(),
			ctx:      reminder.CheckContext{PlanApprovedNotExecuted: true},
			expected: true,
		},
		{
			name:     "plan_not_executed_silent",
			detector: reminder.NewPlanNotExecutedDetector(),
			ctx:      reminder.CheckContext{},
			expected: false,
		},
		{
			name:     "unprocessed_subagent_fires",
			detector: reminder.NewUnprocessedSubagentDetector(),
			ctx:      reminder.CheckContext{HasUnprocessedSubagentResults: true},
			expected: true,
		},
		{
			name:     "unprocessed_subagent_silent",
			detector: reminder.NewUnprocessedSubagentDetector(),
			ctx:      reminder.CheckContext{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.detector.Check(tt.ctx)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestNewDetectors_MaxFires verifies all new detectors respect fire cap.
// Kills mutant: unlimited firing would cause reminder fatigue.
func TestNewDetectors_MaxFires(t *testing.T) {
	t.Parallel()

	detectors := []reminder.Detector{
		reminder.NewDeniedToolRetryDetector(),
		reminder.NewCompletedTodosDetector(),
		reminder.NewPlanNotExecutedDetector(),
		reminder.NewUnprocessedSubagentDetector(),
	}

	for _, det := range detectors {
		t.Run(det.Name(), func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, 3, det.MaxFires())
		})
	}
}

// TestNewDetectors_Names verifies unique names for all new detectors.
// Kills mutant: duplicate names would cause template lookup collisions.
func TestNewDetectors_Names(t *testing.T) {
	t.Parallel()

	expected := map[string]reminder.Detector{
		"denied_tool_retry":    reminder.NewDeniedToolRetryDetector(),
		"completed_todos":      reminder.NewCompletedTodosDetector(),
		"plan_not_executed":    reminder.NewPlanNotExecutedDetector(),
		"unprocessed_subagent": reminder.NewUnprocessedSubagentDetector(),
	}

	for name, det := range expected {
		assert.Equal(t, name, det.Name())
	}
}

// TestDefaultDetectors verifies 8 detectors are returned.
// Kills mutant: missing detector would leave a pattern undetected.
func TestDefaultDetectors(t *testing.T) {
	t.Parallel()

	detectors := reminder.DefaultDetectors()
	assert.Len(t, detectors, 8)
}

// TestDefaultTemplates verifies all 8 templates exist.
// Kills mutant: missing template would produce no reminder for a detector.
func TestDefaultTemplates(t *testing.T) {
	t.Parallel()

	templates := reminder.DefaultTemplates()
	assert.Len(t, templates, 8)

	for _, det := range reminder.DefaultDetectors() {
		_, ok := templates[det.Name()]
		assert.True(t, ok, "template missing for detector %q", det.Name())
	}
}
