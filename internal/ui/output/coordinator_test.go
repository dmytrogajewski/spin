package output

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

// FakeModel implements PromptModel for testing.
type FakeModel struct {
	text   string
	cursor int
}

func (f *FakeModel) Text() string {
	return f.text
}

func (f *FakeModel) Cursor() int {
	return f.cursor
}

// FakeRenderer implements PromptRenderer for testing.
type FakeRenderer struct {
	mu        sync.Mutex
	redraws   []RedrawCall
	redrawErr error
}

// RedrawCall records a single Redraw invocation.
type RedrawCall struct {
	Text   string
	Cursor int
	Status string
}

func (f *FakeRenderer) Redraw(model PromptModel, status string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.redrawErr != nil {
		return f.redrawErr
	}

	f.redraws = append(f.redraws, RedrawCall{
		Text:   model.Text(),
		Cursor: model.Cursor(),
		Status: status,
	})
	return nil
}

func (f *FakeRenderer) Redraws() []RedrawCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]RedrawCall(nil), f.redraws...)
}

func (f *FakeRenderer) SetRedrawError(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.redrawErr = err
}

func (f *FakeRenderer) Reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.redraws = nil
	f.redrawErr = nil
}

// TestCoordinated_NewCoordinatedWriter verifies constructor.
func TestCoordinated_NewCoordinatedWriter(t *testing.T) {
	var buf bytes.Buffer
	printer := NewPrinter(&buf)
	renderer := &FakeRenderer{}
	model := &FakeModel{text: "hello", cursor: 5}

	coord := NewCoordinatedWriter(printer, renderer, model)
	if coord == nil {
		t.Fatal("NewCoordinatedWriter returned nil")
	}
}

// TestCoordinated_PrintLine_SingleCall verifies basic PrintLine functionality.
func TestCoordinated_PrintLine_SingleCall(t *testing.T) {
	var buf bytes.Buffer
	printer := NewPrinter(&buf)
	renderer := &FakeRenderer{}
	model := &FakeModel{text: "hello", cursor: 5}

	coord := NewCoordinatedWriter(printer, renderer, model)

	err := coord.PrintLine("User: Hi!")
	if err != nil {
		t.Fatalf("PrintLine failed: %v", err)
	}

	// Verify output
	output := buf.String()
	// Printer adds \r\n for TTY compatibility
	if output != "User: Hi!\r\n" {
		t.Errorf("expected output %q, got %q", "User: Hi!\r\n", output)
	}

	// Verify redraw was called
	redraws := renderer.Redraws()
	if len(redraws) != 1 {
		t.Fatalf("expected 1 redraw, got %d", len(redraws))
	}

	if redraws[0].Text != "hello" || redraws[0].Cursor != 5 || redraws[0].Status != "" {
		t.Errorf("unexpected redraw: %+v", redraws[0])
	}
}

// TestCoordinated_PrintLine_Multiple verifies multiple PrintLine calls.
func TestCoordinated_PrintLine_Multiple(t *testing.T) {
	var buf bytes.Buffer
	printer := NewPrinter(&buf)
	renderer := &FakeRenderer{}
	model := &FakeModel{text: "prompt", cursor: 6}

	coord := NewCoordinatedWriter(printer, renderer, model)

	lines := []string{"Line 1", "Line 2", "Line 3"}
	for _, line := range lines {
		if err := coord.PrintLine(line); err != nil {
			t.Fatalf("PrintLine(%q) failed: %v", line, err)
		}
	}

	// Verify output - Printer adds \r\n for TTY compatibility
	expected := "Line 1\r\nLine 2\r\nLine 3\r\n"
	output := buf.String()
	if output != expected {
		t.Errorf("expected output %q, got %q", expected, output)
	}

	// Verify redraws (one per line)
	redraws := renderer.Redraws()
	if len(redraws) != 3 {
		t.Fatalf("expected 3 redraws, got %d", len(redraws))
	}
}

// TestCoordinated_PrintLine_Concurrent verifies thread safety.
func TestCoordinated_PrintLine_Concurrent(t *testing.T) {
	var buf bytes.Buffer
	printer := NewPrinter(&buf)
	renderer := &FakeRenderer{}
	model := &FakeModel{text: "test", cursor: 4}

	coord := NewCoordinatedWriter(printer, renderer, model)

	const goroutines = 10
	const linesPerGoroutine = 10

	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < linesPerGoroutine; j++ {
				line := "line"
				if err := coord.PrintLine(line); err != nil {
					t.Errorf("PrintLine failed: %v", err)
				}
			}
		}(i)
	}
	wg.Wait()

	// Verify all lines written
	output := buf.String()
	lineCount := strings.Count(output, "\n")
	expected := goroutines * linesPerGoroutine
	if lineCount != expected {
		t.Errorf("expected %d lines, got %d", expected, lineCount)
	}

	// Verify redraws (one per line)
	redraws := renderer.Redraws()
	if len(redraws) != expected {
		t.Errorf("expected %d redraws, got %d", expected, len(redraws))
	}
}

// TestCoordinated_PrintLine_PrinterError verifies error handling when printer fails.
func TestCoordinated_PrintLine_PrinterError(t *testing.T) {
	// Writer that always fails
	writer := &errorWriter{err: errors.New("write error")}
	printer := NewPrinter(writer)
	renderer := &FakeRenderer{}
	model := &FakeModel{text: "test", cursor: 4}

	coord := NewCoordinatedWriter(printer, renderer, model)

	err := coord.PrintLine("test")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Renderer should not be called on printer error
	redraws := renderer.Redraws()
	if len(redraws) != 0 {
		t.Errorf("expected 0 redraws on error, got %d", len(redraws))
	}
}

// TestCoordinated_PrintLine_RendererError verifies error handling when renderer fails.
func TestCoordinated_PrintLine_RendererError(t *testing.T) {
	var buf bytes.Buffer
	printer := NewPrinter(&buf)
	renderer := &FakeRenderer{}
	renderer.SetRedrawError(errors.New("redraw error"))
	model := &FakeModel{text: "test", cursor: 4}

	coord := NewCoordinatedWriter(printer, renderer, model)

	err := coord.PrintLine("test")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Printer should have succeeded (output written)
	output := buf.String()
	// Printer adds \r\n for TTY compatibility
	if output != "test\r\n" {
		t.Errorf("expected output %q, got %q", "test\r\n", output)
	}
}

// TestCoordinated_PrintChunks_SingleStream verifies basic streaming functionality.
func TestCoordinated_PrintChunks_SingleStream(t *testing.T) {
	var buf bytes.Buffer
	printer := NewPrinter(&buf, WithCoalesceDelay(0)) // No coalescing for deterministic tests
	renderer := &FakeRenderer{}
	model := &FakeModel{text: "prompt", cursor: 6}

	coord := NewCoordinatedWriter(printer, renderer, model)

	chunks := make(chan string, 3)
	chunks <- "Hello"
	chunks <- " "
	chunks <- "World\n"
	close(chunks)

	ctx := context.Background()
	err := coord.PrintChunks(ctx, chunks)
	if err != nil {
		t.Fatalf("PrintChunks failed: %v", err)
	}

	// Verify output
	output := buf.String()
	// Printer adds \r\n for TTY compatibility, plus extra \r\n from chunks
	expected := "Hello World\r\n\r\n"
	if output != expected {
		t.Errorf("expected output %q, got %q", expected, output)
	}

	// Verify final redraw
	redraws := renderer.Redraws()
	if len(redraws) < 1 {
		t.Fatalf("expected at least 1 redraw, got %d", len(redraws))
	}
}

// TestCoordinated_PrintChunks_ContextCancel verifies context cancellation.
func TestCoordinated_PrintChunks_ContextCancel(t *testing.T) {
	var buf bytes.Buffer
	printer := NewPrinter(&buf, WithCoalesceDelay(0))
	renderer := &FakeRenderer{}
	model := &FakeModel{text: "test", cursor: 4}

	coord := NewCoordinatedWriter(printer, renderer, model)

	chunks := make(chan string)
	ctx, cancel := context.WithCancel(context.Background())

	// Start streaming
	errChan := make(chan error, 1)
	go func() {
		errChan <- coord.PrintChunks(ctx, chunks)
	}()

	// Send a chunk
	chunks <- "partial"

	// Cancel context
	cancel()

	// Wait for completion
	err := <-errChan
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}

	// Verify partial output was written
	output := buf.String()
	if !strings.Contains(output, "partial") {
		t.Errorf("expected partial output to contain %q, got %q", "partial", output)
	}

	// Verify final redraw happened
	redraws := renderer.Redraws()
	if len(redraws) < 1 {
		t.Errorf("expected at least 1 redraw on cancel, got %d", len(redraws))
	}
}

// TestCoordinated_PrintChunks_Concurrent verifies concurrent streaming.
func TestCoordinated_PrintChunks_Concurrent(t *testing.T) {
	var buf bytes.Buffer
	printer := NewPrinter(&buf, WithCoalesceDelay(0))
	renderer := &FakeRenderer{}
	model := &FakeModel{text: "test", cursor: 4}

	coord := NewCoordinatedWriter(printer, renderer, model)

	const streams = 3
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < streams; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			chunks := make(chan string, 2)
			chunks <- "stream"
			chunks <- "\n"
			close(chunks)

			if err := coord.PrintChunks(ctx, chunks); err != nil {
				t.Errorf("PrintChunks failed: %v", err)
			}
		}(i)
	}
	wg.Wait()

	// Verify all streams written
	output := buf.String()
	streamCount := strings.Count(output, "stream")
	if streamCount != streams {
		t.Errorf("expected %d streams, got %d", streams, streamCount)
	}
}

// TestCoordinated_SetStatus verifies status updates.
func TestCoordinated_SetStatus(t *testing.T) {
	var buf bytes.Buffer
	printer := NewPrinter(&buf)
	renderer := &FakeRenderer{}
	model := &FakeModel{text: "hello", cursor: 5}

	coord := NewCoordinatedWriter(printer, renderer, model)

	// Set status
	err := coord.SetStatus("typing...")
	if err != nil {
		t.Fatalf("SetStatus failed: %v", err)
	}

	// Verify redraw with status
	redraws := renderer.Redraws()
	if len(redraws) != 1 {
		t.Fatalf("expected 1 redraw, got %d", len(redraws))
	}

	if redraws[0].Status != "typing..." {
		t.Errorf("expected status %q, got %q", "typing...", redraws[0].Status)
	}

	// Clear status
	renderer.Reset()
	err = coord.SetStatus("")
	if err != nil {
		t.Fatalf("SetStatus(\"\") failed: %v", err)
	}

	redraws = renderer.Redraws()
	if len(redraws) != 1 {
		t.Fatalf("expected 1 redraw after clear, got %d", len(redraws))
	}

	if redraws[0].Status != "" {
		t.Errorf("expected empty status, got %q", redraws[0].Status)
	}
}

// TestCoordinated_SetStatus_Persistence verifies status persists across prints.
func TestCoordinated_SetStatus_Persistence(t *testing.T) {
	var buf bytes.Buffer
	printer := NewPrinter(&buf)
	renderer := &FakeRenderer{}
	model := &FakeModel{text: "test", cursor: 4}

	coord := NewCoordinatedWriter(printer, renderer, model)

	// Set status
	coord.SetStatus("loading...")

	// Print line
	renderer.Reset()
	coord.PrintLine("Output line")

	// Verify status persisted in redraw
	redraws := renderer.Redraws()
	if len(redraws) != 1 {
		t.Fatalf("expected 1 redraw, got %d", len(redraws))
	}

	if redraws[0].Status != "loading..." {
		t.Errorf("expected status %q, got %q", "loading...", redraws[0].Status)
	}
}

// TestCoordinated_RedrawPrompt verifies manual redraw.
func TestCoordinated_RedrawPrompt(t *testing.T) {
	var buf bytes.Buffer
	printer := NewPrinter(&buf)
	renderer := &FakeRenderer{}
	model := &FakeModel{text: "prompt", cursor: 6}

	coord := NewCoordinatedWriter(printer, renderer, model)
	coord.SetStatus("test")

	// Manual redraw
	renderer.Reset()
	err := coord.RedrawPrompt()
	if err != nil {
		t.Fatalf("RedrawPrompt failed: %v", err)
	}

	// Verify redraw called with current state
	redraws := renderer.Redraws()
	if len(redraws) != 1 {
		t.Fatalf("expected 1 redraw, got %d", len(redraws))
	}

	if redraws[0].Text != "prompt" || redraws[0].Status != "test" {
		t.Errorf("unexpected redraw: %+v", redraws[0])
	}
}

// TestCoordinated_Integration_InterleavedOperations verifies mixed operations.
func TestCoordinated_Integration_InterleavedOperations(t *testing.T) {
	var buf bytes.Buffer
	printer := NewPrinter(&buf, WithCoalesceDelay(0))
	renderer := &FakeRenderer{}
	model := &FakeModel{text: "prompt", cursor: 6}

	coord := NewCoordinatedWriter(printer, renderer, model)

	// Interleave operations
	coord.PrintLine("Line 1")
	coord.SetStatus("busy")
	coord.PrintLine("Line 2")

	chunks := make(chan string, 2)
	chunks <- "Stream"
	chunks <- "\n"
	close(chunks)
	coord.PrintChunks(context.Background(), chunks)

	coord.SetStatus("")
	coord.RedrawPrompt()

	// Verify output
	output := buf.String()
	// Printer adds \r\n, plus extra \r\n from PrintChunks
	expected := "Line 1\r\nLine 2\r\nStream\r\n\r\n"
	if output != expected {
		t.Errorf("expected output %q, got %q", expected, output)
	}

	// Verify all redraws happened
	redraws := renderer.Redraws()
	if len(redraws) < 5 { // PrintLine(2) + SetStatus(2) + PrintChunks(1) + RedrawPrompt(1) = 6
		t.Errorf("expected at least 5 redraws, got %d", len(redraws))
	}
}

// TestCoordinated_RaceConditions verifies no data races under concurrent access.
func TestCoordinated_RaceConditions(t *testing.T) {
	// This test is designed to be run with -race detector
	var buf bytes.Buffer
	printer := NewPrinter(&buf, WithCoalesceDelay(0))
	renderer := &FakeRenderer{}
	model := &FakeModel{text: "test", cursor: 4}

	coord := NewCoordinatedWriter(printer, renderer, model)

	var wg sync.WaitGroup

	// Concurrent PrintLine
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			coord.PrintLine("line")
		}()
	}

	// Concurrent SetStatus
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			coord.SetStatus("status")
		}(i)
	}

	// Concurrent PrintChunks
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			chunks := make(chan string, 1)
			chunks <- "chunk\n"
			close(chunks)
			coord.PrintChunks(context.Background(), chunks)
		}()
	}

	// Concurrent RedrawPrompt
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			coord.RedrawPrompt()
		}()
	}

	wg.Wait()

	// If we get here without race detector errors, test passes
}

// errorWriter is a writer that always returns an error.
type errorWriter struct {
	err error
}

func (e *errorWriter) Write(p []byte) (int, error) {
	return 0, e.err
}

// TestCoordinated_Performance_CoordinationOverhead measures coordination overhead.
func TestCoordinated_Performance_CoordinationOverhead(t *testing.T) {
	var buf bytes.Buffer
	printer := NewPrinter(&buf)
	renderer := &FakeRenderer{}
	model := &FakeModel{text: "test", cursor: 4}

	coord := NewCoordinatedWriter(printer, renderer, model)

	start := time.Now()
	for i := 0; i < 1000; i++ {
		coord.PrintLine("line")
	}
	duration := time.Since(start)

	// Should complete in reasonable time (< 100ms for 1000 ops)
	if duration > 100*time.Millisecond {
		t.Errorf("coordination overhead too high: %v for 1000 operations", duration)
	}
}
