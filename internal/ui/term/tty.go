package term

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"golang.org/x/term"
)

// Terminal size constraints (Phase 8.2: Defensive error handling).
const (
	MinTerminalWidth  = 40
	MinTerminalHeight = 10
	MaxTerminalWidth  = 1000
	MaxTerminalHeight = 1000
)

// Errors (Phase 8.2).
var (
	ErrNotATTY          = errors.New("not a terminal")
	ErrTerminalTooSmall = errors.New("terminal too small (minimum 40x10)")
	ErrInfdIsNotATerminal = errors.New("inFD  is not a terminal")
	ErrAlreadyInRawMode = errors.New("already in raw mode")
)

// TTY manages terminal state for raw mode interaction.
// It handles entering/exiting raw mode, window size detection,
// and resize event callbacks without using alt-screen buffer.
type TTY struct {
	inFD      int
	outFD     int
	origState *term.State
	mu        sync.RWMutex
	width     int
	height    int
	onResize  []func(int, int)
	entered   bool
	sigCh     chan os.Signal
}

// New creates a TTY from file descriptors.
// inFD is typically os.Stdin.Fd(), outFD is os.Stdout.Fd().
// Returns error if the input FD is not a terminal.
func New(inFD, outFD int) (*TTY, error) {
	if !isTerminal(inFD) {
return nil, fmt.Errorf("inFD %d is not a terminal: %w", inFD, ErrInfdIsNotATerminal)
	}

	tty := &TTY{
		inFD:  inFD,
		outFD: outFD,
	}

	// Read initial size.
	err := tty.updateSize()
	if err != nil {
		return nil, fmt.Errorf("failed to get terminal size: %w", err)
	}

	// Start SIGWINCH handler.
	tty.startSigwinchHandler()

	return tty, nil
}

// Enter enables raw mode and hides cursor.
// Raw mode disables line buffering, echo, and signals.
// Returns error if already in raw mode or if terminal state cannot be saved.
func (tty *TTY) Enter() error {
	tty.mu.Lock()
	defer tty.mu.Unlock()

	if tty.entered {
		return ErrAlreadyInRawMode
	}

	// Save original terminal state.
	state, err := term.MakeRaw(tty.inFD)
	if err != nil {
		return fmt.Errorf("failed to enter raw mode: %w", err)
	}

	tty.origState = state
	tty.entered = true

	// Hide cursor.
	fmt.Fprint(os.Stdout, HideCursor)

	return nil
}

// Exit restores terminal state and shows cursor.
// Safe to call multiple times (idempotent).
// Should be called via defer to ensure cleanup on panic.
func (tty *TTY) Exit() error {
	tty.mu.Lock()
	defer tty.mu.Unlock()

	if !tty.entered || tty.origState == nil {
		return nil // Already exited or never entered.
	}

	// Show cursor first.
	fmt.Fprint(os.Stdout, ShowCursor)

	// Restore terminal state.
	err := term.Restore(tty.inFD, tty.origState)
	if err != nil {
		return fmt.Errorf("failed to restore terminal: %w", err)
	}

	tty.entered = false
	tty.origState = nil

	// Stop signal handler goroutine
	// Must acquire lock before accessing sigCh.
	ch := tty.sigCh
	if ch != nil {
		signal.Stop(ch)

		tty.sigCh = nil // Set to nil first, goroutine will see this and exit.

		close(ch) // Then close channel.
	}

	return nil
}

// Size returns cached terminal dimensions (width, height).
// This is an O(1) operation reading from cache updated by SIGWINCH.
func (tty *TTY) Size() (width, height int) {
	tty.mu.RLock()
	defer tty.mu.RUnlock()

	return tty.width, tty.height
}

// OnResize registers a callback for window size changes.
// Callback is invoked synchronously when SIGWINCH is received.
// Multiple callbacks can be registered.
func (tty *TTY) OnResize(cb func(int, int)) {
	tty.mu.Lock()
	defer tty.mu.Unlock()

	tty.onResize = append(tty.onResize, cb)
}

// updateSize reads current terminal dimensions and updates cache.
// Called internally on SIGWINCH and during initialization.
func (tty *TTY) updateSize() error {
	w, h, err := term.GetSize(tty.outFD)
	if err != nil {
		return fmt.Errorf("get terminal size: %w", err)
	}

	tty.mu.Lock()
	tty.width, tty.height = w, h
	tty.mu.Unlock()

	return nil
}

// startSigwinchHandler installs a SIGWINCH signal handler to detect
// terminal resize events and invoke registered callbacks.
func (tty *TTY) startSigwinchHandler() {
	tty.setupSignalChannel()
	signal.Notify(tty.sigCh, syscall.SIGWINCH)

	go tty.handleSigwinchLoop()
}

// setupSignalChannel initializes the signal channel.
func (tty *TTY) setupSignalChannel() {
	tty.mu.Lock()
	tty.sigCh = make(chan os.Signal, 1)
	tty.mu.Unlock()
}

// handleSigwinchLoop handles SIGWINCH signals in a loop.
func (tty *TTY) handleSigwinchLoop() {
	for {
		if !tty.waitForSignal() {
			return // Channel closed or nil.
		}

		if !tty.updateSizeAndNotify() {
			continue // Error updating size, try again.
		}
	}
}

// waitForSignal waits for a SIGWINCH signal.
func (tty *TTY) waitForSignal() bool {
	tty.mu.RLock()
	ch := tty.sigCh
	tty.mu.RUnlock()

	if ch == nil {
		return false // Channel was closed, exit goroutine.
	}

	sig, ok := <-ch
	if !ok {
		return false // Channel closed.
	}

	_ = sig // Ignore signal value.

	return true
}

// updateSizeAndNotify updates terminal size and notifies callbacks.
func (tty *TTY) updateSizeAndNotify() bool {
	err := tty.updateSize()
	if err != nil {
		// Log error but continue (terminal might be temporarily unavailable).
		return false
	}

	tty.notifyResizeCallbacks()

	return true
}

// notifyResizeCallbacks invokes all resize callbacks with current dimensions.
func (tty *TTY) notifyResizeCallbacks() {
	tty.mu.RLock()
	cbs := make([]func(int, int), len(tty.onResize))
	copy(cbs, tty.onResize)
	w, h := tty.width, tty.height
	tty.mu.RUnlock()

	for _, cb := range cbs {
		cb(w, h)
	}
}

// isTerminal returns true if fd refers to a terminal.
func isTerminal(fd int) bool {
	return term.IsTerminal(fd)
}

// ValidateTerminalType validates that the TERM environment variable is set
// to a supported terminal type. Returns true if the terminal type warrants a warning.

// ValidateWindowSize validates that the terminal window size meets requirements.
// Returns the validated (potentially clamped) dimensions and an error if size is invalid.
// Clamps dimensions that exceed maximum to MaxTerminalWidth/MaxTerminalHeight.
// Returns error if dimensions are below minimum.
