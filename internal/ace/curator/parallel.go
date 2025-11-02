package curator

import (
	"context"
	"runtime"
	"sync"
)

// jobResult holds the result of processing a single merge request
type jobResult struct {
	index  int
	result *MergeResult
	err    error
}

// curateBatchParallel processes requests in parallel using a worker pool
func (c *curator) curateBatchParallel(ctx context.Context, requests []MergeRequest, maxWorkers int) (*BatchMergeResult, error) {
	// Determine worker count
	numWorkers := maxWorkers
	if numWorkers <= 0 {
		numWorkers = runtime.NumCPU()
	}
	// Cap workers at number of requests
	if numWorkers > len(requests) {
		numWorkers = len(requests)
	}

	// Create channels
	jobs := make(chan int, len(requests))
	resultsChan := make(chan jobResult, len(requests))

	// Start workers
	var wg sync.WaitGroup
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case index, ok := <-jobs:
					if !ok {
						return
					}
					result, err := c.Curate(ctx, requests[index])
					resultsChan <- jobResult{
						index:  index,
						result: result,
						err:    err,
					}
				}
			}
		}()
	}

	// Send jobs
	go func() {
		for i := range requests {
			select {
			case <-ctx.Done():
				return
			case jobs <- i:
			}
		}
		close(jobs)
	}()

	// Wait for all workers
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results
	results := make([]MergeResult, len(requests))
	errors := make([]error, len(requests))

	for jobRes := range resultsChan {
		if jobRes.err != nil {
			errors[jobRes.index] = jobRes.err
		} else {
			results[jobRes.index] = *jobRes.result
		}
	}

	// Check for context cancellation
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	return &BatchMergeResult{
		Results: results,
		Errors:  errors,
	}, nil
}
