package delta

import (
	"context"
	"fmt"
	"runtime"
	"sync"

	"github.com/dmytrogajewski/spin/internal/ace/bullet"
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

// batchJob pairs an index with a delta for worker processing.
type batchJob struct {
	index int
	delta Delta
}

// batchJobResult holds the outcome of applying a single delta.
type batchJobResult struct {
	index  int
	result *ApplyResult
	err    error
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

	resultsChan := a.runBatchWorkers(ctx, req.Deltas, workers)
	firstError, successfulIndices := collectBatchResults(result, resultsChan, req.Atomic)

	return a.finalizeBatch(ctx, result, req, firstError, successfulIndices)
}

// runBatchWorkers starts worker goroutines and returns the results channel.
func (a *Applier) runBatchWorkers(ctx context.Context, deltas []Delta, workers int) <-chan batchJobResult {
	jobs := make(chan batchJob, len(deltas))
	results := make(chan batchJobResult, len(deltas))

	var wg sync.WaitGroup

	for range workers {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for j := range jobs {
				res, err := a.Apply(ctx, j.delta)
				results <- batchJobResult{j.index, res, err}
			}
		}()
	}

	for i, d := range deltas {
		jobs <- batchJob{i, d}
	}

	close(jobs)

	go func() {
		wg.Wait()
		close(results)
	}()

	return results
}

// collectBatchResults processes results from workers, tracking successes and failures.
func collectBatchResults(result *BatchApplyResult, resultsChan <-chan batchJobResult, atomic bool) (error, []int) {
	var firstError error

	successfulIndices := make([]int, 0)

	for r := range resultsChan {
		if r.err != nil {
			result.Failed++
			result.Results[r.index] = ApplyResult{Success: false, Error: r.err}

			if firstError == nil {
				firstError = r.err
			}

			if atomic {
				result.RolledBack = true
			}
		} else {
			result.Applied++
			result.Results[r.index] = *r.result
			successfulIndices = append(successfulIndices, r.index)
		}
	}

	return firstError, successfulIndices
}

// finalizeBatch handles atomic rollback if needed.
func (a *Applier) finalizeBatch(ctx context.Context, result *BatchApplyResult, req BatchApplyRequest, firstError error, successfulIndices []int) (*BatchApplyResult, error) {
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
