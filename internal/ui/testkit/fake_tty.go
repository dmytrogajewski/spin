package testkit

import (
	"sync"

	"github.com/dmytrogajewski/spin/internal/ui/term"
)

// FakeTTY simulates a terminal for testing.
// Tracks raw mode state, dimensions, and resize callbacks.
// Thread-safe implementation.
type FakeTTY struct {
	width, height int
	entered       bool
	exited        bool
	resizeCB      []func(w, h int)
	mu            sync.Mutex
}

// NewFakeTTY creates a new fake TTY with the given dimensions.
func NewFakeTTY(width, height int) *FakeTTY {
	return &FakeTTY{
		width:  width,
		height: height,
	}
}

// Enter simulates entering raw mode.
// Sets entered flag, returns error if already entered.
// Thread-safe.
func (f *FakeTTY) Enter() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.entered {
		return &Error{Op: "Enter", Msg: "already in raw mode"}
	}

	f.entered = true
	f.exited = false
	return nil
}

// Exit simulates exiting raw mode.
// Clears entered flag. Idempotent (safe to call multiple times).
// Thread-safe.
func (f *FakeTTY) Exit() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.entered = false
	f.exited = true
	return nil
}

// Size returns the current terminal dimensions.
// Thread-safe.
func (f *FakeTTY) Size() (int, int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.width, f.height
}

// OnResize registers a callback to be invoked on terminal resize.
// Callbacks are invoked in order of registration.
// Thread-safe.
func (f *FakeTTY) OnResize(cb func(w, h int)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.resizeCB = append(f.resizeCB, cb)
}

// SetSize updates terminal dimensions and triggers resize callbacks.
// Simulates SIGWINCH behavior.
// Thread-safe.
func (f *FakeTTY) SetSize(w, h int) {
	f.mu.Lock()
	f.width = w
	f.height = h
	callbacks := make([]func(int, int), len(f.resizeCB))
	copy(callbacks, f.resizeCB)
	f.mu.Unlock()

	// Invoke callbacks outside lock to avoid deadlock
	for _, cb := range callbacks {
		cb(w, h)
	}
}

// IsEntered returns true if Enter() was called and Exit() has not been called.
// Thread-safe.
func (f *FakeTTY) IsEntered() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.entered
}

// IsExited returns true if Exit() was called.
// Thread-safe.
func (f *FakeTTY) IsExited() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.exited
}

// Reset clears all state (useful for test cleanup).
// Thread-safe.
func (f *FakeTTY) Reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.entered = false
	f.exited = false
	f.resizeCB = nil
}

// Error represents a FakeTTY operation error.
type Error struct {
	Op  string
	Msg string
}

func (e *Error) Error() string {
	return e.Op + ": " + e.Msg
}

// Ensure FakeTTY implements term.TerminalController
var _ term.TerminalController = (*FakeTTY)(nil)
