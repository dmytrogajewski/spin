package theme

import "github.com/charmbracelet/lipgloss"

// ColorScheme defines the color palette for a theme.
type ColorScheme struct {
	// Role colors
	User      lipgloss.Color
	Assistant lipgloss.Color
	System    lipgloss.Color
	Tool      lipgloss.Color

	// State colors
	Error   lipgloss.Color
	Success lipgloss.Color
	Warning lipgloss.Color
	Info    lipgloss.Color

	// UI element colors
	Background   lipgloss.Color
	Foreground   lipgloss.Color
	Border       lipgloss.Color
	BorderActive lipgloss.Color
	Selection    lipgloss.Color
	Highlight    lipgloss.Color

	// Status bar colors
	StatusBarBg     lipgloss.Color
	StatusBarFg     lipgloss.Color
	StatusBarActive lipgloss.Color
	StatusBarError  lipgloss.Color
}

// ChatStyleSet contains all styles for the chat component.
type ChatStyleSet struct {
	User       lipgloss.Style
	Assistant  lipgloss.Style
	System     lipgloss.Style
	Tool       lipgloss.Style
	ToolCall   lipgloss.Style // Border style for tool calls
	ToolResult lipgloss.Style // Border style for results
	Reasoning  lipgloss.Style // Border style for reasoning blocks
	Error      lipgloss.Style // Border style for errors
	Highlight  lipgloss.Style // Border style for highlighted messages
}

// StatusBarStyleSet contains all styles for the status bar component.
type StatusBarStyleSet struct {
	Normal lipgloss.Style
	Active lipgloss.Style
	Error  lipgloss.Style
}

// ApprovalStyleSet contains all styles for the approval modal.
type ApprovalStyleSet struct {
	Modal       lipgloss.Style
	Title       lipgloss.Style
	Command     lipgloss.Style
	Reason      lipgloss.Style
	ButtonBase  lipgloss.Style
	ButtonFocus lipgloss.Style
}

// HelpStyleSet contains all styles for the help modal.
type HelpStyleSet struct {
	Modal    lipgloss.Style
	Title    lipgloss.Style
	Section  lipgloss.Style
	Shortcut lipgloss.Style
	Desc     lipgloss.Style
}

// FilePickerStyleSet contains all styles for the file picker widget.
type FilePickerStyleSet struct {
	Modal    lipgloss.Style
	Title    lipgloss.Style
	Selected lipgloss.Style
	Normal   lipgloss.Style
	Matched  lipgloss.Style
}

// InputStyleSet contains all styles for the input widget.
type InputStyleSet struct {
	Normal      lipgloss.Style
	Focused     lipgloss.Style
	Placeholder lipgloss.Style
}
