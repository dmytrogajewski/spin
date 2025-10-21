// Package client provides an MCP (Model Context Protocol) client implementation.
//
// The client connects to MCP servers via stdio transport and communicates using JSON-RPC 2.0.
package client

import (
	"context"
	"encoding/json"
	"time"

	"github.com/dmytrogajewski/spin/internal/mcp/types"
)

// Client represents an MCP client connection to a server.
type Client interface {
	// Initialize establishes the MCP connection and negotiates capabilities
	Initialize(ctx context.Context, req types.InitializeRequest) (*types.InitializeResponse, error)

	// ListTools retrieves available tools from the server
	ListTools(ctx context.Context) (*types.ListToolsResponse, error)

	// CallTool invokes a tool on the server
	CallTool(ctx context.Context, name string, arguments json.RawMessage) (*types.CallToolResponse, error)

	// ListResources retrieves available resources from the server (future)
	ListResources(ctx context.Context) (*types.ListResourcesResponse, error)

	// ReadResource reads a specific resource (future)
	ReadResource(ctx context.Context, uri string) (*types.ReadResourceResponse, error)

	// Close closes the connection and cleans up resources
	Close() error
}

// Config contains configuration for an MCP client.
type Config struct {
	// Command is the executable to spawn for the MCP server
	Command string

	// Args are arguments passed to the command
	Args []string

	// Env contains environment variables for the server process
	Env map[string]string

	// Timeout is the maximum duration for operations (default: 30s)
	Timeout time.Duration
}

// Validate validates the configuration and sets defaults.
func (c *Config) Validate() error {
	if c.Command == "" {
		return &Error{Op: "validate", Err: ErrSpawnFailed}
	}

	// Set default timeout if not specified
	if c.Timeout == 0 {
		c.Timeout = 30 * time.Second
	}

	return nil
}
