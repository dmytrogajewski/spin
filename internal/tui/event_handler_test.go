package tui

import (
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/core"
	"github.com/dmytrogajewski/spin/internal/tui/ui"
	"github.com/stretchr/testify/assert"
)

// NewTestModel creates a test model with minimal setup
func NewTestModel() Model {
	events := make(chan core.Event) // Will be closed in tests
	return Model{
		state:     StateIdle,
		chat:      ui.NewChat(80, 20),
		approval:  ui.NewApproval(),
		statusBar: ui.NewStatusBar(80),
		input:     ui.NewInput(80, 3),
		width:     80,
		height:    24,
		events:    events,
	}
}

func TestHandleStreamDelta(t *testing.T) {
	m := NewTestModel()

	event := core.Event{
		Type:      core.EventContentDelta,
		Timestamp: time.Now(),
		Data: &core.ContentDeltaData{
			Content: "Hello",
			Role:    "assistant",
		},
	}

	m = m.handleStreamDelta(event)

	// Chat should have received the delta
	currentMsg := m.chat.CurrentMessage()
	assert.Contains(t, currentMsg, "Hello")
}

func TestHandleStreamDelta_MultipleDeltas(t *testing.T) {
	m := NewTestModel()

	deltas := []string{"Hello", " ", "world", "!"}
	for _, delta := range deltas {
		event := core.Event{
			Type:      core.EventContentDelta,
			Timestamp: time.Now(),
			Data: &core.ContentDeltaData{
				Content: delta,
				Role:    "assistant",
			},
		}
		m = m.handleStreamDelta(event)
	}

	currentMsg := m.chat.CurrentMessage()
	assert.Equal(t, "Hello world!", currentMsg)
}

func TestHandleApprovalRequest(t *testing.T) {
	m := NewTestModel()

	req := &core.ApprovalRequest{
		ID: "test-123",
		Command: &core.Command{
			Program: "rm",
			Raw:     "rm -rf node_modules",
			Args:    []string{"rm", "-rf", "node_modules"},
		},
		Reason:    "dangerous command",
		WorkDir:   "/test",
		Timestamp: time.Now(),
	}

	event := core.Event{
		Type:      core.EventCommandApproval,
		Timestamp: time.Now(),
		Data:      req,
	}

	m = m.handleApprovalRequest(event)

	assert.Equal(t, StateToolApproval, m.state)
	// Use Request() accessor
	request := m.approval.Request()
	assert.NotNil(t, request)
	assert.Equal(t, "test-123", request.ID)
	assert.Equal(t, "rm -rf node_modules", request.Command.Raw)
}

func TestHandleApprovalResult_Approved(t *testing.T) {
	m := NewTestModel()
	m.state = StateToolApproval

	event := core.Event{
		Type:      core.EventCommandApproved,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"command":  "rm -rf node_modules",
			"approved": true,
		},
	}

	m = m.handleApprovalResult(event, true)

	assert.Equal(t, StateWaitingResponse, m.state)
}

func TestHandleApprovalResult_Denied(t *testing.T) {
	m := NewTestModel()
	m.state = StateToolApproval

	event := core.Event{
		Type:      core.EventCommandDenied,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"command":  "rm -rf node_modules",
			"approved": false,
			"reason":   "user denied",
		},
	}

	m = m.handleApprovalResult(event, false)

	assert.Equal(t, StateWaitingResponse, m.state)
}

func TestHandleTurnStart(t *testing.T) {
	m := NewTestModel()
	m.state = StateIdle

	event := core.Event{
		Type:      core.EventTurnStart,
		Timestamp: time.Now(),
		Data:      map[string]interface{}{"turn_id": "turn-123"},
	}

	m = m.handleTurnStart(event)

	assert.Equal(t, StateWaitingResponse, m.state)
}

func TestHandleTurnComplete(t *testing.T) {
	m := NewTestModel()
	m.state = StateWaitingResponse

	event := core.Event{
		Type:      core.EventTurnComplete,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"turn_id":     "turn-123",
			"tokens_used": 150,
		},
	}

	m = m.handleTurnComplete(event)

	assert.Equal(t, StateIdle, m.state)
	// Status bar should show tokens
	assert.Equal(t, 150, m.statusBar.TurnTokens())
}

func TestHandleTurnPaused(t *testing.T) {
	m := NewTestModel()
	m.state = StateWaitingResponse

	event := core.Event{
		Type:      core.EventTurnPaused,
		Timestamp: time.Now(),
		Data:      nil,
	}

	m = m.handleTurnPaused(event)

	// Should go to idle (can interact while paused)
	assert.Equal(t, StateIdle, m.state)
}

func TestHandleTurnResumed(t *testing.T) {
	m := NewTestModel()
	m.state = StateIdle

	event := core.Event{
		Type:      core.EventTurnResumed,
		Timestamp: time.Now(),
		Data:      nil,
	}

	m = m.handleTurnResumed(event)

	assert.Equal(t, StateWaitingResponse, m.state)
}

func TestHandleError(t *testing.T) {
	m := NewTestModel()

	event := core.Event{
		Type:      core.EventError,
		Timestamp: time.Now(),
		Data: &core.ErrorData{
			Message: "something went wrong",
			Code:    "TEST_ERROR",
		},
	}

	m = m.handleError(event)

	// Should transition to idle
	assert.Equal(t, StateIdle, m.state)

	// Chat should have error message (as system message with IsError=true)
	messages := m.chat.AllMessages()
	found := false
	for _, msg := range messages {
		if msg.IsError {
			found = true
			assert.Equal(t, ui.RoleSystem, msg.Role, "error message should have system role")
			break
		}
	}
	assert.True(t, found, "error message not found in chat")
}

func TestHandleCoreEvent_UnknownEventType(t *testing.T) {
	m := NewTestModel()
	originalState := m.state

	event := core.Event{
		Type:      core.EventType(999), // Unknown type
		Timestamp: time.Now(),
		Data:      nil,
	}

	msg := CoreEventMsg{Event: event}
	m, _ = m.handleCoreEvent(msg) // Keep the _, as this returns (Model, tea.Cmd)

	// Should not change state
	assert.Equal(t, originalState, m.state)
}

func TestWaitForCoreEvent(t *testing.T) {
	events := make(chan core.Event, 1)

	// Send test event
	testEvent := core.Event{
		Type:      core.EventContentDelta,
		Timestamp: time.Now(),
		Data: &core.ContentDeltaData{
			Content: "test",
		},
	}
	events <- testEvent

	// Execute command
	cmd := waitForCoreEvent(events)
	assert.NotNil(t, cmd)

	// Should return CoreEventMsg
	msg := cmd()
	coreMsg, ok := msg.(CoreEventMsg)
	assert.True(t, ok)
	assert.Equal(t, core.EventContentDelta, coreMsg.Event.Type)
}

func TestWaitForCoreEvent_ChannelClosed(t *testing.T) {
	events := make(chan core.Event)
	close(events)

	cmd := waitForCoreEvent(events)
	assert.NotNil(t, cmd)

	// Should return nil when channel closed
	msg := cmd()
	assert.Nil(t, msg)
}
