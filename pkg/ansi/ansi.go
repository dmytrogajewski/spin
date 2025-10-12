package ansi

import "regexp"

// ANSI escape sequence regex for stripping.
// Matches CSI (Control Sequence Introducer) sequences: ESC [ ... letter
// Also matches DEC save/restore cursor sequences: ESC 7 and ESC 8
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]|\x1b[78]`)

// Strip removes all ANSI escape sequences from text.
// This includes CSI sequences (colors, cursor control) and DEC sequences.
//
// Returns plain text without any formatting codes.
// If the input contains no ANSI codes, the original string is returned unchanged.
//
// Example:
//
//	text := "\x1b[31mRed text\x1b[0m"
//	plain := ansi.Strip(text)  // "Red text"
func Strip(text string) string {
	return ansiRegex.ReplaceAllString(text, "")
}

// Length returns the visual length of text, excluding ANSI escape sequences.
// This is useful for calculating proper alignment and text wrapping in terminals.
//
// The length is calculated in runes (Unicode code points), not bytes, making
// it UTF-8 aware and suitable for international text.
//
// Example:
//
//	text := "\x1b[1m\x1b[32mBold Green\x1b[0m"
//	len := ansi.Length(text)  // 10 (visible characters only)
//
//	utf8 := "\x1b[1m你好\x1b[0m"
//	len = ansi.Length(utf8)   // 2 (two Chinese characters)
func Length(text string) int {
	plain := Strip(text)
	return len([]rune(plain))
}

// ANSI color and style constants.
// These are standard SGR (Select Graphic Rendition) codes compatible with
// most modern terminals (xterm, vt100, Windows Terminal, etc.).

// Reset clears all text formatting and returns to default colors and styles.
const Reset = "\x1b[0m"

// Foreground colors (30-37).
const (
	Black   = "\x1b[30m"
	Red     = "\x1b[31m"
	Green   = "\x1b[32m"
	Yellow  = "\x1b[33m"
	Blue    = "\x1b[34m"
	Magenta = "\x1b[35m"
	Cyan    = "\x1b[36m"
	White   = "\x1b[37m"
)

// Text styles (1-4).
const (
	Bold      = "\x1b[1m" // Bright/bold text
	Dim       = "\x1b[2m" // Dimmed/faint text
	Italic    = "\x1b[3m" // Italic text (not universally supported)
	Underline = "\x1b[4m" // Underlined text
)
