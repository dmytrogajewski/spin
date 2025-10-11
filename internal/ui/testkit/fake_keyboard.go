package testkit

import (
	"sync"

	"github.com/dmytrogajewski/spin/internal/ui/term"
)

// FakeKeyboard injects scripted key events for testing.
// Thread-safe implementation that queues events for consumption.
type FakeKeyboard struct {
	events chan term.KeyEvent
	mu     sync.Mutex
	closed bool
}

// NewFakeKeyboard creates a new fake keyboard with buffered event channel.
func NewFakeKeyboard() *FakeKeyboard {
	return &FakeKeyboard{
		events: make(chan term.KeyEvent, 1000), // Large buffer for test scenarios
	}
}

// Events returns the read-only channel of key events.
// This channel is consumed by term.ReadKeys() or prompt.Loop.
func (f *FakeKeyboard) Events() <-chan term.KeyEvent {
	return f.events
}

// InjectKey queues a single key event.
// Thread-safe. Non-blocking if buffer has space.
func (f *FakeKeyboard) InjectKey(kind term.KeyKind, r rune) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.closed {
		return
	}

	f.events <- term.KeyEvent{Kind: kind, Rune: r}
}

// InjectString queues key events for each rune in the string.
// Each character becomes a KeyRune event.
// Thread-safe.
func (f *FakeKeyboard) InjectString(s string) {
	for _, r := range s {
		f.InjectKey(term.KeyRune, r)
	}
}

// InjectEnter queues a KeyEnter event.
func (f *FakeKeyboard) InjectEnter() {
	f.InjectKey(term.KeyEnter, 0)
}

// InjectBackspace queues a KeyBackspace event.
func (f *FakeKeyboard) InjectBackspace() {
	f.InjectKey(term.KeyBackspace, 0)
}

// InjectDelete queues a KeyDelete event.
func (f *FakeKeyboard) InjectDelete() {
	f.InjectKey(term.KeyDelete, 0)
}

// InjectCtrlC queues a KeyCtrlC event.
func (f *FakeKeyboard) InjectCtrlC() {
	f.InjectKey(term.KeyCtrlC, 0)
}

// InjectCtrlD queues a KeyCtrlD event.
func (f *FakeKeyboard) InjectCtrlD() {
	f.InjectKey(term.KeyCtrlD, 0)
}

// InjectCtrlU queues a KeyCtrlU event (clear line left).
func (f *FakeKeyboard) InjectCtrlU() {
	f.InjectKey(term.KeyCtrlU, 0)
}

// InjectCtrlK queues a KeyCtrlK event (clear line right).
func (f *FakeKeyboard) InjectCtrlK() {
	f.InjectKey(term.KeyCtrlK, 0)
}

// InjectCtrlW queues a KeyCtrlW event (delete word).
func (f *FakeKeyboard) InjectCtrlW() {
	f.InjectKey(term.KeyCtrlW, 0)
}

// InjectCtrlL queues a KeyCtrlL event (clear screen).
func (f *FakeKeyboard) InjectCtrlL() {
	f.InjectKey(term.KeyCtrlL, 0)
}

// InjectCtrlP queues a KeyCtrlP event (command palette).
func (f *FakeKeyboard) InjectCtrlP() {
	f.InjectKey(term.KeyCtrlP, 0)
}

// InjectUp queues a KeyUp (arrow up) event.
func (f *FakeKeyboard) InjectUp() {
	f.InjectKey(term.KeyUp, 0)
}

// InjectDown queues a KeyDown (arrow down) event.
func (f *FakeKeyboard) InjectDown() {
	f.InjectKey(term.KeyDown, 0)
}

// InjectLeft queues a KeyLeft (arrow left) event.
func (f *FakeKeyboard) InjectLeft() {
	f.InjectKey(term.KeyLeft, 0)
}

// InjectRight queues a KeyRight (arrow right) event.
func (f *FakeKeyboard) InjectRight() {
	f.InjectKey(term.KeyRight, 0)
}

// InjectHome queues a KeyHome event.
func (f *FakeKeyboard) InjectHome() {
	f.InjectKey(term.KeyHome, 0)
}

// InjectEnd queues a KeyEnd event.
func (f *FakeKeyboard) InjectEnd() {
	f.InjectKey(term.KeyEnd, 0)
}

// InjectPageUp queues a KeyPgUp event.
func (f *FakeKeyboard) InjectPageUp() {
	f.InjectKey(term.KeyPgUp, 0)
}

// InjectPageDown queues a KeyPgDn event.
func (f *FakeKeyboard) InjectPageDown() {
	f.InjectKey(term.KeyPgDn, 0)
}

// InjectEscape queues a KeyEscape event.
func (f *FakeKeyboard) InjectEscape() {
	f.InjectKey(term.KeyEscape, 0)
}

// InjectPaste queues a KeyPaste event with text.
// Simulates bracketed paste by injecting KeyPaste followed by text runes.
func (f *FakeKeyboard) InjectPaste(text string) {
	f.InjectKey(term.KeyPaste, 0)
	f.InjectString(text)
}

// Close closes the event channel, signaling EOF to consumers.
// Safe to call multiple times.
func (f *FakeKeyboard) Close() {
	f.mu.Lock()
	defer f.mu.Unlock()

	if !f.closed {
		f.closed = true
		close(f.events)
	}
}

// IsClosed returns true if the keyboard has been closed.
func (f *FakeKeyboard) IsClosed() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.closed
}
