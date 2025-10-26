package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/dmytrogajewski/spin/internal/tools"
	"github.com/mark3labs/mcp-go/client"
	mcpSDK "github.com/mark3labs/mcp-go/mcp"
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
	clients   map[string]*client.Client
	tools     map[string]*MCPTool
	mu        sync.RWMutex
	closed    bool
	closeOnce sync.Once
}

// MCPTool wraps an MCP tool for integration with the tool registry.
type MCPTool struct {
	ServerName string
	Tool       mcpSDK.Tool
	Client     *client.Client
}

// NewMCPManager creates a new MCP manager.
func NewMCPManager(config *Config, logger *slog.Logger) *MCPManager {
	return &MCPManager{
		config:  config,
		logger:  logger,
		clients: make(map[string]*client.Client),
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

	// Create MCP client using SDK (SDK auto-starts the connection)
	mcpClient, err := m.createSDKClient(serverConfig)
	if err != nil {
		return fmt.Errorf("failed to create MCP client: %w", err)
	}

	// Initialize connection
	initReq := mcpSDK.InitializeRequest{
		Params: mcpSDK.InitializeParams{
			ProtocolVersion: "2024-11-05",
			Capabilities:    mcpSDK.ClientCapabilities{},
			ClientInfo: mcpSDK.Implementation{
				Name:    "spin",
				Version: "0.1.0",
			},
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
	listReq := mcpSDK.ListToolsRequest{}
	toolsResp, err := mcpClient.ListTools(ctx, listReq)
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

// createSDKClient creates an MCP client using the mark3labs/mcp-go SDK.
func (m *MCPManager) createSDKClient(config MCPServerConfig) (*client.Client, error) {
	// Convert env map to slice of KEY=VALUE strings
	env := make([]string, 0, len(config.Env))
	for k, v := range config.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	// Create SDK stdio client
	return client.NewStdioMCPClient(
		config.Command,
		env,
		config.Args...,
	)
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
func (m *MCPManager) CallTool(ctx context.Context, toolName string, arguments json.RawMessage) (tools.ToolResult, error) {
	m.mu.RLock()
	mcpTool, exists := m.tools[toolName]
	m.mu.RUnlock()

	if !exists {
		return tools.ToolResult{}, fmt.Errorf("mcp tool not found: %s", toolName)
	}

	// Parse arguments into map for SDK
	var argsMap map[string]any
	if len(arguments) > 0 {
		if err := json.Unmarshal(arguments, &argsMap); err != nil {
			return tools.ToolResult{
				Success: false,
				Error:   fmt.Sprintf("invalid arguments: %v", err),
			}, nil
		}
	}

	// Build SDK request
	callReq := mcpSDK.CallToolRequest{
		Params: mcpSDK.CallToolParams{
			Name:      mcpTool.Tool.Name,
			Arguments: argsMap,
		},
	}

	// Call the MCP tool
	resp, err := mcpTool.Client.CallTool(ctx, callReq)
	if err != nil {
		return tools.ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	// Check if tool call resulted in error
	if resp.IsError {
		return tools.ToolResult{
			Success: false,
			Error:   "tool execution failed",
		}, nil
	}

	// Convert MCP response to tool result
	var output strings.Builder
	for _, content := range resp.Content {
		// Try to cast to TextContent
		if textContent, ok := mcpSDK.AsTextContent(content); ok {
			output.WriteString(textContent.Text)
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

		m.clients = make(map[string]*client.Client)
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

func getToolDescription(tool mcpSDK.Tool) string {
	if tool.Description != "" {
		return tool.Description
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

// JSONSchemaProperty represents a property in a JSON Schema.
type JSONSchemaProperty struct {
	Type        string `json:"type,omitempty"`
	Description string `json:"description,omitempty"`
}

// JSONSchema represents a simplified JSON Schema for tool parameters.
type JSONSchema struct {
	Type       string                        `json:"type,omitempty"`
	Properties map[string]JSONSchemaProperty `json:"properties,omitempty"`
	Required   []string                      `json:"required,omitempty"`
}

func (w *MCPToolWrapper) Schema() tools.ToolSchema {
	// Marshal tool's InputSchema to JSON for parsing
	schemaBytes, err := json.Marshal(w.mcpTool.Tool.InputSchema)
	if err != nil {
		return w.fallbackSchema()
	}

	// Parse as structured JSON Schema
	var mcpSchema JSONSchema
	if err := json.Unmarshal(schemaBytes, &mcpSchema); err != nil {
		return w.fallbackSchema()
	}

	// Convert properties to tool schema format
	properties := make(map[string]tools.PropertyDefinition)
	for name, prop := range mcpSchema.Properties {
		properties[name] = tools.PropertyDefinition{
			Type:        prop.Type,
			Description: prop.Description,
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
				Required:   mcpSchema.Required,
			},
		},
	}
}

// fallbackSchema returns a basic schema when parsing fails.
func (w *MCPToolWrapper) fallbackSchema() tools.ToolSchema {
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

func (w *MCPToolWrapper) Execute(ctx context.Context, params tools.ToolParameters) (tools.ToolResult, error) {
	// Convert ToolParameters to json.RawMessage
	argsJSON, err := json.Marshal(params.ToMap())
	if err != nil {
		return tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("marshal arguments: %v", err),
		}, nil
	}
	return w.manager.CallTool(ctx, w.name, argsJSON)
}
