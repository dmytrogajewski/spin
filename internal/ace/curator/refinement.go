package curator

import (
	"context"

	"github.com/dmytrogajewski/spin/internal/ace/playbook"
	"github.com/dmytrogajewski/spin/internal/ace/refine"
)

const (
	bytesPerMB              = 1024 * 1024
	targetRefinementQuality = 0.90
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
	// RefinementModeNone defines a RefinementModeNone constant.
	RefinementModeNone RefinementMode = "none" // No refinement.
	// RefinementModeLazy defines a RefinementModeLazy constant.
	RefinementModeLazy RefinementMode = "lazy" // Manual only.
	// RefinementModeProactive refines after each Curate call.
	RefinementModeProactive RefinementMode = "proactive"
)

// noRefinementStrategy never refines.
type noRefinementStrategy struct{}

// ShouldRefine implements the ShouldRefine operation.
func (n *noRefinementStrategy) ShouldRefine(_ context.Context, _ *playbook.Playbook) (bool, error) {
	return false, nil
}

// Refine implements the Refine operation.
func (n *noRefinementStrategy) Refine(_ context.Context, _ *playbook.Playbook) (*RefinementResult, error) {
	return &RefinementResult{Pruned: 0, Reason: "no refinement"}, nil
}

// SetOrchestrator implements the SetOrchestrator operation.
func (n *noRefinementStrategy) SetOrchestrator(_ *refine.RefinementOrchestrator) {
	// No-op.
}

// orchestratorHolder embeds the shared orchestrator field and its
// refineWithOrchestrator method used by both lazy and proactive strategies.
type orchestratorHolder struct {
	orchestrator *refine.RefinementOrchestrator
}

// SetOrchestrator sets the refinement orchestrator for advanced refinement.
func (h *orchestratorHolder) SetOrchestrator(orchestrator *refine.RefinementOrchestrator) {
	h.orchestrator = orchestrator
}

// refineWithOrchestrator delegates to the package-level helper.
func (h *orchestratorHolder) refineWithOrchestrator(
	ctx context.Context, _ *playbook.Playbook,
	minUtilityScore float64, reason string,
) (*RefinementResult, error) {
	return refineWithOrchestrator(ctx, h.orchestrator, minUtilityScore, reason)
}

// LazyRefinementConfig configures lazy refinement.
type LazyRefinementConfig struct {
	MinUtilityScore float64 // Prune bullets below this score (default: 0.1).
}

// lazyRefinementStrategy refines only on explicit call.
type lazyRefinementStrategy struct {
	orchestratorHolder

	minUtilityScore float64
}

func newLazyRefinementStrategy(cfg LazyRefinementConfig) *lazyRefinementStrategy {
	minScore := cfg.MinUtilityScore
	if minScore == 0 {
		minScore = 0.1
	}

	return &lazyRefinementStrategy{
		minUtilityScore: minScore,
	}
}

// ShouldRefine implements the ShouldRefine operation.
func (l *lazyRefinementStrategy) ShouldRefine(_ context.Context, _ *playbook.Playbook) (bool, error) {
	// Lazy never auto-refines.
	return false, nil
}

// Refine implements the Refine operation.
func (l *lazyRefinementStrategy) Refine(ctx context.Context, pb *playbook.Playbook) (*RefinementResult, error) {
	// Use orchestrator if available for advanced refinement.
	if l.orchestrator != nil {
		return l.refineWithOrchestrator(ctx, pb, l.minUtilityScore, "manual refinement")
	}

	return refinePlaybook(ctx, pb, l.minUtilityScore, "manual refinement")
}

// ProactiveRefinementConfig configures proactive refinement.
type ProactiveRefinementConfig struct {
	MaxBullets      int     // Trigger at N bullets (default: 1000).
	MaxSizeBytes    int64   // Trigger at N bytes (default: 1MB).
	MinUtilityScore float64 // Prune bullets below score (default: 0.1).
}

// proactiveRefinementStrategy refines after each curate.
type proactiveRefinementStrategy struct {
	orchestratorHolder

	maxBullets      int
	maxSizeBytes    int64
	minUtilityScore float64
}

func newProactiveRefinementStrategy(cfg ProactiveRefinementConfig) *proactiveRefinementStrategy {
	maxBullets := cfg.MaxBullets
	if maxBullets == 0 {
		maxBullets = 1000
	}

	maxSizeBytes := cfg.MaxSizeBytes
	if maxSizeBytes == 0 {
		maxSizeBytes = bytesPerMB // 1MB.
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

// ShouldRefine implements the ShouldRefine operation.
func (p *proactiveRefinementStrategy) ShouldRefine(_ context.Context, pb *playbook.Playbook) (bool, error) {
	stats := pb.Stats()

	return stats.TotalBullets >= p.maxBullets, nil
}

// Refine implements the Refine operation.
func (p *proactiveRefinementStrategy) Refine(ctx context.Context, pb *playbook.Playbook) (*RefinementResult, error) {
	// Use orchestrator if available for advanced refinement with merging.
	if p.orchestrator != nil {
		return p.refineWithOrchestrator(ctx, pb, p.minUtilityScore, "proactive refinement")
	}

	return refinePlaybook(ctx, pb, p.minUtilityScore, "proactive refinement")
}

// refinePlaybook is the common refinement logic.
func refinePlaybook(ctx context.Context, pb *playbook.Playbook, minUtilityScore float64, reason string) (*RefinementResult, error) {
	pruned, prunedIDs, err := playbook.PruneLowUtility(ctx, pb, minUtilityScore)
	if err != nil {
		return nil, err
	}

	return &RefinementResult{
		Pruned:    pruned,
		PrunedIDs: prunedIDs,
		Reason:    reason,
	}, nil
}

// refineWithOrchestrator is shared logic for strategies using the orchestrator.
func refineWithOrchestrator(
	ctx context.Context, orchestrator *refine.RefinementOrchestrator,
	minUtilityScore float64, reason string,
) (*RefinementResult, error) {
	req := refine.RefinementRequest{
		PruneEnabled:    true,
		MergeEnabled:    true,
		ArchiveEnabled:  false,
		MinUtility:      minUtilityScore,
		MergeSimilarity: targetRefinementQuality,
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
