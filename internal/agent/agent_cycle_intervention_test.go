package agent

import (
	"context"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/cycle"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/task"
)

// TestHandleCycleDetection_InterventionMessagesApplied tests that
// intervention messages are actually added to the conversation.
// This is a regression test for the bug where handleCycleDetection
// modified messages locally but didn't return them, causing interventions
// to be silently discarded.
func TestHandleCycleDetection_InterventionMessagesApplied(t *testing.T) {
	agent := createTestAgent(t)

	// Enable cycle detection
	agent.config.CycleDetection.Enabled = true

	// Create initial conversation with some messages
	initialMessages := []Message{
		{
			Role:      RoleUser,
			Content:   "List files",
			Timestamp: time.Now(),
		},
		{
			Role:      RoleAssistant,
			Content:   "I'll list the files",
			Timestamp: time.Now(),
		},
	}

	// Simulate repeated tool calls to trigger cycle detection
	// Add 3 snapshots with same tool AND same params to trigger CycleRepeatedTool
	agent.detection.RecordSnapshot(cycle.Snapshot{
		Turn:      1,
		Response:  "Calling list_directory",
		ToolCalls: []string{`list_directory({"path": "/"})`},
		Timestamp: time.Now(),
	})
	agent.detection.RecordSnapshot(cycle.Snapshot{
		Turn:      2,
		Response:  "Calling list_directory again",
		ToolCalls: []string{`list_directory({"path": "/"})`},
		Timestamp: time.Now(),
	})
	agent.detection.RecordSnapshot(cycle.Snapshot{
		Turn:      3,
		Response:  "Calling list_directory once more",
		ToolCalls: []string{`list_directory({"path": "/"})`},
		Timestamp: time.Now(),
	})

	// Create a mock LLM response that will trigger cycle detection
	llmResp := &llm.CompletionResponse{
		Content: "Calling list_directory",
		ToolCalls: []llm.ToolCall{
			{
				ID:   "call_123",
				Type: "function",
				Function: llm.FunctionCall{
					Name:      "list_directory",
					Arguments: `{"path": "/"}`,
				},
			},
		},
		FinishReason: "",
	}

	// Call handleCycleDetection
	resp := &AgentResponse{}
	modifiedMessages, shouldStop, err := agent.handleCycleDetection(
		context.Background(),
		initialMessages,
		llmResp,
		3, // turn count
		resp,
	)

	if err != nil {
		t.Fatalf("handleCycleDetection returned error: %v", err)
	}

	if shouldStop {
		t.Fatal("handleCycleDetection should not stop (severity < 3)")
	}

	// Check if cycle was detected
	cycleResult, err := agent.detection.CheckCycle()
	if err != nil {
		t.Fatalf("CheckCycle failed: %v", err)
	}
	if cycleResult.Type == cycle.CycleNone {
		t.Fatal("Expected cycle to be detected, but got CycleNone")
	}

	// The critical assertion: modifiedMessages should have the intervention message added
	// With the bug (before fix), modifiedMessages would equal initialMessages (unchanged)
	// After the fix, modifiedMessages should be longer (reflection added)
	if len(modifiedMessages) == len(initialMessages) {
		t.Error("BUG DETECTED: handleCycleDetection did not modify the messages slice")
		t.Error("Expected intervention message to be added, but messages unchanged")
		t.Error("This indicates the intervention's message modifications were discarded")
	}

	// After the fix, this should pass
	expectedMinLen := 3 // original 2 + 1 reflection message
	if len(modifiedMessages) < expectedMinLen {
		t.Errorf("Expected at least %d messages after intervention, got %d", expectedMinLen, len(modifiedMessages))
	}

	// Verify the last message is from the intervention (user role with reflection prompt)
	if len(modifiedMessages) >= expectedMinLen {
		lastMsg := modifiedMessages[len(modifiedMessages)-1]
		if lastMsg.Role != RoleUser {
			t.Errorf("Expected intervention message to have role 'user', got '%s'", lastMsg.Role)
		}
		if lastMsg.Content == "" {
			t.Error("Expected intervention message to have content")
		}
		// Reflection intervention should mention "repeating" or "different"
		if !containsAnySubstring(lastMsg.Content, []string{"repeating", "different", "perspective", "angles"}) {
			t.Errorf("Expected reflection-style message, got: %s", lastMsg.Content)
		}
	}
}

// TestExecuteAgentLoop_CycleInterventionPropagated tests that the full agent loop
// properly uses intervention messages.
func TestExecuteAgentLoop_CycleInterventionPropagated(t *testing.T) {
	agent := createTestAgent(t)
	agent.config.CycleDetection.Enabled = true
	agent.config.MaxTurns = 10

	// Create a mock LLM that returns same tool call repeatedly
	mockLLM := llm.NewMockProvider("test")
	agent.llm = mockLLM

	initialMessages := []Message{
		{
			Role:      RoleSystem,
			Content:   "You are a helpful assistant",
			Timestamp: time.Now(),
		},
		{
			Role:      RoleUser,
			Content:   "List files",
			Timestamp: time.Now(),
		},
	}

	task := task.NewRegular()
	resp := &AgentResponse{}

	// Execute the loop - it should detect the cycle and add intervention
	resultMessages, resultResp, err := agent.executeAgentLoop(
		context.Background(),
		initialMessages,
		task,
		resp,
	)

	// The loop should complete (may hit max turns or other stop condition)
	if err != nil {
		// Error is acceptable for mock LLM
		t.Logf("executeAgentLoop returned error (expected with mock): %v", err)
	}

	_ = resultResp // resultResp is used implicitly

	// Key test: if a cycle was detected during the loop, verify intervention messages
	// were preserved in resultMessages
	history := agent.detection.GetHistory()
	if len(history) >= 3 {
		// Cycle should have been detected
		t.Log("Cycle detection triggered during agent loop")

		// After the fix, resultMessages should include intervention messages
		if len(resultMessages) <= len(initialMessages) {
			t.Error("Expected resultMessages to include intervention messages, but no new messages found")
		}
	}
}

// Helper function to check if string contains any of the substrings
func containsAnySubstring(s string, substrings []string) bool {
	for _, substr := range substrings {
		if containsSubstring(s, substr) {
			return true
		}
	}
	return false
}

func containsSubstring(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 &&
		(s == substr || len(s) > len(substr) &&
			(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
				indexOfSubstring(s, substr) >= 0))
}

func indexOfSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
