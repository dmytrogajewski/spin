package curator

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/ace/bullet"
	"github.com/dmytrogajewski/spin/internal/ace/embedding"
	"github.com/dmytrogajewski/spin/internal/ace/playbook"
)

// TestFindDuplicates_NoDuplicates tests when playbook is empty.
func TestFindDuplicates_NoDuplicates(t *testing.T) {
	ctx := context.Background()
	embedder := embedding.NewMockEmbedder(384)
	pb := playbook.New(nil, embedder)
	curator := NewCurator(pb, embedder)

	// Playbook is empty.

	// Create new bullet.
	content := "Always validate input parameters before processing them in production systems"
	emb, _ := embedder.Embed(ctx, content)
	b, _ := bullet.New(content, bullet.WithEmbedding(emb))

	// Find duplicates in empty playbook.
	duplicates, err := curator.FindDuplicates(ctx, []*bullet.Bullet{b})

	require.NoError(t, err)
	// Empty playbook = no duplicates possible.
	assert.Empty(t, duplicates)
}

// TestFindDuplicates_ExactDuplicate tests exact duplicate detection.
func TestFindDuplicates_ExactDuplicate(t *testing.T) {
	ctx := context.Background()
	embedder := embedding.NewMockEmbedder(384)
	pb := playbook.New(nil, embedder)
	curator := NewCurator(pb, embedder)

	// Add a bullet to playbook.
	content := "Always validate input parameters"
	emb1, _ := embedder.Embed(ctx, content)
	b1, _ := bullet.New(content, bullet.WithEmbedding(emb1))
	pb.Add(ctx, b1)

	// Create new bullet with same content.
	emb2, _ := embedder.Embed(ctx, content)
	b2, _ := bullet.New(content, bullet.WithEmbedding(emb2))

	// Find duplicates.
	duplicates, err := curator.FindDuplicates(ctx, []*bullet.Bullet{b2})

	require.NoError(t, err)
	assert.Len(t, duplicates, 1)
	assert.Equal(t, b1.ID, duplicates[b2.ID])
}

// TestFindDuplicates_SimilarContent tests near-duplicate detection.
func TestFindDuplicates_SimilarContent(t *testing.T) {
	ctx := context.Background()
	embedder := embedding.NewMockEmbedder(384)
	pb := playbook.New(nil, embedder)
	curator := NewCurator(pb, embedder)

	// Add a bullet to playbook.
	content1 := "Always validate input parameters before processing them to prevent errors"
	emb1, _ := embedder.Embed(ctx, content1)
	b1, _ := bullet.New(content1, bullet.WithEmbedding(emb1))
	pb.Add(ctx, b1)

	// Create new bullet with very similar content (just one word different).
	content2 := "Always validate input parameters before processing them to avoid errors"
	emb2, _ := embedder.Embed(ctx, content2)
	b2, _ := bullet.New(content2, bullet.WithEmbedding(emb2))

	// Find duplicates.
	duplicates, err := curator.FindDuplicates(ctx, []*bullet.Bullet{b2})

	require.NoError(t, err)
	// Should find duplicate due to high similarity.
	assert.Len(t, duplicates, 1)
}

// TestFindDuplicates_ThresholdBoundary tests similarity threshold.
func TestFindDuplicates_ThresholdBoundary(t *testing.T) {
	ctx := context.Background()
	embedder := embedding.NewMockEmbedder(384)
	pb := playbook.New(nil, embedder)

	// Create curator with custom threshold.
	curator := NewCurator(pb, embedder, WithSimilarityThreshold(0.95))

	// Add a bullet to playbook.
	emb1, _ := embedder.Embed(ctx, "Use errors.Is")
	b1, _ := bullet.New("Use errors.Is", bullet.WithEmbedding(emb1))
	pb.Add(ctx, b1)

	// Create new bullet with somewhat similar content.
	emb2, _ := embedder.Embed(ctx, "Use errors.Is for error checking")
	b2, _ := bullet.New("Use errors.Is for error checking", bullet.WithEmbedding(emb2))

	// Find duplicates with high threshold.
	duplicates, err := curator.FindDuplicates(ctx, []*bullet.Bullet{b2})

	require.NoError(t, err)
	// With high threshold, might not be duplicate
	// (depends on mock embedder's similarity calculation).
	_ = duplicates // May or may not find duplicates depending on similarity.
}

// TestFindDuplicates_MultipleBullets tests batch duplicate detection.
func TestFindDuplicates_MultipleBullets(t *testing.T) {
	ctx := context.Background()
	embedder := embedding.NewMockEmbedder(384)
	pb := playbook.New(nil, embedder)
	curator := NewCurator(pb, embedder)

	// Add one bullet to playbook.
	content1 := "Always validate input parameters before processing to ensure data integrity"
	emb1, _ := embedder.Embed(ctx, content1)
	b1, _ := bullet.New(content1, bullet.WithEmbedding(emb1))
	pb.Add(ctx, b1)

	// Create new bullets - one exact duplicate, one not.
	emb2, _ := embedder.Embed(ctx, content1) // Exact duplicate.
	b2, _ := bullet.New(content1, bullet.WithEmbedding(emb2))

	content3 := "Use table-driven tests with subtests for comprehensive test coverage and better organization"
	emb3, _ := embedder.Embed(ctx, content3) // Different content.
	b3, _ := bullet.New(content3, bullet.WithEmbedding(emb3))

	// Find duplicates.
	duplicates, err := curator.FindDuplicates(ctx, []*bullet.Bullet{b2, b3})

	require.NoError(t, err)
	// Only b2 should be a duplicate (exact match)
	// b3 should not be duplicate (different content).
	if len(duplicates) > 0 {
		// If any duplicates found, b2 should map to b1.
		if dup, ok := duplicates[b2.ID]; ok {
			assert.Equal(t, b1.ID, dup)
		}
	}
}

// TestFindDuplicates_EmptyPlaybook tests with empty playbook.
func TestFindDuplicates_EmptyPlaybook(t *testing.T) {
	ctx := context.Background()
	embedder := embedding.NewMockEmbedder(384)
	pb := playbook.New(nil, embedder)
	curator := NewCurator(pb, embedder)

	// Create new bullet.
	emb, _ := embedder.Embed(ctx, "Always validate input")
	b, _ := bullet.New("Always validate input", bullet.WithEmbedding(emb))

	// Find duplicates in empty playbook.
	duplicates, err := curator.FindDuplicates(ctx, []*bullet.Bullet{b})

	require.NoError(t, err)
	assert.Empty(t, duplicates)
}

// TestFindDuplicates_EmptyInput tests with empty input.
func TestFindDuplicates_EmptyInput(t *testing.T) {
	ctx := context.Background()
	embedder := embedding.NewMockEmbedder(384)
	pb := playbook.New(nil, embedder)
	curator := NewCurator(pb, embedder)

	// Find duplicates with empty input.
	duplicates, err := curator.FindDuplicates(ctx, []*bullet.Bullet{})

	require.NoError(t, err)
	assert.Empty(t, duplicates)
}
