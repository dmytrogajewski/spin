package output

import (
	"context"
	"fmt"
	"sync"
)

// PromptModel is the interface for accessing prompt state.
// Implemented by prompt.Model.
type PromptModel interface {
	Text() string
	Cursor() int
}

// PromptRenderer is the interface for rendering the prompt.
// Implemented by prompt.Renderer.
type PromptRenderer interface {
	Redraw(model PromptModel, status string) error
}

// ScrollRegionManager is an optional interface for managing scrolling regions.
// If the status renderer implements this, it will be used to move the cursor
// back to the scrolling region after rendering the prompt.
type ScrollRegionManager interface {
	MoveToScrollRegion() error
}

// CoordinatedWriter wraps a Printer and PromptRenderer to ensure
// atomic write-then-redraw operations. All output writes are followed
// by automatic prompt redraws to keep the prompt at the bottom without
// torn output.
//
// CoordinatedWriter is thread-safe and can be used concurrently from
// multiple goroutines.
type CoordinatedWriter struct {
	printer       *Printer
	renderer      PromptRenderer
	model         PromptModel
	scrollManager ScrollRegionManager // Optional: for scrolling region support
	mu            sync.Mutex
	status        string // Current status text (protected by mu)
}

// NewCoordinatedWriter creates a new coordinator that wraps a printer
// and automatically redraws the prompt after each write.
//
// The coordinator takes ownership of prompt rendering coordination
// and ensures no torn output (partial line + prompt interleaved).
//
// Example:
//
//	coord := output.NewCoordinatedWriter(printer, renderer, model)
//	coord.PrintLine("User: Hello!")
//	// Output: "User: Hello!\n> [cursor]"
func NewCoordinatedWriter(
	printer *Printer,
	renderer PromptRenderer,
	model PromptModel,
) *CoordinatedWriter {
	return &CoordinatedWriter{
		printer:  printer,
		renderer: renderer,
		model:    model,
	}
}

// SetScrollManager sets the scroll region manager for coordinating cursor position.
func (c *CoordinatedWriter) SetScrollManager(mgr ScrollRegionManager) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.scrollManager = mgr
}

// PrintLine writes a line and redraws the prompt atomically.
// Thread-safe. Can be called concurrently.
//
// Returns an error if either the write or redraw fails.
// If the printer fails, the redraw is skipped.
// If the redraw fails, the error is returned after the write succeeds.
func (c *CoordinatedWriter) PrintLine(s string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Move to scrolling region BEFORE writing content
	if c.scrollManager != nil {
		_ = c.scrollManager.MoveToScrollRegion()
	}

	// Write line via printer
	if err := c.printer.PrintLine(s); err != nil {
		return fmt.Errorf("print line: %w", err)
	}

	// Redraw prompt
	if err := c.renderer.Redraw(c.model, c.status); err != nil {
		return fmt.Errorf("redraw prompt: %w", err)
	}

	// Move cursor back to scrolling region after redrawing prompt
	if c.scrollManager != nil {
		_ = c.scrollManager.MoveToScrollRegion()
	}

	return nil
}

// PrintChunks streams chunks and redraws the prompt after completion.
// Thread-safe. Can be called concurrently.
// Blocks until the channel closes or context is canceled.
//
// The printer handles internal coalescing and flicker reduction.
// This coordinator redraws the prompt once after the stream completes.
//
// Returns context.Canceled if context is canceled, or any write/redraw error.
func (c *CoordinatedWriter) PrintChunks(ctx context.Context, chunks <-chan string) error {
	// Move to scrolling region BEFORE streaming content
	c.mu.Lock()
	if c.scrollManager != nil {
		_ = c.scrollManager.MoveToScrollRegion()
	}
	c.mu.Unlock()

	// Let printer handle streaming (it has internal coordination)
	err := c.printer.PrintChunks(ctx, chunks)

	// Redraw prompt after stream completes
	c.mu.Lock()
	defer c.mu.Unlock()

	if rerr := c.renderer.Redraw(c.model, c.status); rerr != nil {
		if err == nil {
			err = fmt.Errorf("redraw prompt: %w", rerr)
		}
		// If both printer and renderer failed, return printer error
		// (redraw error is secondary)
	}

	// Move cursor back to scrolling region after redrawing prompt
	if c.scrollManager != nil {
		_ = c.scrollManager.MoveToScrollRegion()
	}

	return err
}

// SetStatus updates the prompt status text and redraws the prompt.
// Thread-safe. Can be called concurrently.
//
// The status persists across multiple print operations until
// explicitly cleared with SetStatus("").
//
// Example:
//
//	coord.SetStatus("typing...")
//	// Prompt shows: "> [cursor]                    typing..."
//
//	coord.SetStatus("")
//	// Prompt shows: "> [cursor]"
func (c *CoordinatedWriter) SetStatus(status string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.status = status
	if err := c.renderer.Redraw(c.model, c.status); err != nil {
		return fmt.Errorf("redraw prompt: %w", err)
	}

	// Move cursor back to scrolling region if manager is set
	if c.scrollManager != nil {
		_ = c.scrollManager.MoveToScrollRegion()
	}

	return nil
}

// RedrawPrompt manually triggers a prompt redraw.
// Thread-safe. Can be called concurrently.
//
// Useful for window resize (SIGWINCH) or explicit refresh.
//
// Example:
//
//	// On SIGWINCH handler:
//	renderer.SetWidth(newWidth)
//	coord.RedrawPrompt()
func (c *CoordinatedWriter) RedrawPrompt() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.renderer.Redraw(c.model, c.status); err != nil {
		return fmt.Errorf("redraw prompt: %w", err)
	}

	// Move cursor back to scrolling region if manager is set
	if c.scrollManager != nil {
		_ = c.scrollManager.MoveToScrollRegion()
	}

	return nil
}
