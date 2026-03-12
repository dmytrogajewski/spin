package overlay

import (
	"context"

	"github.com/dmytrogajewski/spin/internal/security"
)

// ApprovalDialog represents a TUI approval dialog for command approval.
type ApprovalDialog struct {
	request    *security.ApprovalRequest
	response   *security.ApprovalResponse
	width      int
	height     int
	selected   int  // 0=Approve, 1=Deny, 2=Cancel.
	shown      bool // tracks if Show() has been called.
	responseCh chan security.ApprovalResponse
}

// NewApprovalDialog creates a new approval dialog.
func NewApprovalDialog(request security.ApprovalRequest) *ApprovalDialog {
	return &ApprovalDialog{
		request:    &request,
		selected:   0, // Default to Approve.
		responseCh: make(chan security.ApprovalResponse, 1),
	}
}

// SetDimensions sets the dialog dimensions.
func (d *ApprovalDialog) SetDimensions(width, height int) {
	d.width = width
	d.height = height
}

// Render renders the approval dialog.
// Returns empty string since we now use status bar instead of modal dialog.
func (d *ApprovalDialog) Render(width, height int) string {
	// No longer render modal dialog - status bar handles the display.
	return ""
}

// HandleKey handles keyboard input for the dialog.
// Returns true if the dialog should close, false otherwise.
func (d *ApprovalDialog) HandleKey(key string) bool {
	if len(key) == 0 {
		return false
	}

	// Check first character of key.
	switch key[0] {
	case 'A', 'a':
		// Approve once.
		d.approveWithScope(security.ScopeOnce)

		return true
	case 'S', 's':
		// Approve for session.
		d.approveWithScope(security.ScopeSession)

		return true
	case 'G', 'g':
		// Approve always (global).
		d.approveWithScope(security.ScopeGlobal)

		return true
	case 'D', 'd':
		d.Deny()

		return true
	case '\x1b': // ESC.
		// Cancel - deny with canceled reason.
		resp := security.ApprovalResponse{
			RequestID: d.request.ID,
			Approved:  false,
			Reason:    "canceled",
		}

		d.response = &resp
		select {
		case d.responseCh <- resp:
		default:
		}

		return true
	case '?':
		// Help - don't close.
		return false
	default:
		// Other keys - don't close.
		return false
	}
}

// GetResponse returns the user's response.
func (d *ApprovalDialog) GetResponse() *security.ApprovalResponse {
	return d.response
}

// Show displays the approval dialog and waits for user input.
func (d *ApprovalDialog) Show(ctx context.Context) security.ApprovalResponse {
	// Mark as shown.
	d.shown = true

	// Wait for response from Approve/Deny or context cancellation.
	select {
	case resp := <-d.responseCh:
		d.response = &resp

		return resp
	case <-ctx.Done():
		// Context canceled.
		resp := security.ApprovalResponse{
			RequestID: d.request.ID,
			Approved:  false,
			Reason:    "canceled",
		}
		d.response = &resp

		return resp
	}
}

// IsVisible returns whether the dialog is currently visible.
func (d *ApprovalDialog) IsVisible() bool {
	// Dialog is visible if Show() has been called and not responded to.
	return d.shown && d.response == nil
}

// Approve approves the request and closes the dialog.
func (d *ApprovalDialog) Approve() {
	resp := security.ApprovalResponse{
		RequestID: d.request.ID,
		Approved:  true,
		Reason:    "user approved",
	}
	d.response = &resp
	// Send to channel if Show() is waiting.
	select {
	case d.responseCh <- resp:
	default:
		// Channel already has a response or not being read.
	}
}

// approveWithScope approves the request with a specific persistence scope and closes the dialog.
func (d *ApprovalDialog) approveWithScope(scope string) {
	resp := security.ApprovalResponse{
		RequestID: d.request.ID,
		Approved:  true,
		Reason:    "user approved",
		Scope:     scope,
	}
	d.response = &resp
	// Send to channel if Show() is waiting.
	select {
	case d.responseCh <- resp:
	default:
		// Channel already has a response or not being read.
	}
}

// Deny denies the request and closes the dialog.
func (d *ApprovalDialog) Deny() {
	resp := security.ApprovalResponse{
		RequestID: d.request.ID,
		Approved:  false,
		Reason:    "user denied",
	}
	d.response = &resp
	// Send to channel if Show() is waiting.
	select {
	case d.responseCh <- resp:
	default:
		// Channel already has a response or not being read.
	}
}
