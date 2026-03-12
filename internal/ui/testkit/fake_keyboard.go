package testkit

import (
	"sync"

	"github.com/dmytrogajewski/spin/internal/ui/term"
)

// FakeKeyboard provides scripted key events for testing.
// It implements a channel-based keyboard input simulator.
type FakeKeyboard struct {
	mu     sync.Mutex
	events chan term.KeyEvent
	closed bool
}

// NewFakeKeyboard creates a new fake keyboard.
func NewFakeKeyboard() *FakeKeyboard {
	return &FakeKeyboard{
		events: make(chan term.KeyEvent, 100), // Buffered for burst input.
	}
}

// Events returns the channel of key events for use with PureTTY.
func (f *FakeKeyboard) Events() <-chan term.KeyEvent {
	return f.events
}

// InjectKey queues a single key event.
func (f *FakeKeyboard) InjectKey(kind term.KeyKind, r rune) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.closed {
		return
	}

	f.events <- term.KeyEvent{Kind: kind, Rune: r}
}

// InjectString queues key events for each rune in the string.
func (f *FakeKeyboard) InjectString(s string) {
	for _, r := range s {
		f.InjectKey(term.KeyRune, r)
	}
}

// InjectEnter queues a KeyEnter event.
func (f *FakeKeyboard) InjectEnter() {
	f.InjectKey(term.KeyEnter, 0)
}

// InjectCtrlC queues a KeyCtrlC event.
func (f *FakeKeyboard) InjectCtrlC() {
	f.InjectKey(term.KeyCtrlC, 0)
}

// InjectCtrlD queues a KeyCtrlD event.
func (f *FakeKeyboard) InjectCtrlD() {
	f.InjectKey(term.KeyCtrlD, 0)
}

// InjectEscape queues a KeyEscape event.
func (f *FakeKeyboard) InjectEscape() {
	f.InjectKey(term.KeyEscape, 0)
}

// InjectPgUp queues a KeyPgUp event.
func (f *FakeKeyboard) InjectPgUp() {
	f.InjectKey(term.KeyPgUp, 0)
}

// InjectPgDn queues a KeyPgDn event.
func (f *FakeKeyboard) InjectPgDn() {
	f.InjectKey(term.KeyPgDn, 0)
}

// InjectBackspace queues a KeyBackspace event.
func (f *FakeKeyboard) InjectBackspace() {
	f.InjectKey(term.KeyBackspace, 0)
}

// InjectPaste queues a bracketed paste event.
func (f *FakeKeyboard) InjectPaste(text string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.closed {
		return
	}

	f.events <- term.KeyEvent{
		Kind:  term.KeyPaste,
		Paste: []byte(text),
	}
}

// Close closes the event channel (signals EOF).
func (f *FakeKeyboard) Close() {
	f.mu.Lock()
	defer f.mu.Unlock()

	if !f.closed {
		close(f.events)
		f.closed = true
	}
}
