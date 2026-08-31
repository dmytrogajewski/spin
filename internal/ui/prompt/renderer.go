package prompt

import (
	"fmt"
	"io"
	"strings"

	"github.com/rivo/uniseg"

	"github.com/dmytrogajewski/spin/pkg/ui/textwidth"
)

const (
	defaultTermHeight    = 24
	minStatusGap         = 3 // minimum gap between content and status.
	scrollEllipsisWidth  = 2 // space reserved for scroll indicator ellipses.
	minScrollWindowWidth = 1 // minimum width of scroll window.
)

// TermRenderer renders a prompt model to a terminal using ANSI escape sequences.
// It handles cursor positioning, wide characters, and optional status text.
type TermRenderer struct {
	out      io.Writer // output destination.
	width    int       // terminal width in cells.
	height   int       // terminal height in lines (for positioning at bottom).
	prefix   string    // prompt prefix (e.g., "→ ").
	inputBar bool      // paint a full-width grey background on the prompt line.
	hints    []string  // completion rows drawn above the prompt.
	hintSel  int
	hintPrev int // prior hint row count so leftovers can be cleared.
}

// NewTermRenderer creates a new renderer with the specified output writer,
// terminal width, and prompt prefix.
func NewTermRenderer(out io.Writer, width int, prefix string) *TermRenderer {
	return &TermRenderer{
		out:    out,
		width:  width,
		height: defaultTermHeight, // default height.
		prefix: prefix,
	}
}

// Redraw renders the prompt model to the output.
// It emits: \r + ClearLine + prefix + buffer + cursor positioning + status.
func (r *TermRenderer) Redraw(model *Model, status string) error {
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
func (r *TermRenderer) calculateWidths(bufferText, status string) widthInfo {
	prefixWidth := r.barInset() + uniseg.StringWidth(r.prefix)
	bufferWidth := uniseg.StringWidth(bufferText)
	statusWidth := uniseg.StringWidth(status)

	availableWidth := max(r.width-prefixWidth-r.barInset()-r.caretReserve(), 0)

	return widthInfo{
		prefixWidth:    prefixWidth,
		bufferWidth:    bufferWidth,
		statusWidth:    statusWidth,
		availableWidth: availableWidth,
	}
}

func (r *TermRenderer) barInset() int {
	if r.inputBar {
		return InputBarPad
	}

	return 0
}

func (r *TermRenderer) caretReserve() int {
	if r.inputBar {
		return 1
	}

	return 0
}

// calculateCursorInfo calculates cursor position information.
func (r *TermRenderer) calculateCursorInfo(bufferText string, cursorRune, prefixWidth int) cursorInfo {
	textBeforeCursor := string([]rune(bufferText)[:cursorRune])
	cursorOffset := uniseg.StringWidth(textBeforeCursor)
	cursorCol := prefixWidth + cursorOffset + 1 // +1 for 1-indexed.

	return cursorInfo{
		cursorOffset: cursorOffset,
		cursorCol:    cursorCol,
	}
}

// calculateVisibleContent calculates visible content with scrolling.
func (r *TermRenderer) calculateVisibleContent(bufferText string, cursorInfo cursorInfo, widths widthInfo) visibleInfo {
	if widths.bufferWidth <= widths.availableWidth {
		return visibleInfo{
			visibleBuffer: bufferText,
			scrollOffset:  0,
		}
	}

	return r.calculateScrolledContent(bufferText, cursorInfo, widths)
}

// calculateScrolledContent calculates content when scrolling is needed.
func (r *TermRenderer) calculateScrolledContent(bufferText string, cursorInfo cursorInfo, widths widthInfo) visibleInfo {
	scrollWindowWidth := max(
		widths.availableWidth-scrollEllipsisWidth, minScrollWindowWidth)

	scrollStart, scrollEnd := r.calculateScrollBounds(cursorInfo.cursorOffset, scrollWindowWidth, widths.bufferWidth)
	visibleBuffer, scrollOffset := extractVisibleSlice(bufferText, scrollStart, scrollEnd)

	visibleBuffer = r.addEllipses(visibleBuffer, scrollStart, scrollEnd, widths.bufferWidth)

	return visibleInfo{
		visibleBuffer: visibleBuffer,
		scrollOffset:  scrollOffset,
	}
}

// calculateScrollBounds calculates scroll window bounds.
func (r *TermRenderer) calculateScrollBounds(cursorOffset, scrollWindowWidth, bufferWidth int) (scrollStart, scrollEnd int) {
	scrollStart = max(cursorOffset-scrollWindowWidth/2, 0)

	scrollEnd = scrollStart + scrollWindowWidth
	if scrollEnd > bufferWidth {
		scrollEnd = bufferWidth

		scrollStart = max(scrollEnd-scrollWindowWidth, 0)
	}

	return scrollStart, scrollEnd
}

// addEllipses adds ellipses to indicate scrolling.
func (r *TermRenderer) addEllipses(visibleBuffer string, scrollStart, scrollEnd, bufferWidth int) string {
	if scrollStart > 0 {
		visibleBuffer = "…" + visibleBuffer
	}

	if scrollEnd < bufferWidth {
		visibleBuffer += "…"
	}

	return visibleBuffer
}

// renderOutput renders the final output.
func (r *TermRenderer) renderOutput(model *Model, status string, visibleInfo visibleInfo, cursorInfo cursorInfo, widths widthInfo) error {
	var out strings.Builder

	if r.inputBar {
		r.writeHintRows(&out, r.inputBarTopRow()-InputBarGapLines)
		r.writeGapRow(&out, r.inputBarTopRow()-InputBarGapLines)
		r.writeFullBarRow(&out, r.inputBarTopRow())
		r.writePromptLineAt(&out, r.inputBarTextRow())
		out.WriteString(ColorPromptBar)
	} else {
		r.writeHintRows(&out, r.height)
		r.writePromptLine(&out)
	}

	r.writePrefix(&out)
	r.writeVisibleBuffer(&out, visibleInfo.visibleBuffer, model, widths)
	r.writeStatus(&out, status, widths)

	if r.inputBar {
		r.padInputBar(&out, status, visibleInfo.visibleBuffer, widths, model)
		out.WriteString(ansiReset)
		r.writeFullBarRow(&out, r.inputBarTopRow()+InputBarLines-1)
	}

	cursorCol := widths.prefixWidth + cursorInfo.cursorOffset + 1
	fmt.Fprintf(&out, "\x1b[%dG", cursorCol)

	_, err := r.out.Write([]byte(out.String()))
	if err != nil {
		return fmt.Errorf("render prompt: %w", err)
	}

	return nil
}

// SetHints stores completion rows drawn above the prompt on the next Redraw.
func (r *TermRenderer) SetHints(lines []string, selected int) {
	r.hints = append([]string(nil), lines...)
	r.hintSel = selected
}

func (r *TermRenderer) hintCount() int {
	return len(r.hints)
}

func (r *TermRenderer) inputBarTopRow() int {
	if r.height <= 0 {
		return 1
	}

	return max(r.height-InputBarLines+1, 1)
}

func (r *TermRenderer) inputBarTextRow() int {
	return r.inputBarTopRow() + 1
}

func (r *TermRenderer) writeGapRow(out *strings.Builder, row int) {
	if row < 1 {
		return
	}

	out.WriteString(ansiReset)
	fmt.Fprintf(out, "\x1b[%d;1H\x1b[2K", row)
}

func (r *TermRenderer) writeFullBarRow(out *strings.Builder, row int) {
	fmt.Fprintf(out, "\x1b[%d;1H\x1b[2K", row)
	out.WriteString(ColorPromptBar)

	if r.width > 0 {
		out.WriteString(strings.Repeat(" ", r.width))
	}

	out.WriteString(ansiReset)
}

func (r *TermRenderer) writePromptLineAt(out *strings.Builder, row int) {
	fmt.Fprintf(out, "\x1b[%d;1H", row)
	out.WriteString("\x1b[2K")
}

// writePromptLine writes the prompt line positioning.
func (r *TermRenderer) writePromptLine(out *strings.Builder) {
	if r.height > 0 {
		fmt.Fprintf(out, "\x1b[%d;1H", r.height)
	} else {
		out.WriteString("\r") // fallback to carriage return.
	}

	out.WriteString("\x1b[2K") // Clear the prompt line.
}

// writePrefix writes the prompt prefix.
func (r *TermRenderer) writePrefix(out *strings.Builder) {
	if n := r.barInset(); n > 0 {
		out.WriteString(strings.Repeat(" ", n))
	}

	out.WriteString(r.prefix)
}

// writeVisibleBuffer writes the visible buffer content.
func (r *TermRenderer) writeVisibleBuffer(out *strings.Builder, visibleBuffer string, model *Model, widths widthInfo) {
	if !r.inputBar || model == nil {
		out.WriteString(visibleBuffer)

		return
	}

	full := model.Text()
	cursorRune := model.Cursor()

	if r.isScrolling(widths.bufferWidth, widths.availableWidth) {
		out.WriteString(visibleBuffer)

		if cursorRune >= len([]rune(full)) {
			r.writeCaretCell(out, "")
		}

		return
	}

	before, at, after := splitAtRune(full, cursorRune)
	out.WriteString(before)
	r.writeCaretCell(out, at)
	out.WriteString(after)
}

func (r *TermRenderer) writeCaretCell(out *strings.Builder, at string) {
	out.WriteString(ColorCursorCell)

	if at == "" {
		out.WriteString(" ")
	} else {
		out.WriteString(at)
	}

	out.WriteString(ColorPromptBar)
}

func splitAtRune(text string, cursorRune int) (before, at, after string) {
	runes := []rune(text)

	if cursorRune < 0 {
		cursorRune = 0
	}

	if cursorRune > len(runes) {
		cursorRune = len(runes)
	}

	before = string(runes[:cursorRune])
	if cursorRune < len(runes) {
		at = string(runes[cursorRune])
		after = string(runes[cursorRune+1:])
	}

	return before, at, after
}

// writeStatus writes the status text if there's space.
func (r *TermRenderer) writeStatus(out *strings.Builder, status string, widths widthInfo) {
	if status == "" || r.isScrolling(widths.bufferWidth, widths.availableWidth) {
		return
	}

	contentWidth := widths.prefixWidth + widths.bufferWidth
	requiredSpace := contentWidth + minStatusGap + widths.statusWidth // 3 = minimum gap.

	if requiredSpace <= r.width {
		r.writeFullStatus(out, contentWidth, status, widths.statusWidth)
	} else if r.width-contentWidth >= minStatusGap {
		r.writeTruncatedStatus(out, contentWidth, status)
	}
}

// writeFullStatus writes the full status text.
func (r *TermRenderer) writeFullStatus(out *strings.Builder, contentWidth int, status string, statusWidth int) {
	padding := r.width - contentWidth - statusWidth
	out.WriteString(strings.Repeat(" ", padding))
	out.WriteString(status)
}

// writeTruncatedStatus writes a truncated status text.
func (r *TermRenderer) writeTruncatedStatus(out *strings.Builder, contentWidth int, status string) {
	availableStatusWidth := r.width - contentWidth - minStatusGap
	truncatedStatus := textwidth.TruncateLeft(status, availableStatusWidth)

	out.WriteString(strings.Repeat(" ", minStatusGap))
	out.WriteString(truncatedStatus)
}

// SetWidth updates the terminal width (call on SIGWINCH).
func (r *TermRenderer) SetWidth(width int) {
	r.width = width
}

// SetHeight updates the terminal height (call on SIGWINCH).
func (r *TermRenderer) SetHeight(height int) {
	r.height = height
}

// SetSize updates both terminal dimensions (call on SIGWINCH).
func (r *TermRenderer) SetSize(width, height int) {
	r.width = width
	r.height = height
}

// SetPrefix updates the prompt prefix.
func (r *TermRenderer) SetPrefix(prefix string) {
	r.prefix = prefix
}

// ClearScreen clears the entire screen.
func (r *TermRenderer) ClearScreen() error {
	// Use ANSI clear screen sequence.
	_, err := r.out.Write([]byte("\x1b[2J\x1b[H"))
	if err != nil {
		return fmt.Errorf("clear screen: %w", err)
	}

	return nil
}

// padInputBar fills the remainder of the prompt line so the grey bar is full width.
func (r *TermRenderer) padInputBar(out *strings.Builder, status, visible string, widths widthInfo, model *Model) {
	if r.width <= 0 {
		return
	}

	used := r.inputBarUsedCells(status, visible, widths, model)
	if pad := r.width - used; pad > 0 {
		out.WriteString(strings.Repeat(" ", pad))
	}
}

func (r *TermRenderer) inputBarUsedCells(status, visible string, widths widthInfo, model *Model) int {
	used := widths.prefixWidth + uniseg.StringWidth(visible)
	if r.caretAtEnd(model) {
		used++
	}

	if status == "" || r.isScrolling(widths.bufferWidth, widths.availableWidth) {
		return used
	}

	contentWidth := widths.prefixWidth + widths.bufferWidth

	requiredSpace := contentWidth + minStatusGap + widths.statusWidth
	if requiredSpace <= r.width {
		return r.width
	}

	if r.width-contentWidth >= minStatusGap {
		truncated := textwidth.TruncateLeft(status, r.width-contentWidth-minStatusGap)

		return contentWidth + minStatusGap + uniseg.StringWidth(truncated)
	}

	return used
}

func (r *TermRenderer) caretAtEnd(model *Model) bool {
	if !r.inputBar || model == nil {
		return false
	}

	return model.Cursor() >= len([]rune(model.Text()))
}

// isScrolling returns true if buffer requires horizontal scrolling.
func (r *TermRenderer) isScrolling(bufferWidth, availableWidth int) bool {
	return bufferWidth > availableWidth
}

// extractVisibleSlice extracts a substring from text based on visual width range [start, end).
// Returns the extracted string and the actual start offset in cells.
func extractVisibleSlice(text string, startWidth, endWidth int) (visible string, actualWidth int) {
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

func (r *TermRenderer) writeHintRows(out *strings.Builder, promptTop int) {
	n := r.hintCount()
	if n == 0 && r.hintPrev == 0 {
		return
	}

	clearFrom := max(promptTop-max(r.hintPrev, n), 1)
	for row := clearFrom; row < promptTop; row++ {
		fmt.Fprintf(out, "\x1b[%d;1H\x1b[2K", row)
	}

	start := promptTop - n
	for i, line := range r.hints {
		row := start + i
		if row < 1 {
			continue
		}

		fmt.Fprintf(out, "\x1b[%d;1H\x1b[2K", row)

		if i == r.hintSel {
			out.WriteString(colorHintSel)
		}

		out.WriteString(truncateHint(line, r.width))

		if i == r.hintSel {
			out.WriteString(ansiReset)
		}
	}

	r.hintPrev = n
}

func truncateHint(line string, width int) string {
	if width <= 0 {
		return line
	}

	return textwidth.TruncateRight(line, width)
}
