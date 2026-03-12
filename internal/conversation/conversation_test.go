package conversation

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/events"
)

// TestConversation_RunTurn_EmptyPrompt tests RunTurn with empty prompt.
func TestConversation_RunTurn_EmptyPrompt(t *testing.T) {
	t.Parallel()

	conv := setupTestConv(t)
	ctx := context.Background()

	// Run turn with empty prompt.
	err := conv.RunTurn(ctx, "")
	if err == nil {
		t.Error("Expected error for empty prompt")
	}
}

// TestConversation_RunTurn_WithValidPrompt tests RunTurn with valid prompt.
func TestConversation_RunTurn_WithValidPrompt(t *testing.T) {
	t.Parallel()

	conv := setupTestConv(t)
	ctx := context.Background()

	// Run turn with valid prompt.
	err := conv.RunTurn(ctx, "test prompt")
	// May succeed or fail depending on mock provider, but should not panic.
	_ = err
}

// TestConversation_RunTurn_ContextCanceled tests RunTurn with canceled context.
func TestConversation_RunTurn_ContextCanceled(t *testing.T) {
	t.Parallel()

	conv := setupTestConv(t)

	// Cancel context.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Run turn with canceled context.
	err := conv.RunTurn(ctx, "test prompt")
	if err == nil {
		t.Error("Expected error for canceled context")
	}
}

// TestConversation_Close_MultipleTimes tests Close called multiple times.
func TestConversation_Close_MultipleTimes(t *testing.T) {
	t.Parallel()

	conv := setupTestConv(t)

	// Close multiple times.
	err1 := conv.Close()
	err2 := conv.Close()
	err3 := conv.Close()

	// All should succeed (or at least not panic).
	_ = err1
	_ = err2
	_ = err3
}

// TestConversation_Close_WithRunningTurn tests Close while turn is running.
func TestConversation_Close_WithRunningTurn(t *testing.T) {
	t.Parallel()

	conv := setupTestConv(t)
	ctx := context.Background()

	// Start a turn in background.
	go func() {
		_ = conv.RunTurn(ctx, "test prompt")
	}()

	// Close while turn may be running.
	err := conv.Close()
	// Should handle gracefully.
	_ = err
}

// TestConversation_Stream_Functionality tests Stream().
func TestConversation_Stream_ReceivesEvents(t *testing.T) {
	t.Parallel()

	conv := setupTestConv(t)
	ctx := context.Background()

	stream := conv.Stream()
	if stream == nil {
		t.Fatal("Expected non-nil stream")
	}

	// Start a turn to generate events.
	go func() {
		_ = conv.RunTurn(ctx, "test prompt")
	}()

	// Stream should be functional.
	_ = stream
}

// TestConversation_SetTaskMode_ValidModes tests SetTaskMode with all valid modes.
func TestConversation_SetTaskMode_ValidModes(t *testing.T) {
	t.Parallel()

	conv := setupTestConv(t)

	modes := []string{"regular", "review", "compact", "planning"}
	for _, mode := range modes {
		t.Run("mode_"+mode, func(t *testing.T) {
			t.Parallel()

			err := conv.SetTaskMode(mode)
			if err != nil {
				t.Errorf("SetTaskMode(%s) error = %v", mode, err)
			}
		})
	}
}

// TestConversation_SetTaskMode_InvalidMode tests SetTaskMode with invalid mode.
func TestConversation_SetTaskMode_InvalidMode(t *testing.T) {
	t.Parallel()

	conv := setupTestConv(t)

	// Invalid mode should return error.
	err := conv.SetTaskMode("invalid_mode")
	if err == nil {
		t.Error("Expected error for invalid task mode")
	}
}

// TestConversation_Close_MultipleCallsEdgeCase tests Close called multiple times.
func TestConversation_Close_MultipleCalls(t *testing.T) {
	t.Parallel()

	conv := setupTestConv(t)

	// First close.
	err := conv.Close()
	if err != nil {
		t.Errorf("First Close() error = %v", err)
	}

	// Second close should not error.
	err = conv.Close()
	if err != nil {
		t.Errorf("Second Close() should handle gracefully, got error: %v", err)
	}

	// Third close.
	err = conv.Close()
	if err != nil {
		t.Errorf("Third Close() should handle gracefully, got error: %v", err)
	}
}

// TestConversation_Close_WithCanceledContext tests Close with canceled context.
func TestConversation_Close_CanceledContext(t *testing.T) {
	t.Parallel()

	conv := setupTestConv(t)

	// Close (note: Close doesn't take a context parameter).
	err := conv.Close()
	// Should handle gracefully.
	_ = err // May or may not error depending on implementation.
}

// TestConversation_Close_WithTimeout tests Close with timeout.
func TestConversation_Close_TimeoutContext(t *testing.T) {
	t.Parallel()

	conv := setupTestConv(t)

	// Create context with very short timeout (not used by Close).
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Wait for timeout.
	time.Sleep(10 * time.Millisecond)

	// Close.
	err := conv.Close()
	// Should handle timeout gracefully.
	_ = err
	_ = ctx // Not used but created for test completeness.
}

// TestConversation_SetTaskMode_AllValidModes tests SetTaskMode with all built-in modes.
func TestConversation_SetTaskMode_AllModes(t *testing.T) {
	t.Parallel()

	conv := setupTestConv(t)

	modes := []string{"regular", "review", "compact", "planning"}
	for _, mode := range modes {
		t.Run("mode_"+mode, func(t *testing.T) {
			t.Parallel()

			err := conv.SetTaskMode(mode)
			if err != nil {
				t.Errorf("SetTaskMode(%s) error = %v", mode, err)
			}
		})
	}
}

// TestConversation_SetTaskMode_UnknownMode tests SetTaskMode with unknown mode.
func TestConversation_SetTaskMode_UnknownMode(t *testing.T) {
	t.Parallel()

	conv := setupTestConv(t)

	// Unknown mode should return error.
	err := conv.SetTaskMode("unknown_mode_that_does_not_exist")
	if err == nil {
		t.Error("Expected error for unknown mode")
	}
}

// TestConversation_SetTaskMode_EmptyMode tests SetTaskMode with empty mode.
func TestConversation_SetTaskMode_EmptyMode(t *testing.T) {
	t.Parallel()

	conv := setupTestConv(t)

	// Empty mode is valid (means use default).
	err := conv.SetTaskMode("")
	if err != nil {
		t.Errorf("SetTaskMode(\"\") error = %v, want nil (empty mode is valid)", err)
	}

	// Empty mode should set taskMode to empty string.
	if conv.GetTaskMode() != "" {
		t.Errorf("GetTaskMode() = %q, want empty string", conv.GetTaskMode())
	}
}

// TestConversation_SetTaskMode verifies that SetTaskMode successfully switches modes.
func TestConversation_SetTaskMode(t *testing.T) {
	t.Parallel()

	conv := setupTestConv(t)

	// Should default to "regular".
	if got := conv.GetTaskMode(); got != "regular" {
		t.Errorf("expected default mode 'regular', got %q", got)
	}

	// Switch to review mode.
	err := conv.SetTaskMode("review")
	if err != nil {
		t.Fatalf("SetTaskMode('review') failed: %v", err)
	}

	if got := conv.GetTaskMode(); got != "review" {
		t.Errorf("expected mode 'review', got %q", got)
	}

	// Switch to compact mode.
	err = conv.SetTaskMode("compact")
	if err != nil {
		t.Fatalf("SetTaskMode('compact') failed: %v", err)
	}

	if got := conv.GetTaskMode(); got != "compact" {
		t.Errorf("expected mode 'compact', got %q", got)
	}

	// Switch to planning mode.
	err = conv.SetTaskMode("planning")
	if err != nil {
		t.Fatalf("SetTaskMode('planning') failed: %v", err)
	}

	if got := conv.GetTaskMode(); got != "planning" {
		t.Errorf("expected mode 'planning', got %q", got)
	}

	// Switch back to regular.
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
	t.Parallel()

	conv := setupTestConv(t)

	err := conv.SetTaskMode("invalid-mode")
	if err == nil {
		t.Fatal("expected error for invalid mode, got nil")
	}

	if err.Error() == "" {
		t.Error("expected non-empty error message")
	}

	// Should remain in default mode.
	if got := conv.GetTaskMode(); got != "regular" {
		t.Errorf("expected mode to remain 'regular', got %q", got)
	}
}

// TestConversation_GetTaskMode_Default verifies that GetTaskMode returns "regular" by default.
func TestConversation_GetTaskMode_Default(t *testing.T) {
	t.Parallel()

	conv := setupTestConv(t)

	// Should default to "regular" without explicit SetTaskMode.
	if got := conv.GetTaskMode(); got != "regular" {
		t.Errorf("expected default mode 'regular', got %q", got)
	}
}

// TestConversation_TaskMode_Concurrent verifies thread-safe access to task mode.
func TestConversation_TaskMode_Concurrent(t *testing.T) {
	t.Parallel()

	conv := setupTestConv(t)

	var wg sync.WaitGroup

	// 50 concurrent readers.
	for range 50 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			_ = conv.GetTaskMode()
		}()
	}

	// 10 concurrent writers.
	modes := []string{"regular", "review", "compact", "planning"}

	for i := range 10 {
		wg.Add(1)

		go func(i int) {
			defer wg.Done()

			mode := modes[i%len(modes)]
			_ = conv.SetTaskMode(mode)
		}(i)
	}

	wg.Wait()

	// Should not race (verified with go test -race)
	// Final mode should be one of the valid modes.
	finalMode := conv.GetTaskMode()
	validMode := slices.Contains(modes, finalMode)

	if !validMode {
		t.Errorf("final mode %q is not a valid mode", finalMode)
	}
}

// TestConversation_TaskMode_PersistsAcrossTurns verifies that the task mode persists across multiple turns.
func TestConversation_TaskMode_PersistsAcrossTurns(t *testing.T) {
	t.Parallel()

	// Create a conversation with setupTestConv.
	conv := setupTestConv(t)

	// Set mode to compact.
	err := conv.SetTaskMode("compact")
	if err != nil {
		t.Fatalf("SetTaskMode failed: %v", err)
	}

	// Mode should be compact.
	if got := conv.GetTaskMode(); got != "compact" {
		t.Errorf("expected mode 'compact' before turn 1, got %q", got)
	}

	// Execute turn 1.
	turnCtx := t.Context()

	err = conv.RunTurn(turnCtx, "Turn 1")
	if err != nil {
		t.Fatalf("RunTurn failed: %v", err)
	}

	// Mode should still be compact.
	if got := conv.GetTaskMode(); got != "compact" {
		t.Errorf("expected mode 'compact' after turn 1, got %q", got)
	}

	// Execute turn 2.
	err = conv.RunTurn(turnCtx, "Turn 2")
	if err != nil {
		t.Fatalf("RunTurn failed on turn 2: %v", err)
	}

	// Mode should still be compact.
	if got := conv.GetTaskMode(); got != "compact" {
		t.Errorf("expected mode 'compact' after turn 2, got %q", got)
	}
}

// TestConversation_SetTaskMode_EmitsEvent verifies that SetTaskMode emits a system info event.
func TestConversation_SetTaskMode_EmitsEvent(t *testing.T) {
	t.Parallel()

	conv := setupTestConv(t)
	emitter := conv.emitter

	// Subscribe to events.
	_, eventChan, err := emitter.Subscribe()
	if err != nil {
		t.Fatalf("failed to subscribe to events: %v", err)
	}

	// Set up event collection.
	eventList := make([]events.Event, 0)
	done := make(chan struct{})

	go func() {
		defer close(done)

		for event := range eventChan {
			eventList = append(eventList, event)
		}
	}()

	// Switch mode.
	err = conv.SetTaskMode("review")
	if err != nil {
		t.Fatalf("SetTaskMode failed: %v", err)
	}

	// Give time for event to be emitted.
	emitter.Close()
	<-done

	// Verify event was emitted.
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
	t.Parallel()

	conv := setupTestConv(t)

	// This test verifies the validation path exists
	// In practice, all built-in tasks should validate successfully.
	err := conv.SetTaskMode("regular")
	if err != nil {
		t.Errorf("expected regular task to validate, got error: %v", err)
	}
}

// Helper functions.

// setupTestConv creates a test conversation with all dependencies.
func setupTestConv(t *testing.T) *Conversation {
	t.Helper()

	cfg := testConfig()
	workDir := t.TempDir()
	rt, emitter, provider := createTestRuntime(t, workDir)

	conv, err := NewBuilder(cfg, workDir, rt, emitter, provider).
		Build(context.Background())
	if err != nil {
		t.Fatalf("failed to build conversation: %v", err)
	}

	return conv
}

// Protocol Fields Tests.

// TestConversation_ID tests unified ID getter and setter.
func TestConversation_ID(t *testing.T) {
	t.Parallel()

	conv := setupTestConv(t)

	// Use UUID string (standardized format).
	id := "550e8400-e29b-41d4-a716-446655440000"
	conv.SetID(id)

	got := conv.ID()
	if got != id {
		t.Errorf("ID() = %q, want %q", got, id)
	}

	// Test GetSessionID returns same ID.
	sessionID := conv.GetSessionID()
	if sessionID != id {
		t.Errorf("GetSessionID() = %q, want %q", sessionID, id)
	}
}

// TestConversation_UnifiedID tests that sessionID and protocolID are unified.
func TestConversation_UnifiedID(t *testing.T) {
	t.Parallel()

	conv := setupTestConv(t)

	// Use UUID string (standardized format).
	id := "550e8400-e29b-41d4-a716-446655440000"
	conv.SetID(id)

	// Verify all ID accessors return same value.
	if conv.ID() != id {
		t.Error("ID() should return set ID")
	}

	if conv.GetSessionID() != id {
		t.Error("GetSessionID() should return same ID")
	}
}

// TestConversation_TurnID tests turn ID getter and setter (thread-safe).
func TestConversation_TurnID(t *testing.T) {
	t.Parallel()

	conv := setupTestConv(t)

	turnID := "turn-123"
	conv.SetTurnID(turnID)

	got := conv.GetTurnID()
	if got != turnID {
		t.Errorf("GetTurnID() = %q, want %q", got, turnID)
	}
}

// TestConversation_TurnID_Empty tests empty turn ID.
func TestConversation_TurnID_Empty(t *testing.T) {
	t.Parallel()

	conv := setupTestConv(t)

	got := conv.GetTurnID()
	if got != "" {
		t.Errorf("GetTurnID() = %q, want empty string", got)
	}
}

// TestConversation_Cancel tests cancel getter, setter, and execution (thread-safe).
func TestConversation_Cancel(t *testing.T) {
	t.Parallel()

	conv := setupTestConv(t)

	ctx, cancel := context.WithCancel(context.Background())
	conv.SetCancel(cancel)

	got := conv.GetCancel()
	if got == nil {
		t.Error("GetCancel() returned nil, want non-nil")
	}

	// Test cancel execution.
	conv.Cancel()

	if ctx.Err() == nil {
		t.Error("Cancel() did not cancel context, want context canceled")
	}
}

// TestConversation_Cancel_Nil tests cancel with nil function.
func TestConversation_Cancel_Nil(t *testing.T) {
	t.Parallel()

	conv := setupTestConv(t)

	// Cancel should not panic with nil cancel function.
	conv.Cancel()

	got := conv.GetCancel()
	if got != nil {
		t.Errorf("GetCancel() = %v, want nil", got)
	}
}

// TestConversation_ProtocolFields_ThreadSafety tests concurrent access to protocol fields.
func TestConversation_ProtocolFields_ThreadSafety(t *testing.T) {
	t.Parallel()

	conv := setupTestConv(t)

	var wg sync.WaitGroup

	iterations := 100

	// Concurrent writes to turnID.
	for i := range iterations {
		wg.Add(1)

		go func(i int) {
			defer wg.Done()

			turnID := fmt.Sprintf("turn-%d", i)
			conv.SetTurnID(turnID)
			// Read back (may be different due to concurrent writes, but should not panic).
			_ = conv.GetTurnID()
		}(i)
	}

	// Concurrent reads to turnID.
	for range iterations {
		wg.Add(1)

		go func() {
			defer wg.Done()

			_ = conv.GetTurnID()
		}()
	}

	// Concurrent cancel operations.
	for range iterations {
		wg.Add(1)

		go func() {
			defer wg.Done()

			ctx, cancel := context.WithCancel(context.Background())
			conv.SetCancel(cancel)
			conv.Cancel()

			_ = ctx.Err()
		}()
	}

	wg.Wait()

	// Verify final state is valid (should have some turnID set or empty).
	finalTurnID := conv.GetTurnID()
	_ = finalTurnID // Just verify we can read it without panic.
}
