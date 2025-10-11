package overlay

import (
	"fmt"
	"strings"
)

// FilePreviewRenderer renders a file preview popup
type FilePreviewRenderer struct {
	width int
}

// NewFilePreviewRenderer creates a new file preview renderer
func NewFilePreviewRenderer(width int) *FilePreviewRenderer {
	return &FilePreviewRenderer{width: width}
}

// Render renders the file preview popup as a string
func (r *FilePreviewRenderer) Render(fp *FilePreview) string {
	var buf strings.Builder

	// Calculate content area dimensions
	// Popup has: 1 line header + 1 line border + content + 1 line border
	contentHeight := fp.Height - 3
	if contentHeight < 1 {
		contentHeight = 1
	}

	// Render header
	buf.WriteString(r.renderHeader(fp))
	buf.WriteString("\n")

	// Render top border
	buf.WriteString(r.renderBorder())
	buf.WriteString("\n")

	// Render content (code with gutter)
	buf.WriteString(r.renderContent(fp, contentHeight))

	// Render bottom border with scroll indicator
	buf.WriteString(r.renderBottomBorder(fp))

	return buf.String()
}

// renderHeader renders the popup header with filename and close hint
// Format: ┌─ filename.go ────────────────────────────── [Esc to close] ─┐
func (r *FilePreviewRenderer) renderHeader(fp *FilePreview) string {
	const escHint = " [Esc to close] "
	const prefix = "┌─ "
	const suffix = " ─┐"

	availableWidth := r.width - len(prefix) - len(suffix) - len(escHint)
	if availableWidth < 10 {
		availableWidth = 10
	}

	// Truncate filename if too long
	filename := fp.FilePath
	if len(filename) > availableWidth {
		filename = "..." + filename[len(filename)-(availableWidth-3):]
	}

	// Calculate padding to right-align the hint
	padding := availableWidth - len(filename)
	if padding < 0 {
		padding = 0
	}

	return fmt.Sprintf("%s%s%s%s%s",
		dim(prefix),
		filename,
		dim(strings.Repeat("─", padding)),
		muted(escHint),
		dim(suffix))
}

// renderBorder renders a horizontal border line
func (r *FilePreviewRenderer) renderBorder() string {
	return dim("│" + strings.Repeat(" ", r.width-2) + "│")
}

// renderBottomBorder renders the bottom border with optional scroll indicator
func (r *FilePreviewRenderer) renderBottomBorder(fp *FilePreview) string {
	indicator := ""
	totalLines := len(fp.Lines)
	contentHeight := fp.Height - 3 // -3 for header and top/bottom borders
	visibleEnd := fp.ScrollPos + contentHeight

	// Only show scroll indicator if file is longer than viewport
	if totalLines > contentHeight {
		// Show scroll position indicator
		if visibleEnd > totalLines {
			visibleEnd = totalLines
		}
		indicator = fmt.Sprintf(" [%d-%d/%d] ", fp.ScrollPos+1, visibleEnd, totalLines)
	}

	borderWidth := r.width - 2 - len(indicator)
	if borderWidth < 0 {
		borderWidth = 0
	}

	return dim("└" + strings.Repeat("─", borderWidth) + muted(indicator) + "┘")
}

// renderContent renders the code content with line numbers
func (r *FilePreviewRenderer) renderContent(fp *FilePreview, contentHeight int) string {
	var buf strings.Builder

	lines := fp.GetVisibleLines()
	if len(lines) > contentHeight {
		lines = lines[:contentHeight]
	}

	// Calculate gutter width (dynamic based on max line number)
	maxLineNum := fp.ScrollPos + len(lines)
	gutterWidth := len(fmt.Sprintf("%d", maxLineNum))
	if gutterWidth < 3 {
		gutterWidth = 3
	}

	// Calculate available width for code content
	// Format: │ gutter | code...                                              │
	// Border (1) + space (1) + gutter + space (1) + bar (1) + space (1) + content + space (1) + border (1)
	contentWidth := r.width - 2 - gutterWidth - 4 // -2 borders, -4 for spacing and bar
	if contentWidth < 10 {
		contentWidth = 10
	}

	for i, line := range lines {
		lineNum := fp.ScrollPos + i + 1

		// Highlight target line
		isTarget := lineNum == fp.TargetLine

		// Truncate or pad line to fit content width
		if len(line) > contentWidth {
			line = line[:contentWidth-1] + "…"
		} else if len(line) < contentWidth {
			line = line + strings.Repeat(" ", contentWidth-len(line))
		}

		// Render line: │ linenum │ code...                                              │
		gutter := fmt.Sprintf("%*d", gutterWidth, lineNum)

		if isTarget {
			// Highlight entire target line
			buf.WriteString(fmt.Sprintf("%s %s %s %s %s\n",
				dim("│"),
				yellow(gutter),
				dim("│"),
				yellow(line),
				dim("│")))
		} else {
			buf.WriteString(fmt.Sprintf("%s %s %s %s %s\n",
				dim("│"),
				muted(gutter),
				dim("│"),
				line,
				dim("│")))
		}
	}

	// Fill remaining lines with empty rows if content is shorter than viewport
	for i := len(lines); i < contentHeight; i++ {
		emptyLine := strings.Repeat(" ", r.width-2)
		buf.WriteString(fmt.Sprintf("%s%s%s\n", dim("│"), emptyLine, dim("│")))
	}

	return buf.String()
}

// ANSI color helpers (reusing from palette_renderer.go style)
func dim(s string) string {
	return "\x1b[2m" + s + "\x1b[0m"
}

func muted(s string) string {
	return "\x1b[38;5;242m" + s + "\x1b[0m"
}

func yellow(s string) string {
	return "\x1b[38;5;220m" + s + "\x1b[0m"
}
