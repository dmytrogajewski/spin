package testkit

import (
	"sync"

	"github.com/dmytrogajewski/spin/internal/ui/term"
)

// FakeTTY implements term.TerminalController for testing.
// It simulates terminal dimensions and raw mode without requiring a real TTY.
type FakeTTY struct {
	mu        sync.RWMutex
	width     int
	height    int
	inRawMode bool
	callbacks []func(w, h int)
}

// NewFakeTTY creates a new fake TTY with the given dimensions.
func NewFakeTTY(width, height int) *FakeTTY {
	return &FakeTTY{
		width:  width,
		height: height,
	}
}

// Enter puts the terminal into raw mode (simulated).
func (f *FakeTTY) Enter() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.inRawMode = true
	return nil
}

// Exit restores the terminal from raw mode (simulated).
func (f *FakeTTY) Exit() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.inRawMode = false
	return nil
}

// Size returns the current terminal dimensions.
func (f *FakeTTY) Size() (int, int) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.width, f.height
}

// OnResize registers a callback to be invoked when the terminal is resized.
func (f *FakeTTY) OnResize(cb func(w, h int)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.callbacks = append(f.callbacks, cb)
}

// SetSize simulates a terminal resize and invokes registered callbacks.
func (f *FakeTTY) SetSize(width, height int) {
	f.mu.Lock()
	f.width = width
	f.height = height
	callbacks := make([]func(w, h int), len(f.callbacks))
	copy(callbacks, f.callbacks)
	f.mu.Unlock()

	// Invoke callbacks outside of lock
	for _, cb := range callbacks {
		cb(width, height)
	}
}

// InRawMode returns whether the terminal is in raw mode.
func (f *FakeTTY) InRawMode() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.inRawMode
}

// Ensure FakeTTY implements TerminalController
var _ term.TerminalController = (*FakeTTY)(nil)


