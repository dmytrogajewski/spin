package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	mcpSDK "github.com/mark3labs/mcp-go/mcp"

	"github.com/dmytrogajewski/spin/internal/tools"
)

// SmitheryRegistryConfig holds configuration for a Smithery MCP registry.
type SmitheryRegistryConfig struct {
	Name      string
	APIKey    string
	MCPURL    string
	Namespace string
	Logger    *slog.Logger
}

// SmitheryRegistry wraps a Smithery-hosted MCP server as an Registry.
type SmitheryRegistry struct {
	name      string
	config    SmitheryRegistryConfig
	client    *SmitheryClient
	apiClient *SmitheryAPIClient
	tools     map[string]*Tool
	metadata  RegistryMetadata
	logger    *slog.Logger
	mu        sync.RWMutex
	connected bool

	// Dynamic loading state - stores RemoteRegistry instances for each loaded server.
	loadedServers map[string]*RemoteRegistry // serverPath -> registry.
}

// NewSmitheryRegistry creates a new SmitheryRegistry.
// For dynamic loadout registries, only Name and APIKey are required.
// For static registries that connect to a specific server, MCPURL and Namespace are also required.
func NewSmitheryRegistry(config SmitheryRegistryConfig) (*SmitheryRegistry, error) {
	if config.Name == "" {
		return nil, ErrRegistryNameRequired
	}

	if config.APIKey == "" {
		return nil, ErrAPIKeyRequiredForSmithery
	}
	// MCPURL and Namespace are only required for static registries (non-dynamic loadout)
	// Dynamic loadout registries use the Smithery API for search and create connections on-demand.

	// Create API client for dynamic operations.
	apiClient, err := NewSmitheryAPIClient(SmitheryAPIConfig{
		APIKey: config.APIKey,
		Logger: config.Logger,
	})
	if err != nil {
		return nil, fmt.Errorf("create API client: %w", err)
	}

	return &SmitheryRegistry{
		name:          config.Name,
		config:        config,
		apiClient:     apiClient,
		tools:         make(map[string]*Tool),
		loadedServers: make(map[string]*RemoteRegistry),
		logger:        config.Logger,
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
// For dynamic loadout registries (no MCPURL), this is a no-op - tools are discovered via search.
func (r *SmitheryRegistry) Initialize(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.connected {
		return nil
	}

	// Dynamic loadout mode: no specific server to connect to.
	// Tools are discovered via Smithery API search and loaded on-demand.
	if r.config.MCPURL == "" {
		r.connected = true
		r.metadata.Extra = map[string]any{
			"mode": "dynamic",
		}

		return nil
	}

	// Static mode: connect to a specific Smithery server.
	if err := r.initializeStatic(ctx); err != nil {
		return err
	}

	r.metadata.ToolCount = len(r.tools)
	r.metadata.Connected = true
	r.connected = true

	if r.logger != nil {
		r.logger.InfoContext(ctx, "smithery registry initialized",
			"name", r.name,
			"namespace", r.config.Namespace,
			"tools", len(r.tools))
	}

	return nil
}

// initializeStatic connects to a specific Smithery server, performs the MCP handshake,
// and discovers tools.
func (r *SmitheryRegistry) initializeStatic(ctx context.Context) error {
	if r.config.Namespace == "" {
		return ErrNamespaceRequiredForSmithery
	}

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

	toolsResp, err := r.client.ListTools(ctx, mcpSDK.ListToolsRequest{})
	if err != nil {
		r.client.Close()
		return fmt.Errorf("list tools: %w", err)
	}

	for _, tool := range toolsResp.Tools {
		r.tools[tool.Name] = &Tool{
			ServerName: r.name,
			Tool:       tool,
			Client:     r.client,
		}
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
func (r *SmitheryRegistry) Client() Client {
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
// For static registries, this searches already-loaded tools.
// For dynamic registries (with context), this calls the Smithery API and returns loadable tool stubs.
func (r *SmitheryRegistry) Search(ctx context.Context, _ *SearchContext, query string, maxResults int) []tools.Tool {
	// For static mode, just search loaded tools.
	if !r.IsDynamic() {
		return SearchTools(r.List(), query, maxResults, DefaultSearchOptions())
	}

	// For dynamic mode without context, fall back to loaded tools.
	if ctx == nil {
		return SearchTools(r.List(), query, maxResults, DefaultSearchOptions())
	}

	// Dynamic mode with context: search the Smithery API.
	searchCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	searchResp, err := r.apiClient.SearchTools(searchCtx, query, maxResults)
	if err != nil {
		if r.logger != nil {
			r.logger.WarnContext(searchCtx, "Smithery API search failed", "error", err)
		}
		// Fall back to searching loaded tools.
		return SearchTools(r.List(), query, maxResults, DefaultSearchOptions())
	}

	// Convert API results to loadable tool stubs.
	results := make([]tools.Tool, 0, len(searchResp.Tools))
	for _, t := range searchResp.Tools {
		results = append(results, &smitheryLoadableTool{
			registry:    r,
			toolName:    t.Tool.Name,
			description: t.Tool.Description,
			serverPath:  t.Server.QualifiedName,
		})
	}

	return results
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
return tools.ToolResult{}, fmt.Errorf("tool not found: %s: %w", toolName, ErrToolNotFound)
	}

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

	resp, callErr := client.CallTool(ctx, callReq)
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

// wrapTool wraps an Tool as a tools.Tool with qualified name.
func (r *SmitheryRegistry) wrapTool(mcpTool *Tool) tools.Tool {
	return &smitheryToolWrapper{
		registry: r,
		mcpTool:  mcpTool,
	}
}

// smitheryToolWrapper wraps an Tool to implement tools.Tool.
type smitheryToolWrapper struct {
	registry *SmitheryRegistry
	mcpTool  *Tool
}

// Name implements the Name operation.
func (w *smitheryToolWrapper) Name() string {
	return fmt.Sprintf("mcp_%s_%s", w.registry.name, w.mcpTool.Tool.Name)
}

// Description implements the Description operation.
func (w *smitheryToolWrapper) Description() string {
	if w.mcpTool.Tool.Description != "" {
		return w.mcpTool.Tool.Description
	}

	return fmt.Sprintf("MCP tool: %s", w.mcpTool.Tool.Name)
}

// Schema implements the Schema operation.
func (w *smitheryToolWrapper) Schema() tools.ToolSchema {
	schemaBytes, err := json.Marshal(w.mcpTool.Tool.InputSchema)
	if err != nil {
		return w.fallbackSchema()
	}

	var mcpSchema JSONSchema
	err = json.Unmarshal(schemaBytes, &mcpSchema)
	if err != nil {
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

// Execute implements the Execute operation.
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

// ============================================================================
// Dynamic Tool Loading
// ============================================================================.

// IsDynamic returns true if this registry is in dynamic loadout mode.
func (r *SmitheryRegistry) IsDynamic() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.config.MCPURL == ""
}

// SearchAPI searches the Smithery API for tools matching the query.
// This is used for dynamic tool discovery before loading.
func (r *SmitheryRegistry) SearchAPI(ctx context.Context, query string, limit int) (*SmitherySearchResponse, error) {
	if r.apiClient == nil {
		return nil, ErrAPIClientNotInitialized
	}

	return r.apiClient.SearchTools(ctx, query, limit)
}

// LoadServer connects to a specific Smithery server via streamable-http and registers its tools.
// serverPath is the qualified name like "@namespace/server-name".
// Returns the list of newly loaded tools.
func (r *SmitheryRegistry) LoadServer(ctx context.Context, serverPath string) ([]tools.Tool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if already loaded.
	if _, exists := r.loadedServers[serverPath]; exists {
		if r.logger != nil {
			r.logger.DebugContext(ctx, "server already loaded", "server", serverPath)
		}

		return nil, nil
	}

	// Build the streamable-http URL with API key
	// Format: https://server.smithery.ai/@namespace/server/mcp?api_key=...
	mcpURL := fmt.Sprintf("%s?api_key=%s", r.apiClient.GetServerMCPURL(serverPath), r.config.APIKey)

	// Create a safe registry name from server path.
	safeName := strings.ReplaceAll(serverPath, "@", "")
	safeName = strings.ReplaceAll(safeName, "/", "_")

	if r.logger != nil {
		r.logger.InfoContext(ctx, "loading Smithery server via streamable-http",
			"server", serverPath,
			"registry_name", safeName,
			"url", mcpURL)
	}

	// Create RemoteRegistry with streamable-http transport.
	remoteReg, err := NewRemoteRegistry(RemoteRegistryConfig{
		Name:      safeName,
		Transport: TransportStreamableHTTP,
		URL:       mcpURL,
		Logger:    r.logger,
	})
	if err != nil {
		return nil, fmt.Errorf("create remote registry for %s: %w", serverPath, err)
	}

	// Initialize the connection (this discovers tools).
	err = remoteReg.Initialize(ctx)
	if err != nil {
		remoteReg.Close()

		return nil, fmt.Errorf("initialize %s: %w", serverPath, err)
	}

	// Get tools from the remote registry.
	newTools := remoteReg.List()

	// Track loaded server.
	r.loadedServers[serverPath] = remoteReg
	r.metadata.ToolCount += len(newTools)

	if r.logger != nil {
		r.logger.InfoContext(ctx, "loaded Smithery server",
			"server", serverPath,
			"tools", len(newTools))
	}

	return newTools, nil
}

// GetLoadedServer returns a loaded RemoteRegistry by server path, or nil if not loaded.
func (r *SmitheryRegistry) GetLoadedServer(serverPath string) *RemoteRegistry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.loadedServers[serverPath]
}

// GetLoadedServerNames returns all loaded server paths.
func (r *SmitheryRegistry) GetLoadedServerNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.loadedServers))
	for name := range r.loadedServers {
		names = append(names, name)
	}

	return names
}

// ============================================================================
// Loadable Tool Stub (for dynamic tool discovery)
// ============================================================================.

// smitheryLoadableTool is a tool stub returned from Smithery API search.
// It represents a tool that CAN be loaded but isn't connected yet.
// When Load() is called, it connects to the server and returns the real tool.
type smitheryLoadableTool struct {
	registry    *SmitheryRegistry
	toolName    string
	description string
	serverPath  string
}

// Name returns the tool name (qualified with server path).
func (t *smitheryLoadableTool) Name() string {
	safeName := strings.ReplaceAll(t.serverPath, "@", "")
	safeName = strings.ReplaceAll(safeName, "/", "_")

	return fmt.Sprintf("mcp_%s_%s", safeName, t.toolName)
}

// Description returns the tool description.
func (t *smitheryLoadableTool) Description() string {
	return t.description
}

// Schema returns a minimal schema - full schema available after loading.
func (t *smitheryLoadableTool) Schema() tools.ToolSchema {
	return tools.ToolSchema{
		Type: "function",
		Function: tools.FunctionSchema{
			Name:        t.Name(),
			Description: t.description,
			Parameters: tools.ParameterSchema{
				Type:       "object",
				Properties: make(map[string]tools.PropertyDefinition),
				Required:   []string{},
			},
		},
	}
}

// Execute attempts to load the server and execute the tool.
// On first call, it loads the server. Subsequent calls use the loaded tool.
func (t *smitheryLoadableTool) Execute(ctx context.Context, params tools.ToolParameters) (tools.ToolResult, error) {
	// Load the server if not already loaded.
	_, err := t.registry.LoadServer(ctx, t.serverPath)
	if err != nil {
		return tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("failed to load server %s: %v", t.serverPath, err),
		}, nil
	}

	// Find the loaded tool and execute it.
	loadedReg := t.registry.GetLoadedServer(t.serverPath)
	if loadedReg == nil {
		return tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("server %s not found after loading", t.serverPath),
		}, nil
	}

	// Find and execute the tool.
	argsJSON, err := json.Marshal(params.ToMap())
	if err != nil {
		return tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("marshal arguments: %v", err),
		}, nil
	}

	return loadedReg.Execute(ctx, t.toolName, argsJSON)
}

// ServerPath returns the Smithery server path for this tool.
func (t *smitheryLoadableTool) ServerPath() string {
	return t.serverPath
}

// IsLoadable returns true - this is a loadable tool stub.
func (t *smitheryLoadableTool) IsLoadable() bool {
	return true
}

// Load loads the server and returns the actual tools.
func (t *smitheryLoadableTool) Load(ctx context.Context) ([]tools.Tool, error) {
	return t.registry.LoadServer(ctx, t.serverPath)
}
