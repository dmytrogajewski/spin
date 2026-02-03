package mcp

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/dmytrogajewski/spin/internal/tools"
	mcpSDK "github.com/mark3labs/mcp-go/mcp"
)

// ErrUnsupportedTransport is returned when a transport type is not supported.
var ErrUnsupportedTransport = errors.New("unsupported transport type")

// RegistryMetadata contains information about a registry.
// Implementations can extend this with custom fields via the Extra map.
type RegistryMetadata struct {
	Name         string
	Type         string // Implementation-defined type identifier
	ServerInfo   *mcpSDK.Implementation
	Capabilities mcpSDK.ServerCapabilities
	ToolCount    int
	Connected    bool
	Extra        map[string]any // Implementation-specific metadata
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
	// Ctx is the context for cancellation and timeouts (required for dynamic registries)
	Ctx context.Context

	// TrajectoryContext provides execution context for relevance scoring (optional)
	TrajectoryContext interface {
		GetQuery() string
		GetRecentTools(n int) []string
	}

	// DynamicLoadout enables dynamic tool loading for this search (optional)
	// When true, dynamic registries will search their APIs for matching tools
	DynamicLoadout bool
}

// ToolSearcher can search for tools by query.
type ToolSearcher interface {
	ToolSource

	// Search finds tools matching the query.
	// ctx can be nil for simple searches; required for dynamic registries that call APIs.
	// max limits results (0 = no limit).
	// Returns tools sorted by relevance.
	Search(ctx *SearchContext, query string, max int) []tools.Tool
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
	Client() MCPClient
}

// MCPRegistry is the full interface for MCP-based tool registries.
// It composes all tool source capabilities.
type MCPRegistry interface {
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

// RegistryManager manages multiple MCPRegistry instances.
// It provides a unified view of tools from all registries.
type RegistryManager interface {
	// Register adds a registry to the manager.
	// Returns error if a registry with the same name exists.
	Register(registry MCPRegistry) error

	// Unregister removes a registry by name.
	// Closes the registry before removal.
	Unregister(name string) error

	// Get retrieves a registry by name.
	Get(name string) (MCPRegistry, bool)

	// All returns all registered registries.
	All() []MCPRegistry

	// AllTools returns tools from all registries.
	// Handles naming with mcp_{registry}_{tool} pattern.
	AllTools() []tools.Tool

	// Search searches across all registries.
	// ctx can be nil for simple searches; required for dynamic registries.
	Search(ctx *SearchContext, query string, max int) []tools.Tool

	// Tool finds a tool by name.
	// Supports qualified names (registry:tool) for explicit registry targeting.
	Tool(name string) tools.Tool

	// Close closes all registries.
	Close() error
}
