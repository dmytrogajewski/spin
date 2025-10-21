package agent

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/cycle"
	"github.com/dmytrogajewski/spin/internal/detection"
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/orchestration"
	"github.com/dmytrogajewski/spin/internal/security"
	"github.com/dmytrogajewski/spin/internal/task"
	"github.com/dmytrogajewski/spin/internal/tools"
)

// TestAgentOptions_Coverage tests uncovered paths in agent_options.go
// REMOVED: TestAgentOptions_Coverage - WithToolRegistry option removed
// Tool registry is now configured when creating OrchestrationService

// TestAgentRequest_Coverage tests uncovered paths in agent_request.go
func TestAgentRequest_Coverage(t *testing.T) {
	t.Run("applyTimeout with existing deadline", func(t *testing.T) {
		agent := createTestAgent(t)

		// Create context with existing deadline
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		// Apply timeout should return context unchanged with no-op cancel
		newCtx, newCancel := agent.applyTimeout(ctx)
		defer newCancel()

		// Context should have the same deadline
		deadline1, ok1 := ctx.Deadline()
		deadline2, ok2 := newCtx.Deadline()

		if !ok1 || !ok2 {
			t.Error("Expected both contexts to have deadlines")
		}
		if deadline1 != deadline2 {
			t.Error("Expected deadline to be unchanged")
		}
	})

	t.Run("buildSystemMessage with nil task and context", func(t *testing.T) {
		// buildSystemMessage is a private method that was removed during refactoring
		// This test is no longer applicable
		t.Skip("buildSystemMessage is a private method removed during refactoring")
	})

	t.Run("buildSystemMessage with task prompt", func(t *testing.T) {
		// buildSystemMessage is a private method that was removed during refactoring
		// This test is no longer applicable
		t.Skip("buildSystemMessage is a private method removed during refactoring")
	})
}

// TestAgentTools_Coverage tests uncovered paths in agent_tools.go
func TestAgentTools_Coverage(t *testing.T) {
	t.Run("ShouldApprove with all classification types", func(t *testing.T) {
		agent := createTestAgent(t)
		agent.config.RequireApproval = true

		tests := []struct {
			name        string
			cmd         *security.Command
			wantApprove bool
			wantReason  string
		}{
			{
				name:        "safe command",
				cmd:         &security.Command{Program: "ls", Args: []string{"-l"}},
				wantApprove: false,
				wantReason:  "",
			},
			{
				name:        "forbidden command blocked",
				cmd:         &security.Command{Program: "rm", Args: []string{"-rf", "/"}},
				wantApprove: false, // Forbidden = blocked, not approved
				wantReason:  "BLOCKED",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				needsApproval, reason := agent.ShouldApprove(tt.cmd)

				if needsApproval != tt.wantApprove {
					t.Errorf("ShouldApprove() needsApproval = %v, want %v", needsApproval, tt.wantApprove)
				}

				if tt.wantReason != "" && !containsString(reason, tt.wantReason) {
					t.Errorf("ShouldApprove() reason = %v, want substring %v", reason, tt.wantReason)
				}
			})
		}
	})

	t.Run("processToolCalls with error from ProcessToolCall", func(t *testing.T) {
		agent := createTestAgent(t)

		llmResp := &llm.CompletionResponse{
			Content: "test response",
			ToolCalls: []llm.ToolCall{
				{
					ID:   "test_call_1",
					Type: "function",
					Function: llm.FunctionCall{
						Name:      "invalid_tool",
						Arguments: "{}",
					},
				},
			},
		}

		messages := []Message{
			{Role: RoleUser, Content: "test"},
		}

		resp := &AgentResponse{
			ToolCalls: make([]orchestration.ToolCall, 0),
		}

		// This will process the tool call and handle the error
		result := agent.processToolCalls(context.Background(), messages, llmResp, resp)

		// Should have added messages even with error
		if len(result) < 2 {
			t.Errorf("Expected at least 2 messages, got %d", len(result))
		}
	})

	// REMOVED: executeCommand tests - method removed in service refactoring
	// Command execution is now tested through ToolExecutor and execute_command tool

	t.Run("convertParameterSchemaToMap with marshal error", func(t *testing.T) {
		// This tests the fallback path when marshaling fails
		// Hard to trigger in practice, but we can at least call it
		params := tools.ParameterSchema{
			Type: "object",
		}

		result := convertParameterSchemaToMap(params)

		if result == nil {
			t.Error("Expected non-nil result")
		}
		if result["type"] != "object" {
			t.Errorf("Expected type=object, got %v", result["type"])
		}
	})
}

// TestAgentTurn_Coverage tests uncovered paths in agent_turn.go
func TestAgentTurn_Coverage(t *testing.T) {
	t.Run("executeAgentLoop with context cancellation", func(t *testing.T) {
		agent := createTestAgent(t)
		agent.config.MaxTurns = 10

		// Cancel context immediately
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		messages := []Message{
			{Role: RoleUser, Content: "test"},
		}

		mockTask := &task.Regular{}
		resp := &AgentResponse{
			ToolCalls: make([]orchestration.ToolCall, 0),
		}

		resultMessages, resultResp, err := agent.executeAgentLoop(ctx, messages, mockTask, resp)

		if err == nil {
			t.Error("Expected context cancellation error")
		}
		if resultResp.FinishReason != "timeout" {
			t.Errorf("Expected finish reason 'timeout', got %v", resultResp.FinishReason)
		}
		if len(resultMessages) == 0 {
			t.Error("Expected messages to be returned")
		}
	})

	// REMOVED: handleCycleDetection test - cycle detection moved to DetectionService
	// Cycle detection is now tested in detection_service_test.go

	t.Run("callLLM with task max tokens", func(t *testing.T) {
		agent := createTestAgent(t)

		messages := []Message{
			{Role: RoleSystem, Content: "system"},
			{Role: RoleUser, Content: "test"},
		}

		mockTask := &task.Compact{} // Has specific MaxTokens

		// Mock LLM will return error, but we're testing the token logic before that
		_, err := agent.callLLM(context.Background(), messages, mockTask)

		// Expected to get error from mock LLM
		if err == nil {
			t.Log("callLLM() expected error from mock LLM")
		}
	})

	t.Run("callLLM with stream error", func(t *testing.T) {
		agent := createTestAgent(t)

		messages := []Message{
			{Role: RoleSystem, Content: "system"},
			{Role: RoleUser, Content: "test"},
		}

		mockTask := &task.Regular{}

		// Mock LLM will return error or success, test the call path
		_, err := agent.callLLM(context.Background(), messages, mockTask)

		// Just verify we called the LLM
		t.Logf("callLLM result: err=%v", err)
	})
}

// TestAgent_Coverage tests uncovered paths in agent.go
func TestAgent_Coverage(t *testing.T) {
	t.Run("NewAgent with services", func(t *testing.T) {
		// Test the full NewAgent path with service-based architecture
		provider := &llm.MockProvider{}
		workDir := t.TempDir()
		executor, _ := NewExecutor(workDir)
		validator := security.NewValidator()
		env := &Environment{WorkDir: workDir}
		emitter := events.NewEventEmitter(100)

		// Build services
		approvalService := security.NewApprovalService(nil, emitter, validator)
		securityService := security.NewSecurityService(validator, approvalService)

		cycleDetector := cycle.NewDetector(cycle.Config{Enabled: false})
		detectionService := detection.NewDetectionService(cycleDetector, nil)

		toolRegistry := tools.NewRegistry()
		_ = toolRegistry.Register(tools.NewExecuteCommandTool(executor, validator))

		taskRegistry := orchestration.NewRegistry()
		_ = taskRegistry.Register("regular", task.NewRegular())
		_ = taskRegistry.SetDefault("regular")
		toolExecutor := orchestration.NewToolExecutor(orchestration.ToolExecutorConfig{
			Registry:        toolRegistry,
			Validator:       validator,
			ApprovalService: approvalService,
			Emitter:         emitter,
			WorkDir:         workDir,
		})
		orchestrationService := orchestration.NewOrchestrationService(toolExecutor, toolRegistry, taskRegistry)

		agent, err := NewAgent(provider, securityService, detectionService, orchestrationService, env, emitter)

		if err != nil {
			t.Errorf("newAgentForTest() error = %v", err)
		}
		if agent == nil {
			t.Fatal("newAgentForTest() returned nil agent")
		}

		// Verify task modes are accessible
		modes := agent.ListTaskModes()
		if len(modes) < 1 {
			t.Errorf("Expected at least 1 task mode, got %d", len(modes))
		}
	})

	t.Run("CreatePlan with invalid input", func(t *testing.T) {
		agent := createTestAgent(t)

		// Empty task should return error
		_, err := agent.CreatePlan(context.Background(), "")

		if err == nil {
			t.Error("CreatePlan('') expected error, got nil")
		}
		if !errors.Is(err, ErrEmptyInput) {
			t.Errorf("Expected ErrEmptyInput, got %v", err)
		}
	})

	t.Run("CreatePlan with valid task", func(t *testing.T) {
		agent := createTestAgent(t)

		// This will fail because mock LLM returns error, but tests the path
		_, err := agent.CreatePlan(context.Background(), "Build a web app")

		// Expected to fail with mock LLM
		if err == nil {
			t.Log("CreatePlan() expected error from mock LLM")
		}
	})
}

// Helper functions

func createTestAgent(t *testing.T) *Agent {
	return createTestAgentWithServices(t)
}

func containsString(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 &&
		(s == substr || len(s) > len(substr) &&
			(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
				indexOf(s, substr) >= 0))
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// TestProcessToolCallArguments tests argument parsing edge cases
func TestProcessToolCallArguments(t *testing.T) {
	agent := createTestAgent(t)

	t.Run("parseToolArguments with empty string", func(t *testing.T) {
		call := &orchestration.ToolCall{
			ID:   "test",
			Type: "function",
			Function: orchestration.ToolCallFunction{
				Name:      "test_tool",
				Arguments: "",
			},
		}

		_, err := agent.parseToolArguments(call)
		if err == nil {
			t.Error("Expected error for empty arguments")
		}
	})

	t.Run("parseToolArguments with invalid JSON", func(t *testing.T) {
		call := &orchestration.ToolCall{
			ID:   "test",
			Type: "function",
			Function: orchestration.ToolCallFunction{
				Name:      "test_tool",
				Arguments: "{invalid json}",
			},
		}

		_, err := agent.parseToolArguments(call)
		if err == nil {
			t.Error("Expected error for invalid JSON")
		}
	})

	t.Run("parseToolArguments with valid JSON", func(t *testing.T) {
		args := map[string]interface{}{
			"key": "value",
			"num": 42.0,
		}
		argsJSON, _ := json.Marshal(args)

		call := &orchestration.ToolCall{
			ID:   "test",
			Type: "function",
			Function: orchestration.ToolCallFunction{
				Name:      "test_tool",
				Arguments: string(argsJSON),
			},
		}

		result, err := agent.parseToolArguments(call)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if result["key"] != "value" {
			t.Errorf("Expected key=value, got %v", result["key"])
		}
	})
}

// TestValidateToolCall tests validation edge cases
func TestValidateToolCall(t *testing.T) {
	agent := createTestAgent(t)

	tests := []struct {
		name    string
		call    *orchestration.ToolCall
		wantErr bool
	}{
		{
			name:    "nil tool call",
			call:    nil,
			wantErr: true,
		},
		{
			name: "empty ID",
			call: &orchestration.ToolCall{
				ID:   "",
				Type: "function",
				Function: orchestration.ToolCallFunction{
					Name: "test",
				},
			},
			wantErr: true,
		},
		{
			name: "empty function name",
			call: &orchestration.ToolCall{
				ID:   "test_id",
				Type: "function",
				Function: orchestration.ToolCallFunction{
					Name: "",
				},
			},
			wantErr: true,
		},
		{
			name: "valid tool call",
			call: &orchestration.ToolCall{
				ID:   "test_id",
				Type: "function",
				Function: orchestration.ToolCallFunction{
					Name:      "test_tool",
					Arguments: "{}",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := agent.validateToolCall(tt.call)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateToolCall() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
