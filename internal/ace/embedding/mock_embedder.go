package embedding

import (
	"context"
)

// MockEmbedder is a simple mock implementation for testing.
type MockEmbedder struct {
	dimension  int
	embeddings map[string][]float32
}

// NewMockEmbedder creates a new mock embedder with the given dimension.
func NewMockEmbedder(dimension int) *MockEmbedder {
	return &MockEmbedder{
		dimension:  dimension,
		embeddings: make(map[string][]float32),
	}
}

// SetEmbedding sets a predefined embedding for a text.
func (m *MockEmbedder) SetEmbedding(text string, embedding []float32) {
	m.embeddings[text] = embedding
}

// Embed returns a predefined embedding or generates a simple one.
func (m *MockEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	if embed, ok := m.embeddings[text]; ok {
		return embed, nil
	}

	// Generate simple embedding based on text length.
	embedding := make([]float32, m.dimension)
	for i := range m.dimension {
		embedding[i] = float32(len(text)%10) * 0.1
	}

	return embedding, nil
}

// Dimension returns the embedding dimension.
func (m *MockEmbedder) Dimension() int {
	return m.dimension
}
