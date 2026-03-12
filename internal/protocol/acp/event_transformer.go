package acp

import (
	"context"
	"sync"

	"github.com/coder/acp-go-sdk"

	"github.com/dmytrogajewski/spin/internal/agent"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/planning"
)

// ACPEventTransformer transforms internal events to ACP protocol notifications.
// It implements the conversation.EventTransformer interface.
type ACPEventTransformer struct {
	sessionID   acp.SessionId
	connection  notificationSender
	agent       *agent.Agent
	fileTracker *fileContentTracker

	// Track accumulated content for plan detection.
	accumulatedContent string

	mu sync.RWMutex
}

// NewACPEventTransformer creates a new ACP event transformer.
func NewACPEventTransformer(sessionID acp.SessionId, conn notificationSender, ag *agent.Agent) *ACPEventTransformer {
	return &ACPEventTransformer{
		sessionID:   sessionID,
		connection:  conn,
		agent:       ag,
		fileTracker: newFileContentTracker(),
	}
}

// Transform processes an event and sends ACP notifications.
// Returns true if the event was handled (notification sent).
func (t *ACPEventTransformer) Transform(ctx context.Context, event events.Event) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.connection == nil {
		return false
	}

	// Reset content on turn start.
	if event.Type == events.EventTurnStart {
		t.accumulatedContent = ""

		return false // Don't consume, let others see it.
	}

	// Check for plan detection before tool call starts.
	if event.Type == events.EventToolCallStart {
		if t.accumulatedContent != "" && t.agent != nil && t.agent.GetPlanner() == nil {
			plan := planning.DetectPlanFromText(t.accumulatedContent)
			if plan != nil {
				t.agent.SetPlanner(plan)
				// Send plan notification immediately.
				planEntries := convertOrchestrationPlanToACP(plan)
				planUpdate := acp.UpdatePlan(planEntries...)
				notification := acp.SessionNotification{
					SessionId: t.sessionID,
					Update:    planUpdate,
				}
				_ = t.connection.SessionUpdate(ctx, notification)
			}
		}
	}

	// Handle content delta.
	if event.Type == events.EventContentDelta {
		data, ok := event.ContentDeltaData()
		if !ok || data.Role != "assistant" {
			return false
		}

		t.accumulatedContent += data.Content

		update := acp.UpdateAgentMessageText(data.Content)
		notification := acp.SessionNotification{
			SessionId: t.sessionID,
			Update:    update,
		}
		_ = t.connection.SessionUpdate(ctx, notification)

		return true
	}

	// Handle thinking delta.
	if event.Type == events.EventThinkingDelta {
		data, ok := event.ThinkingDeltaData()
		if !ok {
			return false
		}

		update := acp.UpdateAgentThoughtText(data.Content)
		notification := acp.SessionNotification{
			SessionId: t.sessionID,
			Update:    update,
		}
		_ = t.connection.SessionUpdate(ctx, notification)

		return true
	}

	// Handle plan updates.
	if event.Type == events.EventPlanUpdate {
		data, ok := event.PlanUpdateData()
		if !ok {
			return false
		}

		planEntries := convertOrchestrationPlanToACP(data.Plan)
		if len(planEntries) == 0 {
			return false
		}

		planUpdate := acp.UpdatePlan(planEntries...)
		notification := acp.SessionNotification{
			SessionId: t.sessionID,
			Update:    planUpdate,
		}
		_ = t.connection.SessionUpdate(ctx, notification)

		return true
	}

	// Handle system events (info, warning).
	if event.Type == events.EventInfo || event.Type == events.EventWarning {
		update, ok := convertSystemEvent(event)
		if !ok {
			return false
		}

		notification := acp.SessionNotification{
			SessionId: t.sessionID,
			Update:    update,
		}
		_ = t.connection.SessionUpdate(ctx, notification)

		return true
	}

	// Convert other events to ACP notification.
	update, ok := convertEventToSessionUpdate(event, t.fileTracker)
	if !ok {
		return false
	}

	// Extract terminal ID for release after notification.
	var terminalIDToRelease string

	if event.Type == events.EventToolCallComplete {
		if data, ok := event.ToolCallCompleteData(); ok {
			if terminalID, ok := data.Metadata["terminal_id"].(string); ok && terminalID != "" {
				terminalIDToRelease = terminalID
			}
		}
	}

	// Send notification.
	notification := acp.SessionNotification{
		SessionId: t.sessionID,
		Update:    update,
	}
	_ = t.connection.SessionUpdate(ctx, notification)

	// Release terminal AFTER notification is sent (per ACP spec).
	if terminalIDToRelease != "" {
		if acpConn, ok := t.connection.(*acp.AgentSideConnection); ok {
			terminalClient := NewACPTerminalClient(acpConn)
			_ = terminalClient.Release(ctx, terminalIDToRelease)
		}
	}

	return true
}

// Close releases resources held by the transformer.
func (t *ACPEventTransformer) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.connection = nil
	t.fileTracker = nil

	return nil
}

// SetConnection updates the connection (for reconnection scenarios).
func (t *ACPEventTransformer) SetConnection(conn notificationSender) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.connection = conn
}

// GetSessionID returns the session ID this transformer is associated with.
func (t *ACPEventTransformer) GetSessionID() acp.SessionId {
	return t.sessionID
}
