package agent

import (
	"strings"
	"testing"

	"github.com/dmytrogajewski/spin/internal/ace/generator"
	"github.com/dmytrogajewski/spin/internal/ace/trajectory"
	"github.com/stretchr/testify/assert"
)

// TestBuildQueryFromContext_Initial verifies that TriggerInitial returns base query only.
func TestBuildQueryFromContext_Initial(t *testing.T) {
	// Setup
	agent := &Agent{}
	ctx := trajectory.NewTrajectoryContext("install nodejs")

	// Execute
	query := agent.buildQueryFromContext(ctx, trajectory.TriggerInitial)

	// Verify
	assert.Equal(t, "install nodejs", query)
}

// TestBuildQueryFromContext_Error verifies that TriggerError includes error patterns.
func TestBuildQueryFromContext_Error(t *testing.T) {
	// Setup
	agent := &Agent{
		config: &Config{
			ACE: ACEConfig{
				Retrieval: ACERetrievalConfig{
					ProgressiveContext: ProgressiveContextConfig{
						ErrorLookback: 5,
					},
				},
			},
		},
	}
	ctx := trajectory.NewTrajectoryContext("install nodejs")
	ctx.AppendSteps([]generator.TrajectoryStep{
		{StepNumber: 0, Type: "tool_call", Content: "Tool: bash"},
		{StepNumber: 1, Type: "tool_result", Content: "Error: command not found"},
	})

	// Execute
	query := agent.buildQueryFromContext(ctx, trajectory.TriggerError)

	// Verify
	assert.Contains(t, query, "install nodejs")
	assert.True(t, strings.Contains(query, "command not found") || strings.Contains(query, "Error"),
		"Query should include error pattern")
}

// TestBuildQueryFromContext_ToolChange verifies that TriggerToolChange includes tool names.
func TestBuildQueryFromContext_ToolChange(t *testing.T) {
	// Setup
	agent := &Agent{
		config: &Config{
			ACE: ACEConfig{
				Retrieval: ACERetrievalConfig{
					ProgressiveContext: ProgressiveContextConfig{
						ToolChangeLookback: 3,
					},
				},
			},
		},
	}
	ctx := trajectory.NewTrajectoryContext("debug app")
	ctx.AppendSteps([]generator.TrajectoryStep{
		{StepNumber: 0, Type: "tool_call", Content: "Tool: Read"},
		{StepNumber: 1, Type: "tool_call", Content: "Tool: Bash"},
	})

	// Execute
	query := agent.buildQueryFromContext(ctx, trajectory.TriggerToolChange)

	// Verify
	assert.Contains(t, query, "debug app")
	assert.True(t, strings.Contains(query, "Read") || strings.Contains(query, "Bash"),
		"Query should include tool names")
}

// TestBuildQueryFromContext_Interval verifies that TriggerInterval includes concepts.
func TestBuildQueryFromContext_Interval(t *testing.T) {
	// Setup
	agent := &Agent{
		config: &Config{},
	}
	ctx := trajectory.NewTrajectoryContext("fix build")
	ctx.AppendSteps([]generator.TrajectoryStep{
		{StepNumber: 0, Type: "reasoning", Content: "Checking Dockerfile syntax"},
		{StepNumber: 1, Type: "reasoning", Content: "BuildKit optimization needed"},
	})

	// Execute
	query := agent.buildQueryFromContext(ctx, trajectory.TriggerInterval)

	// Verify
	assert.Contains(t, query, "fix build")
	assert.True(t, strings.Contains(query, "Dockerfile") || strings.Contains(query, "BuildKit"),
		"Query should include extracted concepts")
}

// TestBuildQueryFromContext_EmptySteps verifies fallback to base query when no steps.
func TestBuildQueryFromContext_EmptySteps(t *testing.T) {
	// Setup
	agent := &Agent{
		config: &Config{
			ACE: ACEConfig{
				Retrieval: ACERetrievalConfig{
					ProgressiveContext: ProgressiveContextConfig{
						ErrorLookback: 5,
					},
				},
			},
		},
	}
	ctx := trajectory.NewTrajectoryContext("test query")
	// No steps appended

	// Execute - try error trigger with no steps
	query := agent.buildQueryFromContext(ctx, trajectory.TriggerError)

	// Verify - should fall back to base query
	assert.Equal(t, "test query", query)
}

// TestBuildQueryFromContext_AllTriggers verifies all triggers work.
func TestBuildQueryFromContext_AllTriggers(t *testing.T) {
	// Setup
	agent := &Agent{
		config: &Config{
			ACE: ACEConfig{
				Retrieval: ACERetrievalConfig{
					ProgressiveContext: ProgressiveContextConfig{
						ErrorLookback:      5,
						ToolChangeLookback: 3,
					},
				},
			},
		},
	}

	tests := []struct {
		name    string
		trigger trajectory.TriggerType
	}{
		{"Initial", trajectory.TriggerInitial},
		{"Error", trajectory.TriggerError},
		{"ToolChange", trajectory.TriggerToolChange},
		{"Interval", trajectory.TriggerInterval},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := trajectory.NewTrajectoryContext("base query")
			ctx.AppendSteps([]generator.TrajectoryStep{
				{Content: "Tool: bash"},
				{Content: "Error: test error"},
			})

			query := agent.buildQueryFromContext(ctx, tt.trigger)

			// All should at least contain base query
			assert.Contains(t, query, "base query")
			// Query should not be empty
			assert.NotEmpty(t, query)
		})
	}
}
