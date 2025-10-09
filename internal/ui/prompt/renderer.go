package prompt

import (
	"fmt"
	"io"
	"strings"

	"github.com/rivo/uniseg"
)

// Renderer renders a prompt model to a terminal using ANSI escape sequences.
// It handles cursor positioning, wide characters, and optional status text.
type Renderer struct {
	out    io.Writer // output destination
	width  int       // terminal width in cells
	prefix string    // prompt prefix (e.g., "> ")
}

// NewRenderer creates a new renderer with the specified output writer,
// terminal width, and prompt prefix.
func NewRenderer(out io.Writer, width int, prefix string) *Renderer {
	return &Renderer{
		out:    out,
		width:  width,
		prefix: prefix,
	}
}

// Redraw renders the prompt model to the output.
// It emits: \r + ClearLine + prefix + buffer + cursor positioning + status.
func (r *Renderer) Redraw(model *Model, status string) error {
	// Get buffer text and cursor position
	bufferText := model.Text()
	cursorRune := model.Cursor()

	// Calculate widths using uniseg
	prefixWidth := uniseg.StringWidth(r.prefix)
	bufferWidth := uniseg.StringWidth(bufferText)
	statusWidth := uniseg.StringWidth(status)

	// Calculate cursor column (1-indexed for ANSI)
	textBeforeCursor := string([]rune(bufferText)[:cursorRune])
	cursorOffset := uniseg.StringWidth(textBeforeCursor)
	cursorCol := prefixWidth + cursorOffset + 1 // +1 for 1-indexed

	// Determine available width for content
	availableWidth := r.width - prefixWidth
	if availableWidth < 0 {
		availableWidth = 0
	}

	// Determine visible content (with or without scrolling)
	var visibleBuffer string
	var scrollOffset int
	if bufferWidth <= availableWidth {
		// No scrolling needed
		visibleBuffer = bufferText
		scrollOffset = 0
	} else {
		// Need horizontal scrolling
		scrollWindowWidth := availableWidth - 2 // reserve for ellipses
		if scrollWindowWidth < 1 {
			scrollWindowWidth = 1
		}

		// Center scroll window around cursor
		scrollStart := cursorOffset - scrollWindowWidth/2
		if scrollStart < 0 {
			scrollStart = 0
		}
		scrollEnd := scrollStart + scrollWindowWidth
		if scrollEnd > bufferWidth {
			scrollEnd = bufferWidth
			scrollStart = scrollEnd - scrollWindowWidth
			if scrollStart < 0 {
				scrollStart = 0
			}
		}

		// Extract visible slice using grapheme boundaries
		visibleBuffer, scrollOffset = extractVisibleSlice(bufferText, scrollStart, scrollEnd)

		// Add ellipses
		if scrollStart > 0 {
			visibleBuffer = "…" + visibleBuffer
			scrollOffset++ // adjust for left ellipsis
		}
		if scrollEnd < bufferWidth {
			visibleBuffer = visibleBuffer + "…"
		}

		// Adjust cursor column for scroll offset
		cursorCol = prefixWidth + cursorOffset - scrollStart + scrollOffset + 1
	}

	// Build output
	var out strings.Builder

	// Clear line and return to start
	out.WriteString("\r\x1b[2K") // \r + ClearLine

	// Write prefix
	out.WriteString(r.prefix)

	// Write visible buffer
	out.WriteString(visibleBuffer)

	// Determine status rendering
	if status != "" && !r.isScrolling(bufferWidth, availableWidth) {
		contentWidth := prefixWidth + bufferWidth
		requiredSpace := contentWidth + 3 + statusWidth // 3 = minimum gap

		if requiredSpace <= r.width {
			// Full status fits
			padding := r.width - contentWidth - statusWidth
			out.WriteString(strings.Repeat(" ", padding))
			out.WriteString(status)
		} else if r.width-contentWidth >= 3 {
			// Truncate status from left
			availableStatusWidth := r.width - contentWidth - 3
			truncatedStatus := truncateLeft(status, availableStatusWidth)
			out.WriteString(strings.Repeat(" ", 3))
			out.WriteString(truncatedStatus)
		}
		// else: omit status entirely
	}

	// Position cursor
	out.WriteString(fmt.Sprintf("\x1b[%dG", cursorCol))

	// Write to output
	_, err := r.out.Write([]byte(out.String()))
	return err
}

// SetWidth updates the terminal width (call on SIGWINCH).
func (r *Renderer) SetWidth(width int) {
	r.width = width
}

// SetPrefix updates the prompt prefix.
func (r *Renderer) SetPrefix(prefix string) {
	r.prefix = prefix
}

// ClearScreen clears the entire screen.
func (r *Renderer) ClearScreen() error {
	// Use ANSI clear screen sequence
	_, err := r.out.Write([]byte("\x1b[2J\x1b[H"))
	return err
}

// isScrolling returns true if buffer requires horizontal scrolling.
func (r *Renderer) isScrolling(bufferWidth, availableWidth int) bool {
	return bufferWidth > availableWidth
}

// extractVisibleSlice extracts a substring from text based on visual width range [start, end).
// Returns the extracted string and the actual start offset in cells.
func extractVisibleSlice(text string, startWidth, endWidth int) (string, int) {
	if startWidth < 0 {
		startWidth = 0
	}
	if endWidth < startWidth {
		endWidth = startWidth
	}

	var result strings.Builder
	currentWidth := 0
	actualStart := 0
	started := false

	graphemes := uniseg.NewGraphemes(text)
	for graphemes.Next() {
		cluster := graphemes.Str()
		clusterWidth := uniseg.StringWidth(cluster)

		if currentWidth >= startWidth && !started {
			started = true
			actualStart = currentWidth
		}

		if started {
			if currentWidth+clusterWidth > endWidth {
				break
			}
			result.WriteString(cluster)
		}

		currentWidth += clusterWidth
	}

	return result.String(), actualStart
}

// truncateLeft truncates a string from the left to fit maxWidth, prepending "…".
func truncateLeft(s string, maxWidth int) string {
	width := uniseg.StringWidth(s)
	if width <= maxWidth {
		return s
	}

	// Reserve 1 cell for ellipsis
	targetWidth := maxWidth - 1
	if targetWidth < 0 {
		return "…"
	}

	// Extract from right
	currentWidth := 0
	var result strings.Builder
	runes := []rune(s)
	for i := len(runes) - 1; i >= 0; i-- {
		r := runes[i]
		rWidth := uniseg.StringWidth(string(r))
		if currentWidth+rWidth > targetWidth {
			break
		}
		currentWidth += rWidth
	}

	// Rebuild from right
	start := len(runes) - 1
	for currentWidth > 0 {
		start--
		if start < 0 {
			break
		}
		currentWidth -= uniseg.StringWidth(string(runes[start]))
	}

	if start < 0 {
		start = 0
	}

	result.WriteString("…")
	result.WriteString(string(runes[start+1:]))
	return result.String()
}
