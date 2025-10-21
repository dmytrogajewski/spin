package conversation

import (
	"context"
	"sync"
	"testing"

	"github.com/dmytrogajewski/spin/internal/events"
)

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

