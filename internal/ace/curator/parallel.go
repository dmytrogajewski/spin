package curator

import (
	"context"
	"fmt"

	"github.com/dmytrogajewski/spin/pkg/alg/concurrency"
)

// curateResult holds the result of processing a single merge request.
type curateResult struct {
	result *MergeResult
	err    error
}

// curateBatchParallel processes requests in parallel using a worker pool.
func (c *curator) curateBatchParallel(ctx context.Context, requests []MergeRequest, maxWorkers int) (*BatchMergeResult, error) {
	numWorkers := concurrency.EffectiveWorkers(maxWorkers, len(requests))

	poolResults := concurrency.WorkerPool(ctx, numWorkers, requests, func(ctx context.Context, req MergeRequest) curateResult {
		result, err := c.Curate(ctx, req)

		return curateResult{result: result, err: err}
	})

	if ctx.Err() != nil {
		return nil, fmt.Errorf("batch merge canceled: %w", ctx.Err())
	}

	results := make([]MergeResult, len(requests))
	errs := make([]error, len(requests))

	for idx, res := range poolResults {
		if res.err != nil {
			errs[idx] = res.err
		} else if res.result != nil {
			results[idx] = *res.result
		}
	}

	return &BatchMergeResult{Results: results, Errors: errs}, nil
}
