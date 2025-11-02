package delta

import (
	"context"
	"testing"

	"github.com/dmytrogajewski/spin/internal/ace/bullet"
	"github.com/dmytrogajewski/spin/internal/ace/playbook"
)

func TestDeltaApplier_ApplyContentUpdate(t *testing.T) {
	ctx := context.Background()
	pb := playbook.New(nil, nil)
	applier := NewDeltaApplier(pb)

	// Add bullet
	b, _ := bullet.New("Original content")
	pb.Add(ctx, b)

	// Apply content update delta
	delta := NewContentUpdate(b.ID, "Updated content", DeltaMetadata{Source: "test"})
	result, err := applier.Apply(ctx, *delta)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Errorf("expected success, got failure: %v", result.Error)
	}

	if result.OldValue != "Original content" {
		t.Errorf("expected old value 'Original content', got '%v'", result.OldValue)
	}

	if result.NewValue != "Updated content" {
		t.Errorf("expected new value 'Updated content', got '%v'", result.NewValue)
	}

	// Verify bullet was updated
	updated, _ := pb.Get(b.ID)
	if updated.Content != "Updated content" {
		t.Errorf("expected bullet content 'Updated content', got '%s'", updated.Content)
	}

	// Verify delta was recorded in history
	if applier.GetHistory().Len() != 1 {
		t.Errorf("expected 1 delta in history, got %d", applier.GetHistory().Len())
	}
}

func TestDeltaApplier_ApplyIncrementHelpful(t *testing.T) {
	ctx := context.Background()
	pb := playbook.New(nil, nil)
	applier := NewDeltaApplier(pb)

	// Add bullet
	b, _ := bullet.New("Test content")
	pb.Add(ctx, b)

	// Apply increment helpful delta
	delta := NewIncrementHelpful(b.ID, DeltaMetadata{Source: "curator"})
	result, err := applier.Apply(ctx, *delta)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Errorf("expected success, got failure: %v", result.Error)
	}

	if result.OldValue != 0 {
		t.Errorf("expected old value 0, got %v", result.OldValue)
	}

	if result.NewValue != 1 {
		t.Errorf("expected new value 1, got %v", result.NewValue)
	}

	// Verify bullet was updated
	updated, _ := pb.Get(b.ID)
	if updated.HelpfulCount != 1 {
		t.Errorf("expected helpful count 1, got %d", updated.HelpfulCount)
	}
}

func TestDeltaApplier_ApplyIncrementHarmful(t *testing.T) {
	ctx := context.Background()
	pb := playbook.New(nil, nil)
	applier := NewDeltaApplier(pb)

	// Add bullet
	b, _ := bullet.New("Test content")
	pb.Add(ctx, b)

	// Apply increment harmful delta
	delta := NewIncrementHarmful(b.ID, DeltaMetadata{Source: "feedback"})
	result, err := applier.Apply(ctx, *delta)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Errorf("expected success, got failure: %v", result.Error)
	}

	// Verify bullet was updated
	updated, _ := pb.Get(b.ID)
	if updated.HarmfulCount != 1 {
		t.Errorf("expected harmful count 1, got %d", updated.HarmfulCount)
	}
}

func TestDeltaApplier_ApplyAddTag(t *testing.T) {
	ctx := context.Background()
	pb := playbook.New(nil, nil)
	applier := NewDeltaApplier(pb)

	// Add bullet
	b, _ := bullet.New("Test content")
	pb.Add(ctx, b)

	// Apply add tag delta
	delta := NewAddTag(b.ID, "category", "testing", DeltaMetadata{Source: "adapter"})
	result, err := applier.Apply(ctx, *delta)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Errorf("expected success, got failure: %v", result.Error)
	}

	// Verify bullet was updated
	updated, _ := pb.Get(b.ID)
	if updated.Tags["category"] != "testing" {
		t.Errorf("expected tag 'category'='testing', got '%s'", updated.Tags["category"])
	}

	// Apply another tag
	delta2 := NewAddTag(b.ID, "priority", "high", DeltaMetadata{Source: "adapter"})
	result2, err := applier.Apply(ctx, *delta2)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result2.Success {
		t.Errorf("expected success, got failure: %v", result2.Error)
	}

	// Verify both tags exist
	updated, _ = pb.Get(b.ID)
	if len(updated.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(updated.Tags))
	}
}

func TestDeltaApplier_ApplyRemoveTag(t *testing.T) {
	ctx := context.Background()
	pb := playbook.New(nil, nil)
	applier := NewDeltaApplier(pb)

	// Add bullet with tags
	b, _ := bullet.New("Test content", bullet.WithTags(map[string]string{
		"category": "testing",
		"priority": "high",
	}))
	pb.Add(ctx, b)

	// Apply remove tag delta
	delta := NewRemoveTag(b.ID, "category", DeltaMetadata{Source: "manual"})
	result, err := applier.Apply(ctx, *delta)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Errorf("expected success, got failure: %v", result.Error)
	}

	// Verify tag was removed
	updated, _ := pb.Get(b.ID)
	if _, exists := updated.Tags["category"]; exists {
		t.Error("expected tag 'category' to be removed")
	}

	if updated.Tags["priority"] != "high" {
		t.Error("expected tag 'priority' to remain")
	}
}

func TestDeltaApplier_ApplyUpdateEmbedding(t *testing.T) {
	ctx := context.Background()
	pb := playbook.New(nil, nil)
	applier := NewDeltaApplier(pb)

	// Add bullet
	b, _ := bullet.New("Test content")
	pb.Add(ctx, b)

	// Apply update embedding delta
	newEmbedding := []float32{0.1, 0.2, 0.3}
	delta := NewUpdateEmbedding(b.ID, newEmbedding, DeltaMetadata{Source: "embedder"})
	result, err := applier.Apply(ctx, *delta)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Errorf("expected success, got failure: %v", result.Error)
	}

	// Verify bullet was updated
	updated, _ := pb.Get(b.ID)
	if len(updated.Embedding) != 3 {
		t.Errorf("expected embedding length 3, got %d", len(updated.Embedding))
	}

	for i, v := range newEmbedding {
		if updated.Embedding[i] != v {
			t.Errorf("embedding[%d]: expected %f, got %f", i, v, updated.Embedding[i])
		}
	}
}

func TestDeltaApplier_BulletNotFound(t *testing.T) {
	ctx := context.Background()
	pb := playbook.New(nil, nil)
	applier := NewDeltaApplier(pb)

	// Apply delta for non-existent bullet
	delta := NewContentUpdate("non-existent", "content", DeltaMetadata{Source: "test"})
	result, err := applier.Apply(ctx, *delta)

	if err == nil {
		t.Fatal("expected error for non-existent bullet")
	}

	if result.Success {
		t.Error("expected failure for non-existent bullet")
	}

	// Verify delta was NOT recorded in history
	if applier.GetHistory().Len() != 0 {
		t.Errorf("expected 0 deltas in history for failed apply, got %d", applier.GetHistory().Len())
	}
}

func TestDeltaApplier_InvalidFieldType(t *testing.T) {
	ctx := context.Background()
	pb := playbook.New(nil, nil)
	applier := NewDeltaApplier(pb)

	// Add bullet
	b, _ := bullet.New("Test content")
	pb.Add(ctx, b)

	// Create delta with invalid field type
	// Create delta with missing content field
	invalidDelta := Delta{
		ID:        "delta-1",
		BulletID:  b.ID,
		Operation: OpUpdateContent,
		Fields:    DeltaFields{}, // Missing Content field
		Metadata:  DeltaMetadata{Source: "test"},
		CreatedAt: b.CreatedAt,
	}

	result, err := applier.Apply(ctx, invalidDelta)

	if err == nil {
		t.Fatal("expected error for invalid field type")
	}

	if result.Success {
		t.Error("expected failure for invalid field type")
	}
}

func TestDeltaApplier_UnknownOperation(t *testing.T) {
	ctx := context.Background()
	pb := playbook.New(nil, nil)
	applier := NewDeltaApplier(pb)

	// Add bullet
	b, _ := bullet.New("Test content")
	pb.Add(ctx, b)

	// Create delta with unknown operation
	unknownDelta := Delta{
		ID:        "delta-1",
		BulletID:  b.ID,
		Operation: DeltaOperation("unknown_op"),
		Fields:    DeltaFields{},
		Metadata:  DeltaMetadata{Source: "test"},
		CreatedAt: b.CreatedAt,
	}

	result, err := applier.Apply(ctx, unknownDelta)

	if err == nil {
		t.Fatal("expected error for unknown operation")
	}

	if result.Success {
		t.Error("expected failure for unknown operation")
	}
}

func TestDeltaApplier_MultipleDeltas(t *testing.T) {
	ctx := context.Background()
	pb := playbook.New(nil, nil)
	applier := NewDeltaApplier(pb)

	// Add bullet
	b, _ := bullet.New("Original content")
	pb.Add(ctx, b)

	// Apply multiple deltas
	deltas := []Delta{
		*NewContentUpdate(b.ID, "Updated content", DeltaMetadata{Source: "test"}),
		*NewIncrementHelpful(b.ID, DeltaMetadata{Source: "test"}),
		*NewIncrementHelpful(b.ID, DeltaMetadata{Source: "test"}),
		*NewAddTag(b.ID, "category", "testing", DeltaMetadata{Source: "test"}),
	}

	for _, delta := range deltas {
		result, err := applier.Apply(ctx, delta)
		if err != nil {
			t.Fatalf("unexpected error applying delta: %v", err)
		}
		if !result.Success {
			t.Errorf("expected success, got failure: %v", result.Error)
		}
	}

	// Verify all changes were applied
	updated, _ := pb.Get(b.ID)
	if updated.Content != "Updated content" {
		t.Errorf("expected content 'Updated content', got '%s'", updated.Content)
	}
	if updated.HelpfulCount != 2 {
		t.Errorf("expected helpful count 2, got %d", updated.HelpfulCount)
	}
	if updated.Tags["category"] != "testing" {
		t.Errorf("expected tag 'category'='testing', got '%s'", updated.Tags["category"])
	}

	// Verify all deltas were recorded
	if applier.GetHistory().Len() != 4 {
		t.Errorf("expected 4 deltas in history, got %d", applier.GetHistory().Len())
	}
}
