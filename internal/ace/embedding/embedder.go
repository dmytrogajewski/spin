package embedding

import "context"

// Embedder generates semantic embeddings for text.
// Implementations can use local models or external APIs.
type Embedder interface {
	// Embed generates a semantic vector for the given text.
	Embed(ctx context.Context, text string) ([]float32, error)

	// Dimension returns the embedding dimension.
	Dimension() int
}
