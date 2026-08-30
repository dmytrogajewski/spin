package overlay

import (
	"context"
	"sync"

	"github.com/dmytrogajewski/spin/internal/safety"
)

// ApprovalDialog represents a TUI approval dialog for command approval.
// HandleKey and Show run on different goroutines, so response state is
// guarded by mu.
type ApprovalDialog struct {
	request    *safety.ApprovalRequest
	response   *safety.ApprovalResponse
	width      int
	height     int
	selected   int  // 0=Approve, 1=Deny, 2=Cancel.
	shown      bool // tracks if Show() has been called.
	responseCh chan safety.ApprovalResponse
	mu         sync.Mutex
}

// NewApprovalDialog creates a new approval dialog.
func NewApprovalDialog(request safety.ApprovalRequest) *ApprovalDialog {
	return &ApprovalDialog{
		request:    &request,
		selected:   0, // Default to Approve.
		responseCh: make(chan safety.ApprovalResponse, 1),
	}
}

// SetDimensions sets the dialog dimensions.
func (d *ApprovalDialog) SetDimensions(width, height int) {
	d.width = width
	d.height = height
}

// Render renders the approval dialog.
// Returns empty string since we now use status bar instead of modal dialog.
func (d *ApprovalDialog) Render(_, _ int) string {
	// No longer render modal dialog - status bar handles the display.
	return ""
}

// HandleKey handles keyboard input for the dialog.
// Returns true if the dialog should close, false otherwise.
func (d *ApprovalDialog) HandleKey(key string) bool {
	if key == "" {
		return false
	}

	// Check first character of key.
	switch key[0] {
	case 'A', 'a':
		// Approve once.
		d.approveWithScope(safety.ScopeOnce)

		return true
	case 'S', 's':
		// Approve for session.
		d.approveWithScope(safety.ScopeSession)

		return true
	case 'G', 'g':
		// Approve always (global).
		d.approveWithScope(safety.ScopeGlobal)

		return true
	case 'D', 'd':
		d.Deny()

		return true
	case '\x1b': // ESC.
		// Cancel - deny with canceled reason.
		d.respond(false, "canceled", "")

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
func (d *ApprovalDialog) GetResponse() *safety.ApprovalResponse {
	d.mu.Lock()
	defer d.mu.Unlock()

	return d.response
}

// Show displays the approval dialog and waits for user input.
func (d *ApprovalDialog) Show(ctx context.Context) safety.ApprovalResponse {
	d.mu.Lock()
	d.shown = true
	d.mu.Unlock()

	// Wait for response from Approve/Deny or context cancellation.
	select {
	case resp := <-d.responseCh:
		d.setResponse(resp)

		return resp
	case <-ctx.Done():
		// Context canceled.
		resp := safety.ApprovalResponse{
			RequestID: d.request.ID,
			Approved:  false,
			Reason:    "canceled",
		}
		d.setResponse(resp)

		return resp
	}
}

// IsVisible returns whether the dialog is currently visible.
func (d *ApprovalDialog) IsVisible() bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Dialog is visible if Show() has been called and not responded to.
	return d.shown && d.response == nil
}

// setResponse records the response under the mutex.
func (d *ApprovalDialog) setResponse(resp safety.ApprovalResponse) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.response = &resp
}

// Approve approves the request and closes the dialog.
func (d *ApprovalDialog) Approve() {
	d.respond(true, "user approved", "")
}

// approveWithScope approves the request with a specific persistence scope and closes the dialog.
func (d *ApprovalDialog) approveWithScope(scope string) {
	d.respond(true, "user approved", scope)
}

// Deny denies the request and closes the dialog.
func (d *ApprovalDialog) Deny() {
	d.respond(false, "user denied", "")
}

// respond sends an approval response and closes the dialog.
func (d *ApprovalDialog) respond(approved bool, reason, scope string) {
	resp := safety.ApprovalResponse{
		RequestID: d.request.ID,
		Approved:  approved,
		Reason:    reason,
		Scope:     scope,
	}
	d.setResponse(resp)
	// Send to channel if Show() is waiting.
	select {
	case d.responseCh <- resp:
	default:
		// Channel already has a response or not being read.
	}
}
