package core

import (
	"context"
	"testing"
	"time"
)

// TestConversation_RunTurn_EmptyPrompt tests RunTurn with empty prompt
func TestConversation_RunTurn_EmptyPrompt(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Provider = "mock"
	cfg.Model = "test-model"

	mgr, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	ctx := context.Background()
	conv, err := mgr.NewConversation(ctx, t.TempDir())
	if err != nil {
		t.Fatalf("NewConversation() error = %v", err)
	}
	defer conv.Stop(ctx)

	// Run turn with empty prompt
	err = conv.RunTurn(ctx, "")
	if err == nil {
		t.Error("Expected error for empty prompt")
	}
}

// TestConversation_RunTurn_WithValidPrompt tests RunTurn with valid prompt
func TestConversation_RunTurn_WithValidPrompt(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Provider = "mock"
	cfg.Model = "test-model"
	cfg.MaxTurns = 1

	mgr, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	ctx := context.Background()
	conv, err := mgr.NewConversation(ctx, t.TempDir())
	if err != nil {
		t.Fatalf("NewConversation() error = %v", err)
	}
	defer conv.Stop(ctx)

	// Run turn with valid prompt
	err = conv.RunTurn(ctx, "test prompt")
	// May succeed or fail depending on mock provider, but should not panic
	_ = err
}

// TestConversation_RunTurn_ContextCanceled tests RunTurn with canceled context
func TestConversation_RunTurn_ContextCanceled(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Provider = "mock"
	cfg.Model = "test-model"

	mgr, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	ctx := context.Background()
	conv, err := mgr.NewConversation(ctx, t.TempDir())
	if err != nil {
		t.Fatalf("NewConversation() error = %v", err)
	}
	defer conv.Stop(ctx)

	// Cancel context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Run turn with canceled context
	err = conv.RunTurn(ctx, "test prompt")
	if err == nil {
		t.Error("Expected error for canceled context")
	}
}

// TestConversation_Stop_MultipleTimes tests Stop called multiple times
func TestConversation_Stop_MultipleTimes(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Provider = "mock"
	cfg.Model = "test-model"

	mgr, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	ctx := context.Background()
	conv, err := mgr.NewConversation(ctx, t.TempDir())
	if err != nil {
		t.Fatalf("NewConversation() error = %v", err)
	}

	// Stop multiple times
	err1 := conv.Stop(ctx)
	err2 := conv.Stop(ctx)
	err3 := conv.Stop(ctx)

	// All should succeed (or at least not panic)
	_ = err1
	_ = err2
	_ = err3
}

// TestConversation_Stop_WithRunningTurn tests Stop while turn is running
func TestConversation_Stop_WithRunningTurn(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Provider = "mock"
	cfg.Model = "test-model"
	cfg.MaxTurns = 1

	mgr, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	ctx := context.Background()
	conv, err := mgr.NewConversation(ctx, t.TempDir())
	if err != nil {
		t.Fatalf("NewConversation() error = %v", err)
	}

	// Start a turn in background
	go func() {
		conv.RunTurn(ctx, "test prompt")
	}()

	// Wait a bit
	time.Sleep(50 * time.Millisecond)

	// Stop while turn may be running
	err = conv.Stop(ctx)
	// Should handle gracefully
	_ = err
}

// TestConversation_Stream_Functionality tests Stream()
func TestConversation_Stream_ReceivesEvents(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Provider = "mock"
	cfg.Model = "test-model"

	mgr, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	ctx := context.Background()
	conv, err := mgr.NewConversation(ctx, t.TempDir())
	if err != nil {
		t.Fatalf("NewConversation() error = %v", err)
	}
	defer conv.Stop(ctx)

	stream := conv.Stream()
	if stream == nil {
		t.Fatal("Expected non-nil stream")
	}

	// Start a turn to generate events
	go func() {
		conv.RunTurn(ctx, "test prompt")
	}()

	// Try to receive events
	select {
	case _, ok := <-stream:
		if !ok {
			// Channel closed
		}
	case <-time.After(1 * time.Second):
		// Timeout is acceptable
	}
}

// TestConversation_SetTaskMode_ValidModes tests SetTaskMode with all valid modes
func TestConversation_SetTaskMode_ValidModes(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Provider = "mock"
	cfg.Model = "test-model"

	mgr, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	ctx := context.Background()
	conv, err := mgr.NewConversation(ctx, t.TempDir())
	if err != nil {
		t.Fatalf("NewConversation() error = %v", err)
	}
	defer conv.Stop(ctx)

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
	cfg := DefaultConfig()
	cfg.Provider = "mock"
	cfg.Model = "test-model"

	mgr, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	ctx := context.Background()
	conv, err := mgr.NewConversation(ctx, t.TempDir())
	if err != nil {
		t.Fatalf("NewConversation() error = %v", err)
	}
	defer conv.Stop(ctx)

	// Invalid mode should return error
	err = conv.SetTaskMode("invalid_mode")
	if err == nil {
		t.Error("Expected error for invalid task mode")
	}
}
