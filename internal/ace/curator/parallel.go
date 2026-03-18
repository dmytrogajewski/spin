package curator

import (
	"context"
	"fmt"
	"runtime"
	"sync"
)

// jobResult holds the result of processing a single merge request.
type jobResult struct {
	index  int
	result *MergeResult
	err    error
}

// curateBatchParallel processes requests in parallel using a worker pool.
func (c *curator) curateBatchParallel(ctx context.Context, requests []MergeRequest, maxWorkers int) (*BatchMergeResult, error) {
	numWorkers := effectiveWorkers(maxWorkers, len(requests))

	jobs := make(chan int, len(requests))
	resultsChan := make(chan jobResult, len(requests))

	var wg sync.WaitGroup

	c.startWorkers(ctx, &wg, numWorkers, requests, jobs, resultsChan)

	go sendJobs(ctx, jobs, len(requests))
	go closeOnDone(&wg, resultsChan)

	return collectResults(ctx, resultsChan, len(requests))
}

// effectiveWorkers determines the number of workers capped by CPU count and request count.
func effectiveWorkers(maxWorkers, numRequests int) int {
	n := maxWorkers
	if n <= 0 {
		n = runtime.NumCPU()
	}

	if n > numRequests {
		n = numRequests
	}

	return n
}

// startWorkers launches worker goroutines that process jobs.
func (c *curator) startWorkers(
	ctx context.Context, wg *sync.WaitGroup, numWorkers int,
	requests []MergeRequest, jobs <-chan int, results chan<- jobResult,
) {
	for range numWorkers {
		wg.Go(func() {
			c.processJobs(ctx, requests, jobs, results)
		})
	}
}

// processJobs reads job indices from the channel and curates each request.
func (c *curator) processJobs(ctx context.Context, requests []MergeRequest, jobs <-chan int, results chan<- jobResult) {
	for {
		select {
		case <-ctx.Done():
			return
		case index, ok := <-jobs:
			if !ok {
				return
			}

			result, err := c.Curate(ctx, requests[index])
			results <- jobResult{index: index, result: result, err: err}
		}
	}
}

// sendJobs sends job indices to the jobs channel.
func sendJobs(ctx context.Context, jobs chan<- int, count int) {
	for i := range count {
		select {
		case <-ctx.Done():
			return
		case jobs <- i:
		}
	}

	close(jobs)
}

// closeOnDone waits for the WaitGroup and closes the results channel.
func closeOnDone(wg *sync.WaitGroup, results chan jobResult) {
	wg.Wait()
	close(results)
}

// collectResults gathers results from the results channel.
func collectResults(ctx context.Context, resultsChan <-chan jobResult, count int) (*BatchMergeResult, error) {
	results := make([]MergeResult, count)
	errors := make([]error, count)

	for jobRes := range resultsChan {
		if jobRes.err != nil {
			errors[jobRes.index] = jobRes.err
		} else {
			results[jobRes.index] = *jobRes.result
		}
	}

	if ctx.Err() != nil {
		return nil, fmt.Errorf("batch merge canceled: %w", ctx.Err())
	}

	return &BatchMergeResult{Results: results, Errors: errors}, nil
}
