package refine

import (
	"context"
	"fmt"
	"time"

	"github.com/dmytrogajewski/spin/internal/ace/bullet"
	"github.com/dmytrogajewski/spin/internal/ace/playbook"
)

// RefinementRequest specifies what refinement operations to perform.
type RefinementRequest struct {
	PruneEnabled    bool    // Enable pruning of low-utility bullets.
	MergeEnabled    bool    // Enable merging of similar bullets.
	ArchiveEnabled  bool    // Enable archival of removed bullets.
	MinUtility      float64 // Prune bullets below this utility (default: 0.1).
	MergeSimilarity float64 // Merge bullets above this similarity (default: 0.90).
}

// RefinementResult contains refinement operation outcomes.
type RefinementResult struct {
	Pruned      int           // Number of bullets pruned.
	Merged      int           // Number of bullets merged.
	Archived    int           // Number of bullets archived.
	PrunedIDs   []string      // IDs of pruned bullets.
	MergedPairs []MergePair   // Pairs of merged bullets.
	TokensSaved int           // Estimated tokens saved.
	Duration    time.Duration // Time taken.
}

// PruneFunc is a function that prunes low-utility bullets from playbook.
// It should return the number of pruned bullets and their IDs.
type PruneFunc func(ctx context.Context) (pruned int, prunedIDs []string, err error)

// RefinementOrchestrator coordinates refinement operations.
type RefinementOrchestrator struct {
	playbook    *playbook.Playbook
	mergeEngine *MergeEngine
	archive     *Archive
	pruneFunc   PruneFunc
}

// NewRefinementOrchestrator creates a new orchestrator.
func NewRefinementOrchestrator(
	pb *playbook.Playbook,
	mergeEngine *MergeEngine,
	archive *Archive,
	pruneFunc PruneFunc,
) *RefinementOrchestrator {
	return &RefinementOrchestrator{
		playbook:    pb,
		mergeEngine: mergeEngine,
		archive:     archive,
		pruneFunc:   pruneFunc,
	}
}

// Refine executes the full refinement workflow.
func (o *RefinementOrchestrator) Refine(ctx context.Context, req RefinementRequest) (*RefinementResult, error) {
	start := time.Now()
	result := &RefinementResult{
		PrunedIDs:   make([]string, 0),
		MergedPairs: make([]MergePair, 0),
	}

	// Set defaults.
	if req.MinUtility == 0 {
		req.MinUtility = 0.1
	}

	if req.MergeSimilarity == 0 {
		req.MergeSimilarity = 0.90
	}

	// Step 1: Merge similar bullets (if enabled).
	if req.MergeEnabled && o.mergeEngine != nil {
		bullets := o.playbook.List(nil)

		pairs, err := o.mergeEngine.FindMergeCandidates(ctx, bullets)
		if err != nil {
			return nil, err
		}

		for _, pair := range pairs {
			source, sourceExists := o.playbook.Get(pair.SourceID)
			target, targetExists := o.playbook.Get(pair.TargetID)

			if !sourceExists || !targetExists {
				continue // Skip if bullet was already removed.
			}

			mergeResult, err := o.mergeEngine.MergeBullets(ctx, source, target)
			if err != nil {
				continue // Skip failed merges.
			}

			// Archive removed bullet (if archival enabled).
			if req.ArchiveEnabled && o.archive != nil {
				o.archive.Archive(source, ReasonMerged, map[string]string{
					"merged_into": mergeResult.KeptID,
					"similarity":  formatFloat(pair.Similarity),
				})

				result.Archived++
			}

			// Update the kept bullet in playbook.
			kept, _ := o.playbook.Get(mergeResult.KeptID)
			kept.HelpfulCount += source.HelpfulCount
			kept.HarmfulCount += source.HarmfulCount

			// Merge tags.
			if kept.Tags == nil {
				kept.Tags = make(map[string]string)
			}

			for k, v := range source.Tags {
				if _, exists := kept.Tags[k]; !exists {
					kept.Tags[k] = v
				}
			}

			o.playbook.Update(ctx, kept)

			// Remove source bullet from playbook.
			o.playbook.Delete(ctx, mergeResult.RemovedID)

			result.Merged++
			result.MergedPairs = append(result.MergedPairs, pair)
		}
	}

	// Step 2: Prune low-utility bullets (if enabled).
	if req.PruneEnabled && o.pruneFunc != nil {
		// Get bullets that would be pruned before actual pruning (for archival).
		var bulletsToArchive []*bullet.Bullet

		if req.ArchiveEnabled && o.archive != nil {
			allBullets := o.playbook.List(nil)
			for _, b := range allBullets {
				if b.Score() < req.MinUtility {
					bulletsToArchive = append(bulletsToArchive, b.Clone())
				}
			}
		}

		// Use prune function to remove low-utility bullets.
		pruned, prunedIDs, err := o.pruneFunc(ctx)
		if err != nil {
			return nil, err
		}

		result.Pruned = pruned
		result.PrunedIDs = prunedIDs

		// Archive pruned bullets (if archival enabled).
		if req.ArchiveEnabled && o.archive != nil && len(bulletsToArchive) > 0 {
			for _, b := range bulletsToArchive {
				metadata := map[string]string{
					"utility_score": formatFloat(b.Score()),
					"min_threshold": formatFloat(req.MinUtility),
				}
				o.archive.Archive(b, ReasonLowUtility, metadata)

				result.Archived++
			}
		}
	}

	// Step 3: Calculate tokens saved (rough estimate).
	result.TokensSaved = (result.Pruned + result.Merged) * 50 // ~50 tokens per bullet.
	result.Duration = time.Since(start)

	return result, nil
}

// formatFloat converts float64 to string for metadata.
func formatFloat(f float64) string {
	return fmt.Sprintf("%.4f", f)
}
