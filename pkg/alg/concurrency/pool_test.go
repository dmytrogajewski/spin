package concurrency

// Journey: specs/journeys/JOURNEY-R16.md.

import (
	"context"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestEffectiveWorkers(t *testing.T) {
	t.Parallel()

	cpuCount := runtime.NumCPU()

	tests := []struct {
		name      string
		requested int
		jobCount  int
		want      int
	}{
		{name: "zero_uses_cpu_count", requested: 0, jobCount: 100, want: cpuCount},
		{name: "negative_uses_cpu_count", requested: -1, jobCount: 100, want: cpuCount},
		{name: "capped_by_job_count", requested: 10, jobCount: 3, want: 3},
		{name: "exact_match", requested: 5, jobCount: 5, want: 5},
		{name: "fewer_requested_than_jobs", requested: 2, jobCount: 10, want: 2},
		{name: "zero_jobs", requested: 4, jobCount: 0, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := EffectiveWorkers(tt.requested, tt.jobCount)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestWorkerPool(t *testing.T) {
	t.Parallel()

	double := func(_ context.Context, val int) int {
		return val * 2
	}

	t.Run("preserves_input_order", func(t *testing.T) {
		t.Parallel()

		jobs := []int{1, 2, 3, 4, 5}

		results := WorkerPool(context.Background(), 3, jobs, double)

		require.Equal(t, []int{2, 4, 6, 8, 10}, results)
	})

	t.Run("empty_jobs", func(t *testing.T) {
		t.Parallel()

		results := WorkerPool(context.Background(), 2, []int{}, double)

		require.Empty(t, results)
	})

	t.Run("nil_jobs", func(t *testing.T) {
		t.Parallel()

		results := WorkerPool[int, int](context.Background(), 2, nil, double)

		require.Empty(t, results)
	})

	t.Run("single_worker", func(t *testing.T) {
		t.Parallel()

		jobs := []int{10, 20, 30}

		results := WorkerPool(context.Background(), 1, jobs, double)

		require.Equal(t, []int{20, 40, 60}, results)
	})

	t.Run("context_cancellation_stops_work", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())

		var started atomic.Int32

		slowFn := func(ctx context.Context, val int) int {
			started.Add(1)

			select {
			case <-ctx.Done():
				return -1
			case <-time.After(time.Second):
				return val
			}
		}

		jobs := make([]int, 100)
		for idx := range jobs {
			jobs[idx] = idx
		}

		done := make(chan []int, 1)

		go func() {
			done <- WorkerPool(ctx, 2, jobs, slowFn)
		}()

		// Wait for some workers to start, then cancel.
		time.Sleep(50 * time.Millisecond)
		cancel()

		select {
		case results := <-done:
			// Should have results array of correct length (some may be -1 from cancellation).
			require.Len(t, results, len(jobs))
		case <-time.After(5 * time.Second):
			t.Fatal("WorkerPool did not return after context cancellation")
		}
	})
}
