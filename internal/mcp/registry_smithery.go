package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/dmytrogajewski/spin/internal/tools"
)

const smitherySearchTimeout = 10 * time.Second

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
	baseRegistry

	config    SmitheryRegistryConfig
	client    *SmitheryClient
	apiClient *SmitheryAPIClient
	logger    *slog.Logger

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
		baseRegistry: baseRegistry{
			name:  config.Name,
			tools: make(map[string]*Tool),
			metadata: RegistryMetadata{
				Name: config.Name,
				Type: "smithery",
			},
		},
		config:        config,
		apiClient:     apiClient,
		loadedServers: make(map[string]*RemoteRegistry),
		logger:        config.Logger,
	}, nil
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

	// Use the shared handshake helper.
	meta, toolsMap, err := initializeMCPConnection(ctx, r.client, r.name)
	if err != nil {
		r.client.Close()

		return err
	}

	r.applyHandshakeResult(meta, toolsMap)

	return nil
}

// Client returns the underlying MCP client.
// For SmitheryRegistry this returns the SmitheryClient (which implements Client).
func (r *SmitheryRegistry) Client() Client {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.client
}

// Execute calls a tool with the given arguments.
// Overrides baseRegistry.Execute to use the SmitheryClient.
func (r *SmitheryRegistry) Execute(ctx context.Context, toolName string, args json.RawMessage) (tools.ToolResult, error) {
	r.mu.RLock()
	mcpTool, exists := r.tools[toolName]
	mcpClient := r.client
	r.mu.RUnlock()

	if !exists {
		return tools.ToolResult{}, fmt.Errorf("tool not found: %s: %w", toolName, tools.ErrToolNotFound)
	}

	return executeMCPTool(ctx, mcpClient, mcpTool, args)
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
	searchCtx, cancel := context.WithTimeout(ctx, smitherySearchTimeout)
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

// Close closes the registry, all dynamically loaded servers, and releases resources.
func (r *SmitheryRegistry) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.connected {
		return nil
	}

	r.connected = false
	r.metadata.Connected = false

	var errs []error

	// Close all dynamically loaded servers.
	for path, reg := range r.loadedServers {
		if closeErr := reg.Close(); closeErr != nil {
			errs = append(errs, fmt.Errorf("close loaded server %s: %w", path, closeErr))
		}
	}

	clear(r.loadedServers)

	// Close the static client.
	if r.client != nil {
		if closeErr := r.client.Close(); closeErr != nil {
			errs = append(errs, fmt.Errorf("close static client: %w", closeErr))
		}
	}

	return errors.Join(errs...)
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
