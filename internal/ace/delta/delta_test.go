package delta

import (
	"testing"
)

func TestDelta_NewContentUpdate(t *testing.T) {
	bulletID := "bullet-123"
	newContent := "Updated bullet content"

	delta := NewContentUpdate(bulletID, newContent, DeltaMetadata{
		Source: "test",
		Reason: "testing content update",
	})

	if delta == nil {
		t.Fatal("expected delta, got nil")
	}

	if delta.BulletID != bulletID {
		t.Errorf("expected BulletID %s, got %s", bulletID, delta.BulletID)
	}

	if delta.Operation != OpUpdateContent {
		t.Errorf("expected Operation %s, got %s", OpUpdateContent, delta.Operation)
	}

	if delta.Fields.Content == nil {
		t.Fatal("expected content field to be set")
	}

	if *delta.Fields.Content != newContent {
		t.Errorf("expected content %s, got %s", newContent, *delta.Fields.Content)
	}

	if delta.ID == "" {
		t.Error("expected non-empty ID")
	}

	if delta.CreatedAt.IsZero() {
		t.Error("expected non-zero CreatedAt")
	}

	if delta.Metadata.Source != "test" {
		t.Errorf("expected Metadata.Source 'test', got '%s'", delta.Metadata.Source)
	}
}

func TestDelta_NewIncrementHelpful(t *testing.T) {
	bulletID := "bullet-456"

	delta := NewIncrementHelpful(bulletID, DeltaMetadata{
		Source: "curator",
		Reason: "duplicate insight detected",
	})

	if delta == nil {
		t.Fatal("expected delta, got nil")
	}

	if delta.BulletID != bulletID {
		t.Errorf("expected BulletID %s, got %s", bulletID, delta.BulletID)
	}

	if delta.Operation != OpIncrementHelpful {
		t.Errorf("expected Operation %s, got %s", OpIncrementHelpful, delta.Operation)
	}

	// For increment operations, all fields should be nil.
	if delta.Fields.Content != nil || delta.Fields.TagKey != nil || delta.Fields.TagValue != nil || delta.Fields.Embedding != nil {
		t.Error("expected all fields to be nil for increment operation")
	}

	if delta.Metadata.Source != "curator" {
		t.Errorf("expected Metadata.Source 'curator', got '%s'", delta.Metadata.Source)
	}
}

func TestDelta_NewIncrementHarmful(t *testing.T) {
	bulletID := "bullet-789"

	delta := NewIncrementHarmful(bulletID, DeltaMetadata{
		Source: "feedback",
		Reason: "bullet was harmful",
	})

	if delta == nil {
		t.Fatal("expected delta, got nil")
	}

	if delta.BulletID != bulletID {
		t.Errorf("expected BulletID %s, got %s", bulletID, delta.BulletID)
	}

	if delta.Operation != OpIncrementHarmful {
		t.Errorf("expected Operation %s, got %s", OpIncrementHarmful, delta.Operation)
	}

	// For increment operations, all fields should be nil.
	if delta.Fields.Content != nil || delta.Fields.TagKey != nil || delta.Fields.TagValue != nil || delta.Fields.Embedding != nil {
		t.Error("expected all fields to be nil for increment operation")
	}
}

func TestDelta_NewAddTag(t *testing.T) {
	bulletID := "bullet-abc"
	key := "category"
	value := "error_handling"

	delta := NewAddTag(bulletID, key, value, DeltaMetadata{
		Source: "adapter",
		Reason: "categorizing bullet",
	})

	if delta == nil {
		t.Fatal("expected delta, got nil")
	}

	if delta.BulletID != bulletID {
		t.Errorf("expected BulletID %s, got %s", bulletID, delta.BulletID)
	}

	if delta.Operation != OpAddTag {
		t.Errorf("expected Operation %s, got %s", OpAddTag, delta.Operation)
	}

	if delta.Fields.TagKey == nil {
		t.Fatal("expected tag_key field to be set")
	}

	if *delta.Fields.TagKey != key {
		t.Errorf("expected key %s, got %s", key, *delta.Fields.TagKey)
	}

	if delta.Fields.TagValue == nil {
		t.Fatal("expected tag_value field to be set")
	}

	if *delta.Fields.TagValue != value {
		t.Errorf("expected value %s, got %s", value, *delta.Fields.TagValue)
	}
}

func TestDelta_NewRemoveTag(t *testing.T) {
	bulletID := "bullet-def"
	key := "obsolete"

	delta := NewRemoveTag(bulletID, key, DeltaMetadata{
		Source: "manual",
		Reason: "removing obsolete tag",
	})

	if delta == nil {
		t.Fatal("expected delta, got nil")
	}

	if delta.BulletID != bulletID {
		t.Errorf("expected BulletID %s, got %s", bulletID, delta.BulletID)
	}

	if delta.Operation != OpRemoveTag {
		t.Errorf("expected Operation %s, got %s", OpRemoveTag, delta.Operation)
	}

	if delta.Fields.TagKey == nil {
		t.Fatal("expected tag_key field to be set")
	}

	if *delta.Fields.TagKey != key {
		t.Errorf("expected key %s, got %s", key, *delta.Fields.TagKey)
	}
}

func TestDelta_NewUpdateEmbedding(t *testing.T) {
	bulletID := "bullet-ghi"
	embedding := []float32{0.1, 0.2, 0.3}

	delta := NewUpdateEmbedding(bulletID, embedding, DeltaMetadata{
		Source: "embedder",
		Reason: "re-embedding after content update",
	})

	if delta == nil {
		t.Fatal("expected delta, got nil")
	}

	if delta.BulletID != bulletID {
		t.Errorf("expected BulletID %s, got %s", bulletID, delta.BulletID)
	}

	if delta.Operation != OpUpdateEmbedding {
		t.Errorf("expected Operation %s, got %s", OpUpdateEmbedding, delta.Operation)
	}

	if delta.Fields.Embedding == nil {
		t.Fatal("expected embedding field to be set")
	}

	if len(delta.Fields.Embedding) != len(embedding) {
		t.Errorf("expected embedding length %d, got %d", len(embedding), len(delta.Fields.Embedding))
	}

	for i := range embedding {
		if delta.Fields.Embedding[i] != embedding[i] {
			t.Errorf("embedding[%d]: expected %f, got %f", i, embedding[i], delta.Fields.Embedding[i])
		}
	}
}
