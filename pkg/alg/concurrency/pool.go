package concurrency

import (
	"context"
	"runtime"
	"sync"
)

// indexedResult pairs a result with its original job index for ordered output.
type indexedResult[Result any] struct {
	index  int
	result Result
}

// EffectiveWorkers returns the number of workers to use, capped by both
// the number of available CPUs and the job count. If requested <= 0,
// defaults to [runtime.NumCPU].
func EffectiveWorkers(requested, jobCount int) int {
	workers := requested
	if workers <= 0 {
		workers = runtime.NumCPU()
	}

	if workers > jobCount {
		workers = jobCount
	}

	return workers
}

// WorkerPool executes fn for each job using the specified number of workers.
// Results are returned in the same order as the input jobs slice.
// Respects context cancellation: canceled jobs receive zero-value results.
func WorkerPool[Job, Result any](
	ctx context.Context,
	workers int,
	jobs []Job,
	fn func(context.Context, Job) Result,
) []Result {
	if len(jobs) == 0 {
		return nil
	}

	jobsCh := make(chan int, len(jobs))
	resultsCh := make(chan indexedResult[Result], len(jobs))

	var wg sync.WaitGroup

	startPoolWorkers(ctx, &wg, workers, jobs, jobsCh, resultsCh, fn)
	sendPoolJobs(ctx, jobsCh, len(jobs))

	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	return collectPoolResults(resultsCh, len(jobs))
}

// startPoolWorkers launches workers that read job indices from jobsCh.
func startPoolWorkers[Job, Result any](
	ctx context.Context,
	wg *sync.WaitGroup,
	workers int,
	jobs []Job,
	jobsCh <-chan int,
	resultsCh chan<- indexedResult[Result],
	fn func(context.Context, Job) Result,
) {
	for range workers {
		wg.Go(func() {
			for {
				select {
				case <-ctx.Done():
					return
				case idx, ok := <-jobsCh:
					if !ok {
						return
					}

					res := fn(ctx, jobs[idx])
					resultsCh <- indexedResult[Result]{index: idx, result: res}
				}
			}
		})
	}
}

// sendPoolJobs sends job indices into the channel, respecting context cancellation.
func sendPoolJobs(ctx context.Context, jobsCh chan<- int, count int) {
	for idx := range count {
		select {
		case <-ctx.Done():
			close(jobsCh)

			return
		case jobsCh <- idx:
		}
	}

	close(jobsCh)
}

// collectPoolResults gathers indexed results into an ordered output slice.
func collectPoolResults[Result any](resultsCh <-chan indexedResult[Result], count int) []Result {
	output := make([]Result, count)

	for res := range resultsCh {
		output[res.index] = res.result
	}

	return output
}
