package term

import (
	"os"
	"sync"
	"testing"
	"time"
)

// TestNewWithNonTerminal verifies error when FD is not a terminal.
func TestNewWithNonTerminal(t *testing.T) {
	// Use a regular file FD (guaranteed not terminal)
	tmpFile, err := os.CreateTemp("", "tty_test")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	_, err = New(int(tmpFile.Fd()), int(tmpFile.Fd()))
	if err == nil {
		t.Error("New() with non-terminal FD should return error")
	}
}

// TestNew creates TTY with valid file descriptors.
func TestNew(t *testing.T) {
	if !isTerminal(int(os.Stdin.Fd())) {
		t.Skip("stdin is not a terminal, skipping New test")
	}

	// Use stdin/stdout as valid FDs
	tty, err := New(int(os.Stdin.Fd()), int(os.Stdout.Fd()))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if tty == nil {
		t.Fatal("New() returned nil TTY")
	}
}

// TestSize verifies cached dimensions are returned.
func TestSize(t *testing.T) {
	tty, err := New(int(os.Stdin.Fd()), int(os.Stdout.Fd()))
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
	tty, err := New(int(os.Stdin.Fd()), int(os.Stdout.Fd()))
	if err != nil {
		t.Skipf("New() error = %v, skipping (not a terminal?)", err)
	}

	var called bool
	var mu sync.Mutex
	var gotW, gotH int

	tty.OnResize(func(w, h int) {
		mu.Lock()
		defer mu.Unlock()
		called = true
		gotW, gotH = w, h
	})

	// Simulate resize by calling updateSize and invoking callbacks
	// In real scenario, SIGWINCH triggers this
	if err := tty.updateSize(); err != nil {
		t.Skipf("updateSize() error = %v, skipping (not a terminal?)", err)
	}

	// Manually invoke callbacks to test registration
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
	tty, err := New(int(os.Stdin.Fd()), int(os.Stdout.Fd()))
	if err != nil {
		t.Skipf("New() error = %v, skipping (not a terminal?)", err)
	}

	// Exit without Enter should be safe (no-op or return early)
	if err := tty.Exit(); err != nil {
		t.Errorf("Exit() without Enter error = %v", err)
	}
}

// TestEnterExit verifies state transitions.
func TestEnterExit(t *testing.T) {
	if !isTerminal(int(os.Stdin.Fd())) {
		t.Skip("stdin is not a terminal, skipping Enter/Exit test")
	}

	tty, err := New(int(os.Stdin.Fd()), int(os.Stdout.Fd()))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if err := tty.Enter(); err != nil {
		t.Fatalf("Enter() error = %v", err)
	}

	if tty.origState == nil {
		t.Error("Enter() did not save origState")
	}

	if err := tty.Exit(); err != nil {
		t.Errorf("Exit() error = %v", err)
	}
}

// TestEnterTwice ensures calling Enter twice returns error or is idempotent.
func TestEnterTwice(t *testing.T) {
	if !isTerminal(int(os.Stdin.Fd())) {
		t.Skip("stdin is not a terminal, skipping Enter test")
	}

	tty, err := New(int(os.Stdin.Fd()), int(os.Stdout.Fd()))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if err := tty.Enter(); err != nil {
		t.Fatalf("Enter() error = %v", err)
	}
	defer tty.Exit()

	// Second Enter should error or no-op
	if err := tty.Enter(); err == nil {
		t.Error("Enter() called twice did not return error")
	}
}

// TestPanicCleanup verifies defer Exit works in panic scenario.
func TestPanicCleanup(t *testing.T) {
	if !isTerminal(int(os.Stdin.Fd())) {
		t.Skip("stdin is not a terminal, skipping panic test")
	}

	tty, err := New(int(os.Stdin.Fd()), int(os.Stdout.Fd()))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("did not panic")
			}
		}()
		defer tty.Exit()

		if err := tty.Enter(); err != nil {
			t.Fatalf("Enter() error = %v", err)
		}

		panic("test panic")
	}()

	// If we get here, Exit was called during panic unwinding
}

// TestConcurrentResize ensures no race between resize and Size().
func TestConcurrentResize(t *testing.T) {
	tty, err := New(int(os.Stdin.Fd()), int(os.Stdout.Fd()))
	if err != nil {
		t.Skipf("New() error = %v, skipping (not a terminal?)", err)
	}

	var wg sync.WaitGroup
	wg.Add(2)

	// Goroutine 1: Read size repeatedly
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			_, _ = tty.Size()
		}
	}()

	// Goroutine 2: Simulate resize updates
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			_ = tty.updateSize()
			time.Sleep(1 * time.Millisecond)
		}
	}()

	wg.Wait()
}

// ============================================================================
// Phase 8.2: Defensive Error Handling Tests
// ============================================================================

// TestIsTTY verifies exported IsTTY function
func TestIsTTY(t *testing.T) {
	tests := []struct {
		name     string
		fd       int
		expected bool
	}{
		{
			name:     "stdin is TTY (in CI: false)",
			fd:       int(os.Stdin.Fd()),
			expected: isTerminal(int(os.Stdin.Fd())), // Use actual result
		},
		{
			name:     "regular file is not TTY",
			fd:       -1, // Will use temp file
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fd := tt.fd
			if fd == -1 {
				// Create temp file
				tmpFile, err := os.CreateTemp("", "tty_test")
				if err != nil {
					t.Fatalf("failed to create temp file: %v", err)
				}
				defer os.Remove(tmpFile.Name())
				defer tmpFile.Close()
				fd = int(tmpFile.Fd())
			}

			result := IsTTY(fd)
			if result != tt.expected {
				t.Errorf("IsTTY(%d) = %v, want %v", fd, result, tt.expected)
			}
		})
	}
}

// TestValidateTerminalType verifies TERM environment variable validation
func TestValidateTerminalType(t *testing.T) {
	tests := []struct {
		name     string
		term     string
		wantWarn bool
	}{
		{
			name:     "xterm is valid",
			term:     "xterm",
			wantWarn: false,
		},
		{
			name:     "xterm-256color is valid",
			term:     "xterm-256color",
			wantWarn: false,
		},
		{
			name:     "dumb terminal warns",
			term:     "dumb",
			wantWarn: true,
		},
		{
			name:     "empty TERM warns",
			term:     "",
			wantWarn: true,
		},
		{
			name:     "unknown terminal warns",
			term:     "very-unknown-terminal-xyz",
			wantWarn: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldTerm := os.Getenv("TERM")
			defer os.Setenv("TERM", oldTerm)

			os.Setenv("TERM", tt.term)

			warn := ValidateTerminalType()
			if warn != tt.wantWarn {
				t.Errorf("ValidateTerminalType() warn = %v, want %v (TERM=%s)", warn, tt.wantWarn, tt.term)
			}
		})
	}
}

// TestValidateWindowSize verifies window size validation
func TestValidateWindowSize(t *testing.T) {
	tests := []struct {
		name      string
		width     int
		height    int
		wantErr   bool
		wantClamp bool
	}{
		{
			name:      "normal size (80x24)",
			width:     80,
			height:    24,
			wantErr:   false,
			wantClamp: false,
		},
		{
			name:      "minimum size (40x10)",
			width:     40,
			height:    10,
			wantErr:   false,
			wantClamp: false,
		},
		{
			name:      "below minimum width",
			width:     30,
			height:    24,
			wantErr:   true,
			wantClamp: false,
		},
		{
			name:      "below minimum height",
			width:     80,
			height:    5,
			wantErr:   true,
			wantClamp: false,
		},
		{
			name:      "above maximum (needs clamp)",
			width:     1500,
			height:    800,
			wantErr:   false,
			wantClamp: true,
		},
		{
			name:      "negative width",
			width:     -10,
			height:    24,
			wantErr:   true,
			wantClamp: false,
		},
		{
			name:      "zero dimensions",
			width:     0,
			height:    0,
			wantErr:   true,
			wantClamp: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, h, err := ValidateWindowSize(tt.width, tt.height)

			if tt.wantErr && err == nil {
				t.Errorf("ValidateWindowSize(%d, %d) expected error, got nil", tt.width, tt.height)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("ValidateWindowSize(%d, %d) unexpected error: %v", tt.width, tt.height, err)
			}

			if tt.wantClamp {
				// Should clamp to max dimensions
				if w > MaxTerminalWidth || h > MaxTerminalHeight {
					t.Errorf("ValidateWindowSize(%d, %d) = (%d, %d), expected clamping to max (%d, %d)",
						tt.width, tt.height, w, h, MaxTerminalWidth, MaxTerminalHeight)
				}
			}

			if !tt.wantErr && !tt.wantClamp {
				// Should return unchanged dimensions
				if w != tt.width || h != tt.height {
					t.Errorf("ValidateWindowSize(%d, %d) = (%d, %d), want (%d, %d)",
						tt.width, tt.height, w, h, tt.width, tt.height)
				}
			}
		})
	}
}

// TestErrNotATTY verifies error type for non-TTY
func TestErrNotATTY(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "tty_test")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	_, err = New(int(tmpFile.Fd()), int(tmpFile.Fd()))
	if err == nil {
		t.Fatal("New() with non-TTY should return error")
	}

	// Verify error message contains expected text
	if err.Error() != "not a terminal" && !contains(err.Error(), "not a terminal") {
		t.Errorf("New() error = %q, want error containing 'not a terminal'", err.Error())
	}
}

// contains is a helper to check if string contains substring
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
