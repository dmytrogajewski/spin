package reflector

import (
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/ace/generator"
	"github.com/dmytrogajewski/spin/internal/ace/trajectory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPromptBuilder_New tests creating a new prompt builder
func TestPromptBuilder_New(t *testing.T) {
	builder := NewPromptBuilder()

	require.NotNil(t, builder)
}

// TestPromptBuilder_BuildSingleTrajectory tests building prompt for one trajectory
func TestPromptBuilder_BuildSingleTrajectory(t *testing.T) {
	builder := NewPromptBuilder()

	traj := &generator.Trajectory{
		ID:      "test-1",
		Query:   "How to handle errors in Go?",
		Output:  "Use errors.Is for type checking",
		Success: true,
	}

	prompt := builder.BuildSingleTrajectory(traj)

	require.NotEmpty(t, prompt)
	assert.Contains(t, prompt, "How to handle errors in Go?")
	assert.Contains(t, prompt, "Use errors.Is for type checking")
	assert.Contains(t, prompt, "Success: true")
	assert.Contains(t, prompt, "diagnose")
	assert.Contains(t, prompt, "actionable insights")
	assert.Contains(t, prompt, "JSON")
}

// TestPromptBuilder_BuildSingleTrajectory_WithRetrievalEvents tests prompt includes retrieval events
func TestPromptBuilder_BuildSingleTrajectory_WithRetrievalEvents(t *testing.T) {
	builder := NewPromptBuilder()

	// Create trajectory with retrieval events
	traj := &generator.Trajectory{
		ID:      "test-1",
		Query:   "install nodejs",
		Output:  "Failed to install",
		Success: false,
		Metadata: generator.TrajectoryMetadata{
			RetrievalEvents: []trajectory.RetrievalEvent{
				{
					Turn:         0,
					Trigger:      trajectory.TriggerInitial,
					Query:        "install nodejs",
					BulletsAdded: []string{"b1", "b2", "b3"},
					Timestamp:    time.Now(),
				},
				{
					Turn:         5,
					Trigger:      trajectory.TriggerError,
					Query:        "install nodejs Error: command not found",
					BulletsAdded: []string{"b4", "b5"},
					Timestamp:    time.Now(),
				},
			},
		},
	}

	prompt := builder.BuildSingleTrajectory(traj)

	require.NotEmpty(t, prompt)

	// Verify retrieval events section exists
	assert.Contains(t, prompt, "Retrieval Events:")
	assert.Contains(t, prompt, "when and why bullets were retrieved")

	// Verify first event details
	assert.Contains(t, prompt, "Turn 0")
	assert.Contains(t, prompt, "[initial]")
	assert.Contains(t, prompt, "install nodejs")
	assert.Contains(t, prompt, "Retrieved 3 bullets")

	// Verify second event details
	assert.Contains(t, prompt, "Turn 5")
	assert.Contains(t, prompt, "[error]")
	assert.Contains(t, prompt, "Error: command not found")
	assert.Contains(t, prompt, "Retrieved 2 bullets")
}

// TestPromptBuilder_BuildSingleTrajectory_NoRetrievalEvents tests backward compatibility
func TestPromptBuilder_BuildSingleTrajectory_NoRetrievalEvents(t *testing.T) {
	builder := NewPromptBuilder()

	// Create trajectory without retrieval events (nil)
	traj := &generator.Trajectory{
		ID:      "test-1",
		Query:   "install nodejs",
		Output:  "Success",
		Success: true,
		// No Metadata.RetrievalEvents set (nil)
	}

	prompt := builder.BuildSingleTrajectory(traj)

	require.NotEmpty(t, prompt)

	// Verify prompt builds successfully
	assert.Contains(t, prompt, "install nodejs")
	assert.Contains(t, prompt, "Success: true")

	// Verify no retrieval events section
	assert.NotContains(t, prompt, "Retrieval Events:")
	assert.NotContains(t, prompt, "when and why bullets were retrieved")
}

// TestPromptBuilder_BuildSingleTrajectory_EmptyRetrievalEvents tests empty events array
func TestPromptBuilder_BuildSingleTrajectory_EmptyRetrievalEvents(t *testing.T) {
	builder := NewPromptBuilder()

	// Create trajectory with empty retrieval events
	traj := &generator.Trajectory{
		ID:      "test-1",
		Query:   "test query",
		Output:  "test output",
		Success: true,
		Metadata: generator.TrajectoryMetadata{
			RetrievalEvents: []trajectory.RetrievalEvent{}, // Empty slice
		},
	}

	prompt := builder.BuildSingleTrajectory(traj)

	require.NotEmpty(t, prompt)

	// Verify prompt builds successfully
	assert.Contains(t, prompt, "test query")

	// Verify no retrieval events section (empty array shouldn't show section)
	assert.NotContains(t, prompt, "Retrieval Events:")
}

// TestPromptBuilder_PromptHasInstructions tests that prompt includes task instructions
func TestPromptBuilder_PromptHasInstructions(t *testing.T) {
	builder := NewPromptBuilder()

	traj := &generator.Trajectory{
		ID:      "test-1",
		Query:   "Test query",
		Output:  "Test output",
		Success: true,
	}

	prompt := builder.BuildSingleTrajectory(traj)

	// Should include instructions
	assert.Contains(t, prompt, "Task")
	assert.Contains(t, prompt, "reasoning")
	assert.Contains(t, prompt, "error_identification")
	assert.Contains(t, prompt, "root_cause_analysis")
	assert.Contains(t, prompt, "correct_approach")
	assert.Contains(t, prompt, "key_insight")
	assert.Contains(t, prompt, "confidence")
	assert.Contains(t, prompt, "category")
}

// TestPromptBuilder_BuildRefinementPrompt tests refinement prompt generation
func TestPromptBuilder_BuildRefinementPrompt(t *testing.T) {
	builder := NewPromptBuilder()

	insights := []*Insight{
		{
			Content:    "Always validate input parameters before processing them",
			Confidence: 0.8,
			Category:   CategorySuccessPattern,
			Evidence:   []string{"validation prevented error"},
		},
	}

	prompt := builder.BuildRefinementPrompt(insights)

	require.NotEmpty(t, prompt)
	assert.Contains(t, prompt, "Always validate input parameters")
	assert.Contains(t, prompt, "Refine")
	assert.Contains(t, prompt, "specific")
	assert.Contains(t, prompt, "actionable")
	assert.Contains(t, prompt, "JSON")
}

// TestPromptBuilder_BuildBatchTrajectory tests batch trajectory prompt generation
func TestPromptBuilder_BuildBatchTrajectory(t *testing.T) {
	builder := NewPromptBuilder()

	trajs := []*generator.Trajectory{
		{
			ID:      "test-1",
			Query:   "How to handle errors?",
			Output:  "Use errors.Is",
			Success: true,
		},
		{
			ID:      "test-2",
			Query:   "Error handling best practices",
			Output:  "Avoid panic",
			Success: true,
		},
	}

	prompt := builder.BuildBatchTrajectory(trajs)

	require.NotEmpty(t, prompt)
	assert.Contains(t, prompt, "test-1")
	assert.Contains(t, prompt, "test-2")
	assert.Contains(t, prompt, "How to handle errors?")
	assert.Contains(t, prompt, "Error handling best practices")
	assert.Contains(t, prompt, "patterns")
	assert.Contains(t, prompt, "multiple")
	assert.Contains(t, prompt, "JSON")
}
