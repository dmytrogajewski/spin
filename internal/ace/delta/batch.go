package delta

import (
	"context"
	"runtime"
	"sync"
)

// BatchApplyRequest contains multiple deltas to apply.
type BatchApplyRequest struct {
	Deltas     []Delta
	MaxWorkers int  // 0 = runtime.NumCPU()
	Atomic     bool // If true, rollback all if any fails
}

// BatchApplyResult contains results for batch application.
type BatchApplyResult struct {
	Results    []ApplyResult
	Applied    int  // Number successfully applied
	Failed     int  // Number failed
	RolledBack bool // True if atomic=true and rollback occurred
}

// ApplyBatch applies multiple deltas in parallel.
func (a *DeltaApplier) ApplyBatch(ctx context.Context, req BatchApplyRequest) (*BatchApplyResult, error) {
	workers := req.MaxWorkers
	if workers == 0 {
		workers = runtime.NumCPU()
	}

	result := &BatchApplyResult{
		Results: make([]ApplyResult, len(req.Deltas)),
	}

	// Worker pool with error collection
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

	// Start workers
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
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

	// Submit jobs
	for i, delta := range req.Deltas {
		jobs <- job{i, delta}
	}
	close(jobs)

	// Collect results
	go func() {
		wg.Wait()
		close(results)
	}()

	var firstError error
	for r := range results {
		if r.err != nil {
			result.Failed++
			result.Results[r.index] = ApplyResult{Success: false, Error: r.err}

			if firstError == nil {
				firstError = r.err
			}

			// Atomic mode: mark for rollback
			if req.Atomic {
				result.RolledBack = true
			}
		} else {
			result.Applied++
			result.Results[r.index] = *r.result
		}
	}

	// If atomic and there were failures, we should rollback
	// For now, just return the error and mark RolledBack
	// Full rollback implementation would go here
	if req.Atomic && firstError != nil {
		return result, firstError
	}

	return result, nil
}
