// Package term provides terminal control primitives for raw mode interaction,
// window size detection, and ANSI escape sequence helpers.
package term

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

	// ClearHome homes the cursor, clears the display, and purges scrollback
	// (ED3, same as clear(1)). Used on TUI start and on TUI exit so leftover
	// frames do not remain on the regular screen (no alt-buffer).
	ClearHome = "\x1b[H\x1b[2J\x1b[3J"

	// CursorBlock selects a steady block cursor (DECSCUSR 2).
	CursorBlock = "\x1b[2 q"

	// CursorDefault restores the terminal's default cursor shape.
	CursorDefault = "\x1b[0 q"
)

// MoveCursorToCol returns an ANSI escape sequence that moves the cursor to the specified column.
// Columns are 1-indexed (1 is the leftmost column).
