package acp

import (
	"context"
	"sync"
	"time"

	"github.com/coder/acp-go-sdk"

	"github.com/dmytrogajewski/spin/internal/events"
)

// EventTransformer transforms internal events to ACP protocol notifications.
// It implements the conversation.EventTransformer interface.
type EventTransformer struct {
	sessionID   acp.SessionId
	connection  notificationSender
	fileTracker *fileContentTracker

	// Track accumulated content for plan detection.
	accumulatedContent string
	// Track whether a plan has already been detected this turn.
	planDetected bool
	// Track whether a session title has been sent (once per session).
	titleSent bool

	mu sync.RWMutex
}

// NewEventTransformer creates a new ACP event transformer.
func NewEventTransformer(sessionID acp.SessionId, conn notificationSender) *EventTransformer {
	return &EventTransformer{
		sessionID:   sessionID,
		connection:  conn,
		fileTracker: newFileContentTracker(),
	}
}

// Transform processes an event and sends ACP notifications.
// Returns true if the event was handled (notification sent).
func (t *EventTransformer) Transform(ctx context.Context, event events.Event) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.connection == nil {
		return false
	}

	if event.Type == events.EventTurnStart {
		t.accumulatedContent = ""
		t.planDetected = false

		return false
	}

	if event.Type == events.EventToolCallStart {
		t.detectAndSendPlan(ctx)
	}

	switch event.Type {
	case events.EventContentDelta:
		return t.transformContentDelta(ctx, event)
	case events.EventThinkingDelta:
		return t.transformThinkingDelta(ctx, event)
	case events.EventPlanUpdate:
		return t.transformPlanUpdate(ctx, event)
	case events.EventInfo, events.EventWarning:
		return t.transformSystemEvent(ctx, event)
	case events.EventTurnComplete:
		t.sendSessionInfoIfNeeded(ctx)

		return false
	default:
		return t.transformGenericEvent(ctx, event)
	}
}

// detectAndSendPlan checks accumulated content for a plan and sends it.
func (t *EventTransformer) detectAndSendPlan(ctx context.Context) {
	if t.accumulatedContent == "" || t.planDetected {
		return
	}

	plan := DetectPlanFromText(t.accumulatedContent)
	if plan == nil {
		return
	}

	t.planDetected = true
	planEntries := convertOrchestrationPlanToACP(plan)
	t.sendUpdate(ctx, acp.UpdatePlan(planEntries...))
}

// transformContentDelta handles content delta events.
func (t *EventTransformer) transformContentDelta(ctx context.Context, event events.Event) bool {
	data, ok := event.ContentDeltaData()
	if !ok || data.Role != roleAssistant {
		return false
	}

	t.accumulatedContent += data.Content
	t.sendUpdate(ctx, acp.UpdateAgentMessageText(data.Content))

	return true
}

// transformThinkingDelta handles thinking delta events.
func (t *EventTransformer) transformThinkingDelta(ctx context.Context, event events.Event) bool {
	data, ok := event.ThinkingDeltaData()
	if !ok {
		return false
	}

	t.sendUpdate(ctx, acp.UpdateAgentThoughtText(data.Content))

	return true
}

// transformPlanUpdate handles plan update events.
func (t *EventTransformer) transformPlanUpdate(ctx context.Context, event events.Event) bool {
	data, ok := event.PlanUpdateData()
	if !ok {
		return false
	}

	plan, isPlan := data.Plan.(*Plan)
	if !isPlan || plan == nil {
		return false
	}

	planEntries := convertOrchestrationPlanToACP(plan)
	if len(planEntries) == 0 {
		return false
	}

	t.sendUpdate(ctx, acp.UpdatePlan(planEntries...))

	return true
}

// transformSystemEvent handles system info/warning events.
func (t *EventTransformer) transformSystemEvent(ctx context.Context, event events.Event) bool {
	update, ok := convertSystemEvent(event)
	if !ok {
		return false
	}

	t.sendUpdate(ctx, update)

	return true
}

// transformGenericEvent handles all other event types.
func (t *EventTransformer) transformGenericEvent(ctx context.Context, event events.Event) bool {
	update, ok := convertEventToSessionUpdate(event, t.fileTracker)
	if !ok {
		return false
	}

	terminalID := extractTerminalID(event)

	t.sendUpdate(ctx, update)

	if terminalID != "" {
		if acpConn, isACPConn := t.connection.(*acp.AgentSideConnection); isACPConn {
			terminalClient := NewTerminalClient(acpConn)
			_ = terminalClient.Release(ctx, terminalID)
		}
	}

	return true
}

// sendSessionInfoIfNeeded sends a session_info_update if a title hasn't been sent yet.
func (t *EventTransformer) sendSessionInfoIfNeeded(ctx context.Context) {
	if t.titleSent || t.accumulatedContent == "" {
		return
	}

	title := generateSessionTitle(t.accumulatedContent)
	if title == "" {
		return
	}

	t.titleSent = true

	ts := time.Now().UTC().Format(time.RFC3339)

	t.sendUpdate(ctx, acp.SessionUpdate{
		SessionInfoUpdate: &acp.SessionSessionInfoUpdate{
			Title:     &title,
			UpdatedAt: &ts,
		},
	})
}

// sendUpdate sends a session update notification.
func (t *EventTransformer) sendUpdate(ctx context.Context, update acp.SessionUpdate) {
	notification := acp.SessionNotification{
		SessionId: t.sessionID,
		Update:    update,
	}
	_ = t.connection.SessionUpdate(ctx, notification)
}

// Close releases resources held by the transformer.
func (t *EventTransformer) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.connection = nil
	t.fileTracker = nil

	return nil
}

// SetConnection updates the connection (for reconnection scenarios).
func (t *EventTransformer) SetConnection(conn notificationSender) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.connection = conn
}

// GetSessionID returns the session ID this transformer is associated with.
func (t *EventTransformer) GetSessionID() acp.SessionId {
	return t.sessionID
}
