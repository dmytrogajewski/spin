package overlay

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/dmytrogajewski/spin/internal/core"
)

// ApprovalDialog represents a TUI approval dialog for command approval.
type ApprovalDialog struct {
	request   *core.ApprovalRequest
	response  *core.ApprovalResponse
	width     int
	height    int
	selected  int // 0=Approve, 1=Deny, 2=Cancel
	timeout   time.Duration
	startTime time.Time
}

// NewApprovalDialog creates a new approval dialog.
func NewApprovalDialog(request core.ApprovalRequest, timeout time.Duration) *ApprovalDialog {
	return &ApprovalDialog{
		request:   &request,
		timeout:   timeout,
		selected:  0, // Default to Approve
		startTime: time.Now(),
	}
}

// SetDimensions sets the dialog dimensions.
func (d *ApprovalDialog) SetDimensions(width, height int) {
	d.width = width
	d.height = height
}

// Render renders the approval dialog.
func (d *ApprovalDialog) Render(width, height int) string {
	var sb strings.Builder

	// Calculate dialog dimensions
	dialogWidth := min(width-4, 80)
	dialogHeight := min(height-4, 20)

	// Center the dialog
	startX := (width - dialogWidth) / 2
	startY := (height - dialogHeight) / 2

	// Draw dialog border
	for y := 0; y < dialogHeight; y++ {
		if y == 0 || y == dialogHeight-1 {
			// Top and bottom borders
			sb.WriteString(fmt.Sprintf("\033[%d;%dH", startY+y, startX))
			sb.WriteString("┌" + strings.Repeat("─", dialogWidth-2) + "┐")
		} else {
			// Side borders
			sb.WriteString(fmt.Sprintf("\033[%d;%dH", startY+y, startX))
			sb.WriteString("│" + strings.Repeat(" ", dialogWidth-2) + "│")
		}
	}

	// Dialog title
	title := "Command Approval Required"
	sb.WriteString(fmt.Sprintf("\033[%d;%dH", startY, startX+(dialogWidth-len(title))/2))
	sb.WriteString(fmt.Sprintf("\033[1m%s\033[0m", title))

	// Command information
	contentY := startY + 2
	sb.WriteString(fmt.Sprintf("\033[%d;%dH", contentY, startX+2))
	sb.WriteString(fmt.Sprintf("Command: %s", d.request.Command))

	contentY++
	sb.WriteString(fmt.Sprintf("\033[%d;%dH", contentY, startX+2))
	sb.WriteString(fmt.Sprintf("Reason: %s", d.request.Reason))

	if d.request.WorkDir != "" {
		contentY++
		sb.WriteString(fmt.Sprintf("\033[%d;%dH", contentY, startX+2))
		sb.WriteString(fmt.Sprintf("Working Directory: %s", d.request.WorkDir))
	}

	// Timeout information
	contentY += 2
	remaining := d.timeout - time.Since(d.startTime)
	if remaining > 0 {
		sb.WriteString(fmt.Sprintf("\033[%d;%dH", contentY, startX+2))
		sb.WriteString(fmt.Sprintf("Timeout: %s", remaining.Round(time.Second)))
	}

	// Action buttons
	buttonY := startY + dialogHeight - 3
	buttonX := startX + 2

	// Approve button
	approveText := "Approve (A)"
	if d.selected == 0 {
		sb.WriteString(fmt.Sprintf("\033[%d;%dH", buttonY, buttonX))
		sb.WriteString(fmt.Sprintf("\033[7m%s\033[0m", approveText))
	} else {
		sb.WriteString(fmt.Sprintf("\033[%d;%dH", buttonY, buttonX))
		sb.WriteString(approveText)
	}

	// Deny button
	denyText := "Deny (D)"
	if d.selected == 1 {
		sb.WriteString(fmt.Sprintf("\033[%d;%dH", buttonY, buttonX+20))
		sb.WriteString(fmt.Sprintf("\033[7m%s\033[0m", denyText))
	} else {
		sb.WriteString(fmt.Sprintf("\033[%d;%dH", buttonY, buttonX+20))
		sb.WriteString(denyText)
	}

	// Cancel button
	cancelText := "Cancel (ESC)"
	if d.selected == 2 {
		sb.WriteString(fmt.Sprintf("\033[%d;%dH", buttonY, buttonX+40))
		sb.WriteString(fmt.Sprintf("\033[7m%s\033[0m", cancelText))
	} else {
		sb.WriteString(fmt.Sprintf("\033[%d;%dH", buttonY, buttonX+40))
		sb.WriteString(cancelText)
	}

	// Help text
	helpY := startY + dialogHeight - 1
	sb.WriteString(fmt.Sprintf("\033[%d;%dH", helpY, startX+2))
	sb.WriteString("Use arrow keys to navigate, Enter to select, or press A/D/ESC")

	return sb.String()
}

// HandleKey handles keyboard input for the dialog.
func (d *ApprovalDialog) HandleKey(key string) bool {
	switch key {
	case "left", "h":
		d.selected = (d.selected - 1 + 3) % 3
		return true
	case "right", "l":
		d.selected = (d.selected + 1) % 3
		return true
	case "a", "A":
		d.selected = 0
		d.response = &core.ApprovalResponse{
			Approved: true,
			Reason:   "User approved via TUI dialog",
		}
		return false // Close dialog
	case "d", "D":
		d.selected = 1
		d.response = &core.ApprovalResponse{
			Approved: false,
			Reason:   "User denied via TUI dialog",
		}
		return false // Close dialog
	case "escape", "q", "Q":
		d.selected = 2
		d.response = &core.ApprovalResponse{
			Approved: false,
			Reason:   "User cancelled via TUI dialog",
		}
		return false // Close dialog
	case "enter", " ":
		switch d.selected {
		case 0: // Approve
			d.response = &core.ApprovalResponse{
				Approved: true,
				Reason:   "User approved via TUI dialog",
			}
		case 1: // Deny
			d.response = &core.ApprovalResponse{
				Approved: false,
				Reason:   "User denied via TUI dialog",
			}
		case 2: // Cancel
			d.response = &core.ApprovalResponse{
				Approved: false,
				Reason:   "User cancelled via TUI dialog",
			}
		}
		return false // Close dialog
	}
	return true // Keep dialog open
}

// GetResponse returns the user's response.
func (d *ApprovalDialog) GetResponse() *core.ApprovalResponse {
	return d.response
}

// IsExpired checks if the dialog has expired due to timeout.
func (d *ApprovalDialog) IsExpired() bool {
	return time.Since(d.startTime) > d.timeout
}

// GetTimeoutResponse returns a timeout response.
func (d *ApprovalDialog) GetTimeoutResponse() *core.ApprovalResponse {
	return &core.ApprovalResponse{
		Approved: false,
		Reason:   "Approval dialog timed out",
	}
}

// Show displays the approval dialog and waits for user input.
func (d *ApprovalDialog) Show(ctx context.Context) core.ApprovalResponse {
	// For now, return a default approval response
	// In a real implementation, this would handle keyboard input and display
	return core.ApprovalResponse{
		Approved: true,
		Reason:   "Auto-approved (TUI integration placeholder)",
	}
}
