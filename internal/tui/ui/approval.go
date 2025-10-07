package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dmytrogajewski/spin/internal/core"
)

// ApprovalDecisionMsg is sent when user makes an approval decision.
type ApprovalDecisionMsg struct {
	Response core.ApprovalResponse
}

type approvalRequestState struct {
	ID        string
	Command   string
	WorkDir   string
	Reason    string
	Timestamp time.Time
}

// ApprovalModal is a modal overlay for command approval.
// It displays a command that requires approval and allows the user to
// approve, deny, or modify the command before execution.
type ApprovalModal struct {
	request   approvalRequestState
	editing   bool
	editValue string
	editInput textinput.Model
	width     int
	height    int
}

// NewApprovalModal creates a new approval modal.
func NewApprovalModal(req core.ApprovalEventData, width, height int) ApprovalModal {
	ti := textinput.New()
	ti.Placeholder = "Enter modified command..."
	ti.CharLimit = 500

	return ApprovalModal{
		request: approvalRequestState{
			ID:        req.RequestID,
			Command:   req.Command,
			WorkDir:   req.WorkDir,
			Reason:    req.Reason,
			Timestamp: req.Timestamp,
		},
		editing:   false,
		editValue: "",
		editInput: ti,
		width:     width,
		height:    height,
	}
}

// SetSize updates the modal's dimensions.
func (m *ApprovalModal) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Update handles Bubble Tea messages.
func (m ApprovalModal) Update(msg tea.Msg) (ApprovalModal, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle edit mode separately
		if m.editing {
			return m.handleEditMode(msg)
		}

		// Handle approval/deny/modify keys
		switch msg.String() {
		case "a", "A":
			// Approve
			return m, m.approve()

		case "d", "D":
			// Deny
			return m, m.deny()

		case "m", "M":
			// Enter modification mode
			m.editing = true
			m.editValue = m.request.Command
			m.editInput.SetValue(m.editValue)
			m.editInput.Focus()
			return m, textinput.Blink
		}
	}

	return m, nil
}

// handleEditMode processes keyboard input during command editing.
func (m ApprovalModal) handleEditMode(msg tea.KeyMsg) (ApprovalModal, tea.Cmd) {
	var cmd tea.Cmd

	switch msg.Type {
	case tea.KeyEnter:
		// Confirm modification and approve
		m.editValue = m.editInput.Value()
		return m, m.approveWithModification()

	case tea.KeyEsc:
		// Cancel modification
		m.editing = false
		m.editValue = ""
		m.editInput.SetValue("")
		m.editInput.Blur()
		return m, nil
	}

	// Update text input
	m.editInput, cmd = m.editInput.Update(msg)
	m.editValue = m.editInput.Value()

	return m, cmd
}

// approve creates an approval response.
func (m ApprovalModal) approve() tea.Cmd {
	return func() tea.Msg {
		return ApprovalDecisionMsg{
			Response: core.ApprovalResponse{
				RequestID: m.request.ID,
				Approved:  true,
				Reason:    "user approved via TUI",
				Timestamp: time.Now(),
			},
		}
	}
}

// deny creates a denial response.
func (m ApprovalModal) deny() tea.Cmd {
	return func() tea.Msg {
		return ApprovalDecisionMsg{
			Response: core.ApprovalResponse{
				RequestID: m.request.ID,
				Approved:  false,
				Reason:    "user denied via TUI",
				Timestamp: time.Now(),
			},
		}
	}
}

// approveWithModification creates an approval response with a modified command.
func (m ApprovalModal) approveWithModification() tea.Cmd {
	return func() tea.Msg {
		return ApprovalDecisionMsg{
			Response: core.ApprovalResponse{
				RequestID:       m.request.ID,
				Approved:        true,
				Reason:          "user approved with modification via TUI",
				ModifiedCommand: m.editValue,
				Timestamp:       time.Now(),
			},
		}
	}
}

// View renders the approval modal.
func (m ApprovalModal) View() string {
	if m.editing {
		return m.renderEditMode()
	}
	return m.renderApprovalMode()
}

// renderApprovalMode renders the normal approval dialog.
func (m ApprovalModal) renderApprovalMode() string {
	// Calculate modal dimensions
	modalWidth := minInt(70, m.width-4)
	modalHeight := 12

	// Styles
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("62")).
		Padding(0, 1).
		Width(modalWidth)

	contentStyle := lipgloss.NewStyle().
		Padding(1, 2).
		Width(modalWidth - 4)

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Width(modalWidth).
		Height(modalHeight)

	actionStyle := lipgloss.NewStyle().
		Bold(true).
		Padding(0, 1).
		Width(modalWidth)

	// Build content
	var content strings.Builder

	// Title
	content.WriteString(titleStyle.Render("🔒 Command Approval"))
	content.WriteString("\n\n")

	// Command
	cmdStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("11"))
	content.WriteString(contentStyle.Render(
		fmt.Sprintf("Command:  %s", cmdStyle.Render(m.request.Command)),
	))
	content.WriteString("\n")

	// Working directory
	dirStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	content.WriteString(contentStyle.Render(
		fmt.Sprintf("Directory: %s", dirStyle.Render(m.request.WorkDir)),
	))
	content.WriteString("\n\n")

	// Reason
	reasonStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	content.WriteString(contentStyle.Render(
		fmt.Sprintf("Reason: %s", reasonStyle.Render(m.request.Reason)),
	))
	content.WriteString("\n\n")

	// Actions
	approveStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	denyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	modifyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("11"))

	actions := fmt.Sprintf("%s   %s   %s",
		approveStyle.Render("[A]pprove"),
		denyStyle.Render("[D]eny"),
		modifyStyle.Render("[M]odify"),
	)
	content.WriteString(actionStyle.Render(actions))

	// Wrap in border
	return borderStyle.Render(content.String())
}

// renderEditMode renders the command editing interface.
func (m ApprovalModal) renderEditMode() string {
	// Calculate modal dimensions
	modalWidth := minInt(70, m.width-4)
	modalHeight := 8

	// Styles
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("11")).
		Padding(0, 1).
		Width(modalWidth)

	contentStyle := lipgloss.NewStyle().
		Padding(1, 2).
		Width(modalWidth - 4)

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("11")).
		Width(modalWidth).
		Height(modalHeight)

	hintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Italic(true)

	// Build content
	var content strings.Builder

	// Title
	content.WriteString(titleStyle.Render("✏️  Modify Command"))
	content.WriteString("\n\n")

	// Input
	content.WriteString(contentStyle.Render(m.editInput.View()))
	content.WriteString("\n\n")

	// Hint
	hint := hintStyle.Render("Press Enter to approve, Esc to cancel")
	content.WriteString(contentStyle.Render(hint))

	// Wrap in border
	return borderStyle.Render(content.String())
}

// minInt returns the minimum of two integers.
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// NewApproval creates a new empty approval modal (for tests).
func NewApproval() ApprovalModal {
	ti := textinput.New()
	ti.Placeholder = "Enter modified command..."
	ti.CharLimit = 500

	return ApprovalModal{
		request:   approvalRequestState{},
		editing:   false,
		editValue: "",
		editInput: ti,
		width:     80,
		height:    24,
	}
}

// SetRequest sets the approval request.
func (m *ApprovalModal) SetRequest(req core.ApprovalEventData) {
	m.request = approvalRequestState{
		ID:        req.RequestID,
		Command:   req.Command,
		WorkDir:   req.WorkDir,
		Reason:    req.Reason,
		Timestamp: req.Timestamp,
	}
}

// Clear clears the approval state.
func (m *ApprovalModal) Clear() {
	m.request = approvalRequestState{}
	m.editing = false
	m.editValue = ""
	m.editInput.SetValue("")
	m.editInput.Blur()
}

// Request returns the current approval request (for testing).
func (m ApprovalModal) Request() *core.ApprovalEventData {
	if m.request.ID == "" {
		return nil
	}
	return &core.ApprovalEventData{
		RequestID: m.request.ID,
		Command:   m.request.Command,
		WorkDir:   m.request.WorkDir,
		Reason:    m.request.Reason,
		Status:    "pending",
		Timestamp: m.request.Timestamp,
	}
}
