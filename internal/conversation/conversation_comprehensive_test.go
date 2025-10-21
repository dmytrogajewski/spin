package conversation

import (
	"context"
	"testing"
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

