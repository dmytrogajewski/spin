// Package theme provides color theming for the TUI.
//
// # Overview
//
// The theme package supports three color modes:
// - Dark theme (default): 256-color optimized for dark backgrounds
// - Light theme: 256-color optimized for light backgrounds
// - 8-color fallback: Basic ANSI colors for maximum compatibility
//
// # Usage
//
// Create a theme based on environment:
//
//	theme := theme.GetThemeFromEnv() // Reads SPIN_THEME env var
//
// Or create a specific theme:
//
//	darkTheme := theme.NewDarkTheme()
//	lightTheme := theme.NewLightTheme()
//	fallbackTheme := theme.NewEightColorTheme()
//
// Or use the factory with auto-detection:
//
//	theme := theme.NewTheme("dark", theme.DetectTerminalCapabilities())
//
// # Environment Variables
//
// - SPIN_THEME: "dark" or "light" (default: "dark")
// - TERM: Used to detect 256-color support (e.g., "xterm-256color")
// - COLORTERM: Used to detect true color support (e.g., "truecolor")
//
// # Integration with Renderers
//
// Renderers accept an optional theme parameter. When nil, they use legacy
// hardcoded colors for backward compatibility:
//
//	// Legacy (backward compatible):
//	renderer := blocks.NewRenderer(width)
//
//	// With theme:
//	theme := theme.GetThemeFromEnv()
//	renderer := blocks.NewRendererWithTheme(width, theme)
//
// # Color Conversion
//
// The package automatically converts hex colors to ANSI 256-color codes
// using a 6x6x6 color cube and grayscale ramp. The conversion is approximate
// but provides good visual results for the specified theme colors.
package theme
