package acp

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/coder/acp-go-sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/conversation"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/mcp"
	"github.com/dmytrogajewski/spin/internal/message"
	"github.com/dmytrogajewski/spin/internal/session"
)

func TestCancel_ValidSession_CancelsInProgressExecution(t *testing.T) {
	t.Parallel()

	agentInstance, emitter := createTestAgentWithEmitter(t)
	mcpManager := mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default()))

	storage, err := session.NewFileStorage(t.TempDir())
	require.NoError(t, err)
	acpAgent, err := NewSpinACPAgentWithStorage(agentInstance, mcpManager, emitter, storage)
	require.NoError(t, err)

	// Set up ConversationManager with blocking harness executor.
	factory := func(_ context.Context, _ string, workDir string) (*conversation.Conversation, error) {
		return conversation.NewFromAgent(conversation.NewFromAgentConfig{
			Agent:           agentInstance,
			HarnessExecutor: &blockingHarnessExecutor{},
			Emitter:         emitter,
			WorkDir:         workDir,
		})
	}

	mgr, err := conversation.NewManager(conversation.ManagerConfig{Factory: factory})
	require.NoError(t, err)

	acpAgent.SetConversationManager(mgr)

	mockConn := &mockConnectionForCancel{}
	acpAgent.SetNotificationSender(mockConn)

	sessionResp, err := acpAgent.NewSession(context.Background(), acp.NewSessionRequest{
		Cwd: t.TempDir(),
	})
	require.NoError(t, err)

	sessionID := sessionResp.SessionId

	promptCompleted := make(chan acp.PromptResponse, 1)
	promptStarted := make(chan struct{})

	go func() {
		close(promptStarted)

		resp, promptErr := acpAgent.Prompt(context.Background(), acp.PromptRequest{
			SessionId: sessionID,
			Prompt:    []acp.ContentBlock{acp.TextBlock("test prompt")},
		})
		if promptErr != nil {
			promptCompleted <- acp.PromptResponse{StopReason: acp.StopReasonCancelled}
		} else {
			promptCompleted <- resp
		}
	}()

	<-promptStarted
	time.Sleep(50 * time.Millisecond)

	err = acpAgent.Cancel(context.Background(), acp.CancelNotification{SessionId: sessionID})
	require.NoError(t, err)

	select {
	case resp := <-promptCompleted:
		assert.Equal(t, acp.StopReasonCancelled, resp.StopReason)
	case <-time.After(3 * time.Second):
		t.Fatal("Prompt did not complete after cancellation")
	}
}

func TestCancel_ValidSession_NoInProgressExecution(t *testing.T) {
	t.Parallel(
	// Create test agent.
	)

	agentInstance := createTestAgent(t)
	mcpManager := mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default()))
	emitter := events.NewEventEmitter(100)

	storage, err := session.NewFileStorage(t.TempDir())
	require.NoError(t, err)
	acpAgent, err := NewSpinACPAgentWithStorage(agentInstance, mcpManager, emitter, storage)
	require.NoError(t, err)

	// Create session.
	sessionReq := acp.NewSessionRequest{
		Cwd: t.TempDir(),
	}
	sessionResp, err := acpAgent.NewSession(context.Background(), sessionReq)
	require.NoError(t, err)

	sessionID := sessionResp.SessionId

	// Cancel with no in-progress execution (should be no-op).
	notif := acp.CancelNotification{
		SessionId: sessionID,
	}
	err = acpAgent.Cancel(context.Background(), notif)
	require.NoError(t, err)
}

func TestCancel_InvalidSession_ReturnsError(t *testing.T) {
	t.Parallel(
	// Create test agent.
	)

	agentInstance := createTestAgent(t)
	mcpManager := mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default()))
	emitter := events.NewEventEmitter(100)

	storage, err := session.NewFileStorage(t.TempDir())
	require.NoError(t, err)
	acpAgent, err := NewSpinACPAgentWithStorage(agentInstance, mcpManager, emitter, storage)
	require.NoError(t, err)

	// Cancel with invalid session ID.
	notif := acp.CancelNotification{
		SessionId: acp.SessionId("invalid-session"),
	}
	err = acpAgent.Cancel(context.Background(), notif)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "session not found")
}

// blockingHarnessExecutor blocks until the context is canceled.
type blockingHarnessExecutor struct{}

func (b *blockingHarnessExecutor) Execute(
	ctx context.Context, _ string, _ []message.Message,
) (string, []message.Message, error) {
	<-ctx.Done()

	return "", nil, fmt.Errorf("harness executor blocked: %w", ctx.Err())
}

func TestCancel_CancelsPromptExecution_ReturnsCancelledStopReason(t *testing.T) {
	t.Parallel()

	agentInstance, emitter := createTestAgentWithEmitter(t)
	mcpManager := mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default()))

	storage, err := session.NewFileStorage(t.TempDir())
	require.NoError(t, err)
	acpAgent, err := NewSpinACPAgentWithStorage(agentInstance, mcpManager, emitter, storage)
	require.NoError(t, err)

	// Set up a ConversationManager with a blocking harness executor.
	factory := func(_ context.Context, _ string, workDir string) (*conversation.Conversation, error) {
		return conversation.NewFromAgent(conversation.NewFromAgentConfig{
			Agent:           agentInstance,
			HarnessExecutor: &blockingHarnessExecutor{},
			Emitter:         emitter,
			WorkDir:         workDir,
		})
	}

	mgr, err := conversation.NewManager(conversation.ManagerConfig{Factory: factory})
	require.NoError(t, err)

	acpAgent.SetConversationManager(mgr)

	// Create mock connection.
	mockConn := &mockConnectionForCancel{}
	acpAgent.SetNotificationSender(mockConn)

	// Create session.
	sessionResp, err := acpAgent.NewSession(context.Background(), acp.NewSessionRequest{
		Cwd: t.TempDir(),
	})
	require.NoError(t, err)

	sessionID := sessionResp.SessionId

	// Channel to signal when prompt completes.
	promptDone := make(chan acp.PromptResponse, 1)
	promptStarted := make(chan struct{})

	// Start prompt execution in background.
	go func() {
		close(promptStarted)

		resp, _ := acpAgent.Prompt(context.Background(), acp.PromptRequest{
			SessionId: sessionID,
			Prompt:    []acp.ContentBlock{acp.TextBlock("test prompt")},
		})
		promptDone <- resp
	}()

	// Wait for prompt to start, then cancel.
	<-promptStarted
	time.Sleep(50 * time.Millisecond)

	err = acpAgent.Cancel(context.Background(), acp.CancelNotification{
		SessionId: sessionID,
	})
	require.NoError(t, err)

	// Wait for prompt to complete.
	select {
	case resp := <-promptDone:
		assert.Equal(t, acp.StopReasonCancelled, resp.StopReason,
			"Expected canceled stop reason, but got %s", resp.StopReason)
	case <-time.After(3 * time.Second):
		t.Fatal("Prompt did not complete after cancellation")
	}
}

// mockConnectionForCancel is a mock connection for cancel tests.
type mockConnectionForCancel struct {
	mu            sync.Mutex
	notifications []acp.SessionNotification
}

func (m *mockConnectionForCancel) SessionUpdate(_ context.Context, notification acp.SessionNotification) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.notifications = append(m.notifications, notification)

	return nil
}

func (m *mockConnectionForCancel) RequestPermission(
	_ context.Context, params acp.RequestPermissionRequest,
) (acp.RequestPermissionResponse, error) {
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

func (m *mockConnectionForCancel) GetNotifications() []acp.SessionNotification {
	m.mu.Lock()
	defer m.mu.Unlock()

	return append([]acp.SessionNotification{}, m.notifications...)
}
