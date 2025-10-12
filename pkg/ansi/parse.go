package ansi

import "strings"

// Segment represents a parsed text segment with associated formatting attributes.
//
// Each segment contains visible text and the ANSI formatting that applies to it.
// The Parse function breaks ANSI-formatted text into these structured segments.
//
// Example:
//
//	text := "\x1b[31mError\x1b[0m"
//	segments := ansi.Parse(text)
//	// segments[0] = Segment{Text: "Error", Foreground: "red"}
type Segment struct {
	Text       string // The visible text content
	Foreground string // Foreground color: "red", "green", etc. (empty if no color)
	Background string // Background color (reserved for future use)
	Bold       bool   // True if bold styling is active
	Dim        bool   // True if dim/faint styling is active
	Italic     bool   // True if italic styling is active
	Underline  bool   // True if underline styling is active
}

// Parse parses ANSI-formatted text into structured segments.
//
// The function tracks state changes through ANSI escape sequences and returns
// segments where each segment has uniform formatting. Formatting changes trigger
// new segments.
//
// Supported ANSI codes:
//   - SGR codes (CSI m): colors (30-37), styles (1-4), reset (0)
//   - State is cumulative: applying red then bold results in bold red text
//   - Reset code (0) clears all formatting for subsequent text
//
// Malformed or unknown sequences are handled with best-effort parsing.
//
// Example:
//
//	text := "\x1b[31mRed\x1b[0m Normal \x1b[1mBold\x1b[0m"
//	segments := ansi.Parse(text)
//	// segments[0]: {Text: "Red", Foreground: "red"}
//	// segments[1]: {Text: " Normal "}
//	// segments[2]: {Text: "Bold", Bold: true}
func Parse(text string) []Segment {
	if text == "" {
		return []Segment{}
	}

	segments := make([]Segment, 0)
	current := Segment{}
	inEscape := false
	escapeSeq := ""
	var textBuilder strings.Builder

	for i := 0; i < len(text); i++ {
		ch := text[i]

		// Start of escape sequence
		if ch == '\x1b' {
			// Save current segment if it has text
			if textBuilder.Len() > 0 {
				current.Text = textBuilder.String()
				segments = append(segments, current)
				textBuilder.Reset()
				// Create new segment inheriting formatting
				current = Segment{
					Text:       "",
					Foreground: current.Foreground,
					Background: current.Background,
					Bold:       current.Bold,
					Dim:        current.Dim,
					Italic:     current.Italic,
					Underline:  current.Underline,
				}
			}
			inEscape = true
			escapeSeq = string(ch)
			continue
		}

		// Inside escape sequence
		if inEscape {
			escapeSeq += string(ch)
			// Check if this is a terminator character (a-z, A-Z)
			if isTerminator(ch) {
				// Parse and apply the complete escape sequence
				applyEscapeSequence(escapeSeq, &current)
				inEscape = false
				escapeSeq = ""
			}
			continue
		}

		// Regular character - add to text builder (preserves UTF-8)
		textBuilder.WriteByte(ch)
	}

	// Add final segment if it has text
	if textBuilder.Len() > 0 {
		current.Text = textBuilder.String()
		segments = append(segments, current)
	}

	return segments
}

// isTerminator checks if a character is an ANSI escape sequence terminator.
// For SGR codes, this is any letter (a-z, A-Z).
func isTerminator(ch byte) bool {
	return (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z')
}

// applyEscapeSequence parses an ANSI escape sequence and updates the segment's formatting.
//
// This function handles SGR (Select Graphic Rendition) codes in the format:
//   - ESC [ <params> m
//
// Where <params> can be:
//   - Single code: ESC[31m (red)
//   - Multiple codes: ESC[1;31m (bold + red)
//
// Supported codes:
//   - 0: Reset all
//   - 1: Bold
//   - 2: Dim
//   - 3: Italic
//   - 4: Underline
//   - 30-37: Foreground colors
func applyEscapeSequence(seq string, seg *Segment) {
	// Only process SGR codes (ESC [ ... m)
	if !strings.HasPrefix(seq, "\x1b[") {
		return
	}
	if !strings.HasSuffix(seq, "m") {
		return
	}

	// Extract parameters between ESC[ and m
	params := strings.TrimPrefix(strings.TrimSuffix(seq, "m"), "\x1b[")
	if params == "" {
		return
	}

	// Parse semicolon-separated codes
	codes := strings.Split(params, ";")

	for _, code := range codes {
		switch code {
		case "0":
			// Reset all formatting
			seg.Foreground = ""
			seg.Background = ""
			seg.Bold = false
			seg.Dim = false
			seg.Italic = false
			seg.Underline = false
		case "1":
			seg.Bold = true
		case "2":
			seg.Dim = true
		case "3":
			seg.Italic = true
		case "4":
			seg.Underline = true
		case "30":
			seg.Foreground = "black"
		case "31":
			seg.Foreground = "red"
		case "32":
			seg.Foreground = "green"
		case "33":
			seg.Foreground = "yellow"
		case "34":
			seg.Foreground = "blue"
		case "35":
			seg.Foreground = "magenta"
		case "36":
			seg.Foreground = "cyan"
		case "37":
			seg.Foreground = "white"
		// Note: Background colors (40-47) not implemented yet
		// Unknown codes are silently ignored (best-effort parsing)
		}
	}
}
