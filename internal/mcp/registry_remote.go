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
	"github.com/mark3labs/mcp-go/client/transport"
	mcpSDK "github.com/mark3labs/mcp-go/mcp"
)

// RemoteRegistryConfig holds configuration for a remote MCP registry.
type RemoteRegistryConfig struct {
	Name      string
	Transport TransportType // sse or streamable-http
	URL       string
	Headers   map[string]string
	OAuth     *OAuthConfig
	Logger    *slog.Logger
}

// RemoteRegistry wraps a remote MCP server (SSE or HTTP) as an MCPRegistry.
type RemoteRegistry struct {
	name      string
	config    RemoteRegistryConfig
	sdkClient *client.Client
	mcpClient MCPClient
	tools     map[string]*MCPTool
	metadata  RegistryMetadata
	logger    *slog.Logger
	mu        sync.RWMutex
	connected bool
}

// NewRemoteRegistry creates a new RemoteRegistry for SSE or HTTP MCP servers.
func NewRemoteRegistry(config RemoteRegistryConfig) (*RemoteRegistry, error) {
	if config.Name == "" {
		return nil, fmt.Errorf("registry name is required")
	}
	if config.URL == "" {
		return nil, fmt.Errorf("URL is required for remote registry")
	}
	if config.Transport != TransportSSE && config.Transport != TransportStreamableHTTP {
		return nil, fmt.Errorf("transport must be 'sse' or 'streamable-http'")
	}

	return &RemoteRegistry{
		name:   config.Name,
		config: config,
		tools:  make(map[string]*MCPTool),
		logger: config.Logger,
		metadata: RegistryMetadata{
			Name: config.Name,
			Type: "remote",
		},
	}, nil
}

// Name returns the registry name.
func (r *RemoteRegistry) Name() string {
	return r.name
}

// Initialize connects to the MCP server and discovers tools.
func (r *RemoteRegistry) Initialize(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.connected {
		return nil
	}

	// Create client based on transport
	var sdkClient *client.Client
	var err error

	switch r.config.Transport {
	case TransportSSE:
		sdkClient, err = r.createSSEClient()
	case TransportStreamableHTTP:
		sdkClient, err = r.createStreamableHTTPClient()
	default:
		return fmt.Errorf("unsupported transport: %s", r.config.Transport)
	}

	if err != nil {
		return fmt.Errorf("create client: %w", err)
	}

	// Start the transport
	if err := sdkClient.Start(ctx); err != nil {
		sdkClient.Close()
		return fmt.Errorf("start transport: %w", err)
	}

	r.sdkClient = sdkClient
	r.mcpClient = &sdkClientWrapper{client: sdkClient}

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

	initResp, err := r.mcpClient.Initialize(ctx, initReq)
	if err != nil {
		r.mcpClient.Close()
		return fmt.Errorf("initialize connection: %w", err)
	}

	r.metadata.ServerInfo = &initResp.ServerInfo
	r.metadata.Capabilities = initResp.Capabilities

	// List tools
	listReq := mcpSDK.ListToolsRequest{}
	toolsResp, err := r.mcpClient.ListTools(ctx, listReq)
	if err != nil {
		r.mcpClient.Close()
		return fmt.Errorf("list tools: %w", err)
	}

	// Register tools
	for _, tool := range toolsResp.Tools {
		r.tools[tool.Name] = &MCPTool{
			ServerName: r.name,
			Tool:       tool,
			Client:     r.mcpClient,
		}
	}

	r.metadata.ToolCount = len(r.tools)
	r.metadata.Connected = true
	r.connected = true

	if r.logger != nil {
		r.logger.Info("remote registry initialized",
			"name", r.name,
			"transport", r.config.Transport,
			"tools", len(r.tools))
	}

	return nil
}

// createSSEClient creates an SSE MCP client.
func (r *RemoteRegistry) createSSEClient() (*client.Client, error) {
	var opts []transport.ClientOption
	if len(r.config.Headers) > 0 {
		opts = append(opts, transport.WithHeaders(r.config.Headers))
	}

	if r.config.OAuth != nil {
		oauthConfig := transport.OAuthConfig{
			ClientID:     r.config.OAuth.ClientID,
			ClientSecret: r.config.OAuth.ClientSecret,
			RedirectURI:  r.config.OAuth.RedirectURL,
			Scopes:       r.config.OAuth.Scopes,
		}
		return client.NewOAuthSSEClient(r.config.URL, oauthConfig, opts...)
	}

	return client.NewSSEMCPClient(r.config.URL, opts...)
}

// createStreamableHTTPClient creates a streamable HTTP MCP client.
func (r *RemoteRegistry) createStreamableHTTPClient() (*client.Client, error) {
	var opts []transport.StreamableHTTPCOption
	if len(r.config.Headers) > 0 {
		opts = append(opts, transport.WithHTTPHeaders(r.config.Headers))
	}

	if r.config.OAuth != nil {
		oauthConfig := transport.OAuthConfig{
			ClientID:     r.config.OAuth.ClientID,
			ClientSecret: r.config.OAuth.ClientSecret,
			RedirectURI:  r.config.OAuth.RedirectURL,
			Scopes:       r.config.OAuth.Scopes,
		}
		return client.NewOAuthStreamableHttpClient(r.config.URL, oauthConfig, opts...)
	}

	return client.NewStreamableHttpClient(r.config.URL, opts...)
}

// IsConnected returns true if the registry is connected.
func (r *RemoteRegistry) IsConnected() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.connected
}

// Metadata returns registry metadata.
func (r *RemoteRegistry) Metadata() RegistryMetadata {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.metadata
}

// Client returns the underlying MCP client.
func (r *RemoteRegistry) Client() MCPClient {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.mcpClient
}

// List returns all tools from this registry.
func (r *RemoteRegistry) List() []tools.Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]tools.Tool, 0, len(r.tools))
	for _, mcpTool := range r.tools {
		result = append(result, r.wrapTool(mcpTool))
	}
	return result
}

// Count returns the number of tools.
func (r *RemoteRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.tools)
}

// Search finds tools matching the query.
func (r *RemoteRegistry) Search(query string, max int) []tools.Tool {
	return SearchTools(r.List(), query, max, DefaultSearchOptions())
}

// Tool returns a specific tool by name.
func (r *RemoteRegistry) Tool(name string) tools.Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	mcpTool, exists := r.tools[name]
	if !exists {
		return nil
	}
	return r.wrapTool(mcpTool)
}

// Execute calls a tool with the given arguments.
func (r *RemoteRegistry) Execute(ctx context.Context, toolName string, args json.RawMessage) (tools.ToolResult, error) {
	r.mu.RLock()
	mcpTool, exists := r.tools[toolName]
	mcpClient := r.mcpClient
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

	resp, err := mcpClient.CallTool(ctx, callReq)
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
func (r *RemoteRegistry) Close() error {
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

// wrapTool wraps an MCPTool as a tools.Tool with qualified name.
func (r *RemoteRegistry) wrapTool(mcpTool *MCPTool) tools.Tool {
	return &remoteToolWrapper{
		registry: r,
		mcpTool:  mcpTool,
	}
}

// remoteToolWrapper wraps an MCPTool to implement tools.Tool.
type remoteToolWrapper struct {
	registry *RemoteRegistry
	mcpTool  *MCPTool
}

func (w *remoteToolWrapper) Name() string {
	return fmt.Sprintf("mcp_%s_%s", w.registry.name, w.mcpTool.Tool.Name)
}

func (w *remoteToolWrapper) Description() string {
	if w.mcpTool.Tool.Description != "" {
		return w.mcpTool.Tool.Description
	}
	return fmt.Sprintf("MCP tool: %s", w.mcpTool.Tool.Name)
}

func (w *remoteToolWrapper) Schema() tools.ToolSchema {
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

func (w *remoteToolWrapper) fallbackSchema() tools.ToolSchema {
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

func (w *remoteToolWrapper) Execute(ctx context.Context, params tools.ToolParameters) (tools.ToolResult, error) {
	argsJSON, err := json.Marshal(params.ToMap())
	if err != nil {
		return tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("marshal arguments: %v", err),
		}, nil
	}
	return w.registry.Execute(ctx, w.mcpTool.Tool.Name, argsJSON)
}
