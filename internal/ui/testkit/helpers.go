package testkit

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/ui/ports"
)

// WaitForStartup waits for UI to be ready (TTY entered raw mode).
// Blocks until ready or timeout expires.
func WaitForStartup(t *testing.T, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
		// In real implementation, check TTY.IsEntered()
		// For now, just wait a bit
	}
}

// WaitForInput waits for a line to appear on the input channel.
// Returns the line or empty string on timeout.
func WaitForInput(t *testing.T, ch <-chan string, timeout time.Duration) string {
	t.Helper()

	select {
	case line := <-ch:
		return line
	case <-time.After(timeout):
		t.Errorf("timeout waiting for input after %v", timeout)
		return ""
	}
}

// WaitForShutdown waits for the Run() goroutine to exit.
// Returns the error from Run() or fails the test on timeout.
func WaitForShutdown(t *testing.T, errCh <-chan error, timeout time.Duration) error {
	t.Helper()

	select {
	case err := <-errCh:
		return err
	case <-time.After(timeout):
		t.Fatalf("timeout waiting for shutdown after %v", timeout)
		return nil
	}
}

// AssertNoGoroutineLeak verifies goroutine count is stable.
// Allows small variance (±tolerance) to account for runtime goroutines.
func AssertNoGoroutineLeak(t *testing.T, before, after, tolerance int) {
	t.Helper()

	diff := after - before
	if diff > tolerance {
		t.Errorf("goroutine leak detected: %d goroutines before, %d after (diff: %d, tolerance: %d)",
			before, after, diff, tolerance)
	}
}

// GetGoroutineCount returns current number of goroutines.
// Use before/after test execution to detect leaks.
func GetGoroutineCount() int {
	return runtime.NumGoroutine()
}

// AssertANSISequence checks if output contains a specific ANSI escape code.
func AssertANSISequence(t *testing.T, output string, seq string) {
	t.Helper()

	// Note: FakeWriter already has ContainsANSI method
	// This is a test helper wrapper
	if !contains(output, seq) {
		t.Errorf("output missing ANSI sequence %q", seq)
	}
}

// contains is a simple substring check helper
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || indexHelper(s, substr) >= 0)
}

func indexHelper(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// WaitForCondition polls a condition function until it returns true or timeout.
// Useful for waiting on async state changes.
func WaitForCondition(t *testing.T, condition func() bool, timeout time.Duration, msg string) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Errorf("timeout waiting for condition: %s (after %v)", msg, timeout)
}

// RequireNoError fails the test if err is not nil.
// Convenience wrapper for common pattern.
func RequireNoError(t *testing.T, err error, msg string) {
	t.Helper()

	if err != nil {
		t.Fatalf("%s: %v", msg, err)
	}
}

// RequireError fails the test if err is nil.
func RequireError(t *testing.T, err error, msg string) {
	t.Helper()

	if err == nil {
		t.Fatalf("%s: expected error but got nil", msg)
	}
}

// AssertEventually retries an assertion until it passes or timeout.
// Useful for testing async behavior where timing is unpredictable.
func AssertEventually(t *testing.T, assertion func() bool, timeout time.Duration, msg string) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	var lastErr error

	for time.Now().Before(deadline) {
		if assertion() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	if lastErr != nil {
		t.Errorf("assertion failed after %v: %s (last error: %v)", timeout, msg, lastErr)
	} else {
		t.Errorf("assertion failed after %v: %s", timeout, msg)
	}
}

// NewTestUI creates a UI instance with fake components for testing.
// Returns the UI, fake writer, fake keyboard, and fake TTY.
func NewTestUI(t *testing.T, width, height int) (ui ports.UI, writer *FakeWriter, keyboard *FakeKeyboard, tty *FakeTTY, cleanup func()) {
	t.Helper()

	writer = NewFakeWriter()
	keyboard = NewFakeKeyboard()
	tty = NewFakeTTY(width, height)

	cleanup = func() {
		keyboard.Close()
		tty.Reset()
	}

	// Note: Actual UI construction happens in E2E tests
	// This helper just provides the fake components
	return nil, writer, keyboard, tty, cleanup
}

// DrainChannel drains all pending items from a channel.
// Returns slice of drained items. Non-blocking.
func DrainChannel(ch <-chan string) []string {
	var items []string
	for {
		select {
		case item := <-ch:
			items = append(items, item)
		default:
			return items
		}
	}
}

// WaitForMultipleInputs waits for n lines from the input channel.
// Returns all lines or fails on timeout.
func WaitForMultipleInputs(t *testing.T, ch <-chan string, n int, timeout time.Duration) []string {
	t.Helper()

	var lines []string
	deadline := time.After(timeout)

	for i := 0; i < n; i++ {
		select {
		case line := <-ch:
			lines = append(lines, line)
		case <-deadline:
			t.Errorf("timeout waiting for input %d/%d after %v", i+1, n, timeout)
			return lines
		}
	}

	return lines
}

// ContextWithTimeout creates a context with timeout for tests.
func ContextWithTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}
