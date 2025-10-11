package overlay

import (
	"strings"

	"github.com/rivo/uniseg"
)

// Design tokens (matching blocks package)
const (
	s0  = 0
	s1  = 1
	s2  = 2
	s3  = 3
	s4  = 4
	s6  = 6
	s8  = 8
	s12 = 12
)

// ANSI color codes
const (
	colorReset   = "\x1b[0m"
	colorBold    = "\x1b[1m"
	colorDim     = "\x1b[2m"
	colorInvert  = "\x1b[7m"
	colorFg      = "\x1b[38;5;252m"
	colorBg      = "\x1b[48;5;233m"
	colorMuted   = "\x1b[38;5;244m"
	colorBorder  = "\x1b[38;5;238m"
	colorShadow  = "\x1b[38;5;235m"
	colorBlue    = "\x1b[38;5;39m"
	colorCyan    = "\x1b[38;5;51m"
	colorMagenta = "\x1b[38;5;213m"
)

// PaletteRenderer renders the command palette overlay.
type PaletteRenderer struct {
	width  int
	height int
}

// NewPaletteRenderer creates a new palette renderer.
func NewPaletteRenderer(width, height int) *PaletteRenderer {
	return &PaletteRenderer{
		width:  width,
		height: height,
	}
}

// SetSize updates the renderer dimensions (for resize events).
func (r *PaletteRenderer) SetSize(width, height int) {
	r.width = width
	r.height = height
}

// Render returns ANSI sequences for the palette overlay.
// Returns a multi-line string with embedded newlines.
func (r *PaletteRenderer) Render(p *Palette) string {
	// Calculate palette dimensions
	paletteWidth := min(80, r.width-2*s4)
	maxHeight := int(float64(r.height) * 0.6)
	if maxHeight < 8 {
		maxHeight = 8
	}

	// Calculate centering offset
	leftPad := (r.width - paletteWidth) / 2

	var sb strings.Builder

	// Top border with title
	sb.WriteString(strings.Repeat(" ", leftPad))
	sb.WriteString(colorBorder)
	sb.WriteString("╭─")
	sb.WriteString(colorBold + colorFg)
	sb.WriteString(" Command Palette ")
	sb.WriteString(colorReset + colorBorder)
	sb.WriteString(strings.Repeat("─", paletteWidth-len(" Command Palette ")-len("[Esc]")-4))
	sb.WriteString(colorMuted)
	sb.WriteString("[Esc]")
	sb.WriteString(colorBorder)
	sb.WriteString("─╮")
	sb.WriteString(colorReset)
	sb.WriteString("\n")

	// Empty row
	sb.WriteString(strings.Repeat(" ", leftPad))
	sb.WriteString(colorBorder)
	sb.WriteString("│")
	sb.WriteString(strings.Repeat(" ", paletteWidth-2))
	sb.WriteString(colorBorder)
	sb.WriteString("│")
	sb.WriteString(colorReset)
	sb.WriteString("\n")

	// Input line
	sb.WriteString(strings.Repeat(" ", leftPad))
	sb.WriteString(colorBorder)
	sb.WriteString("│")
	sb.WriteString("  ") // s2 padding
	sb.WriteString(colorBlue)
	sb.WriteString("❯ ")
	sb.WriteString(colorReset + colorFg)
	query := p.Query()
	sb.WriteString(query)
	sb.WriteString("_") // cursor
	// Padding to width
	usedWidth := 2 + 2 + uniseg.StringWidth(query) + 1 + 2 // padding + prompt + query + cursor + padding
	sb.WriteString(strings.Repeat(" ", paletteWidth-2-usedWidth))
	sb.WriteString(colorBorder)
	sb.WriteString("│")
	sb.WriteString(colorReset)
	sb.WriteString("\n")

	// Empty row
	sb.WriteString(strings.Repeat(" ", leftPad))
	sb.WriteString(colorBorder)
	sb.WriteString("│")
	sb.WriteString(strings.Repeat(" ", paletteWidth-2))
	sb.WriteString(colorBorder)
	sb.WriteString("│")
	sb.WriteString(colorReset)
	sb.WriteString("\n")

	// Results list
	filtered := p.FilteredCommands()
	maxItems := min(len(filtered), maxHeight-6) // Reserve rows for border, input, empty rows

	if len(filtered) == 0 {
		// Empty state
		emptyMsg := "No commands match '" + p.Query() + "'"
		if p.Query() == "" {
			emptyMsg = "No commands available"
		}
		sb.WriteString(strings.Repeat(" ", leftPad))
		sb.WriteString(colorBorder)
		sb.WriteString("│")
		sb.WriteString("  ") // s2 padding
		sb.WriteString(colorMuted)
		sb.WriteString(emptyMsg)
		sb.WriteString(strings.Repeat(" ", paletteWidth-2-2-uniseg.StringWidth(emptyMsg)-2))
		sb.WriteString(colorBorder)
		sb.WriteString("│")
		sb.WriteString(colorReset)
		sb.WriteString("\n")
	} else {
		for i := 0; i < maxItems; i++ {
			cmd := filtered[i]
			selected := (i == p.Selection())
			sb.WriteString(r.renderItem(cmd, selected, paletteWidth, leftPad))
		}
	}

	// Empty row
	sb.WriteString(strings.Repeat(" ", leftPad))
	sb.WriteString(colorBorder)
	sb.WriteString("│")
	sb.WriteString(strings.Repeat(" ", paletteWidth-2))
	sb.WriteString(colorBorder)
	sb.WriteString("│")
	sb.WriteString(colorReset)
	sb.WriteString("\n")

	// Bottom border
	sb.WriteString(strings.Repeat(" ", leftPad))
	sb.WriteString(colorBorder)
	sb.WriteString("╰")
	sb.WriteString(strings.Repeat("─", paletteWidth-2))
	sb.WriteString("╯")
	sb.WriteString(colorReset)
	sb.WriteString("\n")

	return sb.String()
}

// renderItem renders a single command item.
func (r *PaletteRenderer) renderItem(cmd Command, selected bool, paletteWidth int, leftPad int) string {
	var sb strings.Builder

	sb.WriteString(strings.Repeat(" ", leftPad))
	sb.WriteString(colorBorder)
	sb.WriteString("│")

	// Background invert if selected
	if selected {
		sb.WriteString(colorInvert)
	}

	sb.WriteString(" ") // s2 padding start
	sb.WriteString(string(cmd.Icon()))
	sb.WriteString("  ")
	sb.WriteString(colorFg)
	sb.WriteString(cmd.Name())

	// Category right-aligned
	category := cmd.Category()
	nameWidth := uniseg.StringWidth(cmd.Name())
	iconWidth := 1
	availableWidth := paletteWidth - 2 - 1 - iconWidth - 2 - nameWidth - 2 - uniseg.StringWidth(category) - 2
	if availableWidth > 0 {
		sb.WriteString(strings.Repeat(" ", availableWidth))
		sb.WriteString(colorMuted)
		sb.WriteString(category)
		sb.WriteString(colorReset)
		if selected {
			sb.WriteString(colorInvert)
		}
		sb.WriteString(" ") // s2 padding end
	} else {
		// Not enough space, just pad to end
		sb.WriteString(strings.Repeat(" ", paletteWidth-2-1-iconWidth-2-nameWidth-2))
	}

	if selected {
		sb.WriteString(colorReset)
	}

	sb.WriteString(colorBorder)
	sb.WriteString("│")
	sb.WriteString(colorReset)
	sb.WriteString("\n")

	return sb.String()
}

// min returns the minimum of two ints.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
