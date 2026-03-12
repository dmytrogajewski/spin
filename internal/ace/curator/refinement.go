package curator

import (
	"context"

	"github.com/dmytrogajewski/spin/internal/ace/playbook"
	"github.com/dmytrogajewski/spin/internal/ace/refine"
)

// RefinementStrategy defines how and when playbook refinement occurs.
type RefinementStrategy interface {
	// ShouldRefine returns true if playbook should be refined now.
	ShouldRefine(ctx context.Context, pb *playbook.Playbook) (bool, error)

	// Refine performs playbook refinement (prune low-utility bullets).
	Refine(ctx context.Context, pb *playbook.Playbook) (*RefinementResult, error)

	// SetOrchestrator sets the refinement orchestrator for advanced refinement.
	SetOrchestrator(orchestrator *refine.RefinementOrchestrator)
}

// RefinementResult contains the result of a refinement operation.
type RefinementResult struct {
	Pruned    int      // Bullets removed.
	PrunedIDs []string // IDs of removed bullets.
	Reason    string   // Why refinement occurred.
}

// RefinementMode specifies when refinement occurs.
type RefinementMode string

const (
	RefinementModeNone      RefinementMode = "none"      // No refinement.
	RefinementModeLazy      RefinementMode = "lazy"      // Manual only.
	RefinementModeProactive RefinementMode = "proactive" // After each Curate.
)

// noRefinementStrategy never refines.
type noRefinementStrategy struct{}

func (n *noRefinementStrategy) ShouldRefine(ctx context.Context, pb *playbook.Playbook) (bool, error) {
	return false, nil
}

func (n *noRefinementStrategy) Refine(ctx context.Context, pb *playbook.Playbook) (*RefinementResult, error) {
	return &RefinementResult{Pruned: 0, Reason: "no refinement"}, nil
}

func (n *noRefinementStrategy) SetOrchestrator(orchestrator *refine.RefinementOrchestrator) {
	// No-op.
}

// LazyRefinementConfig configures lazy refinement.
type LazyRefinementConfig struct {
	MinUtilityScore float64 // Prune bullets below this score (default: 0.1).
}

// lazyRefinementStrategy refines only on explicit call.
type lazyRefinementStrategy struct {
	minUtilityScore float64
	orchestrator    *refine.RefinementOrchestrator
}

func newLazyRefinementStrategy(cfg LazyRefinementConfig) RefinementStrategy {
	minScore := cfg.MinUtilityScore
	if minScore == 0 {
		minScore = 0.1
	}

	return &lazyRefinementStrategy{
		minUtilityScore: minScore,
	}
}

func (l *lazyRefinementStrategy) ShouldRefine(ctx context.Context, pb *playbook.Playbook) (bool, error) {
	// Lazy never auto-refines.
	return false, nil
}

func (l *lazyRefinementStrategy) Refine(ctx context.Context, pb *playbook.Playbook) (*RefinementResult, error) {
	// Use orchestrator if available for advanced refinement.
	if l.orchestrator != nil {
		return l.refineWithOrchestrator(ctx, pb, l.minUtilityScore, "manual refinement")
	}

	return refinePlaybook(ctx, pb, l.minUtilityScore, "manual refinement")
}

func (l *lazyRefinementStrategy) SetOrchestrator(orchestrator *refine.RefinementOrchestrator) {
	l.orchestrator = orchestrator
}

// ProactiveRefinementConfig configures proactive refinement.
type ProactiveRefinementConfig struct {
	MaxBullets      int     // Trigger at N bullets (default: 1000).
	MaxSizeBytes    int64   // Trigger at N bytes (default: 1MB).
	MinUtilityScore float64 // Prune bullets below score (default: 0.1).
}

// proactiveRefinementStrategy refines after each curate.
type proactiveRefinementStrategy struct {
	maxBullets      int
	maxSizeBytes    int64
	minUtilityScore float64
	orchestrator    *refine.RefinementOrchestrator
}

func newProactiveRefinementStrategy(cfg ProactiveRefinementConfig) RefinementStrategy {
	maxBullets := cfg.MaxBullets
	if maxBullets == 0 {
		maxBullets = 1000
	}

	maxSizeBytes := cfg.MaxSizeBytes
	if maxSizeBytes == 0 {
		maxSizeBytes = 1024 * 1024 // 1MB.
	}

	minScore := cfg.MinUtilityScore
	if minScore == 0 {
		minScore = 0.1
	}

	return &proactiveRefinementStrategy{
		maxBullets:      maxBullets,
		maxSizeBytes:    maxSizeBytes,
		minUtilityScore: minScore,
	}
}

func (p *proactiveRefinementStrategy) ShouldRefine(ctx context.Context, pb *playbook.Playbook) (bool, error) {
	stats := pb.Stats()

	return stats.TotalBullets >= p.maxBullets, nil
}

func (p *proactiveRefinementStrategy) Refine(ctx context.Context, pb *playbook.Playbook) (*RefinementResult, error) {
	// Use orchestrator if available for advanced refinement with merging.
	if p.orchestrator != nil {
		return p.refineWithOrchestrator(ctx, pb, p.minUtilityScore, "proactive refinement")
	}

	return refinePlaybook(ctx, pb, p.minUtilityScore, "proactive refinement")
}

func (p *proactiveRefinementStrategy) SetOrchestrator(orchestrator *refine.RefinementOrchestrator) {
	p.orchestrator = orchestrator
}

// refinePlaybook is the common refinement logic.
func refinePlaybook(ctx context.Context, pb *playbook.Playbook, minUtilityScore float64, reason string) (*RefinementResult, error) {
	bullets := pb.List(nil)
	prunedIDs := make([]string, 0)

	for _, b := range bullets {
		score := b.Score()
		if score < minUtilityScore {
			err := pb.Delete(ctx, b.ID)
			if err != nil {
				return nil, err
			}

			prunedIDs = append(prunedIDs, b.ID)
		}
	}

	return &RefinementResult{
		Pruned:    len(prunedIDs),
		PrunedIDs: prunedIDs,
		Reason:    reason,
	}, nil
}

// refineWithOrchestrator is shared logic for strategies using the orchestrator.
func refineWithOrchestrator(ctx context.Context, orchestrator *refine.RefinementOrchestrator, minUtilityScore float64, reason string) (*RefinementResult, error) {
	req := refine.RefinementRequest{
		PruneEnabled:    true,
		MergeEnabled:    true,
		ArchiveEnabled:  false,
		MinUtility:      minUtilityScore,
		MergeSimilarity: 0.90,
	}

	result, err := orchestrator.Refine(ctx, req)
	if err != nil {
		return nil, err
	}

	return &RefinementResult{
		Pruned:    result.Pruned + result.Merged, // Count merged bullets as pruned.
		PrunedIDs: result.PrunedIDs,
		Reason:    reason,
	}, nil
}

func (l *lazyRefinementStrategy) refineWithOrchestrator(ctx context.Context, pb *playbook.Playbook, minUtilityScore float64, reason string) (*RefinementResult, error) {
	return refineWithOrchestrator(ctx, l.orchestrator, minUtilityScore, reason)
}

func (p *proactiveRefinementStrategy) refineWithOrchestrator(ctx context.Context, pb *playbook.Playbook, minUtilityScore float64, reason string) (*RefinementResult, error) {
	return refineWithOrchestrator(ctx, p.orchestrator, minUtilityScore, reason)
}
