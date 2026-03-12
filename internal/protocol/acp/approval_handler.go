package acp

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/coder/acp-go-sdk"

	"github.com/dmytrogajewski/spin/internal/security"
)

// ApprovalHandler coordinates approval requests between Spin's approval service
// and ACP clients. When a tool needs approval, it calls the client's RequestPermission
// method and waits for the client's response with the selected option.
type ApprovalHandler struct {
	mu            sync.RWMutex
	agent         *SpinACPAgent
	timeout       time.Duration
	activeSession acp.SessionId // Currently active session for approval requests.
}

// NewApprovalHandler creates a new ACP approval handler.
func NewApprovalHandler(agent *SpinACPAgent, timeout time.Duration) *ApprovalHandler {
	return &ApprovalHandler{
		agent:   agent,
		timeout: timeout,
	}
}

// SetActiveSession sets the currently active session for approval requests.
// This should be called at the start of each Prompt execution.
func (h *ApprovalHandler) SetActiveSession(sessionID acp.SessionId) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.activeSession = sessionID
}

// ClearActiveSession clears the active session.
// This should be called when a Prompt execution completes.
func (h *ApprovalHandler) ClearActiveSession() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.activeSession = ""
}

// HandleApprovalRequest handles an approval request by calling the client's RequestPermission
// method and waiting for the client's response. This implements the security.ApprovalHandler interface.
func (h *ApprovalHandler) HandleApprovalRequest(ctx context.Context, req security.ApprovalRequest) security.ApprovalResponse {
	// Get the active session ID.
	h.mu.RLock()
	sessionID := h.activeSession
	h.mu.RUnlock()

	if sessionID == "" {
		return security.ApprovalResponse{
			RequestID: req.ID,
			Approved:  false,
			Reason:    "no active session",
		}
	}

	// Extract tool name from command.
	toolName := "unknown"
	if req.Command != nil {
		toolName = req.Command.Program
	}

	// Convert approval request to ACP tool call.
	toolCall, err := h.convertApprovalRequestToToolCall(req)
	if err != nil {
		return security.ApprovalResponse{
			RequestID: req.ID,
			Approved:  false,
			Reason:    fmt.Sprintf("conversion error: %v", err),
		}
	}

	// Get connection to call client.
	h.agent.mu.RLock()
	conn := h.agent.connection
	h.agent.mu.RUnlock()

	if conn == nil {
		return security.ApprovalResponse{
			RequestID: req.ID,
			Approved:  false,
			Reason:    "no connection to client",
		}
	}

	// Build permission options
	// For write_file and other dangerous operations, provide allow/deny options.
	options := []acp.PermissionOption{
		{
			OptionId: acp.PermissionOptionId("allow_once"),
			Name:     "Allow Once",
			Kind:     acp.PermissionOptionKindAllowOnce,
		},
		{
			OptionId: acp.PermissionOptionId("allow_always"),
			Name:     "Allow Always",
			Kind:     acp.PermissionOptionKindAllowAlways,
		},
		{
			OptionId: acp.PermissionOptionId("deny"),
			Name:     "Deny",
			Kind:     acp.PermissionOptionKindRejectOnce,
		},
		{
			OptionId: acp.PermissionOptionId("reject_always"),
			Name:     "Reject Always",
			Kind:     acp.PermissionOptionKindRejectAlways,
		},
	}

	// Send tool_call notification with status "pending" before requesting permission
	// This follows the ACP spec: "When the language model requests a tool invocation,
	// the Agent SHOULD report it to the Client" with status "pending" when awaiting approval
	// conn is guaranteed to be non-nil here (checked above)
	// Map tool name to kind (reuse logic from notifications.go).
	var kind *acp.ToolKind

	switch toolName {
	case "read_file":
		kind = acp.Ptr(acp.ToolKindRead)
	case "write_file":
		kind = acp.Ptr(acp.ToolKindEdit)
	case "shell_command":
		kind = acp.Ptr(acp.ToolKindExecute)
	case "file_search":
		kind = acp.Ptr(acp.ToolKindSearch)
	case "list_directory":
		kind = acp.Ptr(acp.ToolKindRead)
	}

	// Build update using SDK helper.
	update := acp.UpdateToolCall(
		toolCall.ToolCallId,
		acp.WithUpdateStatus(acp.ToolCallStatusPending),
		acp.WithUpdateTitle(toolName),
	)
	if kind != nil {
		update = acp.UpdateToolCall(
			toolCall.ToolCallId,
			acp.WithUpdateStatus(acp.ToolCallStatusPending),
			acp.WithUpdateKind(*kind),
			acp.WithUpdateTitle(toolName),
		)
	}

	notification := acp.SessionNotification{
		SessionId: sessionID,
		Update:    update,
	}
	// Use background context for notification (non-blocking).
	_ = conn.SessionUpdate(context.Background(), notification)

	// Create ACP permission request.
	acpReq := acp.RequestPermissionRequest{
		SessionId: sessionID,
		ToolCall:  toolCall,
		Options:   options,
	}

	// Call client's RequestPermission method
	// Derive timeout context from parent context to propagate cancellation.
	timeoutCtx, cancel := context.WithTimeout(ctx, h.timeout)
	defer cancel()

	acpResp, err := conn.RequestPermission(timeoutCtx, acpReq)
	if err != nil {
		// Check if context was canceled (either parent or timeout).
		if timeoutCtx.Err() != nil {
			return security.ApprovalResponse{
				RequestID: req.ID,
				Approved:  false,
				Reason:    "approval request canceled",
			}
		}

		return security.ApprovalResponse{
			RequestID: req.ID,
			Approved:  false,
			Reason:    fmt.Sprintf("client permission request failed: %v", err),
		}
	}

	// Extract selected option from response.
	approved := false

	var scope string

	if acpResp.Outcome.Selected != nil {
		selectedID := acpResp.Outcome.Selected.OptionId
		// Check if selected option is an allow option.
		for _, opt := range options {
			if opt.OptionId == selectedID {
				switch opt.Kind {
				case acp.PermissionOptionKindAllowOnce:
					approved = true
					scope = security.ScopeOnce
				case acp.PermissionOptionKindAllowAlways:
					approved = true
					// Map "always" to global persistence by default.
					scope = security.ScopeGlobal
				case acp.PermissionOptionKindRejectOnce:
					approved = false
					scope = security.ScopeOnce
				case acp.PermissionOptionKindRejectAlways:
					approved = false
					// For symmetry, treat as global persistent deny (not persisted yet).
					scope = security.ScopeGlobal
				}

				break
			}
		}
	}

	return security.ApprovalResponse{
		RequestID: req.ID,
		Approved:  approved,
		Reason:    "client decision",
		Scope:     scope,
	}
}

// convertApprovalRequestToToolCall converts a security approval request to an ACP tool call.
func (h *ApprovalHandler) convertApprovalRequestToToolCall(req security.ApprovalRequest) (acp.RequestPermissionToolCall, error) {
	// Extract tool name from command.
	toolName := "unknown"
	if req.Command != nil {
		toolName = req.Command.Program
	}

	// Use tool call ID from request if available, otherwise use approval request ID.
	toolCallID := req.ToolCallID
	if toolCallID == "" {
		toolCallID = req.ID
	}

	// Create tool call.
	toolCall := acp.RequestPermissionToolCall{
		ToolCallId: acp.ToolCallId(toolCallID),
		Title:      acp.Ptr(toolName),
	}

	// Note: ACP RequestPermissionToolCall doesn't have a Description field
	// The reason is typically included in the permission options or handled by the client.

	return toolCall, nil
}
