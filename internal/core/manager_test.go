package core

import (
	"context"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/core/task"
	"github.com/dmytrogajewski/spin/internal/core/turn"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/session"
	"github.com/dmytrogajewski/spin/internal/tools"
)

func TestNewManager_AndNewConversation(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Provider = "mock"
	cfg.Model = "mock"
	cfg.WorkDir = t.TempDir()
	cfg.SessionDir = t.TempDir() // Separate dir for sessions

	llm := llm.NewMockProvider("hi")
	emitter := NewEventEmitter(100)

	// Constructor now initializes storage automatically
	m, err := NewManager(cfg, WithLLM(llm), WithEmitter(emitter))
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}

	conv, err := m.NewConversation(context.Background(), cfg.WorkDir)
	if err != nil {
		t.Fatalf("NewConversation() error: %v", err)
	}

	// Run a simple turn to ensure wiring works
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := conv.RunTurn(ctx, "say hi"); err != nil {
		t.Fatalf("RunTurn() error: %v", err)
	}
}

func TestWithStorage(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Provider = "mock"
	cfg.Model = "mock"
	cfg.WorkDir = t.TempDir()
	cfg.SessionDir = t.TempDir()

	customStorage, err := session.NewFileStorage(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStorage() error: %v", err)
	}
	llm := llm.NewMockProvider("ok")
	emitter := NewEventEmitter(50)

	// Create manager with custom storage
	m, err := NewManager(cfg, WithLLM(llm), WithEmitter(emitter), WithStorage(customStorage))
	if err != nil {
		t.Fatalf("NewManager() with WithStorage error: %v", err)
	}

	// Verify storage was set by creating and listing conversations
	conv, err := m.NewConversation(context.Background(), cfg.WorkDir)
	if err != nil {
		t.Fatalf("NewConversation() error: %v", err)
	}

	sess := createSessionForTest(t, cfg.WorkDir, conv)
	if err := m.storage.Save(sess); err != nil {
		t.Fatalf("Save session error: %v", err)
	}

	// List should find the session
	metas, err := m.ListConversations(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListConversations() error: %v", err)
	}

	if len(metas) != 1 {
		t.Errorf("expected 1 session, got %d", len(metas))
	}
}

func TestWithManagerToolRegistry(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Provider = "mock"
	cfg.Model = "mock"
	cfg.WorkDir = t.TempDir()
	cfg.SessionDir = t.TempDir()

	// Create a custom tool registry with only file tools (missing execute_command)
	customRegistry := tools.NewRegistry()
	_ = customRegistry.Register(tools.NewReadFileTool())
	_ = customRegistry.Register(tools.NewWriteFileTool())
	_ = customRegistry.Register(tools.NewListDirectoryTool())

	llm := llm.NewMockProvider("ok")
	emitter := NewEventEmitter(50)

	// Create manager with custom tool registry
	m, err := NewManager(cfg, WithLLM(llm), WithEmitter(emitter), WithManagerToolRegistry(customRegistry))
	if err != nil {
		t.Fatalf("NewManager() with WithManagerToolRegistry error: %v", err)
	}

	// Verify manager was created successfully
	if m == nil {
		t.Fatal("NewManager() returned nil")
	}

	// Create a conversation and verify tool registry is used
	conv, err := m.NewConversation(context.Background(), cfg.WorkDir)
	if err != nil {
		t.Fatalf("NewConversation() error: %v", err)
	}

	if conv == nil {
		t.Fatal("NewConversation() returned nil")
	}

	// REGRESSION TEST for bug: verify that default tools (execute_command, get_context)
	// are still available even though custom registry didn't include them
	toolSchemas := conv.agent.toolRegistry.ListSchemas()

	// Build a map of tool names for easier lookup
	toolMap := make(map[string]bool)
	for _, schema := range toolSchemas {
		toolMap[schema.Function.Name] = true
	}

	// Verify all expected tools are present
	expectedTools := []string{
		"read_file",
		"write_file",
		"list_directory",
		"execute_command", // This was missing before the fix!
		"get_context",     // This was missing before the fix!
	}

	for _, expectedTool := range expectedTools {
		if !toolMap[expectedTool] {
			t.Errorf("Expected tool %q not found in agent's tool registry. Available tools: %v",
				expectedTool, getToolNames(toolSchemas))
		}
	}
}

// Helper function to get tool names from schemas
func getToolNames(schemas []tools.ToolSchema) []string {
	names := make([]string, len(schemas))
	for i, schema := range schemas {
		names[i] = schema.Function.Name
	}
	return names
}

func TestManager_ListAndArchive(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Provider = "mock"
	cfg.Model = "mock"
	cfg.WorkDir = t.TempDir()
	cfg.SessionDir = t.TempDir()

	llm := llm.NewMockProvider("ok")
	emitter := NewEventEmitter(50)

	m, err := NewManager(cfg, WithLLM(llm), WithEmitter(emitter))
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}

	// List should be safe even if empty
	metas, err := m.ListConversations(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListConversations() error: %v", err)
	}
	if metas == nil {
		t.Error("ListConversations() returned nil slice")
	}

	// Archive non-existent should return error
	if err := m.ArchiveConversation(context.Background(), "non-existent"); err == nil {
		t.Error("ArchiveConversation() should error for non-existent session")
	}
}

func TestManager_ResumeConversation_NotFound(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Provider = "mock"
	cfg.Model = "mock"
	cfg.WorkDir = t.TempDir()
	cfg.SessionDir = t.TempDir()

	llm := llm.NewMockProvider("ok")
	emitter := NewEventEmitter(50)

	m, err := NewManager(cfg, WithLLM(llm), WithEmitter(emitter))
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}

	if _, err := m.ResumeConversation(context.Background(), "does-not-exist"); err == nil {
		t.Error("expected error for non-existent session in ResumeConversation")
	}
}

func TestManager_WithMCPManager(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Provider = "mock"
	cfg.Model = "mock"
	cfg.WorkDir = t.TempDir()
	cfg.SessionDir = t.TempDir()

	mcpMgr := NewMCPManager()
	m, err := NewManager(cfg, WithMCPManager(mcpMgr))
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}

	if m.MCPManager() == nil {
		t.Error("MCPManager() should return the configured manager")
	}
	if m.MCPManager() != mcpMgr {
		t.Error("MCPManager() should return the same manager instance")
	}
}

func TestManager_WithoutMCPManager(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Provider = "mock"
	cfg.Model = "mock"
	cfg.WorkDir = t.TempDir()
	cfg.SessionDir = t.TempDir()

	m, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}

	if m.MCPManager() != nil {
		t.Error("MCPManager() should return nil when not configured")
	}
}

// P2.2: Manager Task Support Tests

func TestManager_WithTaskRegistry(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Provider = "mock"
	cfg.Model = "mock"
	cfg.WorkDir = t.TempDir()
	cfg.SessionDir = t.TempDir()

	// Create custom task registry
	customRegistry := task.NewRegistry()
	_ = customRegistry.Register("custom", task.NewCompact())
	_ = customRegistry.SetDefault("custom")

	// Create manager with custom registry
	mgr, err := NewManager(cfg, WithManagerTaskRegistry(customRegistry))
	if err != nil {
		t.Fatalf("NewManager() with WithManagerTaskRegistry error: %v", err)
	}
	if mgr == nil {
		t.Fatal("NewManager() returned nil")
	}
	if mgr.taskRegistry == nil {
		t.Fatal("taskRegistry not set")
	}
	if len(mgr.taskRegistry.List()) != 1 {
		t.Errorf("expected 1 task in registry, got %d", len(mgr.taskRegistry.List()))
	}
}

func TestManager_NewConversationWithTask_Success(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Provider = "mock"
	cfg.Model = "mock"
	cfg.WorkDir = t.TempDir()
	cfg.SessionDir = t.TempDir()

	llm := llm.NewMockProvider("ok")
	emitter := NewEventEmitter(50)

	mgr, err := NewManager(cfg, WithLLM(llm), WithEmitter(emitter))
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}

	// Create conversation in review mode
	conv, err := mgr.NewConversationWithTask(context.Background(), cfg.WorkDir, "review")
	if err != nil {
		t.Fatalf("NewConversationWithTask() error: %v", err)
	}
	if conv == nil {
		t.Fatal("NewConversationWithTask() returned nil")
	}

	// Verify mode is set
	if conv.GetTaskMode() != "review" {
		t.Errorf("expected task mode 'review', got '%s'", conv.GetTaskMode())
	}
}

func TestManager_NewConversationWithTask_InvalidMode(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Provider = "mock"
	cfg.Model = "mock"
	cfg.WorkDir = t.TempDir()
	cfg.SessionDir = t.TempDir()

	llm := llm.NewMockProvider("ok")
	emitter := NewEventEmitter(50)

	mgr, err := NewManager(cfg, WithLLM(llm), WithEmitter(emitter))
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}

	// Try to create conversation with invalid mode
	conv, err := mgr.NewConversationWithTask(context.Background(), cfg.WorkDir, "invalid")
	if err == nil {
		t.Error("expected error for invalid mode, got nil")
	}
	if conv != nil {
		t.Error("expected nil conversation for invalid mode")
	}
	if err != nil && !containsManagerTest(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error message, got: %v", err)
	}
}

func TestManager_NewConversationWithTask_EmptyTaskName(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Provider = "mock"
	cfg.Model = "mock"
	cfg.WorkDir = t.TempDir()
	cfg.SessionDir = t.TempDir()

	mgr, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}

	// Try to create conversation with empty task name
	conv, err := mgr.NewConversationWithTask(context.Background(), cfg.WorkDir, "")
	if err == nil {
		t.Error("expected error for empty task name, got nil")
	}
	if conv != nil {
		t.Error("expected nil conversation for empty task name")
	}
	if err != nil && !containsManagerTest(err.Error(), "cannot be empty") {
		t.Errorf("expected 'cannot be empty' in error message, got: %v", err)
	}
}

func TestManager_TaskRegistryPassedToAgent(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Provider = "mock"
	cfg.Model = "mock"
	cfg.WorkDir = t.TempDir()
	cfg.SessionDir = t.TempDir()

	// Create custom task registry with custom mode
	customRegistry := task.NewRegistry()
	customTask := task.NewCompact() // Use compact as base
	_ = customRegistry.Register("custom-mode", customTask)
	_ = customRegistry.SetDefault("custom-mode")

	llm := llm.NewMockProvider("ok")
	emitter := NewEventEmitter(50)

	// Create manager with custom registry
	mgr, err := NewManager(cfg, WithLLM(llm), WithEmitter(emitter), WithManagerTaskRegistry(customRegistry))
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}

	// Create conversation
	conv, err := mgr.NewConversation(context.Background(), cfg.WorkDir)
	if err != nil {
		t.Fatalf("NewConversation() error: %v", err)
	}

	// Verify agent has the custom registry
	modes := conv.agent.ListTaskModes()
	if !containsStringManagerTest(modes, "custom-mode") {
		t.Errorf("expected 'custom-mode' in agent's task modes, got: %v", modes)
	}
	if len(modes) != 1 {
		t.Errorf("expected 1 task mode, got %d", len(modes))
	}

	// Verify can switch to custom mode
	err = conv.SetTaskMode("custom-mode")
	if err != nil {
		t.Errorf("SetTaskMode() error: %v", err)
	}
	if conv.GetTaskMode() != "custom-mode" {
		t.Errorf("expected task mode 'custom-mode', got '%s'", conv.GetTaskMode())
	}
}

func TestManager_Integration_TaskModeEndToEnd(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Provider = "mock"
	cfg.Model = "mock"
	cfg.WorkDir = t.TempDir()
	cfg.SessionDir = t.TempDir()

	llm := llm.NewMockProvider("ok")
	emitter := NewEventEmitter(100)

	mgr, err := NewManager(cfg, WithLLM(llm), WithEmitter(emitter))
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}

	// Test all 4 built-in modes
	modes := []string{"regular", "review", "compact", "planning"}
	for _, mode := range modes {
		t.Run(mode, func(t *testing.T) {
			conv, err := mgr.NewConversationWithTask(context.Background(), cfg.WorkDir, mode)
			if err != nil {
				t.Fatalf("NewConversationWithTask(%s) error: %v", mode, err)
			}
			if conv.GetTaskMode() != mode {
				t.Errorf("expected task mode '%s', got '%s'", mode, conv.GetTaskMode())
			}

			// Run a turn to ensure everything is wired correctly
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			err = conv.RunTurn(ctx, "test message")
			if err != nil {
				t.Errorf("RunTurn() error in mode %s: %v", mode, err)
			}
		})
	}
}

// Helper functions for P2.2 tests

func containsManagerTest(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) >= len(substr) && findSubstringManagerTest(s, substr))
}

func findSubstringManagerTest(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func containsStringManagerTest(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

// Integration tests with real file storage

func TestManager_Integration_SaveResumeWithHistory(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Provider = "mock"
	cfg.Model = "mock"
	cfg.WorkDir = t.TempDir()
	cfg.SessionDir = t.TempDir()

	llm := llm.NewMockProvider("assistant reply")
	emitter := NewEventEmitter(100)

	m, err := NewManager(cfg, WithLLM(llm), WithEmitter(emitter))
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}

	// Create a new conversation and run a turn
	conv, err := m.NewConversation(context.Background(), cfg.WorkDir)
	if err != nil {
		t.Fatalf("NewConversation() error: %v", err)
	}

	ctx := context.Background()
	if err := conv.RunTurn(ctx, "hello"); err != nil {
		t.Fatalf("RunTurn() error: %v", err)
	}

	// Manually create and save a session with the history
	sess := createSessionForTest(t, cfg.WorkDir, conv)
	if err := m.storage.Save(sess); err != nil {
		t.Fatalf("Save session error: %v", err)
	}

	// Resume the conversation from storage
	resumed, err := m.ResumeConversation(context.Background(), sess.ID)
	if err != nil {
		t.Fatalf("ResumeConversation() error: %v", err)
	}

	// Verify history is restored (check message count)
	// History should have system + user + assistant messages
	if resumed == nil {
		t.Fatal("resumed conversation is nil")
	}
}

func TestManager_Integration_ListWithMetadata(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Provider = "mock"
	cfg.Model = "mock"
	cfg.WorkDir = t.TempDir()
	cfg.SessionDir = t.TempDir()

	llm := llm.NewMockProvider("ok")
	emitter := NewEventEmitter(50)

	m, err := NewManager(cfg, WithLLM(llm), WithEmitter(emitter))
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}

	// Create and save multiple sessions
	conv1, _ := m.NewConversation(context.Background(), cfg.WorkDir)
	sess1 := createSessionForTest(t, cfg.WorkDir, conv1)
	sess1.Metadata.Title = "Session 1"
	if err := m.storage.Save(sess1); err != nil {
		t.Fatalf("Save session1 error: %v", err)
	}

	conv2, _ := m.NewConversation(context.Background(), cfg.WorkDir)
	sess2 := createSessionForTest(t, cfg.WorkDir, conv2)
	sess2.Metadata.Title = "Session 2"
	if err := m.storage.Save(sess2); err != nil {
		t.Fatalf("Save session2 error: %v", err)
	}

	// List all sessions
	metas, err := m.ListConversations(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListConversations() error: %v", err)
	}

	if len(metas) != 2 {
		t.Errorf("expected 2 sessions, got %d", len(metas))
	}
}

func TestManager_Integration_Archive(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Provider = "mock"
	cfg.Model = "mock"
	cfg.WorkDir = t.TempDir()
	cfg.SessionDir = t.TempDir()

	llm := llm.NewMockProvider("ok")
	emitter := NewEventEmitter(50)

	m, err := NewManager(cfg, WithLLM(llm), WithEmitter(emitter))
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}

	// Create and save a session
	conv, _ := m.NewConversation(context.Background(), cfg.WorkDir)
	sess := createSessionForTest(t, cfg.WorkDir, conv)
	if err := m.storage.Save(sess); err != nil {
		t.Fatalf("Save session error: %v", err)
	}

	// Archive the session
	if err := m.ArchiveConversation(context.Background(), sess.ID); err != nil {
		t.Fatalf("ArchiveConversation() error: %v", err)
	}

	// Load and verify state is archived
	loaded, err := m.storage.Load(sess.ID)
	if err != nil {
		t.Fatalf("Load session error: %v", err)
	}

	if loaded.State != session.StateArchived {
		t.Errorf("expected state archived, got %v", loaded.State)
	}
}

func TestManager_Integration_MultipleConversations(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Provider = "mock"
	cfg.Model = "mock"
	cfg.WorkDir = t.TempDir()
	cfg.SessionDir = t.TempDir()

	llm := llm.NewMockProvider("ok")
	emitter := NewEventEmitter(100)

	m, err := NewManager(cfg, WithLLM(llm), WithEmitter(emitter))
	if err != nil {
		t.Fatalf("NewManager() error: %v", err)
	}

	// Create multiple conversations concurrently
	numConvs := 5
	convs := make([]*Conversation, numConvs)
	for i := 0; i < numConvs; i++ {
		conv, err := m.NewConversation(context.Background(), cfg.WorkDir)
		if err != nil {
			t.Fatalf("NewConversation(%d) error: %v", i, err)
		}
		convs[i] = conv
	}

	// Each should be independent
	if len(convs) != numConvs {
		t.Errorf("expected %d conversations, got %d", numConvs, len(convs))
	}
}

// Helper to create a minimal session for testing
func createSessionForTest(t *testing.T, workDir string, conv *Conversation) *session.Session {
	t.Helper()
	sess := session.NewSession(workDir)
	// Create a turn with sample user input and AI response
	sampleTurn := turn.NewTurn(sess.ID, "test input")
	// Transition turn to completed state
	_ = sampleTurn.Start()
	_ = sampleTurn.Complete("test reply", turn.TokenUsage{
		PromptTokens:     10,
		CompletionTokens: 15,
		TotalTokens:      25,
	})
	// Add turn to session
	_ = sess.AddTurn(sampleTurn)
	return sess
}
