package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Help represents the help modal overlay displaying keyboard shortcuts.
type Help struct {
	width  int
	height int
}

// NewHelp creates a new Help modal component.
func NewHelp(width, height int) Help {
	return Help{
		width:  width,
		height: height,
	}
}

// SetSize updates the dimensions of the help modal.
func (h *Help) SetSize(width, height int) {
	h.width = width
	h.height = height
}

// View renders the help modal as a centered overlay.
func (h Help) View() string {
	// Modal styling
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("12")). // Blue
		Align(lipgloss.Center).
		Width(h.width - 4)

	contentStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("7")). // Light gray
		Padding(0, 2)

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("12")). // Blue border
		Padding(1, 2).
		Width(h.width - 10). // Leave margin
		Align(lipgloss.Center)

	// Help content
	content := h.buildHelpContent()

	// Build modal
	modal := lipgloss.JoinVertical(
		lipgloss.Center,
		titleStyle.Render("Keyboard Shortcuts"),
		"",
		contentStyle.Render(content),
		"",
		contentStyle.Render("Press any key to close"),
	)

	// Apply border
	bordered := borderStyle.Render(modal)

	// Center vertically (simple approach)
	lines := strings.Split(bordered, "\n")
	modalHeight := len(lines)
	topPadding := (h.height - modalHeight) / 2
	if topPadding < 0 {
		topPadding = 0
	}

	// Add vertical padding
	var padded strings.Builder
	for i := 0; i < topPadding; i++ {
		padded.WriteString("\n")
	}
	padded.WriteString(bordered)

	return padded.String()
}

// buildHelpContent creates the formatted help text.
func (h Help) buildHelpContent() string {
	var b strings.Builder

	sections := []struct {
		title string
		items []helpItem
	}{
		{
			title: "Global",
			items: []helpItem{
				{"Enter", "Send message"},
				{"Ctrl+C", "Cancel turn / Exit"},
				{"Ctrl+D", "Exit Spin"},
				{"Ctrl+L", "Clear screen"},
				{"Ctrl+H / ?", "Show this help"},
			},
		},
		{
			title: "Navigation",
			items: []helpItem{
				{"Esc-Esc", "Enter backtrack mode"},
				{"@", "Open file picker"},
				{"PgUp/PgDn", "Scroll transcript"},
				{"Home/End", "Jump to top/bottom"},
			},
		},
		{
			title: "Tool Approval",
			items: []helpItem{
				{"A", "Approve"},
				{"D", "Deny"},
				{"M", "Modify command"},
			},
		},
	}

	for i, section := range sections {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(h.formatSection(section.title, section.items))
	}

	return b.String()
}

// helpItem represents a single keyboard shortcut entry.
type helpItem struct {
	key         string
	description string
}

// formatSection formats a section of help items.
func (h Help) formatSection(title string, items []helpItem) string {
	var b strings.Builder

	// Section title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("10")). // Green
		Underline(true)

	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n")

	// Key-value pairs
	keyStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("14")). // Cyan
		Width(15).
		Align(lipgloss.Left)

	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("7")). // Light gray
		Width(30).
		Align(lipgloss.Left)

	for _, item := range items {
		key := keyStyle.Render(item.key)
		desc := descStyle.Render(item.description)
		b.WriteString(key)
		b.WriteString(" - ")
		b.WriteString(desc)
		b.WriteString("\n")
	}

	return b.String()
}
