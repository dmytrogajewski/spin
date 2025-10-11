package adapters

import (
	"bytes"
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/ui/output"
	"github.com/dmytrogajewski/spin/internal/ui/ports"
	"github.com/dmytrogajewski/spin/internal/ui/prompt"
	"github.com/dmytrogajewski/spin/internal/ui/term"
)

// safeBuffer wraps bytes.Buffer with mutex for thread-safe writes
type safeBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (s *safeBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

func (s *safeBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.String()
}

// rendererAdapter adapts prompt.Renderer to output.PromptRenderer for tests
type testRendererAdapter struct {
	renderer *prompt.Renderer
}

func (a *testRendererAdapter) Redraw(model output.PromptModel, status string) error {
	promptModel := model.(*prompt.Model)
	return a.renderer.Redraw(promptModel, status)
}

// fakeTTY wraps term.TTY for testing
type fakeTTY struct {
	*term.TTY
	mu            sync.Mutex
	width, height int
	entered       bool
	exited        bool
	events        chan term.KeyEvent
	out           *bytes.Buffer
}

func newFakeTTY(w, h int) *fakeTTY {
	return &fakeTTY{
		width:  w,
		height: h,
		events: make(chan term.KeyEvent, 100),
		out:    &bytes.Buffer{},
	}
}

func (f *fakeTTY) Enter() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.entered = true
	return nil
}

func (f *fakeTTY) Exit() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.exited = true
	return nil
}

func (f *fakeTTY) Size() (int, int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.width, f.height
}

func (f *fakeTTY) OnResize(cb func(w, h int)) {
	// Simple no-op for testing
}

func (f *fakeTTY) InjectKey(kind term.KeyKind, r rune) {
	f.events <- term.KeyEvent{Kind: kind, Rune: r}
}

func (f *fakeTTY) IsEntered() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.entered
}

func (f *fakeTTY) IsExited() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.exited
}

// TestNewPureTTY tests constructor with default and custom options
func TestNewPureTTY(t *testing.T) {
	tests := []struct {
		name    string
		opts    []PureTTYOption
		wantErr bool
	}{
		{
			name:    "with custom components",
			opts: []PureTTYOption{
				WithModel(prompt.NewModel(50)),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := &safeBuffer{}

			// Create fake TTY and inject it
			fakeTTY := newFakeTTY(80, 24)
			opts := append([]PureTTYOption{WithTTY(fakeTTY)}, tt.opts...)

			ui, err := NewPureTTY(out, opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewPureTTY() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && ui == nil {
				t.Error("NewPureTTY() returned nil UI")
			}
		})
	}
}

// TestRun_Basic tests basic Run cycle with input and shutdown
func TestRun_Basic(t *testing.T) {
	out := &safeBuffer{}

	// Create components manually
	fakeTTY := newFakeTTY(80, 24)
	model := prompt.NewModel(100)
	renderer := prompt.NewRenderer(out, 80, "> ")
	printer := output.NewPrinter(out)
	rendererAdapter := &testRendererAdapter{renderer: renderer}
	coord := output.NewCoordinatedWriter(printer, rendererAdapter, model)

	ui, err := NewPureTTY(out,
		WithTTY(fakeTTY),
		WithModel(model),
		WithCoordinator(coord),
		WithKeyboardEvents(fakeTTY.events))
	if err != nil {
		t.Fatalf("NewPureTTY() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Run in background
	runErr := make(chan error, 1)
	go func() {
		runErr <- ui.Run(ctx)
	}()

	// Wait for startup
	time.Sleep(50 * time.Millisecond)

	// Check TTY entered raw mode
	if !fakeTTY.IsEntered() {
		t.Error("TTY not entered")
	}

	// Simulate input
	fakeTTY.InjectKey(term.KeyRune, 'h')
	fakeTTY.InjectKey(term.KeyRune, 'i')
	fakeTTY.InjectKey(term.KeyEnter, 0)

	// Wait for input processing
	time.Sleep(50 * time.Millisecond)

	// Check RequestInput channel receives line
	inputs := ui.RequestInput()
	select {
	case line := <-inputs:
		if line != "hi" {
			t.Errorf("RequestInput() = %q, want %q", line, "hi")
		}
	case <-time.After(200 * time.Millisecond):
		t.Error("timeout waiting for input")
	}

	// Cancel and check clean shutdown
	cancel()

	select {
	case err := <-runErr:
		if err != context.Canceled {
			t.Errorf("Run() error = %v, want context.Canceled", err)
		}
	case <-time.After(1 * time.Second):
		t.Error("Run() did not exit on context cancel")
	}

	// Check TTY exited
	if !fakeTTY.IsExited() {
		t.Error("TTY not exited")
	}
}

// TestRun_ContextCancel tests graceful shutdown on context cancel
func TestRun_ContextCancel(t *testing.T) {
	out := &safeBuffer{}
	fakeTTY := newFakeTTY(80, 24)
	model := prompt.NewModel(100)
	renderer := prompt.NewRenderer(out, 80, "> ")
	printer := output.NewPrinter(out)
	rendererAdapter := &testRendererAdapter{renderer: renderer}
	coord := output.NewCoordinatedWriter(printer, rendererAdapter, model)

	ui, err := NewPureTTY(out,
		WithTTY(fakeTTY),
		WithModel(model),
		WithCoordinator(coord),
		WithKeyboardEvents(fakeTTY.events))
	if err != nil {
		t.Fatalf("NewPureTTY() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	runErr := make(chan error, 1)
	go func() {
		runErr <- ui.Run(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	// Cancel immediately
	cancel()

	select {
	case err := <-runErr:
		if err != context.Canceled {
			t.Errorf("Run() error = %v, want context.Canceled", err)
		}
	case <-time.After(1 * time.Second):
		t.Error("Run() did not exit on context cancel")
	}

	if !fakeTTY.IsExited() {
		t.Error("TTY not exited on context cancel")
	}
}

// TestRun_CtrlC tests shutdown on Ctrl-C
func TestRun_CtrlC(t *testing.T) {
	out := &safeBuffer{}
	fakeTTY := newFakeTTY(80, 24)
	model := prompt.NewModel(100)
	renderer := prompt.NewRenderer(out, 80, "> ")
	printer := output.NewPrinter(out)
	rendererAdapter := &testRendererAdapter{renderer: renderer}
	coord := output.NewCoordinatedWriter(printer, rendererAdapter, model)

	ui, err := NewPureTTY(out, WithTTY(fakeTTY), WithModel(model), WithCoordinator(coord))
	if err != nil {
		t.Fatalf("NewPureTTY() error = %v", err)
	}

	ctx := context.Background()

	runErr := make(chan error, 1)
	go func() {
		runErr <- ui.Run(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	// Send Ctrl-C
	fakeTTY.InjectKey(term.KeyCtrlC, 0)

	select {
	case err := <-runErr:
		if err != nil {
			t.Errorf("Run() error = %v, want nil", err)
		}
	case <-time.After(1 * time.Second):
		t.Error("Run() did not exit on Ctrl-C")
	}
}

// TestRun_CtrlD tests shutdown on Ctrl-D with empty buffer
func TestRun_CtrlD(t *testing.T) {
	out := &safeBuffer{}
	fakeTTY := newFakeTTY(80, 24)
	model := prompt.NewModel(100)
	renderer := prompt.NewRenderer(out, 80, "> ")
	printer := output.NewPrinter(out)
	rendererAdapter := &testRendererAdapter{renderer: renderer}
	coord := output.NewCoordinatedWriter(printer, rendererAdapter, model)

	ui, err := NewPureTTY(out, WithTTY(fakeTTY), WithModel(model), WithCoordinator(coord))
	if err != nil {
		t.Fatalf("NewPureTTY() error = %v", err)
	}

	ctx := context.Background()

	runErr := make(chan error, 1)
	go func() {
		runErr <- ui.Run(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	// Send Ctrl-D on empty buffer (EOF)
	fakeTTY.InjectKey(term.KeyCtrlD, 0)

	select {
	case err := <-runErr:
		if err != nil {
			t.Errorf("Run() error = %v, want nil", err)
		}
	case <-time.After(1 * time.Second):
		t.Error("Run() did not exit on Ctrl-D")
	}
}

// TestStop_Idempotent tests multiple Stop() calls
func TestStop_Idempotent(t *testing.T) {
	out := &safeBuffer{}
	fakeTTY := newFakeTTY(80, 24)
	model := prompt.NewModel(100)
	renderer := prompt.NewRenderer(out, 80, "> ")
	printer := output.NewPrinter(out)
	rendererAdapter := &testRendererAdapter{renderer: renderer}
	coord := output.NewCoordinatedWriter(printer, rendererAdapter, model)

	ui, err := NewPureTTY(out, WithTTY(fakeTTY), WithModel(model), WithCoordinator(coord))
	if err != nil {
		t.Fatalf("NewPureTTY() error = %v", err)
	}

	// Call Stop multiple times
	for i := 0; i < 3; i++ {
		if err := ui.Stop(); err != nil {
			t.Errorf("Stop() call %d error = %v", i+1, err)
		}
	}
}

// TestPrintLine tests output printing
func TestPrintLine(t *testing.T) {
	out := &safeBuffer{}
	fakeTTY := newFakeTTY(80, 24)
	model := prompt.NewModel(100)
	renderer := prompt.NewRenderer(out, 80, "> ")
	printer := output.NewPrinter(out)
	rendererAdapter := &testRendererAdapter{renderer: renderer}
	coord := output.NewCoordinatedWriter(printer, rendererAdapter, model)

	ui, err := NewPureTTY(out, WithTTY(fakeTTY), WithModel(model), WithCoordinator(coord))
	if err != nil {
		t.Fatalf("NewPureTTY() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go ui.Run(ctx)
	time.Sleep(50 * time.Millisecond)

	// Print line
	line := "Test output"
	if err := ui.PrintLine(line); err != nil {
		t.Errorf("PrintLine() error = %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	// Check output contains line
	if !strings.Contains(out.String(), line) {
		t.Errorf("PrintLine() output does not contain %q", line)
	}

	cancel()
}

// TestPrintChunks tests streaming output
func TestPrintChunks(t *testing.T) {
	out := &safeBuffer{}
	fakeTTY := newFakeTTY(80, 24)
	model := prompt.NewModel(100)
	renderer := prompt.NewRenderer(out, 80, "> ")
	printer := output.NewPrinter(out)
	rendererAdapter := &testRendererAdapter{renderer: renderer}
	coord := output.NewCoordinatedWriter(printer, rendererAdapter, model)

	ui, err := NewPureTTY(out, WithTTY(fakeTTY), WithModel(model), WithCoordinator(coord))
	if err != nil {
		t.Fatalf("NewPureTTY() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go ui.Run(ctx)
	time.Sleep(50 * time.Millisecond)

	// Create chunks
	chunks := make(chan string, 10)
	chunks <- "chunk1"
	chunks <- "chunk2"
	chunks <- "\n"
	close(chunks)

	// Print chunks
	chunkCtx := context.Background()
	if err := ui.PrintChunks(chunkCtx, chunks); err != nil {
		t.Errorf("PrintChunks() error = %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Check output contains chunks
	output := out.String()
	if !strings.Contains(output, "chunk1") || !strings.Contains(output, "chunk2") {
		t.Errorf("PrintChunks() output missing chunks")
	}

	cancel()
}

// TestSetStatus tests status update
func TestSetStatus(t *testing.T) {
	out := &safeBuffer{}
	fakeTTY := newFakeTTY(80, 24)
	model := prompt.NewModel(100)
	renderer := prompt.NewRenderer(out, 80, "> ")
	printer := output.NewPrinter(out)
	rendererAdapter := &testRendererAdapter{renderer: renderer}
	coord := output.NewCoordinatedWriter(printer, rendererAdapter, model)

	ui, err := NewPureTTY(out, WithTTY(fakeTTY), WithModel(model), WithCoordinator(coord))
	if err != nil {
		t.Fatalf("NewPureTTY() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go ui.Run(ctx)
	time.Sleep(50 * time.Millisecond)

	// Set status
	if err := ui.SetStatus("typing..."); err != nil {
		t.Errorf("SetStatus() error = %v", err)
	}

	time.Sleep(50 * time.Millisecond)
	cancel()
}

// TestRequestInput tests input channel
func TestRequestInput(t *testing.T) {
	out := &safeBuffer{}
	fakeTTY := newFakeTTY(80, 24)
	model := prompt.NewModel(100)
	renderer := prompt.NewRenderer(out, 80, "> ")
	printer := output.NewPrinter(out)
	rendererAdapter := &testRendererAdapter{renderer: renderer}
	coord := output.NewCoordinatedWriter(printer, rendererAdapter, model)

	ui, err := NewPureTTY(out,
		WithTTY(fakeTTY),
		WithModel(model),
		WithCoordinator(coord),
		WithKeyboardEvents(fakeTTY.events))
	if err != nil {
		t.Fatalf("NewPureTTY() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go ui.Run(ctx)
	time.Sleep(50 * time.Millisecond)

	// Get input channel
	inputs := ui.RequestInput()

	// Inject keys
	fakeTTY.InjectKey(term.KeyRune, 'a')
	fakeTTY.InjectKey(term.KeyRune, 'b')
	fakeTTY.InjectKey(term.KeyRune, 'c')
	fakeTTY.InjectKey(term.KeyEnter, 0)

	// Read from channel
	select {
	case line := <-inputs:
		if line != "abc" {
			t.Errorf("RequestInput() = %q, want %q", line, "abc")
		}
	case <-time.After(200 * time.Millisecond):
		t.Error("timeout waiting for input")
	}

	// Verify same channel returned on repeated calls
	inputs2 := ui.RequestInput()
	if inputs != inputs2 {
		t.Error("RequestInput() returned different channel on second call")
	}

	cancel()
}

// TestCleanShutdown tests no goroutine leaks
func TestCleanShutdown(t *testing.T) {
	out := &safeBuffer{}
	fakeTTY := newFakeTTY(80, 24)
	model := prompt.NewModel(100)
	renderer := prompt.NewRenderer(out, 80, "> ")
	printer := output.NewPrinter(out)
	rendererAdapter := &testRendererAdapter{renderer: renderer}
	coord := output.NewCoordinatedWriter(printer, rendererAdapter, model)

	ui, err := NewPureTTY(out, WithTTY(fakeTTY), WithModel(model), WithCoordinator(coord))
	if err != nil {
		t.Fatalf("NewPureTTY() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	runErr := make(chan error, 1)
	go func() {
		runErr <- ui.Run(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	// Cancel and wait
	cancel()

	select {
	case <-runErr:
		// Success
	case <-time.After(1 * time.Second):
		t.Fatal("Run() did not exit, potential goroutine leak")
	}

	// Additional verification: Stop() should be idempotent
	if err := ui.Stop(); err != nil {
		t.Errorf("Stop() error = %v", err)
	}
}

// Verify PureTTY implements ports.UI
var _ ports.UI = (*PureTTY)(nil)
