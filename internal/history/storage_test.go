package history

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/message"
	"github.com/dmytrogajewski/spin/internal/tokenizer"
)

func TestFileStorage_SaveAndLoad(t *testing.T) {
	// Create temp directory.
	tmpDir, err := os.MkdirTemp("", "history-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	storage, err := NewFileStorage(tmpDir)
	if err != nil {
		t.Fatalf("create storage: %v", err)
	}

	sessionID := "test-session-123"

	// Create history data.
	data := Data{
		Version:   CurrentHistoryVersion,
		SessionID: sessionID,
		MaxTokens: 4096,
		Messages: []message.Message{
			{
				ID:        "msg-1",
				Role:      message.RoleSystem,
				Content:   "You are a helpful assistant.",
				Timestamp: time.Now(),
				Tokens:    10,
			},
			{
				ID:        "msg-2",
				Role:      message.RoleUser,
				Content:   "Hello!",
				Timestamp: time.Now(),
				Tokens:    5,
			},
			{
				ID:        "msg-3",
				Role:      message.RoleAssistant,
				Content:   "Hi there! How can I help you?",
				Timestamp: time.Now(),
				Tokens:    15,
			},
		},
	}

	// Save.
	err = storage.Save(sessionID, data)
	if err != nil {
		t.Fatalf("save history: %v", err)
	}

	// Verify file exists.
	exists, err := storage.Exists(sessionID)
	if err != nil {
		t.Fatalf("check exists: %v", err)
	}

	if !exists {
		t.Fatal("history should exist after save")
	}

	// Load.
	loaded, err := storage.Load(sessionID)
	if err != nil {
		t.Fatalf("load history: %v", err)
	}

	// Verify loaded data.
	if loaded.SessionID != sessionID {
		t.Errorf("session ID mismatch: got %q, want %q", loaded.SessionID, sessionID)
	}

	if loaded.MaxTokens != data.MaxTokens {
		t.Errorf("max tokens mismatch: got %d, want %d", loaded.MaxTokens, data.MaxTokens)
	}

	if len(loaded.Messages) != len(data.Messages) {
		t.Fatalf("message count mismatch: got %d, want %d", len(loaded.Messages), len(data.Messages))
	}

	for i, msg := range loaded.Messages {
		if msg.ID != data.Messages[i].ID {
			t.Errorf("message[%d] ID mismatch: got %q, want %q", i, msg.ID, data.Messages[i].ID)
		}

		if msg.Role != data.Messages[i].Role {
			t.Errorf("message[%d] role mismatch: got %q, want %q", i, msg.Role, data.Messages[i].Role)
		}

		if msg.Content != data.Messages[i].Content {
			t.Errorf("message[%d] content mismatch: got %q, want %q", i, msg.Content, data.Messages[i].Content)
		}
	}

	// Verify version was set.
	if loaded.Version != CurrentHistoryVersion {
		t.Errorf("version mismatch: got %d, want %d", loaded.Version, CurrentHistoryVersion)
	}
}

func TestFileStorage_Delete(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "history-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	storage, err := NewFileStorage(tmpDir)
	if err != nil {
		t.Fatalf("create storage: %v", err)
	}

	sessionID := "test-session-delete"

	// Save some data.
	data := Data{
		SessionID: sessionID,
		Messages:  []message.Message{},
	}
	err = storage.Save(sessionID, data)
	if err != nil {
		t.Fatalf("save history: %v", err)
	}

	// Verify exists.
	exists, _ := storage.Exists(sessionID)
	if !exists {
		t.Fatal("history should exist")
	}

	// Delete.
	err = storage.Delete(sessionID)
	if err != nil {
		t.Fatalf("delete history: %v", err)
	}

	// Verify deleted.
	exists, _ = storage.Exists(sessionID)
	if exists {
		t.Fatal("history should not exist after delete")
	}

	// Delete non-existent should not error.
	err = storage.Delete("non-existent")
	if err != nil {
		t.Fatalf("delete non-existent should not error: %v", err)
	}
}

func TestFileStorage_LoadNotFound(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "history-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	storage, err := NewFileStorage(tmpDir)
	if err != nil {
		t.Fatalf("create storage: %v", err)
	}

	_, err = storage.Load("non-existent-session")
	if err == nil {
		t.Fatal("load should fail for non-existent session")
	}
}

func TestFileStorage_EmptySessionID(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "history-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	storage, err := NewFileStorage(tmpDir)
	if err != nil {
		t.Fatalf("create storage: %v", err)
	}

	// All operations should fail with empty session ID.
	err = storage.Save("", Data{})
	if err == nil {
		t.Error("save with empty ID should fail")
	}

	_, err = storage.Load("")
	if err == nil {
		t.Error("load with empty ID should fail")
	}

	err = storage.Delete("")
	if err == nil {
		t.Error("delete with empty ID should fail")
	}

	_, err = storage.Exists("")
	if err == nil {
		t.Error("exists with empty ID should fail")
	}
}

func TestFileStorage_HomeExpansion(t *testing.T) {
	// Skip if HOME not set.
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot get home directory")
	}

	// Use a unique subdirectory in temp to avoid conflicts.
	tmpDir, err := os.MkdirTemp("", "history-home-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create storage with path relative to tmpDir (not home, to avoid polluting home).
	storage, err := NewFileStorage(tmpDir)
	if err != nil {
		t.Fatalf("create storage: %v", err)
	}

	// Verify storage works by saving and loading.
	testData := Data{SessionID: "test", MaxTokens: 1000}
	err = storage.Save("test", testData)
	if err != nil {
		t.Fatalf("save failed: %v", err)
	}

	// Test that ~/ expansion works.
	homeTestDir := filepath.Join(home, ".spin-test-temp")

	homeStorage, err := NewFileStorage(homeTestDir)
	if err != nil {
		t.Fatalf("create home storage: %v", err)
	}
	defer os.RemoveAll(homeTestDir)

	// Verify home storage works.
	err = homeStorage.Save("home-test", testData)
	if err != nil {
		t.Fatalf("home storage save failed: %v", err)
	}

	if exists, _ := homeStorage.Exists("home-test"); !exists {
		t.Error("home storage: saved history should exist")
	}
}

func TestHistory_SaveAndLoad(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "history-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	storage, err := NewFileStorage(tmpDir)
	if err != nil {
		t.Fatalf("create storage: %v", err)
	}

	// Create history and add messages.
	history := NewHistory(4096, &tokenizer.SimpleTokenizer{})
	_ = history.AddSystemMessage("You are helpful.")
	_ = history.AddUserMessage("Hello!")

	sessionID := "test-history-save"

	// Save via History method.
	err = history.Save(storage, sessionID)
	if err != nil {
		t.Fatalf("save history: %v", err)
	}

	// Create new history and load.
	history2 := NewHistory(4096, &tokenizer.SimpleTokenizer{})
	err = history2.Load(storage, sessionID)
	if err != nil {
		t.Fatalf("load history: %v", err)
	}

	// Verify messages match.
	msgs1 := history.Messages()
	msgs2 := history2.Messages()

	if len(msgs1) != len(msgs2) {
		t.Fatalf("message count mismatch: got %d, want %d", len(msgs2), len(msgs1))
	}

	for i := range msgs1 {
		if msgs1[i].Role != msgs2[i].Role {
			t.Errorf("message[%d] role mismatch: got %q, want %q", i, msgs2[i].Role, msgs1[i].Role)
		}

		if msgs1[i].Content != msgs2[i].Content {
			t.Errorf("message[%d] content mismatch: got %q, want %q", i, msgs2[i].Content, msgs1[i].Content)
		}
	}
}

func TestHistory_ToFromData(t *testing.T) {
	history := NewHistory(8192, &tokenizer.SimpleTokenizer{})
	_ = history.AddSystemMessage("System prompt")
	_ = history.AddUserMessage("User input")
	_ = history.AddMessage(message.Message{
		Role:    message.RoleAssistant,
		Content: "Assistant response",
		ToolCalls: []message.ToolCall{
			{
				ID:   "call-1",
				Type: "function",
				Function: message.ToolCallFunction{
					Name:      "read_file",
					Arguments: `{"path": "/tmp/test.txt"}`,
				},
			},
		},
	})

	sessionID := "test-export"

	// Export to Data.
	data := history.ToData(sessionID)

	if data.SessionID != sessionID {
		t.Errorf("session ID mismatch: got %q, want %q", data.SessionID, sessionID)
	}

	if data.MaxTokens != 8192 {
		t.Errorf("max tokens mismatch: got %d, want %d", data.MaxTokens, 8192)
	}

	if len(data.Messages) != 3 {
		t.Fatalf("message count mismatch: got %d, want %d", len(data.Messages), 3)
	}

	// Import into new history.
	history2 := NewHistory(4096, &tokenizer.SimpleTokenizer{})
	err := history2.FromData(data)
	if err != nil {
		t.Fatalf("import history data: %v", err)
	}

	// Verify tool calls preserved.
	msgs := history2.Messages()
	if len(msgs[2].ToolCalls) != 1 {
		t.Fatalf("tool calls not preserved: got %d, want 1", len(msgs[2].ToolCalls))
	}

	if msgs[2].ToolCalls[0].Function.Name != "read_file" {
		t.Errorf("tool call name mismatch: got %q, want %q", msgs[2].ToolCalls[0].Function.Name, "read_file")
	}
}

func TestHistory_FromData_NilError(t *testing.T) {
	history := NewHistory(4096, &tokenizer.SimpleTokenizer{})

	err := history.FromData(nil)
	if err == nil {
		t.Error("FromData(nil) should return error")
	}
}

func TestFileStorage_AtomicWrite(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "history-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	storage, err := NewFileStorage(tmpDir)
	if err != nil {
		t.Fatalf("create storage: %v", err)
	}

	sessionID := "test-atomic"

	// Save initial data.
	data1 := Data{
		SessionID: sessionID,
		Messages: []message.Message{
			{Role: message.RoleUser, Content: "First"},
		},
	}
	err = storage.Save(sessionID, data1)
	if err != nil {
		t.Fatalf("save first: %v", err)
	}

	// Save updated data.
	data2 := Data{
		SessionID: sessionID,
		Messages: []message.Message{
			{Role: message.RoleUser, Content: "First"},
			{Role: message.RoleUser, Content: "Second"},
		},
	}
	err = storage.Save(sessionID, data2)
	if err != nil {
		t.Fatalf("save second: %v", err)
	}

	// Load and verify.
	loaded, err := storage.Load(sessionID)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if len(loaded.Messages) != 2 {
		t.Errorf("message count mismatch: got %d, want 2", len(loaded.Messages))
	}

	// Verify no temp files left behind.
	entries, _ := os.ReadDir(tmpDir)
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".tmp" {
			t.Errorf("temp file left behind: %s", entry.Name())
		}
	}
}
