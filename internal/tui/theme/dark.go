package theme

import "github.com/charmbracelet/lipgloss"

// darkTheme implements Theme for dark terminal backgrounds.
type darkTheme struct {
	colors     ColorScheme
	chat       ChatStyleSet
	statusBar  StatusBarStyleSet
	approval   ApprovalStyleSet
	help       HelpStyleSet
	filePicker FilePickerStyleSet
	input      InputStyleSet
}

// newDarkTheme creates a new dark theme with pre-computed styles.
func newDarkTheme() *darkTheme {
	colors := ColorScheme{
		// Role colors (bright colors for dark background)
		User:      lipgloss.Color("12"), // Bright blue
		Assistant: lipgloss.Color("10"), // Bright green
		System:    lipgloss.Color("11"), // Bright yellow
		Tool:      lipgloss.Color("14"), // Bright cyan

		// State colors
		Error:   lipgloss.Color("9"),  // Bright red
		Success: lipgloss.Color("10"), // Bright green
		Warning: lipgloss.Color("11"), // Bright yellow
		Info:    lipgloss.Color("14"), // Bright cyan

		// UI element colors
		Background:   lipgloss.Color("0"),   // Black
		Foreground:   lipgloss.Color("7"),   // White
		Border:       lipgloss.Color("240"), // Dark gray
		BorderActive: lipgloss.Color("12"),  // Blue
		Selection:    lipgloss.Color("4"),   // Blue bg
		Highlight:    lipgloss.Color("226"), // Yellow

		// Status bar
		StatusBarBg:     lipgloss.Color("236"), // Very dark gray
		StatusBarFg:     lipgloss.Color("250"), // Light gray
		StatusBarActive: lipgloss.Color("10"),  // Green
		StatusBarError:  lipgloss.Color("9"),   // Red
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
			BorderForeground(lipgloss.Color("226")). // Yellow
			Padding(0, 1),
		ToolResult: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("10")). // Green
			Padding(0, 1),
		Reasoning: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")). // Purple
			Padding(0, 1),
		Error: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("9")). // Red
			Padding(0, 1),
		Highlight: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("12")). // Blue
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
			Background(lipgloss.Color("234")).
			Padding(0, 1),
		Reason: lipgloss.NewStyle().
			Foreground(colors.Warning),
		ButtonBase: lipgloss.NewStyle().
			Foreground(colors.Foreground).
			Background(lipgloss.Color("238")).
			Padding(0, 2),
		ButtonFocus: lipgloss.NewStyle().
			Foreground(lipgloss.Color("0")).
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
			Background(lipgloss.Color("234")).
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
			Foreground(lipgloss.Color("0")).
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
			Foreground(lipgloss.Color("8")), // Dark gray
	}

	return &darkTheme{
		colors:     colors,
		chat:       chat,
		statusBar:  statusBar,
		approval:   approval,
		help:       help,
		filePicker: filePicker,
		input:      input,
	}
}

func (t *darkTheme) Name() string                         { return "dark" }
func (t *darkTheme) Colors() ColorScheme                  { return t.colors }
func (t *darkTheme) ChatStyles() ChatStyleSet             { return t.chat }
func (t *darkTheme) StatusBarStyles() StatusBarStyleSet   { return t.statusBar }
func (t *darkTheme) ApprovalStyles() ApprovalStyleSet     { return t.approval }
func (t *darkTheme) HelpStyles() HelpStyleSet             { return t.help }
func (t *darkTheme) FilePickerStyles() FilePickerStyleSet { return t.filePicker }
func (t *darkTheme) InputStyles() InputStyleSet           { return t.input }
func (t *darkTheme) SupportsColors() bool                 { return true }
