package refine

import (
	"context"
	"errors"

	"github.com/dmytrogajewski/spin/internal/ace/bullet"
	"github.com/dmytrogajewski/spin/internal/ace/embedding"
	"github.com/dmytrogajewski/spin/pkg/alg/similarity"
	"github.com/dmytrogajewski/spin/pkg/alg/vector"
)

// ErrSourceAndTargetBulletsCannotBe is a sentinel error.
var ErrSourceAndTargetBulletsCannotBe = errors.New("source and target bullets cannot be nil")

// MergePair represents two bullets to merge.
type MergePair struct {
	SourceID   string  // Bullet to merge from (removed during merge).
	TargetID   string  // Bullet to merge into (kept after merge).
	Similarity float64 // Similarity score.
}

// MergeResult contains merge operation outcome.
type MergeResult struct {
	KeptID        string // ID of kept bullet.
	RemovedID     string // ID of removed bullet.
	MergedContent string // Content of kept bullet.
}

// MergeEngine identifies and merges similar bullets.
type MergeEngine struct {
	embedder   embedding.Embedder
	similarity float64 // Threshold for merging (0.0-1.0).
}

// NewMergeEngine creates a new merge engine.
func NewMergeEngine(embedder embedding.Embedder, similarityThreshold float64) *MergeEngine {
	if similarityThreshold < 0.0 || similarityThreshold > 1.0 {
		similarityThreshold = 0.90 // Default to 0.90.
	}

	return &MergeEngine{
		embedder:   embedder,
		similarity: similarityThreshold,
	}
}

// FindMergeCandidates identifies bullet pairs to merge based on similarity.
func (m *MergeEngine) FindMergeCandidates(ctx context.Context, bullets []*bullet.Bullet) ([]MergePair, error) {
	pairs := make([]MergePair, 0)

	// O(n²) similarity comparison (acceptable for < 1000 bullets).
	for i := range bullets {
		for j := i + 1; j < len(bullets); j++ {
			similarity, err := m.calculateSimilarity(ctx, bullets[i], bullets[j])
			if err != nil {
				continue // Skip on error.
			}

			if similarity >= m.similarity {
				// Choose which to keep based on utility score.
				sourceID, targetID := m.chooseMergeDirection(bullets[i], bullets[j])

				pairs = append(pairs, MergePair{
					SourceID:   sourceID,
					TargetID:   targetID,
					Similarity: similarity,
				})
			}
		}
	}

	return pairs, nil
}

// MergeBullets combines source into target.
// Returns the result with kept and removed bullet IDs.
func (m *MergeEngine) MergeBullets(_ context.Context, source, target *bullet.Bullet) (*MergeResult, error) {
	if source == nil || target == nil {
		return nil, ErrSourceAndTargetBulletsCannotBe
	}

	// Determine which to keep based on utility score.
	kept := target
	removed := source

	if source.Score() > target.Score() {
		kept = source
		removed = target
	}

	// Clone the kept bullet to avoid modifying the original.
	merged := kept.Clone()

	// Transfer utility counters.
	merged.HelpfulCount += removed.HelpfulCount
	merged.HarmfulCount += removed.HarmfulCount

	// Merge tags (kept's tags take precedence).
	if merged.Tags == nil {
		merged.Tags = make(map[string]string)
	}

	for k, v := range removed.Tags {
		if _, exists := merged.Tags[k]; !exists {
			merged.Tags[k] = v
		}
	}

	return &MergeResult{
		KeptID:        merged.ID,
		RemovedID:     removed.ID,
		MergedContent: merged.Content,
	}, nil
}

// calculateSimilarity computes cosine similarity between two bullets.
func (m *MergeEngine) calculateSimilarity(ctx context.Context, b1, b2 *bullet.Bullet) (float64, error) {
	// If both have embeddings, use them.
	if len(b1.Embedding) > 0 && len(b2.Embedding) > 0 {
		return vector.CosineSimilarity(b1.Embedding, b2.Embedding), nil
	}

	// If embedder is available, generate embeddings.
	if m.embedder != nil {
		emb1, err := m.embedder.Embed(ctx, b1.Content)
		if err != nil {
			return 0.0, err
		}

		emb2, err := m.embedder.Embed(ctx, b2.Content)
		if err != nil {
			return 0.0, err
		}

		return vector.CosineSimilarity(emb1, emb2), nil
	}

	// No embeddings available — fall back to word-set (Jaccard) comparison.
	return similarity.JaccardSimilarity(b1.Content, b2.Content), nil
}


// chooseMergeDirection determines which bullet should be kept.
// Returns (sourceID, targetID) where source is merged into target.
func (m *MergeEngine) chooseMergeDirection(b1, b2 *bullet.Bullet) (sourceID, targetID string) {
	// Keep the bullet with higher utility score.
	score1 := b1.Score()
	score2 := b2.Score()

	if score1 > score2 {
		return b2.ID, b1.ID // b2 is source (removed), b1 is target (kept).
	} else if score2 > score1 {
		return b1.ID, b2.ID // b1 is source (removed), b2 is target (kept).
	}

	// If equal scores, prefer higher helpful count.
	if b1.HelpfulCount > b2.HelpfulCount {
		return b2.ID, b1.ID
	}

	return b1.ID, b2.ID
}
