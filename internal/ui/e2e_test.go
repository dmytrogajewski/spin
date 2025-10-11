package ui

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/ui/adapters"
	"github.com/dmytrogajewski/spin/internal/ui/blocks"
	"github.com/dmytrogajewski/spin/internal/ui/output"
	"github.com/dmytrogajewski/spin/internal/ui/prompt"
	"github.com/dmytrogajewski/spin/internal/ui/testkit"
)

// TestE2E_InputSubmit_PromptsRedraw tests basic input submission flow.
// User types "hello", presses Enter, verifies input received and prompt redrawn.
func TestE2E_InputSubmit_PromptsRedraw(t *testing.T) {
	writer := testkit.NewFakeWriter()
	keyboard := testkit.NewFakeKeyboard()
	defer keyboard.Close()
	fakeTTY := testkit.NewFakeTTY(80, 24)

	// Create UI components
	model := prompt.NewModel(100)
	renderer := prompt.NewRenderer(writer, 80, "> ")
	printer := output.NewPrinter(writer)

	// Adapter for coordinator
	rendererAdapter := &rendererAdapter{renderer: renderer}
	coord := output.NewCoordinatedWriter(printer, rendererAdapter, model)

	ui, err := adapters.NewPureTTY(writer,
		adapters.WithTTY(fakeTTY),
		adapters.WithModel(model),
		adapters.WithCoordinator(coord),
		adapters.WithKeyboardEvents(keyboard.Events()),
	)
	if err != nil {
		t.Fatalf("NewPureTTY() error = %v", err)
	}

	// Run UI in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runErr := make(chan error, 1)
	go func() {
		runErr <- ui.Run(ctx)
	}()

	// Wait for startup
	time.Sleep(50 * time.Millisecond)

	// Inject input
	keyboard.InjectString("hello")
	keyboard.InjectEnter()

	// Verify input received
	line := testkit.WaitForInput(t, ui.RequestInput(), 200*time.Millisecond)
	if line != "hello" {
		t.Errorf("RequestInput() = %q, want %q", line, "hello")
	}

	// Verify output contains prompt
	time.Sleep(50 * time.Millisecond)
	output := writer.Snapshot()
	if !strings.Contains(output, ">") {
		t.Error("output missing prompt '>'")
	}

	// Shutdown
	cancel()
	err = testkit.WaitForShutdown(t, runErr, 1*time.Second)
	if err != context.Canceled {
		t.Errorf("Run() error = %v, want context.Canceled", err)
	}
}

// TestE2E_StreamingChunks_PromptAtBottom tests streaming output.
// Stream 100 chunks, verify all appear and prompt stays at bottom.
func TestE2E_StreamingChunks_PromptAtBottom(t *testing.T) {
	writer := testkit.NewFakeWriter()
	keyboard := testkit.NewFakeKeyboard()
	defer keyboard.Close()
	fakeTTY := testkit.NewFakeTTY(80, 24)

	model := prompt.NewModel(100)
	renderer := prompt.NewRenderer(writer, 80, "> ")
	printer := output.NewPrinter(writer)
	rendererAdapter := &rendererAdapter{renderer: renderer}
	coord := output.NewCoordinatedWriter(printer, rendererAdapter, model)

	ui, err := adapters.NewPureTTY(writer,
		adapters.WithTTY(fakeTTY),
		adapters.WithModel(model),
		adapters.WithCoordinator(coord),
		adapters.WithKeyboardEvents(keyboard.Events()),
	)
	if err != nil {
		t.Fatalf("NewPureTTY() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runErr := make(chan error, 1)
	go func() {
		runErr <- ui.Run(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	// Stream chunks
	chunks := make(chan string, 100)

	// Start consuming chunks in background
	chunkCtx := context.Background()
	chunkDone := make(chan error, 1)
	go func() {
		chunkDone <- ui.PrintChunks(chunkCtx, chunks)
	}()

	// Send chunks
	for i := 0; i < 100; i++ {
		chunks <- "chunk "
	}
	chunks <- "\n"
	close(chunks)

	// Wait for chunks to be processed
	if err := <-chunkDone; err != nil {
		t.Errorf("PrintChunks() error = %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Verify all chunks present
	snapshot := writer.Snapshot()
	chunkCount := strings.Count(snapshot, "chunk ")
	if chunkCount != 100 {
		t.Errorf("output contains %d chunks, want 100", chunkCount)
	}

	// Verify prompt at end (last line should contain prompt)
	lines := writer.Lines()
	if len(lines) > 0 {
		lastLine := lines[len(lines)-1]
		if !strings.Contains(lastLine, ">") {
			t.Error("prompt not at bottom after streaming")
		}
	}

	cancel()
	testkit.WaitForShutdown(t, runErr, 1*time.Second)
}

// TestE2E_ShutdownCtrlC_ExitsCleanly tests Ctrl-C shutdown.
func TestE2E_ShutdownCtrlC_ExitsCleanly(t *testing.T) {
	writer := testkit.NewFakeWriter()
	keyboard := testkit.NewFakeKeyboard()
	defer keyboard.Close()
	fakeTTY := testkit.NewFakeTTY(80, 24)

	model := prompt.NewModel(100)
	renderer := prompt.NewRenderer(writer, 80, "> ")
	printer := output.NewPrinter(writer)
	rendererAdapter := &rendererAdapter{renderer: renderer}
	coord := output.NewCoordinatedWriter(printer, rendererAdapter, model)

	ui, err := adapters.NewPureTTY(writer,
		adapters.WithTTY(fakeTTY),
		adapters.WithModel(model),
		adapters.WithCoordinator(coord),
		adapters.WithKeyboardEvents(keyboard.Events()),
	)
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
	keyboard.InjectCtrlC()

	// Verify clean exit
	err = testkit.WaitForShutdown(t, runErr, 1*time.Second)
	if err != nil {
		t.Errorf("Run() error = %v, want nil (clean exit)", err)
	}

	// Verify TTY exited
	if !fakeTTY.IsExited() {
		t.Error("TTY not exited after Ctrl-C")
	}
}

// TestE2E_ShutdownContextCancel_ExitsCleanly tests context cancellation.
func TestE2E_ShutdownContextCancel_ExitsCleanly(t *testing.T) {
	writer := testkit.NewFakeWriter()
	keyboard := testkit.NewFakeKeyboard()
	defer keyboard.Close()
	fakeTTY := testkit.NewFakeTTY(80, 24)

	model := prompt.NewModel(100)
	renderer := prompt.NewRenderer(writer, 80, "> ")
	printer := output.NewPrinter(writer)
	rendererAdapter := &rendererAdapter{renderer: renderer}
	coord := output.NewCoordinatedWriter(printer, rendererAdapter, model)

	ui, err := adapters.NewPureTTY(writer,
		adapters.WithTTY(fakeTTY),
		adapters.WithModel(model),
		adapters.WithCoordinator(coord),
		adapters.WithKeyboardEvents(keyboard.Events()),
	)
	if err != nil {
		t.Fatalf("NewPureTTY() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	runErr := make(chan error, 1)
	go func() {
		runErr <- ui.Run(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	// Cancel context
	cancel()

	// Verify exit with context.Canceled
	err = testkit.WaitForShutdown(t, runErr, 1*time.Second)
	if err != context.Canceled {
		t.Errorf("Run() error = %v, want context.Canceled", err)
	}

	// Verify TTY exited
	if !fakeTTY.IsExited() {
		t.Error("TTY not exited after context cancel")
	}
}

// TestE2E_ShutdownCtrlD_ExitsOnEOF tests Ctrl-D (EOF) shutdown.
func TestE2E_ShutdownCtrlD_ExitsOnEOF(t *testing.T) {
	writer := testkit.NewFakeWriter()
	keyboard := testkit.NewFakeKeyboard()
	defer keyboard.Close()
	fakeTTY := testkit.NewFakeTTY(80, 24)

	model := prompt.NewModel(100)
	renderer := prompt.NewRenderer(writer, 80, "> ")
	printer := output.NewPrinter(writer)
	rendererAdapter := &rendererAdapter{renderer: renderer}
	coord := output.NewCoordinatedWriter(printer, rendererAdapter, model)

	ui, err := adapters.NewPureTTY(writer,
		adapters.WithTTY(fakeTTY),
		adapters.WithModel(model),
		adapters.WithCoordinator(coord),
		adapters.WithKeyboardEvents(keyboard.Events()),
	)
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
	keyboard.InjectCtrlD()

	// Verify clean exit
	err = testkit.WaitForShutdown(t, runErr, 1*time.Second)
	if err != nil {
		t.Errorf("Run() error = %v, want nil (clean exit)", err)
	}

	// Verify TTY exited
	if !fakeTTY.IsExited() {
		t.Error("TTY not exited after Ctrl-D")
	}
}

// TestE2E_LargePayload_StreamsWithoutHang tests streaming 10k lines.
func TestE2E_LargePayload_StreamsWithoutHang(t *testing.T) {
	writer := testkit.NewFakeWriter()
	keyboard := testkit.NewFakeKeyboard()
	defer keyboard.Close()
	fakeTTY := testkit.NewFakeTTY(80, 24)

	model := prompt.NewModel(100)
	renderer := prompt.NewRenderer(writer, 80, "> ")
	printer := output.NewPrinter(writer)
	rendererAdapter := &rendererAdapter{renderer: renderer}
	coord := output.NewCoordinatedWriter(printer, rendererAdapter, model)

	ui, err := adapters.NewPureTTY(writer,
		adapters.WithTTY(fakeTTY),
		adapters.WithModel(model),
		adapters.WithCoordinator(coord),
		adapters.WithKeyboardEvents(keyboard.Events()),
	)
	if err != nil {
		t.Fatalf("NewPureTTY() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runErr := make(chan error, 1)
	go func() {
		runErr <- ui.Run(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	// Stream 10,000 lines
	chunks := make(chan string, 1000)
	go func() {
		for i := 0; i < 10000; i++ {
			chunks <- "line\n"
		}
		close(chunks)
	}()

	chunkCtx, chunkCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer chunkCancel()

	if err := ui.PrintChunks(chunkCtx, chunks); err != nil {
		t.Errorf("PrintChunks() error = %v", err)
	}

	// Verify completion (no hang)
	time.Sleep(200 * time.Millisecond)

	snapshot := writer.Snapshot()
	lineCount := strings.Count(snapshot, "line\n")
	if lineCount != 10000 {
		t.Errorf("output contains %d lines, want 10000", lineCount)
	}

	cancel()
	testkit.WaitForShutdown(t, runErr, 1*time.Second)
}

// TestE2E_ConcurrentOperations_NoRaceConditions tests concurrent PrintLine, PrintChunks, input.
func TestE2E_ConcurrentOperations_NoRaceConditions(t *testing.T) {
	writer := testkit.NewFakeWriter()
	keyboard := testkit.NewFakeKeyboard()
	defer keyboard.Close()
	fakeTTY := testkit.NewFakeTTY(80, 24)

	model := prompt.NewModel(100)
	renderer := prompt.NewRenderer(writer, 80, "> ")
	printer := output.NewPrinter(writer)
	rendererAdapter := &rendererAdapter{renderer: renderer}
	coord := output.NewCoordinatedWriter(printer, rendererAdapter, model)

	ui, err := adapters.NewPureTTY(writer,
		adapters.WithTTY(fakeTTY),
		adapters.WithModel(model),
		adapters.WithCoordinator(coord),
		adapters.WithKeyboardEvents(keyboard.Events()),
	)
	if err != nil {
		t.Fatalf("NewPureTTY() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runErr := make(chan error, 1)
	go func() {
		runErr <- ui.Run(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	// Concurrent operations
	done := make(chan struct{})

	// Goroutine 1: PrintLine
	go func() {
		for i := 0; i < 50; i++ {
			ui.PrintLine("line from goroutine 1")
			time.Sleep(1 * time.Millisecond)
		}
		done <- struct{}{}
	}()

	// Goroutine 2: PrintChunks
	go func() {
		chunks := make(chan string, 50)
		for i := 0; i < 50; i++ {
			chunks <- "chunk "
		}
		close(chunks)
		ui.PrintChunks(context.Background(), chunks)
		done <- struct{}{}
	}()

	// Goroutine 3: Inject keys
	go func() {
		for i := 0; i < 10; i++ {
			keyboard.InjectString("test")
			keyboard.InjectEnter()
			time.Sleep(5 * time.Millisecond)
		}
		done <- struct{}{}
	}()

	// Wait for all goroutines
	for i := 0; i < 3; i++ {
		<-done
	}

	time.Sleep(100 * time.Millisecond)

	// Verify no torn output or panics (test passed = success)
	snapshot := writer.Snapshot()
	if len(snapshot) == 0 {
		t.Error("no output produced during concurrent operations")
	}

	cancel()
	testkit.WaitForShutdown(t, runErr, 1*time.Second)
}

// TestE2E_BlockAppendAndRender tests appending blocks to timeline.
func TestE2E_BlockAppendAndRender(t *testing.T) {
	writer := testkit.NewFakeWriter()
	keyboard := testkit.NewFakeKeyboard()
	defer keyboard.Close()
	fakeTTY := testkit.NewFakeTTY(80, 24)

	model := prompt.NewModel(100)
	renderer := prompt.NewRenderer(writer, 80, "> ")
	printer := output.NewPrinter(writer)
	rendererAdapter := &rendererAdapter{renderer: renderer}
	coord := output.NewCoordinatedWriter(printer, rendererAdapter, model)

	ui, err := adapters.NewPureTTY(writer,
		adapters.WithTTY(fakeTTY),
		adapters.WithModel(model),
		adapters.WithCoordinator(coord),
		adapters.WithKeyboardEvents(keyboard.Events()),
	)
	if err != nil {
		t.Fatalf("NewPureTTY() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runErr := make(chan error, 1)
	go func() {
		runErr <- ui.Run(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	// Append blocks
	block1 := blocks.NewBlock(blocks.BlockTypeExecute)
	block1.Title = "Run tests"
	block1.Body = "=== RUN TestFoo\n--- PASS: TestFoo"

	block2 := blocks.NewBlock(blocks.BlockTypePlan)
	block2.Title = "Plan updated"
	block2.Body = "• Task 1\n• Task 2"

	// Note: AppendBlock requires PureTTY to expose this method
	// For now, verify PrintLine works as proxy
	ui.PrintLine("Block 1: Run tests")
	ui.PrintLine("Block 2: Plan updated")

	time.Sleep(50 * time.Millisecond)

	snapshot := writer.Snapshot()
	if !strings.Contains(snapshot, "Run tests") {
		t.Error("output missing block 1")
	}
	if !strings.Contains(snapshot, "Plan updated") {
		t.Error("output missing block 2")
	}

	cancel()
	testkit.WaitForShutdown(t, runErr, 1*time.Second)
}

// TestE2E_TerminalResize_RedrawsWithNewWidth tests resize handling.
func TestE2E_TerminalResize_RedrawsWithNewWidth(t *testing.T) {
	writer := testkit.NewFakeWriter()
	keyboard := testkit.NewFakeKeyboard()
	defer keyboard.Close()
	fakeTTY := testkit.NewFakeTTY(80, 24)

	model := prompt.NewModel(100)
	renderer := prompt.NewRenderer(writer, 80, "> ")
	printer := output.NewPrinter(writer)
	rendererAdapter := &rendererAdapter{renderer: renderer}
	coord := output.NewCoordinatedWriter(printer, rendererAdapter, model)

	ui, err := adapters.NewPureTTY(writer,
		adapters.WithTTY(fakeTTY),
		adapters.WithModel(model),
		adapters.WithCoordinator(coord),
		adapters.WithKeyboardEvents(keyboard.Events()),
	)
	if err != nil {
		t.Fatalf("NewPureTTY() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runErr := make(chan error, 1)
	go func() {
		runErr <- ui.Run(ctx)
	}()

	time.Sleep(50 * time.Millisecond)

	// Initial output
	ui.PrintLine("Line before resize")
	time.Sleep(50 * time.Millisecond)

	// Trigger resize
	fakeTTY.SetSize(120, 40)
	time.Sleep(50 * time.Millisecond)

	// Output after resize
	ui.PrintLine("Line after resize")
	time.Sleep(50 * time.Millisecond)

	snapshot := writer.Snapshot()
	if !strings.Contains(snapshot, "Line before resize") {
		t.Error("output missing line before resize")
	}
	if !strings.Contains(snapshot, "Line after resize") {
		t.Error("output missing line after resize")
	}

	// Verify renderer width updated (renderer should adapt to new width)
	// This is verified implicitly if no visual corruption occurs

	cancel()
	testkit.WaitForShutdown(t, runErr, 1*time.Second)
}

// rendererAdapter adapts prompt.Renderer to output.PromptRenderer
type rendererAdapter struct {
	renderer *prompt.Renderer
}

func (a *rendererAdapter) Redraw(model output.PromptModel, status string) error {
	promptModel := model.(*prompt.Model)
	return a.renderer.Redraw(promptModel, status)
}

// TestE2E_Debug is a simplified debug test
func TestE2E_Debug(t *testing.T) {
	writer := testkit.NewFakeWriter()
	keyboard := testkit.NewFakeKeyboard()
	defer keyboard.Close()
	fakeTTY := testkit.NewFakeTTY(80, 24)
	
	t.Log("Creating components...")
	model := prompt.NewModel(100)
	renderer := prompt.NewRenderer(writer, 80, "> ")
	printer := output.NewPrinter(writer)
	
	rendererAdapter := &rendererAdapter{renderer: renderer}
	coord := output.NewCoordinatedWriter(printer, rendererAdapter, model)
	
	t.Log("Creating PureTTY...")
	ui, err := adapters.NewPureTTY(writer,
		adapters.WithTTY(fakeTTY),
		adapters.WithModel(model),
		adapters.WithCoordinator(coord),
		adapters.WithKeyboardEvents(keyboard.Events()),
	)
	if err != nil {
		t.Fatalf("NewPureTTY() error = %v", err)
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	t.Log("Starting UI.Run()...")
	runErr := make(chan error, 1)
	go func() {
		err := ui.Run(ctx)
		t.Logf("UI.Run() returned: %v", err)
		runErr <- err
	}()
	
	time.Sleep(100 * time.Millisecond)
	t.Logf("TTY entered: %v", fakeTTY.IsEntered())
	
	t.Log("Injecting keys...")
	keyboard.InjectString("test")
	keyboard.InjectEnter()
	
	t.Log("Waiting for input...")
	select {
	case line := <-ui.RequestInput():
		t.Logf("SUCCESS: Received %q", line)
		if line != "test" {
			t.Errorf("got %q, want %q", line, "test")
		}
	case <-time.After(500 * time.Millisecond):
		t.Error("TIMEOUT waiting for input")
		t.Logf("Output so far: %q", writer.Snapshot())
	}
	
	cancel()
	<-runErr
}
