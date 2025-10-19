package core

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/dmytrogajewski/spin/internal/mcp/client"
	"github.com/dmytrogajewski/spin/internal/mcp/types"
)

func TestMCPManager_GetConnectedServers(t *testing.T) {
	mgr := &MCPManager{
		clients: make(map[string]client.Client),
		tools:   make(map[string]*MCPTool),
	}

	servers := mgr.GetConnectedServers()
	if servers == nil {
		t.Error("GetConnectedServers() returned nil")
	}
}

func TestMCPManager_Initialize_NoServers(t *testing.T) {
	ctx := context.Background()

	mgr := &MCPManager{
		config:  &Config{},
		logger:  slog.New(slog.NewTextHandler(os.Stderr, nil)),
		clients: make(map[string]client.Client),
		tools:   make(map[string]*MCPTool),
	}

	err := mgr.Initialize(ctx)
	if err != nil {
		t.Errorf("Initialize() error = %v, want nil for no servers", err)
	}
}

func TestMCPTool_Methods(t *testing.T) {
	desc := "test description"
	mockTool := types.Tool{
		Name:        "test_tool",
		Description: &desc,
	}

	tool := &MCPTool{
		ServerName: "test_server",
		Tool:       mockTool,
		Client:     nil,
	}

	// Test that the tool struct contains correct data
	if tool.ServerName != "test_server" {
		t.Errorf("ServerName = %v, want test_server", tool.ServerName)
	}
	if tool.Tool.Name != "test_tool" {
		t.Errorf("Tool.Name = %v, want test_tool", tool.Tool.Name)
	}
	if tool.Tool.Description == nil || *tool.Tool.Description != "test description" {
		t.Errorf("Tool.Description = %v, want 'test description'", tool.Tool.Description)
	}
}
