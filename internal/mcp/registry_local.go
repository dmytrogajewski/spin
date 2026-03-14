package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"github.com/mark3labs/mcp-go/client"
	mcpSDK "github.com/mark3labs/mcp-go/mcp"

	"github.com/dmytrogajewski/spin/internal/tools"
)

// LocalRegistryConfig holds configuration for a local stdio MCP registry.
type LocalRegistryConfig struct {
	Name    string
	Command string
	Args    []string
	Env     map[string]string
	Logger  *slog.Logger
}

// LocalRegistry wraps a local stdio MCP server as an Registry.
type LocalRegistry struct {
	name      string
	config    LocalRegistryConfig
	sdkClient *client.Client
	mcpClient Client
	tools     map[string]*Tool
	metadata  RegistryMetadata
	logger    *slog.Logger
	mu        sync.RWMutex
	connected bool
}

// NewLocalRegistry creates a new LocalRegistry for stdio MCP servers.
func NewLocalRegistry(config LocalRegistryConfig) (*LocalRegistry, error) {
	if config.Name == "" {
		return nil, ErrRegistryNameRequired
	}

	if config.Command == "" {
		return nil, ErrCommandRequiredForLocalRegistry
	}

	return &LocalRegistry{
		name:   config.Name,
		config: config,
		tools:  make(map[string]*Tool),
		logger: config.Logger,
		metadata: RegistryMetadata{
			Name: config.Name,
			Type: "local",
		},
	}, nil
}

// Name returns the registry name.
func (r *LocalRegistry) Name() string {
	return r.name
}

// Initialize connects to the MCP server and discovers tools.
func (r *LocalRegistry) Initialize(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.connected {
		return nil
	}

	// Build environment slice.
	var env []string
	for k, v := range r.config.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	// Create stdio client.
	sdkClient, err := client.NewStdioMCPClient(r.config.Command, env, r.config.Args...)
	if err != nil {
		return fmt.Errorf("create stdio client: %w", err)
	}

	r.sdkClient = sdkClient
	r.mcpClient = &sdkClientWrapper{client: sdkClient}

	// Initialize connection.
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

	initResp, err := r.mcpClient.Initialize(ctx, initReq)
	if err != nil {
		r.mcpClient.Close()

		return fmt.Errorf("initialize connection: %w", err)
	}

	r.metadata.ServerInfo = &initResp.ServerInfo
	r.metadata.Capabilities = initResp.Capabilities

	// List tools.
	listReq := mcpSDK.ListToolsRequest{}

	toolsResp, err := r.mcpClient.ListTools(ctx, listReq)
	if err != nil {
		r.mcpClient.Close()

		return fmt.Errorf("list tools: %w", err)
	}

	// Register tools.
	for _, tool := range toolsResp.Tools {
		r.tools[tool.Name] = &Tool{
			ServerName: r.name,
			Tool:       tool,
			Client:     r.mcpClient,
		}
	}

	r.metadata.ToolCount = len(r.tools)
	r.metadata.Connected = true
	r.connected = true

	if r.logger != nil {
		r.logger.InfoContext(ctx, "local registry initialized",
			"name", r.name,
			"tools", len(r.tools))
	}

	return nil
}

// IsConnected returns true if the registry is connected.
func (r *LocalRegistry) IsConnected() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.connected
}

// Metadata returns registry metadata.
func (r *LocalRegistry) Metadata() RegistryMetadata {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.metadata
}

// Client returns the underlying MCP client.
func (r *LocalRegistry) Client() Client {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.mcpClient
}

// List returns all tools from this registry.
func (r *LocalRegistry) List() []tools.Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]tools.Tool, 0, len(r.tools))
	for _, mcpTool := range r.tools {
		result = append(result, r.wrapTool(mcpTool))
	}

	return result
}

// Count returns the number of tools.
func (r *LocalRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.tools)
}

// Search finds tools matching the query.
func (r *LocalRegistry) Search(_ context.Context, _ *SearchContext, query string, maxResults int) []tools.Tool {
	return SearchTools(r.List(), query, maxResults, DefaultSearchOptions())
}

// Tool returns a specific tool by name.
func (r *LocalRegistry) Tool(name string) tools.Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	mcpTool, exists := r.tools[name]
	if !exists {
		return nil
	}

	return r.wrapTool(mcpTool)
}

// Execute calls a tool with the given arguments.
func (r *LocalRegistry) Execute(ctx context.Context, toolName string, args json.RawMessage) (tools.ToolResult, error) {
	r.mu.RLock()
	mcpTool, exists := r.tools[toolName]
	mcpClient := r.mcpClient
	r.mu.RUnlock()

	if !exists {
		return tools.ToolResult{}, fmt.Errorf("tool not found: %s: %w", toolName, tools.ErrToolNotFound)
	}

	return executeMCPTool(ctx, mcpClient, mcpTool, args)
}

// Close closes the registry and releases resources.
func (r *LocalRegistry) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.connected {
		return nil
	}

	r.connected = false
	r.metadata.Connected = false

	if r.mcpClient != nil {
		return r.mcpClient.Close()
	}

	return nil
}

// wrapTool wraps an Tool as a tools.Tool with qualified name.
func (r *LocalRegistry) wrapTool(mcpTool *Tool) tools.Tool {
	return &registryToolWrapper{
		registry: r,
		mcpTool:  mcpTool,
	}
}

// registryToolWrapper wraps an Tool to implement tools.Tool.
type registryToolWrapper struct {
	registry *LocalRegistry
	mcpTool  *Tool
}

// Name implements the Name operation.
func (w *registryToolWrapper) Name() string {
	return fmt.Sprintf("mcp_%s_%s", w.registry.name, w.mcpTool.Tool.Name)
}

// Description implements the Description operation.
func (w *registryToolWrapper) Description() string {
	return toolDescription(w.mcpTool)
}

// Schema implements the Schema operation.
func (w *registryToolWrapper) Schema() tools.ToolSchema {
	return buildToolSchema(w, w.mcpTool)
}

// Execute implements the Execute operation.
func (w *registryToolWrapper) Execute(ctx context.Context, params tools.ToolParameters) (tools.ToolResult, error) {
	argsJSON, err := json.Marshal(params.ToMap())
	if err != nil {
		return tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("marshal arguments: %v", err),
		}, nil
	}

	return w.registry.Execute(ctx, w.mcpTool.Tool.Name, argsJSON)
}
