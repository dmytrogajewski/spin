package delta

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/ace/bullet"
	"github.com/dmytrogajewski/spin/internal/ace/playbook"
)

const (
	testUpdatedContent = "Updated content"
)


func TestDeltaApplier_ApplyContentUpdate(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pb := playbook.New(nil, nil)
	applier := NewApplier(pb)

	// Add bullet.
	b, err := bullet.New("Original content")
	require.NoError(t, err)

	require.NoError(t, pb.Add(ctx, b))

	// Apply content update delta.
	delta := NewContentUpdate(b.ID, testUpdatedContent, Metadata{Source: "test"})

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

	if result.NewValue != testUpdatedContent {
		t.Errorf("expected new value 'Updated content', got '%v'", result.NewValue)
	}

	// Verify bullet was updated.
	updated, ok := pb.Get(b.ID)
	require.True(t, ok, "bullet not found")

	if updated.Content != testUpdatedContent {
		t.Errorf("expected bullet content 'Updated content', got '%s'", updated.Content)
	}

	// Verify delta was recorded in history.
	if applier.GetHistory().Len() != 1 {
		t.Errorf("expected 1 delta in history, got %d", applier.GetHistory().Len())
	}
}

func TestDeltaApplier_ApplyIncrementHelpful(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pb := playbook.New(nil, nil)
	applier := NewApplier(pb)

	// Add bullet.
	b, err := bullet.New("Test content")
	require.NoError(t, err)

	require.NoError(t, pb.Add(ctx, b))

	// Apply increment helpful delta.
	delta := NewIncrementHelpful(b.ID, Metadata{Source: "curator"})

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

	// Verify bullet was updated.
	updated, ok := pb.Get(b.ID)
	require.True(t, ok, "bullet not found")

	if updated.HelpfulCount != 1 {
		t.Errorf("expected helpful count 1, got %d", updated.HelpfulCount)
	}
}

func TestDeltaApplier_ApplyIncrementHarmful(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pb := playbook.New(nil, nil)
	applier := NewApplier(pb)

	// Add bullet.
	b, err := bullet.New("Test content")
	require.NoError(t, err)

	require.NoError(t, pb.Add(ctx, b))

	// Apply increment harmful delta.
	delta := NewIncrementHarmful(b.ID, Metadata{Source: "feedback"})

	result, err := applier.Apply(ctx, *delta)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Errorf("expected success, got failure: %v", result.Error)
	}

	// Verify bullet was updated.
	updated, ok := pb.Get(b.ID)
	require.True(t, ok, "bullet not found")

	if updated.HarmfulCount != 1 {
		t.Errorf("expected harmful count 1, got %d", updated.HarmfulCount)
	}
}

func TestDeltaApplier_ApplyAddTag(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pb := playbook.New(nil, nil)
	applier := NewApplier(pb)

	// Add bullet.
	b, err := bullet.New("Test content")
	require.NoError(t, err)

	require.NoError(t, pb.Add(ctx, b))

	// Apply add tag delta.
	delta := NewAddTag(b.ID, "category", "testing", Metadata{Source: "adapter"})

	result, err := applier.Apply(ctx, *delta)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Errorf("expected success, got failure: %v", result.Error)
	}

	// Verify bullet was updated.
	updated, ok := pb.Get(b.ID)
	require.True(t, ok, "bullet not found")

	if updated.Tags["category"] != "testing" {
		t.Errorf("expected tag 'category'='testing', got '%s'", updated.Tags["category"])
	}

	// Apply another tag.
	delta2 := NewAddTag(b.ID, "priority", "high", Metadata{Source: "adapter"})

	result2, err := applier.Apply(ctx, *delta2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result2.Success {
		t.Errorf("expected success, got failure: %v", result2.Error)
	}

	// Verify both tags exist.
	updated, ok = pb.Get(b.ID)
	require.True(t, ok, "bullet not found")

	if len(updated.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(updated.Tags))
	}
}

func TestDeltaApplier_ApplyRemoveTag(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pb := playbook.New(nil, nil)
	applier := NewApplier(pb)

	// Add bullet with tags.
	b, err := bullet.New("Test content", bullet.WithTags(map[string]string{
		"category": "testing",
		"priority": "high",
	}))
	require.NoError(t, err)

	require.NoError(t, pb.Add(ctx, b))

	// Apply remove tag delta.
	delta := NewRemoveTag(b.ID, "category", Metadata{Source: "manual"})

	result, err := applier.Apply(ctx, *delta)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Errorf("expected success, got failure: %v", result.Error)
	}

	// Verify tag was removed.
	updated, ok := pb.Get(b.ID)
	require.True(t, ok, "bullet not found")

	if _, exists := updated.Tags["category"]; exists {
		t.Error("expected tag 'category' to be removed")
	}

	if updated.Tags["priority"] != "high" {
		t.Error("expected tag 'priority' to remain")
	}
}

func TestDeltaApplier_ApplyUpdateEmbedding(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pb := playbook.New(nil, nil)
	applier := NewApplier(pb)

	// Add bullet.
	b, err := bullet.New("Test content")
	require.NoError(t, err)

	require.NoError(t, pb.Add(ctx, b))

	// Apply update embedding delta.
	newEmbedding := []float32{0.1, 0.2, 0.3}
	delta := NewUpdateEmbedding(b.ID, newEmbedding, Metadata{Source: "embedder"})

	result, err := applier.Apply(ctx, *delta)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Errorf("expected success, got failure: %v", result.Error)
	}

	// Verify bullet was updated.
	updated, ok := pb.Get(b.ID)
	require.True(t, ok, "bullet not found")

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
	t.Parallel()

	ctx := context.Background()
	pb := playbook.New(nil, nil)
	applier := NewApplier(pb)

	// Apply delta for non-existent bullet.
	delta := NewContentUpdate("non-existent", "content", Metadata{Source: "test"})

	result, err := applier.Apply(ctx, *delta)
	if err == nil {
		t.Fatal("expected error for non-existent bullet")
	}

	if result.Success {
		t.Error("expected failure for non-existent bullet")
	}

	// Verify delta was NOT recorded in history.
	if applier.GetHistory().Len() != 0 {
		t.Errorf("expected 0 deltas in history for failed apply, got %d", applier.GetHistory().Len())
	}
}

func TestDeltaApplier_InvalidFieldType(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pb := playbook.New(nil, nil)
	applier := NewApplier(pb)

	// Add bullet.
	b, err := bullet.New("Test content")
	require.NoError(t, err)

	require.NoError(t, pb.Add(ctx, b))

	// Create delta with invalid field type
	// Create delta with missing content field.
	invalidDelta := Delta{
		ID:        "delta-1",
		BulletID:  b.ID,
		Operation: OpUpdateContent,
		Fields:    Fields{}, // Missing Content field.
		Metadata:  Metadata{Source: "test"},
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
	t.Parallel()

	ctx := context.Background()
	pb := playbook.New(nil, nil)
	applier := NewApplier(pb)

	// Add bullet.
	b, err := bullet.New("Test content")
	require.NoError(t, err)

	require.NoError(t, pb.Add(ctx, b))

	// Create delta with unknown operation.
	unknownDelta := Delta{
		ID:        "delta-1",
		BulletID:  b.ID,
		Operation: Operation("unknown_op"),
		Fields:    Fields{},
		Metadata:  Metadata{Source: "test"},
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
	t.Parallel()

	ctx := context.Background()
	pb := playbook.New(nil, nil)
	applier := NewApplier(pb)

	// Add bullet.
	b, err := bullet.New("Original content")
	require.NoError(t, err)

	require.NoError(t, pb.Add(ctx, b))

	// Apply multiple deltas.
	deltas := []Delta{
		*NewContentUpdate(b.ID, testUpdatedContent, Metadata{Source: "test"}),
		*NewIncrementHelpful(b.ID, Metadata{Source: "test"}),
		*NewIncrementHelpful(b.ID, Metadata{Source: "test"}),
		*NewAddTag(b.ID, "category", "testing", Metadata{Source: "test"}),
	}

	for _, delta := range deltas {
		var result *ApplyResult
		result, err = applier.Apply(ctx, delta)
		if err != nil {
			t.Fatalf("unexpected error applying delta: %v", err)
		}

		if !result.Success {
			t.Errorf("expected success, got failure: %v", result.Error)
		}
	}

	// Verify all changes were applied.
	updated, ok := pb.Get(b.ID)
	require.True(t, ok, "bullet not found")

	if updated.Content != testUpdatedContent {
		t.Errorf("expected content 'Updated content', got '%s'", updated.Content)
	}

	if updated.HelpfulCount != 2 {
		t.Errorf("expected helpful count 2, got %d", updated.HelpfulCount)
	}

	if updated.Tags["category"] != "testing" {
		t.Errorf("expected tag 'category'='testing', got '%s'", updated.Tags["category"])
	}

	// Verify all deltas were recorded.
	if applier.GetHistory().Len() != 4 {
		t.Errorf("expected 4 deltas in history, got %d", applier.GetHistory().Len())
	}
}
