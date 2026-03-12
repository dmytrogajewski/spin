package acp

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/coder/acp-go-sdk"

	"github.com/dmytrogajewski/spin/internal/security"
)

var (
	// ErrNoActiveSession is returned when no active session exists.
	ErrNoActiveSession = errors.New("no active session")
	// ErrNoConnectionToClient is returned when there is no client connection.
	ErrNoConnectionToClient = errors.New("no connection to client")
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
	sessionID, conn, err := h.getSessionAndConnection()
	if err != nil {
		return h.denyResponse(req.ID, err.Error())
	}

	toolName := extractToolName(req)

	toolCall, err := h.convertApprovalRequestToToolCall(req)
	if err != nil {
		return h.denyResponse(req.ID, fmt.Sprintf("conversion error: %v", err))
	}

	options := buildPermissionOptions()

	h.sendPendingNotification(ctx, sessionID, conn, toolCall.ToolCallId, toolName)

	acpResp, err := h.requestPermission(ctx, conn, sessionID, toolCall, options)
	if err != nil {
		return h.handlePermissionError(ctx, req.ID, err)
	}

	return h.buildApprovalResponse(req.ID, acpResp, options)
}

// getSessionAndConnection returns the active session ID and connection, or an error.
func (h *ApprovalHandler) getSessionAndConnection() (acp.SessionId, notificationSender, error) {
	h.mu.RLock()
	sessionID := h.activeSession
	h.mu.RUnlock()

	if sessionID == "" {
		return "", nil, ErrNoActiveSession
	}

	h.agent.mu.RLock()
	conn := h.agent.connection
	h.agent.mu.RUnlock()

	if conn == nil {
		return "", nil, ErrNoConnectionToClient
	}

	return sessionID, conn, nil
}

// extractToolName extracts the tool name from a request.
func extractToolName(req security.ApprovalRequest) string {
	if req.Command != nil {
		return req.Command.Program
	}

	return unknownValue
}

// denyResponse creates a denied approval response.
func (h *ApprovalHandler) denyResponse(reqID, reason string) security.ApprovalResponse {
	return security.ApprovalResponse{
		RequestID: reqID,
		Approved:  false,
		Reason:    reason,
	}
}

// buildPermissionOptions returns the standard permission options.
func buildPermissionOptions() []acp.PermissionOption {
	return []acp.PermissionOption{
		{OptionId: acp.PermissionOptionId("allow_once"), Name: "Allow Once", Kind: acp.PermissionOptionKindAllowOnce},
		{OptionId: acp.PermissionOptionId("allow_always"), Name: "Allow Always", Kind: acp.PermissionOptionKindAllowAlways},
		{OptionId: acp.PermissionOptionId("deny"), Name: "Deny", Kind: acp.PermissionOptionKindRejectOnce},
		{OptionId: acp.PermissionOptionId("reject_always"), Name: "Reject Always", Kind: acp.PermissionOptionKindRejectAlways},
	}
}

// sendPendingNotification sends a pending tool call notification.
func (h *ApprovalHandler) sendPendingNotification(
	ctx context.Context, sessionID acp.SessionId, conn notificationSender,
	toolCallID acp.ToolCallId, toolName string,
) {
	kind := mapToolNameToKind(toolName)

	update := acp.UpdateToolCall(
		toolCallID,
		acp.WithUpdateStatus(acp.ToolCallStatusPending),
		acp.WithUpdateTitle(toolName),
	)
	if kind != nil {
		update = acp.UpdateToolCall(
			toolCallID,
			acp.WithUpdateStatus(acp.ToolCallStatusPending),
			acp.WithUpdateKind(*kind),
			acp.WithUpdateTitle(toolName),
		)
	}

	notification := acp.SessionNotification{
		SessionId: sessionID,
		Update:    update,
	}
	_ = conn.SessionUpdate(ctx, notification)
}

// mapToolNameToKind maps a tool name to an ACP tool kind.
func mapToolNameToKind(toolName string) *acp.ToolKind {
	switch toolName {
	case "read_file", "list_directory":
		return acp.Ptr(acp.ToolKindRead)
	case toolWriteFile:
		return acp.Ptr(acp.ToolKindEdit)
	case "shell_command":
		return acp.Ptr(acp.ToolKindExecute)
	case "file_search":
		return acp.Ptr(acp.ToolKindSearch)
	default:
		return nil
	}
}

// requestPermission sends the permission request to the client.
func (h *ApprovalHandler) requestPermission(
	ctx context.Context, conn notificationSender, sessionID acp.SessionId,
	toolCall acp.RequestPermissionToolCall, options []acp.PermissionOption,
) (acp.RequestPermissionResponse, error) {
	acpReq := acp.RequestPermissionRequest{
		SessionId: sessionID,
		ToolCall:  toolCall,
		Options:   options,
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, h.timeout)
	defer cancel()

	return conn.RequestPermission(timeoutCtx, acpReq)
}

// handlePermissionError handles errors from the permission request.
func (h *ApprovalHandler) handlePermissionError(ctx context.Context, reqID string, err error) security.ApprovalResponse {
	if ctx.Err() != nil {
		return h.denyResponse(reqID, "approval request canceled")
	}

	return h.denyResponse(reqID, fmt.Sprintf("client permission request failed: %v", err))
}

// buildApprovalResponse converts the ACP permission response to a security approval response.
func (h *ApprovalHandler) buildApprovalResponse(
	reqID string, acpResp acp.RequestPermissionResponse, options []acp.PermissionOption,
) security.ApprovalResponse {
	approved, scope := resolvePermissionOutcome(acpResp, options)

	return security.ApprovalResponse{
		RequestID: reqID,
		Approved:  approved,
		Reason:    "client decision",
		Scope:     scope,
	}
}

// resolvePermissionOutcome determines the approval decision from the ACP response.
func resolvePermissionOutcome(acpResp acp.RequestPermissionResponse, options []acp.PermissionOption) (approved bool, reason string) {
	if acpResp.Outcome.Selected == nil {
		return false, ""
	}

	selectedID := acpResp.Outcome.Selected.OptionId
	for _, opt := range options {
		if opt.OptionId != selectedID {
			continue
		}

		switch opt.Kind {
		case acp.PermissionOptionKindAllowOnce:
			return true, security.ScopeOnce
		case acp.PermissionOptionKindAllowAlways:
			return true, security.ScopeGlobal
		case acp.PermissionOptionKindRejectOnce:
			return false, security.ScopeOnce
		case acp.PermissionOptionKindRejectAlways:
			return false, security.ScopeGlobal
		}
	}

	return false, ""
}

// convertApprovalRequestToToolCall converts a security approval request to an ACP tool call.
func (h *ApprovalHandler) convertApprovalRequestToToolCall(req security.ApprovalRequest) (acp.RequestPermissionToolCall, error) {
	// Extract tool name from command.
	toolName := unknownValue
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
