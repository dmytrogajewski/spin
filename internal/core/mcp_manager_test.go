package core

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/dmytrogajewski/spin/internal/mcp/types"
)

func TestNewMCPManager(t *testing.T) {
	mgr := NewMCPManager()

	if mgr == nil {
		t.Fatal("NewMCPManager() returned nil")
	}
	if mgr.clients == nil {
		t.Error("clients map should be initialized")
	}
	if mgr.tools == nil {
		t.Error("tools map should be initialized")
	}
	if mgr.ServerCount() != 0 {
		t.Errorf("ServerCount() = %d, want 0", mgr.ServerCount())
	}
	if mgr.ToolCount() != 0 {
		t.Errorf("ToolCount() = %d, want 0", mgr.ToolCount())
	}
}

func TestMCPManager_ListAllTools_Empty(t *testing.T) {
	mgr := NewMCPManager()
	tools := mgr.ListAllTools()

	if len(tools) != 0 {
		t.Errorf("ListAllTools() returned %d tools, want 0", len(tools))
	}
}

func TestMCPManager_GetTool_NotFound(t *testing.T) {
	mgr := NewMCPManager()
	tool, ok := mgr.GetTool("server1", "tool1")

	if ok {
		t.Error("GetTool() should return false for non-existent tool")
	}
	if tool != nil {
		t.Error("GetTool() should return nil for non-existent tool")
	}
}

func TestMCPManager_Close_Empty(t *testing.T) {
	mgr := NewMCPManager()
	err := mgr.Close()

	if err != nil {
		t.Errorf("Close() error = %v, want nil for empty manager", err)
	}
}

func TestMCPManager_CallTool_NotFound(t *testing.T) {
	mgr := NewMCPManager()
	ctx := context.Background()

	args := json.RawMessage(`{"test":"value"}`)
	_, err := mgr.CallTool(ctx, "server1", "tool1", args)

	if err == nil {
		t.Error("CallTool() should return error for non-existent tool")
	}
}

func TestMCPManager_ConnectServers_EmptyConfig(t *testing.T) {
	mgr := NewMCPManager()
	ctx := context.Background()

	// Empty config should succeed (no servers to connect)
	err := mgr.ConnectServers(ctx, map[string]MCPConfig{})

	if err != nil {
		t.Errorf("ConnectServers() error = %v, want nil for empty config", err)
	}
}

func TestMCPManager_Concurrent_Access(t *testing.T) {
	// Test concurrent access to ensure proper locking
	mgr := NewMCPManager()

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			// Concurrent reads
			_ = mgr.ListAllTools()
			_ = mgr.ServerCount()
			_ = mgr.ToolCount()
			_, _ = mgr.GetTool("test", "tool")
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestMCPTool_Structure(t *testing.T) {
	// Test that MCPTool can be created
	desc := "Test tool"
	tool := &MCPTool{
		ServerID: "test-server",
		Tool: types.Tool{
			Name:        "test_tool",
			Description: &desc,
			InputSchema: json.RawMessage(`{"type":"object"}`),
		},
		Client: nil, // Would be a real client in practice
	}

	if tool.ServerID != "test-server" {
		t.Errorf("ServerID = %s, want test-server", tool.ServerID)
	}
	if tool.Tool.Name != "test_tool" {
		t.Errorf("Tool.Name = %s, want test_tool", tool.Tool.Name)
	}
}

func TestMCPConfig_Structure(t *testing.T) {
	// Test that MCPConfig can be created with all fields
	cfg := MCPConfig{
		Command: "npx",
		Args:    []string{"-y", "server"},
		Env: map[string]string{
			"KEY": "value",
		},
	}

	if cfg.Command != "npx" {
		t.Errorf("Command = %s, want npx", cfg.Command)
	}
	if len(cfg.Args) != 2 {
		t.Errorf("len(Args) = %d, want 2", len(cfg.Args))
	}
	if cfg.Env["KEY"] != "value" {
		t.Errorf("Env[KEY] = %s, want value", cfg.Env["KEY"])
	}
}
