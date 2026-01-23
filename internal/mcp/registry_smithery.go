package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/dmytrogajewski/spin/internal/tools"
	mcpSDK "github.com/mark3labs/mcp-go/mcp"
)

// SmitheryRegistryConfig holds configuration for a Smithery MCP registry.
type SmitheryRegistryConfig struct {
	Name      string
	APIKey    string
	MCPURL    string
	Namespace string
	Logger    *slog.Logger
}

// SmitheryRegistry wraps a Smithery-hosted MCP server as an MCPRegistry.
type SmitheryRegistry struct {
	name      string
	config    SmitheryRegistryConfig
	client    *SmitheryClient
	tools     map[string]*MCPTool
	metadata  RegistryMetadata
	logger    *slog.Logger
	mu        sync.RWMutex
	connected bool
}

// NewSmitheryRegistry creates a new SmitheryRegistry.
func NewSmitheryRegistry(config SmitheryRegistryConfig) (*SmitheryRegistry, error) {
	if config.Name == "" {
		return nil, fmt.Errorf("registry name is required")
	}
	if config.APIKey == "" {
		return nil, fmt.Errorf("API key is required for smithery registry")
	}
	if config.MCPURL == "" {
		return nil, fmt.Errorf("MCP URL is required for smithery registry")
	}
	if config.Namespace == "" {
		return nil, fmt.Errorf("namespace is required for smithery registry")
	}

	return &SmitheryRegistry{
		name:   config.Name,
		config: config,
		tools:  make(map[string]*MCPTool),
		logger: config.Logger,
		metadata: RegistryMetadata{
			Name: config.Name,
			Type: "smithery",
		},
	}, nil
}

// Name returns the registry name.
func (r *SmitheryRegistry) Name() string {
	return r.name
}

// Initialize connects to the Smithery server and discovers tools.
func (r *SmitheryRegistry) Initialize(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.connected {
		return nil
	}

	// Create Smithery client
	smitheryClient, err := NewSmitheryClient(SmitheryConfig{
		APIKey:    r.config.APIKey,
		MCPURL:    r.config.MCPURL,
		Namespace: r.config.Namespace,
		Logger:    r.logger,
	})
	if err != nil {
		return fmt.Errorf("create smithery client: %w", err)
	}

	r.client = smitheryClient

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

	initResp, err := r.client.Initialize(ctx, initReq)
	if err != nil {
		r.client.Close()
		return fmt.Errorf("initialize connection: %w", err)
	}

	r.metadata.ServerInfo = &initResp.ServerInfo
	r.metadata.Capabilities = initResp.Capabilities

	// List tools
	listReq := mcpSDK.ListToolsRequest{}
	toolsResp, err := r.client.ListTools(ctx, listReq)
	if err != nil {
		r.client.Close()
		return fmt.Errorf("list tools: %w", err)
	}

	// Register tools
	for _, tool := range toolsResp.Tools {
		r.tools[tool.Name] = &MCPTool{
			ServerName: r.name,
			Tool:       tool,
			Client:     r.client,
		}
	}

	r.metadata.ToolCount = len(r.tools)
	r.metadata.Connected = true
	r.connected = true

	if r.logger != nil {
		r.logger.Info("smithery registry initialized",
			"name", r.name,
			"namespace", r.config.Namespace,
			"tools", len(r.tools))
	}

	return nil
}

// IsConnected returns true if the registry is connected.
func (r *SmitheryRegistry) IsConnected() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.connected
}

// Metadata returns registry metadata.
func (r *SmitheryRegistry) Metadata() RegistryMetadata {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.metadata
}

// Client returns the underlying MCP client.
func (r *SmitheryRegistry) Client() MCPClient {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.client
}

// List returns all tools from this registry.
func (r *SmitheryRegistry) List() []tools.Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]tools.Tool, 0, len(r.tools))
	for _, mcpTool := range r.tools {
		result = append(result, r.wrapTool(mcpTool))
	}
	return result
}

// Count returns the number of tools.
func (r *SmitheryRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.tools)
}

// Search finds tools matching the query.
func (r *SmitheryRegistry) Search(query string, max int) []tools.Tool {
	return SearchTools(r.List(), query, max, DefaultSearchOptions())
}

// Tool returns a specific tool by name.
func (r *SmitheryRegistry) Tool(name string) tools.Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	mcpTool, exists := r.tools[name]
	if !exists {
		return nil
	}
	return r.wrapTool(mcpTool)
}

// Execute calls a tool with the given arguments.
func (r *SmitheryRegistry) Execute(ctx context.Context, toolName string, args json.RawMessage) (tools.ToolResult, error) {
	r.mu.RLock()
	mcpTool, exists := r.tools[toolName]
	client := r.client
	r.mu.RUnlock()

	if !exists {
		return tools.ToolResult{}, fmt.Errorf("tool not found: %s", toolName)
	}

	// Parse arguments
	var argsMap map[string]any
	if len(args) > 0 {
		if err := json.Unmarshal(args, &argsMap); err != nil {
			return tools.ToolResult{
				Success: false,
				Error:   fmt.Sprintf("invalid arguments: %v", err),
			}, nil
		}
	}

	// Call tool
	callReq := mcpSDK.CallToolRequest{
		Params: mcpSDK.CallToolParams{
			Name:      mcpTool.Tool.Name,
			Arguments: argsMap,
		},
	}

	resp, err := client.CallTool(ctx, callReq)
	if err != nil {
		return tools.ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	if resp.IsError {
		return tools.ToolResult{
			Success: false,
			Error:   "tool execution failed",
		}, nil
	}

	// Convert response
	var output strings.Builder
	for _, content := range resp.Content {
		if textContent, ok := mcpSDK.AsTextContent(content); ok {
			output.WriteString(textContent.Text)
		}
	}

	return tools.ToolResult{
		Success: true,
		Output:  output.String(),
	}, nil
}

// Close closes the registry and releases resources.
func (r *SmitheryRegistry) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.connected {
		return nil
	}

	r.connected = false
	r.metadata.Connected = false

	if r.client != nil {
		return r.client.Close()
	}
	return nil
}

// wrapTool wraps an MCPTool as a tools.Tool with qualified name.
func (r *SmitheryRegistry) wrapTool(mcpTool *MCPTool) tools.Tool {
	return &smitheryToolWrapper{
		registry: r,
		mcpTool:  mcpTool,
	}
}

// smitheryToolWrapper wraps an MCPTool to implement tools.Tool.
type smitheryToolWrapper struct {
	registry *SmitheryRegistry
	mcpTool  *MCPTool
}

func (w *smitheryToolWrapper) Name() string {
	return fmt.Sprintf("mcp_%s_%s", w.registry.name, w.mcpTool.Tool.Name)
}

func (w *smitheryToolWrapper) Description() string {
	if w.mcpTool.Tool.Description != "" {
		return w.mcpTool.Tool.Description
	}
	return fmt.Sprintf("MCP tool: %s", w.mcpTool.Tool.Name)
}

func (w *smitheryToolWrapper) Schema() tools.ToolSchema {
	schemaBytes, err := json.Marshal(w.mcpTool.Tool.InputSchema)
	if err != nil {
		return w.fallbackSchema()
	}

	var mcpSchema JSONSchema
	if err := json.Unmarshal(schemaBytes, &mcpSchema); err != nil {
		return w.fallbackSchema()
	}

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
			Name:        w.Name(),
			Description: w.Description(),
			Parameters: tools.ParameterSchema{
				Type:       "object",
				Properties: properties,
				Required:   mcpSchema.Required,
			},
		},
	}
}

func (w *smitheryToolWrapper) fallbackSchema() tools.ToolSchema {
	return tools.ToolSchema{
		Type: "function",
		Function: tools.FunctionSchema{
			Name:        w.Name(),
			Description: w.Description(),
			Parameters: tools.ParameterSchema{
				Type:       "object",
				Properties: make(map[string]tools.PropertyDefinition),
				Required:   []string{},
			},
		},
	}
}

func (w *smitheryToolWrapper) Execute(ctx context.Context, params tools.ToolParameters) (tools.ToolResult, error) {
	argsJSON, err := json.Marshal(params.ToMap())
	if err != nil {
		return tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("marshal arguments: %v", err),
		}, nil
	}
	return w.registry.Execute(ctx, w.mcpTool.Tool.Name, argsJSON)
}
