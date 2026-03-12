// Package ports defines interfaces for UI implementations.
package ports

import (
	"context"

	"github.com/dmytrogajewski/spin/internal/ui/blocks"
)

// UI defines the port interface for TUI implementations.
// Implementations provide a complete terminal user interface with:
// - Lifecycle management (Run/Stop)
// - Append-only output printing
// - Single-line prompt with editing
// - Status display
//
// The UI follows Factory Droid principles:
// - Native terminal scrollback (no alt-screen buffer)
// - Append-only transcript
// - Single-line prompt redraw
// - Zero full-screen repainting.
type UI interface {
	// Lifecycle.

	// Run starts the UI event loop and blocks until context is canceled,
	// user quits (Ctrl-C/Ctrl-D), or an error occurs.
	// Enters raw terminal mode, hides cursor, starts keyboard reader.
	// Returns nil on clean shutdown, context.Canceled on context cancel,
	// or error on failure.
	Run(ctx context.Context) error

	// Stop gracefully shuts down the UI.
	// Exits raw mode, shows cursor, cleans up goroutines.
	// Safe to call multiple times (idempotent).
	Stop() error

	// Output (append-only).

	// PrintLine prints a line to the transcript with newline appended.
	// Automatically redraws prompt after printing.
	// Thread-safe. Returns error if write fails.
	PrintLine(line string) error

	// PrintChunks streams chunks to the transcript.
	// Chunks are coalesced with optional delay to reduce flicker.
	// Automatically redraws prompt after stream completes.
	// Blocks until channel closes or context cancels.
	// Thread-safe. Returns nil on success, context.Err() on cancel,
	// or error if write fails.
	PrintChunks(ctx context.Context, chunks <-chan string) error

	// Prompt control.

	// SetStatus sets transient right-aligned status text in prompt.
	// Automatically redraws prompt.
	// Thread-safe. Returns error if redraw fails.
	SetStatus(text string) error

	// SetMaxTokens sets the maximum token limit for context window percentage calculation.
	// This is used to display context usage (e.g., "45%") in the status bar.
	// Thread-safe.
	SetMaxTokens(maxTokens int64)

	// RequestInput returns a channel that emits user-submitted lines.
	// Channel emits on Enter key, closes on shutdown.
	// Returns the same channel on repeated calls.
	RequestInput() <-chan string

	// Block timeline operations (Phase 6.1).

	// AppendBlock appends a new block to the timeline and triggers render.
	// Thread-safe. Returns error if block validation fails or duplicate ID.
	AppendBlock(block *blocks.Block) error

	// UpdateBlock updates an existing block by ID and triggers render.
	// Thread-safe. Returns ErrBlockNotFound if ID doesn't exist.
	UpdateBlock(blockID string, block *blocks.Block) error

	// DeleteBlock removes a block from the timeline and triggers render.
	// Thread-safe. Returns ErrBlockNotFound if ID doesn't exist.
	DeleteBlock(blockID string) error
}
