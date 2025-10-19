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
	height int       // terminal height in lines (for positioning at bottom)
	prefix string    // prompt prefix (e.g., "> ")
}

// NewRenderer creates a new renderer with the specified output writer,
// terminal width, and prompt prefix.
func NewRenderer(out io.Writer, width int, prefix string) *Renderer {
	return &Renderer{
		out:    out,
		width:  width,
		height: 24, // default height
		prefix: prefix,
	}
}

// Redraw renders the prompt model to the output.
// It emits: \r + ClearLine + prefix + buffer + cursor positioning + status.
func (r *Renderer) Redraw(model *Model, status string) error {
	bufferText := model.Text()
	cursorRune := model.Cursor()

	widths := r.calculateWidths(bufferText, status)
	cursorInfo := r.calculateCursorInfo(bufferText, cursorRune, widths.prefixWidth)
	visibleInfo := r.calculateVisibleContent(bufferText, cursorInfo, widths)

	return r.renderOutput(model, status, visibleInfo, cursorInfo, widths)
}

// widthInfo holds calculated width information.
type widthInfo struct {
	prefixWidth    int
	bufferWidth    int
	statusWidth    int
	availableWidth int
}

// cursorInfo holds cursor position information.
type cursorInfo struct {
	cursorOffset int
	cursorCol    int
}

// visibleInfo holds visible content information.
type visibleInfo struct {
	visibleBuffer string
	scrollOffset  int
}

// calculateWidths calculates all width-related values.
func (r *Renderer) calculateWidths(bufferText, status string) widthInfo {
	prefixWidth := uniseg.StringWidth(r.prefix)
	bufferWidth := uniseg.StringWidth(bufferText)
	statusWidth := uniseg.StringWidth(status)
	availableWidth := r.width - prefixWidth
	if availableWidth < 0 {
		availableWidth = 0
	}

	return widthInfo{
		prefixWidth:    prefixWidth,
		bufferWidth:    bufferWidth,
		statusWidth:    statusWidth,
		availableWidth: availableWidth,
	}
}

// calculateCursorInfo calculates cursor position information.
func (r *Renderer) calculateCursorInfo(bufferText string, cursorRune int, prefixWidth int) cursorInfo {
	textBeforeCursor := string([]rune(bufferText)[:cursorRune])
	cursorOffset := uniseg.StringWidth(textBeforeCursor)
	cursorCol := prefixWidth + cursorOffset + 1 // +1 for 1-indexed

	return cursorInfo{
		cursorOffset: cursorOffset,
		cursorCol:    cursorCol,
	}
}

// calculateVisibleContent calculates visible content with scrolling.
func (r *Renderer) calculateVisibleContent(bufferText string, cursorInfo cursorInfo, widths widthInfo) visibleInfo {
	if widths.bufferWidth <= widths.availableWidth {
		return visibleInfo{
			visibleBuffer: bufferText,
			scrollOffset:  0,
		}
	}

	return r.calculateScrolledContent(bufferText, cursorInfo, widths)
}

// calculateScrolledContent calculates content when scrolling is needed.
func (r *Renderer) calculateScrolledContent(bufferText string, cursorInfo cursorInfo, widths widthInfo) visibleInfo {
	scrollWindowWidth := widths.availableWidth - 2 // reserve for ellipses
	if scrollWindowWidth < 1 {
		scrollWindowWidth = 1
	}

	scrollStart, scrollEnd := r.calculateScrollBounds(cursorInfo.cursorOffset, scrollWindowWidth, widths.bufferWidth)
	visibleBuffer, scrollOffset := extractVisibleSlice(bufferText, scrollStart, scrollEnd)

	visibleBuffer = r.addEllipses(visibleBuffer, scrollStart, scrollEnd, widths.bufferWidth)

	return visibleInfo{
		visibleBuffer: visibleBuffer,
		scrollOffset:  scrollOffset,
	}
}

// calculateScrollBounds calculates scroll window bounds.
func (r *Renderer) calculateScrollBounds(cursorOffset, scrollWindowWidth, bufferWidth int) (int, int) {
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
	return scrollStart, scrollEnd
}

// addEllipses adds ellipses to indicate scrolling.
func (r *Renderer) addEllipses(visibleBuffer string, scrollStart, scrollEnd, bufferWidth int) string {
	if scrollStart > 0 {
		visibleBuffer = "…" + visibleBuffer
	}
	if scrollEnd < bufferWidth {
		visibleBuffer = visibleBuffer + "…"
	}
	return visibleBuffer
}

// renderOutput renders the final output.
func (r *Renderer) renderOutput(model *Model, status string, visibleInfo visibleInfo, cursorInfo cursorInfo, widths widthInfo) error {
	var out strings.Builder

	r.writePromptLine(&out)
	r.writePrefix(&out)
	r.writeVisibleBuffer(&out, visibleInfo.visibleBuffer)
	r.writeStatus(&out, status, widths)

	cursorCol := widths.prefixWidth + cursorInfo.cursorOffset + 1
	out.WriteString(fmt.Sprintf("\x1b[%dG", cursorCol))

	_, err := r.out.Write([]byte(out.String()))
	return err
}

// writePromptLine writes the prompt line positioning.
func (r *Renderer) writePromptLine(out *strings.Builder) {
	if r.height > 0 {
		out.WriteString(fmt.Sprintf("\x1b[%d;1H", r.height))
	} else {
		out.WriteString("\r") // fallback to carriage return
	}
	out.WriteString("\x1b[2K") // Clear the prompt line
}

// writePrefix writes the prompt prefix.
func (r *Renderer) writePrefix(out *strings.Builder) {
	out.WriteString(r.prefix)
}

// writeVisibleBuffer writes the visible buffer content.
func (r *Renderer) writeVisibleBuffer(out *strings.Builder, visibleBuffer string) {
	out.WriteString(visibleBuffer)
}

// writeStatus writes the status text if there's space.
func (r *Renderer) writeStatus(out *strings.Builder, status string, widths widthInfo) {
	if status == "" || r.isScrolling(widths.bufferWidth, widths.availableWidth) {
		return
	}

	contentWidth := widths.prefixWidth + widths.bufferWidth
	requiredSpace := contentWidth + 3 + widths.statusWidth // 3 = minimum gap

	if requiredSpace <= r.width {
		r.writeFullStatus(out, contentWidth, status, widths.statusWidth)
	} else if r.width-contentWidth >= 3 {
		r.writeTruncatedStatus(out, contentWidth, status)
	}
}

// writeFullStatus writes the full status text.
func (r *Renderer) writeFullStatus(out *strings.Builder, contentWidth int, status string, statusWidth int) {
	padding := r.width - contentWidth - statusWidth
	out.WriteString(strings.Repeat(" ", padding))
	out.WriteString(status)
}

// writeTruncatedStatus writes a truncated status text.
func (r *Renderer) writeTruncatedStatus(out *strings.Builder, contentWidth int, status string) {
	availableStatusWidth := r.width - contentWidth - 3
	truncatedStatus := truncateLeft(status, availableStatusWidth)
	out.WriteString(strings.Repeat(" ", 3))
	out.WriteString(truncatedStatus)
}

// SetWidth updates the terminal width (call on SIGWINCH).
func (r *Renderer) SetWidth(width int) {
	r.width = width
}

// SetHeight updates the terminal height (call on SIGWINCH).
func (r *Renderer) SetHeight(height int) {
	r.height = height
}

// SetSize updates both terminal dimensions (call on SIGWINCH).
func (r *Renderer) SetSize(width, height int) {
	r.width = width
	r.height = height
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
