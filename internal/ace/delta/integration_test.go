package delta

import (
	"context"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/ace/bullet"
	"github.com/dmytrogajewski/spin/internal/ace/playbook"
)

// TestIntegration_FullWorkflow tests the complete delta workflow:
// 1. Create bullets in playbook
// 2. Apply various delta operations
// 3. Verify history tracking
// 4. Query deltas by bullet
// 5. Get recent deltas.
func TestIntegration_FullWorkflow(t *testing.T) {
	ctx := context.Background()
	pb := playbook.New(nil, nil)
	applier := NewDeltaApplier(pb)

	// Create initial bullets.
	b1, _ := bullet.New("Initial content for bullet 1")
	b2, _ := bullet.New("Initial content for bullet 2")
	b3, _ := bullet.New("Initial content for bullet 3")

	pb.Add(ctx, b1)
	pb.Add(ctx, b2)
	pb.Add(ctx, b3)

	// Apply various deltas.
	deltas := []Delta{
		*NewContentUpdate(b1.ID, "Updated content", DeltaMetadata{
			Source: "reflector",
			Reason: "Content refinement",
		}),
		*NewIncrementHelpful(b1.ID, DeltaMetadata{
			Source: "curator",
			Reason: "Duplicate insight",
		}),
		*NewAddTag(b1.ID, "category", "testing", DeltaMetadata{
			Source: "adapter",
			Reason: "Categorization",
		}),
		*NewIncrementHelpful(b2.ID, DeltaMetadata{
			Source: "curator",
			Reason: "Useful pattern",
		}),
		*NewIncrementHarmful(b2.ID, DeltaMetadata{
			Source: "feedback",
			Reason: "Misleading advice",
		}),
		*NewUpdateEmbedding(b3.ID, []float32{0.1, 0.2, 0.3}, DeltaMetadata{
			Source: "embedder",
			Reason: "Re-embedding",
		}),
	}

	for _, delta := range deltas {
		result, err := applier.Apply(ctx, delta)
		if err != nil {
			t.Fatalf("failed to apply delta: %v", err)
		}

		if !result.Success {
			t.Errorf("delta application failed: %v", result.Error)
		}
	}

	// Verify bullet 1 state.
	updated1, _ := pb.Get(b1.ID)
	if updated1.Content != "Updated content" {
		t.Errorf("bullet 1: expected content 'Updated content', got '%s'", updated1.Content)
	}

	if updated1.HelpfulCount != 1 {
		t.Errorf("bullet 1: expected helpful count 1, got %d", updated1.HelpfulCount)
	}

	if updated1.Tags["category"] != "testing" {
		t.Errorf("bullet 1: expected tag category=testing, got %s", updated1.Tags["category"])
	}

	// Verify bullet 2 state.
	updated2, _ := pb.Get(b2.ID)
	if updated2.HelpfulCount != 1 {
		t.Errorf("bullet 2: expected helpful count 1, got %d", updated2.HelpfulCount)
	}

	if updated2.HarmfulCount != 1 {
		t.Errorf("bullet 2: expected harmful count 1, got %d", updated2.HarmfulCount)
	}

	// Verify bullet 3 state.
	updated3, _ := pb.Get(b3.ID)
	if len(updated3.Embedding) != 3 {
		t.Errorf("bullet 3: expected embedding length 3, got %d", len(updated3.Embedding))
	}

	// Verify history tracking.
	history := applier.GetHistory()
	if history.Len() != 6 {
		t.Errorf("expected 6 deltas in history, got %d", history.Len())
	}

	// Query deltas by bullet.
	b1Deltas := history.GetByBullet(b1.ID)
	if len(b1Deltas) != 3 {
		t.Errorf("expected 3 deltas for bullet 1, got %d", len(b1Deltas))
	}

	b2Deltas := history.GetByBullet(b2.ID)
	if len(b2Deltas) != 2 {
		t.Errorf("expected 2 deltas for bullet 2, got %d", len(b2Deltas))
	}

	b3Deltas := history.GetByBullet(b3.ID)
	if len(b3Deltas) != 1 {
		t.Errorf("expected 1 delta for bullet 3, got %d", len(b3Deltas))
	}

	// Get recent deltas.
	recent := history.GetRecent(3)
	if len(recent) != 3 {
		t.Errorf("expected 3 recent deltas, got %d", len(recent))
	}

	// Verify stats.
	stats := history.Stats()
	if stats.TotalDeltas != 6 {
		t.Errorf("stats: expected 6 total deltas, got %d", stats.TotalDeltas)
	}

	if stats.UniqueBullets != 3 {
		t.Errorf("stats: expected 3 unique bullets, got %d", stats.UniqueBullets)
	}

	if stats.DeltasByOperation[OpUpdateContent] != 1 {
		t.Errorf("stats: expected 1 content update, got %d", stats.DeltasByOperation[OpUpdateContent])
	}

	if stats.DeltasByOperation[OpIncrementHelpful] != 2 {
		t.Errorf("stats: expected 2 increment helpful, got %d", stats.DeltasByOperation[OpIncrementHelpful])
	}
}

// TestIntegration_BatchWithHistory tests batch processing with history queries.
func TestIntegration_BatchWithHistory(t *testing.T) {
	ctx := context.Background()
	pb := playbook.New(nil, nil)
	applier := NewDeltaApplier(pb)

	// Create 10 bullets.
	bulletIDs := make([]string, 10)

	for i := range 10 {
		b, _ := bullet.New("Content")
		pb.Add(ctx, b)
		bulletIDs[i] = b.ID
	}

	// Create batch of deltas.
	deltas := make([]Delta, 20)
	for i := range 10 {
		deltas[i*2] = *NewIncrementHelpful(bulletIDs[i], DeltaMetadata{Source: "test"})
		deltas[i*2+1] = *NewAddTag(bulletIDs[i], "batch", "true", DeltaMetadata{Source: "test"})
	}

	// Apply batch.
	req := BatchApplyRequest{
		Deltas:     deltas,
		MaxWorkers: 4,
		Atomic:     false,
	}

	result, err := applier.ApplyBatch(ctx, req)
	if err != nil {
		t.Fatalf("batch apply failed: %v", err)
	}

	if result.Applied != 20 {
		t.Errorf("expected 20 applied, got %d", result.Applied)
	}

	// Verify history.
	history := applier.GetHistory()
	if history.Len() != 20 {
		t.Errorf("expected 20 deltas in history, got %d", history.Len())
	}

	// Verify each bullet has 2 deltas.
	for i, id := range bulletIDs {
		bulletDeltas := history.GetByBullet(id)
		if len(bulletDeltas) != 2 {
			t.Errorf("bullet %d: expected 2 deltas, got %d", i, len(bulletDeltas))
		}
	}

	// Verify stats.
	stats := history.Stats()
	if stats.UniqueBullets != 10 {
		t.Errorf("expected 10 unique bullets, got %d", stats.UniqueBullets)
	}

	if stats.DeltasByOperation[OpIncrementHelpful] != 10 {
		t.Errorf("expected 10 increment helpful, got %d", stats.DeltasByOperation[OpIncrementHelpful])
	}

	if stats.DeltasByOperation[OpAddTag] != 10 {
		t.Errorf("expected 10 add tag, got %d", stats.DeltasByOperation[OpAddTag])
	}
}

// TestIntegration_TimestampQueries tests time-based delta queries.
func TestIntegration_TimestampQueries(t *testing.T) {
	ctx := context.Background()
	pb := playbook.New(nil, nil)
	applier := NewDeltaApplier(pb)

	// Create bullet.
	b, _ := bullet.New("Content")
	pb.Add(ctx, b)

	// Apply delta and record timestamp.
	beforeFirst := time.Now()

	time.Sleep(10 * time.Millisecond)

	delta1 := NewIncrementHelpful(b.ID, DeltaMetadata{Source: "test"})
	applier.Apply(ctx, *delta1)

	afterFirst := time.Now()

	time.Sleep(10 * time.Millisecond)

	delta2 := NewIncrementHelpful(b.ID, DeltaMetadata{Source: "test"})
	applier.Apply(ctx, *delta2)

	afterSecond := time.Now()

	// Query deltas since beforeFirst (should get both).
	deltas := applier.GetHistory().GetSince(beforeFirst)
	if len(deltas) != 2 {
		t.Errorf("expected 2 deltas since beforeFirst, got %d", len(deltas))
	}

	// Query deltas since afterFirst (should get only second).
	deltas = applier.GetHistory().GetSince(afterFirst)
	if len(deltas) != 1 {
		t.Errorf("expected 1 delta since afterFirst, got %d", len(deltas))
	}

	// Query deltas since afterSecond (should get none).
	deltas = applier.GetHistory().GetSince(afterSecond)
	if len(deltas) != 0 {
		t.Errorf("expected 0 deltas since afterSecond, got %d", len(deltas))
	}
}

// TestIntegration_ErrorRecovery tests error handling and recovery.
func TestIntegration_ErrorRecovery(t *testing.T) {
	ctx := context.Background()
	pb := playbook.New(nil, nil)
	applier := NewDeltaApplier(pb)

	// Create bullet.
	b, _ := bullet.New("Content")
	pb.Add(ctx, b)

	// Apply successful delta.
	delta1 := NewIncrementHelpful(b.ID, DeltaMetadata{Source: "test"})

	result1, err1 := applier.Apply(ctx, *delta1)
	if err1 != nil || !result1.Success {
		t.Fatal("first delta should succeed")
	}

	// Apply delta for non-existent bullet (should fail).
	delta2 := NewIncrementHelpful("non-existent", DeltaMetadata{Source: "test"})

	result2, err2 := applier.Apply(ctx, *delta2)
	if err2 == nil || result2.Success {
		t.Fatal("second delta should fail")
	}

	// Apply another successful delta.
	delta3 := NewIncrementHelpful(b.ID, DeltaMetadata{Source: "test"})

	result3, err3 := applier.Apply(ctx, *delta3)
	if err3 != nil || !result3.Success {
		t.Fatal("third delta should succeed")
	}

	// Verify history only contains successful deltas.
	history := applier.GetHistory()
	if history.Len() != 2 {
		t.Errorf("expected 2 deltas in history (failed delta not recorded), got %d", history.Len())
	}

	// Verify bullet state.
	updated, _ := pb.Get(b.ID)
	if updated.HelpfulCount != 2 {
		t.Errorf("expected helpful count 2, got %d", updated.HelpfulCount)
	}
}
