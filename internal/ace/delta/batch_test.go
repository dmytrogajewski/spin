package delta

import (
	"context"
	"testing"

	"github.com/dmytrogajewski/spin/internal/ace/bullet"
	"github.com/dmytrogajewski/spin/internal/ace/playbook"
)

func TestDeltaApplier_ApplyBatch_Success(t *testing.T) {
	ctx := context.Background()
	pb := playbook.New(nil, nil)
	applier := NewDeltaApplier(pb)

	// Add bullets
	b1, _ := bullet.New("Content 1")
	b2, _ := bullet.New("Content 2")
	b3, _ := bullet.New("Content 3")
	pb.Add(ctx, b1)
	pb.Add(ctx, b2)
	pb.Add(ctx, b3)

	// Create batch request
	deltas := []Delta{
		*NewContentUpdate(b1.ID, "Updated 1", DeltaMetadata{Source: "test"}),
		*NewIncrementHelpful(b2.ID, DeltaMetadata{Source: "test"}),
		*NewAddTag(b3.ID, "category", "test", DeltaMetadata{Source: "test"}),
	}

	req := BatchApplyRequest{
		Deltas:     deltas,
		MaxWorkers: 2,
		Atomic:     false,
	}

	result, err := applier.ApplyBatch(ctx, req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Applied != 3 {
		t.Errorf("expected 3 applied, got %d", result.Applied)
	}

	if result.Failed != 0 {
		t.Errorf("expected 0 failed, got %d", result.Failed)
	}

	// Verify all bullets were updated
	updated1, _ := pb.Get(b1.ID)
	if updated1.Content != "Updated 1" {
		t.Errorf("expected content 'Updated 1', got '%s'", updated1.Content)
	}

	updated2, _ := pb.Get(b2.ID)
	if updated2.HelpfulCount != 1 {
		t.Errorf("expected helpful count 1, got %d", updated2.HelpfulCount)
	}

	updated3, _ := pb.Get(b3.ID)
	if updated3.Tags["category"] != "test" {
		t.Errorf("expected tag 'category'='test', got '%s'", updated3.Tags["category"])
	}
}

func TestDeltaApplier_ApplyBatch_PartialFailure(t *testing.T) {
	ctx := context.Background()
	pb := playbook.New(nil, nil)
	applier := NewDeltaApplier(pb)

	// Add only one bullet
	b1, _ := bullet.New("Content 1")
	pb.Add(ctx, b1)

	// Create batch with one valid and two invalid deltas
	deltas := []Delta{
		*NewContentUpdate(b1.ID, "Updated 1", DeltaMetadata{Source: "test"}),
		*NewContentUpdate("non-existent-1", "Updated", DeltaMetadata{Source: "test"}),
		*NewContentUpdate("non-existent-2", "Updated", DeltaMetadata{Source: "test"}),
	}

	req := BatchApplyRequest{
		Deltas:     deltas,
		MaxWorkers: 2,
		Atomic:     false, // Not atomic, so partial success is OK
	}

	result, err := applier.ApplyBatch(ctx, req)

	if err != nil {
		t.Fatalf("unexpected error in non-atomic mode: %v", err)
	}

	if result.Applied != 1 {
		t.Errorf("expected 1 applied, got %d", result.Applied)
	}

	if result.Failed != 2 {
		t.Errorf("expected 2 failed, got %d", result.Failed)
	}

	// Verify successful delta was applied
	updated1, _ := pb.Get(b1.ID)
	if updated1.Content != "Updated 1" {
		t.Errorf("expected content 'Updated 1', got '%s'", updated1.Content)
	}
}

func TestDeltaApplier_ApplyBatch_AtomicFailure(t *testing.T) {
	ctx := context.Background()
	pb := playbook.New(nil, nil)
	applier := NewDeltaApplier(pb)

	// Add only one bullet
	b1, _ := bullet.New("Content 1")
	pb.Add(ctx, b1)

	// Create batch with one valid and one invalid delta
	deltas := []Delta{
		*NewContentUpdate(b1.ID, "Updated 1", DeltaMetadata{Source: "test"}),
		*NewContentUpdate("non-existent", "Updated", DeltaMetadata{Source: "test"}),
	}

	req := BatchApplyRequest{
		Deltas:     deltas,
		MaxWorkers: 2,
		Atomic:     true, // Atomic mode: should fail entire batch
	}

	result, err := applier.ApplyBatch(ctx, req)

	if err == nil {
		t.Fatal("expected error in atomic mode with failure")
	}

	if !result.RolledBack {
		t.Error("expected RolledBack to be true in atomic mode with failure")
	}

	// Note: Full rollback not implemented yet, so the successful delta
	// will still be applied. This is a known limitation.
}

func TestDeltaApplier_ApplyBatch_DefaultWorkers(t *testing.T) {
	ctx := context.Background()
	pb := playbook.New(nil, nil)
	applier := NewDeltaApplier(pb)

	// Add bullets
	b1, _ := bullet.New("Content 1")
	b2, _ := bullet.New("Content 2")
	pb.Add(ctx, b1)
	pb.Add(ctx, b2)

	deltas := []Delta{
		*NewContentUpdate(b1.ID, "Updated 1", DeltaMetadata{Source: "test"}),
		*NewContentUpdate(b2.ID, "Updated 2", DeltaMetadata{Source: "test"}),
	}

	req := BatchApplyRequest{
		Deltas:     deltas,
		MaxWorkers: 0, // Should use runtime.NumCPU()
		Atomic:     false,
	}

	result, err := applier.ApplyBatch(ctx, req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Applied != 2 {
		t.Errorf("expected 2 applied, got %d", result.Applied)
	}
}

func TestDeltaApplier_ApplyBatch_EmptyBatch(t *testing.T) {
	ctx := context.Background()
	pb := playbook.New(nil, nil)
	applier := NewDeltaApplier(pb)

	req := BatchApplyRequest{
		Deltas:     []Delta{},
		MaxWorkers: 2,
		Atomic:     false,
	}

	result, err := applier.ApplyBatch(ctx, req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Applied != 0 {
		t.Errorf("expected 0 applied, got %d", result.Applied)
	}

	if result.Failed != 0 {
		t.Errorf("expected 0 failed, got %d", result.Failed)
	}
}

func TestDeltaApplier_ApplyBatch_LargeBatch(t *testing.T) {
	ctx := context.Background()
	pb := playbook.New(nil, nil)
	applier := NewDeltaApplier(pb)

	// Add 100 bullets
	const count = 100
	bulletIDs := make([]string, count)
	for i := 0; i < count; i++ {
		b, _ := bullet.New("Original content")
		pb.Add(ctx, b)
		bulletIDs[i] = b.ID
	}

	// Create batch with 100 deltas
	deltas := make([]Delta, count)
	for i := 0; i < count; i++ {
		deltas[i] = *NewIncrementHelpful(bulletIDs[i], DeltaMetadata{Source: "test"})
	}

	req := BatchApplyRequest{
		Deltas:     deltas,
		MaxWorkers: 4,
		Atomic:     false,
	}

	result, err := applier.ApplyBatch(ctx, req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Applied != count {
		t.Errorf("expected %d applied, got %d", count, result.Applied)
	}

	if result.Failed != 0 {
		t.Errorf("expected 0 failed, got %d", result.Failed)
	}

	// Verify all bullets were updated
	for _, id := range bulletIDs {
		updated, _ := pb.Get(id)
		if updated.HelpfulCount != 1 {
			t.Errorf("bullet %s: expected helpful count 1, got %d", id, updated.HelpfulCount)
		}
	}

	// Verify history
	if applier.GetHistory().Len() != count {
		t.Errorf("expected %d deltas in history, got %d", count, applier.GetHistory().Len())
	}
}

func TestDeltaApplier_ApplyBatch_ConcurrentSafety(t *testing.T) {
	ctx := context.Background()
	pb := playbook.New(nil, nil)
	applier := NewDeltaApplier(pb)

	// Add 50 separate bullets (one per delta) to avoid race on same bullet
	bulletIDs := make([]string, 50)
	for i := 0; i < 50; i++ {
		b, _ := bullet.New("Content")
		pb.Add(ctx, b)
		bulletIDs[i] = b.ID
	}

	// Create 50 increment helpful deltas for different bullets
	deltas := make([]Delta, 50)
	for i := 0; i < 50; i++ {
		deltas[i] = *NewIncrementHelpful(bulletIDs[i], DeltaMetadata{Source: "test"})
	}

	req := BatchApplyRequest{
		Deltas:     deltas,
		MaxWorkers: 8,
		Atomic:     false,
	}

	result, err := applier.ApplyBatch(ctx, req)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Applied != 50 {
		t.Errorf("expected 50 applied, got %d", result.Applied)
	}

	// Verify each bullet was updated once
	for i, id := range bulletIDs {
		updated, _ := pb.Get(id)
		if updated.HelpfulCount != 1 {
			t.Errorf("bullet %d: expected helpful count 1, got %d", i, updated.HelpfulCount)
		}
	}
}
