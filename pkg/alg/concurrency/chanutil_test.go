package concurrency

// Journey: specs/journeys/JOURNEY-R4.md.

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// testTimeout is the maximum time any individual test should take.
const testTimeout = 2 * time.Second

// shortDelay is a short delay used in tests to ensure ordering.
const shortDelay = 50 * time.Millisecond

// errBoom is a sentinel error for testing handler error propagation.
var errBoom = errors.New("boom")

func TestTrySend(t *testing.T) {
	t.Parallel()

	t.Run("full_channel_returns_false", func(t *testing.T) {
		t.Parallel()

		ch := make(chan int, 1)
		ch <- 42 // Fill the channel.

		got := TrySend(ch, 99)
		require.False(t, got)
	})

	t.Run("buffered_channel_returns_true", func(t *testing.T) {
		t.Parallel()

		ch := make(chan int, 1)

		got := TrySend(ch, 42)
		require.True(t, got)
		require.Equal(t, 42, <-ch)
	})

	t.Run("unbuffered_no_receiver_returns_false", func(t *testing.T) {
		t.Parallel()

		ch := make(chan int)

		got := TrySend(ch, 1)
		require.False(t, got)
	})
}

func TestSendOrCancel(t *testing.T) {
	t.Parallel()

	t.Run("canceled_context_returns_false", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		ch := make(chan int) // Unbuffered, no receiver.

		got := SendOrCancel(ctx, ch, 42)
		require.False(t, got)
	})

	t.Run("successful_send_returns_true", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		ch := make(chan int, 1)

		got := SendOrCancel(ctx, ch, 42)
		require.True(t, got)
		require.Equal(t, 42, <-ch)
	})

	t.Run("blocks_until_cancel", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		ch := make(chan int) // Unbuffered, no receiver.
		done := make(chan bool, 1)

		go func() {
			done <- SendOrCancel(ctx, ch, 1)
		}()

		// Cancel after short delay.
		time.Sleep(shortDelay)
		cancel()

		select {
		case result := <-done:
			require.False(t, result)
		case <-time.After(testTimeout):
			t.Fatal("timed out waiting for SendOrCancel to return")
		}
	})
}

func TestSendWithTimeout(t *testing.T) {
	t.Parallel()

	t.Run("expired_timeout_returns_false", func(t *testing.T) {
		t.Parallel()

		ch := make(chan int) // Unbuffered, no receiver.

		got := SendWithTimeout(ch, 42, time.Millisecond)
		require.False(t, got)
	})

	t.Run("immediate_space_returns_true", func(t *testing.T) {
		t.Parallel()

		ch := make(chan int, 1)

		got := SendWithTimeout(ch, 42, testTimeout)
		require.True(t, got)
		require.Equal(t, 42, <-ch)
	})
}

func TestDrainChannel(t *testing.T) {
	t.Parallel()

	t.Run("channel_close_terminates", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		ch := make(chan int, 3)

		for _, val := range []int{1, 2, 3} {
			ch <- val
		}

		close(ch)

		var collected []int

		err := DrainChannel(ctx, ch, func(val int) error {
			collected = append(collected, val)

			return nil
		})

		require.NoError(t, err)
		require.Equal(t, []int{1, 2, 3}, collected)
	})

	t.Run("context_cancel_terminates", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		ch := make(chan int) // Will never close.
		done := make(chan error, 1)

		go func() {
			done <- DrainChannel(ctx, ch, func(_ int) error {
				return nil
			})
		}()

		time.Sleep(shortDelay)
		cancel()

		select {
		case err := <-done:
			require.ErrorIs(t, err, context.Canceled)
		case <-time.After(testTimeout):
			t.Fatal("timed out waiting for DrainChannel to return")
		}
	})

	t.Run("handler_error_propagates", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		ch := make(chan int, 1)
		ch <- 1

		err := DrainChannel(ctx, ch, func(_ int) error {
			return errBoom
		})

		require.ErrorIs(t, err, errBoom)
	})
}

func TestSleepCtx(t *testing.T) {
	t.Parallel()

	t.Run("canceled_context_returns_false", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan bool, 1)

		go func() {
			done <- SleepCtx(ctx, testTimeout)
		}()

		time.Sleep(shortDelay)
		cancel()

		select {
		case result := <-done:
			require.False(t, result)
		case <-time.After(testTimeout):
			t.Fatal("timed out waiting for SleepCtx to return")
		}
	})

	t.Run("full_duration_returns_true", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		got := SleepCtx(ctx, time.Millisecond)
		require.True(t, got)
	})

	t.Run("already_canceled_returns_false", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		got := SleepCtx(ctx, time.Hour)
		require.False(t, got)
	})
}

func TestCallWithContext(t *testing.T) {
	t.Parallel()

	t.Run("blocking_fn_completes_returns_true", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		result, ok := CallWithContext(ctx, func() int {
			return 42
		})

		require.True(t, ok)
		require.Equal(t, 42, result)
	})

	t.Run("canceled_context_returns_false", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())

		type callResult struct {
			value int
			ok    bool
		}

		done := make(chan callResult, 1)

		go func() {
			val, ok := CallWithContext(ctx, func() int {
				time.Sleep(testTimeout) // Block longer than cancel.

				return 42
			})
			done <- callResult{value: val, ok: ok}
		}()

		time.Sleep(shortDelay)
		cancel()

		select {
		case cr := <-done:
			require.False(t, cr.ok)
			require.Equal(t, 0, cr.value)
		case <-time.After(testTimeout):
			t.Fatal("timed out waiting for CallWithContext to return")
		}
	})
}
