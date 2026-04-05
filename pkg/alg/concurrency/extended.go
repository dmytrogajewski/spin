package concurrency

import (
	"context"
	"fmt"
	"time"
)

// SleepWithBackoff sleeps for an exponentially increasing duration based on
// the attempt number (1s, 2s, 4s, 8s, ...). Returns ctx.Err() wrapped if
// the context is canceled before the sleep completes.
func SleepWithBackoff(ctx context.Context, attempt int, base time.Duration) error {
	shift := uint(0)
	if attempt > 0 {
		shift = uint(attempt)
	}

	backoff := base << shift

	timer := time.NewTimer(backoff)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return fmt.Errorf("backoff sleep canceled: %w", ctx.Err())
	case <-timer.C:
		return nil
	}
}

// Semaphore is a counting semaphore backed by a buffered channel.
// It supports context-aware acquisition and non-blocking release.
type Semaphore struct {
	ch chan struct{}
}

// NewSemaphore creates a semaphore with the given maximum concurrency.
func NewSemaphore(maxConcurrent int) *Semaphore {
	return &Semaphore{ch: make(chan struct{}, maxConcurrent)}
}

// Acquire blocks until a slot is available or ctx is canceled.
// Returns ctx.Err() wrapped if the context is canceled.
func (sem *Semaphore) Acquire(ctx context.Context) error {
	select {
	case sem.ch <- struct{}{}:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("semaphore acquire: %w", ctx.Err())
	}
}

// Release frees a previously acquired slot. Must be called exactly once
// for each successful [Semaphore.Acquire] call.
func (sem *Semaphore) Release() {
	<-sem.ch
}
