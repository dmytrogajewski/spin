// Package theme provides centralized color schemes and styling for the Spin TUI.
//
// The theme package supports multiple color schemes (dark, light, auto-detect)
// and integrates with the configuration system. It respects the NO_COLOR
// environment variable for accessibility.
//
// # Themes
//
// Three built-in themes are available:
//
//   - dark: Optimized for dark terminal backgrounds (default)
//   - light: Optimized for light terminal backgrounds
//   - auto: Automatically detects terminal background color
//
// # Usage
//
// Create a theme:
//
//	theme, err := theme.New("dark", false)
//	if err != nil {
//	    // Handle error
//	}
//
// Apply theme to UI components:
//
//	chat := ui.NewChatWithTheme(width, height, theme)
//	statusBar := ui.NewStatusBarWithTheme(width, theme)
//
// # NO_COLOR Support
//
// The theme system respects the NO_COLOR environment variable for accessibility.
// When NO_COLOR is set, all colors are disabled and a plain theme is used:
//
//	NO_COLOR=1 spin tui
//
// This can also be configured via config file:
//
//	appearance:
//	  no_color: true
//
// # Configuration
//
// Themes can be configured via:
//
//   - Config file: ~/.spin/spin.yaml (appearance.theme)
//   - CLI flag: --theme dark|light|auto
//   - Environment: SPIN_APPEARANCE_THEME=dark
//   - NO_COLOR env var (disables all colors)
//
// # Performance
//
// All styles are pre-computed in the theme constructor for optimal performance.
// Style access (e.g., theme.ChatStyles()) returns cached style objects with
// no runtime overhead.
package theme
