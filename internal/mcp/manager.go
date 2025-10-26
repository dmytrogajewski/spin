package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/dmytrogajewski/spin/internal/mcp/client"
	"github.com/dmytrogajewski/spin/internal/mcp/types"
	"github.com/dmytrogajewski/spin/internal/tools"
)

// Config holds MCP manager configuration.
type Config struct {
	EnableMCP  bool
	MCPServers []MCPServerConfig
}

// ServerConfig holds configuration for a single MCP server.
type ServerConfig struct {
	Command string
	Args    []string
	Env     map[string]string
}

// MCPServerConfig holds configuration for a single MCP server.
type MCPServerConfig struct {
	Name    string
	Command string
	Args    []string
	Env     map[string]string
}

// MCPManager manages MCP server connections and tool registration.
type MCPManager struct {
	config    *Config
	logger    *slog.Logger
	clients   map[string]client.Client
	tools     map[string]*MCPTool
	mu        sync.RWMutex
	closed    bool
	closeOnce sync.Once
}

// MCPTool wraps an MCP tool for integration with the tool registry.
type MCPTool struct {
	ServerName string
	Tool       types.Tool
	Client     client.Client
}

// NewMCPManager creates a new MCP manager.
func NewMCPManager(config *Config, logger *slog.Logger) *MCPManager {
	return &MCPManager{
		config:  config,
		logger:  logger,
		clients: make(map[string]client.Client),
		tools:   make(map[string]*MCPTool),
	}
}

// Initialize connects to all configured MCP servers and registers their tools.
func (m *MCPManager) Initialize(ctx context.Context) error {
	if !m.config.EnableMCP {
		m.logger.Debug("MCP disabled, skipping initialization")
		return nil
	}

	if len(m.config.MCPServers) == 0 {
		m.logger.Debug("No MCP servers configured")
		return nil
	}

	m.logger.Info("Initializing MCP servers", "count", len(m.config.MCPServers))

	for _, serverConfig := range m.config.MCPServers {
		if err := m.connectServer(ctx, serverConfig); err != nil {
			m.logger.Error("Failed to connect to MCP server",
				"server", serverConfig.Name,
				"error", err)
			// Continue with other servers
			continue
		}
	}

	m.logger.Info("MCP initialization complete",
		"servers", len(m.clients),
		"tools", len(m.tools))

	return nil
}

// connectServer connects to a single MCP server and registers its tools.
func (m *MCPManager) connectServer(ctx context.Context, serverConfig MCPServerConfig) error {
	m.logger.Debug("Connecting to MCP server", "server", serverConfig.Name)

	// Create client config
	clientConfig := client.Config{
		Command: serverConfig.Command,
		Args:    serverConfig.Args,
		Env:     serverConfig.Env,
		Timeout: 30 * time.Second,
	}

	// Create MCP client using stdio transport
	mcpClient, err := m.createClient(clientConfig)
	if err != nil {
		return fmt.Errorf("failed to create MCP client: %w", err)
	}

	// Initialize connection
	initReq := types.InitializeRequest{
		ProtocolVersion: "2024-11-05",
		Capabilities: types.ClientCapabilities{
			Tools: &types.ToolsCapability{
				ListChanged: true,
			},
		},
		ClientInfo: types.Implementation{
			Name:    "spin",
			Version: "0.1.0",
		},
	}

	initResp, err := mcpClient.Initialize(ctx, initReq)
	if err != nil {
		mcpClient.Close()
		return fmt.Errorf("failed to initialize MCP connection: %w", err)
	}

	m.logger.Debug("MCP server initialized",
		"server", serverConfig.Name,
		"protocol", initResp.ProtocolVersion,
		"capabilities", initResp.Capabilities)

	// List available tools
	toolsResp, err := mcpClient.ListTools(ctx)
	if err != nil {
		mcpClient.Close()
		return fmt.Errorf("failed to list MCP tools: %w", err)
	}

	// Register tools
	m.mu.Lock()
	m.clients[serverConfig.Name] = mcpClient

	for _, tool := range toolsResp.Tools {
		toolKey := fmt.Sprintf("mcp_%s_%s", serverConfig.Name, tool.Name)
		m.tools[toolKey] = &MCPTool{
			ServerName: serverConfig.Name,
			Tool:       tool,
			Client:     mcpClient,
		}
	}
	m.mu.Unlock()

	m.logger.Info("MCP server connected",
		"server", serverConfig.Name,
		"tools", len(toolsResp.Tools))

	return nil
}

// createClient creates an MCP client.
func (m *MCPManager) createClient(config client.Config) (client.Client, error) {
	// Use real stdio client
	return client.NewStdioClient(config)
}

// GetTools returns all registered MCP tools as tool registry entries.
func (m *MCPManager) GetTools() []tools.Tool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]tools.Tool, 0, len(m.tools))
	for key, mcpTool := range m.tools {
		tool := &MCPToolWrapper{
			name:        key,
			description: getToolDescription(mcpTool.Tool),
			mcpTool:     mcpTool,
			manager:     m,
		}

		result = append(result, tool)
	}

	return result
}

// CallTool invokes an MCP tool.
func (m *MCPManager) CallTool(ctx context.Context, toolName string, arguments map[string]interface{}) (tools.ToolResult, error) {
	m.mu.RLock()
	mcpTool, exists := m.tools[toolName]
	m.mu.RUnlock()

	if !exists {
		return tools.ToolResult{}, fmt.Errorf("mcp tool not found: %s", toolName)
	}

	// Convert arguments to JSON
	argsJSON, err := json.Marshal(arguments)
	if err != nil {
		return tools.ToolResult{}, fmt.Errorf("failed to marshal arguments: %w", err)
	}

	// Call the MCP tool
	resp, err := mcpTool.Client.CallTool(ctx, mcpTool.Tool.Name, argsJSON)
	if err != nil {
		return tools.ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	// Convert MCP response to tool result
	var output strings.Builder
	for _, content := range resp.Content {
		if content.Text != nil {
			output.WriteString(*content.Text)
		}
	}

	return tools.ToolResult{
		Success: true,
		Output:  output.String(),
	}, nil
}

// Close closes all MCP connections.
func (m *MCPManager) Close() error {
	var err error
	m.closeOnce.Do(func() {
		m.mu.Lock()
		m.closed = true

		for name, client := range m.clients {
			if closeErr := client.Close(); closeErr != nil {
				m.logger.Error("Failed to close MCP client",
					"server", name,
					"error", closeErr)
				if err == nil {
					err = closeErr
				}
			}
		}

		m.clients = make(map[string]client.Client)
		m.tools = make(map[string]*MCPTool)
		m.mu.Unlock()

		m.logger.Info("MCP manager closed")
	})

	return err
}

// IsConnected returns true if any MCP servers are connected.
func (m *MCPManager) IsConnected() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.clients) > 0
}

// GetConnectedServers returns a list of connected server names.
func (m *MCPManager) GetConnectedServers() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	servers := make([]string, 0, len(m.clients))
	for name := range m.clients {
		servers = append(servers, name)
	}
	return servers
}

// Helper functions

func getToolDescription(tool types.Tool) string {
	if tool.Description != nil {
		return *tool.Description
	}
	return fmt.Sprintf("MCP tool: %s", tool.Name)
}

// MCPToolWrapper wraps an MCP tool to implement the tools.Tool interface.
type MCPToolWrapper struct {
	name        string
	description string
	mcpTool     *MCPTool
	manager     *MCPManager
}

func (w *MCPToolWrapper) Name() string {
	return w.name
}

func (w *MCPToolWrapper) Description() string {
	return w.description
}

func (w *MCPToolWrapper) Schema() tools.ToolSchema {
	// Convert MCP tool schema to OpenAI-compatible schema
	var mcpSchema map[string]interface{}
	if err := json.Unmarshal(w.mcpTool.Tool.InputSchema, &mcpSchema); err != nil {
		// Fallback to basic schema
		return tools.ToolSchema{
			Type: "function",
			Function: tools.FunctionSchema{
				Name:        w.name,
				Description: w.description,
				Parameters: tools.ParameterSchema{
					Type:       "object",
					Properties: make(map[string]tools.PropertyDefinition),
					Required:   []string{},
				},
			},
		}
	}

	// Convert properties
	properties := make(map[string]tools.PropertyDefinition)
	if props, ok := mcpSchema["properties"].(map[string]interface{}); ok {
		for name, prop := range props {
			if propMap, ok := prop.(map[string]interface{}); ok {
				desc := ""
				if descVal, ok := propMap["description"].(string); ok {
					desc = descVal
				}
				propType := "string"
				if typeVal, ok := propMap["type"].(string); ok {
					propType = typeVal
				}
				properties[name] = tools.PropertyDefinition{
					Type:        propType,
					Description: desc,
				}
			}
		}
	}

	// Convert required fields
	var required []string
	if req, ok := mcpSchema["required"].([]interface{}); ok {
		for _, reqItem := range req {
			if reqStr, ok := reqItem.(string); ok {
				required = append(required, reqStr)
			}
		}
	}

	return tools.ToolSchema{
		Type: "function",
		Function: tools.FunctionSchema{
			Name:        w.name,
			Description: w.description,
			Parameters: tools.ParameterSchema{
				Type:       "object",
				Properties: properties,
				Required:   required,
			},
		},
	}
}

func (w *MCPToolWrapper) Execute(ctx context.Context, params tools.ToolParameters) (tools.ToolResult, error) {
	// Convert ToolParameters back to map for MCP call
	paramsMap := params.ToMap()
	return w.manager.CallTool(ctx, w.name, paramsMap)
}
