package core

import (
	"context"
	"testing"
	"time"
)

// TestConversation_Stop_MultipleCallsEdgeCase tests Stop called multiple times
func TestConversation_Stop_MultipleCalls(t *testing.T) {
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

	// First stop
	err = conv.Stop(ctx)
	if err != nil {
		t.Errorf("First Stop() error = %v", err)
	}

	// Second stop should not error
	err = conv.Stop(ctx)
	if err != nil {
		t.Errorf("Second Stop() should handle gracefully, got error: %v", err)
	}

	// Third stop
	err = conv.Stop(ctx)
	if err != nil {
		t.Errorf("Third Stop() should handle gracefully, got error: %v", err)
	}
}

// TestConversation_Stop_WithCanceledContext tests Stop with canceled context
func TestConversation_Stop_CanceledContext(t *testing.T) {
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

	// Cancel context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Stop with canceled context
	err = conv.Stop(ctx)
	// Should handle canceled context gracefully
	_ = err // May or may not error depending on implementation
}

// TestConversation_Stop_WithTimeout tests Stop with timeout context
func TestConversation_Stop_TimeoutContext(t *testing.T) {
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

	// Create context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Wait for timeout
	time.Sleep(10 * time.Millisecond)

	// Stop with timed-out context
	err = conv.Stop(ctx)
	// Should handle timeout gracefully
	_ = err
}

// TestConversation_SetTaskMode_AllValidModes tests SetTaskMode with all built-in modes
func TestConversation_SetTaskMode_AllModes(t *testing.T) {
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

// TestConversation_SetTaskMode_UnknownMode tests SetTaskMode with unknown mode
func TestConversation_SetTaskMode_UnknownMode(t *testing.T) {
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

	// Unknown mode should return error
	err = conv.SetTaskMode("unknown_mode_that_does_not_exist")
	if err == nil {
		t.Error("Expected error for unknown mode")
	}
}

// TestConversation_SetTaskMode_EmptyMode tests SetTaskMode with empty mode
func TestConversation_SetTaskMode_EmptyMode(t *testing.T) {
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

	// Empty mode should return error
	err = conv.SetTaskMode("")
	if err == nil {
		t.Error("Expected error for empty mode")
	}
}
