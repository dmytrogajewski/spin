package theme

// DarkTheme implements the Theme interface with the Dark color scheme.
// Uses 256-color ANSI codes as specified in the TUI specification.
type DarkTheme struct{}

// NewDarkTheme creates a new dark theme with 256-color support.
func NewDarkTheme() Theme {
	return &DarkTheme{}
}

// Neutral colors per spec:
// bg=#0b0e12, fg=#dde3ea, muted=#9aa4b2, border=#2d3640, shadow=#1a212a

// Fg returns the default foreground color (#dde3ea → ANSI 254).
func (t *DarkTheme) Fg() string {
	return fg256(hexToANSI256("#dde3ea"))
}

// Bg returns the default background color (#0b0e12 → ANSI 233).
func (t *DarkTheme) Bg() string {
	return bg256(hexToANSI256("#0b0e12"))
}

// Muted returns the dimmed text color (#9aa4b2 → ANSI 247).
func (t *DarkTheme) Muted() string {
	return fg256(hexToANSI256("#9aa4b2"))
}

// Border returns the border/separator color (#2d3640 → ANSI 237).
func (t *DarkTheme) Border() string {
	return fg256(hexToANSI256("#2d3640"))
}

// Shadow returns the very dim color (#1a212a → ANSI 235).
func (t *DarkTheme) Shadow() string {
	return fg256(hexToANSI256("#1a212a"))
}

// Accent colors per spec:
// blue=#5aa6ff, green=#57d98d, yellow=#f5c156, red=#ff6b6b, magenta=#d08bff, cyan=#7adcf3

// Blue returns the blue accent color (#5aa6ff → ANSI 75).
func (t *DarkTheme) Blue() string {
	return fg256(hexToANSI256("#5aa6ff"))
}

// Green returns the green accent color (#57d98d → ANSI 114).
func (t *DarkTheme) Green() string {
	return fg256(hexToANSI256("#57d98d"))
}

// Yellow returns the yellow accent color (#f5c156 → ANSI 221).
func (t *DarkTheme) Yellow() string {
	return fg256(hexToANSI256("#f5c156"))
}

// Red returns the red accent color (#ff6b6b → ANSI 203).
func (t *DarkTheme) Red() string {
	return fg256(hexToANSI256("#ff6b6b"))
}

// Magenta returns the magenta accent color (#d08bff → ANSI 177).
func (t *DarkTheme) Magenta() string {
	return fg256(hexToANSI256("#d08bff"))
}

// Cyan returns the cyan accent color (#7adcf3 → ANSI 117).
func (t *DarkTheme) Cyan() string {
	return fg256(hexToANSI256("#7adcf3"))
}

// Bold returns the bold text attribute.
func (t *DarkTheme) Bold() string {
	return "\x1b[1m"
}

// Dim returns the dim text attribute.
func (t *DarkTheme) Dim() string {
	return "\x1b[2m"
}

// Reset returns the reset all attributes code.
func (t *DarkTheme) Reset() string {
	return "\x1b[0m"
}
