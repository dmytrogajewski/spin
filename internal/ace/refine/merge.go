package refine

import (
	"context"
	"fmt"
	"math"

	"github.com/dmytrogajewski/spin/internal/ace/bullet"
	"github.com/dmytrogajewski/spin/internal/ace/embedding"
)

// MergePair represents two bullets to merge.
type MergePair struct {
	SourceID   string  // Bullet to merge from (removed during merge)
	TargetID   string  // Bullet to merge into (kept after merge)
	Similarity float64 // Similarity score
}

// MergeResult contains merge operation outcome.
type MergeResult struct {
	KeptID        string // ID of kept bullet
	RemovedID     string // ID of removed bullet
	MergedContent string // Content of kept bullet
}

// MergeEngine identifies and merges similar bullets.
type MergeEngine struct {
	embedder   embedding.Embedder
	similarity float64 // Threshold for merging (0.0-1.0)
}

// NewMergeEngine creates a new merge engine.
func NewMergeEngine(embedder embedding.Embedder, similarityThreshold float64) *MergeEngine {
	if similarityThreshold < 0.0 || similarityThreshold > 1.0 {
		similarityThreshold = 0.90 // Default to 0.90
	}

	return &MergeEngine{
		embedder:   embedder,
		similarity: similarityThreshold,
	}
}

// FindMergeCandidates identifies bullet pairs to merge based on similarity.
func (m *MergeEngine) FindMergeCandidates(ctx context.Context, bullets []*bullet.Bullet) ([]MergePair, error) {
	pairs := make([]MergePair, 0)

	// O(n²) similarity comparison (acceptable for < 1000 bullets)
	for i := 0; i < len(bullets); i++ {
		for j := i + 1; j < len(bullets); j++ {
			similarity, err := m.calculateSimilarity(ctx, bullets[i], bullets[j])
			if err != nil {
				continue // Skip on error
			}

			if similarity >= m.similarity {
				// Choose which to keep based on utility score
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
func (m *MergeEngine) MergeBullets(ctx context.Context, source, target *bullet.Bullet) (*MergeResult, error) {
	if source == nil || target == nil {
		return nil, fmt.Errorf("source and target bullets cannot be nil")
	}

	// Determine which to keep based on utility score
	kept := target
	removed := source

	if source.Score() > target.Score() {
		kept = source
		removed = target
	}

	// Clone the kept bullet to avoid modifying the original
	merged := kept.Clone()

	// Transfer utility counters
	merged.HelpfulCount += removed.HelpfulCount
	merged.HarmfulCount += removed.HarmfulCount

	// Merge tags (kept's tags take precedence)
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
	// If both have embeddings, use them
	if len(b1.Embedding) > 0 && len(b2.Embedding) > 0 {
		return m.cosineSimilarity(b1.Embedding, b2.Embedding), nil
	}

	// If embedder is available, generate embeddings
	if m.embedder != nil {
		emb1, err := m.embedder.Embed(ctx, b1.Content)
		if err != nil {
			return 0.0, err
		}

		emb2, err := m.embedder.Embed(ctx, b2.Content)
		if err != nil {
			return 0.0, err
		}

		return m.cosineSimilarity(emb1, emb2), nil
	}

	// No embeddings available - fall back to simple content comparison
	return m.simpleSimilarity(b1.Content, b2.Content), nil
}

// cosineSimilarity calculates cosine similarity between two vectors.
func (m *MergeEngine) cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0.0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	if normA == 0 || normB == 0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

// simpleSimilarity provides basic content-based similarity when no embeddings available.
func (m *MergeEngine) simpleSimilarity(content1, content2 string) float64 {
	// Exact match
	if content1 == content2 {
		return 1.0
	}

	// Length-based heuristic (very basic)
	minLen := len(content1)
	maxLen := len(content2)
	if maxLen < minLen {
		minLen, maxLen = maxLen, minLen
	}

	if maxLen == 0 {
		return 0.0
	}

	// Simple ratio (not very accurate, but better than nothing)
	return float64(minLen) / float64(maxLen)
}

// chooseMergeDirection determines which bullet should be kept.
// Returns (sourceID, targetID) where source is merged into target.
func (m *MergeEngine) chooseMergeDirection(b1, b2 *bullet.Bullet) (sourceID, targetID string) {
	// Keep the bullet with higher utility score
	score1 := b1.Score()
	score2 := b2.Score()

	if score1 > score2 {
		return b2.ID, b1.ID // b2 is source (removed), b1 is target (kept)
	} else if score2 > score1 {
		return b1.ID, b2.ID // b1 is source (removed), b2 is target (kept)
	}

	// If equal scores, prefer higher helpful count
	if b1.HelpfulCount > b2.HelpfulCount {
		return b2.ID, b1.ID
	}

	return b1.ID, b2.ID
}
