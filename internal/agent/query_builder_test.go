package agent

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dmytrogajewski/spin/internal/ace/generator"
	"github.com/dmytrogajewski/spin/internal/ace/trajectory"
)

// TestBuildQueryFromContext_Initial verifies that TriggerInitial returns base query only.
func TestBuildQueryFromContext_Initial(t *testing.T) {
	t.Parallel()
	// Setup.
	agent := &Agent{}
	ctx := trajectory.NewContext("install nodejs")

	// Execute.
	query := agent.buildQueryFromContext(ctx, trajectory.TriggerInitial)

	// Verify.
	assert.Equal(t, "install nodejs", query)
}

// queryFromContextCase describes a buildQueryFromContext test case.
type queryFromContextCase struct {
	name        string
	config      ProgressiveContextConfig
	queryText   string
	steps       []generator.TrajectoryStep
	trigger     trajectory.TriggerType
	wantContain string
	wantAnyOf   []string
}

func runQueryFromContextTests(t *testing.T, cases []queryFromContextCase) {
	t.Helper()

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			agent := &Agent{
				aceConfig: &ACEConfig{
					Retrieval: ACERetrievalConfig{
						ProgressiveContext: tt.config,
					},
				},
			}
			ctx := trajectory.NewContext(tt.queryText)
			ctx.AppendSteps(tt.steps)

			query := agent.buildQueryFromContext(ctx, tt.trigger)

			assert.Contains(t, query, tt.wantContain)

			if len(tt.wantAnyOf) > 0 {
				found := false

				for _, s := range tt.wantAnyOf {
					if strings.Contains(query, s) {
						found = true

						break
					}
				}

				assert.True(t, found, "Query should include one of %v", tt.wantAnyOf)
			}
		})
	}
}

// TestBuildQueryFromContext_Error verifies that TriggerError includes error patterns.
func TestBuildQueryFromContext_Error(t *testing.T) {
	t.Parallel()

	runQueryFromContextTests(t, []queryFromContextCase{
		{
			name:      "error includes pattern",
			config:    ProgressiveContextConfig{ErrorLookback: 5},
			queryText: "install nodejs",
			steps: []generator.TrajectoryStep{
				{StepNumber: 0, Type: "tool_call", Content: "Tool: bash"},
				{StepNumber: 1, Type: "tool_result", Content: "Error: command not found"},
			},
			trigger:     trajectory.TriggerError,
			wantContain: "install nodejs",
			wantAnyOf:   []string{"command not found", "Error"},
		},
	})
}

// TestBuildQueryFromContext_ToolChange verifies that TriggerToolChange includes tool names.
func TestBuildQueryFromContext_ToolChange(t *testing.T) {
	t.Parallel()

	runQueryFromContextTests(t, []queryFromContextCase{
		{
			name:      "tool change includes tool names",
			config:    ProgressiveContextConfig{ToolChangeLookback: 3},
			queryText: "debug app",
			steps: []generator.TrajectoryStep{
				{StepNumber: 0, Type: "tool_call", Content: "Tool: Read"},
				{StepNumber: 1, Type: "tool_call", Content: "Tool: Bash"},
			},
			trigger:     trajectory.TriggerToolChange,
			wantContain: "debug app",
			wantAnyOf:   []string{"Read", "Bash"},
		},
	})
}

// TestBuildQueryFromContext_Interval verifies that TriggerInterval includes concepts.
func TestBuildQueryFromContext_Interval(t *testing.T) {
	t.Parallel()
	// Setup.
	agent := &Agent{
		aceConfig: &ACEConfig{},
	}
	ctx := trajectory.NewContext("fix build")
	ctx.AppendSteps([]generator.TrajectoryStep{
		{StepNumber: 0, Type: "reasoning", Content: "Checking Dockerfile syntax"},
		{StepNumber: 1, Type: "reasoning", Content: "BuildKit optimization needed"},
	})

	// Execute.
	query := agent.buildQueryFromContext(ctx, trajectory.TriggerInterval)

	// Verify.
	assert.Contains(t, query, "fix build")
	assert.True(t, strings.Contains(query, "Dockerfile") || strings.Contains(query, "BuildKit"),
		"Query should include extracted concepts")
}

// TestBuildQueryFromContext_EmptySteps verifies fallback to base query when no steps.
func TestBuildQueryFromContext_EmptySteps(t *testing.T) {
	t.Parallel()
	// Setup.
	agent := &Agent{
		aceConfig: &ACEConfig{
			Retrieval: ACERetrievalConfig{
				ProgressiveContext: ProgressiveContextConfig{
					ErrorLookback: 5,
				},
			},
		},
	}
	ctx := trajectory.NewContext("test query")
	// No steps appended.

	// Execute - try error trigger with no steps.
	query := agent.buildQueryFromContext(ctx, trajectory.TriggerError)

	// Verify - should fall back to base query.
	assert.Equal(t, "test query", query)
}

// TestBuildQueryFromContext_AllTriggers verifies all triggers work.
func TestBuildQueryFromContext_AllTriggers(t *testing.T) {
	t.Parallel()
	// Setup.
	agent := &Agent{
		aceConfig: &ACEConfig{
			Retrieval: ACERetrievalConfig{
				ProgressiveContext: ProgressiveContextConfig{
					ErrorLookback:      5,
					ToolChangeLookback: 3,
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
			t.Parallel()

			ctx := trajectory.NewContext("base query")
			ctx.AppendSteps([]generator.TrajectoryStep{
				{Content: "Tool: bash"},
				{Content: "Error: test error"},
			})

			query := agent.buildQueryFromContext(ctx, tt.trigger)

			// All should at least contain base query.
			assert.Contains(t, query, "base query")
			// Query should not be empty.
			assert.NotEmpty(t, query)
		})
	}
}
