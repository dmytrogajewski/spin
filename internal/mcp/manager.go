package mcp

import (
	"context"

	"github.com/mark3labs/mcp-go/client"
	mcpSDK "github.com/mark3labs/mcp-go/mcp"
)

// Config holds MCP configuration.
type Config struct {
	EnableMCP  bool
	MCPServers []MCPServerConfig
}

// OAuthConfig holds OAuth configuration for protected MCP servers.
type OAuthConfig struct {
	ClientID     string   `yaml:"client_id" mapstructure:"client_id"`
	ClientSecret string   `yaml:"client_secret,omitempty" mapstructure:"client_secret"`
	RedirectURL  string   `yaml:"redirect_url,omitempty" mapstructure:"redirect_url"`
	Scopes       []string `yaml:"scopes,omitempty" mapstructure:"scopes"`
}

// MCPServerConfig holds configuration for a single MCP server.
type MCPServerConfig struct {
	// Common fields
	Name      string        `yaml:"name" mapstructure:"name"`
	Transport TransportType `yaml:"transport,omitempty" mapstructure:"transport"`

	// Stdio transport fields (mutually exclusive with URL)
	Command string            `yaml:"command,omitempty" mapstructure:"command"`
	Args    []string          `yaml:"args,omitempty" mapstructure:"args"`
	Env     map[string]string `yaml:"env,omitempty" mapstructure:"env"`

	// Remote transport fields (mutually exclusive with Command)
	URL     string            `yaml:"url,omitempty" mapstructure:"url"`
	Headers map[string]string `yaml:"headers,omitempty" mapstructure:"headers"`

	// OAuth configuration (optional, for protected servers)
	OAuth *OAuthConfig `yaml:"oauth,omitempty" mapstructure:"oauth"`

	// Smithery-specific fields
	SmitheryAPIKey    string `yaml:"smithery_api_key,omitempty" mapstructure:"smithery_api_key"`
	SmitheryNamespace string `yaml:"smithery_namespace,omitempty" mapstructure:"smithery_namespace"`
}

// MCPClient is the interface for MCP clients (SDK client or Smithery client).
type MCPClient interface {
	Initialize(ctx context.Context, request mcpSDK.InitializeRequest) (*mcpSDK.InitializeResult, error)
	ListTools(ctx context.Context, request mcpSDK.ListToolsRequest) (*mcpSDK.ListToolsResult, error)
	CallTool(ctx context.Context, request mcpSDK.CallToolRequest) (*mcpSDK.CallToolResult, error)
	Close() error
}

// sdkClientWrapper wraps the SDK client to implement MCPClient interface.
type sdkClientWrapper struct {
	client *client.Client
}

func (w *sdkClientWrapper) Initialize(ctx context.Context, request mcpSDK.InitializeRequest) (*mcpSDK.InitializeResult, error) {
	return w.client.Initialize(ctx, request)
}

func (w *sdkClientWrapper) ListTools(ctx context.Context, request mcpSDK.ListToolsRequest) (*mcpSDK.ListToolsResult, error) {
	return w.client.ListTools(ctx, request)
}

func (w *sdkClientWrapper) CallTool(ctx context.Context, request mcpSDK.CallToolRequest) (*mcpSDK.CallToolResult, error) {
	return w.client.CallTool(ctx, request)
}

func (w *sdkClientWrapper) Close() error {
	return w.client.Close()
}

// MCPTool wraps an MCP tool for integration with registries.
type MCPTool struct {
	ServerName string
	Tool       mcpSDK.Tool
	Client     MCPClient
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
