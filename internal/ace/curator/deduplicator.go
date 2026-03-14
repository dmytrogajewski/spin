package curator

import (
	"context"

	"github.com/dmytrogajewski/spin/internal/ace/bullet"
	"github.com/dmytrogajewski/spin/internal/mathutil"
)

// FindDuplicates detects semantic duplicates using cosine similarity.
// Returns a map from new bullet ID to existing bullet ID for duplicates found.
func (c *curator) FindDuplicates(_ context.Context, newBullets []*bullet.Bullet) (map[string]string, error) {
	duplicates := make(map[string]string)

	// Handle empty input.
	if len(newBullets) == 0 {
		return duplicates, nil
	}

	// Get all existing bullets from playbook.
	existingBullets := c.playbook.List(nil)

	// Handle empty playbook.
	if len(existingBullets) == 0 {
		return duplicates, nil
	}

	// Check each new bullet against existing bullets.
	for _, newBullet := range newBullets {
		// Skip if no embedding.
		if len(newBullet.Embedding) == 0 {
			continue
		}

		// Find most similar existing bullet.
		maxSimilarity := 0.0

		var mostSimilarID string

		for _, existingBullet := range existingBullets {
			// Skip if no embedding.
			if len(existingBullet.Embedding) == 0 {
				continue
			}

			// Calculate cosine similarity.
			similarity := mathutil.CosineSimilarity(newBullet.Embedding, existingBullet.Embedding)

			if similarity > maxSimilarity {
				maxSimilarity = similarity
				mostSimilarID = existingBullet.ID
			}
		}

		// Check if similarity exceeds threshold.
		if maxSimilarity >= c.threshold {
			duplicates[newBullet.ID] = mostSimilarID
		}
	}

	return duplicates, nil
}
