package acp

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/coder/acp-go-sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/agent"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/mcp"
	"github.com/dmytrogajewski/spin/internal/session"
	"github.com/dmytrogajewski/spin/internal/tools"
)

// mockConnection is a mock AgentSideConnection for testing.
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
	// Auto-approve for testing by selecting the first allow option.
	for _, opt := range params.Options {
		if opt.Kind == acp.PermissionOptionKindAllowOnce || opt.Kind == acp.PermissionOptionKindAllowAlways {
			return acp.RequestPermissionResponse{
				Outcome: acp.NewRequestPermissionOutcomeSelected(opt.OptionId),
			}, nil
		}
	}
	// No allow option found, return canceled.
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

// TestProcessEvents_ContentDelta tests event processing for content delta.
func TestProcessEvents_ContentDelta(t *testing.T) {
	t.Parallel()
	agentInstance := &agent.Agent{}
	mcpManager := mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default()))
	emitter := events.NewEventEmitter(100)

	storage, err := session.NewFileStorage(t.TempDir())
	require.NoError(t, err)
	acpAgent, err := NewSpinACPAgentWithStorage(agentInstance, mcpManager, emitter, storage)
	require.NoError(t, err)

	mockConn := &mockConnection{}
	acpAgent.SetNotificationSender(mockConn)

	ctx := t.Context()

	sessionID := acp.SessionId("test-session")

	// Subscribe to events.
	subID, eventCh, err := emitter.Subscribe()
	require.NoError(t, err)

	defer emitter.Unsubscribe(subID)

	// Start event processing.
	go acpAgent.processEvents(ctx, sessionID, eventCh)

	// Emit a content delta event.
	emitter.Emit(events.Event{
		Type:      events.EventContentDelta,
		Timestamp: time.Now(),
		Data: events.ContentDeltaData{
			Content: "Hello",
			Role:    "assistant",
		},
	})

	// Give it time to process.
	time.Sleep(50 * time.Millisecond)

	// Verify notification was sent.
	notifications := mockConn.GetNotifications()
	require.Len(t, notifications, 1, "should have one notification")
	assert.Equal(t, sessionID, notifications[0].SessionId)
}

// TestProcessEvents_ToolCallStart tests event processing for tool call start.
func TestProcessEvents_ToolCallStart(t *testing.T) {
	t.Parallel()
	agentInstance := &agent.Agent{}
	mcpManager := mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default()))
	emitter := events.NewEventEmitter(100)

	storage, err := session.NewFileStorage(t.TempDir())
	require.NoError(t, err)
	acpAgent, err := NewSpinACPAgentWithStorage(agentInstance, mcpManager, emitter, storage)
	require.NoError(t, err)

	mockConn := &mockConnection{}
	acpAgent.SetNotificationSender(mockConn)

	ctx := t.Context()

	sessionID := acp.SessionId("test-session")

	// Subscribe to events.
	subID, eventCh, err := emitter.Subscribe()
	require.NoError(t, err)

	defer emitter.Unsubscribe(subID)

	// Start event processing.
	go acpAgent.processEvents(ctx, sessionID, eventCh)

	// Emit a tool call start event.
	emitter.Emit(events.Event{
		Type:      events.EventToolCallStart,
		Timestamp: time.Now(),
		Data: events.ToolCallStartData{
			ToolID:   "tool-123",
			ToolName: "read_file",
		},
	})

	// Give it time to process.
	time.Sleep(50 * time.Millisecond)

	// Verify notification was sent.
	notifications := mockConn.GetNotifications()
	require.Len(t, notifications, 1, "should have one notification")
	assert.Equal(t, sessionID, notifications[0].SessionId)
}

// TestProcessEvents_NoConnection tests that events are not sent when connection is nil.
func TestProcessEvents_NoConnection(t *testing.T) {
	t.Parallel()
	agentInstance := &agent.Agent{}
	mcpManager := mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default()))
	emitter := events.NewEventEmitter(100)

	storage, err := session.NewFileStorage(t.TempDir())
	require.NoError(t, err)
	acpAgent, err := NewSpinACPAgentWithStorage(agentInstance, mcpManager, emitter, storage)
	require.NoError(t, err)

	// Don't set connection.

	ctx := t.Context()

	sessionID := acp.SessionId("test-session")

	// Subscribe to events.
	subID, eventCh, err := emitter.Subscribe()
	require.NoError(t, err)

	defer emitter.Unsubscribe(subID)

	// Start event processing.
	go acpAgent.processEvents(ctx, sessionID, eventCh)

	// Emit an event.
	emitter.Emit(events.Event{
		Type:      events.EventContentDelta,
		Timestamp: time.Now(),
		Data: events.ContentDeltaData{
			Content: "Hello",
			Role:    "assistant",
		},
	})

	// Give it time to process.
	time.Sleep(50 * time.Millisecond)

	// No connection, so no notifications should be sent (no panic)
	// This test just verifies graceful handling.
}

// TestProcessEvents_ContextCancellation tests that event processing stops on context cancellation.
func TestProcessEvents_ContextCancellation(t *testing.T) {
	t.Parallel()
	agentInstance := &agent.Agent{}
	mcpManager := mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default()))
	emitter := events.NewEventEmitter(100)

	storage, err := session.NewFileStorage(t.TempDir())
	require.NoError(t, err)
	acpAgent, err := NewSpinACPAgentWithStorage(agentInstance, mcpManager, emitter, storage)
	require.NoError(t, err)

	mockConn := &mockConnection{}
	acpAgent.SetNotificationSender(mockConn)

	ctx, cancel := context.WithCancel(context.Background())

	sessionID := acp.SessionId("test-session")

	// Subscribe to events.
	subID, eventCh, err := emitter.Subscribe()
	require.NoError(t, err)

	defer emitter.Unsubscribe(subID)

	// Start event processing.
	done := make(chan struct{})

	go func() {
		acpAgent.processEvents(ctx, sessionID, eventCh)
		close(done)
	}()

	// Cancel context.
	cancel()

	// Wait for goroutine to finish.
	select {
	case <-done:
		// Success.
	case <-time.After(1 * time.Second):
		t.Fatal("processEvents did not stop on context cancellation")
	}
}

// TestProcessEvents_WriteFile_GeneratesDiff tests that write_file operations generate diff notifications.
func setupWriteFileTest(t *testing.T) (*SpinACPAgent, *mockConnection, *events.EventEmitter, acp.SessionId) {
	t.Helper()

	agentInstance := &agent.Agent{}
	mcpManager := mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default()))
	emitter := events.NewEventEmitter(100)

	storage, err := session.NewFileStorage(t.TempDir())
	require.NoError(t, err)
	acpAgent, err := NewSpinACPAgentWithStorage(agentInstance, mcpManager, emitter, storage)
	require.NoError(t, err)

	mockConn := &mockConnection{}
	acpAgent.SetNotificationSender(mockConn)

	return acpAgent, mockConn, emitter, acp.SessionId("test-session")
}

func TestProcessEvents_WriteFile_GeneratesDiff(t *testing.T) {
	t.Parallel()

	acpAgent, mockConn, emitter, sessionID := setupWriteFileTest(t)

	ctx := t.Context()
	subID, eventCh, err := emitter.Subscribe()
	require.NoError(t, err)
	defer emitter.Unsubscribe(subID)

	go acpAgent.processEvents(ctx, sessionID, eventCh)

	// Create a temporary file with existing content.
	tmpFile := t.TempDir() + "/test.txt"
	err = os.WriteFile(tmpFile, []byte("old content\nline 2"), 0644)
	require.NoError(t, err)

	// Emit tool call start event for write_file.
	params, err := tools.FromMap(map[string]any{
		"path":    tmpFile,
		"content": "new content\nline 2\nline 3",
	})
	require.NoError(t, err)

	emitter.Emit(events.Event{
		Type:      events.EventToolCallStart,
		Timestamp: time.Now(),
		Data: events.ToolCallStartData{
			ToolID:     "tool-write",
			ToolName:   "write_file",
			Parameters: params,
		},
	})

	// Give it time to process.
	time.Sleep(50 * time.Millisecond)

	// Emit tool call complete event.
	emitter.Emit(events.Event{
		Type:      events.EventToolCallComplete,
		Timestamp: time.Now(),
		Data: events.ToolCallCompleteData{
			ToolID:   "tool-write",
			ToolName: "write_file",
			Success:  true,
			Output:   "Successfully wrote file",
		},
	})

	// Give it time to process.
	time.Sleep(50 * time.Millisecond)

	// Verify notifications were sent (start and complete).
	notifications := mockConn.GetNotifications()
	require.GreaterOrEqual(t, len(notifications), 2, "should have at least start and complete notifications")

	// Find the complete notification.
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

	// Check if diff content is present (ToolDiffContent).
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
