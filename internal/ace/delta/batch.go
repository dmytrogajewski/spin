package delta

import (
	"context"
	"fmt"
	"runtime"
	"sync"
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

// ApplyBatch applies multiple deltas in parallel.
func (a *Applier) ApplyBatch(ctx context.Context, req BatchApplyRequest) (*BatchApplyResult, error) {
	workers := req.MaxWorkers
	if workers == 0 {
		workers = runtime.NumCPU()
	}

	result := &BatchApplyResult{
		Results: make([]ApplyResult, len(req.Deltas)),
	}

	// Worker pool with error collection.
	type job struct {
		index int
		delta Delta
	}

	jobs := make(chan job, len(req.Deltas))
	results := make(chan struct {
		index  int
		result *ApplyResult
		err    error
	}, len(req.Deltas))

	// Start workers.
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for j := range jobs {
				res, err := a.Apply(ctx, j.delta)
				results <- struct {
					index  int
					result *ApplyResult
					err    error
				}{j.index, res, err}
			}
		}()
	}

	// Submit jobs.
	for i, delta := range req.Deltas {
		jobs <- job{i, delta}
	}

	close(jobs)

	// Collect results.
	go func() {
		wg.Wait()
		close(results)
	}()

	var firstError error

	successfulIndices := make([]int, 0)

	for r := range results {
		if r.err != nil {
			result.Failed++
			result.Results[r.index] = ApplyResult{Success: false, Error: r.err}

			if firstError == nil {
				firstError = r.err
			}

			// Atomic mode: mark for rollback.
			if req.Atomic {
				result.RolledBack = true
			}
		} else {
			result.Applied++
			result.Results[r.index] = *r.result
			successfulIndices = append(successfulIndices, r.index)
		}
	}

	// If atomic mode and there were failures, rollback successful deltas.
	if req.Atomic && firstError != nil {
		err := a.rollbackDeltas(ctx, req.Deltas, successfulIndices)
		if err != nil {
			// Log rollback error but return original error.
			return result, fmt.Errorf("rollback failed after error: %w (original error: %w)", err, firstError)
		}

		return result, firstError
	}

	return result, nil
}

// rollbackDeltas reverses the effects of successfully applied deltas.
// This is called when atomic mode is enabled and a failure occurs.
func (a *Applier) rollbackDeltas(ctx context.Context, deltas []Delta, successfulIndices []int) error {
	if len(successfulIndices) == 0 {
		return nil // Nothing to rollback.
	}

	// Create inverse deltas to undo the changes.
	for _, idx := range successfulIndices {
		delta := deltas[idx]

		// Get the bullet.
		b, exists := a.playbook.Get(delta.BulletID)
		if !exists {
			continue // Bullet may have been deleted, skip.
		}

		// Apply inverse operation based on operation type.
		switch delta.Operation {
		case OpUpdateContent:
			// Cannot reliably rollback content updates without storing old value
			// This is a limitation of the current design.
			continue

		case OpIncrementHelpful:
			// Decrement helpful count.
			if b.HelpfulCount > 0 {
				b.HelpfulCount--
				err := a.playbook.Update(ctx, b)
				if err != nil {
					return fmt.Errorf("failed to rollback helpful increment for bullet %s: %w", delta.BulletID, err)
				}
			}

		case OpIncrementHarmful:
			// Decrement harmful count.
			if b.HarmfulCount > 0 {
				b.HarmfulCount--
				err := a.playbook.Update(ctx, b)
				if err != nil {
					return fmt.Errorf("failed to rollback harmful increment for bullet %s: %w", delta.BulletID, err)
				}
			}

		case OpAddTag:
			// Remove the tag that was added.
			if delta.Fields.TagKey != nil && b.Tags != nil {
				delete(b.Tags, *delta.Fields.TagKey)

				err := a.playbook.Update(ctx, b)
				if err != nil {
					return fmt.Errorf("failed to rollback tag addition for bullet %s: %w", delta.BulletID, err)
				}
			}

		case OpRemoveTag:
			// Cannot reliably restore removed tag without storing old value.
			continue

		case OpUpdateEmbedding:
			// Cannot reliably rollback embedding updates without storing old value.
			continue
		}
	}

	return nil
}
