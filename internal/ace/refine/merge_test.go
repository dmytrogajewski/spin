package refine

import (
	"context"
	"testing"

	"github.com/dmytrogajewski/spin/internal/ace/bullet"
	"github.com/dmytrogajewski/spin/internal/ace/embedding"
)

func TestNewMergeEngine(t *testing.T) {
	t.Parallel()

	embedder := embedding.NewMockEmbedder(384)

	// Valid threshold.
	engine := NewMergeEngine(embedder, 0.85)
	if engine.similarity != 0.85 {
		t.Errorf("expected similarity 0.85, got %f", engine.similarity)
	}

	// Invalid threshold (too high).
	engine = NewMergeEngine(embedder, 1.5)
	if engine.similarity != 0.90 {
		t.Errorf("expected default similarity 0.90 for invalid threshold, got %f", engine.similarity)
	}

	// Invalid threshold (negative).
	engine = NewMergeEngine(embedder, -0.1)
	if engine.similarity != 0.90 {
		t.Errorf("expected default similarity 0.90 for invalid threshold, got %f", engine.similarity)
	}
}

func TestMergeEngine_MergeBullets(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	embedder := embedding.NewMockEmbedder(384)
	engine := NewMergeEngine(embedder, 0.90)

	b1, _ := bullet.New("Content 1")
	b1.IncrementHelpful()
	b1.IncrementHelpful()
	b1.Tags = map[string]string{"source": "test1"}

	b2, _ := bullet.New("Content 2")
	b2.IncrementHelpful()
	b2.Tags = map[string]string{"source": "test2", "extra": "value"}

	// Merge b2 into b1 (b1 has higher utility).
	result, err := engine.MergeBullets(ctx, b2, b1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.KeptID != b1.ID {
		t.Errorf("expected kept ID %s, got %s", b1.ID, result.KeptID)
	}

	if result.RemovedID != b2.ID {
		t.Errorf("expected removed ID %s, got %s", b2.ID, result.RemovedID)
	}

	// Original bullets should be unchanged.
	if b1.HelpfulCount != 2 {
		t.Errorf("expected original b1 helpful count 2, got %d", b1.HelpfulCount)
	}
}

func TestMergeEngine_MergeBullets_UtilityTransfer(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	embedder := embedding.NewMockEmbedder(384)
	engine := NewMergeEngine(embedder, 0.90)

	b1, _ := bullet.New("High utility")
	b1.IncrementHelpful()
	b1.IncrementHelpful()
	b1.IncrementHelpful()

	b2, _ := bullet.New("Low utility")
	b2.IncrementHelpful()

	result, err := engine.MergeBullets(ctx, b2, b1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// b1 should be kept (higher utility: 3 vs 1).
	if result.KeptID != b1.ID {
		t.Errorf("expected b1 to be kept (higher utility)")
	}
}

func TestMergeEngine_MergeBullets_NilBullets(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	embedder := embedding.NewMockEmbedder(384)
	engine := NewMergeEngine(embedder, 0.90)

	b, _ := bullet.New("Test")

	// Nil source.
	_, err := engine.MergeBullets(ctx, nil, b)
	if err == nil {
		t.Error("expected error for nil source bullet")
	}

	// Nil target.
	_, err = engine.MergeBullets(ctx, b, nil)
	if err == nil {
		t.Error("expected error for nil target bullet")
	}

	// Both nil.
	_, err = engine.MergeBullets(ctx, nil, nil)
	if err == nil {
		t.Error("expected error for both nil bullets")
	}
}

func TestMergeEngine_FindMergeCandidates_WithEmbeddings(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	embedder := embedding.NewMockEmbedder(384)
	engine := NewMergeEngine(embedder, 0.90)

	// Create bullets with embeddings.
	b1, _ := bullet.New("Similar content about Go testing")
	emb1, _ := embedder.Embed(ctx, b1.Content)
	b1.Embedding = emb1

	b2, _ := bullet.New("Similar content about Go testing") // Exact same.
	emb2, _ := embedder.Embed(ctx, b2.Content)
	b2.Embedding = emb2

	b3, _ := bullet.New("Completely different topic")
	emb3, _ := embedder.Embed(ctx, b3.Content)
	b3.Embedding = emb3

	bullets := []*bullet.Bullet{b1, b2, b3}

	pairs, err := engine.FindMergeCandidates(ctx, bullets)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should find b1 and b2 as similar (same content = high similarity).
	if len(pairs) == 0 {
		t.Error("expected at least one merge pair for identical content")
	}
}

func TestMergeEngine_FindMergeCandidates_WithoutEmbeddings(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	engine := NewMergeEngine(nil, 0.90) // No embedder.

	b1, _ := bullet.New("Test content")
	b2, _ := bullet.New("Test content") // Exact match.
	b3, _ := bullet.New("Different")

	bullets := []*bullet.Bullet{b1, b2, b3}

	pairs, err := engine.FindMergeCandidates(ctx, bullets)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should find b1 and b2 as exact match (similarity = 1.0).
	foundPair := false

	for _, pair := range pairs {
		if (pair.SourceID == b1.ID && pair.TargetID == b2.ID) ||
			(pair.SourceID == b2.ID && pair.TargetID == b1.ID) {
			foundPair = true

			if pair.Similarity != 1.0 {
				t.Errorf("expected similarity 1.0 for exact match, got %f", pair.Similarity)
			}
		}
	}

	if !foundPair {
		t.Error("expected to find merge pair for exact match content")
	}
}

func TestMergeEngine_FindMergeCandidates_Empty(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	embedder := embedding.NewMockEmbedder(384)
	engine := NewMergeEngine(embedder, 0.90)

	pairs, err := engine.FindMergeCandidates(ctx, []*bullet.Bullet{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(pairs) != 0 {
		t.Errorf("expected 0 pairs for empty bullets, got %d", len(pairs))
	}
}

func TestMergeEngine_FindMergeCandidates_Single(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	embedder := embedding.NewMockEmbedder(384)
	engine := NewMergeEngine(embedder, 0.90)

	b, _ := bullet.New("Single bullet")
	bullets := []*bullet.Bullet{b}

	pairs, err := engine.FindMergeCandidates(ctx, bullets)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(pairs) != 0 {
		t.Errorf("expected 0 pairs for single bullet, got %d", len(pairs))
	}
}

func TestMergeEngine_CosineSimilarity(t *testing.T) {
	t.Parallel()

	embedder := embedding.NewMockEmbedder(384)
	engine := NewMergeEngine(embedder, 0.90)

	// Identical vectors.
	v1 := []float32{1.0, 2.0, 3.0}
	v2 := []float32{1.0, 2.0, 3.0}

	sim := engine.cosineSimilarity(v1, v2)
	if sim < 0.999 { // Allow for floating point error.
		t.Errorf("expected similarity ~1.0 for identical vectors, got %f", sim)
	}

	// Orthogonal vectors.
	v3 := []float32{1.0, 0.0, 0.0}
	v4 := []float32{0.0, 1.0, 0.0}

	sim = engine.cosineSimilarity(v3, v4)
	if sim > 0.001 { // Should be very close to 0.
		t.Errorf("expected similarity ~0.0 for orthogonal vectors, got %f", sim)
	}

	// Different length vectors.
	v5 := []float32{1.0, 2.0}
	v6 := []float32{1.0, 2.0, 3.0}

	sim = engine.cosineSimilarity(v5, v6)
	if sim != 0.0 {
		t.Errorf("expected similarity 0.0 for different length vectors, got %f", sim)
	}

	// Zero vectors.
	v7 := []float32{0.0, 0.0, 0.0}
	v8 := []float32{0.0, 0.0, 0.0}

	sim = engine.cosineSimilarity(v7, v8)
	if sim != 0.0 {
		t.Errorf("expected similarity 0.0 for zero vectors, got %f", sim)
	}
}

func TestMergeEngine_SimpleSimilarity(t *testing.T) {
	t.Parallel()

	embedder := embedding.NewMockEmbedder(384)
	engine := NewMergeEngine(embedder, 0.90)

	// Exact match.
	sim := engine.simpleSimilarity("test", "test")
	if sim != 1.0 {
		t.Errorf("expected similarity 1.0 for exact match, got %f", sim)
	}

	// Different lengths.
	sim = engine.simpleSimilarity("short", "longer string")
	if sim <= 0.0 || sim >= 1.0 {
		t.Errorf("expected similarity between 0 and 1, got %f", sim)
	}

	// Empty strings.
	sim = engine.simpleSimilarity("", "")
	if sim != 1.0 {
		t.Errorf("expected similarity 1.0 for both empty, got %f", sim)
	}

	// One empty.
	sim = engine.simpleSimilarity("", "non-empty")
	if sim != 0.0 {
		t.Errorf("expected similarity 0.0 for one empty, got %f", sim)
	}
}

func TestMergeEngine_ChooseMergeDirection(t *testing.T) {
	t.Parallel()

	embedder := embedding.NewMockEmbedder(384)
	engine := NewMergeEngine(embedder, 0.90)

	b1, _ := bullet.New("High utility")
	b1.IncrementHelpful()
	b1.IncrementHelpful()
	b1.IncrementHelpful()

	b2, _ := bullet.New("Low utility")
	b2.IncrementHelpful()

	sourceID, targetID := engine.chooseMergeDirection(b1, b2)

	// b1 has higher utility, so it should be target (kept).
	if targetID != b1.ID {
		t.Errorf("expected b1 (high utility) to be target, got %s", targetID)
	}

	if sourceID != b2.ID {
		t.Errorf("expected b2 (low utility) to be source, got %s", sourceID)
	}

	// Reverse order.
	_, targetID = engine.chooseMergeDirection(b2, b1)
	if targetID != b1.ID {
		t.Errorf("expected b1 (high utility) to be target regardless of order, got %s", targetID)
	}
}
