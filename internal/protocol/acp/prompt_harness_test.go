package acp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/coder/acp-go-sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/agent"
	"github.com/dmytrogajewski/spin/internal/agent/caller"
	"github.com/dmytrogajewski/spin/internal/agent/harness"
	"github.com/dmytrogajewski/spin/internal/agent/harness/bridge"
	"github.com/dmytrogajewski/spin/internal/agent/prompt"
	"github.com/dmytrogajewski/spin/internal/agent/scaffold"
	"github.com/dmytrogajewski/spin/internal/agent/tool"
	"github.com/dmytrogajewski/spin/internal/conversation"
	"github.com/dmytrogajewski/spin/internal/cycle"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/mcp"
	"github.com/dmytrogajewski/spin/internal/message"
	"github.com/dmytrogajewski/spin/internal/safety"
	"github.com/dmytrogajewski/spin/internal/session"
	"github.com/dmytrogajewski/spin/internal/tools"
)

// errLLMFailed simulates an LLM provider failure.
var errLLMFailed = errors.New("LLM provider connection failed")

// failingHarnessExecutor simulates a harness executor whose LLM call fails.
type failingHarnessExecutor struct{}

func (f *failingHarnessExecutor) Execute(
	_ context.Context, _ string, _ []message.Message,
) (string, []message.Message, error) {
	return "", nil, fmt.Errorf("harness execution failed: %w", errLLMFailed)
}

// TestSpinACPAgent_Prompt_RealHarnessExecutor verifies notifications flow through
// the real harness executor (LLMCaller → MockProvider) to the ACP subscriber.
func TestSpinACPAgent_Prompt_RealHarnessExecutor(t *testing.T) {
	t.Parallel()

	emitter := events.NewEventEmitter(100)
	defer emitter.Close()

	mockProvider := llm.NewMockProvider("test-provider")

	// Build agent (same as production).
	validator := safety.NewValidator()
	approvalService := safety.NewApprovalServiceWithConfig(safety.ApprovalServiceConfig{
		Handler: nil, Emitter: emitter, Validator: validator,
	})
	securityService := safety.NewService(validator, approvalService)
	detectionService := cycle.NewService(cycle.NewDetector(cycle.Config{Enabled: false}), nil)
	toolRegistry := tools.NewRegistry()
	toolRuntime := tool.NewRuntime(tool.RuntimeConfig{
		Registry:        toolRegistry,
		Validator:       validator,
		ApprovalService: approvalService,
		Emitter:         emitter,
		WorkDir:         t.TempDir(),
	})

	agentInstance, err := agent.NewAgent(
		mockProvider,
		securityService,
		detectionService,
		toolRuntime,
		&agent.Environment{WorkDir: t.TempDir()},
		emitter,
	)
	require.NoError(t, err)

	// Build real harness executor (mirrors buildACPHarnessExecutor in acp.go).
	logger := slog.Default()
	pb := prompt.New(mockProvider, logger)
	llmCaller := caller.New(caller.Config{
		Provider:      mockProvider,
		PromptBuilder: pb,
		Emitter:       emitter,
		Logger:        logger,
	})

	spec := &scaffold.Spec{
		SystemPrompt: "You are a test assistant.",
		Config: scaffold.SpecConfig{
			MaxTurns: 1,
		},
	}

	harnessExec, err := bridge.BuildExecutor(bridge.Config{
		Spec:        spec,
		LLMCaller:   llmCaller,
		Registry:    toolRegistry,
		Runtime:     toolRuntime,
		Logger:      logger,
		HarnessOpts: []harness.Option{harness.WithEmitter(emitter)},
	})
	require.NoError(t, err)

	turnExec := bridge.NewTurnExecutor(harnessExec)

	// Build conversation manager with real executor.
	factory := func(_ context.Context, _ string, workDir string) (*conversation.Conversation, error) {
		return conversation.NewFromAgent(conversation.NewFromAgentConfig{
			Agent:           agentInstance,
			HarnessExecutor: turnExec,
			Emitter:         emitter,
			WorkDir:         workDir,
		})
	}

	mgr, err := conversation.NewManager(conversation.ManagerConfig{
		Factory: factory,
	})
	require.NoError(t, err)

	// Create ACP agent.
	mcpManager := mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default()))
	storage, err := session.NewFileStorage(t.TempDir())
	require.NoError(t, err)

	acpAgent, err := NewSpinACPAgentWithStorage(agentInstance, mcpManager, emitter, storage)
	require.NoError(t, err)
	acpAgent.SetConversationManager(mgr)

	mockConn := &mockConnection{}
	acpAgent.SetNotificationSender(mockConn)

	// Create session.
	sessionResp, err := acpAgent.NewSession(context.Background(), acp.NewSessionRequest{
		Cwd: t.TempDir(),
	})
	require.NoError(t, err)

	// Execute prompt.
	resp, err := acpAgent.Prompt(context.Background(), acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt:    []acp.ContentBlock{acp.TextBlock("hi")},
	})
	require.NoError(t, err)
	assert.Equal(t, acp.StopReasonEndTurn, resp.StopReason)

	// Check notifications — there should be at least one agent_message_chunk.
	require.Eventually(t, func() bool {
		for _, n := range mockConn.GetNotifications() {
			if chunk := n.Update.AgentMessageChunk; chunk != nil {
				if chunk.Content.Text != nil && strings.TrimSpace(chunk.Content.Text.Text) != "" {
					return true
				}
			}
		}

		return false
	}, 2*time.Second, 10*time.Millisecond,
		"expected agent_message_chunk notification from real harness executor, got: %v",
		mockConn.GetNotifications())
}

// TestSpinACPAgent_Prompt_ErrorSwallowed reproduces the ACP empty response bug:
// when RunTurn fails (e.g. LLM error), the error is silently mapped to end_turn
// with NO notification sent to the client — the client sees an empty response.
func TestSpinACPAgent_Prompt_ErrorSwallowed(t *testing.T) {
	t.Parallel()

	agentInstance, emitter := createTestAgentWithEmitter(t)
	mcpManager := mcp.NewService(mcp.NewDefaultRegistryManager(slog.Default()))

	storage, err := session.NewFileStorage(t.TempDir())
	require.NoError(t, err)

	acpAgent, err := NewSpinACPAgentWithStorage(agentInstance, mcpManager, emitter, storage)
	require.NoError(t, err)

	// Use a failing executor to simulate LLM failure.
	factory := func(_ context.Context, _ string, workDir string) (*conversation.Conversation, error) {
		return conversation.NewFromAgent(conversation.NewFromAgentConfig{
			Agent:           agentInstance,
			HarnessExecutor: &failingHarnessExecutor{},
			Emitter:         emitter,
			WorkDir:         workDir,
		})
	}

	mgr, err := conversation.NewManager(conversation.ManagerConfig{
		Factory: factory,
	})
	require.NoError(t, err)
	acpAgent.SetConversationManager(mgr)

	mockConn := &mockConnection{}
	acpAgent.SetNotificationSender(mockConn)

	sessionResp, err := acpAgent.NewSession(context.Background(), acp.NewSessionRequest{
		Cwd: t.TempDir(),
	})
	require.NoError(t, err)

	// Execute prompt — RunTurn will fail.
	resp, err := acpAgent.Prompt(context.Background(), acp.PromptRequest{
		SessionId: sessionResp.SessionId,
		Prompt:    []acp.ContentBlock{acp.TextBlock("hi")},
	})
	// Prompt currently returns nil even when RunTurn failed.
	require.NoError(t, err)
	assert.Equal(t, acp.StopReasonEndTurn, resp.StopReason)

	// After fix: client receives an error notification explaining what went wrong.
	time.Sleep(50 * time.Millisecond)

	notifications := mockConn.GetNotifications()

	hasErrorNotification := false

	for _, n := range notifications {
		if chunk := n.Update.AgentMessageChunk; chunk != nil {
			if chunk.Content.Text != nil && strings.Contains(chunk.Content.Text.Text, "Error") {
				hasErrorNotification = true
			}
		}
	}

	assert.True(t, hasErrorNotification,
		"expected error notification when RunTurn fails, got %d notifications: %v", len(notifications), notifications)
}
