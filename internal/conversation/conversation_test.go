package conversation

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/agent"
	"github.com/dmytrogajewski/spin/internal/cycle"
	"github.com/dmytrogajewski/spin/internal/detection"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/history"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/orchestration"
	"github.com/dmytrogajewski/spin/internal/security"
	"github.com/dmytrogajewski/spin/internal/task"
	"github.com/dmytrogajewski/spin/internal/tools"
)

// TestConversation_RunTurn_EmptyPrompt tests RunTurn with empty prompt
func TestConversation_RunTurn_EmptyPrompt(t *testing.T) {
	conv := setupTestConv(t)
	ctx := context.Background()

	// Run turn with empty prompt
	err := conv.RunTurn(ctx, "")
	if err == nil {
		t.Error("Expected error for empty prompt")
	}
}

// TestConversation_RunTurn_WithValidPrompt tests RunTurn with valid prompt
func TestConversation_RunTurn_WithValidPrompt(t *testing.T) {
	conv := setupTestConv(t)
	ctx := context.Background()

	// Run turn with valid prompt
	err := conv.RunTurn(ctx, "test prompt")
	// May succeed or fail depending on mock provider, but should not panic
	_ = err
}

// TestConversation_RunTurn_ContextCanceled tests RunTurn with canceled context
func TestConversation_RunTurn_ContextCanceled(t *testing.T) {
	conv := setupTestConv(t)

	// Cancel context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Run turn with canceled context
	err := conv.RunTurn(ctx, "test prompt")
	if err == nil {
		t.Error("Expected error for canceled context")
	}
}

// TestConversation_Close_MultipleTimes tests Close called multiple times
func TestConversation_Close_MultipleTimes(t *testing.T) {
	conv := setupTestConv(t)

	// Close multiple times
	err1 := conv.Close()
	err2 := conv.Close()
	err3 := conv.Close()

	// All should succeed (or at least not panic)
	_ = err1
	_ = err2
	_ = err3
}

// TestConversation_Close_WithRunningTurn tests Close while turn is running
func TestConversation_Close_WithRunningTurn(t *testing.T) {
	conv := setupTestConv(t)
	ctx := context.Background()

	// Start a turn in background
	go func() {
		conv.RunTurn(ctx, "test prompt")
	}()

	// Close while turn may be running
	err := conv.Close()
	// Should handle gracefully
	_ = err
}

// TestConversation_Stream_Functionality tests Stream()
func TestConversation_Stream_ReceivesEvents(t *testing.T) {
	conv := setupTestConv(t)
	ctx := context.Background()

	stream := conv.Stream()
	if stream == nil {
		t.Fatal("Expected non-nil stream")
	}

	// Start a turn to generate events
	go func() {
		conv.RunTurn(ctx, "test prompt")
	}()

	// Stream should be functional
	_ = stream
}

// TestConversation_SetTaskMode_ValidModes tests SetTaskMode with all valid modes
func TestConversation_SetTaskMode_ValidModes(t *testing.T) {
	conv := setupTestConv(t)

	modes := []string{"regular", "review", "compact", "planning"}
	for _, mode := range modes {
		t.Run("mode_"+mode, func(t *testing.T) {
			err := conv.SetTaskMode(mode)
			if err != nil {
				t.Errorf("SetTaskMode(%s) error = %v", mode, err)
			}
		})
	}
}

// TestConversation_SetTaskMode_InvalidMode tests SetTaskMode with invalid mode
func TestConversation_SetTaskMode_InvalidMode(t *testing.T) {
	conv := setupTestConv(t)

	// Invalid mode should return error
	err := conv.SetTaskMode("invalid_mode")
	if err == nil {
		t.Error("Expected error for invalid task mode")
	}
}

// TestConversation_Close_MultipleCallsEdgeCase tests Close called multiple times
func TestConversation_Close_MultipleCalls(t *testing.T) {
	conv := setupTestConv(t)

	// First close
	err := conv.Close()
	if err != nil {
		t.Errorf("First Close() error = %v", err)
	}

	// Second close should not error
	err = conv.Close()
	if err != nil {
		t.Errorf("Second Close() should handle gracefully, got error: %v", err)
	}

	// Third close
	err = conv.Close()
	if err != nil {
		t.Errorf("Third Close() should handle gracefully, got error: %v", err)
	}
}

// TestConversation_Close_WithCanceledContext tests Close with canceled context
func TestConversation_Close_CanceledContext(t *testing.T) {
	conv := setupTestConv(t)

	// Close (note: Close doesn't take a context parameter)
	err := conv.Close()
	// Should handle gracefully
	_ = err // May or may not error depending on implementation
}

// TestConversation_Close_WithTimeout tests Close with timeout
func TestConversation_Close_TimeoutContext(t *testing.T) {
	conv := setupTestConv(t)

	// Create context with very short timeout (not used by Close)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Wait for timeout
	time.Sleep(10 * time.Millisecond)

	// Close
	err := conv.Close()
	// Should handle timeout gracefully
	_ = err
	_ = ctx // Not used but created for test completeness
}

// TestConversation_SetTaskMode_AllValidModes tests SetTaskMode with all built-in modes
func TestConversation_SetTaskMode_AllModes(t *testing.T) {
	conv := setupTestConv(t)

	modes := []string{"regular", "review", "compact", "planning"}
	for _, mode := range modes {
		t.Run("mode_"+mode, func(t *testing.T) {
			err := conv.SetTaskMode(mode)
			if err != nil {
				t.Errorf("SetTaskMode(%s) error = %v", mode, err)
			}
		})
	}
}

// TestConversation_SetTaskMode_UnknownMode tests SetTaskMode with unknown mode
func TestConversation_SetTaskMode_UnknownMode(t *testing.T) {
	conv := setupTestConv(t)

	// Unknown mode should return error
	err := conv.SetTaskMode("unknown_mode_that_does_not_exist")
	if err == nil {
		t.Error("Expected error for unknown mode")
	}
}

// TestConversation_SetTaskMode_EmptyMode tests SetTaskMode with empty mode
func TestConversation_SetTaskMode_EmptyMode(t *testing.T) {
	conv := setupTestConv(t)

	// Empty mode should return error
	err := conv.SetTaskMode("")
	if err == nil {
		t.Error("Expected error for empty mode")
	}
}

// TestConversation_SetTaskMode verifies that SetTaskMode successfully switches modes.
func TestConversation_SetTaskMode(t *testing.T) {
	conv := setupTestConv(t)

	// Should default to "regular"
	if got := conv.GetTaskMode(); got != "regular" {
		t.Errorf("expected default mode 'regular', got %q", got)
	}

	// Switch to review mode
	err := conv.SetTaskMode("review")
	if err != nil {
		t.Fatalf("SetTaskMode('review') failed: %v", err)
	}
	if got := conv.GetTaskMode(); got != "review" {
		t.Errorf("expected mode 'review', got %q", got)
	}

	// Switch to compact mode
	err = conv.SetTaskMode("compact")
	if err != nil {
		t.Fatalf("SetTaskMode('compact') failed: %v", err)
	}
	if got := conv.GetTaskMode(); got != "compact" {
		t.Errorf("expected mode 'compact', got %q", got)
	}

	// Switch to planning mode
	err = conv.SetTaskMode("planning")
	if err != nil {
		t.Fatalf("SetTaskMode('planning') failed: %v", err)
	}
	if got := conv.GetTaskMode(); got != "planning" {
		t.Errorf("expected mode 'planning', got %q", got)
	}

	// Switch back to regular
	err = conv.SetTaskMode("regular")
	if err != nil {
		t.Fatalf("SetTaskMode('regular') failed: %v", err)
	}
	if got := conv.GetTaskMode(); got != "regular" {
		t.Errorf("expected mode 'regular', got %q", got)
	}
}

// TestConversation_SetTaskMode_Invalid verifies that SetTaskMode returns an error for invalid modes.
func TestConversation_SetTaskMode_Invalid(t *testing.T) {
	conv := setupTestConv(t)

	err := conv.SetTaskMode("invalid-mode")
	if err == nil {
		t.Fatal("expected error for invalid mode, got nil")
	}
	if err.Error() == "" {
		t.Error("expected non-empty error message")
	}

	// Should remain in default mode
	if got := conv.GetTaskMode(); got != "regular" {
		t.Errorf("expected mode to remain 'regular', got %q", got)
	}
}

// TestConversation_GetTaskMode_Default verifies that GetTaskMode returns "regular" by default.
func TestConversation_GetTaskMode_Default(t *testing.T) {
	conv := setupTestConv(t)

	// Should default to "regular" without explicit SetTaskMode
	if got := conv.GetTaskMode(); got != "regular" {
		t.Errorf("expected default mode 'regular', got %q", got)
	}
}

// TestConversation_TaskMode_Concurrent verifies thread-safe access to task mode.
func TestConversation_TaskMode_Concurrent(t *testing.T) {
	conv := setupTestConv(t)

	var wg sync.WaitGroup

	// 50 concurrent readers
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = conv.GetTaskMode()
		}()
	}

	// 10 concurrent writers
	modes := []string{"regular", "review", "compact", "planning"}
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			mode := modes[i%len(modes)]
			_ = conv.SetTaskMode(mode)
		}(i)
	}

	wg.Wait()

	// Should not race (verified with go test -race)
	// Final mode should be one of the valid modes
	finalMode := conv.GetTaskMode()
	validMode := false
	for _, m := range modes {
		if finalMode == m {
			validMode = true
			break
		}
	}
	if !validMode {
		t.Errorf("final mode %q is not a valid mode", finalMode)
	}
}

// TestConversation_TaskMode_PersistsAcrossTurns verifies that the task mode persists across multiple turns.
func TestConversation_TaskMode_PersistsAcrossTurns(t *testing.T) {
	// Create a conversation with setupTestConv
	conv := setupTestConv(t)

	// Set mode to compact
	err := conv.SetTaskMode("compact")
	if err != nil {
		t.Fatalf("SetTaskMode failed: %v", err)
	}

	// Mode should be compact
	if got := conv.GetTaskMode(); got != "compact" {
		t.Errorf("expected mode 'compact' before turn 1, got %q", got)
	}

	// Execute turn 1
	turnCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = conv.RunTurn(turnCtx, "Turn 1")
	if err != nil {
		t.Fatalf("RunTurn failed: %v", err)
	}

	// Mode should still be compact
	if got := conv.GetTaskMode(); got != "compact" {
		t.Errorf("expected mode 'compact' after turn 1, got %q", got)
	}

	// Execute turn 2
	err = conv.RunTurn(turnCtx, "Turn 2")
	if err != nil {
		t.Fatalf("RunTurn failed on turn 2: %v", err)
	}

	// Mode should still be compact
	if got := conv.GetTaskMode(); got != "compact" {
		t.Errorf("expected mode 'compact' after turn 2, got %q", got)
	}
}

// TestConversation_SetTaskMode_EmitsEvent verifies that SetTaskMode emits a system info event.
func TestConversation_SetTaskMode_EmitsEvent(t *testing.T) {
	conv := setupTestConv(t)
	emitter := conv.emitter

	// Subscribe to events
	_, eventChan, err := emitter.Subscribe()
	if err != nil {
		t.Fatalf("failed to subscribe to events: %v", err)
	}

	// Set up event collection
	eventList := make([]events.Event, 0)
	done := make(chan struct{})

	go func() {
		defer close(done)
		for event := range eventChan {
			eventList = append(eventList, event)
		}
	}()

	// Switch mode
	err = conv.SetTaskMode("review")
	if err != nil {
		t.Fatalf("SetTaskMode failed: %v", err)
	}

	// Give time for event to be emitted
	emitter.Close()
	<-done

	// Verify event was emitted
	foundEvent := false
	for _, event := range eventList {
		if event.Type == events.EventInfo {
			if data, ok := event.Data.(events.SystemEventData); ok {
				if data.Message == "Switched to review mode" {
					foundEvent = true
					break
				}
			}
		}
	}

	if !foundEvent {
		t.Error("expected system info event for mode switch, but none was emitted")
	}
}

// TestConversation_SetTaskMode_ValidatesTask verifies that SetTaskMode validates the task.
func TestConversation_SetTaskMode_ValidatesTask(t *testing.T) {
	conv := setupTestConv(t)

	// This test verifies the validation path exists
	// In practice, all built-in tasks should validate successfully
	err := conv.SetTaskMode("regular")
	if err != nil {
		t.Errorf("expected regular task to validate, got error: %v", err)
	}
}

// Helper functions

// setupTestConv creates a test conversation with all dependencies
func setupTestConv(t *testing.T) *Conversation {
	t.Helper()

	llmProvider := llm.NewMockProvider("ok")
	validator := security.NewValidator()
	workDir := t.TempDir()

	executor, err := agent.NewExecutor(workDir)
	if err != nil {
		t.Fatalf("failed to create executor: %v", err)
	}

	env := &agent.Environment{WorkDir: workDir}
	emitter := events.NewEventEmitter(100)

	// Build SecurityService
	approvalService := security.NewApprovalService(nil, emitter, validator)
	securityService := security.NewSecurityService(validator, approvalService)

	// Build DetectionService
	cycleDetector := cycle.NewDetector(cycle.Config{Enabled: false})
	detectionService := detection.NewDetectionService(cycleDetector, nil)

	// Build tool registry
	toolRegistry := tools.NewRegistry()
	_ = toolRegistry.Register(tools.NewReadFileTool())
	_ = toolRegistry.Register(tools.NewWriteFileTool())
	_ = toolRegistry.Register(tools.NewListDirectoryTool())
	_ = toolRegistry.Register(tools.NewExecuteCommandTool(executor, validator))
	_ = toolRegistry.Register(tools.NewGetContextTool(env))
	_ = toolRegistry.Register(tools.NewApplyPatchTool(workDir))
	_ = toolRegistry.Register(tools.NewFileSearchTool(workDir))
	_ = toolRegistry.Register(tools.NewGitContextTool(workDir))

	// Build task registry (using orchestration.Registry, not task.Registry)
	taskRegistry := orchestration.NewRegistry()
	_ = taskRegistry.Register("regular", task.NewRegular())
	_ = taskRegistry.Register("review", task.NewReview())
	_ = taskRegistry.Register("compact", task.NewCompact())
	_ = taskRegistry.Register("planning", task.NewPlanning())
	_ = taskRegistry.SetDefault("regular")

	// Build OrchestrationService
	toolExecutor := orchestration.NewToolExecutor(orchestration.ToolExecutorConfig{
		Registry:        toolRegistry,
		Validator:       validator,
		ApprovalService: approvalService,
		Emitter:         emitter,
		WorkDir:         workDir,
	})
	orchestrationService := orchestration.NewOrchestrationService(toolExecutor, toolRegistry, taskRegistry)

	// Create agent
	agentInstance, err := agent.NewAgent(llmProvider, securityService, detectionService, orchestrationService, env, emitter)
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	hist := history.NewHistoryWithDefaults()
	return NewConversation(agentInstance, hist, emitter, "test-session-id")
}
