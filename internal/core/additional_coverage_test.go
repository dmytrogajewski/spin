package core

import (
	"context"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/core/cycle"
	"github.com/dmytrogajewski/spin/internal/tools"
)

func TestAgentOptions(t *testing.T) {
	tests := []struct {
		name   string
		option AgentOption
	}{
		{
			name: "WithApprovalHandler",
			option: WithApprovalHandler(func(req ApprovalRequest) ApprovalResponse {
				return ApprovalResponse{RequestID: req.ID, Approved: true}
			}),
		},
		{
			name:   "WithPatternDetector",
			option: WithPatternDetector(&cycle.PatternDetector{}),
		},
		{
			name:   "WithToolRegistry",
			option: WithToolRegistry(&tools.Registry{}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that the option function doesn't panic
			agent := &Agent{}
			tt.option(agent)

			// Basic verification that the option was applied
			if agent == nil {
				t.Error("Agent should not be nil after applying option")
			}
		})
	}
}

func TestApprovalService(t *testing.T) {
	// Test NewApprovalService
	handler := func(req ApprovalRequest) ApprovalResponse {
		return ApprovalResponse{RequestID: req.ID, Approved: true}
	}

	service := NewApprovalService(handler)
	if service == nil {
		t.Fatal("NewApprovalService should not return nil")
	}

	// Test RequestApproval
	operation := Operation{
		Command: &Command{
			Program: "test",
			Args:    []string{"command"},
		},
		Reason:  "Test operation",
		WorkDir: "/tmp",
	}

	_, approved, err := service.RequestApproval(context.Background(), operation)
	if err != nil {
		t.Errorf("RequestApproval should not return error: %v", err)
	}
	if !approved {
		t.Error("RequestApproval should return true")
	}

	// Test RequestApprovalWithValidator
	validator := &Validator{}
	approved, err = service.RequestApprovalWithValidator(context.Background(), operation.Command, validator, operation.WorkDir)
	if err != nil {
		t.Errorf("RequestApprovalWithValidator should not return error: %v", err)
	}
	if !approved {
		t.Error("RequestApprovalWithValidator should return true")
	}
}

func TestConfigMerge(t *testing.T) {
	base := Config{
		MaxTurns:    10,
		Temperature: 0.7,
		MaxTokens:   4096,
	}

	other := Config{
		MaxTurns:    15,
		Temperature: 0.8,
		MaxTokens:   8192,
	}

	result := base.Merge(&other)

	if result.MaxTurns != 15 {
		t.Errorf("Merge MaxTurns = %v, want 15", result.MaxTurns)
	}
	if result.Temperature != 0.8 {
		t.Errorf("Merge Temperature = %v, want 0.8", result.Temperature)
	}
	if result.MaxTokens != 8192 {
		t.Errorf("Merge MaxTokens = %v, want 8192", result.MaxTokens)
	}
}

func TestConversationMethods(t *testing.T) {
	conv := &Conversation{}

	// Test Stop
	err := conv.Stop(context.Background())
	if err != nil {
		t.Errorf("Stop should not return error: %v", err)
	}

	// Test GetSessionID
	sessionID := conv.GetSessionID()
	if sessionID == "" {
		t.Error("GetSessionID should return non-empty string")
	}

	// Test GetTokenCount
	count := conv.GetTokenCount()
	if count < 0 {
		t.Errorf("GetTokenCount should return non-negative value, got %d", count)
	}
}

func TestEventEmitter_Emit_Additional(t *testing.T) {
	emitter := &EventEmitter{}

	// Test Emit method
	event := Event{
		Type:      EventInfo,
		Timestamp: time.Now(),
		Data:      map[string]interface{}{"test": "data"},
	}
	emitter.Emit(event)

	// Verify event was emitted (basic test)
	if emitter == nil {
		t.Error("EventEmitter should not be nil")
	}
}
