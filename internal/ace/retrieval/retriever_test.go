package retrieval

import (
	"context"
	"testing"

	"github.com/dmytrogajewski/spin/internal/ace/bullet"
	"github.com/dmytrogajewski/spin/internal/ace/embedding"
	"github.com/dmytrogajewski/spin/internal/ace/playbook"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSemanticRetriever_Creation(t *testing.T) {
	// Test creating a semantic retriever
	pb := playbook.New(nil, nil)
	embedder := embedding.NewMockEmbedder(1536)

	retriever := NewSemanticRetriever(pb, embedder)

	require.NotNil(t, retriever)
}

func TestSemanticRetriever_RetrieveEmpty(t *testing.T) {
	// Test retrieving from empty playbook
	ctx := context.Background()
	pb := playbook.New(nil, nil)
	embedder := embedding.NewMockEmbedder(1536)

	retriever := NewSemanticRetriever(pb, embedder)
	bullets, err := retriever.Retrieve(ctx, "test query", 5)

	require.NoError(t, err)
	assert.Empty(t, bullets)
}

func TestSemanticRetriever_RetrieveTopK(t *testing.T) {
	// Test retrieving top-K bullets
	ctx := context.Background()
	embedder := embedding.NewMockEmbedder(1536)

	// Create playbook with bullets
	pb := playbook.New(nil, embedder)

	// Generate embeddings for bullets
	emb1, err := embedder.Embed(ctx, "Always validate input")
	require.NoError(t, err)
	emb2, err := embedder.Embed(ctx, "Use context.Context for cancellation")
	require.NoError(t, err)
	emb3, err := embedder.Embed(ctx, "Prefer table-driven tests")
	require.NoError(t, err)

	b1, err := bullet.New("Always validate input", bullet.WithEmbedding(emb1))
	require.NoError(t, err)
	b2, err := bullet.New("Use context.Context for cancellation", bullet.WithEmbedding(emb2))
	require.NoError(t, err)
	b3, err := bullet.New("Prefer table-driven tests", bullet.WithEmbedding(emb3))
	require.NoError(t, err)

	require.NoError(t, pb.Add(ctx, b1))
	require.NoError(t, pb.Add(ctx, b2))
	require.NoError(t, pb.Add(ctx, b3))

	// Retrieve top 2
	retriever := NewSemanticRetriever(pb, embedder)
	bullets, err := retriever.Retrieve(ctx, "testing best practices", 2)

	require.NoError(t, err)
	assert.Len(t, bullets, 2)
}

func TestSemanticRetriever_RetrieveWithScores(t *testing.T) {
	// Test retrieving bullets with scores
	ctx := context.Background()
	embedder := embedding.NewMockEmbedder(1536)

	pb := playbook.New(nil, embedder)

	// Generate embedding for bullet
	emb, err := embedder.Embed(ctx, "Test bullet")
	require.NoError(t, err)

	b1, err := bullet.New("Test bullet", bullet.WithEmbedding(emb))
	require.NoError(t, err)
	require.NoError(t, pb.Add(ctx, b1))

	retriever := NewSemanticRetriever(pb, embedder)
	scored, err := retriever.RetrieveWithScores(ctx, "test", 1)

	require.NoError(t, err)
	assert.Len(t, scored, 1)
	assert.NotNil(t, scored[0].Bullet)
	assert.GreaterOrEqual(t, scored[0].Score, 0.0)
	assert.LessOrEqual(t, scored[0].Score, 1.0)
}

func TestSemanticRetriever_RetrieveTopKExceedsAvailable(t *testing.T) {
	// Test requesting more bullets than available
	ctx := context.Background()
	embedder := embedding.NewMockEmbedder(1536)

	pb := playbook.New(nil, embedder)

	// Add only 2 bullets
	emb1, err := embedder.Embed(ctx, "Bullet 1")
	require.NoError(t, err)
	emb2, err := embedder.Embed(ctx, "Bullet 2")
	require.NoError(t, err)

	b1, err := bullet.New("Bullet 1", bullet.WithEmbedding(emb1))
	require.NoError(t, err)
	b2, err := bullet.New("Bullet 2", bullet.WithEmbedding(emb2))
	require.NoError(t, err)

	require.NoError(t, pb.Add(ctx, b1))
	require.NoError(t, pb.Add(ctx, b2))

	// Request top 10 (more than available)
	retriever := NewSemanticRetriever(pb, embedder)
	bullets, err := retriever.Retrieve(ctx, "test", 10)

	require.NoError(t, err)
	// Should return all available bullets (2), not 10
	assert.Len(t, bullets, 2)
}
