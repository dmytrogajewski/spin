package security

import (
	"time"
)

// ApprovalRequest represents a command approval request sent to the approval handler.
type ApprovalRequest struct {
	// ID is a unique identifier for this approval request (UUID)
	ID string

	// Command is the command requiring approval
	Command *Command

	// Reason explains why approval is needed (from Validator)
	Reason string

	// WorkDir is the working directory where the command will execute
	WorkDir string

	// Timestamp is when the request was created
	Timestamp time.Time
}

// ApprovalResponse represents the user's approval decision.
type ApprovalResponse struct {
	// RequestID must match the ApprovalRequest.ID
	RequestID string

	// Approved indicates whether the command was approved (true) or denied (false)
	Approved bool

	// Reason is an optional user-provided reason for the decision
	Reason string

	// ModifiedCommand is an optional modified version of the command.
	// If provided, the original command will be replaced and re-validated.
	// If empty, the original command is used as-is.
	ModifiedCommand string

	// Timestamp is when the response was created
	Timestamp time.Time
}

// ApprovalHandler is a callback function for handling approval requests.
// It receives an ApprovalRequest and must return an ApprovalResponse.
// The handler should block until the user makes a decision or timeout occurs.
// If the handler is nil, commands requiring approval are automatically denied.
type ApprovalHandler func(ApprovalRequest) ApprovalResponse
