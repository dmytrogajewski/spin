package overlay

import (
	"strings"

	"github.com/rivo/uniseg"
)

const (
	paletteHeightRatio   = 0.6
	paletteBorderWidth   = 2
	paletteBorderPadding = 4
	paletteMaxWidth      = 80
	paletteMinHeight     = 8
	paletteFrameRows     = 6
)

// Design tokens (matching blocks package).
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

// ANSI color codes.
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
	paletteWidth := min(paletteMaxWidth, r.width-2*s4)
	maxHeight := max(int(float64(r.height)*paletteHeightRatio), paletteMinHeight)
	leftPad := (r.width - paletteWidth) / paletteBorderWidth

	var sb strings.Builder

	r.renderTopBorder(&sb, paletteWidth, leftPad)
	r.renderEmptyRow(&sb, paletteWidth, leftPad)
	r.renderInputLine(&sb, p, paletteWidth, leftPad)
	r.renderEmptyRow(&sb, paletteWidth, leftPad)
	r.renderResultsList(&sb, p, paletteWidth, maxHeight, leftPad)
	r.renderEmptyRow(&sb, paletteWidth, leftPad)
	r.renderBottomBorder(&sb, paletteWidth, leftPad)

	return sb.String()
}

// renderTopBorder writes the top border with title and escape hint.
func (r *PaletteRenderer) renderTopBorder(sb *strings.Builder, paletteWidth, leftPad int) {
	sb.WriteString(strings.Repeat(" ", leftPad))
	sb.WriteString(colorBorder)
	sb.WriteString("╭─")
	sb.WriteString(colorBold + colorFg)
	sb.WriteString(" Command Palette ")
	sb.WriteString(colorReset + colorBorder)
	sb.WriteString(strings.Repeat("─", paletteWidth-len(" Command Palette ")-len("[Esc]")-paletteBorderPadding))
	sb.WriteString(colorMuted)
	sb.WriteString("[Esc]")
	sb.WriteString(colorBorder)
	sb.WriteString("─╮")
	sb.WriteString(colorReset)
	sb.WriteString("\n")
}

// renderEmptyRow writes an empty bordered row.
func (r *PaletteRenderer) renderEmptyRow(sb *strings.Builder, paletteWidth, leftPad int) {
	sb.WriteString(strings.Repeat(" ", leftPad))
	sb.WriteString(colorBorder)
	sb.WriteString("│")
	sb.WriteString(strings.Repeat(" ", paletteWidth-paletteBorderWidth))
	sb.WriteString(colorBorder)
	sb.WriteString("│")
	sb.WriteString(colorReset)
	sb.WriteString("\n")
}

// renderInputLine writes the query input line with cursor.
func (r *PaletteRenderer) renderInputLine(sb *strings.Builder, p *Palette, paletteWidth, leftPad int) {
	sb.WriteString(strings.Repeat(" ", leftPad))
	sb.WriteString(colorBorder)
	sb.WriteString("│")
	sb.WriteString("  ")
	sb.WriteString(colorBlue)
	sb.WriteString("❯ ")
	sb.WriteString(colorReset + colorFg)

	query := p.Query()
	sb.WriteString(query)
	sb.WriteString("_")

	usedWidth := paletteBorderWidth + paletteBorderWidth + uniseg.StringWidth(query) + 1 + paletteBorderWidth
	sb.WriteString(strings.Repeat(" ", paletteWidth-paletteBorderWidth-usedWidth))
	sb.WriteString(colorBorder)
	sb.WriteString("│")
	sb.WriteString(colorReset)
	sb.WriteString("\n")
}

// renderResultsList writes the filtered command results or empty state.
func (r *PaletteRenderer) renderResultsList(sb *strings.Builder, p *Palette, paletteWidth, maxHeight, leftPad int) {
	filtered := p.FilteredCommands()
	maxItems := min(len(filtered), maxHeight-paletteFrameRows)

	if len(filtered) == 0 {
		r.renderEmptyState(sb, p, paletteWidth, leftPad)

		return
	}

	for i := range maxItems {
		cmd := filtered[i]
		selected := (i == p.Selection())
		sb.WriteString(r.renderItem(cmd, selected, paletteWidth, leftPad))
	}
}

// renderEmptyState writes the "no results" message.
func (r *PaletteRenderer) renderEmptyState(sb *strings.Builder, p *Palette, paletteWidth, leftPad int) {
	emptyMsg := "No commands match '" + p.Query() + "'"
	if p.Query() == "" {
		emptyMsg = "No commands available"
	}

	sb.WriteString(strings.Repeat(" ", leftPad))
	sb.WriteString(colorBorder)
	sb.WriteString("│")
	sb.WriteString("  ")
	sb.WriteString(colorMuted)
	sb.WriteString(emptyMsg)
	sb.WriteString(strings.Repeat(" ", paletteWidth-paletteBorderWidth-paletteBorderWidth-uniseg.StringWidth(emptyMsg)-paletteBorderWidth))
	sb.WriteString(colorBorder)
	sb.WriteString("│")
	sb.WriteString(colorReset)
	sb.WriteString("\n")
}

// renderBottomBorder writes the bottom border.
func (r *PaletteRenderer) renderBottomBorder(sb *strings.Builder, paletteWidth, leftPad int) {
	sb.WriteString(strings.Repeat(" ", leftPad))
	sb.WriteString(colorBorder)
	sb.WriteString("╰")
	sb.WriteString(strings.Repeat("─", paletteWidth-paletteBorderWidth))
	sb.WriteString("╯")
	sb.WriteString(colorReset)
	sb.WriteString("\n")
}

// renderItem renders a single command item.
func (r *PaletteRenderer) renderItem(cmd Command, selected bool, paletteWidth, leftPad int) string {
	var sb strings.Builder

	sb.WriteString(strings.Repeat(" ", leftPad))
	sb.WriteString(colorBorder)
	sb.WriteString("│")

	// Background invert if selected.
	if selected {
		sb.WriteString(colorInvert)
	}

	sb.WriteString(" ") // s2 padding start.
	sb.WriteRune(cmd.Icon())
	sb.WriteString("  ")
	sb.WriteString(colorFg)
	sb.WriteString(cmd.Name())

	// Category right-aligned.
	category := cmd.Category()
	nameWidth := uniseg.StringWidth(cmd.Name())
	iconWidth := 1

	availableWidth := paletteWidth - paletteBorderWidth - 1 - iconWidth -
		paletteBorderWidth - nameWidth - paletteBorderWidth -
		uniseg.StringWidth(category) - paletteBorderWidth
	if availableWidth > 0 {
		sb.WriteString(strings.Repeat(" ", availableWidth))
		sb.WriteString(colorMuted)
		sb.WriteString(category)
		sb.WriteString(colorReset)

		if selected {
			sb.WriteString(colorInvert)
		}

		sb.WriteString(" ") // s2 padding end.
	} else {
		// Not enough space, just pad to end.
		sb.WriteString(strings.Repeat(" ", paletteWidth-paletteBorderWidth-1-iconWidth-paletteBorderWidth-nameWidth-paletteBorderWidth))
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
