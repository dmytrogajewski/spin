package curator

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/ace/embedding"
	"github.com/dmytrogajewski/spin/internal/ace/playbook"
	"github.com/dmytrogajewski/spin/internal/ace/reflector"
)

// TestInterfaceSatisfaction verifies that the curator struct implements
// all the segregated interfaces at compile time.
func TestInterfaceSatisfaction(t *testing.T) {
	t.Parallel()

	// These assignments verify interface satisfaction at compile time.
	// If curator does not implement an interface, this test will not compile.
	var (
		_ BulletMerger  = (*curator)(nil)
		_ BulletRefiner = (*curator)(nil)
		_ BulletUpdater = (*curator)(nil)
		_ Curator       = (*curator)(nil)
	)
}

// TestNewCurator tests creating a new curator.
func TestNewCurator(t *testing.T) {
	t.Parallel()

	embedder := embedding.NewMockEmbedder(384)
	pb := playbook.New(nil, embedder)

	curator := NewCurator(pb, embedder)

	require.NotNil(t, curator)
}

// TestCurator_Curate_NewBullets tests adding new insights to empty playbook.
func TestCurator_Curate_NewBullets(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	embedder := embedding.NewMockEmbedder(384)
	pb := playbook.New(nil, embedder)

	curator := NewCurator(pb, embedder)

	insights := []*reflector.Insight{
		{
			Content:    "Always validate input parameters before processing them",
			Source:     "traj-123",
			Confidence: 0.85,
			Category:   reflector.CategorySuccessPattern,
		},
	}

	req := MergeRequest{
		Insights:            insights,
		SimilarityThreshold: 0.85,
	}

	result, err := curator.Curate(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 1, result.Added)
	assert.Equal(t, 0, result.Skipped)
	assert.Equal(t, 0, result.Updated)
	assert.Len(t, result.AddedBullets, 1)
}

// TestCurator_Curate_MultipleBullets tests adding multiple insights.
func TestCurator_Curate_MultipleBullets(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	embedder := embedding.NewMockEmbedder(384)
	pb := playbook.New(nil, embedder)

	curator := NewCurator(pb, embedder)

	insights := []*reflector.Insight{
		{
			Content:    "Always validate input parameters before processing them",
			Confidence: 0.85,
			Category:   reflector.CategorySuccessPattern,
		},
		{
			Content:    "Use errors.Is for error type checking",
			Confidence: 0.90,
			Category:   reflector.CategorySuccessPattern,
		},
	}

	req := MergeRequest{
		Insights:            insights,
		SimilarityThreshold: 0.85,
	}

	result, err := curator.Curate(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, 2, result.Added)
	assert.Equal(t, 0, result.Skipped)
}

// TestCurator_Curate_EmptyInsights tests empty insights list.
func TestCurator_Curate_EmptyInsights(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	embedder := embedding.NewMockEmbedder(384)
	pb := playbook.New(nil, embedder)

	curator := NewCurator(pb, embedder)

	req := MergeRequest{
		Insights:            []*reflector.Insight{},
		SimilarityThreshold: 0.85,
	}

	result, err := curator.Curate(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, 0, result.Added)
	assert.Empty(t, result.AddedBullets)
}

// TestCurator_Curate_WithDeduplication tests duplicate detection during curation.
func TestCurator_Curate_WithDeduplication(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	embedder := embedding.NewMockEmbedder(384)
	pb := playbook.New(nil, embedder)
	curator := NewCurator(pb, embedder)

	// First curate - should add bullet.
	content := "Always validate input parameters before processing them to ensure data integrity and security"
	insights1 := []*reflector.Insight{
		{
			Content:    content,
			Confidence: 0.9,
			Category:   reflector.CategorySuccessPattern,
			Source:     "traj-1",
		},
	}

	req1 := MergeRequest{Insights: insights1}
	result1, err := curator.Curate(ctx, req1)

	require.NoError(t, err)
	assert.Equal(t, 1, result1.Added)
	assert.Equal(t, 0, result1.Skipped)

	// Second curate with duplicate - should skip.
	insights2 := []*reflector.Insight{
		{
			Content:    content, // Exact same content.
			Confidence: 0.85,
			Category:   reflector.CategorySuccessPattern,
			Source:     "traj-2",
		},
	}

	req2 := MergeRequest{Insights: insights2}
	result2, err := curator.Curate(ctx, req2)

	require.NoError(t, err)
	assert.Equal(t, 0, result2.Added)
	assert.Equal(t, 1, result2.Skipped)
	assert.Equal(t, 1, result2.Updated)
	assert.Len(t, result2.Duplicates, 1)
}

// TestCurator_Curate_UpdatesHelpfulCount tests that duplicates increment helpful count.
func TestCurator_Curate_UpdatesHelpfulCount(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	embedder := embedding.NewMockEmbedder(384)
	pb := playbook.New(nil, embedder)
	curator := NewCurator(pb, embedder)

	// First curate - add bullet with helpful count from confidence.
	content := "Always validate input parameters before processing to ensure data integrity and prevent security issues"
	insights1 := []*reflector.Insight{
		{
			Content:    content,
			Confidence: 0.5, // Will create helpful count of 5.
			Category:   reflector.CategorySuccessPattern,
		},
	}

	req1 := MergeRequest{Insights: insights1}
	result1, err := curator.Curate(ctx, req1)
	require.NoError(t, err)

	// Get the added bullet.
	addedBullet := result1.AddedBullets[0]
	initialCount := addedBullet.HelpfulCount

	// Second curate with same content - should increment.
	insights2 := []*reflector.Insight{
		{
			Content:    content, // Exact duplicate.
			Confidence: 0.8,
			Category:   reflector.CategorySuccessPattern,
		},
	}

	req2 := MergeRequest{Insights: insights2}
	result2, err := curator.Curate(ctx, req2)

	require.NoError(t, err)
	assert.Equal(t, 0, result2.Added)   // No new bullets.
	assert.Equal(t, 1, result2.Skipped) // Duplicate skipped.
	assert.Equal(t, 1, result2.Updated) // Existing updated.

	// Verify helpful count was incremented.
	updatedBullet, _ := pb.Get(addedBullet.ID)
	assert.Equal(t, initialCount+1, updatedBullet.HelpfulCount)
}
