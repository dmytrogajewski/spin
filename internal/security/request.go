package security

import (
	"context"
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

	// ToolCallID is the LLM tool call ID (e.g., "call-0") when this approval
	// request is associated with a tool call. This allows approval notifications
	// to use the same tool call ID as the tool call events.
	ToolCallID string
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

	// Scope indicates how long the approval should persist.
	// Supported values: "once", "session", "global"
	Scope string

	// TTL is an optional time-to-live for persisted approvals (session/global).
	// If nil, a default is applied by the persistence layer.
	TTL *time.Duration

	// PolicyNote is an optional human-readable note saved with the persisted policy.
	PolicyNote string

	// Timestamp is when the response was created
	Timestamp time.Time
}

// ApprovalHandler is a callback function for handling approval requests.
// It receives a context and an ApprovalRequest, and must return an ApprovalResponse.
// The handler should block until the user makes a decision, timeout occurs, or context is cancelled.
// The handler should respect context cancellation and return promptly when ctx.Done() is signaled.
// If the handler is nil, commands requiring approval are automatically denied.
type ApprovalHandler func(ctx context.Context, req ApprovalRequest) ApprovalResponse
