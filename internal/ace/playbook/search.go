package playbook

import (
	"context"
	"math"
	"sort"

	"github.com/dmytrogajewski/spin/internal/ace/bullet"
)

// searchResult holds a bullet with its similarity score.
type searchResult struct {
	bullet     *bullet.Bullet
	similarity float64
}

// Search finds bullets by semantic similarity to the query.
// Returns top-k most similar bullets sorted by similarity (descending).
// Returns empty slice if embedder is nil or no bullets have embeddings.
func (p *Playbook) Search(ctx context.Context, query string, topK int) ([]*bullet.Bullet, error) {
	if p.embedder == nil {
		return []*bullet.Bullet{}, nil
	}

	// Generate query embedding
	queryEmbed, err := p.embedder.Embed(ctx, query)
	if err != nil {
		return nil, err
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	// Calculate similarities
	results := make([]searchResult, 0)
	for _, b := range p.bullets {
		if len(b.Embedding) == 0 {
			continue
		}

		similarity := cosineSimilarity(queryEmbed, b.Embedding)
		results = append(results, searchResult{
			bullet:     b,
			similarity: similarity,
		})
	}

	// Sort by similarity descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].similarity > results[j].similarity
	})

	// Return top-k
	if topK > len(results) {
		topK = len(results)
	}

	bullets := make([]*bullet.Bullet, topK)
	for i := 0; i < topK; i++ {
		bullets[i] = results[i].bullet
	}

	return bullets, nil
}

// cosineSimilarity calculates cosine similarity between two vectors.
func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0.0
	}

	var dotProduct, normA, normB float64
	for i := 0; i < len(a); i++ {
		dotProduct += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	if normA == 0 || normB == 0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}
