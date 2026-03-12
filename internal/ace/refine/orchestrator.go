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

	if err := o.executeMergeStep(ctx, req, result); err != nil {
		return nil, err
	}

	if err := o.executePruneStep(ctx, req, result); err != nil {
		return nil, err
	}

	// Calculate tokens saved (rough estimate).
	result.TokensSaved = (result.Pruned + result.Merged) * tokensPerBulletEstimate // ~50 tokens per bullet.
	result.Duration = time.Since(start)

	return result, nil
}

// executeMergeStep merges similar bullets if enabled.
func (o *RefinementOrchestrator) executeMergeStep(ctx context.Context, req RefinementRequest, result *RefinementResult) error {
	if !req.MergeEnabled || o.mergeEngine == nil {
		return nil
	}

	bullets := o.playbook.List(nil)

	pairs, err := o.mergeEngine.FindMergeCandidates(ctx, bullets)
	if err != nil {
		return err
	}

	for _, pair := range pairs {
		o.processMergePair(ctx, req, pair, result)
	}

	return nil
}

// processMergePair handles merging a single pair of similar bullets.
func (o *RefinementOrchestrator) processMergePair(ctx context.Context, req RefinementRequest, pair MergePair, result *RefinementResult) {
	source, sourceExists := o.playbook.Get(pair.SourceID)

	target, targetExists := o.playbook.Get(pair.TargetID)
	if !sourceExists || !targetExists {
		return
	}

	mergeResult, err := o.mergeEngine.MergeBullets(ctx, source, target)
	if err != nil {
		return
	}

	if req.ArchiveEnabled && o.archive != nil {
		o.archive.Archive(source, ReasonMerged, map[string]string{
			"merged_into": mergeResult.KeptID,
			"similarity":  formatFloat(pair.Similarity),
		})

		result.Archived++
	}

	o.updateKeptBullet(ctx, mergeResult.KeptID, source)
	_ = o.playbook.Delete(ctx, mergeResult.RemovedID)

	result.Merged++
	result.MergedPairs = append(result.MergedPairs, pair)
}

// updateKeptBullet transfers counts and tags from the source bullet to the kept bullet.
func (o *RefinementOrchestrator) updateKeptBullet(ctx context.Context, keptID string, source *bullet.Bullet) {
	kept, _ := o.playbook.Get(keptID)
	kept.HelpfulCount += source.HelpfulCount
	kept.HarmfulCount += source.HarmfulCount

	if kept.Tags == nil {
		kept.Tags = make(map[string]string)
	}

	for k, v := range source.Tags {
		if _, exists := kept.Tags[k]; !exists {
			kept.Tags[k] = v
		}
	}

	_ = o.playbook.Update(ctx, kept)
}

// executePruneStep prunes low-utility bullets if enabled.
func (o *RefinementOrchestrator) executePruneStep(ctx context.Context, req RefinementRequest, result *RefinementResult) error {
	if !req.PruneEnabled || o.pruneFunc == nil {
		return nil
	}

	bulletsToArchive := o.collectBulletsForArchival(req)

	pruned, prunedIDs, err := o.pruneFunc(ctx)
	if err != nil {
		return err
	}

	result.Pruned = pruned
	result.PrunedIDs = prunedIDs

	// Archive pruned bullets.
	for _, b := range bulletsToArchive {
		o.archive.Archive(b, ReasonLowUtility, map[string]string{
			"utility_score": formatFloat(b.Score()),
			"min_threshold": formatFloat(req.MinUtility),
		})

		result.Archived++
	}

	return nil
}

// collectBulletsForArchival identifies low-utility bullets to archive before pruning.
func (o *RefinementOrchestrator) collectBulletsForArchival(req RefinementRequest) []*bullet.Bullet {
	if !req.ArchiveEnabled || o.archive == nil {
		return nil
	}

	var bulletsToArchive []*bullet.Bullet

	for _, b := range o.playbook.List(nil) {
		if b.Score() < req.MinUtility {
			bulletsToArchive = append(bulletsToArchive, b.Clone())
		}
	}

	return bulletsToArchive
}

// formatFloat converts float64 to string for metadata.
func formatFloat(f float64) string {
	return fmt.Sprintf("%.4f", f)
}
