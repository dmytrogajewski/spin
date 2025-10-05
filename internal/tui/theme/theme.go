package theme

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Theme provides consistent styling for all TUI components.
type Theme interface {
	// Name returns the theme name ("dark", "light", "auto", "plain").
	Name() string

	// Colors returns the color scheme.
	Colors() ColorScheme

	// ChatStyles returns pre-computed styles for chat component.
	ChatStyles() ChatStyleSet

	// StatusBarStyles returns pre-computed styles for status bar.
	StatusBarStyles() StatusBarStyleSet

	// ApprovalStyles returns pre-computed styles for approval modal.
	ApprovalStyles() ApprovalStyleSet

	// HelpStyles returns pre-computed styles for help modal.
	HelpStyles() HelpStyleSet

	// FilePickerStyles returns pre-computed styles for file picker.
	FilePickerStyles() FilePickerStyleSet

	// InputStyles returns pre-computed styles for input widget.
	InputStyles() InputStyleSet

	// SupportsColors returns true if colors are enabled.
	SupportsColors() bool
}

// New creates a new theme based on name and NO_COLOR setting.
// Supported theme names: "dark", "light", "auto", "" (defaults to "dark").
// If noColor is true or NO_COLOR env var is set, returns a plain theme.
func New(name string, noColor bool) (Theme, error) {
	// Check NO_COLOR environment variable
	if noColor || os.Getenv("NO_COLOR") != "" {
		return newPlainTheme(), nil
	}

	// Default to dark theme if name is empty
	if name == "" {
		name = "dark"
	}

	switch name {
	case "dark":
		return newDarkTheme(), nil
	case "light":
		return newLightTheme(), nil
	case "auto":
		return newAutoTheme(), nil
	default:
		return nil, fmt.Errorf("unknown theme: %s (supported: dark, light, auto)", name)
	}
}

// detectDarkTerminal attempts to detect if terminal has dark background.
// Uses heuristics from COLORFGBG env var and TERM name.
func detectDarkTerminal() bool {
	// Method 1: Check $COLORFGBG environment variable (some terminals)
	// Format: "foreground;background"
	// Dark bg typically has low number (0-7), light bg has high number (8-15)
	if colorfgbg := os.Getenv("COLORFGBG"); colorfgbg != "" {
		parts := strings.Split(colorfgbg, ";")
		if len(parts) == 2 {
			if bg, err := strconv.Atoi(parts[1]); err == nil {
				return bg < 8 // Colors 0-7 are considered dark
			}
		}
	}

	// Method 2: Check terminal type
	term := os.Getenv("TERM")
	if strings.Contains(term, "dark") {
		return true
	}
	if strings.Contains(term, "light") {
		return false
	}

	// Method 3: Fallback to dark (most common for developers)
	return true
}
