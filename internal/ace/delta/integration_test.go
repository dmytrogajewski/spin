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
	t.Parallel()

	ctx := context.Background()
	pb := playbook.New(nil, nil)
	applier := NewApplier(pb)

	b1, _ := bullet.New("Initial content for bullet 1")
	b2, _ := bullet.New("Initial content for bullet 2")
	b3, _ := bullet.New("Initial content for bullet 3")

	_ = pb.Add(ctx, b1)
	_ = pb.Add(ctx, b2)
	_ = pb.Add(ctx, b3)

	deltas := buildTestDeltas(b1.ID, b2.ID, b3.ID)
	applyAllDeltas(t, ctx, applier, deltas)

	verifyBullet1(t, pb, b1.ID)
	verifyBullet2(t, pb, b2.ID)
	verifyBullet3(t, pb, b3.ID)
	verifyWorkflowHistory(t, applier, b1.ID, b2.ID, b3.ID)
}

func buildTestDeltas(b1ID, b2ID, b3ID string) []Delta {
	return []Delta{
		*NewContentUpdate(b1ID, "Updated content", Metadata{Source: "reflector", Reason: "Content refinement"}),
		*NewIncrementHelpful(b1ID, Metadata{Source: "curator", Reason: "Duplicate insight"}),
		*NewAddTag(b1ID, "category", "testing", Metadata{Source: "adapter", Reason: "Categorization"}),
		*NewIncrementHelpful(b2ID, Metadata{Source: "curator", Reason: "Useful pattern"}),
		*NewIncrementHarmful(b2ID, Metadata{Source: "feedback", Reason: "Misleading advice"}),
		*NewUpdateEmbedding(b3ID, []float32{0.1, 0.2, 0.3}, Metadata{Source: "embedder", Reason: "Re-embedding"}),
	}
}

func applyAllDeltas(t *testing.T, ctx context.Context, applier *Applier, deltas []Delta) {
	t.Helper()

	for _, delta := range deltas {
		result, err := applier.Apply(ctx, delta)
		if err != nil {
			t.Fatalf("failed to apply delta: %v", err)
		}

		if !result.Success {
			t.Errorf("delta application failed: %v", result.Error)
		}
	}
}

func verifyBullet1(t *testing.T, pb *playbook.Playbook, id string) {
	t.Helper()

	b, _ := pb.Get(id)
	expectEqual(t, "bullet 1 content", b.Content, "Updated content")
	expectIntEqual(t, "bullet 1 helpful count", b.HelpfulCount, 1)
	expectEqual(t, "bullet 1 tag category", b.Tags["category"], "testing")
}

func verifyBullet2(t *testing.T, pb *playbook.Playbook, id string) {
	t.Helper()

	b, _ := pb.Get(id)
	expectIntEqual(t, "bullet 2 helpful count", b.HelpfulCount, 1)
	expectIntEqual(t, "bullet 2 harmful count", b.HarmfulCount, 1)
}

func verifyBullet3(t *testing.T, pb *playbook.Playbook, id string) {
	t.Helper()

	b, _ := pb.Get(id)
	expectIntEqual(t, "bullet 3 embedding length", len(b.Embedding), 3)
}

func verifyWorkflowHistory(t *testing.T, applier *Applier, b1ID, b2ID, b3ID string) {
	t.Helper()

	history := applier.GetHistory()
	expectIntEqual(t, "total deltas in history", history.Len(), 6)
	expectIntEqual(t, "deltas for bullet 1", len(history.GetByBullet(b1ID)), 3)
	expectIntEqual(t, "deltas for bullet 2", len(history.GetByBullet(b2ID)), 2)
	expectIntEqual(t, "deltas for bullet 3", len(history.GetByBullet(b3ID)), 1)
	expectIntEqual(t, "recent deltas", len(history.GetRecent(3)), 3)

	stats := history.Stats()
	expectIntEqual(t, "stats total deltas", stats.TotalDeltas, 6)
	expectIntEqual(t, "stats unique bullets", stats.UniqueBullets, 3)
	expectIntEqual(t, "stats content updates", stats.DeltasByOperation[OpUpdateContent], 1)
	expectIntEqual(t, "stats increment helpful", stats.DeltasByOperation[OpIncrementHelpful], 2)
}

func expectEqual(t *testing.T, name, got, want string) {
	t.Helper()

	if got != want {
		t.Errorf("%s: expected '%s', got '%s'", name, want, got)
	}
}

func expectIntEqual(t *testing.T, name string, got, want int) {
	t.Helper()

	if got != want {
		t.Errorf("%s: expected %d, got %d", name, want, got)
	}
}

// TestIntegration_BatchWithHistory tests batch processing with history queries.
func TestIntegration_BatchWithHistory(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pb := playbook.New(nil, nil)
	applier := NewApplier(pb)

	// Create 10 bullets.
	bulletIDs := make([]string, 10)

	for i := range 10 {
		b, _ := bullet.New("Content")
		_ = pb.Add(ctx, b)
		bulletIDs[i] = b.ID
	}

	// Create batch of deltas.
	deltas := make([]Delta, 20)
	for i := range 10 {
		deltas[i*2] = *NewIncrementHelpful(bulletIDs[i], Metadata{Source: "test"})
		deltas[i*2+1] = *NewAddTag(bulletIDs[i], "batch", "true", Metadata{Source: "test"})
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
	t.Parallel()

	ctx := context.Background()
	pb := playbook.New(nil, nil)
	applier := NewApplier(pb)

	// Create bullet.
	b, _ := bullet.New("Content")
	_ = pb.Add(ctx, b)

	// Apply delta and record timestamp.
	beforeFirst := time.Now()

	time.Sleep(10 * time.Millisecond)

	delta1 := NewIncrementHelpful(b.ID, Metadata{Source: "test"})
	_, _ = applier.Apply(ctx, *delta1)

	afterFirst := time.Now()

	time.Sleep(10 * time.Millisecond)

	delta2 := NewIncrementHelpful(b.ID, Metadata{Source: "test"})
	_, _ = applier.Apply(ctx, *delta2)

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
	t.Parallel()

	ctx := context.Background()
	pb := playbook.New(nil, nil)
	applier := NewApplier(pb)

	// Create bullet.
	b, _ := bullet.New("Content")
	_ = pb.Add(ctx, b)

	// Apply successful delta.
	delta1 := NewIncrementHelpful(b.ID, Metadata{Source: "test"})

	result1, err1 := applier.Apply(ctx, *delta1)
	if err1 != nil || !result1.Success {
		t.Fatal("first delta should succeed")
	}

	// Apply delta for non-existent bullet (should fail).
	delta2 := NewIncrementHelpful("non-existent", Metadata{Source: "test"})

	result2, err2 := applier.Apply(ctx, *delta2)
	if err2 == nil || result2.Success {
		t.Fatal("second delta should fail")
	}

	// Apply another successful delta.
	delta3 := NewIncrementHelpful(b.ID, Metadata{Source: "test"})

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
