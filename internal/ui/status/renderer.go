package status

import (
	"fmt"
	"io"
	"strings"
)

// Renderer handles rendering the status bar to the terminal.
// It uses ANSI escape sequences for positioning and scrolling regions.
type Renderer struct {
	out            io.Writer
	width          int
	height         int
	scrollingSetup bool // Track if scrolling region is set up
}

// NewRenderer creates a new status bar renderer.
func NewRenderer(out io.Writer, width, height int) *Renderer {
	r := &Renderer{
		out:    out,
		width:  width,
		height: height,
	}
	r.setupScrollingRegion()
	return r
}

// setupScrollingRegion sets up the terminal scrolling region.
// This reserves the bottom 2 lines for status bar and prompt,
// allowing content to scroll only in the top area.
func (r *Renderer) setupScrollingRegion() error {
	if r.height < 3 {
		// Terminal too small
		return nil
	}

	// Set scrolling region to lines 1 through (height - 2)
	// This leaves the last 2 lines for status bar and prompt
	scrollableLines := r.height - 2
	fmt.Fprintf(r.out, "\x1b[1;%dr", scrollableLines)

	// Move cursor to the bottom of the scrolling region
	// This ensures new content appears at the bottom of the scrollable area
	fmt.Fprintf(r.out, "\x1b[%d;1H", scrollableLines)

	r.scrollingSetup = true
	return nil
}

// SetSize updates the terminal dimensions and re-establishes scrolling region.
func (r *Renderer) SetSize(width, height int) {
	r.width = width
	r.height = height
	r.setupScrollingRegion()
}

// Render renders the status bar at the bottom of the terminal.
// It positions the status bar at line (height - 1) and the prompt at line (height).
// Uses save/restore cursor to avoid disrupting the scrolling region.
func (r *Renderer) Render(statusText string) error {
	if r.height < 3 || r.width < 10 {
		// Terminal too small, don't render status bar
		return nil
	}

	// Save cursor position
	fmt.Fprint(r.out, "\x1b7")

	// Position cursor at the status bar line (second to last line)
	// This is outside the scrolling region
	statusLine := r.height - 1
	fmt.Fprintf(r.out, "\x1b[%d;1H", statusLine)

	// Clear the status bar line
	fmt.Fprint(r.out, "\x1b[2K")

	// Render status text if provided
	if statusText != "" {
		// Strip any existing ANSI codes to prevent color bleeding
		cleanText := stripANSI(statusText)

		// Truncate if too long (now measuring actual visible length)
		if len(cleanText) > r.width-2 {
			cleanText = cleanText[:r.width-5] + "..."
		}

		// Center the status text
		padding := (r.width - len(cleanText)) / 2
		if padding > 0 {
			fmt.Fprint(r.out, strings.Repeat(" ", padding))
		}

		// Apply consistent color: bright white for status bar
		fmt.Fprint(r.out, "\x1b[0m")    // Reset any previous formatting
		fmt.Fprint(r.out, "\x1b[37;1m") // Bright white
		fmt.Fprint(r.out, cleanText)    // Render clean text
		fmt.Fprint(r.out, "\x1b[0m")    // Reset formatting
	}

	// Restore cursor position
	fmt.Fprint(r.out, "\x1b8")

	return nil
}

// Clear clears the status bar.
func (r *Renderer) Clear() error {
	return r.Render("")
}

// MoveToPrompt moves cursor to the prompt line.
func (r *Renderer) MoveToPrompt() error {
	promptLine := r.height
	fmt.Fprintf(r.out, "\x1b[%d;1H", promptLine)
	return nil
}

// MoveToScrollRegion moves cursor to the bottom of the scrolling region.
// This ensures new content is printed in the scrollable area, not
// at the fixed status/prompt lines.
func (r *Renderer) MoveToScrollRegion() error {
	if r.height < 3 {
		return nil
	}
	scrollableLines := r.height - 2
	fmt.Fprintf(r.out, "\x1b[%d;1H", scrollableLines)
	return nil
}

// RenderMetrics renders comprehensive completion metrics in the status bar.
func (r *Renderer) RenderMetrics(metrics *Metrics) error {
	if r.height < 3 || r.width < 10 {
		// Terminal too small, don't render status bar
		return nil
	}

	// Save cursor position
	fmt.Fprint(r.out, "\x1b7")

	// Position cursor at the status bar line (second to last line)
	statusLine := r.height - 1
	fmt.Fprintf(r.out, "\x1b[%d;1H", statusLine)

	// Clear the status bar line
	fmt.Fprint(r.out, "\x1b[2K")

	// Build comprehensive status line
	metricsLine := r.buildMetricsLine(metrics)

	// Render the status line
	fmt.Fprint(r.out, "\x1b[0m")    // Reset formatting
	fmt.Fprint(r.out, "\x1b[37;1m") // Bright white
	fmt.Fprint(r.out, metricsLine)
	fmt.Fprint(r.out, "\x1b[0m") // Reset formatting

	// Restore cursor position
	fmt.Fprint(r.out, "\x1b8")
	return nil
}

// buildMetricsLine builds the comprehensive metrics status line.
func (r *Renderer) buildMetricsLine(metrics *Metrics) string {
	var parts []string

	// Activity indicator
	if metrics.AgentState != "" {
		parts = append(parts, "[●]")
	} else {
		parts = append(parts, "[○]")
	}

	// Context usage percentage
	if metrics.MaxTokens > 0 {
		percentage := float64(metrics.TokenCount) / float64(metrics.MaxTokens) * 100
		usageStr := fmt.Sprintf("%.0f%%", percentage)
		if percentage > 80 {
			usageStr = fmt.Sprintf("\x1b[33m%s\x1b[0m", usageStr) // Yellow for high usage
		}
		parts = append(parts, usageStr)
	} else {
		parts = append(parts, "N/A")
	}

	// Agent state
	if metrics.AgentState != "" {
		parts = append(parts, metrics.AgentState)
	}

	// Provider and model
	if metrics.Provider != "" && metrics.Model != "" {
		parts = append(parts, fmt.Sprintf("%s/%s", metrics.Provider, metrics.Model))
	}

	// Tokens per second
	if metrics.TokensPerSec > 0 {
		parts = append(parts, fmt.Sprintf("%.0f tok/s", metrics.TokensPerSec))
	}

	// Conversation ID (shortened)
	if metrics.ConversationID != "" {
		shortID := metrics.ConversationID
		if len(shortID) > 8 {
			shortID = shortID[:8]
		}
		parts = append(parts, fmt.Sprintf("conv:%s", shortID))
	}

	// Hotkey hint
	parts = append(parts, "?:help")

	// Join parts with spaces
	fullLine := strings.Join(parts, " ")

	// Truncate if too long
	if len(fullLine) > r.width-2 {
		fullLine = fullLine[:r.width-5] + "..."
	}

	return fullLine
}

// stripANSI removes ANSI escape sequences from text.
func stripANSI(text string) string {
	// Simple ANSI strip - remove escape sequences
	var result strings.Builder
	inEscape := false

	for _, r := range text {
		if r == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' {
				inEscape = false
			}
			continue
		}
		result.WriteRune(r)
	}

	return result.String()
}
