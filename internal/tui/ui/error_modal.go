package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ErrorDisplay represents an error for display in the UI.
// This is a simplified version that UI components use.
// It's populated from internal/tui.ErrorDisplay.
type ErrorDisplay struct {
	Message     string
	Code        string
	Details     string
	Operation   string
	Severity    int // 0=info, 1=warning, 2=error, 3=critical
	Timestamp   string
	Dismissible bool
	Dismissed   bool
	AutoDismiss int // seconds (0 = never)
}

// ErrorModal displays critical errors in a centered modal overlay.
// It supports multiple errors with navigation, dismissal actions,
// and scrolling for long error messages.
type ErrorModal struct {
	// Errors is the list of errors to display.
	Errors []ErrorDisplay
	// CurrentIdx is the index of the currently displayed error.
	CurrentIdx int
	// Width is the modal width in columns.
	Width int
	// Height is the modal height in rows.
	Height int
	// Visible indicates if the modal is currently shown.
	Visible bool
}

// NewErrorModal creates a new ErrorModal with the given dimensions.
func NewErrorModal(width, height int) ErrorModal {
	return ErrorModal{
		Errors:     make([]ErrorDisplay, 0),
		CurrentIdx: 0,
		Width:      width,
		Height:     height,
		Visible:    false,
	}
}

// Show displays the modal with the given error.
// The error is appended to the error history and becomes current.
func (m *ErrorModal) Show(err ErrorDisplay) {
	m.Errors = append(m.Errors, err)
	m.CurrentIdx = len(m.Errors) - 1
	m.Visible = true
}

// Hide dismisses the modal.
// Errors are preserved in history.
func (m *ErrorModal) Hide() {
	m.Visible = false
}

// PrevError navigates to the previous (older) error.
func (m *ErrorModal) PrevError() {
	if m.CurrentIdx > 0 {
		m.CurrentIdx--
	}
}

// NextError navigates to the next (newer) error.
func (m *ErrorModal) NextError() {
	if m.CurrentIdx < len(m.Errors)-1 {
		m.CurrentIdx++
	}
}

// CurrentError returns the currently displayed error, or nil if none.
func (m *ErrorModal) CurrentError() *ErrorDisplay {
	if len(m.Errors) == 0 || m.CurrentIdx >= len(m.Errors) {
		return nil
	}
	return &m.Errors[m.CurrentIdx]
}

// ErrorCount returns the total number of errors in history.
func (m *ErrorModal) ErrorCount() int {
	return len(m.Errors)
}

// Clear removes all errors and hides the modal.
func (m *ErrorModal) Clear() {
	m.Errors = make([]ErrorDisplay, 0)
	m.CurrentIdx = 0
	m.Visible = false
}

// Resize updates the modal dimensions.
func (m *ErrorModal) Resize(width, height int) {
	m.Width = width
	m.Height = height
}

// Update handles keyboard input for the error modal (Bubble Tea pattern).
// Returns updated model and optional command.
func (m ErrorModal) Update(msg tea.Msg) (ErrorModal, tea.Cmd) {
	// Ignore all input when hidden
	if !m.Visible {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc, tea.KeyEnter:
			// Dismiss modal
			m.Hide()
			return m, nil

		case tea.KeyUp:
			// Navigate to previous error
			m.PrevError()
			return m, nil

		case tea.KeyDown:
			// Navigate to next error
			m.NextError()
			return m, nil
		}
	}

	return m, nil
}

// View renders the modal (Bubble Tea pattern).
// Returns empty string if hidden or no errors.
func (m ErrorModal) View() string {
	if !m.Visible || len(m.Errors) == 0 {
		return ""
	}

	err := m.CurrentError()
	if err == nil {
		return ""
	}

	// Define styles
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("9")). // Red border
		Padding(1, 2).
		Width(m.Width - 4). // Account for border
		MaxWidth(60)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("9")). // Red
		MarginBottom(1)

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")). // Gray
		Bold(true)

	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("15")) // White

	detailsStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("7")). // Light gray
		MarginTop(1).
		MarginBottom(1)

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")). // Gray
		MarginTop(1)

	// Build error content
	var content strings.Builder

	// Title
	icon := m.getIcon(err.Severity)
	content.WriteString(titleStyle.Render(fmt.Sprintf("%s Error", icon)))
	content.WriteString("\n\n")

	// Message
	content.WriteString(labelStyle.Render("Message: "))
	content.WriteString(valueStyle.Render(err.Message))
	content.WriteString("\n")

	// Operation
	if err.Operation != "" {
		content.WriteString(labelStyle.Render("Operation: "))
		content.WriteString(valueStyle.Render(err.Operation))
		content.WriteString("\n")
	}

	// Code
	content.WriteString(labelStyle.Render("Code: "))
	content.WriteString(valueStyle.Render(err.Code))
	content.WriteString("\n")

	// Timestamp
	if err.Timestamp != "" {
		content.WriteString(labelStyle.Render("Time: "))
		content.WriteString(valueStyle.Render(err.Timestamp))
		content.WriteString("\n")
	}

	// Details (if present)
	if err.Details != "" {
		content.WriteString("\n")
		content.WriteString(labelStyle.Render("Details:"))
		content.WriteString("\n")
		content.WriteString(detailsStyle.Render(err.Details))
	}

	// Navigation help (if multiple errors)
	if len(m.Errors) > 1 {
		navInfo := fmt.Sprintf("Error %d of %d", m.CurrentIdx+1, len(m.Errors))
		content.WriteString("\n")
		content.WriteString(helpStyle.Render(navInfo))
		content.WriteString("\n")
		content.WriteString(helpStyle.Render("[↑↓] Navigate  [Esc] Dismiss"))
	} else {
		content.WriteString("\n")
		content.WriteString(helpStyle.Render("[Esc] Dismiss  [Enter] Close"))
	}

	// Render with box
	box := boxStyle.Render(content.String())

	// Center the box
	centered := lipgloss.Place(
		m.Width,
		m.Height,
		lipgloss.Center,
		lipgloss.Center,
		box,
	)

	return centered
}

// getIcon returns the emoji icon for error severity.
func (m ErrorModal) getIcon(severity int) string {
	switch severity {
	case 0:
		return "ℹ️" // Info
	case 1:
		return "⚠️" // Warning
	case 2:
		return "❌" // Error
	case 3:
		return "🔥" // Critical
	default:
		return "❓"
	}
}
