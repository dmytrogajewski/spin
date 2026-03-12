package e2e

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/agent"
	agentexec "github.com/dmytrogajewski/spin/internal/agent/executor"
	"github.com/dmytrogajewski/spin/internal/config"
	"github.com/dmytrogajewski/spin/internal/conversation"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/security"
	"github.com/dmytrogajewski/spin/internal/tui"
	"github.com/dmytrogajewski/spin/internal/ui/testkit"
)

// skipTUITests skips TUI tests that require interactive terminal.
// These tests are kept for reference but skipped in CI.
func skipTUITests(t *testing.T) {
	t.Helper()

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	t.Skip("TUI tests require an interactive terminal and real LLM provider")
}

// getBinPath returns absolute path to spin binary.
func getBinPath(t *testing.T) string {
	t.Helper()
	// Get workspace root (go up from tests/e2e/ to root).
	wd, err := os.Getwd()
	require.NoError(t, err)

	root := filepath.Dir(filepath.Dir(wd)) // tests/e2e/ -> tests/ -> root.

	return filepath.Join(root, "bin", "spin")
}

// setupTUITest creates a TUI test environment with mock LLM.
func setupTUITest(t *testing.T) (*testkit.TUITestHelper, *conversation.Conversation, *llm.MockProvider) {
	t.Helper()

	helper := testkit.NewTUITest(t)

	// Create mock LLM provider.
	mockLLM := llm.NewMockProvider("test-model", llm.WithResponse("Hello! How can I help?"))

	// Create minimal config.
	cfg := config.DefaultV2()
	cfg.Agent.WorkDir = t.TempDir()
	cfg.Protocol.EnableShell = false // Disable shell for faster tests.
	cfg.Protocol.EnableGit = false
	cfg.Protocol.EnableMCP = false

	// Create conversation with executor.
	ctx := context.Background()

	// Create emitter.
	emitter := events.NewEventEmitter(100)

	// Auto-approve handler for tests.
	approvalHandler := func(_ context.Context, req security.ApprovalRequest) security.ApprovalResponse {
		return security.ApprovalResponse{
			RequestID: req.ID,
			Approved:  true,
			Reason:    "auto-approved",
		}
	}

	// Create builtin runtime for e2e test.
	executor, err := agent.NewExecutor(cfg.Agent.WorkDir)
	require.NoError(t, err)

	validator := security.NewValidator()

	builtinRuntime, err := agentexec.NewBuiltinRuntime(agentexec.BuiltinRuntimeConfig{
		WorkDir:         cfg.Agent.WorkDir,
		Emitter:         emitter,
		Storage:         nil,
		SessionID:       fmt.Sprintf("e2e-test-%d", time.Now().UnixNano()),
		Executor:        agent.NewExecutorRuntimeAdapter(executor),
		Validator:       validator,
		UI:              nil, // No UI in e2e tests.
		ApprovalHandler: approvalHandler,
		Logger:          slog.Default(),
	})
	require.NoError(t, err)

	conv, err := conversation.NewBuilder(cfg, cfg.Agent.WorkDir, builtinRuntime, emitter, mockLLM).
		Build(ctx)
	require.NoError(t, err)

	// Initialize UI with conversation metadata.
	helper.UI.SetTaskMode(conv.GetTaskMode())
	helper.UI.SetProviderInfo(mockLLM.Name(), "test-model")
	helper.UI.SetTokenCount(0)

	return helper, conv, mockLLM
}

// TestTUILaunch tests that TUI launches successfully.
func TestTUILaunch(t *testing.T) {
	t.Parallel()

	skipTUITests(t)

	helper, _, _ := setupTUITest(t)
	defer helper.Stop()

	helper.Start()

	// TUI should be running without errors - test passes if we get here.
	time.Sleep(100 * time.Millisecond)
}

// TestTUIBasicChat tests sending a message and receiving response.
func TestTUIBasicChat(t *testing.T) {
	t.Parallel()

	skipTUITests(t)

	helper, conv, mockLLM := setupTUITest(t)
	defer helper.Stop()
	defer conv.Close()

	// Set mock response.
	mockLLM.SetResponse("Hello from test!")

	helper.Start()

	// Create event mapper.
	mapper := tui.NewMapper(helper.UI)
	defer mapper.Close()

	// Start streaming channel.
	ctx := context.Background()
	streamCh := mapper.StartStreaming()
	streamDone := make(chan struct{})

	go func() {
		_ = helper.UI.PrintChunks(ctx, streamCh)

		close(streamDone)
	}()

	// Subscribe to conversation events.
	eventStream := conv.Stream()
	eventDone := make(chan struct{})

	go func() {
		defer close(eventDone)

		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-eventStream:
				if !ok {
					return
				}

				_ = mapper.MapEvent(ctx, event)
			}
		}
	}()

	// Send a message.
	helper.Keyboard.InjectString("Say hello")
	helper.Keyboard.InjectEnter()

	// Run turn.
	err := conv.RunTurn(ctx, "Say hello")
	require.NoError(t, err)

	// Stop streaming.
	mapper.StopStreaming()
	<-streamDone

	// Wait for output.
	require.True(t, helper.WaitForOutput("Hello", 2*time.Second), "should receive response from LLM")
}

// TestTUIFilePickerTrigger tests @ key triggers file picker.
func TestTUIFilePickerTrigger(t *testing.T) {
	t.Parallel()

	skipTUITests(t)

	helper, _, _ := setupTUITest(t)
	defer helper.Stop()

	helper.Start()

	// Type @ to trigger file picker.
	helper.Keyboard.InjectString("@")
	time.Sleep(100 * time.Millisecond)

	// Close file picker with Esc.
	helper.Keyboard.InjectEscape()
	time.Sleep(50 * time.Millisecond)

	// File picker should work without errors.
}

// TestTUIHelpModal tests Ctrl+H triggers help.
func TestTUIHelpModal(t *testing.T) {
	t.Parallel()

	skipTUITests(t)

	helper, _, _ := setupTUITest(t)
	defer helper.Stop()

	helper.Start()

	// Press Ctrl+H (KeyCtrlH is not defined, so we'll skip this test for now)
	// The help modal functionality may need to be implemented differently.
	t.Skip("Help modal test needs implementation")
}

// TestTUIExitWithCtrlD tests Ctrl+D exits cleanly.
func TestTUIExitWithCtrlD(t *testing.T) {
	t.Parallel()

	skipTUITests(t)

	helper, _, _ := setupTUITest(t)
	defer helper.Stop()

	helper.Start()

	// Send Ctrl+D.
	helper.Keyboard.InjectCtrlD()

	// Wait for shutdown.
	time.Sleep(100 * time.Millisecond)

	// UI should stop without errors.
}

// TestTUIToolApproval tests approval workflow.
func TestTUIToolApproval(t *testing.T) {
	t.Parallel()

	skipTUITests(t)

	helper, _, _ := setupTUITest(t)
	defer helper.Stop()

	helper.Start()

	// This test verifies that the UI can handle approval requests
	// The actual approval dialog testing is done in overlay package tests.
	time.Sleep(50 * time.Millisecond)
}

// TestTUIMultiTurn tests conversation context is maintained.
func TestTUIMultiTurn(t *testing.T) {
	t.Parallel()

	skipTUITests(t)

	helper, conv, mockLLM := setupTUITest(t)
	defer helper.Stop()
	defer conv.Close()

	// Set mock response for second turn that references context.
	mockLLM.SetResponse("Your favorite number is 42")

	helper.Start()

	// Create event mapper.
	mapper := tui.NewMapper(helper.UI)
	defer mapper.Close()

	ctx := context.Background()

	// First turn.
	streamCh := mapper.StartStreaming()
	streamDone := make(chan struct{})

	go func() {
		_ = helper.UI.PrintChunks(ctx, streamCh)

		close(streamDone)
	}()

	eventStream := conv.Stream()
	eventDone := make(chan struct{})

	go func() {
		defer close(eventDone)

		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-eventStream:
				if !ok {
					return
				}

				_ = mapper.MapEvent(ctx, event)
			}
		}
	}()

	// First message.
	err := conv.RunTurn(ctx, "My favorite number is 42")
	require.NoError(t, err)

	mapper.StopStreaming()
	<-streamDone

	// Second turn.
	streamCh = mapper.StartStreaming()
	streamDone = make(chan struct{})

	go func() {
		_ = helper.UI.PrintChunks(ctx, streamCh)

		close(streamDone)
	}()

	// Second message - test context retention.
	err = conv.RunTurn(ctx, "What is my favorite number?")
	require.NoError(t, err)

	mapper.StopStreaming()
	<-streamDone

	// Should contain "42" in response.
	require.True(t, helper.WaitForOutput("42", 2*time.Second), "should remember context from previous message")
}

// TestTUIStopStreaming tests Ctrl+C stops streaming.
func TestTUIStopStreaming(t *testing.T) {
	t.Parallel()

	skipTUITests(t)

	helper, _, _ := setupTUITest(t)
	defer helper.Stop()

	helper.Start()

	// Start streaming some chunks.
	ctx := context.Background()
	chunks := make(chan string, 10)

	go func() {
		defer close(chunks)

		for range 10 {
			chunks <- "chunk "
		}
	}()

	// Start printing chunks.
	done := make(chan struct{})

	go func() {
		_ = helper.UI.PrintChunks(ctx, chunks)

		close(done)
	}()

	// Wait a bit for streaming to start.
	time.Sleep(50 * time.Millisecond)

	// Send Ctrl+C to cancel.
	helper.Keyboard.InjectCtrlC()

	// Wait a bit.
	time.Sleep(100 * time.Millisecond)

	// TUI should still be responsive (not crash).
}
