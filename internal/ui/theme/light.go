package theme

// LightTheme implements the Theme interface with the Light color scheme.
// Uses 256-color ANSI codes as specified in the TUI specification.
type LightTheme struct{}

// NewLightTheme creates a new light theme with 256-color support.
func NewLightTheme() Theme {
	return &LightTheme{}
}

// Neutral colors per spec:
// bg=#f7f9fc, fg=#1e2a35, muted=#6b7580, border=#cfd6de, shadow=#e9eef3

// Fg returns the default foreground color (#1e2a35 → ANSI 237).
func (t *LightTheme) Fg() string {
	return fg256(hexToANSI256("#1e2a35"))
}

// Bg returns the default background color (#f7f9fc → ANSI 255).
func (t *LightTheme) Bg() string {
	return bg256(hexToANSI256("#f7f9fc"))
}

// Muted returns the dimmed text color (#6b7580 → ANSI 243).
func (t *LightTheme) Muted() string {
	return fg256(hexToANSI256("#6b7580"))
}

// Border returns the border/separator color (#cfd6de → ANSI 252).
func (t *LightTheme) Border() string {
	return fg256(hexToANSI256("#cfd6de"))
}

// Shadow returns the very dim color (#e9eef3 → ANSI 254).
func (t *LightTheme) Shadow() string {
	return fg256(hexToANSI256("#e9eef3"))
}

// Accent colors per spec:
// blue=#2a7fff, green=#0dbf6f, yellow=#c28a00, red=#d23a3a, magenta=#8e4dff, cyan=#1ca8c7

// Blue returns the blue accent color (#2a7fff → ANSI 33).
func (t *LightTheme) Blue() string {
	return fg256(hexToANSI256("#2a7fff"))
}

// Green returns the green accent color (#0dbf6f → ANSI 35).
func (t *LightTheme) Green() string {
	return fg256(hexToANSI256("#0dbf6f"))
}

// Yellow returns the yellow accent color (#c28a00 → ANSI 178).
func (t *LightTheme) Yellow() string {
	return fg256(hexToANSI256("#c28a00"))
}

// Red returns the red accent color (#d23a3a → ANSI 167).
func (t *LightTheme) Red() string {
	return fg256(hexToANSI256("#d23a3a"))
}

// Magenta returns the magenta accent color (#8e4dff → ANSI 99).
func (t *LightTheme) Magenta() string {
	return fg256(hexToANSI256("#8e4dff"))
}

// Cyan returns the cyan accent color (#1ca8c7 → ANSI 38).
func (t *LightTheme) Cyan() string {
	return fg256(hexToANSI256("#1ca8c7"))
}

// Bold returns the bold text attribute.
func (t *LightTheme) Bold() string {
	return "\x1b[1m"
}

// Dim returns the dim text attribute.
func (t *LightTheme) Dim() string {
	return "\x1b[2m"
}

// Reset returns the reset all attributes code.
func (t *LightTheme) Reset() string {
	return "\x1b[0m"
}
