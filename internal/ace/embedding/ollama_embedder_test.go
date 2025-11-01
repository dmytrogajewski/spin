package embedding

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOllamaEmbedder(t *testing.T) {
	config := DefaultOllamaEmbedderConfig()
	embedder, err := NewOllamaEmbedder(config)

	require.NoError(t, err)
	assert.NotNil(t, embedder)
	assert.Equal(t, 768, embedder.Dimension())
	assert.Equal(t, "nomic-embed-text", embedder.model)
}

func TestNewOllamaEmbedder_CustomConfig(t *testing.T) {
	config := OllamaEmbedderConfig{
		BaseURL:   "http://custom:11434",
		Model:     "mxbai-embed-large",
		Dimension: 1024,
	}

	embedder, err := NewOllamaEmbedder(config)

	require.NoError(t, err)
	assert.NotNil(t, embedder)
	assert.Equal(t, 1024, embedder.Dimension())
	assert.Equal(t, "mxbai-embed-large", embedder.model)
}

func TestNewOllamaEmbedder_InvalidURL(t *testing.T) {
	config := OllamaEmbedderConfig{
		BaseURL: "://invalid-url",
	}

	_, err := NewOllamaEmbedder(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid base URL")
}

func TestOllamaEmbedder_Embed_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	config := DefaultOllamaEmbedderConfig()
	embedder, err := NewOllamaEmbedder(config)
	require.NoError(t, err)

	ctx := context.Background()

	// Test embedding generation
	embedding, err := embedder.Embed(ctx, "Hello world")

	// If Ollama is not running, the test will fail - that's expected
	if err != nil {
		t.Skipf("Ollama not available: %v", err)
	}

	require.NoError(t, err)
	assert.NotNil(t, embedding)
	assert.Equal(t, 768, len(embedding), "Expected 768-dimensional embedding from nomic-embed-text")

	// Verify embedding is not all zeros
	hasNonZero := false
	for _, val := range embedding {
		if val != 0 {
			hasNonZero = true
			break
		}
	}
	assert.True(t, hasNonZero, "Embedding should not be all zeros")
}

func TestOllamaEmbedder_Embed_Similarity(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	config := DefaultOllamaEmbedderConfig()
	embedder, err := NewOllamaEmbedder(config)
	require.NoError(t, err)

	ctx := context.Background()

	// Generate embeddings for similar and dissimilar texts
	embed1, err1 := embedder.Embed(ctx, "The cat sits on the mat")
	embed2, err2 := embedder.Embed(ctx, "A feline rests on the rug")
	embed3, err3 := embedder.Embed(ctx, "Quantum physics and relativity")

	// Skip if Ollama is not available
	if err1 != nil || err2 != nil || err3 != nil {
		t.Skipf("Ollama not available")
	}

	// Calculate cosine similarities
	sim12 := cosineSimilarity(embed1, embed2)
	sim13 := cosineSimilarity(embed1, embed3)

	// Similar texts should have higher similarity than dissimilar ones
	assert.Greater(t, sim12, sim13, "Similar texts should have higher cosine similarity")
	assert.Greater(t, sim12, 0.5, "Similar texts should have similarity > 0.5")
}

// cosineSimilarity computes cosine similarity between two vectors
func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (normA * normB)
}
