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

// WithCoalesceDelay sets the delay for coalescing chunks.
// Setting to 0 disables coalescing and writes chunks immediately.

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

	return p.printChunksWithCoalescing(ctx, chunks)
}

// printChunksWithCoalescing prints chunks with coalescing to reduce flicker.
func (p *Printer) printChunksWithCoalescing(ctx context.Context, chunks <-chan string) error {
	var buf strings.Builder
	var wroteContent bool
	timer := time.NewTimer(p.coalesceDelay)
	defer timer.Stop()

	flush := p.createFlushFunc(&buf, &wroteContent)

	for {
		select {
		case <-ctx.Done():
			flush()
			return ctx.Err()

		case chunk, ok := <-chunks:
			if !ok {
				return p.handleChannelClosed(flush, wroteContent)
			}

			if err := p.handleChunk(chunk, &buf, &wroteContent, flush, timer); err != nil {
				return err
			}

		case <-timer.C:
			if err := flush(); err != nil {
				return err
			}
			timer.Reset(p.coalesceDelay)
		}
	}
}

// createFlushFunc creates a flush function for the buffer.
func (p *Printer) createFlushFunc(buf *strings.Builder, wroteContent *bool) func() error {
	return func() error {
		if buf.Len() == 0 {
			return nil
		}

		*wroteContent = true
		output := strings.ReplaceAll(buf.String(), "\n", "\r\n")

		p.mu.Lock()
		_, err := io.WriteString(p.out, output)
		p.mu.Unlock()

		buf.Reset()
		return err
	}
}

// handleChannelClosed handles the case when the chunks channel is closed.
func (p *Printer) handleChannelClosed(flush func() error, wroteContent bool) error {
	if err := flush(); err != nil {
		return err
	}

	// Ensure output ends with newline to prevent prompt overlap
	// This guarantees the prompt will appear on a fresh line
	// Only add newline if content was actually written
	if wroteContent {
		p.mu.Lock()
		io.WriteString(p.out, "\r\n")
		p.mu.Unlock()
	}
	return nil
}

// handleChunk handles a single chunk.
func (p *Printer) handleChunk(chunk string, buf *strings.Builder, wroteContent *bool, flush func() error, timer *time.Timer) error {
	if len(chunk) > largeChunkThreshold {
		return p.handleLargeChunk(chunk, wroteContent, flush)
	}

	if len(chunk) > 0 {
		*wroteContent = true
	}
	buf.WriteString(chunk)

	if strings.Contains(chunk, "\n") {
		if err := flush(); err != nil {
			return err
		}
		p.resetTimer(timer)
	}

	return nil
}

// handleLargeChunk handles large chunks by writing them immediately.
func (p *Printer) handleLargeChunk(chunk string, wroteContent *bool, flush func() error) error {
	if err := flush(); err != nil {
		return err
	}

	if len(chunk) > 0 {
		*wroteContent = true
	}

	output := strings.ReplaceAll(chunk, "\n", "\r\n")
	p.mu.Lock()
	_, err := io.WriteString(p.out, output)
	p.mu.Unlock()

	return err
}

// resetTimer resets the coalescing timer.
func (p *Printer) resetTimer(timer *time.Timer) {
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
	timer.Reset(p.coalesceDelay)
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
