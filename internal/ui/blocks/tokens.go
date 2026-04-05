package blocks

// Spacing constants define the spacing scale in terminal cells.
// These match the design tokens from the TUI specification.
const (
	// S0 is zero spacing (no gap).
	S0 = 0
	// S1 is 1 cell spacing.
	S1 = 1
	// S2 is 2 cell spacing (typical margin).
	S2 = 2
	// S3 is 3 cell spacing (gap between components).
	S3 = 3
	// S4 is 4 cell spacing (indent level).
	S4 = 4
	// S6 is 6 cell spacing (medium gap).
	S6 = 6
	// S8 is 8 cell spacing (large gap).
	S8 = 8
	// S12 is 12 cell spacing (extra large gap).
	S12 = 12
)

// Color represents an ANSI escape code for terminal colors.
type Color string

// String returns the ANSI escape code as a string.
func (c Color) String() string {
	return string(c)
}

// Color constants define the color palette per TUI specification.
const (
	// ColorReset resets all attributes.
	ColorReset Color = "\x1b[0m"

	// ColorBold enables bold text.
	ColorBold Color = "\x1b[1m"

	// ColorDim enables dim text.
	ColorDim Color = "\x1b[2m"

	// ColorFg is the default foreground color (light gray).
	ColorFg Color = "\x1b[38;5;252m"

	// ColorBg is the default background color (dark gray).
	ColorBg Color = "\x1b[48;5;233m"

	// ColorMuted is dimmed text color (medium gray).
	ColorMuted Color = "\x1b[38;5;244m"

	// ColorBorder is the border/separator color (dark gray).
	ColorBorder Color = "\x1b[38;5;238m"

	// ColorShadow is very dim color (very dark gray).
	ColorShadow Color = "\x1b[38;5;235m"

	// ColorBlue is the blue accent (EXECUTE, TESTING).
	ColorBlue Color = "\x1b[38;5;39m"

	// ColorGreen is the green accent (APPLY_PATCH, success).
	ColorGreen Color = "\x1b[38;5;42m"

	// ColorYellow is the yellow accent (GREP, warnings).
	ColorYellow Color = "\x1b[38;5;221m"

	// ColorRed is the red accent (ERROR).
	ColorRed Color = "\x1b[38;5;203m"

	// ColorMagenta is the magenta accent (PLAN).
	ColorMagenta Color = "\x1b[38;5;170m"

	// ColorCyan is the cyan accent (READ, SUMMARY).
	ColorCyan Color = "\x1b[38;5;51m"
)

// TagColors maps block types to their accent colors.
var TagColors = map[BlockType]Color{
	BlockTypeExecute:    ColorBlue,
	BlockTypePlan:       ColorMagenta,
	BlockTypeRead:       ColorCyan,
	BlockTypeGrep:       ColorYellow,
	BlockTypeApplyPatch: ColorGreen,
	BlockTypeSummary:    ColorCyan,
	BlockTypeTool:       ColorCyan,
	BlockTypeTesting:    ColorBlue,
	BlockTypeNotice:     ColorMuted,
	BlockTypeError:      ColorRed,
}

// GetTagColor returns the accent color for a given block type.
// Returns ColorMuted if the block type is unknown.
func GetTagColor(bt BlockType) Color {
	if color, ok := TagColors[bt]; ok {
		return color
	}

	return ColorMuted
}
