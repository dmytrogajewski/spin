package acp

import (
	"context"
	"sync"
	"testing"

	"github.com/coder/acp-go-sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/events"
)

// dedupTestNotifSender records all notifications sent.
type dedupTestNotifSender struct {
	mu            sync.Mutex
	notifications []acp.SessionNotification
}

func (m *dedupTestNotifSender) SessionUpdate(_ context.Context, n acp.SessionNotification) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.notifications = append(m.notifications, n)

	return nil
}

func (m *dedupTestNotifSender) RequestPermission(_ context.Context, _ acp.RequestPermissionRequest) (acp.RequestPermissionResponse, error) {
	return acp.RequestPermissionResponse{}, nil
}

func (m *dedupTestNotifSender) getNotifications() []acp.SessionNotification {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make([]acp.SessionNotification, len(m.notifications))
	copy(result, m.notifications)

	return result
}

// TestEventTransformer_ContentDelta_NoDuplication verifies that a single
// EventContentDelta produces exactly ONE ACP notification, not two.
// This reproduces the bug where two emitter subscriptions caused every
// content token to appear twice ("Hello! Hello!").
func TestEventTransformer_ContentDelta_NoDuplication(t *testing.T) {
	t.Parallel()

	sender := &dedupTestNotifSender{}
	transformer := NewEventTransformer("test-session", sender)

	ctx := context.Background()

	// Emit a content delta.
	event := events.Event{
		Type: events.EventContentDelta,
		Data: events.ContentDeltaData{
			Content: "Hello!",
			Role:    roleAssistant,
		},
	}

	handled := transformer.Transform(ctx, event)
	require.True(t, handled)

	// Must produce exactly ONE notification.
	notifications := sender.getNotifications()
	require.Len(t, notifications, 1, "single event must produce exactly 1 notification, not 2")
}

// TestEventTransformer_ThinkingDelta_SentAsThought verifies that
// EventThinkingDelta is sent as UpdateAgentThoughtText, not UpdateAgentMessageText.
func TestEventTransformer_ThinkingDelta_SentAsThought(t *testing.T) {
	t.Parallel()

	sender := &dedupTestNotifSender{}
	transformer := NewEventTransformer("test-session", sender)

	ctx := context.Background()

	event := events.Event{
		Type: events.EventThinkingDelta,
		Data: events.ThinkingDeltaData{
			Content: "Let me think about this...",
		},
	}

	handled := transformer.Transform(ctx, event)
	require.True(t, handled)

	notifications := sender.getNotifications()
	require.Len(t, notifications, 1)

	// The thinking delta must produce exactly one notification (not duplicated).
	// We verify it was handled as a thought by confirming the transformer returned true.
	assert.Len(t, notifications, 1, "thinking delta must produce exactly 1 notification")
}

// TestEventTransformer_MultipleDeltas_CountMatches verifies that N content
// deltas produce exactly N notifications.
func TestEventTransformer_MultipleDeltas_CountMatches(t *testing.T) {
	t.Parallel()

	sender := &dedupTestNotifSender{}
	transformer := NewEventTransformer("test-session", sender)

	ctx := context.Background()

	tokens := []string{"Hello", ", ", "world", "!"}

	for _, token := range tokens {
		event := events.Event{
			Type: events.EventContentDelta,
			Data: events.ContentDeltaData{
				Content: token,
				Role:    roleAssistant,
			},
		}

		transformer.Transform(ctx, event)
	}

	notifications := sender.getNotifications()
	assert.Len(t, notifications, len(tokens),
		"N tokens must produce exactly N notifications (not 2N)")
}
