package theme

import "github.com/charmbracelet/lipgloss"

// lightTheme implements Theme for light terminal backgrounds.
type lightTheme struct {
	colors     ColorScheme
	chat       ChatStyleSet
	statusBar  StatusBarStyleSet
	approval   ApprovalStyleSet
	help       HelpStyleSet
	filePicker FilePickerStyleSet
	input      InputStyleSet
}

// newLightTheme creates a new light theme with pre-computed styles.
func newLightTheme() *lightTheme {
	colors := ColorScheme{
		// Role colors (darker colors for light background)
		User:      lipgloss.Color("4"), // Dark blue
		Assistant: lipgloss.Color("2"), // Dark green
		System:    lipgloss.Color("3"), // Dark yellow/brown
		Tool:      lipgloss.Color("6"), // Dark cyan

		// State colors
		Error:   lipgloss.Color("1"), // Dark red
		Success: lipgloss.Color("2"), // Dark green
		Warning: lipgloss.Color("3"), // Dark yellow
		Info:    lipgloss.Color("4"), // Dark blue

		// UI element colors
		Background:   lipgloss.Color("15"), // White
		Foreground:   lipgloss.Color("0"),  // Black
		Border:       lipgloss.Color("8"),  // Gray
		BorderActive: lipgloss.Color("4"),  // Blue
		Selection:    lipgloss.Color("12"), // Light blue bg
		Highlight:    lipgloss.Color("3"),  // Dark yellow

		// Status bar
		StatusBarBg:     lipgloss.Color("7"), // Light gray
		StatusBarFg:     lipgloss.Color("0"), // Black
		StatusBarActive: lipgloss.Color("2"), // Green
		StatusBarError:  lipgloss.Color("1"), // Red
	}

	// Pre-compute chat styles
	chat := ChatStyleSet{
		User: lipgloss.NewStyle().
			Foreground(colors.User).
			Bold(true),
		Assistant: lipgloss.NewStyle().
			Foreground(colors.Assistant).
			Bold(true),
		System: lipgloss.NewStyle().
			Foreground(colors.System).
			Bold(true),
		Tool: lipgloss.NewStyle().
			Foreground(colors.Tool).
			Bold(true),
		ToolCall: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("3")). // Dark yellow
			Padding(0, 1),
		ToolResult: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("2")). // Dark green
			Padding(0, 1),
		Reasoning: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("5")). // Magenta
			Padding(0, 1),
		Error: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("1")). // Dark red
			Padding(0, 1),
		Highlight: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("4")). // Dark blue
			Padding(0, 1),
	}

	// Pre-compute status bar styles
	statusBar := StatusBarStyleSet{
		Normal: lipgloss.NewStyle().
			Background(colors.StatusBarBg).
			Foreground(colors.StatusBarFg).
			Padding(0, 1),
		Active: lipgloss.NewStyle().
			Background(colors.StatusBarBg).
			Foreground(colors.StatusBarActive).
			Padding(0, 1),
		Error: lipgloss.NewStyle().
			Background(colors.StatusBarBg).
			Foreground(colors.StatusBarError).
			Padding(0, 1),
	}

	// Pre-compute approval styles
	approval := ApprovalStyleSet{
		Modal: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colors.BorderActive).
			Padding(1, 2).
			Width(70),
		Title: lipgloss.NewStyle().
			Foreground(colors.Warning).
			Bold(true),
		Command: lipgloss.NewStyle().
			Foreground(colors.Info).
			Background(lipgloss.Color("7")).
			Padding(0, 1),
		Reason: lipgloss.NewStyle().
			Foreground(colors.Warning),
		ButtonBase: lipgloss.NewStyle().
			Foreground(colors.Foreground).
			Background(lipgloss.Color("7")).
			Padding(0, 2),
		ButtonFocus: lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")).
			Background(colors.BorderActive).
			Padding(0, 2).
			Bold(true),
	}

	// Pre-compute help styles
	help := HelpStyleSet{
		Modal: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colors.BorderActive).
			Padding(1, 2),
		Title: lipgloss.NewStyle().
			Foreground(colors.Info).
			Bold(true),
		Section: lipgloss.NewStyle().
			Foreground(colors.Warning).
			Bold(true),
		Shortcut: lipgloss.NewStyle().
			Foreground(colors.User).
			Background(lipgloss.Color("7")).
			Padding(0, 1),
		Desc: lipgloss.NewStyle().
			Foreground(colors.Foreground),
	}

	// Pre-compute file picker styles
	filePicker := FilePickerStyleSet{
		Modal: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colors.BorderActive).
			Padding(1, 2),
		Title: lipgloss.NewStyle().
			Foreground(colors.Info).
			Bold(true),
		Selected: lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")).
			Background(colors.BorderActive).
			Bold(true),
		Normal: lipgloss.NewStyle().
			Foreground(colors.Foreground),
		Matched: lipgloss.NewStyle().
			Foreground(colors.Warning).
			Bold(true),
	}

	// Pre-compute input styles
	input := InputStyleSet{
		Normal: lipgloss.NewStyle().
			Foreground(colors.Foreground),
		Focused: lipgloss.NewStyle().
			Foreground(colors.Foreground).
			Border(lipgloss.NormalBorder()).
			BorderForeground(colors.BorderActive),
		Placeholder: lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")), // Gray
	}

	return &lightTheme{
		colors:     colors,
		chat:       chat,
		statusBar:  statusBar,
		approval:   approval,
		help:       help,
		filePicker: filePicker,
		input:      input,
	}
}

func (t *lightTheme) Name() string                         { return "light" }
func (t *lightTheme) Colors() ColorScheme                  { return t.colors }
func (t *lightTheme) ChatStyles() ChatStyleSet             { return t.chat }
func (t *lightTheme) StatusBarStyles() StatusBarStyleSet   { return t.statusBar }
func (t *lightTheme) ApprovalStyles() ApprovalStyleSet     { return t.approval }
func (t *lightTheme) HelpStyles() HelpStyleSet             { return t.help }
func (t *lightTheme) FilePickerStyles() FilePickerStyleSet { return t.filePicker }
func (t *lightTheme) InputStyles() InputStyleSet           { return t.input }
func (t *lightTheme) SupportsColors() bool                 { return true }
