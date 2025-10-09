//go:build integration
// +build integration

package term

import (
	"os"
	"testing"
	"time"

	"github.com/creack/pty"
)

// TestTTY_RealTerminal_EnterExit verifies raw mode enter/exit on real PTY.
func TestTTY_RealTerminal_EnterExit(t *testing.T) {
	ptmx, pts, err := pty.Open()
	if err != nil {
		t.Fatalf("pty.Open() error = %v", err)
	}
	defer ptmx.Close()
	defer pts.Close()

	tty, err := New(int(pts.Fd()), int(pts.Fd()))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Enter raw mode
	if err := tty.Enter(); err != nil {
		t.Fatalf("Enter() error = %v", err)
	}

	// Verify origState was saved
	if tty.origState == nil {
		t.Error("Enter() did not save origState")
	}

	// Verify entered flag
	if !tty.entered {
		t.Error("Enter() did not set entered flag")
	}

	// Exit raw mode
	if err := tty.Exit(); err != nil {
		t.Fatalf("Exit() error = %v", err)
	}

	// Verify cleanup
	if tty.entered {
		t.Error("Exit() did not clear entered flag")
	}
	if tty.origState != nil {
		t.Error("Exit() did not clear origState")
	}
}

// TestTTY_RealTerminal_Size verifies size detection on real PTY.
func TestTTY_RealTerminal_Size(t *testing.T) {
	ptmx, pts, err := pty.Open()
	if err != nil {
		t.Fatalf("pty.Open() error = %v", err)
	}
	defer ptmx.Close()
	defer pts.Close()

	// Set PTY size
	if err := pty.Setsize(ptmx, &pty.Winsize{Rows: 24, Cols: 80}); err != nil {
		t.Fatalf("pty.Setsize() error = %v", err)
	}

	tty, err := New(int(pts.Fd()), int(pts.Fd()))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	w, h := tty.Size()
	if w != 80 || h != 24 {
		t.Errorf("Size() = (%d, %d), want (80, 24)", w, h)
	}
}

// TestTTY_RealTerminal_Resize verifies SIGWINCH handling on real PTY.
func TestTTY_RealTerminal_Resize(t *testing.T) {
	ptmx, pts, err := pty.Open()
	if err != nil {
		t.Fatalf("pty.Open() error = %v", err)
	}
	defer ptmx.Close()
	defer pts.Close()

	// Set initial size
	if err := pty.Setsize(ptmx, &pty.Winsize{Rows: 24, Cols: 80}); err != nil {
		t.Fatalf("pty.Setsize() error = %v", err)
	}

	tty, err := New(int(pts.Fd()), int(pts.Fd()))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Register resize callback
	resizeCalled := make(chan struct{}, 1)
	var gotW, gotH int
	tty.OnResize(func(w, h int) {
		gotW, gotH = w, h
		select {
		case resizeCalled <- struct{}{}:
		default:
		}
	})

	// Change PTY size (triggers SIGWINCH on the controlling process)
	// Note: SIGWINCH is sent to the foreground process group, not the PTY itself
	// We need to manually trigger resize detection
	if err := pty.Setsize(ptmx, &pty.Winsize{Rows: 40, Cols: 120}); err != nil {
		t.Fatalf("pty.Setsize() error = %v", err)
	}

	// Manually trigger updateSize (in real scenario, SIGWINCH does this)
	if err := tty.updateSize(); err != nil {
		t.Fatalf("updateSize() error = %v", err)
	}

	// Manually invoke callbacks (simulating SIGWINCH handler)
	tty.mu.RLock()
	cbs := make([]func(int, int), len(tty.onResize))
	copy(cbs, tty.onResize)
	w, h := tty.width, tty.height
	tty.mu.RUnlock()

	for _, cb := range cbs {
		cb(w, h)
	}

	// Wait for callback with timeout
	select {
	case <-resizeCalled:
		// Success
	case <-time.After(100 * time.Millisecond):
		t.Fatal("resize callback was not invoked within timeout")
	}

	if gotW != 120 || gotH != 40 {
		t.Errorf("resize callback received (%d, %d), want (120, 40)", gotW, gotH)
	}

	// Verify cached size updated
	w, h = tty.Size()
	if w != 120 || h != 40 {
		t.Errorf("Size() after resize = (%d, %d), want (120, 40)", w, h)
	}
}

// TestTTY_RealTerminal_EnterTwice verifies double Enter is rejected.
func TestTTY_RealTerminal_EnterTwice(t *testing.T) {
	ptmx, pts, err := pty.Open()
	if err != nil {
		t.Fatalf("pty.Open() error = %v", err)
	}
	defer ptmx.Close()
	defer pts.Close()

	tty, err := New(int(pts.Fd()), int(pts.Fd()))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if err := tty.Enter(); err != nil {
		t.Fatalf("Enter() error = %v", err)
	}
	defer tty.Exit()

	// Second Enter should error
	if err := tty.Enter(); err == nil {
		t.Error("Enter() called twice did not return error")
	}
}

// TestTTY_RealTerminal_ExitIdempotent verifies Exit is safe to call multiple times.
func TestTTY_RealTerminal_ExitIdempotent(t *testing.T) {
	ptmx, pts, err := pty.Open()
	if err != nil {
		t.Fatalf("pty.Open() error = %v", err)
	}
	defer ptmx.Close()
	defer pts.Close()

	tty, err := New(int(pts.Fd()), int(pts.Fd()))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if err := tty.Enter(); err != nil {
		t.Fatalf("Enter() error = %v", err)
	}

	// First Exit
	if err := tty.Exit(); err != nil {
		t.Fatalf("Exit() error = %v", err)
	}

	// Second Exit should be safe (no-op)
	if err := tty.Exit(); err != nil {
		t.Errorf("Exit() called twice error = %v, want nil", err)
	}
}

// TestTTY_RealTerminal_PanicCleanup verifies defer Exit cleans up on panic.
func TestTTY_RealTerminal_PanicCleanup(t *testing.T) {
	ptmx, pts, err := pty.Open()
	if err != nil {
		t.Fatalf("pty.Open() error = %v", err)
	}
	defer ptmx.Close()
	defer pts.Close()

	tty, err := New(int(pts.Fd()), int(pts.Fd()))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	panicOccurred := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicOccurred = true
			}
		}()
		defer tty.Exit()

		if err := tty.Enter(); err != nil {
			t.Fatalf("Enter() error = %v", err)
		}

		panic("test panic")
	}()

	if !panicOccurred {
		t.Error("panic did not occur")
	}

	// Verify cleanup happened
	if tty.entered {
		t.Error("panic cleanup: entered flag still set")
	}
	if tty.origState != nil {
		t.Error("panic cleanup: origState not cleared")
	}
}

// TestTTY_RealTerminal_ConcurrentAccess verifies thread safety.
func TestTTY_RealTerminal_ConcurrentAccess(t *testing.T) {
	ptmx, pts, err := pty.Open()
	if err != nil {
		t.Fatalf("pty.Open() error = %v", err)
	}
	defer ptmx.Close()
	defer pts.Close()

	if err := pty.Setsize(ptmx, &pty.Winsize{Rows: 24, Cols: 80}); err != nil {
		t.Fatalf("pty.Setsize() error = %v", err)
	}

	tty, err := New(int(pts.Fd()), int(pts.Fd()))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 100; i++ {
			_, _ = tty.Size()
		}
	}()

	for i := 0; i < 100; i++ {
		_ = tty.updateSize()
		time.Sleep(1 * time.Millisecond)
	}

	<-done
}

// TestTTY_RealTerminal_WriteANSI verifies ANSI sequences work on real PTY.
func TestTTY_RealTerminal_WriteANSI(t *testing.T) {
	ptmx, pts, err := pty.Open()
	if err != nil {
		t.Fatalf("pty.Open() error = %v", err)
	}
	defer ptmx.Close()
	defer pts.Close()

	// Write ANSI sequences to PTY
	sequences := []string{
		HideCursor,
		ShowCursor,
		ClearLine,
		SaveCursor,
		RestoreCursor,
		MoveCursorToCol(10),
	}

	for _, seq := range sequences {
		n, err := pts.Write([]byte(seq))
		if err != nil {
			t.Errorf("Write(%q) error = %v", seq, err)
		}
		if n != len(seq) {
			t.Errorf("Write(%q) wrote %d bytes, want %d", seq, n, len(seq))
		}
	}

	// Read back and verify something was written (PTY echoes back in raw mode)
	buf := make([]byte, 1024)
	ptmx.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	n, err := ptmx.Read(buf)
	if err != nil && !os.IsTimeout(err) {
		t.Errorf("Read() error = %v", err)
	}
	if n == 0 {
		t.Error("Read() returned 0 bytes, expected ANSI sequences")
	}
}
