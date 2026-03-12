package testkit

import (
	"context"
	"time"

	"github.com/dmytrogajewski/spin/internal/ui/adapters"
)

// TUITestHelper provides utilities for testing TUI components.
type TUITestHelper struct {
	UI       *adapters.PureTTY
	TTY      *FakeTTY
	Keyboard *FakeKeyboard
	Writer   *FakeWriter
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewTUITest creates a new TUI test helper with all fake components.
func NewTUITest(t interface {
	Helper()
	Cleanup(func())
}) *TUITestHelper {
	t.Helper()

	fakeTTY := NewFakeTTY(80, 24)
	fakeKB := NewFakeKeyboard()
	fakeOut := NewFakeWriter()

	ctx, cancel := context.WithCancel(context.Background())

	ui, err := adapters.NewPureTTY(fakeOut,
		adapters.WithTTY(fakeTTY),
		adapters.WithKeyboardEvents(fakeKB.Events()),
	)
	if err != nil {
		t.(interface{ Fatalf(string, ...any) }).Fatalf("NewPureTTY: %v", err)
	}

	helper := &TUITestHelper{
		UI:       ui,
		TTY:      fakeTTY,
		Keyboard: fakeKB,
		Writer:   fakeOut,
		ctx:      ctx,
		cancel:   cancel,
	}

	t.Cleanup(func() {
		cancel()
		ui.Stop()
		fakeKB.Close()
	})

	return helper
}

// Start runs the UI in a background goroutine.
func (h *TUITestHelper) Start() {
	go func() {
		_ = h.UI.Run(h.ctx)
	}()
	// Give UI time to initialize.
	time.Sleep(50 * time.Millisecond)
}

// Stop stops the UI and cancels the context.
func (h *TUITestHelper) Stop() {
	h.cancel()
	h.UI.Stop()
	h.Keyboard.Close()
}

// WaitForOutput waits for the output to contain the given string.
func (h *TUITestHelper) WaitForOutput(s string, timeout time.Duration) bool {
	return h.Writer.WaitForContent(s, timeout)
}

// AssertContains checks if output contains the given string.
func (h *TUITestHelper) AssertContains(t interface {
	Helper()
	Errorf(string, ...any)
}, s string) {
	t.Helper()

	if !h.Writer.Contains(s) {
		t.Errorf("output does not contain %q\nGot: %s", s, h.Writer.StripANSI())
	}
}

// AssertNotContains checks if output does not contain the given string.
func (h *TUITestHelper) AssertNotContains(t interface {
	Helper()
	Errorf(string, ...any)
}, s string) {
	t.Helper()

	if h.Writer.Contains(s) {
		t.Errorf("output contains %q (should not)\nGot: %s", s, h.Writer.StripANSI())
	}
}

// AssertANSISequence checks if output contains the given ANSI sequence.
func (h *TUITestHelper) AssertANSISequence(t interface {
	Helper()
	Errorf(string, ...any)
}, seq string) {
	t.Helper()

	if !h.Writer.ContainsANSI(seq) {
		t.Errorf("output does not contain ANSI sequence %q\nGot: %s", seq, h.Writer.Snapshot())
	}
}
