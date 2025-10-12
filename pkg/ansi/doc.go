// Package ansi provides ANSI escape sequence handling for terminal formatting.
//
// This package supports parsing, stripping, and generating ANSI escape sequences
// for styling terminal output. It provides a fluent API for text styling and
// structured parsing of ANSI-formatted text.
//
// # Basic Usage
//
// Strip ANSI codes:
//
//	text := "\x1b[31mRed text\x1b[0m"
//	plain := ansi.Strip(text)  // "Red text"
//
// Calculate visual length:
//
//	text := "\x1b[1m\x1b[32mBold Green\x1b[0m"
//	len := ansi.Length(text)  // 10
//
// Style text with fluent API:
//
//	styled := ansi.New("Error").Red().Bold().String()
//	// Output: "\x1b[31m\x1b[1mError\x1b[0m"
//
// Parse ANSI text into structured segments:
//
//	text := "\x1b[31mRed\x1b[0m Normal \x1b[1mBold\x1b[0m"
//	segments := ansi.Parse(text)
//	// segments[0]: {Text: "Red", Foreground: "red"}
//	// segments[1]: {Text: " Normal "}
//	// segments[2]: {Text: "Bold", Bold: true}
//
// # Supported Features
//
// - Strip and parse common ANSI SGR (Select Graphic Rendition) codes
// - Basic 16-color palette (30-37 foreground)
// - Text styles: Bold, Dim, Italic, Underline
// - Visual length calculation (UTF-8 aware)
// - Fluent API for text styling
//
// # Limitations
//
// - No 256-color or true-color (RGB) support
// - No cursor positioning (see internal/ui/term)
// - No background colors in parser (reserved for future)
// - Best-effort parsing of malformed sequences
//
// # Performance
//
// All operations are O(n) in input size with minimal allocations:
//   - Strip: <1μs for 1KB text
//   - Length: <1μs for 1KB text
//   - Parse: <10μs for 1KB text
//   - Style.String(): Zero allocations for empty styles
package ansi
