package playbook

import (
	"context"
	"log/slog"

	"github.com/dmytrogajewski/spin/internal/ace/bullet"
	"github.com/dmytrogajewski/spin/internal/ace/embedding"
)

// PruneLowUtility removes bullets with a score below the given threshold.
// It returns the number of pruned bullets and their IDs.
func PruneLowUtility(ctx context.Context, pb *Playbook, threshold float64) (pruned int, ids []string, err error) {
	bullets := pb.List(nil)
	prunedIDs := make([]string, 0)

	for _, b := range bullets {
		if b.Score() < threshold {
			if delErr := pb.Delete(ctx, b.ID); delErr != nil {
				return 0, nil, delErr
			}

			prunedIDs = append(prunedIDs, b.ID)
		}
	}

	return len(prunedIDs), prunedIDs, nil
}

// EmbedAndAdd creates a bullet from content, generates its embedding using the
// provided embedder (if non-nil), and adds it to the playbook.
// Options are forwarded to bullet.New (e.g. bullet.WithID).
func EmbedAndAdd(
	ctx context.Context,
	pb *Playbook,
	embedder embedding.Embedder,
	content string,
	logger *slog.Logger,
	opts ...bullet.Option,
) (*bullet.Bullet, error) {
	b, err := bullet.New(content, opts...)
	if err != nil {
		return nil, err
	}

	if embedder != nil {
		emb, embErr := embedder.Embed(ctx, b.Content)
		if embErr != nil {
			if logger != nil {
				logger.WarnContext(ctx, "Failed to generate embedding", "error", embErr)
			}

			return nil, embErr
		}

		b.Embedding = emb
	}

	if addErr := pb.Add(ctx, b); addErr != nil {
		return nil, addErr
	}

	return b, nil
}
