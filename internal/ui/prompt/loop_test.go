package prompt

import (
	"bytes"
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/ui/term"
)

const (
	testHelloInput = "hello"
)


// FakeRenderer records redraw calls for testing.
type FakeRenderer struct {
	mu          sync.Mutex
	RedrawCount int
	LastText    string
	LastStatus  string
	ClearCount  int
	buf         *bytes.Buffer
	done        chan struct{} // Signal when render completes.
}

func NewFakeRenderer() *FakeRenderer {
	return &FakeRenderer{
		buf:  &bytes.Buffer{},
		done: make(chan struct{}, 100),
	}
}

func (f *FakeRenderer) Redraw(m *Model, status string) error {
	f.mu.Lock()
	f.RedrawCount++
	f.LastText = m.Text()
	f.LastStatus = status
	f.mu.Unlock()

	// Signal that a redraw completed.
	select {
	case f.done <- struct{}{}:
	default:
	}

	return nil
}

func (f *FakeRenderer) ClearScreen() error {
	f.mu.Lock()
	f.ClearCount++
	f.mu.Unlock()

	return nil
}

func (f *FakeRenderer) SetWidth(_ int)     {}
func (f *FakeRenderer) SetPrefix(_ string) {}

func (f *FakeRenderer) GetRedrawCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.RedrawCount
}

func (f *FakeRenderer) GetLastText() string {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.LastText
}

func (f *FakeRenderer) GetClearCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.ClearCount
}

// WaitForRedraws waits for n redraws to complete or times out.
func (f *FakeRenderer) WaitForRedraws(n int, timeout time.Duration) bool {
	deadline := time.After(timeout)

	for range n {
		select {
		case <-f.done:
		case <-deadline:
			return false
		}
	}

	return true
}

func TestNewLoop(t *testing.T) {
	t.Parallel()
	model := NewModel(100)
	renderer := NewFakeRenderer()
	keys := make(chan term.KeyEvent)

	loop := NewLoop(model, renderer, keys)

	if loop == nil {
		t.Fatal("Expected non-nil loop")
	}

	if loop.model != model {
		t.Error("Expected model to be set")
	}
}

func TestLoop_Insert(t *testing.T) {
	t.Parallel()
	renderer := NewFakeRenderer()
	keys := make(chan term.KeyEvent, 10)
	model := NewModel(100)

	loop := NewLoop(model, renderer, keys)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	out := loop.Run(ctx)

	// Send keys: testHelloInput.
	for _, r := range testHelloInput {
		keys <- term.KeyEvent{Kind: term.KeyRune, Rune: r}
	}

	// Wait for all redraws.
	if !renderer.WaitForRedraws(5, 100*time.Millisecond) {
		t.Fatal("Timeout waiting for redraws")
	}

	if text := renderer.GetLastText(); text != testHelloInput {
		t.Errorf("Expected 'hello', got %q", text)
	}

	cancel()
	<-out
}

func TestLoop_Backspace(t *testing.T) {
	t.Parallel()
	renderer := NewFakeRenderer()
	keys := make(chan term.KeyEvent, 10)
	model := NewModel(100)

	loop := NewLoop(model, renderer, keys)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	out := loop.Run(ctx)

	// Send: "abc" + backspace.
	keys <- term.KeyEvent{Kind: term.KeyRune, Rune: 'a'}

	keys <- term.KeyEvent{Kind: term.KeyRune, Rune: 'b'}

	keys <- term.KeyEvent{Kind: term.KeyRune, Rune: 'c'}

	keys <- term.KeyEvent{Kind: term.KeyBackspace}

	if !renderer.WaitForRedraws(4, 100*time.Millisecond) {
		t.Fatal("Timeout waiting for redraws")
	}

	if text := renderer.GetLastText(); text != "ab" {
		t.Errorf("Expected 'ab', got %q", text)
	}

	cancel()
	<-out
}

func TestLoop_Navigation(t *testing.T) {
	t.Parallel()
	renderer := NewFakeRenderer()
	keys := make(chan term.KeyEvent, 10)
	model := NewModel(100)

	loop := NewLoop(model, renderer, keys)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	out := loop.Run(ctx)

	// Send: "abc" + left + left.
	keys <- term.KeyEvent{Kind: term.KeyRune, Rune: 'a'}

	keys <- term.KeyEvent{Kind: term.KeyRune, Rune: 'b'}

	keys <- term.KeyEvent{Kind: term.KeyRune, Rune: 'c'}

	keys <- term.KeyEvent{Kind: term.KeyLeft}

	keys <- term.KeyEvent{Kind: term.KeyLeft}

	if !renderer.WaitForRedraws(5, 100*time.Millisecond) {
		t.Fatal("Timeout waiting for redraws")
	}

	cancel()
	<-out
}

func TestLoop_Submit(t *testing.T) {
	t.Parallel()
	renderer := NewFakeRenderer()
	keys := make(chan term.KeyEvent, 10)
	model := NewModel(100)

	loop := NewLoop(model, renderer, keys)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	out := loop.Run(ctx)

	// Send: testHelloInput + enter.
	for _, r := range testHelloInput {
		keys <- term.KeyEvent{Kind: term.KeyRune, Rune: r}
	}

	keys <- term.KeyEvent{Kind: term.KeyEnter}

	// Wait for submit.
	select {
	case line := <-out:
		if line != testHelloInput {
			t.Errorf("Expected 'hello', got %q", line)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for submitted line")
	}

	// Wait for final redraw (after submit).
	if !renderer.WaitForRedraws(6, 100*time.Millisecond) {
		t.Fatal("Timeout waiting for redraws")
	}

	// Buffer should be cleared.
	if text := renderer.GetLastText(); text != "" {
		t.Errorf("Expected empty buffer after submit, got %q", text)
	}

	cancel()
	<-out
}

// loopExitCase describes a test case where a key event should cause the loop to exit.
type loopExitCase struct {
	name string
	key  term.KeyKind
}

func runLoopExitTests(t *testing.T, cases []loopExitCase) {
	t.Helper()

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			renderer := NewFakeRenderer()
			keys := make(chan term.KeyEvent, 10)
			model := NewModel(100)

			loop := NewLoop(model, renderer, keys)

			ctx := t.Context()

			out := loop.Run(ctx)

			keys <- term.KeyEvent{Kind: tt.key}

			select {
			case _, ok := <-out:
				if ok {
					t.Errorf("Expected output channel to close on %v", tt.key)
				}
			case <-time.After(100 * time.Millisecond):
				t.Error("Expected output channel to close quickly")
			}
		})
	}
}

func TestLoop_CtrlC(t *testing.T) {
	t.Parallel()
	runLoopExitTests(t, []loopExitCase{
		{"ctrl-c exits", term.KeyCtrlC},
	})
}

func TestLoop_CtrlD_EmptyBuffer(t *testing.T) {
	t.Parallel()
	runLoopExitTests(t, []loopExitCase{
		{"ctrl-d on empty exits", term.KeyCtrlD},
	})
}

func TestLoop_CtrlD_NonEmptyBuffer(t *testing.T) {
	t.Parallel()
	renderer := NewFakeRenderer()
	keys := make(chan term.KeyEvent, 10)
	model := NewModel(100)

	loop := NewLoop(model, renderer, keys)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	out := loop.Run(ctx)

	// Send: "abc" + home + Ctrl-D.
	keys <- term.KeyEvent{Kind: term.KeyRune, Rune: 'a'}

	keys <- term.KeyEvent{Kind: term.KeyRune, Rune: 'b'}

	keys <- term.KeyEvent{Kind: term.KeyRune, Rune: 'c'}

	keys <- term.KeyEvent{Kind: term.KeyHome}

	keys <- term.KeyEvent{Kind: term.KeyCtrlD}

	if !renderer.WaitForRedraws(5, 100*time.Millisecond) {
		t.Fatal("Timeout waiting for redraws")
	}

	// Should delete character at cursor (should be "bc").
	if text := renderer.GetLastText(); text != "bc" {
		t.Errorf("Expected 'bc' after Ctrl-D, got %q", text)
	}

	cancel()
	<-out
}

func TestLoop_ClearScreen(t *testing.T) {
	t.Parallel()
	renderer := NewFakeRenderer()
	keys := make(chan term.KeyEvent, 10)
	model := NewModel(100)

	loop := NewLoop(model, renderer, keys)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	out := loop.Run(ctx)

	// Send Ctrl-L.
	keys <- term.KeyEvent{Kind: term.KeyCtrlL}

	if !renderer.WaitForRedraws(1, 100*time.Millisecond) {
		t.Fatal("Timeout waiting for redraw")
	}

	if count := renderer.GetClearCount(); count != 1 {
		t.Errorf("Expected 1 clear screen, got %d", count)
	}

	cancel()
	<-out
}

func TestLoop_ContextCancel(t *testing.T) {
	t.Parallel()
	renderer := NewFakeRenderer()
	keys := make(chan term.KeyEvent, 10)
	model := NewModel(100)

	loop := NewLoop(model, renderer, keys)
	ctx, cancel := context.WithCancel(context.Background())

	out := loop.Run(ctx)

	// Cancel context.
	cancel()

	// Output channel should close.
	select {
	case _, ok := <-out:
		if ok {
			t.Error("Expected output channel to close on context cancel")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected output channel to close quickly")
	}
}

func TestLoop_KeyChannelClose(t *testing.T) {
	t.Parallel()
	renderer := NewFakeRenderer()
	keys := make(chan term.KeyEvent, 10)
	model := NewModel(100)

	loop := NewLoop(model, renderer, keys)

	ctx := t.Context()

	out := loop.Run(ctx)

	// Close key channel.
	close(keys)

	// Output channel should close.
	select {
	case _, ok := <-out:
		if ok {
			t.Error("Expected output channel to close when key channel closes")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected output channel to close quickly")
	}
}

func TestLoop_FullInteraction(t *testing.T) {
	t.Parallel()
	model := NewModel(100)

	var buf bytes.Buffer

	renderer := NewTermRenderer(&buf, 80, "> ")
	keys := make(chan term.KeyEvent, 20)

	loop := NewLoop(model, renderer, keys)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	out := loop.Run(ctx)

	// Send sequence: testHelloInput + enter.
	for _, r := range testHelloInput {
		keys <- term.KeyEvent{Kind: term.KeyRune, Rune: r}
	}

	keys <- term.KeyEvent{Kind: term.KeyEnter}

	select {
	case line := <-out:
		if line != testHelloInput {
			t.Errorf("Expected 'hello', got %q", line)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for submitted line")
	}

	// Verify redraws occurred.
	output := buf.String()
	if !strings.Contains(output, testHelloInput) {
		t.Errorf("Expected redraws to contain 'hello'")
	}

	cancel()
	<-out
}

func TestLoop_History(t *testing.T) {
	t.Parallel()
	renderer := NewFakeRenderer()
	keys := make(chan term.KeyEvent, 20)
	model := NewModel(100)

	loop := NewLoop(model, renderer, keys)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	out := loop.Run(ctx)

	// Send: "line1" + enter.
	for _, r := range "line1" {
		keys <- term.KeyEvent{Kind: term.KeyRune, Rune: r}
	}

	keys <- term.KeyEvent{Kind: term.KeyEnter}

	// Wait for submit.
	line := <-out
	if line != "line1" {
		t.Fatalf("Expected 'line1', got %q", line)
	}

	// Navigate up to history.
	keys <- term.KeyEvent{Kind: term.KeyUp}

	if !renderer.WaitForRedraws(7, 100*time.Millisecond) {
		t.Fatal("Timeout waiting for redraws")
	}

	// Should show line1 from history.
	if text := renderer.GetLastText(); text != "line1" {
		t.Errorf("Expected 'line1' from history, got %q", text)
	}

	cancel()
	<-out
}
