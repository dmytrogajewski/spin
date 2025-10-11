package theme

// EightColorTheme implements the Theme interface with 8-color fallback.
// Uses basic ANSI 8/16 color codes for maximum compatibility.
type EightColorTheme struct{}

// NewEightColorTheme creates a new 8-color fallback theme.
func NewEightColorTheme() Theme {
	return &EightColorTheme{}
}

// 8-color fallback map per spec:
// fg→white, bg→black, muted→brightBlack, border→brightBlack
// blue→blue, green→green, yellow→yellow, red→red, magenta→magenta, cyan→cyan

// Fg returns white (color 7).
func (t *EightColorTheme) Fg() string {
	return fg8(7) // White
}

// Bg returns black (color 0).
func (t *EightColorTheme) Bg() string {
	return "\x1b[40m" // Black background
}

// Muted returns bright black / gray (color 8).
func (t *EightColorTheme) Muted() string {
	return fg8(8) // Bright black (gray)
}

// Border returns bright black / gray (color 8).
func (t *EightColorTheme) Border() string {
	return fg8(8) // Bright black (gray)
}

// Shadow returns bright black / gray (color 8).
func (t *EightColorTheme) Shadow() string {
	return fg8(8) // Bright black (gray)
}

// Blue returns blue (color 4).
func (t *EightColorTheme) Blue() string {
	return fg8(4) // Blue
}

// Green returns green (color 2).
func (t *EightColorTheme) Green() string {
	return fg8(2) // Green
}

// Yellow returns yellow (color 3).
func (t *EightColorTheme) Yellow() string {
	return fg8(3) // Yellow
}

// Red returns red (color 1).
func (t *EightColorTheme) Red() string {
	return fg8(1) // Red
}

// Magenta returns magenta (color 5).
func (t *EightColorTheme) Magenta() string {
	return fg8(5) // Magenta
}

// Cyan returns cyan (color 6).
func (t *EightColorTheme) Cyan() string {
	return fg8(6) // Cyan
}

// Bold returns the bold text attribute.
func (t *EightColorTheme) Bold() string {
	return "\x1b[1m"
}

// Dim returns the dim text attribute.
func (t *EightColorTheme) Dim() string {
	return "\x1b[2m"
}

// Reset returns the reset all attributes code.
func (t *EightColorTheme) Reset() string {
	return "\x1b[0m"
}
