package core

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/dmytrogajewski/spin/internal/mcp/client"
	"github.com/dmytrogajewski/spin/internal/mcp/types"
)

// MCPManager manages connections to MCP servers and integrates their tools.
type MCPManager struct {
	mu      sync.RWMutex
	clients map[string]client.Client // server ID -> client
	tools   map[string]*MCPTool      // "serverID/toolName" -> tool
}

// MCPTool represents a tool from an MCP server.
type MCPTool struct {
	ServerID string        // MCP server ID
	Tool     types.Tool    // Tool definition
	Client   client.Client // Client to invoke the tool
}

// MCPConfig holds configuration for a single MCP server.
type MCPConfig struct {
	Command string            `yaml:"command"`
	Args    []string          `yaml:"args"`
	Env     map[string]string `yaml:"env"`
}

// NewMCPManager creates a new MCP manager.
func NewMCPManager() *MCPManager {
	return &MCPManager{
		clients: make(map[string]client.Client),
		tools:   make(map[string]*MCPTool),
	}
}

// ConnectServers connects to all configured MCP servers.
//
// For each server, it:
//  1. Creates and initializes an MCP client
//  2. Discovers available tools
//  3. Registers tools in the manager
//
// If any server fails to connect, an error is returned and successfully
// connected servers are left connected.
func (m *MCPManager) ConnectServers(ctx context.Context, configs map[string]MCPConfig) error {
	for serverID, cfg := range configs {
		if err := m.connectServer(ctx, serverID, cfg); err != nil {
			return fmt.Errorf("connect to %s: %w", serverID, err)
		}
	}
	return nil
}

// connectServer connects to a single MCP server.
func (m *MCPManager) connectServer(ctx context.Context, serverID string, cfg MCPConfig) error {
	// Create client config
	clientCfg := client.Config{
		Command: cfg.Command,
		Args:    cfg.Args,
		Env:     cfg.Env,
	}

	// Create client
	c, err := client.NewStdioClient(clientCfg)
	if err != nil {
		return err
	}

	// Initialize connection
	initReq := types.InitializeRequest{
		ProtocolVersion: "2024-11-05",
		Capabilities:    types.ClientCapabilities{},
		ClientInfo: types.Implementation{
			Name:    "spin",
			Version: "0.1.0",
		},
	}

	if _, err := c.Initialize(ctx, initReq); err != nil {
		c.Close()
		return fmt.Errorf("initialize: %w", err)
	}

	// Discover tools
	toolsResp, err := c.ListTools(ctx)
	if err != nil {
		c.Close()
		return fmt.Errorf("list tools: %w", err)
	}

	// Register client and tools
	m.mu.Lock()
	defer m.mu.Unlock()

	m.clients[serverID] = c

	for _, tool := range toolsResp.Tools {
		toolKey := fmt.Sprintf("%s/%s", serverID, tool.Name)
		m.tools[toolKey] = &MCPTool{
			ServerID: serverID,
			Tool:     tool,
			Client:   c,
		}
	}

	return nil
}

// CallTool invokes an MCP tool by server ID and tool name.
func (m *MCPManager) CallTool(ctx context.Context, serverID, toolName string, arguments json.RawMessage) (*types.CallToolResponse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	toolKey := fmt.Sprintf("%s/%s", serverID, toolName)
	mcpTool, ok := m.tools[toolKey]
	if !ok {
		return nil, fmt.Errorf("tool not found: %s", toolKey)
	}

	return mcpTool.Client.CallTool(ctx, toolName, arguments)
}

// ListAllTools returns all registered MCP tools.
func (m *MCPManager) ListAllTools() []*MCPTool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tools := make([]*MCPTool, 0, len(m.tools))
	for _, tool := range m.tools {
		tools = append(tools, tool)
	}
	return tools
}

// GetTool retrieves a specific tool by server ID and name.
func (m *MCPManager) GetTool(serverID, toolName string) (*MCPTool, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	toolKey := fmt.Sprintf("%s/%s", serverID, toolName)
	tool, ok := m.tools[toolKey]
	return tool, ok
}

// Close closes all MCP client connections.
func (m *MCPManager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errs []error
	for id, c := range m.clients {
		if err := c.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close %s: %w", id, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("close errors: %v", errs)
	}
	return nil
}

// ServerCount returns the number of connected MCP servers.
func (m *MCPManager) ServerCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.clients)
}

// ToolCount returns the total number of registered MCP tools.
func (m *MCPManager) ToolCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.tools)
}
