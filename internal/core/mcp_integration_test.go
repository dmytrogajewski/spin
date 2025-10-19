package core

import (
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/llm"
)

func TestMCPIntegration(t *testing.T) {
	// Create config with MCP enabled but no servers (to avoid hanging)
	cfg := DefaultConfig()
	cfg.Provider = "mock"
	cfg.Model = "test-model"
	cfg.MaxTurns = 10
	cfg.Timeout = 5 * time.Minute
	cfg.MaxTokens = 1000
	cfg.RequireApproval = false
	cfg.EnableMCP = true
	cfg.MCPServers = []MCPServerConfig{} // Empty servers list

	// Create manager
	mgr, err := NewManager(cfg, WithLLM(llm.NewMockProvider("test")))
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer mgr.Close()

	// Verify MCP manager was created
	if mgr.mcpManager == nil {
		t.Fatal("MCP manager should be initialized when EnableMCP is true")
	}

	// Verify MCP manager is not connected (no servers)
	if mgr.mcpManager.IsConnected() {
		t.Fatal("MCP manager should not be connected without servers")
	}

	// Verify no MCP tools are available (no servers)
	mcpTools := mgr.mcpManager.GetTools()
	if len(mcpTools) != 0 {
		t.Fatalf("Expected no MCP tools without servers, got %d", len(mcpTools))
	}

	t.Logf("MCP integration successful: manager initialized but no servers configured")
}

func TestMCPDisabled(t *testing.T) {
	// Create config with MCP disabled
	cfg := DefaultConfig()
	cfg.Provider = "mock"
	cfg.Model = "test-model"
	cfg.MaxTurns = 10
	cfg.Timeout = 5 * time.Minute
	cfg.MaxTokens = 1000
	cfg.RequireApproval = false
	cfg.EnableMCP = false

	// Create manager
	mgr, err := NewManager(cfg, WithLLM(llm.NewMockProvider("test")))
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer mgr.Close()

	// Verify MCP manager was not created
	if mgr.mcpManager != nil {
		t.Fatal("MCP manager should not be initialized when EnableMCP is false")
	}
}

func TestMCPToolExecution(t *testing.T) {
	t.Skip("Skipping MCP tool execution test - requires real MCP server")
	// This test would require a real MCP server implementation
	// For now, we skip it to avoid hanging tests
}
