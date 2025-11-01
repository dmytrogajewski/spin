package curator

import (
	"context"
	"testing"

	"github.com/dmytrogajewski/spin/internal/ace/embedding"
	"github.com/dmytrogajewski/spin/internal/ace/generator"
	"github.com/dmytrogajewski/spin/internal/ace/playbook"
	"github.com/dmytrogajewski/spin/internal/ace/reflector"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCurator_Integration_WithReflector tests end-to-end Reflector → Curator flow
func TestCurator_Integration_WithReflector(t *testing.T) {
	ctx := context.Background()

	// Setup
	embedder := embedding.NewMockEmbedder(384)
	pb := playbook.New(nil, embedder)
	mockLLM := llm.NewMockProvider("test")

	// Mock LLM response with insights
	mockLLM.SetResponse(`[
		{
			"content": "Always validate input parameters before processing to prevent nil pointer errors and security issues",
			"evidence": ["validation prevented error"],
			"confidence": 0.9,
			"category": "success_pattern"
		},
		{
			"content": "Use errors.Is and errors.As for error type checking in Go instead of direct comparison",
			"evidence": ["errors.Is worked correctly"],
			"confidence": 0.85,
			"category": "success_pattern"
		}
	]`)

	// Create reflector and curator
	ref := reflector.NewReflector(mockLLM)
	cur := NewCurator(pb, embedder)

	// Step 1: Reflect on trajectory
	traj := &generator.Trajectory{
		ID:      "integration-test-1",
		Query:   "How to validate inputs?",
		Output:  "Always validate before processing",
		Success: true,
	}

	reflectReq := reflector.ReflectionRequest{
		Trajectories: []*generator.Trajectory{traj},
	}

	reflectResp, err := ref.Reflect(ctx, reflectReq)
	require.NoError(t, err)
	require.Len(t, reflectResp.Insights, 2)

	// Step 2: Curate insights into playbook
	curateReq := MergeRequest{
		Insights: reflectResp.Insights,
	}

	curateResp, err := cur.Curate(ctx, curateReq)
	require.NoError(t, err)
	assert.Equal(t, 2, curateResp.Added)
	assert.Equal(t, 0, curateResp.Skipped)

	// Verify bullets were added to playbook
	bullets := pb.List(nil)
	assert.Len(t, bullets, 2)

	// Verify bullet properties
	for _, bullet := range bullets {
		assert.NotEmpty(t, bullet.Content)
		assert.NotEmpty(t, bullet.Embedding)
		assert.Greater(t, bullet.HelpfulCount, 0) // From confidence scaling
	}
}

// TestCurator_Integration_IdempotentCuration tests that curating twice doesn't duplicate
func TestCurator_Integration_IdempotentCuration(t *testing.T) {
	ctx := context.Background()

	embedder := embedding.NewMockEmbedder(384)
	pb := playbook.New(nil, embedder)
	mockLLM := llm.NewMockProvider("test")

	mockLLM.SetResponse(`[
		{
			"content": "Always use defer for resource cleanup in Go to prevent leaks even on panic",
			"evidence": ["defer prevented leak"],
			"confidence": 0.95,
			"category": "success_pattern"
		}
	]`)

	ref := reflector.NewReflector(mockLLM)
	cur := NewCurator(pb, embedder)

	traj := &generator.Trajectory{
		ID:      "idempotent-test",
		Query:   "Resource cleanup?",
		Output:  "Use defer",
		Success: true,
	}

	// First reflection and curation
	reflectReq := reflector.ReflectionRequest{
		Trajectories: []*generator.Trajectory{traj},
	}

	reflectResp1, _ := ref.Reflect(ctx, reflectReq)
	curateReq1 := MergeRequest{Insights: reflectResp1.Insights}
	curateResp1, err := cur.Curate(ctx, curateReq1)
	require.NoError(t, err)
	assert.Equal(t, 1, curateResp1.Added)

	// Capture initial helpful count before second curation
	initialCount := curateResp1.AddedBullets[0].HelpfulCount

	// Second reflection and curation (same trajectory)
	reflectResp2, _ := ref.Reflect(ctx, reflectReq)
	curateReq2 := MergeRequest{Insights: reflectResp2.Insights}
	curateResp2, err := cur.Curate(ctx, curateReq2)
	require.NoError(t, err)

	// Should skip duplicate
	assert.Equal(t, 0, curateResp2.Added)
	assert.Equal(t, 1, curateResp2.Skipped)
	assert.Equal(t, 1, curateResp2.Updated)

	// Playbook should still have only 1 bullet
	bullets := pb.List(nil)
	assert.Len(t, bullets, 1)

	// Bullet's helpful count should have been incremented by 1
	assert.Equal(t, initialCount+1, bullets[0].HelpfulCount)
}

// TestCurator_Integration_MultipleTrajectories tests curating insights from multiple trajectories
func TestCurator_Integration_MultipleTrajectories(t *testing.T) {
	ctx := context.Background()

	embedder := embedding.NewMockEmbedder(384)
	pb := playbook.New(nil, embedder)
	mockLLM := llm.NewMockProvider("test")

	// Mock batch reflection response
	mockLLM.SetResponse(`[
		{
			"content": "Always use table-driven tests with subtests for better organization and parallel execution",
			"evidence": ["Multiple trajectories used table-driven tests successfully"],
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

	ref := reflector.NewReflector(mockLLM)
	cur := NewCurator(pb, embedder)

	// Multiple trajectories
	trajectories := []*generator.Trajectory{
		{
			ID:      "multi-1",
			Query:   "Testing patterns",
			Output:  "Use table-driven tests",
			Success: true,
		},
		{
			ID:      "multi-2",
			Query:   "Error handling",
			Output:  "Return errors, not panic",
			Success: false,
		},
	}

	// Reflect on batch
	reflectReq := reflector.ReflectionRequest{
		Trajectories: trajectories,
	}

	reflectResp, err := ref.Reflect(ctx, reflectReq)
	require.NoError(t, err)

	// Curate all insights
	curateReq := MergeRequest{Insights: reflectResp.Insights}
	curateResp, err := cur.Curate(ctx, curateReq)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, curateResp.Added, 2)
	assert.Equal(t, 0, curateResp.Skipped) // No duplicates in first curation

	// Verify different categories were preserved
	bullets := pb.List(nil)
	categories := make(map[string]bool)
	for _, b := range bullets {
		if cat, ok := b.Tags["category"]; ok {
			categories[cat] = true
		}
	}
	assert.True(t, len(categories) > 0)
}
