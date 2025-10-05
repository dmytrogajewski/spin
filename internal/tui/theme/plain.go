package theme

import "github.com/charmbracelet/lipgloss"

// plainTheme implements Theme with no colors for accessibility (NO_COLOR support).
type plainTheme struct {
	colors     ColorScheme
	chat       ChatStyleSet
	statusBar  StatusBarStyleSet
	approval   ApprovalStyleSet
	help       HelpStyleSet
	filePicker FilePickerStyleSet
	input      InputStyleSet
}

// newPlainTheme creates a new plain theme with no colors.
func newPlainTheme() *plainTheme {
	// All colors are empty (no color)
	colors := ColorScheme{
		User:      lipgloss.Color(""),
		Assistant: lipgloss.Color(""),
		System:    lipgloss.Color(""),
		Tool:      lipgloss.Color(""),

		Error:   lipgloss.Color(""),
		Success: lipgloss.Color(""),
		Warning: lipgloss.Color(""),
		Info:    lipgloss.Color(""),

		Background:   lipgloss.Color(""),
		Foreground:   lipgloss.Color(""),
		Border:       lipgloss.Color(""),
		BorderActive: lipgloss.Color(""),
		Selection:    lipgloss.Color(""),
		Highlight:    lipgloss.Color(""),

		StatusBarBg:     lipgloss.Color(""),
		StatusBarFg:     lipgloss.Color(""),
		StatusBarActive: lipgloss.Color(""),
		StatusBarError:  lipgloss.Color(""),
	}

	// Styles with no colors, only structural formatting
	chat := ChatStyleSet{
		User:      lipgloss.NewStyle().Bold(true),
		Assistant: lipgloss.NewStyle().Bold(true),
		System:    lipgloss.NewStyle().Bold(true),
		Tool:      lipgloss.NewStyle().Bold(true),
		ToolCall: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(0, 1),
		ToolResult: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(0, 1),
		Reasoning: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(0, 1),
		Error: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(0, 1),
		Highlight: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(0, 1),
	}

	// Status bar styles (no colors)
	statusBar := StatusBarStyleSet{
		Normal: lipgloss.NewStyle().Padding(0, 1),
		Active: lipgloss.NewStyle().Padding(0, 1),
		Error:  lipgloss.NewStyle().Padding(0, 1),
	}

	// Approval styles (no colors)
	approval := ApprovalStyleSet{
		Modal: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(1, 2).
			Width(70),
		Title:       lipgloss.NewStyle().Bold(true),
		Command:     lipgloss.NewStyle().Padding(0, 1),
		Reason:      lipgloss.NewStyle(),
		ButtonBase:  lipgloss.NewStyle().Padding(0, 2),
		ButtonFocus: lipgloss.NewStyle().Padding(0, 2).Bold(true),
	}

	// Help styles (no colors)
	help := HelpStyleSet{
		Modal: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(1, 2),
		Title:    lipgloss.NewStyle().Bold(true),
		Section:  lipgloss.NewStyle().Bold(true),
		Shortcut: lipgloss.NewStyle().Padding(0, 1),
		Desc:     lipgloss.NewStyle(),
	}

	// File picker styles (no colors)
	filePicker := FilePickerStyleSet{
		Modal: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(1, 2),
		Title:    lipgloss.NewStyle().Bold(true),
		Selected: lipgloss.NewStyle().Bold(true),
		Normal:   lipgloss.NewStyle(),
		Matched:  lipgloss.NewStyle().Bold(true),
	}

	// Input styles (no colors)
	input := InputStyleSet{
		Normal: lipgloss.NewStyle(),
		Focused: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()),
		Placeholder: lipgloss.NewStyle(),
	}

	return &plainTheme{
		colors:     colors,
		chat:       chat,
		statusBar:  statusBar,
		approval:   approval,
		help:       help,
		filePicker: filePicker,
		input:      input,
	}
}

func (t *plainTheme) Name() string                         { return "plain" }
func (t *plainTheme) Colors() ColorScheme                  { return t.colors }
func (t *plainTheme) ChatStyles() ChatStyleSet             { return t.chat }
func (t *plainTheme) StatusBarStyles() StatusBarStyleSet   { return t.statusBar }
func (t *plainTheme) ApprovalStyles() ApprovalStyleSet     { return t.approval }
func (t *plainTheme) HelpStyles() HelpStyleSet             { return t.help }
func (t *plainTheme) FilePickerStyles() FilePickerStyleSet { return t.filePicker }
func (t *plainTheme) InputStyles() InputStyleSet           { return t.input }
func (t *plainTheme) SupportsColors() bool                 { return false }
