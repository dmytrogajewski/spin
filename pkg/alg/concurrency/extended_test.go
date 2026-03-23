package concurrency

// Journey: specs/journeys/JOURNEY-S4.md.

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSleepWithBackoff(t *testing.T) {
	t.Parallel()

	t.Run("returns_nil_after_sleep", func(t *testing.T) {
		t.Parallel()

		err := SleepWithBackoff(context.Background(), 0, time.Millisecond)
		require.NoError(t, err)
	})

	t.Run("canceled_context_returns_error", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := SleepWithBackoff(ctx, 0, time.Hour)
		require.ErrorIs(t, err, context.Canceled)
	})

	t.Run("exponential_increase", func(t *testing.T) {
		t.Parallel()

		// Attempt 0: base*1, attempt 1: base*2, attempt 2: base*4.
		start := time.Now()

		err := SleepWithBackoff(context.Background(), 2, time.Millisecond)
		require.NoError(t, err)

		elapsed := time.Since(start)
		// 4ms expected (1ms << 2), allow generous tolerance.
		require.GreaterOrEqual(t, elapsed, 3*time.Millisecond)
	})

	t.Run("negative_attempt_treated_as_zero", func(t *testing.T) {
		t.Parallel()

		err := SleepWithBackoff(context.Background(), -1, time.Millisecond)
		require.NoError(t, err)
	})
}

// semaphoreSize is the capacity for test semaphores.
const semaphoreSize = 2

func TestSemaphore(t *testing.T) {
	t.Parallel()

	t.Run("acquire_and_release", func(t *testing.T) {
		t.Parallel()

		sem := NewSemaphore(semaphoreSize)

		require.NoError(t, sem.Acquire(context.Background()))
		require.NoError(t, sem.Acquire(context.Background()))

		sem.Release()
		sem.Release()
	})

	t.Run("blocks_at_capacity", func(t *testing.T) {
		t.Parallel()

		sem := NewSemaphore(1)
		require.NoError(t, sem.Acquire(context.Background()))

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		err := sem.Acquire(ctx)
		require.ErrorIs(t, err, context.DeadlineExceeded)

		sem.Release()
	})

	t.Run("canceled_context", func(t *testing.T) {
		t.Parallel()

		sem := NewSemaphore(1)
		require.NoError(t, sem.Acquire(context.Background()))

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := sem.Acquire(ctx)
		require.ErrorIs(t, err, context.Canceled)

		sem.Release()
	})

	t.Run("concurrent_access", func(t *testing.T) {
		t.Parallel()

		sem := NewSemaphore(semaphoreSize)
		goroutines := 10

		var active atomic.Int32

		var maxActive atomic.Int32

		var wg sync.WaitGroup

		wg.Add(goroutines)

		for range goroutines {
			go func() {
				defer wg.Done()

				err := sem.Acquire(context.Background())
				assert.NoError(t, err)

				cur := active.Add(1)

				// Track max concurrent.
				for {
					old := maxActive.Load()
					if cur <= old || maxActive.CompareAndSwap(old, cur) {
						break
					}
				}

				time.Sleep(time.Millisecond)

				active.Add(-1)

				sem.Release()
			}()
		}

		wg.Wait()

		// Max concurrent should never exceed semaphore capacity.
		require.LessOrEqual(t, int(maxActive.Load()), semaphoreSize)
	})
}
