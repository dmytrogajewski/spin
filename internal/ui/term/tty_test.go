package term

import (
	"os"
	"sync"
	"testing"
	"time"
)

// TestNewWithNonTerminal verifies error when FD is not a terminal.
func TestNewWithNonTerminal(t *testing.T) {
	t.Parallel(
	// Use a regular file FD (guaranteed not terminal).
	)

	tmpFile, err := os.CreateTemp(t.TempDir(), "tty_test")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer tmpFile.Close()

	_, err = New(SafeFd(tmpFile.Fd()), SafeFd(tmpFile.Fd()))
	if err == nil {
		t.Error("New() with non-terminal FD should return error")
	}
}

// TestNew creates TTY with valid file descriptors.
func TestNew(t *testing.T) {
	t.Parallel()
	if !isTerminal(SafeFd(os.Stdin.Fd())) {
		t.Skip("stdin is not a terminal, skipping New test")
	}

	// Use stdin/stdout as valid FDs.
	tty, err := New(SafeFd(os.Stdin.Fd()), SafeFd(os.Stdout.Fd()))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if tty == nil {
		t.Fatal("New() returned nil TTY")
	}
}

// TestSize verifies cached dimensions are returned.
func TestSize(t *testing.T) {
	t.Parallel()
	tty, err := New(SafeFd(os.Stdin.Fd()), SafeFd(os.Stdout.Fd()))
	if err != nil {
		t.Skipf("New() error = %v, skipping (not a terminal?)", err)
	}

	w, h := tty.Size()
	if w <= 0 || h <= 0 {
		t.Errorf("Size() = (%d, %d), want positive dimensions", w, h)
	}
}

// TestOnResize registers callback and verifies it can be called.
func TestOnResize(t *testing.T) {
	t.Parallel()
	tty, err := New(SafeFd(os.Stdin.Fd()), SafeFd(os.Stdout.Fd()))
	if err != nil {
		t.Skipf("New() error = %v, skipping (not a terminal?)", err)
	}

	var (
		called     bool
		mu         sync.Mutex
		gotW, gotH int
	)

	tty.OnResize(func(w, h int) {
		mu.Lock()
		defer mu.Unlock()

		called = true
		gotW, gotH = w, h
	})

	// Simulate resize by calling updateSize and invoking callbacks
	// In real scenario, SIGWINCH triggers this.
	err = tty.updateSize()
	if err != nil {
		t.Skipf("updateSize() error = %v, skipping (not a terminal?)", err)
	}

	// Manually invoke callbacks to test registration.
	tty.mu.RLock()
	cbs := tty.onResize
	w, h := tty.width, tty.height
	tty.mu.RUnlock()

	for _, cb := range cbs {
		cb(w, h)
	}

	mu.Lock()
	defer mu.Unlock()

	if !called {
		t.Error("OnResize callback was not invoked")
	}

	if gotW <= 0 || gotH <= 0 {
		t.Errorf("callback received (%d, %d), want positive dimensions", gotW, gotH)
	}
}

// TestExitWithoutEnter ensures Exit is idempotent and safe.
func TestExitWithoutEnter(t *testing.T) {
	t.Parallel()
	tty, err := New(SafeFd(os.Stdin.Fd()), SafeFd(os.Stdout.Fd()))
	if err != nil {
		t.Skipf("New() error = %v, skipping (not a terminal?)", err)
	}

	// Exit without Enter should be safe (no-op or return early).
	err = tty.Exit()
	if err != nil {
		t.Errorf("Exit() without Enter error = %v", err)
	}
}

// TestEnterExit verifies state transitions.
func TestEnterExit(t *testing.T) {
	t.Parallel()
	if !isTerminal(SafeFd(os.Stdin.Fd())) {
		t.Skip("stdin is not a terminal, skipping Enter/Exit test")
	}

	tty, err := New(SafeFd(os.Stdin.Fd()), SafeFd(os.Stdout.Fd()))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	err = tty.Enter()
	if err != nil {
		t.Fatalf("Enter() error = %v", err)
	}

	if tty.origState == nil {
		t.Error("Enter() did not save origState")
	}

	err = tty.Exit()
	if err != nil {
		t.Errorf("Exit() error = %v", err)
	}
}

// TestEnterTwice ensures calling Enter twice returns error or is idempotent.
func TestEnterTwice(t *testing.T) {
	t.Parallel()
	if !isTerminal(SafeFd(os.Stdin.Fd())) {
		t.Skip("stdin is not a terminal, skipping Enter test")
	}

	tty, err := New(SafeFd(os.Stdin.Fd()), SafeFd(os.Stdout.Fd()))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	err = tty.Enter()
	if err != nil {
		t.Fatalf("Enter() error = %v", err)
	}
	defer func() { _ = tty.Exit() }()

	// Second Enter should error or no-op.
	err = tty.Enter()
	if err == nil {
		t.Error("Enter() called twice did not return error")
	}
}

// TestPanicCleanup verifies defer Exit works in panic scenario.
func TestPanicCleanup(t *testing.T) {
	t.Parallel()
	if !isTerminal(SafeFd(os.Stdin.Fd())) {
		t.Skip("stdin is not a terminal, skipping panic test")
	}

	tty, err := New(SafeFd(os.Stdin.Fd()), SafeFd(os.Stdout.Fd()))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("did not panic")
			}
		}()
		defer func() { _ = tty.Exit() }()

		enterErr := tty.Enter()
		if enterErr != nil {
			t.Fatalf("Enter() error = %v", enterErr)
		}

		panic("test panic")
	}()

	// If we get here, Exit was called during panic unwinding.
}

// TestConcurrentResize ensures no race between resize and Size().
func TestConcurrentResize(t *testing.T) {
	t.Parallel()
	tty, err := New(SafeFd(os.Stdin.Fd()), SafeFd(os.Stdout.Fd()))
	if err != nil {
		t.Skipf("New() error = %v, skipping (not a terminal?)", err)
	}

	var wg sync.WaitGroup
	wg.Add(2)

	// Goroutine 1: Read size repeatedly.
	go func() {
		defer wg.Done()

		for range 100 {
			_, _ = tty.Size()
		}
	}()

	// Goroutine 2: Simulate resize updates.
	go func() {
		defer wg.Done()

		for range 100 {
			_ = tty.updateSize()

			time.Sleep(1 * time.Millisecond)
		}
	}()

	wg.Wait()
}

// ============================================================================
// Phase 8.2: Defensive Error Handling Tests
// ============================================================================.

// TestIsTTY verifies exported IsTTY function.

// TestValidateTerminalType verifies TERM environment variable validation.

// TestValidateWindowSize verifies window size validation.

// TestErrNotATTY verifies error type for non-TTY.
func TestErrNotATTY(t *testing.T) {
	t.Parallel()
	tmpFile, err := os.CreateTemp(t.TempDir(), "tty_test")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer tmpFile.Close()

	_, err = New(SafeFd(tmpFile.Fd()), SafeFd(tmpFile.Fd()))
	if err == nil {
		t.Fatal("New() with non-TTY should return error")
	}

	// Verify error message contains expected text.
	if err.Error() != "not a terminal" && !contains(err.Error(), "not a terminal") {
		t.Errorf("New() error = %q, want error containing 'not a terminal'", err.Error())
	}
}

// contains is a helper to check if string contains substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && hasSubstring(s, substr))
}

func hasSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}

	return false
}
