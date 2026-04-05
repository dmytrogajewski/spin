package overlay

import (
	"strings"

	"github.com/rivo/uniseg"

	"github.com/dmytrogajewski/spin/internal/ui/blocks"
)

const (
	paletteHeightRatio   = 0.6
	paletteBorderWidth   = 2
	paletteBorderPadding = 4
	paletteMaxWidth      = 80
	paletteMinHeight     = 8
	paletteFrameRows     = 6
)

// colorInvert has no equivalent in the blocks package.
const colorInvert = "\x1b[7m"

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
	paletteWidth := min(paletteMaxWidth, r.width-2*blocks.S4)
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
	sb.WriteString(string(blocks.ColorBorder))
	sb.WriteString("╭─")
	sb.WriteString(string(blocks.ColorBold) + string(blocks.ColorFg))
	sb.WriteString(" Command Palette ")
	sb.WriteString(string(blocks.ColorReset) + string(blocks.ColorBorder))
	sb.WriteString(strings.Repeat("─", paletteWidth-len(" Command Palette ")-len("[Esc]")-paletteBorderPadding))
	sb.WriteString(string(blocks.ColorMuted))
	sb.WriteString("[Esc]")
	sb.WriteString(string(blocks.ColorBorder))
	sb.WriteString("─╮")
	sb.WriteString(string(blocks.ColorReset))
	sb.WriteString("\n")
}

// renderEmptyRow writes an empty bordered row.
func (r *PaletteRenderer) renderEmptyRow(sb *strings.Builder, paletteWidth, leftPad int) {
	sb.WriteString(strings.Repeat(" ", leftPad))
	sb.WriteString(string(blocks.ColorBorder))
	sb.WriteString("│")
	sb.WriteString(strings.Repeat(" ", paletteWidth-paletteBorderWidth))
	sb.WriteString(string(blocks.ColorBorder))
	sb.WriteString("│")
	sb.WriteString(string(blocks.ColorReset))
	sb.WriteString("\n")
}

// renderInputLine writes the query input line with cursor.
func (r *PaletteRenderer) renderInputLine(sb *strings.Builder, p *Palette, paletteWidth, leftPad int) {
	sb.WriteString(strings.Repeat(" ", leftPad))
	sb.WriteString(string(blocks.ColorBorder))
	sb.WriteString("│")
	sb.WriteString("  ")
	sb.WriteString(string(blocks.ColorBlue))
	sb.WriteString("❯ ")
	sb.WriteString(string(blocks.ColorReset) + string(blocks.ColorFg))

	query := p.Query()
	sb.WriteString(query)
	sb.WriteString("_")

	usedWidth := paletteBorderWidth + paletteBorderWidth + uniseg.StringWidth(query) + 1 + paletteBorderWidth
	sb.WriteString(strings.Repeat(" ", paletteWidth-paletteBorderWidth-usedWidth))
	sb.WriteString(string(blocks.ColorBorder))
	sb.WriteString("│")
	sb.WriteString(string(blocks.ColorReset))
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
	sb.WriteString(string(blocks.ColorBorder))
	sb.WriteString("│")
	sb.WriteString("  ")
	sb.WriteString(string(blocks.ColorMuted))
	sb.WriteString(emptyMsg)
	sb.WriteString(strings.Repeat(" ", paletteWidth-paletteBorderWidth-paletteBorderWidth-uniseg.StringWidth(emptyMsg)-paletteBorderWidth))
	sb.WriteString(string(blocks.ColorBorder))
	sb.WriteString("│")
	sb.WriteString(string(blocks.ColorReset))
	sb.WriteString("\n")
}

// renderBottomBorder writes the bottom border.
func (r *PaletteRenderer) renderBottomBorder(sb *strings.Builder, paletteWidth, leftPad int) {
	sb.WriteString(strings.Repeat(" ", leftPad))
	sb.WriteString(string(blocks.ColorBorder))
	sb.WriteString("╰")
	sb.WriteString(strings.Repeat("─", paletteWidth-paletteBorderWidth))
	sb.WriteString("╯")
	sb.WriteString(string(blocks.ColorReset))
	sb.WriteString("\n")
}

// renderItem renders a single command item.
func (r *PaletteRenderer) renderItem(cmd Command, selected bool, paletteWidth, leftPad int) string {
	var sb strings.Builder

	sb.WriteString(strings.Repeat(" ", leftPad))
	sb.WriteString(string(blocks.ColorBorder))
	sb.WriteString("│")

	// Background invert if selected.
	if selected {
		sb.WriteString(colorInvert)
	}

	sb.WriteString(" ") // s2 padding start.
	sb.WriteRune(cmd.Icon())
	sb.WriteString("  ")
	sb.WriteString(string(blocks.ColorFg))
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
		sb.WriteString(string(blocks.ColorMuted))
		sb.WriteString(category)
		sb.WriteString(string(blocks.ColorReset))

		if selected {
			sb.WriteString(colorInvert)
		}

		sb.WriteString(" ") // s2 padding end.
	} else {
		// Not enough space, just pad to end.
		sb.WriteString(strings.Repeat(" ", paletteWidth-paletteBorderWidth-1-iconWidth-paletteBorderWidth-nameWidth-paletteBorderWidth))
	}

	if selected {
		sb.WriteString(string(blocks.ColorReset))
	}

	sb.WriteString(string(blocks.ColorBorder))
	sb.WriteString("│")
	sb.WriteString(string(blocks.ColorReset))
	sb.WriteString("\n")

	return sb.String()
}
