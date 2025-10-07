package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// renderPlaceholder renders the TUI view with all components.
func (m Model) renderChat() string {
	// Render chat component (Phase 3.2)
	chatView := m.chat.View()

	// Add spinner line if active
	if m.spinner.IsActive() {
		spinnerLine := m.spinner.ViewWithText("Thinking...")
		chatView = lipgloss.JoinVertical(lipgloss.Top, chatView, "", spinnerLine)
	}

	// Render input widget (Phase 3.3)
	inputView := m.input.View()

	// Render status bar (Phase 3.6)
	statusView := m.statusBar.View()

	// Join vertically: chat, input, status
	base := lipgloss.JoinVertical(
		lipgloss.Top,
		chatView,
		inputView,
		statusView,
	)

	// Overlay approval modal if in approval state (Phase 3.5)
	if m.state == StateToolApproval {
		approvalOverlay := m.approval.View()
		if approvalOverlay != "" {
			return lipgloss.Place(
				m.width,
				m.height,
				lipgloss.Center,
				lipgloss.Center,
				approvalOverlay,
				lipgloss.WithWhitespaceChars(" "),
				lipgloss.WithWhitespaceForeground(lipgloss.Color("0")),
			)
		}
	}

	// Overlay error modal if visible (Phase 3.12)
	errorOverlay := m.errorModal.View()
	if errorOverlay != "" {
		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			errorOverlay,
			lipgloss.WithWhitespaceChars(" "),
			lipgloss.WithWhitespaceForeground(lipgloss.Color("0")),
		)
	}

	return base
}
