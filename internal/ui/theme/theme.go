// Package theme provides theming support for the TUI with Dark, Light, and 8-color fallback themes.
package theme

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Theme defines the color scheme interface for TUI rendering.
// All color methods return ANSI escape sequences ready for terminal output.
type Theme interface {
	// Neutral colors
	Fg() string     // Default foreground color
	Bg() string     // Default background color
	Muted() string  // Dimmed text color
	Border() string // Border/separator color
	Shadow() string // Very dim color

	// Accent colors
	Blue() string    // Blue accent (EXECUTE, TESTING)
	Green() string   // Green accent (APPLY_PATCH, success)
	Yellow() string  // Yellow accent (GREP, warnings)
	Red() string     // Red accent (ERROR)
	Magenta() string // Magenta accent (PLAN)
	Cyan() string    // Cyan accent (READ, SUMMARY)

	// Utility
	Bold() string  // Bold text attribute
	Dim() string   // Dim text attribute
	Reset() string // Reset all attributes
}

// TerminalCapability represents the color capability of the terminal.
type TerminalCapability int

const (
	// TerminalCapability8Color represents terminals with 8 basic colors.
	TerminalCapability8Color TerminalCapability = 8
	// TerminalCapability256Color represents terminals with 256 colors.
	TerminalCapability256Color TerminalCapability = 256
	// TerminalCapabilityTrueColor represents terminals with 24-bit true color.
	TerminalCapabilityTrueColor TerminalCapability = 16777216
)

// NewTheme creates a new theme based on the name and terminal capability.
// If capability is 8-color, returns an 8-color fallback theme regardless of name.
// Otherwise returns the specified theme (dark or light).
func NewTheme(name string, capability TerminalCapability) Theme {
	// For 8-color terminals, always use 8-color fallback
	if capability == TerminalCapability8Color {
		return NewEightColorTheme()
	}

	// For 256-color and true-color terminals, use specified theme
	switch strings.ToLower(name) {
	case "light":
		return NewLightTheme()
	case "dark", "":
		return NewDarkTheme()
	default:
		return NewDarkTheme()
	}
}

// GetThemeFromEnv returns a theme based on environment variables.
// Reads SPIN_THEME (dark/light) and detects terminal capabilities.
// Falls back to dark theme with auto-detected capabilities.
func GetThemeFromEnv() Theme {
	themeName := os.Getenv("SPIN_THEME")
	if themeName == "" {
		themeName = "dark"
	}

	capability := DetectTerminalCapabilities()
	return NewTheme(themeName, capability)
}

// DetectTerminalCapabilities detects the color capability of the current terminal.
// Returns 8-color, 256-color, or true-color based on TERM and COLORTERM env vars.
func DetectTerminalCapabilities() TerminalCapability {
	// Check COLORTERM for true color support
	colorterm := os.Getenv("COLORTERM")
	if colorterm == "truecolor" || colorterm == "24bit" {
		return TerminalCapabilityTrueColor
	}

	// Check TERM for 256 color support
	term := os.Getenv("TERM")
	if strings.Contains(term, "256color") || strings.Contains(term, "256colour") {
		return TerminalCapability256Color
	}

	// Check for xterm-like terminals (usually support 256 colors)
	if strings.HasPrefix(term, "xterm") || strings.HasPrefix(term, "screen") {
		return TerminalCapability256Color
	}

	// Default to 8-color for unknown terminals
	return TerminalCapability8Color
}

// hexToANSI256 converts a hex color (#RRGGBB) to the nearest ANSI 256 color code.
// Uses a simplified color space conversion algorithm.
func hexToANSI256(hex string) int {
	// Remove leading # if present
	hex = strings.TrimPrefix(hex, "#")

	// Parse RGB values
	if len(hex) != 6 {
		return 15 // Default to white on error
	}

	r, _ := strconv.ParseInt(hex[0:2], 16, 64)
	g, _ := strconv.ParseInt(hex[2:4], 16, 64)
	b, _ := strconv.ParseInt(hex[4:6], 16, 64)

	// Check for grayscale (r ≈ g ≈ b, within tolerance)
	maxDiff := max(abs64(r-g), max(abs64(r-b), abs64(g-b)))
	if maxDiff < 8 {
		// Grayscale ramp (232-255, 24 shades)
		avg := (r + g + b) / 3
		if avg < 8 {
			return 16 // Black
		}
		if avg > 238 {
			return 231 // White
		}
		return int(232 + (avg-8)/10)
	}

	// Convert to 6x6x6 color cube (16-231)
	// Map 0-255 to 0-5 using proper quantization
	rIdx := quantize6(r)
	gIdx := quantize6(g)
	bIdx := quantize6(b)

	return int(16 + 36*rIdx + 6*gIdx + bIdx)
}

// quantize6 maps a 0-255 value to a 0-5 index for the 6x6x6 color cube.
func quantize6(val int64) int64 {
	// ANSI 256 color cube uses values: 0, 95, 135, 175, 215, 255
	// Map input to nearest index
	if val < 48 {
		return 0
	}
	if val < 115 {
		return 1
	}
	if val < 155 {
		return 2
	}
	if val < 195 {
		return 3
	}
	if val < 235 {
		return 4
	}
	return 5
}

// abs64 returns the absolute value of an int64.
func abs64(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

// max returns the maximum of two int64 values.
func max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

// fg256 returns an ANSI 256-color foreground escape sequence.
func fg256(code int) string {
	return fmt.Sprintf("\x1b[38;5;%dm", code)
}

// bg256 returns an ANSI 256-color background escape sequence.
func bg256(code int) string {
	return fmt.Sprintf("\x1b[48;5;%dm", code)
}

// fg8 returns an ANSI 8-color foreground escape sequence.
func fg8(code int) string {
	if code >= 8 {
		return fmt.Sprintf("\x1b[9%dm", code-8) // Bright colors (90-97)
	}
	return fmt.Sprintf("\x1b[3%dm", code) // Normal colors (30-37)
}
