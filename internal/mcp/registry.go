package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	mcpSDK "github.com/mark3labs/mcp-go/mcp"

	"github.com/dmytrogajewski/spin/internal/tools"
)

// toolSchemaBuilder provides a named interface for building schemas from MCP tools.
type toolSchemaBuilder interface {
	Name() string
	Description() string
}

// buildToolSchema converts an MCP tool's InputSchema into a tools.ToolSchema.
// Falls back to an empty schema if the input schema cannot be parsed.
func buildToolSchema(wrapper toolSchemaBuilder, mcpTool *Tool) tools.ToolSchema {
	schemaBytes, err := json.Marshal(mcpTool.Tool.InputSchema)
	if err != nil {
		return buildFallbackSchema(wrapper)
	}

	var mcpSchema JSONSchema

	err = json.Unmarshal(schemaBytes, &mcpSchema)
	if err != nil {
		return buildFallbackSchema(wrapper)
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
			Name:        wrapper.Name(),
			Description: wrapper.Description(),
			Parameters: tools.ParameterSchema{
				Type:       "object",
				Properties: properties,
				Required:   mcpSchema.Required,
			},
		},
	}
}

// buildFallbackSchema returns a minimal schema when the MCP tool's input schema cannot be parsed.
func buildFallbackSchema(wrapper toolSchemaBuilder) tools.ToolSchema {
	return tools.ToolSchema{
		Type: "function",
		Function: tools.FunctionSchema{
			Name:        wrapper.Name(),
			Description: wrapper.Description(),
			Parameters: tools.ParameterSchema{
				Type:       "object",
				Properties: make(map[string]tools.PropertyDefinition),
				Required:   []string{},
			},
		},
	}
}

// toolDescription returns the description of an MCP tool, with a fallback to a generated name.
func toolDescription(mcpTool *Tool) string {
	if mcpTool.Tool.Description != "" {
		return mcpTool.Tool.Description
	}

	return fmt.Sprintf("MCP tool: %s", mcpTool.Tool.Name)
}


// RegistryMetadata contains information about a registry.
// Implementations can extend this with custom fields via the Extra map.
type RegistryMetadata struct {
	Name         string
	Type         string // Implementation-defined type identifier.
	ServerInfo   *mcpSDK.Implementation
	Capabilities mcpSDK.ServerCapabilities
	ToolCount    int
	Connected    bool
	Extra        map[string]any // Implementation-specific metadata.
}

// ToolSource is the minimal interface for any source of tools.
// All registries must implement at least this.
type ToolSource interface {
	// Name returns a unique identifier for this source.
	Name() string

	// Close releases resources held by this source.
	Close() error
}

// ToolLister can enumerate its available tools.
type ToolLister interface {
	ToolSource

	// List returns all tools from this source.
	List() []tools.Tool

	// Count returns the number of available tools.
	Count() int
}

// SearchContext provides optional context for tool search operations.
// Pass nil for simple searches without trajectory context.
type SearchContext struct {
	// TrajectoryContext provides execution context for relevance scoring (optional).
	TrajectoryContext interface {
		GetQuery() string
		GetRecentTools(n int) []string
	}

	// DynamicLoadout enables dynamic tool loading for this search (optional)
	// When true, dynamic registries will search their APIs for matching tools.
	DynamicLoadout bool
}

// ToolSearcher can search for tools by query.
type ToolSearcher interface {
	ToolSource

	// Search finds tools matching the query.
	// ctx is for cancellation and timeouts; searchCtx provides additional search options (can be nil).
	// max limits results (0 = no limit).
	// Returns tools sorted by relevance.
	Search(ctx context.Context, searchCtx *SearchContext, query string, maxResults int) []tools.Tool
}

// ToolExecutor can execute tools directly.
type ToolExecutor interface {
	ToolSource

	// Tool returns a specific tool by name, or nil if not found.
	Tool(name string) tools.Tool

	// Execute calls a tool with the given arguments.
	Execute(ctx context.Context, toolName string, args json.RawMessage) (tools.ToolResult, error)
}

// ClientProvider exposes the underlying MCP client.
// Only applicable to registries backed by MCP servers.
type ClientProvider interface {
	// Client returns the MCP client for direct protocol access.
	// Returns nil if no client is available.
	Client() Client
}

// Registry is the full interface for MCP-based tool registries.
// It composes all tool source capabilities.
type Registry interface {
	ToolLister
	ToolSearcher
	ToolExecutor
	ClientProvider

	// Initialize connects to the underlying source and discovers tools.
	Initialize(ctx context.Context) error

	// IsConnected returns true if the registry is ready.
	IsConnected() bool

	// Metadata returns registry metadata (server info, capabilities).
	Metadata() RegistryMetadata
}

// RegistryManager manages multiple Registry instances.
// It provides a unified view of tools from all registries.
type RegistryManager interface {
	// Register adds a registry to the manager.
	// Returns error if a registry with the same name exists.
	Register(registry Registry) error

	// Unregister removes a registry by name.
	// Closes the registry before removal.
	Unregister(name string) error

	// Get retrieves a registry by name.
	Get(name string) (Registry, bool)

	// All returns all registered registries.
	All() []Registry

	// AllTools returns tools from all registries.
	// Handles naming with mcp_{registry}_{tool} pattern.
	AllTools() []tools.Tool

	// Search searches across all registries.
	// ctx is for cancellation and timeouts; searchCtx provides additional search options (can be nil).
	Search(ctx context.Context, searchCtx *SearchContext, query string, maxResults int) []tools.Tool

	// Tool finds a tool by name.
	// Supports qualified names (registry:tool) for explicit registry targeting.
	Tool(name string) tools.Tool

	// Close closes all registries.
	Close() error
}

// executeMCPTool executes a tool via an MCP client, handling argument parsing and response conversion.
// This is the shared implementation used by all registry Execute methods.
func executeMCPTool(ctx context.Context, mcpClient Client, mcpTool *Tool, args json.RawMessage) (tools.ToolResult, error) {
	// Parse arguments.
	var argsMap map[string]any
	if len(args) > 0 {
		err := json.Unmarshal(args, &argsMap)
		if err != nil {
			return tools.ToolResult{
				Success: false,
				Error:   fmt.Sprintf("invalid arguments: %v", err),
			}, nil
		}
	}

	// Call tool.
	callReq := mcpSDK.CallToolRequest{
		Params: mcpSDK.CallToolParams{
			Name:      mcpTool.Tool.Name,
			Arguments: argsMap,
		},
	}

	resp, callErr := mcpClient.CallTool(ctx, callReq)
	if callErr != nil {
		return tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("tool call failed: %v", callErr),
		}, nil
	}

	if resp.IsError {
		return tools.ToolResult{
			Success: false,
			Error:   "tool execution failed",
		}, nil
	}

	// Convert response.
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
