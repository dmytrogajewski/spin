// Package retrieval provides HNSW-based vector retrieval.
package retrieval

import (
	"context"
	"errors"
	"sort"

	"github.com/coder/hnsw"

	"github.com/dmytrogajewski/spin/internal/ace/bullet"
	"github.com/dmytrogajewski/spin/internal/ace/embedding"
	"github.com/dmytrogajewski/spin/internal/ace/playbook"
)

var ErrBulletHasNoEmbedding = errors.New("bullet has no embedding")

// HNSWRetriever uses HNSW (Hierarchical Navigable Small World) graph for fast vector search.
// Provides O(log n) search complexity instead of O(n) with linear scan.
type HNSWRetriever struct {
	playbook *playbook.Playbook
	embedder embedding.Embedder
	graph    *hnsw.Graph[string]       // Maps bullet ID to embedding.
	indexMap map[string]*bullet.Bullet // Maps bullet ID to bullet.
}

// NewHNSWRetriever creates a new HNSW-based retriever.
func NewHNSWRetriever(pb *playbook.Playbook, emb embedding.Embedder) *HNSWRetriever {
	retriever := &HNSWRetriever{
		playbook: pb,
		embedder: emb,
		graph:    hnsw.NewGraph[string](),
		indexMap: make(map[string]*bullet.Bullet),
	}

	// Build HNSW index from existing bullets.
	retriever.rebuildIndex()

	return retriever
}

// rebuildIndex builds the HNSW graph from all bullets in the playbook.
func (r *HNSWRetriever) rebuildIndex() {
	// Clear existing index.
	r.graph = hnsw.NewGraph[string]()
	r.indexMap = make(map[string]*bullet.Bullet)

	// Add all bullets with embeddings to the graph.
	allBullets := r.playbook.List(func(_ *bullet.Bullet) bool { return true })
	for _, b := range allBullets {
		if len(b.Embedding) == 0 {
			continue // Skip bullets without embeddings.
		}

		// Add to HNSW graph.
		node := hnsw.MakeNode(b.ID, b.Embedding)
		r.graph.Add(node)

		// Add to index map for lookup.
		r.indexMap[b.ID] = b
	}
}

// Retrieve implements Retriever interface using HNSW search.
func (r *HNSWRetriever) Retrieve(ctx context.Context, query string, topK int) ([]*bullet.Bullet, error) {
	scored, err := r.RetrieveWithScores(ctx, query, topK)
	if err != nil {
		return nil, err
	}

	results := make([]*bullet.Bullet, len(scored))
	for i, s := range scored {
		results[i] = s.Bullet
	}

	return results, nil
}

// RetrieveWithScores returns bullets with relevance scores using HNSW search.
func (r *HNSWRetriever) RetrieveWithScores(ctx context.Context, query string, topK int) ([]ScoredBullet, error) {
	if r.embedder == nil {
		return []ScoredBullet{}, nil
	}

	// Generate query embedding.
	queryEmbed, err := r.embedder.Embed(ctx, query)
	if err != nil {
		return nil, err
	}

	// Check if index needs rebuilding (bullets may have been added).
	bulletsWithEmbeddings := r.playbook.List(func(b *bullet.Bullet) bool { return len(b.Embedding) > 0 })
	currentBulletCount := len(bulletsWithEmbeddings)

	if currentBulletCount != len(r.indexMap) {
		// Index is stale, rebuild it.
		r.rebuildIndex()
	}

	// Search HNSW graph for nearest neighbors
	// HNSW returns nodes sorted by distance (closest first).
	neighbors := r.graph.Search(queryEmbed, topK)

	// Convert to ScoredBullet results.
	results := make([]ScoredBullet, 0, len(neighbors))
	for _, neighbor := range neighbors {
		bulletID := neighbor.Key

		b, ok := r.indexMap[bulletID]
		if !ok {
			continue // Bullet was deleted.
		}

		// Calculate cosine similarity from L2 distance
		// HNSW uses L2 distance by default, we need to convert to similarity.
		similarity := cosineSimilarity(queryEmbed, b.Embedding)

		results = append(results, ScoredBullet{
			Bullet: b,
			Score:  similarity,
		})
	}

	// Sort by score descending (HNSW returns by distance, not similarity).
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results, nil
}

// AddBullet adds a new bullet to the HNSW index.
// This is more efficient than rebuilding the entire index.
func (r *HNSWRetriever) AddBullet(b *bullet.Bullet) error {
	if len(b.Embedding) == 0 {
		return ErrBulletHasNoEmbedding
	}

	// Add to HNSW graph.
	node := hnsw.MakeNode(b.ID, b.Embedding)
	r.graph.Add(node)

	// Add to index map.
	r.indexMap[b.ID] = b

	return nil
}

// RemoveBullet removes a bullet from the HNSW index.
func (r *HNSWRetriever) RemoveBullet(bulletID string) error {
	// Remove from graph.
	r.graph.Delete(bulletID)

	// Remove from index map.
	delete(r.indexMap, bulletID)

	return nil
}

// cosineSimilarity calculates cosine similarity between two vectors.
func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
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

	return dotProduct / (sqrt(normA) * sqrt(normB))
}

// sqrt is a simple square root implementation.
func sqrt(x float64) float64 {
	if x == 0 {
		return 0
	}
	// Use Newton's method for square root.
	z := x
	for range 10 {
		z = (z + x/z) / 2
	}

	return z
}
