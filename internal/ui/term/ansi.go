// Package term provides terminal control primitives for raw mode interaction,
// window size detection, and ANSI escape sequence helpers.
package term

import "fmt"

// ANSI escape sequences for terminal control without alt-screen buffer.
// These are zero-allocation constants for use in hot paths.
const (
	// ClearLine clears the entire current line.
	ClearLine = "\x1b[2K"

	// HideCursor makes the cursor invisible.
	HideCursor = "\x1b[?25l"

	// ShowCursor makes the cursor visible.
	ShowCursor = "\x1b[?25h"

	// SaveCursor saves the current cursor position.
	// Uses DEC save (ESC 7) for broader terminal compatibility.
	SaveCursor = "\x1b7"

	// RestoreCursor restores the previously saved cursor position.
	// Uses DEC restore (ESC 8) for broader terminal compatibility.
	RestoreCursor = "\x1b8"

	// CarriageRet moves cursor to column 0 of current line.
	CarriageRet = "\r"
)

// MoveCursorToCol returns an ANSI escape sequence to move the cursor
// to the specified column (1-indexed) on the current line.
// Column 1 is the leftmost position.
func MoveCursorToCol(col int) string {
	return fmt.Sprintf("\x1b[%dG", col)
}
