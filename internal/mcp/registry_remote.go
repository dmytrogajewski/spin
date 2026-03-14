package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	mcpSDK "github.com/mark3labs/mcp-go/mcp"

	"github.com/dmytrogajewski/spin/internal/tools"
)

// RemoteRegistryConfig holds configuration for a remote MCP registry.
type RemoteRegistryConfig struct {
	Name      string
	Transport TransportType // sse or streamable-http.
	URL       string
	Headers   map[string]string
	OAuth     *OAuthConfig
	Logger    *slog.Logger
}

// RemoteRegistry wraps a remote MCP server (SSE or HTTP) as an Registry.
type RemoteRegistry struct {
	name      string
	config    RemoteRegistryConfig
	sdkClient *client.Client
	mcpClient Client
	tools     map[string]*Tool
	metadata  RegistryMetadata
	logger    *slog.Logger
	mu        sync.RWMutex
	connected bool
}

// NewRemoteRegistry creates a new RemoteRegistry for SSE or HTTP MCP servers.
func NewRemoteRegistry(config RemoteRegistryConfig) (*RemoteRegistry, error) {
	if config.Name == "" {
		return nil, ErrRegistryNameRequired
	}

	if config.URL == "" {
		return nil, ErrURLRequiredForRemoteRegistry
	}

	if config.Transport != TransportSSE && config.Transport != TransportStreamableHTTP {
		return nil, ErrTransportMustBeSseOrStreamable
	}

	return &RemoteRegistry{
		name:   config.Name,
		config: config,
		tools:  make(map[string]*Tool),
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

	// Create and start the transport client.
	if err := r.connectTransport(ctx); err != nil {
		return err
	}

	// Initialize the MCP protocol and discover tools.
	if err := r.initializeProtocol(ctx); err != nil {
		r.mcpClient.Close()

		return err
	}

	r.metadata.ToolCount = len(r.tools)
	r.metadata.Connected = true
	r.connected = true

	if r.logger != nil {
		r.logger.InfoContext(ctx, "remote registry initialized",
			"name", r.name,
			"transport", r.config.Transport,
			"tools", len(r.tools))
	}

	return nil
}

// connectTransport creates the SDK client, starts its transport, and stores it.
func (r *RemoteRegistry) connectTransport(ctx context.Context) error {
	var (
		sdkClient *client.Client
		err       error
	)

	switch r.config.Transport {
	case TransportSSE:
		sdkClient, err = r.createSSEClient()
	case TransportStreamableHTTP:
		sdkClient, err = r.createStreamableHTTPClient()
	default:
		return fmt.Errorf("unsupported transport: %s: %w", r.config.Transport, ErrUnsupportedTransport)
	}

	if err != nil {
		return fmt.Errorf("create client: %w", err)
	}

	if err = sdkClient.Start(ctx); err != nil {
		sdkClient.Close()

		return fmt.Errorf("start transport: %w", err)
	}

	r.sdkClient = sdkClient
	r.mcpClient = &sdkClientWrapper{client: sdkClient}

	return nil
}

// initializeProtocol performs the MCP handshake and discovers tools.
func (r *RemoteRegistry) initializeProtocol(ctx context.Context) error {
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
		return fmt.Errorf("initialize connection: %w", err)
	}

	r.metadata.ServerInfo = &initResp.ServerInfo
	r.metadata.Capabilities = initResp.Capabilities

	toolsResp, err := r.mcpClient.ListTools(ctx, mcpSDK.ListToolsRequest{})
	if err != nil {
		return fmt.Errorf("list tools: %w", err)
	}

	for _, tool := range toolsResp.Tools {
		r.tools[tool.Name] = &Tool{
			ServerName: r.name,
			Tool:       tool,
			Client:     r.mcpClient,
		}
	}

	return nil
}

// buildOAuthConfig extracts the OAuth configuration from the registry config.
func (r *RemoteRegistry) buildOAuthConfig() transport.OAuthConfig {
	return transport.OAuthConfig{
		ClientID:     r.config.OAuth.ClientID,
		ClientSecret: r.config.OAuth.ClientSecret,
		RedirectURI:  r.config.OAuth.RedirectURL,
		Scopes:       r.config.OAuth.Scopes,
	}
}

// createMCPClient creates an MCP client using the provided factory functions.
// This eliminates duplication between SSE and StreamableHTTP client creation paths.
func (r *RemoteRegistry) createMCPClient(
	newPlainClient func(url string) (*client.Client, error),
	newOAuthClient func(url string, oauth transport.OAuthConfig) (*client.Client, error),
	label string,
) (*client.Client, error) {
	if r.config.OAuth != nil {
		c, err := newOAuthClient(r.config.URL, r.buildOAuthConfig())
		if err != nil {
			return nil, fmt.Errorf("create OAuth %s client: %w", label, err)
		}

		return c, nil
	}

	c, err := newPlainClient(r.config.URL)
	if err != nil {
		return nil, fmt.Errorf("create %s client: %w", label, err)
	}

	return c, nil
}

// sseClientFactories returns plain and OAuth factory functions for SSE transport.
func sseClientFactories(headers map[string]string) (
	func(string) (*client.Client, error),
	func(string, transport.OAuthConfig) (*client.Client, error),
) {
	var opts []transport.ClientOption
	if len(headers) > 0 {
		opts = append(opts, transport.WithHeaders(headers))
	}

	return func(url string) (*client.Client, error) {
			return client.NewSSEMCPClient(url, opts...)
		}, func(url string, oauth transport.OAuthConfig) (*client.Client, error) {
			return client.NewOAuthSSEClient(url, oauth, opts...)
		}
}

// createSSEClient creates an SSE MCP client.
func (r *RemoteRegistry) createSSEClient() (*client.Client, error) {
	plain, oauth := sseClientFactories(r.config.Headers)

	return r.createMCPClient(plain, oauth, "SSE")
}

// createStreamableHTTPClient creates a streamable HTTP MCP client.
func (r *RemoteRegistry) createStreamableHTTPClient() (*client.Client, error) {
	var opts []transport.StreamableHTTPCOption
	if len(r.config.Headers) > 0 {
		opts = append(opts, transport.WithHTTPHeaders(r.config.Headers))
	}

	return r.createMCPClient(
		func(url string) (*client.Client, error) {
			return client.NewStreamableHttpClient(url, opts...)
		},
		func(url string, oauth transport.OAuthConfig) (*client.Client, error) {
			return client.NewOAuthStreamableHttpClient(url, oauth, opts...)
		},
		"streamable HTTP",
	)
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
func (r *RemoteRegistry) Client() Client {
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
func (r *RemoteRegistry) Search(_ context.Context, _ *SearchContext, query string, maxResults int) []tools.Tool {
	return SearchTools(r.List(), query, maxResults, DefaultSearchOptions())
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
		return tools.ToolResult{}, fmt.Errorf("tool not found: %s: %w", toolName, tools.ErrToolNotFound)
	}

	return executeMCPTool(ctx, mcpClient, mcpTool, args)
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

// wrapTool wraps an Tool as a tools.Tool with qualified name.
func (r *RemoteRegistry) wrapTool(mcpTool *Tool) tools.Tool {
	return &remoteToolWrapper{
		registry: r,
		mcpTool:  mcpTool,
	}
}

// remoteToolWrapper wraps an Tool to implement tools.Tool.
type remoteToolWrapper struct {
	registry *RemoteRegistry
	mcpTool  *Tool
}

// Name implements the Name operation.
func (w *remoteToolWrapper) Name() string {
	return fmt.Sprintf("mcp_%s_%s", w.registry.name, w.mcpTool.Tool.Name)
}

// Description implements the Description operation.
func (w *remoteToolWrapper) Description() string {
	return toolDescription(w.mcpTool)
}

// Schema implements the Schema operation.
func (w *remoteToolWrapper) Schema() tools.ToolSchema {
	return buildToolSchema(w, w.mcpTool)
}

// Execute implements the Execute operation.
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
