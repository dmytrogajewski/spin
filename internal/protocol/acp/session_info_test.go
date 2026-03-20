package acp

// Journey: specs/journeys/JOURNEY-R3.1-acp-session-info-update.md.

import (
	"context"
	"testing"

	"github.com/coder/acp-go-sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/events"
)

// TestEventTransformer_SessionInfoOnTurnComplete tests that turn complete triggers session info update.
func TestEventTransformer_SessionInfoOnTurnComplete(t *testing.T) {
	t.Parallel()

	mockConn := &mockConnection{}
	transformer := NewEventTransformer(acp.SessionId("test-session"), mockConn)

	ctx := context.Background()

	// Simulate content accumulation.
	contentEvent := newContentEvent("assistant", "Fix the authentication bug.")
	transformer.Transform(ctx, contentEvent)

	// Simulate turn complete.
	turnCompleteEvent := events.Event{Type: events.EventTurnComplete}
	transformer.Transform(ctx, turnCompleteEvent)

	// Check for session_info_update notification.
	notifications := mockConn.GetNotifications()
	found := false

	for _, notif := range notifications {
		if notif.Update.SessionInfoUpdate != nil {
			found = true

			require.NotNil(t, notif.Update.SessionInfoUpdate.Title)
			assert.Equal(t, "Fix the authentication bug.", *notif.Update.SessionInfoUpdate.Title)
			assert.NotNil(t, notif.Update.SessionInfoUpdate.UpdatedAt)
		}
	}

	assert.True(t, found, "should send session_info_update on first turn complete")
}

// TestEventTransformer_NoSessionInfoOnEmptyContent tests no notification when content is empty.
func TestEventTransformer_NoSessionInfoOnEmptyContent(t *testing.T) {
	t.Parallel()

	mockConn := &mockConnection{}
	transformer := NewEventTransformer(acp.SessionId("test-session"), mockConn)

	// Turn complete without any content.
	turnCompleteEvent := events.Event{Type: events.EventTurnComplete}
	transformer.Transform(context.Background(), turnCompleteEvent)

	notifications := mockConn.GetNotifications()
	for _, notif := range notifications {
		assert.Nil(t, notif.Update.SessionInfoUpdate, "should not send session_info_update without content")
	}
}

// TestEventTransformer_SessionInfoSentOnce tests that session info is sent only once.
func TestEventTransformer_SessionInfoSentOnce(t *testing.T) {
	t.Parallel()

	mockConn := &mockConnection{}
	transformer := NewEventTransformer(acp.SessionId("test-session"), mockConn)

	ctx := context.Background()

	// First turn.
	contentEvent := newContentEvent("assistant", "First response.")
	transformer.Transform(ctx, contentEvent)

	turnCompleteEvent := events.Event{Type: events.EventTurnComplete}
	transformer.Transform(ctx, turnCompleteEvent)

	// Reset for second turn.
	transformer.Transform(ctx, events.Event{Type: events.EventTurnStart})

	// Second turn.
	contentEvent2 := newContentEvent("assistant", "Second response.")
	transformer.Transform(ctx, contentEvent2)
	transformer.Transform(ctx, turnCompleteEvent)

	// Count session_info_update notifications.
	notifications := mockConn.GetNotifications()
	infoCount := 0

	for _, notif := range notifications {
		if notif.Update.SessionInfoUpdate != nil {
			infoCount++
		}
	}

	assert.Equal(t, 1, infoCount, "should send session_info_update exactly once")
}

// newContentEvent creates a content delta event for testing.
func newContentEvent(role, content string) events.Event {
	return events.Event{
		Type: events.EventContentDelta,
		Data: events.ContentDeltaData{Role: role, Content: content},
	}
}
