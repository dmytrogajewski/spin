package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	mcpSDK "github.com/mark3labs/mcp-go/mcp"

	"github.com/dmytrogajewski/spin/internal/tools"
)

// baseRegistry contains the shared fields and methods for all MCP registry types.
// Embed this struct and provide an Initialize method to create a concrete registry.
type baseRegistry struct {
	name      string
	mcpClient Client
	tools     map[string]*Tool
	metadata  RegistryMetadata
	mu        sync.RWMutex
	connected bool
}

// Name returns the registry name.
func (b *baseRegistry) Name() string {
	return b.name
}

// IsConnected returns true if the registry is connected.
func (b *baseRegistry) IsConnected() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return b.connected
}

// Metadata returns registry metadata.
func (b *baseRegistry) Metadata() RegistryMetadata {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return b.metadata
}

// Client returns the underlying MCP client.
func (b *baseRegistry) Client() Client {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return b.mcpClient
}

// List returns all tools from this registry.
func (b *baseRegistry) List() []tools.Tool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	result := make([]tools.Tool, 0, len(b.tools))
	for _, mcpTool := range b.tools {
		result = append(result, b.wrapTool(mcpTool))
	}

	return result
}

// Count returns the number of tools.
func (b *baseRegistry) Count() int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return len(b.tools)
}

// Search finds tools matching the query.
func (b *baseRegistry) Search(_ context.Context, _ *SearchContext, query string, maxResults int) []tools.Tool {
	return SearchTools(b.List(), query, maxResults, DefaultSearchOptions())
}

// Tool returns a specific tool by name.
func (b *baseRegistry) Tool(name string) tools.Tool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	mcpTool, exists := b.tools[name]
	if !exists {
		return nil
	}

	return b.wrapTool(mcpTool)
}

// Execute calls a tool with the given arguments.
func (b *baseRegistry) Execute(ctx context.Context, toolName string, args json.RawMessage) (tools.ToolResult, error) {
	b.mu.RLock()
	mcpTool, exists := b.tools[toolName]
	mcpClient := b.mcpClient
	b.mu.RUnlock()

	if !exists {
		return tools.ToolResult{}, fmt.Errorf("tool not found: %s: %w", toolName, tools.ErrToolNotFound)
	}

	return executeMCPTool(ctx, mcpClient, mcpTool, args)
}

// Close closes the registry and releases resources.
func (b *baseRegistry) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.connected {
		return nil
	}

	b.connected = false
	b.metadata.Connected = false

	if b.mcpClient != nil {
		return b.mcpClient.Close()
	}

	return nil
}

// applyHandshakeResult copies metadata and tools from an MCP handshake into the registry.
// The caller must hold the write lock.
func (b *baseRegistry) applyHandshakeResult(meta RegistryMetadata, toolsMap map[string]*Tool) {
	b.metadata.ServerInfo = meta.ServerInfo
	b.metadata.Capabilities = meta.Capabilities
	b.tools = toolsMap
	b.metadata.ToolCount = len(b.tools)
	b.metadata.Connected = true
	b.connected = true
}

// wrapTool wraps an MCP Tool as a tools.Tool with qualified name.
func (b *baseRegistry) wrapTool(mcpTool *Tool) tools.Tool {
	return &mcpToolWrapper{
		registryName: b.name,
		mcpTool:      mcpTool,
		executeFunc:  b.Execute,
	}
}

// mcpToolWrapper is the unified tool wrapper for all registry types.
// It wraps an MCP Tool to implement tools.Tool.
type mcpToolWrapper struct {
	registryName string
	mcpTool      *Tool
	executeFunc  func(ctx context.Context, toolName string, args json.RawMessage) (tools.ToolResult, error)
}

// Name implements the Name operation.
func (w *mcpToolWrapper) Name() string {
	return fmt.Sprintf("mcp_%s_%s", w.registryName, w.mcpTool.Tool.Name)
}

// Description implements the Description operation.
func (w *mcpToolWrapper) Description() string {
	return toolDescription(w.mcpTool)
}

// Schema implements the Schema operation.
func (w *mcpToolWrapper) Schema() tools.ToolSchema {
	return buildToolSchema(w, w.mcpTool)
}

// Execute implements the Execute operation.
func (w *mcpToolWrapper) Execute(ctx context.Context, params tools.ToolParameters) (tools.ToolResult, error) {
	argsJSON, err := json.Marshal(params.ToMap())
	if err != nil {
		return tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("marshal arguments: %v", err),
		}, nil
	}

	return w.executeFunc(ctx, w.mcpTool.Tool.Name, argsJSON)
}

// initializeMCPConnection performs the MCP initialize handshake and discovers tools.
// It sends the initialize request, stores server info, lists tools, and populates the tools map.
// The caller must hold the lock and set connected/metadata afterwards.
func initializeMCPConnection(ctx context.Context, mcpClient Client, serverName string) (RegistryMetadata, map[string]*Tool, error) {
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
		return RegistryMetadata{}, nil, fmt.Errorf("initialize connection: %w", err)
	}

	meta := RegistryMetadata{
		ServerInfo:   &initResp.ServerInfo,
		Capabilities: initResp.Capabilities,
	}

	toolsResp, err := mcpClient.ListTools(ctx, mcpSDK.ListToolsRequest{})
	if err != nil {
		return RegistryMetadata{}, nil, fmt.Errorf("list tools: %w", err)
	}

	toolsMap := make(map[string]*Tool, len(toolsResp.Tools))
	for _, tool := range toolsResp.Tools {
		toolsMap[tool.Name] = &Tool{
			ServerName: serverName,
			Tool:       tool,
			Client:     mcpClient,
		}
	}

	return meta, toolsMap, nil
}
