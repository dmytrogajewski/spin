package refine

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/ace/bullet"
	"github.com/dmytrogajewski/spin/internal/ace/embedding"
	"github.com/dmytrogajewski/spin/internal/ace/playbook"
)

// TestIntegration_FullRefinementWorkflow tests the complete grow-and-refine mechanism.
func TestIntegration_FullRefinementWorkflow(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	embedder := embedding.NewMockEmbedder(384)
	pb := playbook.New(nil, embedder)

	archive := NewArchive()
	mergeEngine := NewMergeEngine(embedder, 0.90)
	pruneFunc := makePruneFunc(pb, 0.1)
	orchestrator := NewRefinementOrchestrator(pb, mergeEngine, archive, pruneFunc)

	addEmbeddedBullet(t, ctx, pb, embedder, "Always validate user input", 3, 0)  // b1: high utility
	addEmbeddedBullet(t, ctx, pb, embedder, "Always validate user input", 1, 0)  // b2: duplicate of b1
	addEmbeddedBullet(t, ctx, pb, embedder, "Low utility content", 0, 3)         // b3: low utility
	addEmbeddedBullet(t, ctx, pb, embedder, "Use errors.Is for error checking", 2, 0) // b4: unique

	initialCount := pb.Stats().TotalBullets
	require.Equal(t, 4, initialCount, "expected 4 initial bullets")

	result, err := orchestrator.Refine(ctx, RefinementRequest{
		PruneEnabled: true, MergeEnabled: true, ArchiveEnabled: true,
		MinUtility: 0.1, MergeSimilarity: 0.90,
	})
	require.NoError(t, err, "refinement failed")

	if result.Merged == 0 {
		t.Error("expected at least 1 merge (b1 and b2 are identical)")
	}

	finalCount := pb.Stats().TotalBullets
	if finalCount >= initialCount {
		t.Errorf("expected fewer bullets after refinement, got %d (was %d)", finalCount, initialCount)
	}

	if archive.Len() == 0 {
		t.Error("expected archived bullets")
	}

	if result.TokensSaved <= 0 {
		t.Error("expected positive tokens saved")
	}

	if result.Duration == 0 {
		t.Error("expected non-zero duration")
	}
}

// addEmbeddedBullet creates a bullet with content, helpful/harmful counts, embedding, and adds it to the playbook.
func addEmbeddedBullet(
	t *testing.T, ctx context.Context, pb *playbook.Playbook,
	embedder *embedding.MockEmbedder, content string, helpful, harmful int,
) {
	t.Helper()

	b, err := bullet.New(content)
	require.NoError(t, err)

	for range helpful {
		b.IncrementHelpful()
	}

	for range harmful {
		b.IncrementHarmful()
	}

	emb, err := embedder.Embed(ctx, b.Content)
	require.NoError(t, err)

	b.Embedding = emb

	err = pb.Add(ctx, b)
	require.NoError(t, err)
}

// makePruneFunc creates a prune function that removes bullets below minUtility score.
func makePruneFunc(pb *playbook.Playbook, minUtilityScore float64) func(context.Context) (int, []string, error) {
	return func(ctx context.Context) (int, []string, error) {
		bullets := pb.List(nil)
		prunedIDs := make([]string, 0)

		for _, b := range bullets {
			if b.Score() < minUtilityScore {
				if err := pb.Delete(ctx, b.ID); err != nil {
					return 0, nil, err
				}

				prunedIDs = append(prunedIDs, b.ID)
			}
		}

		return len(prunedIDs), prunedIDs, nil
	}
}

// TestIntegration_GrowthMonitoring tests growth tracking and refinement triggering.
func TestIntegration_GrowthMonitoring(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pb := playbook.New(nil, nil)

	thresholds := GrowthThresholds{
		MaxBullets: 10,
		MaxTokens:  500,
		MinUtility: 0.5,
	}

	monitor := NewGrowthMonitor(pb, thresholds)

	// Add bullets gradually.
	for i := range 5 {
		b, err := bullet.New("Test content")
		require.NoError(t, err)

		b.IncrementHelpful()

		err = pb.Add(ctx, b)
		require.NoError(t, err)

		metrics, needsRefine := monitor.CheckGrowth(ctx)

		if i < 4 && needsRefine {
			t.Errorf("iteration %d: unexpected refinement trigger (below threshold)", i)
		}

		if metrics.BulletCount != i+1 {
			t.Errorf("iteration %d: expected %d bullets, got %d", i, i+1, metrics.BulletCount)
		}
	}

	// Add more to exceed threshold.
	for i := 5; i < 12; i++ {
		b, err := bullet.New("Test content")
		require.NoError(t, err)

		err = pb.Add(ctx, b)
		require.NoError(t, err)

		monitor.CheckGrowth(ctx)
	}

	// Should now trigger refinement.
	if !monitor.ShouldRefine() {
		t.Error("expected refinement needed after exceeding threshold")
	}
}

// TestIntegration_MergeOnly tests merge-only refinement.
func TestIntegration_MergeOnly(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	embedder := embedding.NewMockEmbedder(384)
	pb := playbook.New(nil, embedder)

	archive := NewArchive()
	mergeEngine := NewMergeEngine(embedder, 0.90)
	orchestrator := NewRefinementOrchestrator(pb, mergeEngine, archive, nil) // No curator.

	// Add identical bullets.
	b1, err := bullet.New("Identical content")
	require.NoError(t, err)

	b1.IncrementHelpful()

	emb1, err := embedder.Embed(ctx, b1.Content)
	require.NoError(t, err)

	b1.Embedding = emb1

	err = pb.Add(ctx, b1)
	require.NoError(t, err)

	b2, err := bullet.New("Identical content")
	require.NoError(t, err)

	emb2, err := embedder.Embed(ctx, b2.Content)
	require.NoError(t, err)

	b2.Embedding = emb2

	err = pb.Add(ctx, b2)
	require.NoError(t, err)

	result, err := orchestrator.Refine(ctx, RefinementRequest{
		PruneEnabled:   false, // Merge only.
		MergeEnabled:   true,
		ArchiveEnabled: true,
	})
	if err != nil {
		t.Fatalf("refinement failed: %v", err)
	}

	if result.Merged != 1 {
		t.Errorf("expected 1 merge, got %d", result.Merged)
	}

	if result.Pruned != 0 {
		t.Errorf("expected 0 prunes (disabled), got %d", result.Pruned)
	}

	if pb.Stats().TotalBullets != 1 {
		t.Errorf("expected 1 bullet after merge, got %d", pb.Stats().TotalBullets)
	}
}

// TestIntegration_PruneOnly tests prune-only refinement.
func TestIntegration_PruneOnly(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	embedder := embedding.NewMockEmbedder(384)
	pb := playbook.New(nil, embedder)

	// Create prune function for testing.
	pruneFunc := func(ctx context.Context) (int, []string, error) {
		bullets := pb.List(nil)
		prunedIDs := make([]string, 0)
		minUtilityScore := 0.1

		for _, b := range bullets {
			score := b.Score()
			if score < minUtilityScore {
				err := pb.Delete(ctx, b.ID)
				if err != nil {
					return 0, nil, err
				}

				prunedIDs = append(prunedIDs, b.ID)
			}
		}

		return len(prunedIDs), prunedIDs, nil
	}

	orchestrator := NewRefinementOrchestrator(pb, nil, nil, pruneFunc) // No merge engine or archive.

	// Add low utility bullet.
	b, err := bullet.New("Low utility")
	require.NoError(t, err)

	b.IncrementHarmful()
	b.IncrementHarmful()
	b.IncrementHarmful()

	err = pb.Add(ctx, b)
	require.NoError(t, err)

	result, err := orchestrator.Refine(ctx, RefinementRequest{
		PruneEnabled:   true,
		MergeEnabled:   false, // Prune only.
		ArchiveEnabled: false,
	})
	if err != nil {
		t.Fatalf("refinement failed: %v", err)
	}

	if result.Pruned != 1 {
		t.Errorf("expected 1 prune, got %d", result.Pruned)
	}

	if result.Merged != 0 {
		t.Errorf("expected 0 merges (disabled), got %d", result.Merged)
	}
}

// TestIntegration_NoRefinementNeeded tests when no refinement occurs.
func TestIntegration_NoRefinementNeeded(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	embedder := embedding.NewMockEmbedder(384)
	pb := playbook.New(nil, embedder)

	archive := NewArchive()
	mergeEngine := NewMergeEngine(embedder, 0.99)                            // Very high threshold to prevent merging.
	orchestrator := NewRefinementOrchestrator(pb, mergeEngine, archive, nil) // No curator to avoid pruning.

	// Add high utility, unique bullets.
	b1, err := bullet.New("Unique content 1")
	require.NoError(t, err)

	b1.IncrementHelpful()
	b1.IncrementHelpful()

	err = pb.Add(ctx, b1)
	require.NoError(t, err)

	b2, err := bullet.New("Unique content 2")
	require.NoError(t, err)

	b2.IncrementHelpful()
	b2.IncrementHelpful()

	err = pb.Add(ctx, b2)
	require.NoError(t, err)

	initialCount := pb.Stats().TotalBullets

	result, err := orchestrator.Refine(ctx, RefinementRequest{
		PruneEnabled:   false, // No pruning (no curator).
		MergeEnabled:   false, // No merging.
		ArchiveEnabled: false,
	})
	if err != nil {
		t.Fatalf("refinement failed: %v", err)
	}

	// No operations enabled - nothing should change.
	if result.Merged != 0 {
		t.Errorf("expected 0 merges (disabled), got %d", result.Merged)
	}

	if result.Pruned != 0 {
		t.Errorf("expected 0 prunes (disabled), got %d", result.Pruned)
	}

	if pb.Stats().TotalBullets != initialCount {
		t.Errorf("expected %d bullets (no refinement), got %d", initialCount, pb.Stats().TotalBullets)
	}
}
