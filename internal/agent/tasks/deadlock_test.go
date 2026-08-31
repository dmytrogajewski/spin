package tasks

// Journey: specs/journeys/JOURNEY-020-agent-task-registry.md.

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/pkg/alg/concurrency"
)

// TestWait_DoesNotAcquireSemaphore documents the admission contract:
// Wait must run outside the spawn semaphore. If Wait called Acquire while
// every slot was held by a background child, the parent would deadlock.
func TestWait_DoesNotAcquireSemaphore(t *testing.T) {
	t.Parallel()

	sem := concurrency.NewSemaphore(1)
	require.NoError(t, sem.Acquire(t.Context()))

	released := false

	t.Cleanup(func() {
		if !released {
			sem.Release()
		}
	})

	reg := New()
	reg.Register(testTaskID, testSpecExplorer, StateCompleted, nil)

	done := make(chan error, 1)

	go func() {
		_, err := reg.Wait(t.Context(), testTaskID)
		done <- err
	}()

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("Wait deadlocked; it must not acquire the spawn semaphore")
	}

	sem.Release()

	released = true
}
