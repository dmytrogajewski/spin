package curator

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/ace/bullet"
	"github.com/dmytrogajewski/spin/internal/ace/embedding"
	"github.com/dmytrogajewski/spin/internal/ace/playbook"
	"github.com/dmytrogajewski/spin/internal/ace/reflector"
)

// TestRefinement_NoRefine tests no refinement strategy.
func TestRefinement_NoRefine(t *testing.T) {
	ctx := context.Background()
	embedder := embedding.NewMockEmbedder(384)
	pb := playbook.New(nil, embedder)

	// Default curator has no refinement.
	cur := NewCurator(pb, embedder)

	// Add bullets.
	for range 100 {
		b, _ := bullet.New("bullet content")
		pb.Add(ctx, b)
	}

	insights := []*reflector.Insight{
		{Content: "test insight", Confidence: 0.85, Category: reflector.CategorySuccessPattern},
	}

	result, err := cur.Curate(ctx, MergeRequest{Insights: insights})

	require.NoError(t, err)
	assert.False(t, result.Refined, "Should not refine with no refinement mode")
	assert.Nil(t, result.Refinement)
}

// TestRefinement_Lazy_NoAutoRefine tests lazy refinement doesn't auto-refine.
func TestRefinement_Lazy_NoAutoRefine(t *testing.T) {
	ctx := context.Background()
	embedder := embedding.NewMockEmbedder(384)
	pb := playbook.New(nil, embedder)

	cur := NewCurator(pb, embedder,
		WithRefinementMode(RefinementModeLazy, LazyRefinementConfig{
			MinUtilityScore: 0.1,
		}),
	)

	// Add many bullets.
	for range 100 {
		b, _ := bullet.New("bullet content")
		pb.Add(ctx, b)
	}

	insights := []*reflector.Insight{
		{Content: "test insight", Confidence: 0.85, Category: reflector.CategorySuccessPattern},
	}

	result, err := cur.Curate(ctx, MergeRequest{Insights: insights})

	require.NoError(t, err)
	assert.False(t, result.Refined, "Lazy mode should not auto-refine")
	assert.Nil(t, result.Refinement)
}

// TestRefinement_Lazy_ManualRefine tests manual refinement in lazy mode.
func TestRefinement_Lazy_ManualRefine(t *testing.T) {
	ctx := context.Background()
	embedder := embedding.NewMockEmbedder(384)
	pb := playbook.New(nil, embedder)

	cur := NewCurator(pb, embedder,
		WithRefinementMode(RefinementModeLazy, LazyRefinementConfig{
			MinUtilityScore: 0.0, // Prune bullets with score = 0.
		}),
	)

	// Add bullets with low utility (score = 0).
	for range 5 {
		b, _ := bullet.New("low utility bullet")
		pb.Add(ctx, b)
	}

	// Add bullets with high utility.
	for range 5 {
		b, _ := bullet.New("high utility bullet")
		b.IncrementHelpful()
		pb.Add(ctx, b)
	}

	initialCount := pb.Stats().TotalBullets
	assert.Equal(t, 10, initialCount)

	// Manually trigger refinement.
	refinement, err := cur.Refine(ctx)

	require.NoError(t, err)
	assert.Equal(t, 5, refinement.Pruned, "Should prune 5 low-utility bullets")
	assert.Equal(t, 5, len(refinement.PrunedIDs))
	assert.Equal(t, "manual refinement", refinement.Reason)

	// Check playbook was pruned.
	finalCount := pb.Stats().TotalBullets
	assert.Equal(t, 5, finalCount)
}

// TestRefinement_Proactive_Trigger tests proactive refinement triggers at threshold.
func TestRefinement_Proactive_Trigger(t *testing.T) {
	ctx := context.Background()
	embedder := embedding.NewMockEmbedder(384)
	pb := playbook.New(nil, embedder)

	cur := NewCurator(pb, embedder,
		WithRefinementMode(RefinementModeProactive, ProactiveRefinementConfig{
			MaxBullets:      10, // Trigger at 10 bullets.
			MinUtilityScore: 0.0,
		}),
	)

	// Add 5 low-utility bullets.
	for range 5 {
		b, _ := bullet.New("low utility")
		pb.Add(ctx, b)
	}

	// Add 4 high-utility bullets (total = 9, below threshold).
	for range 4 {
		b, _ := bullet.New("high utility")
		b.IncrementHelpful()
		pb.Add(ctx, b)
	}

	assert.Equal(t, 9, pb.Stats().TotalBullets)

	// Add one more bullet to trigger refinement (total = 10).
	insights := []*reflector.Insight{
		{Content: "trigger refinement", Confidence: 0.85, Category: reflector.CategorySuccessPattern},
	}

	result, err := cur.Curate(ctx, MergeRequest{Insights: insights})

	require.NoError(t, err)
	assert.True(t, result.Refined, "Should refine when threshold reached")
	assert.NotNil(t, result.Refinement)
	assert.Equal(t, 5, result.Refinement.Pruned, "Should prune 5 low-utility bullets")
	assert.Equal(t, "proactive refinement", result.Refinement.Reason)

	// Playbook should have 5 bullets remaining (4 high + 1 new).
	assert.Equal(t, 5, pb.Stats().TotalBullets)
}

// TestRefinement_Proactive_NoTrigger tests proactive refinement doesn't trigger below threshold.
func TestRefinement_Proactive_NoTrigger(t *testing.T) {
	ctx := context.Background()
	embedder := embedding.NewMockEmbedder(384)
	pb := playbook.New(nil, embedder)

	cur := NewCurator(pb, embedder,
		WithRefinementMode(RefinementModeProactive, ProactiveRefinementConfig{
			MaxBullets:      100, // High threshold.
			MinUtilityScore: 0.1,
		}),
	)

	// Add only 10 bullets (well below threshold).
	for range 10 {
		b, _ := bullet.New("bullet")
		pb.Add(ctx, b)
	}

	insights := []*reflector.Insight{
		{Content: "test", Confidence: 0.85, Category: reflector.CategorySuccessPattern},
	}

	result, err := cur.Curate(ctx, MergeRequest{Insights: insights})

	require.NoError(t, err)
	assert.False(t, result.Refined, "Should not refine below threshold")
	assert.Nil(t, result.Refinement)
}

// TestRefinement_DefaultConfig tests default configuration values.
func TestRefinement_DefaultConfig(t *testing.T) {
	ctx := context.Background()
	embedder := embedding.NewMockEmbedder(384)
	pb := playbook.New(nil, embedder)

	// Test lazy with defaults.
	curLazy := NewCurator(pb, embedder,
		WithRefinementMode(RefinementModeLazy, nil), WithRefinementMode(RefinementModeLazy, nil), // nil config should use defaults.
	)

	refinement, err := curLazy.Refine(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, refinement.Pruned)

	// Test proactive with defaults.
	curProactive := NewCurator(pb, embedder,
		WithRefinementMode(RefinementModeProactive, nil), WithRefinementMode(RefinementModeProactive, nil), // nil config should use defaults.
	)

	insights := []*reflector.Insight{
		{Content: "test", Confidence: 0.85, Category: reflector.CategorySuccessPattern},
	}

	result, err := curProactive.Curate(ctx, MergeRequest{Insights: insights})
	require.NoError(t, err)
	assert.False(t, result.Refined, "Should not refine with only 1 bullet (default threshold 1000)")
}
