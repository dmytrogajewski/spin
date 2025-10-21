package theme

// TerminalCapability represents the color rendering capability of the terminal.
type TerminalCapability int

const (
	// TerminalCapability8Color represents basic 8/16 ANSI colors.
	TerminalCapability8Color TerminalCapability = iota
	// TerminalCapability256Color represents 256-color support.
	TerminalCapability256Color
	// TerminalCapabilityTrueColor represents 24-bit true color support.
	TerminalCapabilityTrueColor
)

// Theme defines color values for the TUI.
type Theme interface {
	// Neutral colors
	Fg() string     // Foreground text
	Bg() string     // Background
	Muted() string  // Muted/dimmed text
	Border() string // Border characters
	Shadow() string // Shadow/overlay background

	// Accent colors
	Blue() string    // Primary accent
	Green() string   // Success
	Yellow() string  // Warning
	Red() string     // Error/danger
	Magenta() string // Secondary accent
	Cyan() string    // Info/highlight
}


// NewDarkTheme creates a dark theme using 256 colors.

// NewLightTheme creates a light theme using 256 colors.

// NewEightColorTheme creates a basic 8-color theme for maximum compatibility.

// DetectTerminalCapabilities detects the color capability of the current terminal.

// NewTheme creates a theme based on name and terminal capabilities.
// themeName can be "dark" or "light" (defaults to "dark" for unknown values).
// For 8-color terminals, returns the 8-color fallback theme.

// GetThemeFromEnv creates a theme based on environment variables.
// Reads SPIN_THEME (default: "dark") and auto-detects terminal capabilities.

// hexToANSI256 converts a hex color string to the closest ANSI 256 color code.
// The 256-color palette consists of:
// - 0-15: Standard colors (same as 8/16 color mode)
// - 16-231: 6x6x6 color cube (216 colors)
// - 232-255: Grayscale ramp (24 shades)

// isGrayscale checks if RGB values are close enough to be considered grayscale.

// abs returns the absolute value of x.

// grayscaleToANSI256 maps a grayscale value (0-255) to the grayscale ramp (232-255).

// ansi256Color wraps a 256-color code in ANSI escape sequence.
