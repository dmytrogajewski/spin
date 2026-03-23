// Package concurrency provides generic, zero-dependency concurrency primitives
// for context-guarded channel operations, draining, sleeping, and
// blocking-call-with-cancellation.
package concurrency

import (
	"context"
	"fmt"
	"time"
)

// TrySend attempts a non-blocking send of val to ch.
// Returns true if the value was sent, false if the channel is full.
func TrySend[Elem any](ch chan<- Elem, val Elem) bool {
	select {
	case ch <- val:
		return true
	default:
		return false
	}
}

// SendOrCancel blocks until val is sent to ch or ctx is canceled.
// Returns true if sent, false if the context was canceled.
func SendOrCancel[Elem any](ctx context.Context, ch chan<- Elem, val Elem) bool {
	select {
	case ch <- val:
		return true
	case <-ctx.Done():
		return false
	}
}

// SendWithTimeout blocks until val is sent to ch or the timeout expires.
// Returns true if sent, false on timeout.
func SendWithTimeout[Elem any](ch chan<- Elem, val Elem, timeout time.Duration) bool {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case ch <- val:
		return true
	case <-timer.C:
		return false
	}
}

// DrainChannel reads from ch until it is closed or ctx is canceled.
// Each received value is passed to handle. Returns the first non-nil
// error from handle, or a wrapped ctx.Err() if the context is canceled.
func DrainChannel[Elem any](ctx context.Context, ch <-chan Elem, handle func(Elem) error) error {
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("drain channel: %w", ctx.Err())
		case val, ok := <-ch:
			if !ok {
				return nil
			}

			if err := handle(val); err != nil {
				return err
			}
		}
	}
}

// SleepCtx sleeps for the given duration or until ctx is canceled.
// Returns true if the full duration elapsed, false if canceled early.
func SleepCtx(ctx context.Context, dur time.Duration) bool {
	timer := time.NewTimer(dur)
	defer timer.Stop()

	select {
	case <-timer.C:
		return true
	case <-ctx.Done():
		return false
	}
}

// CallWithContext runs fn in a goroutine and waits for it to complete
// or for ctx to be canceled. Returns (result, true) if fn completes,
// or (zero, false) if the context is canceled first.
// Note: the goroutine running fn is not killed on cancellation — it
// will complete in the background.
func CallWithContext[Result any](ctx context.Context, fn func() Result) (Result, bool) {
	ch := make(chan Result, 1)

	go func() {
		ch <- fn()
	}()

	select {
	case result := <-ch:
		return result, true
	case <-ctx.Done():
		var zero Result

		return zero, false
	}
}
