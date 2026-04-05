package delta

import (
	"context"
	"fmt"

	"github.com/dmytrogajewski/spin/internal/ace/bullet"
	"github.com/dmytrogajewski/spin/pkg/alg/concurrency"
)

// BatchApplyRequest contains multiple deltas to apply.
type BatchApplyRequest struct {
	Deltas     []Delta
	MaxWorkers int  // 0 = runtime.NumCPU().
	Atomic     bool // If true, rollback all if any fails.
}

// BatchApplyResult contains results for batch application.
type BatchApplyResult struct {
	Results    []ApplyResult
	Applied    int  // Number successfully applied.
	Failed     int  // Number failed.
	RolledBack bool // True if atomic=true and rollback occurred.
}

// applyResult holds the outcome of applying a single delta.
type applyResult struct {
	result *ApplyResult
	err    error
}

// ApplyBatch applies multiple deltas in parallel.
func (a *Applier) ApplyBatch(ctx context.Context, req BatchApplyRequest) (*BatchApplyResult, error) {
	workers := concurrency.EffectiveWorkers(req.MaxWorkers, len(req.Deltas))

	poolResults := concurrency.WorkerPool(ctx, workers, req.Deltas, func(ctx context.Context, d Delta) applyResult {
		res, err := a.Apply(ctx, d)

		return applyResult{result: res, err: err}
	})

	result := &BatchApplyResult{
		Results: make([]ApplyResult, len(req.Deltas)),
	}

	successfulIndices, firstError := collectBatchResults(result, poolResults, req.Atomic)

	return a.finalizeBatch(ctx, result, req, firstError, successfulIndices)
}

// collectBatchResults processes results from workers, tracking successes and failures.
func collectBatchResults(result *BatchApplyResult, poolResults []applyResult, atomic bool) (successfulIndices []int, firstError error) {
	successfulIndices = make([]int, 0)

	for idx, res := range poolResults {
		if res.err != nil {
			result.Failed++
			result.Results[idx] = ApplyResult{Success: false, Error: res.err}

			if firstError == nil {
				firstError = res.err
			}

			if atomic {
				result.RolledBack = true
			}
		} else if res.result != nil {
			result.Applied++
			result.Results[idx] = *res.result

			successfulIndices = append(successfulIndices, idx)
		}
	}

	return successfulIndices, firstError
}

// finalizeBatch handles atomic rollback if needed.
func (a *Applier) finalizeBatch(
	ctx context.Context, result *BatchApplyResult,
	req BatchApplyRequest, firstError error, successfulIndices []int,
) (*BatchApplyResult, error) {
	if !req.Atomic || firstError == nil {
		return result, nil
	}

	if err := a.rollbackDeltas(ctx, req.Deltas, successfulIndices); err != nil {
		return result, fmt.Errorf("rollback failed after error: %w (original error: %w)", err, firstError)
	}

	return result, firstError
}

// rollbackDeltas reverses the effects of successfully applied deltas.
// This is called when atomic mode is enabled and a failure occurs.
func (a *Applier) rollbackDeltas(ctx context.Context, deltas []Delta, successfulIndices []int) error {
	if len(successfulIndices) == 0 {
		return nil // Nothing to rollback.
	}

	// Create inverse deltas to undo the changes.
	for _, idx := range successfulIndices {
		if err := a.rollbackSingleDelta(ctx, deltas[idx]); err != nil {
			return err
		}
	}

	return nil
}

// rollbackSingleDelta reverses a single successfully applied delta.
func (a *Applier) rollbackSingleDelta(ctx context.Context, d Delta) error {
	b, exists := a.playbook.Get(d.BulletID)
	if !exists {
		return nil
	}

	switch d.Operation {
	case OpUpdateContent, OpRemoveTag, OpUpdateEmbedding:
		return nil
	case OpIncrementHelpful:
		return a.rollbackCounter(ctx, b, d.BulletID, &b.HelpfulCount, "helpful")
	case OpIncrementHarmful:
		return a.rollbackCounter(ctx, b, d.BulletID, &b.HarmfulCount, "harmful")
	case OpAddTag:
		return a.rollbackTagAdd(ctx, b, d)
	}

	return nil
}

// rollbackCounter decrements a counter and persists if it was positive.
func (a *Applier) rollbackCounter(ctx context.Context, b *bullet.Bullet, bulletID string, counter *int, label string) error {
	if *counter <= 0 {
		return nil
	}

	*counter--

	if err := a.playbook.Update(ctx, b); err != nil {
		return fmt.Errorf("failed to rollback %s increment for bullet %s: %w", label, bulletID, err)
	}

	return nil
}

// rollbackTagAdd removes a tag that was added.
func (a *Applier) rollbackTagAdd(ctx context.Context, b *bullet.Bullet, d Delta) error {
	if d.Fields.TagKey == nil || b.Tags == nil {
		return nil
	}

	delete(b.Tags, *d.Fields.TagKey)

	if err := a.playbook.Update(ctx, b); err != nil {
		return fmt.Errorf("failed to rollback tag addition for bullet %s: %w", d.BulletID, err)
	}

	return nil
}
