package acp

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/coder/acp-go-sdk"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/mcp"
	"github.com/dmytrogajewski/spin/internal/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPrompt_NoUserMessageChunk tests that Prompt does NOT send user_message_chunk notification.
// The client already knows what they sent in the session/prompt request, so we shouldn't echo it back.
// user_message_chunk is only sent when replaying history in LoadSession.
func TestPrompt_NoUserMessageChunk(t *testing.T) {
	agentInstance := createTestAgent(t)

	mcpManager := mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default()))
	emitter := events.NewEventEmitter(100)
	storage, err := session.NewFileStorage(t.TempDir())
	require.NoError(t, err)

	acpAgent, err := NewSpinACPAgentWithStorage(agentInstance, mcpManager, emitter, storage)
	require.NoError(t, err)

	// Create a session
	sess := session.NewSession("/tmp/test")
	err = storage.Save(sess.ID, *sess)
	require.NoError(t, err)

	sessionID := acp.SessionId(sess.ID)
	acpAgent.mu.Lock()
	acpAgent.sessions[sessionID] = sess
	acpAgent.mu.Unlock()

	// Create mock connection
	mockConn := &mockNotificationSender{
		mu:            sync.Mutex{},
		notifications: []acp.SessionNotification{},
	}
	acpAgent.SetNotificationSender(mockConn)

	// Call Prompt
	req := acp.PromptRequest{
		SessionId: sessionID,
		Prompt: []acp.ContentBlock{
			acp.TextBlock("Hello, world!"),
		},
	}

	ctx := context.Background()
	resp, err := acpAgent.Prompt(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Give event processing time to send notifications
	time.Sleep(100 * time.Millisecond)

	// Get all notifications
	notifications := mockConn.GetNotifications()

	// Verify NO user_message_chunk was sent (client already knows what they sent)
	// This is the key fix: we should NOT echo back the user's prompt
	for _, notif := range notifications {
		assert.Nil(t, notif.Update.UserMessageChunk, "should not send user_message_chunk in response to Prompt request - client already knows what they sent")
	}
}

// mockNotificationSender is a mock notification sender for testing.
type mockNotificationSender struct {
	mu            sync.Mutex
	notifications []acp.SessionNotification
}

func (m *mockNotificationSender) SessionUpdate(ctx context.Context, notification acp.SessionNotification) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.notifications = append(m.notifications, notification)
	return nil
}

func (m *mockNotificationSender) RequestPermission(ctx context.Context, params acp.RequestPermissionRequest) (acp.RequestPermissionResponse, error) {
	// Auto-approve for testing by selecting the first allow option
	for _, opt := range params.Options {
		if opt.Kind == acp.PermissionOptionKindAllowOnce || opt.Kind == acp.PermissionOptionKindAllowAlways {
			return acp.RequestPermissionResponse{
				Outcome: acp.NewRequestPermissionOutcomeSelected(opt.OptionId),
			}, nil
		}
	}
	// No allow option found, return cancelled
	return acp.RequestPermissionResponse{
		Outcome: acp.NewRequestPermissionOutcomeCancelled(),
	}, nil
}

func (m *mockNotificationSender) GetNotifications() []acp.SessionNotification {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]acp.SessionNotification, len(m.notifications))
	copy(result, m.notifications)
	return result
}

func (m *mockNotificationSender) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.notifications = nil
}
