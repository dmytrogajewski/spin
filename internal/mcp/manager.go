// Package mcp provides MCP (Model Context Protocol) client and registry.
package mcp

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/client"
	mcpSDK "github.com/mark3labs/mcp-go/mcp"
)

// Config holds MCP configuration.
type Config struct {
	EnableMCP  bool
	MCPServers []ServerConfig
}

// OAuthConfig holds OAuth configuration for protected MCP servers.
type OAuthConfig struct {
	ClientID     string   `mapstructure:"client_id"     yaml:"client_id"`
	ClientSecret string   `mapstructure:"client_secret" yaml:"client_secret,omitempty"`
	RedirectURL  string   `mapstructure:"redirect_url"  yaml:"redirect_url,omitempty"`
	Scopes       []string `mapstructure:"scopes"        yaml:"scopes,omitempty"`
}

// ServerConfig holds configuration for a single MCP server.
type ServerConfig struct {
	// Common fields.
	Name      string        `mapstructure:"name"      yaml:"name"`
	Transport TransportType `mapstructure:"transport" yaml:"transport,omitempty"`

	// Stdio transport fields (mutually exclusive with URL).
	Command string            `mapstructure:"command" yaml:"command,omitempty"`
	Args    []string          `mapstructure:"args"    yaml:"args,omitempty"`
	Env     map[string]string `mapstructure:"env"     yaml:"env,omitempty"`

	// Remote transport fields (mutually exclusive with Command).
	URL     string            `mapstructure:"url"     yaml:"url,omitempty"`
	Headers map[string]string `mapstructure:"headers" yaml:"headers,omitempty"`

	// OAuth configuration (optional, for protected servers).
	OAuth *OAuthConfig `mapstructure:"oauth" yaml:"oauth,omitempty"`

	// Smithery-specific fields.
	SmitheryAPIKey    string `mapstructure:"smithery_api_key"   yaml:"smithery_api_key,omitempty"`
	SmitheryNamespace string `mapstructure:"smithery_namespace" yaml:"smithery_namespace,omitempty"`
}

// Client is the interface for MCP clients (SDK client or Smithery client).
type Client interface {
	Initialize(ctx context.Context, request mcpSDK.InitializeRequest) (*mcpSDK.InitializeResult, error)
	ListTools(ctx context.Context, request mcpSDK.ListToolsRequest) (*mcpSDK.ListToolsResult, error)
	CallTool(ctx context.Context, request mcpSDK.CallToolRequest) (*mcpSDK.CallToolResult, error)
	Close() error
}

// sdkClientWrapper wraps the SDK client to implement Client interface.
type sdkClientWrapper struct {
	client *client.Client
}

// wrapErr wraps an error with an MCP operation prefix.
func wrapErr(op string, err error) error {
	return fmt.Errorf("mcp %s: %w", op, err)
}

// Initialize implements the Initialize operation.
func (w *sdkClientWrapper) Initialize(ctx context.Context, request mcpSDK.InitializeRequest) (*mcpSDK.InitializeResult, error) {
	result, err := w.client.Initialize(ctx, request)
	if err != nil {
		return nil, wrapErr("initialize", err)
	}

	return result, nil
}

// ListTools implements the ListTools operation.
func (w *sdkClientWrapper) ListTools(ctx context.Context, request mcpSDK.ListToolsRequest) (*mcpSDK.ListToolsResult, error) {
	result, err := w.client.ListTools(ctx, request)
	if err != nil {
		return nil, wrapErr("list tools", err)
	}

	return result, nil
}

// CallTool implements the CallTool operation.
func (w *sdkClientWrapper) CallTool(ctx context.Context, request mcpSDK.CallToolRequest) (*mcpSDK.CallToolResult, error) {
	result, err := w.client.CallTool(ctx, request)
	if err != nil {
		return nil, wrapErr("call tool", err)
	}

	return result, nil
}

// Close implements the Close operation.
func (w *sdkClientWrapper) Close() error {
	if err := w.client.Close(); err != nil {
		return wrapErr("close", err)
	}

	return nil
}

// Tool wraps an MCP tool for integration with registries.
type Tool struct {
	ServerName string
	Tool       mcpSDK.Tool
	Client     Client
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
