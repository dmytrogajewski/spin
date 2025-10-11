package output_test

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/dmytrogajewski/spin/internal/ui/output"
)

// BenchmarkPrinterPrintLine measures single line printing performance.
func BenchmarkPrinterPrintLine(b *testing.B) {
	printer := output.NewPrinter(io.Discard)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = printer.PrintLine("User: What is the meaning of life?")
	}
}

// BenchmarkPrinterPrintLine_Long measures long line printing performance.
func BenchmarkPrinterPrintLine_Long(b *testing.B) {
	printer := output.NewPrinter(io.Discard)
	longLine := strings.Repeat("A very long line with lots of text. ", 100) // ~3.7KB

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = printer.PrintLine(longLine)
	}
}

// BenchmarkPrinterPrintChunks_Small measures small chunk streaming (LLM-like).
func BenchmarkPrinterPrintChunks_Small(b *testing.B) {
	printer := output.NewPrinter(io.Discard)

	// Simulate LLM token stream: ~50 bytes per chunk
	tokens := []string{
		"The ", "answer ", "to ", "life, ", "the ", "universe, ",
		"and ", "everything ", "is ", "42", ".\n",
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		chunks := make(chan string, len(tokens))
		for _, token := range tokens {
			chunks <- token
		}
		close(chunks)

		_ = printer.PrintChunks(context.Background(), chunks)
	}

	b.ReportMetric(float64(len(tokens)*b.N), "chunks")
}

// BenchmarkPrinterPrintChunks_100k measures throughput for 100k chunks.
func BenchmarkPrinterPrintChunks_100k(b *testing.B) {
	printer := output.NewPrinter(io.Discard, output.WithCoalesceDelay(0)) // Disable coalescing for pure throughput

	// Generate 100k chunks
	const chunkCount = 100000
	const chunkSize = 50 // Average token size

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		chunks := make(chan string, 1000)

		// Producer goroutine
		go func() {
			defer close(chunks)
			for j := 0; j < chunkCount; j++ {
				chunks <- "Lorem ipsum dolor sit amet, consectetur adipiscing"
			}
		}()

		_ = printer.PrintChunks(context.Background(), chunks)
	}

	totalBytes := float64(chunkCount * chunkSize * b.N)
	b.ReportMetric(totalBytes/1024/1024, "MB")
	b.ReportMetric(float64(chunkCount*b.N)/b.Elapsed().Seconds(), "chunks/sec")
}

// BenchmarkPrinterPrintChunks_LargeChunks measures large chunk bypass performance.
func BenchmarkPrinterPrintChunks_LargeChunks(b *testing.B) {
	printer := output.NewPrinter(io.Discard)

	// Simulate file dump: 1000 chunks of 10KB each
	const chunkCount = 1000
	const chunkSize = 10 * 1024 // 10KB

	largeChunk := strings.Repeat("A", chunkSize)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		chunks := make(chan string, 100)

		go func() {
			defer close(chunks)
			for j := 0; j < chunkCount; j++ {
				chunks <- largeChunk
			}
		}()

		_ = printer.PrintChunks(context.Background(), chunks)
	}

	totalMB := float64(chunkCount * chunkSize * b.N) / 1024 / 1024
	b.ReportMetric(totalMB, "MB")
	b.ReportMetric(totalMB/b.Elapsed().Seconds(), "MB/sec")
}

// BenchmarkPrinterPrintChunks_Newlines measures newline fast-path performance.
func BenchmarkPrinterPrintChunks_Newlines(b *testing.B) {
	printer := output.NewPrinter(io.Discard)

	// Stream with many newlines (triggers fast-path flush)
	lines := make([]string, 1000)
	for i := range lines {
		lines[i] = "This is a complete line of text\n"
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		chunks := make(chan string, len(lines))
		for _, line := range lines {
			chunks <- line
		}
		close(chunks)

		_ = printer.PrintChunks(context.Background(), chunks)
	}

	b.ReportMetric(float64(len(lines)*b.N), "lines")
}

// BenchmarkPrinterConcurrentPrintLine measures concurrent line printing.
func BenchmarkPrinterConcurrentPrintLine(b *testing.B) {
	printer := output.NewPrinter(io.Discard)

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			_ = printer.PrintLine("Concurrent message from goroutine")
			i++
		}
	})
}

// BenchmarkPrinterPrintChunks_Coalescing measures coalescing effectiveness.
func BenchmarkPrinterPrintChunks_Coalescing(b *testing.B) {
	// Create printer with realistic coalesce delay
	printer := output.NewPrinter(io.Discard, output.WithCoalesceDelay(50_000_000)) // 50ms

	// Small chunks that should coalesce
	const chunkCount = 10000
	chunks := make([]string, chunkCount)
	for i := range chunks {
		chunks[i] = "Token" // 5 bytes
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ch := make(chan string, 100)

		go func() {
			defer close(ch)
			for _, chunk := range chunks {
				ch <- chunk
			}
		}()

		_ = printer.PrintChunks(context.Background(), ch)
	}

	totalBytes := float64(len(chunks) * 5 * b.N)
	b.ReportMetric(totalBytes/1024, "KB")
}

// BenchmarkPrinterPrintChunks_Mixed measures mixed chunk sizes (realistic scenario).
func BenchmarkPrinterPrintChunks_Mixed(b *testing.B) {
	printer := output.NewPrinter(io.Discard)

	// Mix of small tokens, medium lines, and occasional large chunks
	chunks := []string{
		"Small", " token", " stream", "\n",
		"Medium length line with some content about 80 chars or so for realism here\n",
		"Small", " tokens", " again", "\n",
		strings.Repeat("Large chunk simulating file content or log dump. ", 200), // ~10KB
		"Back", " to", " small", " tokens", "\n",
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ch := make(chan string, len(chunks))
		for _, chunk := range chunks {
			ch <- chunk
		}
		close(ch)

		_ = printer.PrintChunks(context.Background(), ch)
	}
}

// BenchmarkPrinterAllocation measures allocation overhead.
func BenchmarkPrinterAllocation(b *testing.B) {
	printer := output.NewPrinter(io.Discard)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = printer.PrintLine("Test message for allocation measurement")
	}
}

// BenchmarkCoordinatedWriter_PrintLine measures coordinated output + redraw.
func BenchmarkCoordinatedWriter_PrintLine(b *testing.B) {
	// Create fake renderer and model
	renderer := &fakeRenderer{}
	model := &fakeModel{}
	printer := output.NewPrinter(io.Discard)

	coord := output.NewCoordinatedWriter(printer, renderer, model)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = coord.PrintLine("User: Hello!")
	}
}

// BenchmarkCoordinatedWriter_SetStatus measures status update performance.
func BenchmarkCoordinatedWriter_SetStatus(b *testing.B) {
	renderer := &fakeRenderer{}
	model := &fakeModel{}
	printer := output.NewPrinter(io.Discard)

	coord := output.NewCoordinatedWriter(printer, renderer, model)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = coord.SetStatus("thinking...")
		_ = coord.SetStatus("")
	}
}

// Fake renderer for benchmarking
type fakeRenderer struct{}

func (f *fakeRenderer) Redraw(model output.PromptModel, status string) error {
	return nil
}

// Fake model for benchmarking
type fakeModel struct{}

func (f *fakeModel) Text() string {
	return "> "
}

func (f *fakeModel) Cursor() int {
	return 2
}
