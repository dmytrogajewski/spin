package status

import (
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/rivo/uniseg"

	"github.com/dmytrogajewski/spin/pkg/ui/textwidth"
)

// Status bar rendering constants.
const (
	minTermHeight      = 3  // minimum terminal height to render status bar.
	minTermWidth       = 10 // minimum terminal width to render status bar.
	statusBarLines     = 2  // lines reserved for status bar and prompt.
	highUsageThreshold = 80 // context usage percentage for warning color.
)

// Renderer handles rendering the status bar to the terminal.
// It uses ANSI escape sequences for positioning and scrolling regions.
// All rendering methods are safe for concurrent use.
type Renderer struct {
	mu             sync.Mutex
	out            io.Writer
	width          int
	height         int
	scrollingSetup bool // Track if scrolling region is set up.
}

// NewRenderer creates a new status bar renderer.
func NewRenderer(out io.Writer, width, height int) *Renderer {
	r := &Renderer{
		out:    out,
		width:  width,
		height: height,
	}
	r.setupScrollingRegionLocked()

	return r
}

// setupScrollingRegionLocked sets up the terminal scrolling region.
// This reserves the bottom 2 lines for status bar and prompt,
// allowing content to scroll only in the top area.
// Caller must hold r.mu or be in the constructor.
func (r *Renderer) setupScrollingRegionLocked() {
	if r.height < minTermHeight {
		// Terminal too small.
		return
	}

	// Set scrolling region to lines 1 through (height - 2).
	// This leaves the last 2 lines for status bar and prompt.
	scrollableLines := r.height - statusBarLines
	fmt.Fprintf(r.out, "\x1b[1;%dr", scrollableLines)

	// Move cursor to the bottom of the scrolling region.
	// This ensures new content appears at the bottom of the scrollable area.
	fmt.Fprintf(r.out, "\x1b[%d;1H", scrollableLines)

	r.scrollingSetup = true
}

// SetSize updates the terminal dimensions and re-establishes scrolling region.
func (r *Renderer) SetSize(width, height int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.width = width
	r.height = height
	r.setupScrollingRegionLocked()
}

// renderAtStatusLine saves cursor, positions at the status line, clears it,
// writes content, then restores the cursor. This centralizes the
// save/position/clear/restore boilerplate used by Render and RenderMetrics.
func (r *Renderer) renderAtStatusLine(content string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Save cursor position.
	fmt.Fprint(r.out, "\x1b7")

	// Position cursor at the status bar line (second to last line).
	statusLine := r.height - 1
	fmt.Fprintf(r.out, "\x1b[%d;1H", statusLine)

	// Clear the status bar line.
	fmt.Fprint(r.out, "\x1b[2K")

	// Write content if provided.
	if content != "" {
		fmt.Fprint(r.out, content)
	}

	// Restore cursor position.
	fmt.Fprint(r.out, "\x1b8")
}

// Render renders the status bar at the bottom of the terminal.
// It positions the status bar at line (height - 1) and the prompt at line (height).
// Uses save/restore cursor to avoid disrupting the scrolling region.
func (r *Renderer) Render(statusText string) error {
	if r.height < minTermHeight || r.width < minTermWidth {
		// Terminal too small, don't render status bar.
		return nil
	}

	var content string

	// Render status text if provided.
	if statusText != "" {
		// Strip any existing ANSI codes to prevent color bleeding.
		cleanText := textwidth.StripANSI(statusText)

		// Truncate if too long (measuring actual display width).
		displayWidth := uniseg.StringWidth(cleanText)
		if displayWidth > r.width-statusBarLines {
			cleanText = textwidth.MidEllipsize(cleanText, r.width-statusBarLines)
			displayWidth = uniseg.StringWidth(cleanText)
		}

		// Center the status text.
		var buf strings.Builder

		padding := (r.width - displayWidth) / statusBarLines
		if padding > 0 {
			buf.WriteString(strings.Repeat(" ", padding))
		}

		// Apply consistent color: bright white for status bar.
		buf.WriteString("\x1b[0m")    // Reset any previous formatting.
		buf.WriteString("\x1b[37;1m") // Bright white.
		buf.WriteString(cleanText)    // Render clean text.
		buf.WriteString("\x1b[0m")    // Reset formatting.

		content = buf.String()
	}

	r.renderAtStatusLine(content)

	return nil
}

// Clear clears the status bar.
func (r *Renderer) Clear() error {
	return r.Render("")
}

// MoveToPrompt moves cursor to the prompt line.
func (r *Renderer) MoveToPrompt() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	promptLine := r.height
	fmt.Fprintf(r.out, "\x1b[%d;1H", promptLine)

	return nil
}

// MoveToScrollRegion moves cursor to the bottom of the scrolling region.
// This ensures new content is printed in the scrollable area, not
// at the fixed status/prompt lines.
func (r *Renderer) MoveToScrollRegion() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.height < minTermHeight {
		return nil
	}

	scrollableLines := r.height - statusBarLines
	fmt.Fprintf(r.out, "\x1b[%d;1H", scrollableLines)

	return nil
}

// RenderMetrics renders comprehensive completion metrics in the status bar.
func (r *Renderer) RenderMetrics(metrics *Metrics) error {
	if r.height < minTermHeight || r.width < minTermWidth {
		// Terminal too small, don't render status bar.
		return nil
	}

	// Build comprehensive status line.
	metricsLine := r.buildMetricsLine(metrics)

	// Wrap with formatting.
	content := "\x1b[0m" + "\x1b[37;1m" + metricsLine + "\x1b[0m"

	r.renderAtStatusLine(content)

	return nil
}

// buildMetricsLine builds the comprehensive metrics status line.
// It delegates to the shared FormatMetrics formatter for the core layout,
// then applies renderer-specific post-processing (high-usage color, truncation).
func (r *Renderer) buildMetricsLine(metrics *Metrics) string {
	// Use the shared formatter at DetailFull level (no spinner in this path).
	fullLine := FormatMetrics(metrics, DetailFull, "")

	// Renderer-specific: apply ANSI yellow for high context usage.
	if metrics.MaxTokens > 0 {
		percentage := calculateTokenUsage(metrics.TokenCount, metrics.MaxTokens)
		if percentage > highUsageThreshold {
			usageStr := fmt.Sprintf("%.0f%%", percentage)
			coloredUsage := fmt.Sprintf("\x1b[33m%s\x1b[0m", usageStr)
			fullLine = strings.Replace(fullLine, usageStr, coloredUsage, 1)
		}
	}

	// Truncate if too long (measuring actual display width).
	if uniseg.StringWidth(fullLine) > r.width-statusBarLines {
		fullLine = textwidth.MidEllipsize(fullLine, r.width-statusBarLines)
	}

	return fullLine
}
