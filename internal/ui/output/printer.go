// Package output provides append-only output capabilities for the TUI.
// It supports both immediate line printing and streaming chunks with
// optional coalescing to reduce flicker.
package output

import (
	"context"
	"io"
	"strings"
	"sync"
	"time"
)

const (
	// defaultCoalesceDelay is the default delay for coalescing chunks.
	defaultCoalesceDelay = 50 * time.Millisecond

	// largeChunkThreshold is the size above which chunks are written immediately.
	largeChunkThreshold = 10 * 1024 // 10KB
)

// Printer handles append-only output to stdout for chat transcript.
// It supports both immediate line printing and streaming chunks with
// optional coalescing to reduce flicker.
//
// Printer is thread-safe and can be used concurrently from multiple goroutines.
type Printer struct {
	out           io.Writer
	mu            sync.Mutex
	coalesceDelay time.Duration
}

// PrinterOption is a functional option for configuring the Printer.
type PrinterOption func(*Printer)

// WithCoalesceDelay sets the coalescing delay for streaming chunks.
// A delay of 0 means no coalescing (immediate writes).
// Default is 50ms.
func WithCoalesceDelay(d time.Duration) PrinterOption {
	return func(p *Printer) {
		p.coalesceDelay = d
	}
}

// NewPrinter creates a new Printer with optional configuration.
// The printer writes to the provided io.Writer.
//
// Example:
//
//	printer := output.NewPrinter(os.Stdout)
//	printer.PrintLine("Hello, world!")
func NewPrinter(out io.Writer, opts ...PrinterOption) *Printer {
	p := &Printer{
		out:           out,
		coalesceDelay: defaultCoalesceDelay,
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

// PrintLine writes a line immediately with a newline.
// Thread-safe. Can be called concurrently.
//
// Returns an error if the write fails.
func (p *Printer) PrintLine(s string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Convert \n to \r\n for raw terminal mode
	output := strings.ReplaceAll(s+"\n", "\n", "\r\n")
	_, err := io.WriteString(p.out, output)
	return err
}

// PrintChunks streams chunks from a channel with optional coalescing.
// Flushes immediately on newline or after coalesceDelay.
// Large chunks (>10KB) are written immediately without buffering.
//
// Blocks until the channel closes or context is canceled.
// Thread-safe. Can be called concurrently.
//
// Returns context.Canceled if context is canceled, or any write error.
func (p *Printer) PrintChunks(ctx context.Context, chunks <-chan string) error {
	if p.coalesceDelay == 0 {
		return p.printChunksImmediate(ctx, chunks)
	}

	var buf strings.Builder
	var wroteContent bool // Track if any content was written
	timer := time.NewTimer(p.coalesceDelay)
	defer timer.Stop()

	flush := func() error {
		if buf.Len() == 0 {
			return nil
		}

		// Mark that we wrote content
		wroteContent = true

		// Convert \n to \r\n for raw terminal mode
		output := strings.ReplaceAll(buf.String(), "\n", "\r\n")

		p.mu.Lock()
		_, err := io.WriteString(p.out, output)
		p.mu.Unlock()

		buf.Reset()
		return err
	}

	for {
		select {
		case <-ctx.Done():
			// Flush remaining buffer before returning
			flush()
			return ctx.Err()

		case chunk, ok := <-chunks:
			if !ok {
				// Channel closed, flush remaining and ensure final newline
				if err := flush(); err != nil {
					return err
				}
				// Always ensure output ends with newline to prevent prompt overlap
				// This guarantees the prompt will appear on a fresh line
				if wroteContent {
					p.mu.Lock()
					io.WriteString(p.out, "\r\n")
					p.mu.Unlock()
				}
				return nil
			}

			// Large chunks: write immediately
			if len(chunk) > largeChunkThreshold {
				if err := flush(); err != nil {
					return err
				}
				// Mark that we wrote content
				if len(chunk) > 0 {
					wroteContent = true
				}
				// Convert \n to \r\n for raw terminal mode
				output := strings.ReplaceAll(chunk, "\n", "\r\n")
				p.mu.Lock()
				_, err := io.WriteString(p.out, output)
				p.mu.Unlock()
				if err != nil {
					return err
				}
				continue
			}

			// Append to buffer
			buf.WriteString(chunk)

			// Fast-path: flush immediately on newline
			if strings.Contains(chunk, "\n") {
				if err := flush(); err != nil {
					return err
				}
				// Reset timer
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				timer.Reset(p.coalesceDelay)
			}

		case <-timer.C:
			// Timer fired, flush buffer
			if err := flush(); err != nil {
				return err
			}
			timer.Reset(p.coalesceDelay)
		}
	}
}

// printChunksImmediate writes chunks immediately without coalescing.
// Used when coalesceDelay is 0.
func (p *Printer) printChunksImmediate(ctx context.Context, chunks <-chan string) error {
	var buf strings.Builder
	var wroteContent bool // Track if any content was written

	for {
		select {
		case <-ctx.Done():
			// Flush remaining buffer
			if buf.Len() > 0 {
				str := buf.String()
				output := strings.ReplaceAll(str, "\n", "\r\n")
				wroteContent = true
				p.mu.Lock()
				io.WriteString(p.out, output)
				p.mu.Unlock()
			}
			return ctx.Err()

		case chunk, ok := <-chunks:
			if !ok {
				// Channel closed, flush and ensure final newline
				if buf.Len() > 0 {
					str := buf.String()
					output := strings.ReplaceAll(str, "\n", "\r\n")
					wroteContent = true
					p.mu.Lock()
					_, err := io.WriteString(p.out, output)
					p.mu.Unlock()
					if err != nil {
						return err
					}
				}
				// Always ensure output ends with newline to prevent prompt overlap
				// This guarantees the prompt will appear on a fresh line
				if wroteContent {
					p.mu.Lock()
					io.WriteString(p.out, "\r\n")
					p.mu.Unlock()
				}
				return nil
			}

			buf.WriteString(chunk)
		}
	}
}
