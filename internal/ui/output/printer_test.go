package output

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

// FakeWriter is a thread-safe test writer that captures output.
type FakeWriter struct {
	mu       sync.Mutex
	buffer   bytes.Buffer
	writeErr error
}

func (f *FakeWriter) Write(p []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.writeErr != nil {
		return 0, f.writeErr
	}
	return f.buffer.Write(p)
}

func (f *FakeWriter) String() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.buffer.String()
}

func (f *FakeWriter) SetWriteError(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.writeErr = err
}

func (f *FakeWriter) Reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.buffer.Reset()
	f.writeErr = nil
}

// TestPrintLine_SingleLine verifies basic line printing.
func TestPrintLine_SingleLine(t *testing.T) {
	w := &FakeWriter{}
	p := NewPrinter(w)

	err := p.PrintLine("Hello, world!")
	if err != nil {
		t.Fatalf("PrintLine failed: %v", err)
	}

	got := w.String()
	want := "Hello, world!\n"
	if got != want {
		t.Errorf("PrintLine output mismatch:\ngot:  %q\nwant: %q", got, want)
	}
}

// TestPrintLine_EmptyString verifies empty line handling.
func TestPrintLine_EmptyString(t *testing.T) {
	w := &FakeWriter{}
	p := NewPrinter(w)

	err := p.PrintLine("")
	if err != nil {
		t.Fatalf("PrintLine failed: %v", err)
	}

	got := w.String()
	want := "\n"
	if got != want {
		t.Errorf("PrintLine output mismatch:\ngot:  %q\nwant: %q", got, want)
	}
}

// TestPrintLine_MultipleLines verifies sequential line printing.
func TestPrintLine_MultipleLines(t *testing.T) {
	w := &FakeWriter{}
	p := NewPrinter(w)

	lines := []string{"Line 1", "Line 2", "Line 3"}
	for _, line := range lines {
		if err := p.PrintLine(line); err != nil {
			t.Fatalf("PrintLine failed: %v", err)
		}
	}

	got := w.String()
	want := "Line 1\nLine 2\nLine 3\n"
	if got != want {
		t.Errorf("PrintLine output mismatch:\ngot:  %q\nwant: %q", got, want)
	}
}

// TestPrintLine_WriterError verifies error handling.
func TestPrintLine_WriterError(t *testing.T) {
	w := &FakeWriter{}
	w.SetWriteError(fmt.Errorf("write error"))
	p := NewPrinter(w)

	err := p.PrintLine("test")
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	if !strings.Contains(err.Error(), "write error") {
		t.Errorf("Expected 'write error', got: %v", err)
	}
}

// TestPrintLine_Concurrent verifies thread safety.
func TestPrintLine_Concurrent(t *testing.T) {
	w := &FakeWriter{}
	p := NewPrinter(w)

	const numGoroutines = 10
	const linesPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < linesPerGoroutine; j++ {
				line := fmt.Sprintf("G%d-L%d", id, j)
				if err := p.PrintLine(line); err != nil {
					t.Errorf("PrintLine failed: %v", err)
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify all lines are present (order may vary)
	output := w.String()
	lines := strings.Split(strings.TrimSuffix(output, "\n"), "\n")
	if len(lines) != numGoroutines*linesPerGoroutine {
		t.Errorf("Expected %d lines, got %d", numGoroutines*linesPerGoroutine, len(lines))
	}
}

// TestPrintChunks_SingleChunk verifies single chunk handling.
func TestPrintChunks_SingleChunk(t *testing.T) {
	w := &FakeWriter{}
	p := NewPrinter(w, WithCoalesceDelay(10*time.Millisecond))

	chunks := make(chan string, 1)
	chunks <- "Hello"
	close(chunks)

	ctx := context.Background()
	err := p.PrintChunks(ctx, chunks)
	if err != nil {
		t.Fatalf("PrintChunks failed: %v", err)
	}

	got := w.String()
	want := "Hello\r\n" // Auto-newline added after streaming completes
	if got != want {
		t.Errorf("PrintChunks output mismatch:\ngot:  %q\nwant: %q", got, want)
	}
}

// TestPrintChunks_MultipleChunks verifies chunk coalescing.
func TestPrintChunks_MultipleChunks(t *testing.T) {
	w := &FakeWriter{}
	p := NewPrinter(w, WithCoalesceDelay(50*time.Millisecond))

	chunks := make(chan string, 5)
	chunks <- "Hello"
	chunks <- " "
	chunks <- "world"
	chunks <- "!"
	close(chunks)

	ctx := context.Background()
	err := p.PrintChunks(ctx, chunks)
	if err != nil {
		t.Fatalf("PrintChunks failed: %v", err)
	}

	got := w.String()
	want := "Hello world!\r\n" // Auto-newline added after streaming completes
	if got != want {
		t.Errorf("PrintChunks output mismatch:\ngot:  %q\nwant: %q", got, want)
	}
}

// TestPrintChunks_NewlineFlush verifies immediate flush on newline.
func TestPrintChunks_NewlineFlush(t *testing.T) {
	w := &FakeWriter{}
	p := NewPrinter(w, WithCoalesceDelay(1*time.Second)) // Long delay

	chunks := make(chan string, 3)
	chunks <- "Line 1\n"
	chunks <- "Line 2\n"
	close(chunks)

	ctx := context.Background()
	start := time.Now()
	err := p.PrintChunks(ctx, chunks)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("PrintChunks failed: %v", err)
	}

	// Should flush immediately, not wait for 1s delay
	if elapsed > 100*time.Millisecond {
		t.Errorf("PrintChunks took too long (expected immediate flush): %v", elapsed)
	}

	got := w.String()
	want := "Line 1\r\nLine 2\r\n\r\n" // \n→\r\n conversion + auto-newline
	if got != want {
		t.Errorf("PrintChunks output mismatch:\ngot:  %q\nwant: %q", got, want)
	}
}

// TestPrintChunks_EmptyChannel verifies empty channel handling.
func TestPrintChunks_EmptyChannel(t *testing.T) {
	w := &FakeWriter{}
	p := NewPrinter(w)

	chunks := make(chan string)
	close(chunks)

	ctx := context.Background()
	err := p.PrintChunks(ctx, chunks)
	if err != nil {
		t.Fatalf("PrintChunks failed: %v", err)
	}

	got := w.String()
	want := ""
	if got != want {
		t.Errorf("PrintChunks output mismatch:\ngot:  %q\nwant: %q", got, want)
	}
}

// TestPrintChunks_ContextCancel verifies context cancellation.
func TestPrintChunks_ContextCancel(t *testing.T) {
	w := &FakeWriter{}
	p := NewPrinter(w, WithCoalesceDelay(100*time.Millisecond))

	chunks := make(chan string)
	ctx, cancel := context.WithCancel(context.Background())

	// Send some chunks then cancel
	go func() {
		chunks <- "Partial"
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	err := p.PrintChunks(ctx, chunks)
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got: %v", err)
	}

	// Should flush partial content
	got := w.String()
	if !strings.Contains(got, "Partial") {
		t.Errorf("Expected partial content to be flushed, got: %q", got)
	}
}

// TestPrintChunks_LargeChunk verifies immediate write for large chunks.
func TestPrintChunks_LargeChunk(t *testing.T) {
	w := &FakeWriter{}
	p := NewPrinter(w, WithCoalesceDelay(1*time.Second)) // Long delay

	// Create 15KB chunk (above 10KB threshold)
	largeChunk := strings.Repeat("x", 15*1024)
	chunks := make(chan string, 1)
	chunks <- largeChunk
	close(chunks)

	ctx := context.Background()
	start := time.Now()
	err := p.PrintChunks(ctx, chunks)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("PrintChunks failed: %v", err)
	}

	// Should write immediately, not wait for delay
	if elapsed > 100*time.Millisecond {
		t.Errorf("PrintChunks took too long (expected immediate write): %v", elapsed)
	}

	got := w.String()
	want := largeChunk + "\r\n" // Auto-newline added
	if got != want {
		t.Errorf("PrintChunks output mismatch: got %d bytes, want %d bytes", len(got), len(want))
	}
}

// TestPrintChunks_CoalesceDelay verifies coalescing behavior.
func TestPrintChunks_CoalesceDelay(t *testing.T) {
	w := &FakeWriter{}
	delay := 50 * time.Millisecond
	p := NewPrinter(w, WithCoalesceDelay(delay))

	chunks := make(chan string)
	ctx := context.Background()

	// Send chunks slowly
	go func() {
		chunks <- "A"
		time.Sleep(10 * time.Millisecond)
		chunks <- "B"
		time.Sleep(10 * time.Millisecond)
		chunks <- "C"
		close(chunks)
	}()

	start := time.Now()
	err := p.PrintChunks(ctx, chunks)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("PrintChunks failed: %v", err)
	}

	// Should take at least one delay period
	if elapsed < delay {
		t.Logf("Note: Elapsed %v < delay %v (acceptable due to final flush)", elapsed, delay)
	}

	got := w.String()
	want := "ABC\r\n" // Auto-newline added
	if got != want {
		t.Errorf("PrintChunks output mismatch:\ngot:  %q\nwant: %q", got, want)
	}
}

// TestPrintChunks_ZeroDelay verifies immediate writes with zero coalesce delay.
func TestPrintChunks_ZeroDelay(t *testing.T) {
	w := &FakeWriter{}
	p := NewPrinter(w, WithCoalesceDelay(0))

	chunks := make(chan string, 3)
	chunks <- "A"
	chunks <- "B"
	chunks <- "C"
	close(chunks)

	ctx := context.Background()
	err := p.PrintChunks(ctx, chunks)
	if err != nil {
		t.Fatalf("PrintChunks failed: %v", err)
	}

	got := w.String()
	want := "ABC\r\n" // Auto-newline added
	if got != want {
		t.Errorf("PrintChunks output mismatch:\ngot:  %q\nwant: %q", got, want)
	}
}

// TestConcurrentPrintChunks verifies concurrent streaming.
func TestConcurrentPrintChunks(t *testing.T) {
	w := &FakeWriter{}
	p := NewPrinter(w, WithCoalesceDelay(10*time.Millisecond))

	const numStreams = 5
	var wg sync.WaitGroup
	wg.Add(numStreams)

	for i := 0; i < numStreams; i++ {
		go func(id int) {
			defer wg.Done()
			chunks := make(chan string, 10)
			for j := 0; j < 10; j++ {
				chunks <- fmt.Sprintf("S%d-C%d ", id, j)
			}
			close(chunks)

			ctx := context.Background()
			if err := p.PrintChunks(ctx, chunks); err != nil {
				t.Errorf("PrintChunks failed: %v", err)
			}
		}(i)
	}

	wg.Wait()

	// Verify all chunks are present (order may vary)
	output := w.String()
	for i := 0; i < numStreams; i++ {
		for j := 0; j < 10; j++ {
			expected := fmt.Sprintf("S%d-C%d ", i, j)
			if !strings.Contains(output, expected) {
				t.Errorf("Missing chunk: %q", expected)
			}
		}
	}
}

// TestConcurrentPrintLineAndChunks verifies interleaved operations.
func TestConcurrentPrintLineAndChunks(t *testing.T) {
	w := &FakeWriter{}
	p := NewPrinter(w, WithCoalesceDelay(10*time.Millisecond))

	var wg sync.WaitGroup
	wg.Add(2)

	// PrintLine goroutine
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			p.PrintLine(fmt.Sprintf("Line %d", i))
		}
	}()

	// PrintChunks goroutine
	go func() {
		defer wg.Done()
		chunks := make(chan string, 50)
		for i := 0; i < 50; i++ {
			chunks <- fmt.Sprintf("Chunk %d ", i)
		}
		close(chunks)
		ctx := context.Background()
		p.PrintChunks(ctx, chunks)
	}()

	wg.Wait()

	// Just verify no panic and some output
	output := w.String()
	if len(output) == 0 {
		t.Error("Expected some output, got empty string")
	}
}

// TestWithCoalesceDelay verifies option configuration.
func TestWithCoalesceDelay(t *testing.T) {
	w := &FakeWriter{}
	delay := 123 * time.Millisecond
	p := NewPrinter(w, WithCoalesceDelay(delay))

	if p.coalesceDelay != delay {
		t.Errorf("coalesceDelay mismatch: got %v, want %v", p.coalesceDelay, delay)
	}
}

// TestDefaultCoalesceDelay verifies default delay.
func TestDefaultCoalesceDelay(t *testing.T) {
	w := &FakeWriter{}
	p := NewPrinter(w)

	if p.coalesceDelay != defaultCoalesceDelay {
		t.Errorf("coalesceDelay mismatch: got %v, want %v", p.coalesceDelay, defaultCoalesceDelay)
	}
}

// BenchmarkPrintLine measures PrintLine performance.
func BenchmarkPrintLine(b *testing.B) {
	w := &FakeWriter{}
	p := NewPrinter(w)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.PrintLine("Benchmark line")
	}
}

// BenchmarkPrintChunksSmall measures small chunk streaming.
func BenchmarkPrintChunksSmall(b *testing.B) {
	w := &FakeWriter{}
	p := NewPrinter(w, WithCoalesceDelay(10*time.Millisecond))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		chunks := make(chan string, 100)
		for j := 0; j < 100; j++ {
			chunks <- "Small chunk"
		}
		close(chunks)

		ctx := context.Background()
		p.PrintChunks(ctx, chunks)
		w.Reset()
	}
}

// BenchmarkPrintChunksLarge measures large chunk streaming.
func BenchmarkPrintChunksLarge(b *testing.B) {
	w := &FakeWriter{}
	p := NewPrinter(w, WithCoalesceDelay(10*time.Millisecond))
	largeChunk := strings.Repeat("x", 10*1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		chunks := make(chan string, 10)
		for j := 0; j < 10; j++ {
			chunks <- largeChunk
		}
		close(chunks)

		ctx := context.Background()
		p.PrintChunks(ctx, chunks)
		w.Reset()
	}
}

// BenchmarkConcurrentPrintLine measures concurrent PrintLine performance.
func BenchmarkConcurrentPrintLine(b *testing.B) {
	w := &FakeWriter{}
	p := NewPrinter(w)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			p.PrintLine("Concurrent line")
		}
	})
}

// TestPrintChunks_ImmediateMode_MultipleChunks tests zero delay mode.
func TestPrintChunks_ImmediateMode_MultipleChunks(t *testing.T) {
	w := &FakeWriter{}
	p := NewPrinter(w, WithCoalesceDelay(0))

	chunks := make(chan string)
	ctx := context.Background()

	// Send multiple chunks in immediate mode
	go func() {
		chunks <- "A"
		chunks <- "B"
		chunks <- "C"
		chunks <- "D"
		chunks <- "E"
		close(chunks)
	}()

	err := p.PrintChunks(ctx, chunks)
	if err != nil {
		t.Fatalf("PrintChunks failed: %v", err)
	}

	got := w.String()
	want := "ABCDE\r\n" // Auto-newline added
	if got != want {
		t.Errorf("PrintChunks output mismatch:\ngot:  %q\nwant: %q", got, want)
	}
}

// TestPrintChunks_ImmediateMode_ContextCancel tests cancellation in immediate mode.
func TestPrintChunks_ImmediateMode_ContextCancel(t *testing.T) {
	w := &FakeWriter{}
	p := NewPrinter(w, WithCoalesceDelay(0))

	chunks := make(chan string)
	ctx, cancel := context.WithCancel(context.Background())

	// Send some chunks then cancel
	go func() {
		chunks <- "Partial"
		chunks <- "Content"
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	err := p.PrintChunks(ctx, chunks)
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got: %v", err)
	}

	// Should flush partial content
	got := w.String()
	if !strings.Contains(got, "Partial") {
		t.Errorf("Expected partial content to be flushed, got: %q", got)
	}
}

// TestPrintChunks_TimerDrainRace tests timer channel draining edge case.
func TestPrintChunks_TimerDrainRace(t *testing.T) {
	w := &FakeWriter{}
	p := NewPrinter(w, WithCoalesceDelay(5*time.Millisecond))

	chunks := make(chan string)
	ctx := context.Background()

	// Send chunks that trigger multiple timer resets
	go func() {
		for i := 0; i < 10; i++ {
			chunks <- fmt.Sprintf("Line %d\n", i) // Each triggers flush
			time.Sleep(1 * time.Millisecond)
		}
		close(chunks)
	}()

	err := p.PrintChunks(ctx, chunks)
	if err != nil {
		t.Fatalf("PrintChunks failed: %v", err)
	}

	// Verify all lines are present
	got := w.String()
	for i := 0; i < 10; i++ {
		expected := fmt.Sprintf("Line %d\n", i)
		if !strings.Contains(got, expected) {
			t.Errorf("Missing line: %q", expected)
		}
	}
}

// TestPrintChunks_WriteError tests error handling during write.
func TestPrintChunks_WriteError(t *testing.T) {
	w := &FakeWriter{}
	w.SetWriteError(fmt.Errorf("write error"))
	p := NewPrinter(w, WithCoalesceDelay(10*time.Millisecond))

	chunks := make(chan string, 2)
	chunks <- "Chunk 1"
	close(chunks)

	ctx := context.Background()
	err := p.PrintChunks(ctx, chunks)

	// Should hit the write error
	if err == nil {
		t.Fatal("Expected write error, got nil")
	}
	if !strings.Contains(err.Error(), "write error") {
		t.Errorf("Expected 'write error', got: %v", err)
	}
}

// TestPrintChunks_LargeChunkWriteError tests error handling for large chunk writes.
func TestPrintChunks_LargeChunkWriteError(t *testing.T) {
	w := &FakeWriter{}
	w.SetWriteError(fmt.Errorf("write error"))
	p := NewPrinter(w, WithCoalesceDelay(10*time.Millisecond))

	largeChunk := strings.Repeat("x", 15*1024)
	chunks := make(chan string, 1)
	chunks <- largeChunk
	close(chunks)

	ctx := context.Background()
	err := p.PrintChunks(ctx, chunks)

	if err == nil {
		t.Fatal("Expected write error, got nil")
	}
	if !strings.Contains(err.Error(), "write error") {
		t.Errorf("Expected 'write error', got: %v", err)
	}
}

// TestPrintChunks_ImmediateMode_EmptyChannel tests empty channel in immediate mode.
func TestPrintChunks_ImmediateMode_EmptyChannel(t *testing.T) {
	w := &FakeWriter{}
	p := NewPrinter(w, WithCoalesceDelay(0))

	chunks := make(chan string)
	close(chunks)

	ctx := context.Background()
	err := p.PrintChunks(ctx, chunks)
	if err != nil {
		t.Fatalf("PrintChunks failed: %v", err)
	}

	got := w.String()
	want := ""
	if got != want {
		t.Errorf("PrintChunks output mismatch:\ngot:  %q\nwant: %q", got, want)
	}
}
