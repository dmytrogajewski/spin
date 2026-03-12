package reflector

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/ace/generator"
	"github.com/dmytrogajewski/spin/internal/llm"
)

// TestReflector_Integration_WithGenerator tests end-to-end reflection with generator output.
func TestReflector_Integration_WithGenerator(t *testing.T) {
	t.Parallel()

	mockLLM := llm.NewMockProvider("test")
	mockLLM.SetResponse(`[
		{
			"content": "Always validate input parameters before processing to prevent nil pointer errors",
			"evidence": ["Validation prevented nil pointer dereference"],
			"confidence": 0.9,
			"category": "success_pattern"
		}
	]`)

	reflector := NewReflector(mockLLM)

	// Simulate a trajectory from Generator.
	traj := &generator.Trajectory{
		ID:    "integration-test-1",
		Query: "How to validate function inputs in Go?",
		Steps: []generator.TrajectoryStep{
			{
				StepNumber: 0,
				Type:       "reasoning",
				Content:    "Need to check for nil pointers",
			},
			{
				StepNumber: 1,
				Type:       "code",
				Content:    "if input == nil { return err }",
			},
		},
		Output:  "Always check for nil before dereferencing",
		Success: true,
	}

	ctx := context.Background()
	req := ReflectionRequest{
		Trajectories: []*generator.Trajectory{traj},
	}

	resp, err := reflector.Reflect(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Len(t, resp.Insights, 1)
	assert.Equal(t, "integration-test-1", resp.Insights[0].Source)
	assert.Greater(t, resp.Insights[0].Confidence, 0.5)
	assert.NotZero(t, resp.TotalTokens)
	assert.GreaterOrEqual(t, resp.Duration.Microseconds(), int64(0))
}

// TestReflector_Integration_FullWorkflow tests complete reflection workflow.
func TestReflector_Integration_FullWorkflow(t *testing.T) {
	t.Parallel()

	mockLLM := llm.NewMockProvider("test")

	// First response for initial reflection.
	mockLLM.SetResponse(`[
		{
			"content": "Use table-driven tests for better test organization and coverage",
			"evidence": ["Table-driven tests improved readability"],
			"confidence": 0.75,
			"category": "success_pattern"
		}
	]`)

	reflector := NewReflector(mockLLM)

	// Step 1: Initial reflection.
	traj := &generator.Trajectory{
		ID:      "workflow-test-1",
		Query:   "How to write better tests in Go?",
		Output:  "Use table-driven tests with subtests",
		Success: true,
	}

	ctx := context.Background()
	req := ReflectionRequest{
		Trajectories: []*generator.Trajectory{traj},
	}

	resp, err := reflector.Reflect(ctx, req)
	require.NoError(t, err)
	require.Len(t, resp.Insights, 1)

	initialInsight := resp.Insights[0]
	assert.Equal(t, 0, initialInsight.Iteration)

	// Step 2: Refine insights.
	mockLLM.SetResponse(`[
		{
			"content": "Use table-driven tests with t.Run() for better test organization and parallel execution",
			"evidence": ["Table-driven tests improved readability"],
			"confidence": 0.9,
			"category": "success_pattern"
		}
	]`)

	refined, err := reflector.RefineInsights(ctx, resp.Insights, 2)
	require.NoError(t, err)
	require.Len(t, refined, 1)

	refinedInsight := refined[0]
	assert.Equal(t, 2, refinedInsight.Iteration)
	assert.Greater(t, refinedInsight.Confidence, initialInsight.Confidence)
	assert.Greater(t, len(refinedInsight.Content), len(initialInsight.Content))

	// Step 3: Validate quality.
	validator := NewInsightValidator()
	err = validator.Validate(refinedInsight)
	require.NoError(t, err)

	// Step 4: Filter by quality threshold.
	filtered := validator.FilterByQuality(refined, 0.85)
	assert.Len(t, filtered, 1)
	assert.GreaterOrEqual(t, filtered[0].Confidence, 0.85)
}

// TestReflector_Integration_BatchAnalysis tests batch trajectory analysis.
func TestReflector_Integration_BatchAnalysis(t *testing.T) {
	t.Parallel()

	mockLLM := llm.NewMockProvider("test")
	mockLLM.SetResponse(`[
		{
			"content": "Always use defer for resource cleanup in Go to prevent resource leaks even on panic",
			"evidence": ["Trajectory 1 and 2 both used defer successfully", "Prevented file handle leak"],
			"confidence": 0.95,
			"category": "success_pattern"
		},
		{
			"content": "Avoid direct channel closes in goroutines without coordination to prevent panic on send",
			"evidence": ["Trajectory 3 had panic from closed channel"],
			"confidence": 0.85,
			"category": "error_mode"
		}
	]`)

	reflector := NewReflector(mockLLM)

	// Multiple related trajectories.
	trajectories := []*generator.Trajectory{
		{
			ID:      "batch-1",
			Query:   "How to manage file handles?",
			Output:  "Use defer file.Close()",
			Success: true,
		},
		{
			ID:      "batch-2",
			Query:   "Resource cleanup patterns",
			Output:  "Always defer cleanup operations",
			Success: true,
		},
		{
			ID:      "batch-3",
			Query:   "Channel management",
			Output:  "Don't close channels in goroutines",
			Success: false,
		},
	}

	ctx := context.Background()
	req := ReflectionRequest{
		Trajectories: trajectories,
	}

	resp, err := reflector.Reflect(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.GreaterOrEqual(t, len(resp.Insights), 2)

	// Verify insights from batch analysis.
	foundPattern := false
	foundError := false

	for _, insight := range resp.Insights {
		if insight.Category == CategorySuccessPattern {
			foundPattern = true
		}

		if insight.Category == CategoryErrorMode {
			foundError = true
		}

		assert.Equal(t, "batch", insight.Source)
		assert.NotEmpty(t, insight.Evidence)
	}

	assert.True(t, foundPattern, "Should find success patterns")
	assert.True(t, foundError, "Should find error modes")
}

// TestReflector_Integration_QualityFiltering tests quality-based filtering.
func TestReflector_Integration_QualityFiltering(t *testing.T) {
	t.Parallel()

	mockLLM := llm.NewMockProvider("test")
	mockLLM.SetResponse(`[
		{
			"content": "Always use errors.Is and errors.As for error type checking in Go",
			"evidence": ["Successful error handling in all cases"],
			"confidence": 0.95,
			"category": "success_pattern"
		},
		{
			"content": "Consider using context.Context for cancellation in long-running operations",
			"evidence": ["One trajectory used context"],
			"confidence": 0.6,
			"category": "optimization"
		},
		{
			"content": "Avoid using panic in library code, prefer returning errors for better composability",
			"evidence": ["Limited evidence from one failure"],
			"confidence": 0.4,
			"category": "anti_pattern"
		}
	]`)

	reflector := NewReflector(mockLLM)

	traj := &generator.Trajectory{
		ID:      "quality-test-1",
		Query:   "Error handling best practices",
		Output:  "Use errors.Is for type checking",
		Success: true,
	}

	ctx := context.Background()
	req := ReflectionRequest{
		Trajectories: []*generator.Trajectory{traj},
	}

	resp, err := reflector.Reflect(ctx, req)
	require.NoError(t, err)
	assert.Len(t, resp.Insights, 3)

	// Filter by quality threshold.
	validator := NewInsightValidator()

	highQuality := validator.FilterByQuality(resp.Insights, 0.8)
	assert.Len(t, highQuality, 1)
	assert.GreaterOrEqual(t, highQuality[0].Confidence, 0.8)

	mediumQuality := validator.FilterByQuality(resp.Insights, 0.5)
	assert.Len(t, mediumQuality, 2)

	allInsights := validator.FilterByQuality(resp.Insights, 0.0)
	assert.Len(t, allInsights, 3)
}

// TestReflector_Integration_ErrorHandling tests error handling in integration scenarios.
func TestReflector_Integration_ErrorHandling(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		trajectories []*generator.Trajectory
		wantErr      bool
		wantEmpty    bool
	}{
		{
			name:         "empty trajectories",
			trajectories: []*generator.Trajectory{},
			wantErr:      false,
			wantEmpty:    true,
		},
		{
			name: "incomplete trajectory",
			trajectories: []*generator.Trajectory{
				{
					ID:     "incomplete-1",
					Query:  "",
					Output: "",
				},
			},
			wantErr:   false,
			wantEmpty: false,
		},
	}

	mockLLM := llm.NewMockProvider("test")
	mockLLM.SetResponse(`[
		{
			"content": "Handle edge cases in input validation to prevent unexpected behavior",
			"evidence": ["Edge case handling"],
			"confidence": 0.7,
			"category": "success_pattern"
		}
	]`)

	reflector := NewReflector(mockLLM)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			req := ReflectionRequest{
				Trajectories: tt.trajectories,
			}

			resp, err := reflector.Reflect(ctx, req)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				if tt.wantEmpty {
					assert.Empty(t, resp.Insights)
				}
			}
		})
	}
}
