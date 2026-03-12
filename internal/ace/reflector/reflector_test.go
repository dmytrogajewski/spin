package reflector

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/ace/generator"
	"github.com/dmytrogajewski/spin/internal/llm"
)

// TestNewReflector tests creating a new reflector.
func TestNewReflector(t *testing.T) {
	mockLLM := llm.NewMockProvider("test")

	reflector := NewReflector(mockLLM)

	require.NotNil(t, reflector)
}

// TestReflector_Reflect tests reflection on a single trajectory.
func TestReflector_Reflect(t *testing.T) {
	mockLLM := llm.NewMockProvider("test")
	mockLLM.SetResponse(`[
		{
			"content": "Always use errors.Is for error type checking in Go",
			"evidence": ["Use errors.Is for type checking"],
			"confidence": 0.9,
			"category": "success_pattern"
		}
	]`)

	reflector := NewReflector(mockLLM)

	ctx := context.Background()
	req := ReflectionRequest{
		Trajectories: []*generator.Trajectory{
			{
				ID:      "test-1",
				Query:   "How to handle errors?",
				Output:  "Use errors.Is for type checking",
				Success: true,
			},
		},
	}

	resp, err := reflector.Reflect(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 1, len(resp.Insights))
	assert.Equal(t, "Always use errors.Is for error type checking in Go", resp.Insights[0].Content)
	assert.Equal(t, 0.9, resp.Insights[0].Confidence)
	assert.Equal(t, CategorySuccessPattern, resp.Insights[0].Category)
}

// TestReflector_RefineInsights tests multi-iteration refinement.
func TestReflector_RefineInsights(t *testing.T) {
	mockLLM := llm.NewMockProvider("test")
	mockLLM.SetResponse(`[
		{
			"content": "Always use errors.Is for error type checking in Go to avoid interface comparison issues",
			"evidence": ["Use errors.Is for type checking"],
			"confidence": 0.95,
			"category": "success_pattern"
		}
	]`)

	reflector := NewReflector(mockLLM)

	ctx := context.Background()
	initialInsights := []*Insight{
		{
			Content:    "Always use errors.Is for error type checking in Go",
			Confidence: 0.8,
			Category:   CategorySuccessPattern,
		},
	}

	refined, err := reflector.RefineInsights(ctx, initialInsights, 2)

	require.NoError(t, err)
	require.NotNil(t, refined)
	assert.Equal(t, 1, len(refined))
	assert.Greater(t, refined[0].Confidence, initialInsights[0].Confidence)
	assert.Equal(t, 2, refined[0].Iteration)
}

// TestReflector_RefineInsights_MaxIterations tests iteration limit.
func TestReflector_RefineInsights_MaxIterations(t *testing.T) {
	mockLLM := llm.NewMockProvider("test")
	mockLLM.SetResponse(`[
		{
			"content": "Always use errors.Is for error type checking in Go applications",
			"evidence": ["Use errors.Is for type checking"],
			"confidence": 0.95,
			"category": "success_pattern"
		}
	]`)

	reflector := NewReflector(mockLLM)

	ctx := context.Background()
	initialInsights := []*Insight{
		{
			Content:    "Always use errors.Is for error type checking in Go",
			Confidence: 0.8,
			Category:   CategorySuccessPattern,
		},
	}

	refined, err := reflector.RefineInsights(ctx, initialInsights, 5)

	require.NoError(t, err)
	require.NotNil(t, refined)
	assert.LessOrEqual(t, refined[0].Iteration, 5)
}

// TestReflector_RefineInsights_EmptySlice tests empty input.
func TestReflector_RefineInsights_EmptySlice(t *testing.T) {
	mockLLM := llm.NewMockProvider("test")
	reflector := NewReflector(mockLLM)

	ctx := context.Background()
	refined, err := reflector.RefineInsights(ctx, []*Insight{}, 3)

	require.NoError(t, err)
	assert.Empty(t, refined)
}

// TestReflector_Reflect_MultipleTrajectories tests batch trajectory analysis.
func TestReflector_Reflect_MultipleTrajectories(t *testing.T) {
	mockLLM := llm.NewMockProvider("test")
	mockLLM.SetResponse(`[
		{
			"content": "Always use errors.Is for error type checking in Go to avoid interface comparison issues",
			"evidence": ["Multiple trajectories used errors.Is successfully"],
			"confidence": 0.95,
			"category": "success_pattern"
		},
		{
			"content": "Avoid using panic in library code, return errors instead for better error handling",
			"evidence": ["panic caused issues in trajectory 2"],
			"confidence": 0.85,
			"category": "error_mode"
		}
	]`)

	reflector := NewReflector(mockLLM)

	ctx := context.Background()
	req := ReflectionRequest{
		Trajectories: []*generator.Trajectory{
			{
				ID:      "test-1",
				Query:   "How to handle errors?",
				Output:  "Use errors.Is for type checking",
				Success: true,
			},
			{
				ID:      "test-2",
				Query:   "Error handling best practices",
				Output:  "Avoid panic, return errors",
				Success: true,
			},
		},
	}

	resp, err := reflector.Reflect(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.GreaterOrEqual(t, len(resp.Insights), 1)
}

// TestReflector_Reflect_EmptyTrajectories tests empty trajectory list.
func TestReflector_Reflect_EmptyTrajectories(t *testing.T) {
	mockLLM := llm.NewMockProvider("test")
	reflector := NewReflector(mockLLM)

	ctx := context.Background()
	req := ReflectionRequest{
		Trajectories: []*generator.Trajectory{},
	}

	resp, err := reflector.Reflect(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Empty(t, resp.Insights)
	assert.Equal(t, 0, resp.Iterations)
}

// TestCleanJSONResponse tests extraction of JSON from markdown code blocks.
func TestCleanJSONResponse(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain JSON",
			input:    `[{"content": "test"}]`,
			expected: `[{"content": "test"}]`,
		},
		{
			name:     "JSON with markdown json block",
			input:    "```json\n[{\"content\": \"test\"}]\n```",
			expected: `[{"content": "test"}]`,
		},
		{
			name:     "JSON with markdown generic block",
			input:    "```\n[{\"content\": \"test\"}]\n```",
			expected: `[{"content": "test"}]`,
		},
		{
			name:     "JSON with extra whitespace",
			input:    "  \n```json\n[{\"content\": \"test\"}]\n```\n  ",
			expected: `[{"content": "test"}]`,
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "whitespace only",
			input:    "   \n  ",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanJSONResponse(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestReflector_Reflect_WithMarkdownJSON tests reflection with markdown-wrapped JSON.
func TestReflector_Reflect_WithMarkdownJSON(t *testing.T) {
	mockLLM := llm.NewMockProvider("test")
	// Simulate LLM returning JSON wrapped in markdown code block.
	mockLLM.SetResponse("```json\n" + `[
		{
			"content": "Always use errors.Is for error type checking in Go",
			"evidence": ["Use errors.Is for type checking"],
			"confidence": 0.9,
			"category": "success_pattern"
		}
	]` + "\n```")

	reflector := NewReflector(mockLLM)

	ctx := context.Background()
	req := ReflectionRequest{
		Trajectories: []*generator.Trajectory{
			{
				ID:      "test-1",
				Query:   "How to handle errors?",
				Output:  "Use errors.Is for type checking",
				Success: true,
			},
		},
	}

	resp, err := reflector.Reflect(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 1, len(resp.Insights))
	assert.Equal(t, "Always use errors.Is for error type checking in Go", resp.Insights[0].Content)
	assert.Equal(t, 0.9, resp.Insights[0].Confidence)
	assert.Equal(t, CategorySuccessPattern, resp.Insights[0].Category)
}
