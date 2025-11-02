package refine

import (
	"context"
	"testing"

	"github.com/dmytrogajewski/spin/internal/ace/bullet"
	"github.com/dmytrogajewski/spin/internal/ace/embedding"
	"github.com/dmytrogajewski/spin/internal/ace/playbook"
)

// TestIntegration_FullRefinementWorkflow tests the complete grow-and-refine mechanism.
func TestIntegration_FullRefinementWorkflow(t *testing.T) {
	ctx := context.Background()
	embedder := embedding.NewMockEmbedder(384)
	pb := playbook.New(nil, embedder)

	// Create components
	archive := NewArchive()
	mergeEngine := NewMergeEngine(embedder, 0.90)

	// Create prune function for testing
	pruneFunc := func(ctx context.Context) (int, []string, error) {
		bullets := pb.List(nil)
		prunedIDs := make([]string, 0)
		minUtilityScore := 0.1

		for _, b := range bullets {
			score := b.Score()
			if score < minUtilityScore {
				if err := pb.Delete(ctx, b.ID); err != nil {
					return 0, nil, err
				}
				prunedIDs = append(prunedIDs, b.ID)
			}
		}

		return len(prunedIDs), prunedIDs, nil
	}

	orchestrator := NewRefinementOrchestrator(pb, mergeEngine, archive, pruneFunc)

	// Add bullets to playbook
	// High utility bullets
	b1, _ := bullet.New("Always validate user input")
	b1.IncrementHelpful()
	b1.IncrementHelpful()
	b1.IncrementHelpful()
	emb1, _ := embedder.Embed(ctx, b1.Content)
	b1.Embedding = emb1
	pb.Add(ctx, b1)

	// Similar to b1 (should be merged)
	b2, _ := bullet.New("Always validate user input") // Exact duplicate
	b2.IncrementHelpful()
	emb2, _ := embedder.Embed(ctx, b2.Content)
	b2.Embedding = emb2
	pb.Add(ctx, b2)

	// Low utility bullet (should be pruned)
	b3, _ := bullet.New("Low utility content")
	b3.IncrementHarmful()
	b3.IncrementHarmful()
	b3.IncrementHarmful()
	emb3, _ := embedder.Embed(ctx, b3.Content)
	b3.Embedding = emb3
	pb.Add(ctx, b3)

	// Different bullet (should remain)
	b4, _ := bullet.New("Use errors.Is for error checking")
	b4.IncrementHelpful()
	b4.IncrementHelpful()
	emb4, _ := embedder.Embed(ctx, b4.Content)
	b4.Embedding = emb4
	pb.Add(ctx, b4)

	initialCount := pb.Stats().TotalBullets
	if initialCount != 4 {
		t.Fatalf("expected 4 initial bullets, got %d", initialCount)
	}

	// Execute refinement
	result, err := orchestrator.Refine(ctx, RefinementRequest{
		PruneEnabled:    true,
		MergeEnabled:    true,
		ArchiveEnabled:  true,
		MinUtility:      0.1,
		MergeSimilarity: 0.90,
	})

	if err != nil {
		t.Fatalf("refinement failed: %v", err)
	}

	// Verify results
	if result.Merged == 0 {
		t.Error("expected at least 1 merge (b1 and b2 are identical)")
	}

	// Note: Pruning happens based on curator's RefineMode settings
	// In lazy mode, it only prunes when threshold is reached
	// The test may or may not trigger pruning depending on bullet count

	// Playbook should have fewer bullets
	finalCount := pb.Stats().TotalBullets
	if finalCount >= initialCount {
		t.Errorf("expected fewer bullets after refinement, got %d (was %d)", finalCount, initialCount)
	}

	// Archive should contain removed bullets
	if archive.Len() == 0 {
		t.Error("expected archived bullets")
	}

	// Tokens saved should be positive
	if result.TokensSaved <= 0 {
		t.Error("expected positive tokens saved")
	}

	// Duration should be recorded
	if result.Duration == 0 {
		t.Error("expected non-zero duration")
	}
}

// TestIntegration_GrowthMonitoring tests growth tracking and refinement triggering.
func TestIntegration_GrowthMonitoring(t *testing.T) {
	ctx := context.Background()
	pb := playbook.New(nil, nil)

	thresholds := GrowthThresholds{
		MaxBullets: 10,
		MaxTokens:  500,
		MinUtility: 0.5,
	}

	monitor := NewGrowthMonitor(pb, thresholds)

	// Add bullets gradually
	for i := 0; i < 5; i++ {
		b, _ := bullet.New("Test content")
		b.IncrementHelpful()
		pb.Add(ctx, b)

		metrics, needsRefine := monitor.CheckGrowth(ctx)

		if i < 4 && needsRefine {
			t.Errorf("iteration %d: unexpected refinement trigger (below threshold)", i)
		}

		if metrics.BulletCount != i+1 {
			t.Errorf("iteration %d: expected %d bullets, got %d", i, i+1, metrics.BulletCount)
		}
	}

	// Add more to exceed threshold
	for i := 5; i < 12; i++ {
		b, _ := bullet.New("Test content")
		pb.Add(ctx, b)
		monitor.CheckGrowth(ctx)
	}

	// Should now trigger refinement
	if !monitor.ShouldRefine() {
		t.Error("expected refinement needed after exceeding threshold")
	}
}

// TestIntegration_MergeOnly tests merge-only refinement.
func TestIntegration_MergeOnly(t *testing.T) {
	ctx := context.Background()
	embedder := embedding.NewMockEmbedder(384)
	pb := playbook.New(nil, embedder)

	archive := NewArchive()
	mergeEngine := NewMergeEngine(embedder, 0.90)
	orchestrator := NewRefinementOrchestrator(pb, mergeEngine, archive, nil) // No curator

	// Add identical bullets
	b1, _ := bullet.New("Identical content")
	b1.IncrementHelpful()
	emb1, _ := embedder.Embed(ctx, b1.Content)
	b1.Embedding = emb1
	pb.Add(ctx, b1)

	b2, _ := bullet.New("Identical content")
	emb2, _ := embedder.Embed(ctx, b2.Content)
	b2.Embedding = emb2
	pb.Add(ctx, b2)

	result, err := orchestrator.Refine(ctx, RefinementRequest{
		PruneEnabled:   false, // Merge only
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
	ctx := context.Background()
	embedder := embedding.NewMockEmbedder(384)
	pb := playbook.New(nil, embedder)

	// Create prune function for testing
	pruneFunc := func(ctx context.Context) (int, []string, error) {
		bullets := pb.List(nil)
		prunedIDs := make([]string, 0)
		minUtilityScore := 0.1

		for _, b := range bullets {
			score := b.Score()
			if score < minUtilityScore {
				if err := pb.Delete(ctx, b.ID); err != nil {
					return 0, nil, err
				}
				prunedIDs = append(prunedIDs, b.ID)
			}
		}

		return len(prunedIDs), prunedIDs, nil
	}

	orchestrator := NewRefinementOrchestrator(pb, nil, nil, pruneFunc) // No merge engine or archive

	// Add low utility bullet
	b, _ := bullet.New("Low utility")
	b.IncrementHarmful()
	b.IncrementHarmful()
	b.IncrementHarmful()
	pb.Add(ctx, b)

	result, err := orchestrator.Refine(ctx, RefinementRequest{
		PruneEnabled:   true,
		MergeEnabled:   false, // Prune only
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
	ctx := context.Background()
	embedder := embedding.NewMockEmbedder(384)
	pb := playbook.New(nil, embedder)

	archive := NewArchive()
	mergeEngine := NewMergeEngine(embedder, 0.99)                            // Very high threshold to prevent merging
	orchestrator := NewRefinementOrchestrator(pb, mergeEngine, archive, nil) // No curator to avoid pruning

	// Add high utility, unique bullets
	b1, _ := bullet.New("Unique content 1")
	b1.IncrementHelpful()
	b1.IncrementHelpful()
	pb.Add(ctx, b1)

	b2, _ := bullet.New("Unique content 2")
	b2.IncrementHelpful()
	b2.IncrementHelpful()
	pb.Add(ctx, b2)

	initialCount := pb.Stats().TotalBullets

	result, err := orchestrator.Refine(ctx, RefinementRequest{
		PruneEnabled:   false, // No pruning (no curator)
		MergeEnabled:   false, // No merging
		ArchiveEnabled: false,
	})

	if err != nil {
		t.Fatalf("refinement failed: %v", err)
	}

	// No operations enabled - nothing should change
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
