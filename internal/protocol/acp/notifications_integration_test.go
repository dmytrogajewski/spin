package acp

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/coder/acp-go-sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/session"
	"github.com/dmytrogajewski/spin/internal/tools"
)

// mockConnection is a mock notificationSender for testing.
type mockConnection struct {
	mu            sync.Mutex
	notifications []acp.SessionNotification
}

func (m *mockConnection) SessionUpdate(_ context.Context, notification acp.SessionNotification) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.notifications = append(m.notifications, notification)

	return nil
}

func (m *mockConnection) RequestPermission(_ context.Context, params acp.RequestPermissionRequest) (acp.RequestPermissionResponse, error) {
	for _, opt := range params.Options {
		if opt.Kind == acp.PermissionOptionKindAllowOnce || opt.Kind == acp.PermissionOptionKindAllowAlways {
			return acp.RequestPermissionResponse{
				Outcome: newOutcomeSelected(opt.OptionId),
			}, nil
		}
	}

	return acp.RequestPermissionResponse{
		Outcome: acp.NewRequestPermissionOutcomeCancelled(),
	}, nil
}

func (m *mockConnection) GetNotifications() []acp.SessionNotification {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make([]acp.SessionNotification, len(m.notifications))
	copy(result, m.notifications)

	return result
}

func (m *mockConnection) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.notifications = nil
}

// TestTransformer_ContentDelta tests that content delta events produce notifications.
func TestTransformer_ContentDelta(t *testing.T) {
	t.Parallel()

	mockConn := &mockConnection{}
	sessionID := acp.SessionId("test-session")
	transformer := NewEventTransformer(sessionID, mockConn, "")

	transformer.Transform(context.Background(), events.Event{
		Type:      events.EventContentDelta,
		Timestamp: time.Now(),
		Data:      events.ContentDeltaData{Content: "Hello", Role: "assistant"},
	})

	notifications := mockConn.GetNotifications()
	require.Len(t, notifications, 1, "should have one notification")
	assert.Equal(t, sessionID, notifications[0].SessionId)
}

// TestTransformer_ToolCallStart tests that tool call start events produce notifications.
func TestTransformer_ToolCallStart(t *testing.T) {
	t.Parallel()

	mockConn := &mockConnection{}
	sessionID := acp.SessionId("test-session")
	transformer := NewEventTransformer(sessionID, mockConn, "")

	transformer.Transform(context.Background(), events.Event{
		Type:      events.EventToolCallStart,
		Timestamp: time.Now(),
		Data:      events.ToolCallStartData{ToolID: "tool-123", ToolName: "read_file"},
	})

	notifications := mockConn.GetNotifications()
	require.Len(t, notifications, 1, "should have one notification")
	assert.Equal(t, sessionID, notifications[0].SessionId)
}

// TestTransformer_NilConnection tests that events are not sent when connection is nil.
func TestTransformer_NilConnection(t *testing.T) {
	t.Parallel()

	transformer := NewEventTransformer("test-session", nil, "")

	handled := transformer.Transform(context.Background(), events.Event{
		Type:      events.EventContentDelta,
		Timestamp: time.Now(),
		Data:      events.ContentDeltaData{Content: "Hello", Role: "assistant"},
	})

	assert.False(t, handled, "should not handle events without connection")
}

// TestTransformer_WriteFile_GeneratesDiff tests that write_file operations generate diff notifications.
func TestTransformer_WriteFile_GeneratesDiff(t *testing.T) {
	t.Parallel()

	mockConn := &mockConnection{}
	sessionID := acp.SessionId("test-session")
	transformer := NewEventTransformer(sessionID, mockConn, "")

	ctx := context.Background()

	// Create a temporary file with existing content.
	tmpFile := t.TempDir() + "/test.txt"

	err := os.WriteFile(tmpFile, []byte("old content\nline 2"), 0o600)
	require.NoError(t, err)

	// Emit tool call start event for write_file.
	params, paramsErr := tools.FromMap(map[string]any{
		"path":    tmpFile,
		"content": "new content\nline 2\nline 3",
	})
	require.NoError(t, paramsErr)

	transformer.Transform(ctx, events.Event{
		Type:      events.EventToolCallStart,
		Timestamp: time.Now(),
		Data: events.ToolCallStartData{
			ToolID:     "tool-write",
			ToolName:   "write_file",
			Parameters: params,
		},
	})

	// Emit tool call complete event.
	transformer.Transform(ctx, events.Event{
		Type:      events.EventToolCallComplete,
		Timestamp: time.Now(),
		Data: events.ToolCallCompleteData{
			ToolID:   "tool-write",
			ToolName: "write_file",
			Success:  true,
			Output:   "Successfully wrote file",
		},
	})

	// Verify notifications were sent.
	notifications := mockConn.GetNotifications()
	require.GreaterOrEqual(t, len(notifications), 2, "should have at least start and complete notifications")

	// Find the complete notification with tool call update.
	var completeNotification *acp.SessionNotification

	for i := range notifications {
		if notifications[i].Update.ToolCallUpdate != nil {
			completeNotification = &notifications[i]

			break
		}
	}

	require.NotNil(t, completeNotification, "should have tool call update notification")
	assert.Equal(t, sessionID, completeNotification.SessionId)

	// Verify diff content is included.
	update := completeNotification.Update
	require.NotNil(t, update.ToolCallUpdate, "should have tool call update")
	require.NotNil(t, update.ToolCallUpdate.Content, "should have content")
	require.NotEmpty(t, update.ToolCallUpdate.Content, "should have at least one content item")

	hasDiff := false

	for _, content := range update.ToolCallUpdate.Content {
		if content.Diff != nil {
			hasDiff = true

			assert.Equal(t, tmpFile, content.Diff.Path, "diff should have correct file path")

			break
		}
	}

	assert.True(t, hasDiff, "should include diff content in notification")
}

// TestSubscribeTransformerEvents_NoHangOnSlowRelease tests that the event goroutine
// cleanup doesn't hang when the context is canceled promptly.
// This reproduces a bug where cancel() was called AFTER <-eventsDone, causing
// the goroutine to hang if it was blocked on a synchronous RPC.
func TestSubscribeTransformerEvents_NoHangOnSlowRelease(t *testing.T) {
	t.Parallel()

	emitter := events.NewEventEmitter(10)
	defer emitter.Close()

	mockConn := &mockConnection{}
	sessionID := acp.SessionId("test-session")
	transformer := NewEventTransformer(sessionID, mockConn, "")

	ctx, cancel := context.WithCancel(context.Background())

	acpAgent := createTestACPAgent(t, emitter)
	acpAgent.SetNotificationSender(mockConn)

	unsubscribe, eventsDone := acpAgent.subscribeTransformerEvents(ctx, mockConn, transformer)
	require.NotNil(t, unsubscribe)
	require.NotNil(t, eventsDone)

	// Emit a content event to prove the goroutine is running.
	emitter.Emit(events.Event{
		Type:      events.EventContentDelta,
		Timestamp: time.Now(),
		Data:      events.ContentDeltaData{Content: "Hello", Role: "assistant"},
	})

	// Give the goroutine time to process.
	time.Sleep(50 * time.Millisecond)

	// Now test that cleanup doesn't hang:
	// cancel() first (so goroutine can exit via ctx.Done()), then unsubscribe, then wait.
	done := make(chan struct{})

	go func() {
		cancel()
		unsubscribe()
		<-eventsDone
		close(done)
	}()

	select {
	case <-done:
		// Success - cleanup completed without hanging.
	case <-time.After(2 * time.Second):
		t.Fatal("cleanup hung - event goroutine did not exit within 2 seconds")
	}
}

// createTestACPAgent creates a minimal SpinACPAgent for testing.
func createTestACPAgent(t *testing.T, emitter *events.EventEmitter) *SpinACPAgent {
	t.Helper()

	return &SpinACPAgent{
		emitter:      emitter,
		sessions:     make(map[acp.SessionId]*session.Session),
		sessionModes: make(map[acp.SessionId]acp.SessionModeId),
		cancels:      make(map[acp.SessionId]context.CancelFunc),
		transformers: make(map[acp.SessionId]*EventTransformer),
	}
}
