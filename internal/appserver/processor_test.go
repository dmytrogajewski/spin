package appserver

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/dmytrogajewski/spin/internal/protocol/jsonrpc"
)

func TestProcessor_HandleInitialize(t *testing.T) {
	tmpDir := t.TempDir()
	processor, err := NewProcessor(ProcessorConfig{
		WorkspacePath: tmpDir,
		Version:       "0.1.0",
	})
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	params := jsonrpc.InitializeParams{
		WorkspacePath: "/tmp/workspace",
		Config:        map[string]interface{}{"key": "value"},
	}

	result, err := processor.HandleInitialize(context.Background(), params)
	if err != nil {
		t.Fatalf("HandleInitialize failed: %v", err)
	}

	if result.Status != "ok" {
		t.Errorf("Expected status 'ok', got '%s'", result.Status)
	}
	if result.Version != "0.1.0" {
		t.Errorf("Expected version '0.1.0', got '%s'", result.Version)
	}
}

func TestProcessor_HandleSendMessage_NewConversation(t *testing.T) {
	tmpDir := t.TempDir()
	processor, err := NewProcessor(ProcessorConfig{WorkspacePath: tmpDir, Version: "0.1.0"})
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	// Set output to capture notifications
	output := &bytes.Buffer{}
	processor.SetOutput(output)

	params := jsonrpc.SendMessageParams{
		ConversationID: nil, // New conversation
		Message:        "Hello",
	}

	result, err := processor.HandleSendMessage(context.Background(), params)
	if err != nil {
		t.Fatalf("HandleSendMessage failed: %v", err)
	}

	if result.ConversationID == "" {
		t.Error("Expected non-empty conversation ID")
	}
	if result.TurnID == "" {
		t.Error("Expected non-empty turn ID")
	}

	// Verify conversation was created
	processor.mu.Lock()
	conv, exists := processor.conversations[result.ConversationID]
	processor.mu.Unlock()

	if !exists {
		t.Error("Conversation should have been created")
	}
	if conv.TurnID != result.TurnID {
		t.Error("Turn ID mismatch")
	}
}

func TestProcessor_HandleSendMessage_ExistingConversation(t *testing.T) {
	tmpDir := t.TempDir()
	processor, err := NewProcessor(ProcessorConfig{WorkspacePath: tmpDir, Version: "0.1.0"})
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	// Create first conversation
	params1 := jsonrpc.SendMessageParams{
		ConversationID: nil,
		Message:        "First message",
	}

	result1, err := processor.HandleSendMessage(context.Background(), params1)
	if err != nil {
		t.Fatalf("First HandleSendMessage failed: %v", err)
	}

	// Send message to existing conversation
	params2 := jsonrpc.SendMessageParams{
		ConversationID: &result1.ConversationID,
		Message:        "Second message",
	}

	result2, err := processor.HandleSendMessage(context.Background(), params2)
	if err != nil {
		t.Fatalf("Second HandleSendMessage failed: %v", err)
	}

	if result2.ConversationID != result1.ConversationID {
		t.Error("Conversation ID should match")
	}
	if result2.TurnID == result1.TurnID {
		t.Error("Turn IDs should be different")
	}
}

func TestProcessor_HandleSendMessage_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	processor, err := NewProcessor(ProcessorConfig{WorkspacePath: tmpDir, Version: "0.1.0"})
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	nonexistentID := "nonexistent-conv-id"
	params := jsonrpc.SendMessageParams{
		ConversationID: &nonexistentID,
		Message:        "Hello",
	}

	_, err = processor.HandleSendMessage(context.Background(), params)
	if err == nil {
		t.Error("Expected error for nonexistent conversation")
	}

	// Check error type
	rpcErr, ok := err.(*jsonrpc.Error)
	if !ok {
		t.Error("Expected JSON-RPC error")
	}
	if rpcErr.Code != jsonrpc.ConversationNotFound {
		t.Errorf("Expected code %d, got %d", jsonrpc.ConversationNotFound, rpcErr.Code)
	}
}

func TestProcessor_HandleApproveTool(t *testing.T) {
	tmpDir := t.TempDir()
	processor, err := NewProcessor(ProcessorConfig{WorkspacePath: tmpDir, Version: "0.1.0"})
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	params := jsonrpc.ApproveToolParams{
		ToolCallID: "call-123",
		Approved:   true,
	}

	result, err := processor.HandleApproveTool(context.Background(), params)
	if err != nil {
		t.Fatalf("HandleApproveTool failed: %v", err)
	}

	if result.Status != "ok" {
		t.Errorf("Expected status 'ok', got '%s'", result.Status)
	}
}

func TestProcessor_HandleCancelTurn(t *testing.T) {
	tmpDir := t.TempDir()
	processor, err := NewProcessor(ProcessorConfig{WorkspacePath: tmpDir, Version: "0.1.0"})
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	// Create a conversation with a turn
	params := jsonrpc.SendMessageParams{
		Message: "Test message",
	}

	result, err := processor.HandleSendMessage(context.Background(), params)
	if err != nil {
		t.Fatalf("HandleSendMessage failed: %v", err)
	}

	// Cancel the turn
	cancelParams := jsonrpc.CancelTurnParams{
		TurnID: result.TurnID,
	}

	cancelResult, err := processor.HandleCancelTurn(context.Background(), cancelParams)
	if err != nil {
		t.Fatalf("HandleCancelTurn failed: %v", err)
	}

	if cancelResult.Status != "ok" {
		t.Errorf("Expected status 'ok', got '%s'", cancelResult.Status)
	}
}

func TestProcessor_HandleCancelTurn_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	processor, err := NewProcessor(ProcessorConfig{WorkspacePath: tmpDir, Version: "0.1.0"})
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	params := jsonrpc.CancelTurnParams{
		TurnID: "nonexistent-turn",
	}

	_, err = processor.HandleCancelTurn(context.Background(), params)
	if err == nil {
		t.Error("Expected error for nonexistent turn")
	}
}

func TestProcessor_SendNotification(t *testing.T) {
	tmpDir := t.TempDir()
	processor, err := NewProcessor(ProcessorConfig{WorkspacePath: tmpDir, Version: "0.1.0"})
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	output := &bytes.Buffer{}
	processor.SetOutput(output)

	params := map[string]string{"key": "value"}
	processor.sendNotification("test_method", params)

	var notif jsonrpc.Notification
	if err := json.NewDecoder(output).Decode(&notif); err != nil {
		t.Fatalf("Failed to decode notification: %v", err)
	}

	if notif.JSONRPC != "2.0" {
		t.Errorf("Expected jsonrpc '2.0', got '%s'", notif.JSONRPC)
	}
	if notif.Method != "test_method" {
		t.Errorf("Expected method 'test_method', got '%s'", notif.Method)
	}
}

func TestProcessor_SendNotification_NoOutput(t *testing.T) {
	tmpDir := t.TempDir()
	processor, err := NewProcessor(ProcessorConfig{WorkspacePath: tmpDir, Version: "0.1.0"})
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}
	// Don't set output

	// Should not panic
	processor.sendNotification("test_method", map[string]string{"key": "value"})
}

func TestGenerateTurnID(t *testing.T) {
	id1 := generateTurnID()
	id2 := generateTurnID()

	if id1 == id2 {
		t.Error("generateTurnID should generate unique IDs")
	}

	if len(id1) == 0 {
		t.Error("generateTurnID should not return empty string")
	}

	// Should start with "turn-"
	if len(id1) < 5 || id1[:5] != "turn-" {
		t.Errorf("Turn ID should start with 'turn-', got '%s'", id1)
	}
}

// Task Mode Tests

func TestProcessor_NewConversationWithTaskMode(t *testing.T) {
	tmpDir := t.TempDir()
	processor, err := NewProcessor(ProcessorConfig{WorkspacePath: tmpDir, Version: "0.1.0"})
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	mode := "review"
	params := jsonrpc.SendMessageParams{
		Message:  "Hello",
		TaskMode: &mode,
	}

	result, err := processor.HandleSendMessage(context.Background(), params)
	if err != nil {
		t.Fatalf("HandleSendMessage failed: %v", err)
	}

	if result.TaskMode != "review" {
		t.Errorf("Expected task mode 'review', got '%s'", result.TaskMode)
	}

	// Verify conversation has the task mode
	processor.mu.Lock()
	conv := processor.conversations[result.ConversationID]
	processor.mu.Unlock()

	conv.mu.RLock()
	taskMode := conv.taskMode
	conv.mu.RUnlock()

	if taskMode != "review" {
		t.Errorf("Expected conversation task mode 'review', got '%s'", taskMode)
	}
}

func TestProcessor_SwitchTaskMode(t *testing.T) {
	tmpDir := t.TempDir()
	processor, err := NewProcessor(ProcessorConfig{WorkspacePath: tmpDir, Version: "0.1.0"})
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	// Create conversation in regular mode
	regularMode := "regular"
	params1 := jsonrpc.SendMessageParams{
		Message:  "Hello",
		TaskMode: &regularMode,
	}
	result1, err := processor.HandleSendMessage(context.Background(), params1)
	if err != nil {
		t.Fatalf("First HandleSendMessage failed: %v", err)
	}

	if result1.TaskMode != "regular" {
		t.Errorf("Expected task mode 'regular', got '%s'", result1.TaskMode)
	}

	// Switch to review mode
	reviewMode := "review"
	params2 := jsonrpc.SendMessageParams{
		ConversationID: &result1.ConversationID,
		Message:        "Review code",
		TaskMode:       &reviewMode,
	}
	result2, err := processor.HandleSendMessage(context.Background(), params2)
	if err != nil {
		t.Fatalf("Second HandleSendMessage failed: %v", err)
	}

	if result2.TaskMode != "review" {
		t.Errorf("Expected task mode 'review', got '%s'", result2.TaskMode)
	}

	// Verify conversation was updated
	processor.mu.Lock()
	conv := processor.conversations[result1.ConversationID]
	processor.mu.Unlock()

	conv.mu.RLock()
	taskMode := conv.taskMode
	conv.mu.RUnlock()

	if taskMode != "review" {
		t.Errorf("Expected conversation task mode 'review', got '%s'", taskMode)
	}
}

func TestProcessor_InvalidTaskMode(t *testing.T) {
	tmpDir := t.TempDir()
	processor, err := NewProcessor(ProcessorConfig{WorkspacePath: tmpDir, Version: "0.1.0"})
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	invalidMode := "invalid"
	params := jsonrpc.SendMessageParams{
		Message:  "Hello",
		TaskMode: &invalidMode,
	}

	_, err = processor.HandleSendMessage(context.Background(), params)
	if err == nil {
		t.Fatal("Expected error for invalid task mode")
	}

	rpcErr, ok := err.(*jsonrpc.Error)
	if !ok {
		t.Fatalf("Expected JSON-RPC error, got %T", err)
	}

	if rpcErr.Code != jsonrpc.InvalidParams {
		t.Errorf("Expected error code %d, got %d", jsonrpc.InvalidParams, rpcErr.Code)
	}

	if !contains(rpcErr.Message, "invalid task mode") {
		t.Errorf("Expected error message to contain 'invalid task mode', got '%s'", rpcErr.Message)
	}
}

func TestProcessor_NoTaskModeUsesDefault(t *testing.T) {
	tmpDir := t.TempDir()
	processor, err := NewProcessor(ProcessorConfig{WorkspacePath: tmpDir, Version: "0.1.0"})
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	params := jsonrpc.SendMessageParams{
		Message: "Hello",
		// TaskMode is nil
	}

	result, err := processor.HandleSendMessage(context.Background(), params)
	if err != nil {
		t.Fatalf("HandleSendMessage failed: %v", err)
	}

	if result.TaskMode != "regular" {
		t.Errorf("Expected default task mode 'regular', got '%s'", result.TaskMode)
	}
}

func TestProcessor_AllTaskModesValid(t *testing.T) {
	tmpDir := t.TempDir()

	validModes := []string{"regular", "review", "compact", "planning"}

	for _, mode := range validModes {
		t.Run(mode, func(t *testing.T) {
			processor, err := NewProcessor(ProcessorConfig{WorkspacePath: tmpDir, Version: "0.1.0"})
			if err != nil {
				t.Fatalf("Failed to create processor: %v", err)
			}

			modePtr := mode
			params := jsonrpc.SendMessageParams{
				Message:  "Test message",
				TaskMode: &modePtr,
			}

			result, err := processor.HandleSendMessage(context.Background(), params)
			if err != nil {
				t.Fatalf("HandleSendMessage failed for mode '%s': %v", mode, err)
			}

			if result.TaskMode != mode {
				t.Errorf("Expected task mode '%s', got '%s'", mode, result.TaskMode)
			}
		})
	}
}

func TestProcessor_TaskModePersistsAcrossTurns(t *testing.T) {
	tmpDir := t.TempDir()
	processor, err := NewProcessor(ProcessorConfig{WorkspacePath: tmpDir, Version: "0.1.0"})
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	// Create conversation in compact mode
	compactMode := "compact"
	params1 := jsonrpc.SendMessageParams{
		Message:  "First message",
		TaskMode: &compactMode,
	}
	result1, err := processor.HandleSendMessage(context.Background(), params1)
	if err != nil {
		t.Fatalf("First HandleSendMessage failed: %v", err)
	}

	if result1.TaskMode != "compact" {
		t.Errorf("Expected task mode 'compact', got '%s'", result1.TaskMode)
	}

	// Send second message without specifying task mode
	params2 := jsonrpc.SendMessageParams{
		ConversationID: &result1.ConversationID,
		Message:        "Second message",
		// TaskMode not specified - should use current mode
	}
	result2, err := processor.HandleSendMessage(context.Background(), params2)
	if err != nil {
		t.Fatalf("Second HandleSendMessage failed: %v", err)
	}

	// Should still be in compact mode
	if result2.TaskMode != "compact" {
		t.Errorf("Expected task mode to persist as 'compact', got '%s'", result2.TaskMode)
	}
}

// Helper function for string contains check
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || (len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
