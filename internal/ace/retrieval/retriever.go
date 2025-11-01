package retrieval

import (
	"context"

	"github.com/dmytrogajewski/spin/internal/ace/bullet"
	"github.com/dmytrogajewski/spin/internal/ace/embedding"
	"github.com/dmytrogajewski/spin/internal/ace/playbook"
)

// Retriever retrieves relevant bullets for a query.
type Retriever interface {
	// Retrieve finds top-K relevant bullets for query.
	Retrieve(ctx context.Context, query string, topK int) ([]*bullet.Bullet, error)

	// RetrieveWithScores returns bullets with relevance scores.
	RetrieveWithScores(ctx context.Context, query string, topK int) ([]ScoredBullet, error)
}

// ScoredBullet is a bullet with relevance score.
type ScoredBullet struct {
	Bullet *bullet.Bullet
	Score  float64 // Relevance score (0.0 to 1.0)
}

// SemanticRetriever uses embeddings for retrieval.
type SemanticRetriever struct {
	playbook *playbook.Playbook
	embedder embedding.Embedder
}

// NewSemanticRetriever creates a semantic retriever.
func NewSemanticRetriever(pb *playbook.Playbook, emb embedding.Embedder) *SemanticRetriever {
	return &SemanticRetriever{
		playbook: pb,
		embedder: emb,
	}
}

// Retrieve implements Retriever interface.
func (r *SemanticRetriever) Retrieve(ctx context.Context, query string, topK int) ([]*bullet.Bullet, error) {
	// Delegate to playbook's Search method
	return r.playbook.Search(ctx, query, topK)
}

// RetrieveWithScores returns bullets with scores.
func (r *SemanticRetriever) RetrieveWithScores(ctx context.Context, query string, topK int) ([]ScoredBullet, error) {
	// For now, retrieve bullets and assign placeholder scores
	// TODO: Expose actual similarity scores from playbook.Search
	bullets, err := r.Retrieve(ctx, query, topK)
	if err != nil {
		return nil, err
	}

	scored := make([]ScoredBullet, len(bullets))
	for i, b := range bullets {
		scored[i] = ScoredBullet{
			Bullet: b,
			Score:  1.0, // Placeholder - will be real similarity score
		}
	}

	return scored, nil
}
