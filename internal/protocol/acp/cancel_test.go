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

	"github.com/dmytrogajewski/spin/internal/agent"
	"github.com/dmytrogajewski/spin/internal/cycle"
	"github.com/dmytrogajewski/spin/internal/detection"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/mcp"
	"github.com/dmytrogajewski/spin/internal/planning"
	"github.com/dmytrogajewski/spin/internal/security"
	"github.com/dmytrogajewski/spin/internal/session"
	"github.com/dmytrogajewski/spin/internal/tools"
)

func TestCancel_ValidSession_CancelsInProgressExecution(t *testing.T) {
	t.Parallel(
	// Create test agent with blocking mock provider and mock connection.
	)

	agentInstance := createBlockingTestAgent(t)
	mcpManager := mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default()))
	emitter := events.NewEventEmitter(100)

	storage, err := session.NewFileStorage(t.TempDir())
	require.NoError(t, err)
	acpAgent, err := NewSpinACPAgentWithStorage(agentInstance, mcpManager, emitter, storage)
	require.NoError(t, err)

	// Create mock connection.
	mockConn := &mockConnectionForCancel{}
	acpAgent.SetNotificationSender(mockConn)

	// Create session.
	sessionReq := acp.NewSessionRequest{
		Cwd: t.TempDir(),
	}
	sessionResp, err := acpAgent.NewSession(context.Background(), sessionReq)
	require.NoError(t, err)

	sessionID := sessionResp.SessionId

	// Channel to signal when prompt completes.
	promptCompleted := make(chan acp.PromptResponse, 1)
	// Channel to signal when prompt starts (so we can cancel after it starts).
	promptStarted := make(chan struct{})

	// Start prompt in goroutine
	// Use background context - cancellation will be done via Cancel method.
	go func() {
		close(promptStarted)

		promptReq := acp.PromptRequest{
			SessionId: sessionID,
			Prompt:    []acp.ContentBlock{acp.TextBlock("test prompt")},
		}

		resp, promptErr := acpAgent.Prompt(context.Background(), promptReq)
		if promptErr != nil {
			// Error is expected when canceled.
			promptCompleted <- acp.PromptResponse{
				StopReason: acp.StopReasonCancelled,
			}
		} else {
			promptCompleted <- resp
		}
	}()

	// Wait for prompt to start.
	<-promptStarted
	// Give it a moment to get into the LLM call (mock provider will block on first chunk delay)
	// The delay is 200ms, so we wait 50ms to ensure we're in the delay.
	time.Sleep(50 * time.Millisecond)

	// Cancel the execution.
	notif := acp.CancelNotification{
		SessionId: sessionID,
	}
	err = acpAgent.Cancel(context.Background(), notif)
	require.NoError(t, err)

	// Wait for prompt to complete (should be canceled).
	select {
	case resp := <-promptCompleted:
		// Verify stop reason is canceled.
		assert.Equal(t, acp.StopReasonCancelled, resp.StopReason, "Expected canceled stop reason, but got %s", resp.StopReason)
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

func TestCancel_CancelsPromptExecution_ReturnsCancelledStopReason(t *testing.T) {
	t.Parallel(
	// Create test agent with blocking mock provider and mock connection.
	)

	agentInstance := createBlockingTestAgent(t)
	mcpManager := mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default()))
	emitter := events.NewEventEmitter(100)

	storage, err := session.NewFileStorage(t.TempDir())
	require.NoError(t, err)
	acpAgent, err := NewSpinACPAgentWithStorage(agentInstance, mcpManager, emitter, storage)
	require.NoError(t, err)

	// Create mock connection.
	mockConn := &mockConnectionForCancel{}
	acpAgent.SetNotificationSender(mockConn)

	// Create session.
	sessionReq := acp.NewSessionRequest{
		Cwd: t.TempDir(),
	}
	sessionResp, err := acpAgent.NewSession(context.Background(), sessionReq)
	require.NoError(t, err)

	sessionID := sessionResp.SessionId

	// Channel to signal when prompt completes.
	promptDone := make(chan acp.PromptResponse, 1)
	// Channel to signal when prompt starts.
	promptStarted := make(chan struct{})

	// Start prompt execution in background.
	go func() {
		close(promptStarted)

		promptReq := acp.PromptRequest{
			SessionId: sessionID,
			Prompt:    []acp.ContentBlock{acp.TextBlock("test prompt")},
		}

		resp, _ := acpAgent.Prompt(context.Background(), promptReq)
		promptDone <- resp
	}()

	// Wait for prompt to start.
	<-promptStarted
	// Give it a moment to get into the LLM call (mock provider will block on first chunk delay).
	time.Sleep(50 * time.Millisecond)

	// Cancel the execution.
	notif := acp.CancelNotification{
		SessionId: sessionID,
	}
	err = acpAgent.Cancel(context.Background(), notif)
	require.NoError(t, err)

	// Wait for prompt to complete.
	select {
	case resp := <-promptDone:
		// Verify stop reason is canceled.
		assert.Equal(t, acp.StopReasonCancelled, resp.StopReason, "Expected canceled stop reason, but got %s", resp.StopReason)
	case <-time.After(3 * time.Second):
		t.Fatal("Prompt did not complete after cancellation")
	}

	// Verify notifications were sent.
	notifications := mockConn.GetNotifications()
	// Should have at least the user message notification.
	assert.GreaterOrEqual(t, len(notifications), 1)
}

// createBlockingTestAgent creates a test agent with a blocking mock provider.
// The provider blocks until the context is canceled, allowing cancellation tests.
func createBlockingTestAgent(t *testing.T) *agent.Agent {
	t.Helper()

	// Create a mock provider with a long delay on the first chunk
	// This makes the Stream method block long enough for us to cancel it
	// Use many chunks with delay to ensure the agent execution blocks.
	chunks := make([]string, 10)
	for i := range chunks {
		chunks[i] = fmt.Sprintf("chunk%d", i+1)
	}

	mockProvider := llm.NewMockProvider("test",
		llm.WithStreaming(chunks),
		llm.WithDelay(500*time.Millisecond), llm.WithDelay(500*time.Millisecond), // Long delay between chunks to allow cancellation.
	)
	validator := security.NewValidator()
	emitter := events.NewEventEmitter(100)
	approvalService := security.NewApprovalServiceWithConfig(security.ApprovalServiceConfig{Handler: nil, Emitter: emitter, Validator: validator})
	securityService := security.NewService(validator, approvalService)
	detectionService := detection.NewService(cycle.NewDetector(cycle.Config{Enabled: false}), nil)
	toolRuntime := agent.NewToolRuntime(agent.ToolRuntimeConfig{
		Registry:        tools.NewRegistry(),
		Validator:       validator,
		ApprovalService: approvalService,
		Emitter:         emitter,
		WorkDir:         t.TempDir(),
	})
	planningService := planning.NewService(mockProvider)

	agentInstance, err := agent.NewAgent(
		mockProvider,
		securityService,
		detectionService,
		toolRuntime,
		planningService,
		&agent.Environment{WorkDir: t.TempDir()},
		emitter,
	)
	require.NoError(t, err)

	return agentInstance
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

func (m *mockConnectionForCancel) RequestPermission(_ context.Context, params acp.RequestPermissionRequest) (acp.RequestPermissionResponse, error) {
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
