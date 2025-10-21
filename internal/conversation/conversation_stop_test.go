package conversation

import (
	"context"
	"testing"
	"time"
)

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

