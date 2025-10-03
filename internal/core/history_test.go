package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// TestMessage_Creation tests basic message creation
func TestMessage_Creation(t *testing.T) {
	msg := Message{
		Role:    RoleUser,
		Content: "Test message",
	}

	if msg.Role != RoleUser {
		t.Errorf("Expected role %s, got %s", RoleUser, msg.Role)
	}
	if msg.Content != "Test message" {
		t.Errorf("Expected content 'Test message', got %s", msg.Content)
	}
}

// TestMessage_JSONSerialization tests message JSON serialization
func TestMessage_JSONSerialization(t *testing.T) {
	now := time.Now()
	msg := Message{
		ID:        "msg-1",
		Role:      RoleAssistant,
		Content:   "Response",
		Timestamp: now,
		Tokens:    10,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded Message
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.Role != msg.Role {
		t.Errorf("Expected role %s, got %s", msg.Role, decoded.Role)
	}
	if decoded.Content != msg.Content {
		t.Errorf("Expected content %s, got %s", msg.Content, decoded.Content)
	}
	if decoded.Tokens != msg.Tokens {
		t.Errorf("Expected tokens %d, got %d", msg.Tokens, decoded.Tokens)
	}
}

// TestHistory_NewHistory tests history creation
func TestHistory_NewHistory(t *testing.T) {
	tokenizer := &SimpleTokenizer{}
	h := NewHistory(1000, tokenizer)

	if h == nil {
		t.Fatal("NewHistory returned nil")
	}

	if h.Count() != 0 {
		t.Errorf("Expected empty history, got count %d", h.Count())
	}

	if h.IsEmpty() != true {
		t.Error("Expected IsEmpty() to return true for new history")
	}
}

// TestHistory_AddMessage tests adding a message
func TestHistory_AddMessage(t *testing.T) {
	h := NewHistory(1000, &SimpleTokenizer{})

	msg := Message{
		ID:      "msg-1",
		Role:    RoleUser,
		Content: "Hello",
	}

	err := h.AddMessage(msg)
	if err != nil {
		t.Fatalf("AddMessage failed: %v", err)
	}

	if h.Count() != 1 {
		t.Errorf("Expected count 1, got %d", h.Count())
	}

	messages := h.Messages()
	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}

	if messages[0].Role != RoleUser {
		t.Errorf("Expected role %s, got %s", RoleUser, messages[0].Role)
	}
	if messages[0].Content != "Hello" {
		t.Errorf("Expected content 'Hello', got %s", messages[0].Content)
	}
}

// TestHistory_AddUserMessage tests convenience method
func TestHistory_AddUserMessage(t *testing.T) {
	h := NewHistory(1000, &SimpleTokenizer{})

	err := h.AddUserMessage("Hello world")
	if err != nil {
		t.Fatalf("AddUserMessage failed: %v", err)
	}

	if h.Count() != 1 {
		t.Errorf("Expected count 1, got %d", h.Count())
	}

	messages := h.Messages()
	if messages[0].Role != RoleUser {
		t.Errorf("Expected role %s, got %s", RoleUser, messages[0].Role)
	}
}

// TestHistory_AddSystemMessage tests system message
func TestHistory_AddSystemMessage(t *testing.T) {
	h := NewHistory(1000, &SimpleTokenizer{})

	err := h.AddSystemMessage("You are a helpful assistant")
	if err != nil {
		t.Fatalf("AddSystemMessage failed: %v", err)
	}

	messages := h.Messages()
	if messages[0].Role != RoleSystem {
		t.Errorf("Expected role %s, got %s", RoleSystem, messages[0].Role)
	}
}

// TestHistory_AddAssistantMessage tests assistant message
func TestHistory_AddAssistantMessage(t *testing.T) {
	h := NewHistory(1000, &SimpleTokenizer{})

	err := h.AddAssistantMessage("I can help you")
	if err != nil {
		t.Fatalf("AddAssistantMessage failed: %v", err)
	}

	messages := h.Messages()
	if messages[0].Role != RoleAssistant {
		t.Errorf("Expected role %s, got %s", RoleAssistant, messages[0].Role)
	}
}

// TestHistory_AddToolMessage tests tool message
func TestHistory_AddToolMessage(t *testing.T) {
	h := NewHistory(1000, &SimpleTokenizer{})

	err := h.AddToolMessage("call-1", "Tool result")
	if err != nil {
		t.Fatalf("AddToolMessage failed: %v", err)
	}

	messages := h.Messages()
	if messages[0].Role != RoleTool {
		t.Errorf("Expected role %s, got %s", RoleTool, messages[0].Role)
	}
	if messages[0].ToolCallID != "call-1" {
		t.Errorf("Expected tool call ID 'call-1', got %s", messages[0].ToolCallID)
	}
}

// TestHistory_TokenCount tests token counting
func TestHistory_TokenCount(t *testing.T) {
	h := NewHistory(1000, &SimpleTokenizer{})

	err := h.AddUserMessage("Hello world")
	if err != nil {
		t.Fatalf("AddUserMessage failed: %v", err)
	}

	count := h.TokenCount()
	if count <= 0 {
		t.Errorf("Expected positive token count, got %d", count)
	}
}

// TestHistory_Messages tests message retrieval
func TestHistory_Messages(t *testing.T) {
	h := NewHistory(1000, &SimpleTokenizer{})

	_ = h.AddUserMessage("Message 1")
	_ = h.AddAssistantMessage("Response 1")
	_ = h.AddUserMessage("Message 2")

	messages := h.Messages()
	if len(messages) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(messages))
	}

	// Should return a copy, not original
	messages[0].Content = "Modified"
	original := h.Messages()
	if original[0].Content == "Modified" {
		t.Error("Messages() should return a defensive copy")
	}
}

// TestHistory_LastMessage tests retrieving last message
func TestHistory_LastMessage(t *testing.T) {
	h := NewHistory(1000, &SimpleTokenizer{})

	// Empty history
	_, err := h.LastMessage()
	if err == nil {
		t.Error("Expected error for empty history")
	}

	// Add messages
	_ = h.AddUserMessage("Message 1")
	_ = h.AddUserMessage("Message 2")

	last, err := h.LastMessage()
	if err != nil {
		t.Fatalf("LastMessage failed: %v", err)
	}

	if last.Content != "Message 2" {
		t.Errorf("Expected 'Message 2', got %s", last.Content)
	}
}

// TestHistory_GetMessage tests getting message by ID
func TestHistory_GetMessage(t *testing.T) {
	h := NewHistory(1000, &SimpleTokenizer{})

	msg := Message{
		ID:      "msg-123",
		Role:    RoleUser,
		Content: "Test",
	}
	_ = h.AddMessage(msg)

	retrieved, err := h.GetMessage("msg-123")
	if err != nil {
		t.Fatalf("GetMessage failed: %v", err)
	}

	if retrieved.ID != "msg-123" {
		t.Errorf("Expected ID 'msg-123', got %s", retrieved.ID)
	}

	// Non-existent ID
	_, err = h.GetMessage("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent message")
	}
}

// TestHistory_Truncate_PreservesSystemMessage tests system message preservation
func TestHistory_Truncate_PreservesSystemMessage(t *testing.T) {
	h := NewHistory(10000, &SimpleTokenizer{})

	_ = h.AddSystemMessage("You are a helpful assistant")
	_ = h.AddUserMessage("Message 1")
	_ = h.AddAssistantMessage("Response 1")
	_ = h.AddUserMessage("Message 2")
	_ = h.AddAssistantMessage("Response 2")
	_ = h.AddUserMessage("Message 3")
	_ = h.AddAssistantMessage("Response 3")

	// Truncate to small budget
	err := h.Truncate(50)
	if err != nil {
		t.Fatalf("Truncate failed: %v", err)
	}

	messages := h.Messages()

	// System message should be first
	if len(messages) == 0 {
		t.Fatal("Expected at least system message after truncation")
	}

	if messages[0].Role != RoleSystem {
		t.Errorf("Expected first message to be system, got %s", messages[0].Role)
	}
}

// TestHistory_Truncate_KeepsRecentMessages tests recent message retention
func TestHistory_Truncate_KeepsRecentMessages(t *testing.T) {
	h := NewHistory(10000, &SimpleTokenizer{})

	// Add many messages
	for i := 0; i < 20; i++ {
		_ = h.AddUserMessage(fmt.Sprintf("Message %d", i))
		_ = h.AddAssistantMessage(fmt.Sprintf("Response %d", i))
	}

	initialCount := h.Count()

	err := h.Truncate(200)
	if err != nil {
		t.Fatalf("Truncate failed: %v", err)
	}

	finalCount := h.Count()
	if finalCount >= initialCount {
		t.Errorf("Expected truncation, but count went from %d to %d", initialCount, finalCount)
	}

	// Most recent messages should be present
	messages := h.Messages()
	lastMsg := messages[len(messages)-1]
	if lastMsg.Content != "Response 19" {
		t.Errorf("Expected last message to be 'Response 19', got %s", lastMsg.Content)
	}
}

// TestHistory_Truncate_NoSystemMessage tests truncation without system message
func TestHistory_Truncate_NoSystemMessage(t *testing.T) {
	h := NewHistory(10000, &SimpleTokenizer{})

	// Add messages without system message
	for i := 0; i < 10; i++ {
		_ = h.AddUserMessage(fmt.Sprintf("Message %d", i))
	}

	err := h.Truncate(100)
	if err != nil {
		t.Fatalf("Truncate failed: %v", err)
	}

	// Should work fine without system message
	if h.Count() == 0 {
		t.Error("Expected some messages after truncation")
	}
}

// TestHistory_Truncate_WithinBudget tests truncation when already within budget
func TestHistory_Truncate_WithinBudget(t *testing.T) {
	h := NewHistory(10000, &SimpleTokenizer{})

	_ = h.AddUserMessage("Short message")

	initialCount := h.Count()
	err := h.Truncate(10000)
	if err != nil {
		t.Fatalf("Truncate failed: %v", err)
	}

	if h.Count() != initialCount {
		t.Error("Should not truncate when within budget")
	}
}

// TestHistory_TruncateToFit tests truncating to max tokens
func TestHistory_TruncateToFit(t *testing.T) {
	h := NewHistory(100, &SimpleTokenizer{})

	// Add many messages exceeding budget
	for i := 0; i < 50; i++ {
		_ = h.AddUserMessage("This is a test message")
	}

	err := h.TruncateToFit()
	if err != nil {
		t.Fatalf("TruncateToFit failed: %v", err)
	}

	if h.TokenCount() > 100 {
		t.Errorf("Token count %d exceeds budget 100", h.TokenCount())
	}
}

// TestHistory_WouldExceedBudget tests budget checking
func TestHistory_WouldExceedBudget(t *testing.T) {
	h := NewHistory(100, &SimpleTokenizer{})

	_ = h.AddUserMessage("Short message")

	// Small message should not exceed
	smallMsg := Message{Role: RoleUser, Content: "Hi"}
	smallMsg.Tokens = 5

	if h.WouldExceedBudget(smallMsg) {
		t.Error("Small message should not exceed budget")
	}

	// Large message should exceed
	largeMsg := Message{Role: RoleUser, Content: "Very long message..."}
	largeMsg.Tokens = 200

	if !h.WouldExceedBudget(largeMsg) {
		t.Error("Large message should exceed budget")
	}
}

// TestHistory_Clear tests clearing history
func TestHistory_Clear(t *testing.T) {
	h := NewHistory(1000, &SimpleTokenizer{})

	_ = h.AddSystemMessage("System")
	_ = h.AddUserMessage("User")
	_ = h.AddAssistantMessage("Assistant")

	err := h.Clear()
	if err != nil {
		t.Fatalf("Clear failed: %v", err)
	}

	// Should have only system message
	if h.Count() != 1 {
		t.Errorf("Expected 1 message (system) after Clear, got %d", h.Count())
	}

	messages := h.Messages()
	if messages[0].Role != RoleSystem {
		t.Error("System message should be preserved")
	}
}

// TestHistory_ClearAll tests clearing all messages
func TestHistory_ClearAll(t *testing.T) {
	h := NewHistory(1000, &SimpleTokenizer{})

	_ = h.AddSystemMessage("System")
	_ = h.AddUserMessage("User")

	err := h.ClearAll()
	if err != nil {
		t.Fatalf("ClearAll failed: %v", err)
	}

	if h.Count() != 0 {
		t.Errorf("Expected 0 messages after ClearAll, got %d", h.Count())
	}

	if !h.IsEmpty() {
		t.Error("History should be empty after ClearAll")
	}
}

// TestHistory_Reset tests resetting with new system message
func TestHistory_Reset(t *testing.T) {
	h := NewHistory(1000, &SimpleTokenizer{})

	_ = h.AddSystemMessage("Old system")
	_ = h.AddUserMessage("User")

	err := h.Reset("New system message")
	if err != nil {
		t.Fatalf("Reset failed: %v", err)
	}

	if h.Count() != 1 {
		t.Errorf("Expected 1 message after Reset, got %d", h.Count())
	}

	messages := h.Messages()
	if messages[0].Content != "New system message" {
		t.Errorf("Expected 'New system message', got %s", messages[0].Content)
	}
}

// TestHistory_Clone tests cloning history
func TestHistory_Clone(t *testing.T) {
	h := NewHistory(1000, &SimpleTokenizer{})

	_ = h.AddUserMessage("Message 1")
	_ = h.AddAssistantMessage("Response 1")

	clone := h.Clone()

	if clone.Count() != h.Count() {
		t.Errorf("Clone count mismatch: %d vs %d", clone.Count(), h.Count())
	}

	// Modifying clone should not affect original
	_ = clone.AddUserMessage("New message")

	if clone.Count() == h.Count() {
		t.Error("Clone should be independent of original")
	}
}

// TestHistory_Export tests exporting to file
func TestHistory_Export(t *testing.T) {
	h := NewHistory(1000, &SimpleTokenizer{})
	_ = h.AddSystemMessage("System message")
	_ = h.AddUserMessage("User message")

	tmpFile := filepath.Join(t.TempDir(), "history.json")
	err := h.Export(tmpFile)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// File should exist
	_, err = os.Stat(tmpFile)
	if err != nil {
		t.Errorf("Export file does not exist: %v", err)
	}
}

// TestHistory_ExportJSON tests exporting to JSON bytes
func TestHistory_ExportJSON(t *testing.T) {
	h := NewHistory(1000, &SimpleTokenizer{})
	_ = h.AddSystemMessage("System")
	_ = h.AddUserMessage("User")

	data, err := h.ExportJSON()
	if err != nil {
		t.Fatalf("ExportJSON failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("ExportJSON returned empty data")
	}

	// Should be valid JSON
	var parsed map[string]interface{}
	err = json.Unmarshal(data, &parsed)
	if err != nil {
		t.Errorf("ExportJSON did not produce valid JSON: %v", err)
	}
}

// TestHistory_Import tests importing from file
func TestHistory_Import(t *testing.T) {
	h := NewHistory(1000, &SimpleTokenizer{})
	_ = h.AddSystemMessage("System")
	_ = h.AddUserMessage("User")

	tmpFile := filepath.Join(t.TempDir(), "history.json")
	err := h.Export(tmpFile)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Import
	imported, err := Import(tmpFile, &SimpleTokenizer{})
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	if imported.Count() != h.Count() {
		t.Errorf("Import count mismatch: %d vs %d", imported.Count(), h.Count())
	}
}

// TestHistory_ImportJSON tests importing from JSON bytes
func TestHistory_ImportJSON(t *testing.T) {
	h := NewHistory(1000, &SimpleTokenizer{})
	_ = h.AddSystemMessage("System")
	_ = h.AddUserMessage("User")

	data, err := h.ExportJSON()
	if err != nil {
		t.Fatalf("ExportJSON failed: %v", err)
	}

	imported, err := ImportJSON(data, &SimpleTokenizer{})
	if err != nil {
		t.Fatalf("ImportJSON failed: %v", err)
	}

	if imported.Count() != h.Count() {
		t.Errorf("Import count mismatch: %d vs %d", imported.Count(), h.Count())
	}
}

// TestHistory_ImportExport_RoundTrip tests full round trip
func TestHistory_ImportExport_RoundTrip(t *testing.T) {
	h := NewHistory(1000, &SimpleTokenizer{})
	_ = h.AddSystemMessage("System")
	_ = h.AddUserMessage("User")
	_ = h.AddAssistantMessage("Assistant")

	tmpFile := filepath.Join(t.TempDir(), "history.json")
	err := h.Export(tmpFile)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	imported, err := Import(tmpFile, &SimpleTokenizer{})
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Compare
	if imported.Count() != h.Count() {
		t.Errorf("Count mismatch: %d vs %d", imported.Count(), h.Count())
	}

	originalMsgs := h.Messages()
	importedMsgs := imported.Messages()

	for i := range originalMsgs {
		if originalMsgs[i].Content != importedMsgs[i].Content {
			t.Errorf("Message %d content mismatch", i)
		}
		if originalMsgs[i].Role != importedMsgs[i].Role {
			t.Errorf("Message %d role mismatch", i)
		}
	}
}

// TestHistory_ConcurrentAdd tests concurrent message addition
func TestHistory_ConcurrentAdd(t *testing.T) {
	h := NewHistory(100000, &SimpleTokenizer{})

	var wg sync.WaitGroup
	numGoroutines := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			_ = h.AddUserMessage(fmt.Sprintf("Message %d", id))
		}(i)
	}

	wg.Wait()

	if h.Count() != numGoroutines {
		t.Errorf("Expected %d messages, got %d", numGoroutines, h.Count())
	}
}

// TestHistory_ConcurrentReadWrite tests concurrent reads and writes
func TestHistory_ConcurrentReadWrite(t *testing.T) {
	h := NewHistory(100000, &SimpleTokenizer{})

	var wg sync.WaitGroup

	// Writers
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			_ = h.AddUserMessage(fmt.Sprintf("Message %d", id))
		}(i)
	}

	// Readers
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = h.Messages()
			_ = h.TokenCount()
			_ = h.Count()
		}()
	}

	wg.Wait()
}

// TestHistory_InvalidMessage tests invalid message handling
func TestHistory_InvalidMessage(t *testing.T) {
	h := NewHistory(1000, &SimpleTokenizer{})

	// Empty role
	msg := Message{
		Content: "Test",
	}

	err := h.AddMessage(msg)
	if err == nil {
		t.Error("Expected error for message with empty role")
	}
}

// TestSimpleTokenizer_Count tests token counting
func TestSimpleTokenizer_Count(t *testing.T) {
	tokenizer := &SimpleTokenizer{}

	count := tokenizer.Count("hello world")
	if count <= 0 {
		t.Errorf("Expected positive token count, got %d", count)
	}

	// More words should mean more tokens
	count1 := tokenizer.Count("hello")
	count2 := tokenizer.Count("hello world test message")

	if count2 <= count1 {
		t.Error("Longer text should have more tokens")
	}
}

// TestSimpleTokenizer_CountMessages tests message token counting
func TestSimpleTokenizer_CountMessages(t *testing.T) {
	tokenizer := &SimpleTokenizer{}

	messages := []Message{
		{Role: RoleUser, Content: "Hello"},
		{Role: RoleAssistant, Content: "Hi there"},
	}

	count := tokenizer.CountMessages(messages)
	if count <= 0 {
		t.Errorf("Expected positive token count, got %d", count)
	}
}
