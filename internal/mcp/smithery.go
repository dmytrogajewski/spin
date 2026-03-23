package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"time"

	mcpSDK "github.com/mark3labs/mcp-go/mcp"
)

const (
	smitheryClientTimeout = 30 * time.Second
	schemeHTTPS           = "https"
)

// SmitheryClient implements an MCP client for Smithery's connection-based API.
// Smithery uses a different protocol than standard MCP transports:
// 1. Create a connection: POST /connect/{namespace}
// 2. Make RPC calls: POST /connect/{namespace}/{connectionId}/rpc.
type SmitheryClient struct {
	apiKey       string
	mcpURL       string
	namespace    string
	connectionID string
	httpClient   *http.Client
	logger       *slog.Logger
	mu           sync.RWMutex
	initialized  bool
	serverInfo   *mcpSDK.Implementation
	capabilities mcpSDK.ServerCapabilities
}

// SmitheryConfig holds configuration for creating a Smithery client.
type SmitheryConfig struct {
	// APIKey is the Smithery API key (Bearer token).
	APIKey string
	// MCPURL is the MCP server URL (e.g., https://server.smithery.ai/@user/server)
	MCPURL string
	// Namespace is your Smithery namespace (e.g., your username).
	Namespace string
	// Logger for debug output.
	Logger *slog.Logger
}

// smitheryConnectRequest is the request body for creating a connection.
type smitheryConnectRequest struct {
	MCPURL string         `json:"mcpUrl"`
	Config map[string]any `json:"config,omitempty"`
}

// smitheryConnectResponse is the response from creating a connection.
type smitheryConnectResponse struct {
	ConnectionID string `json:"connectionId"`
}

// smitheryRPCRequest is the request body for RPC calls.
type smitheryRPCRequest struct {
	Method string `json:"method"`
	Params any    `json:"params,omitempty"`
}

// smitheryRPCResponse is the response from RPC calls.
type smitheryRPCResponse struct {
	Result json.RawMessage `json:"result,omitempty"`
	Error  *smitheryError  `json:"error,omitempty"`
}

// smitheryError represents an error from Smithery API.
type smitheryError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// NewSmitheryClient creates a new Smithery MCP client.
// APIKey is always required. MCPURL and Namespace are required for connection-based operations.
func NewSmitheryClient(config SmitheryConfig) (*SmitheryClient, error) {
	if config.APIKey == "" {
		return nil, ErrSmitheryAPIKeyRequired
	}

	if config.MCPURL == "" {
		return nil, ErrSmitheryMcpURLRequired
	}

	if config.Namespace == "" {
		return nil, ErrSmitheryNamespaceRequired
	}

	return &SmitheryClient{
		apiKey:    config.APIKey,
		mcpURL:    config.MCPURL,
		namespace: config.Namespace,
		httpClient: &http.Client{
			Timeout: smitheryClientTimeout,
		},
		logger: config.Logger,
	}, nil
}

// Connect establishes a connection to the Smithery server.
func (c *SmitheryClient) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connectionID != "" {
		return nil // Already connected.
	}

	// Create connection request.
	reqBody := smitheryConnectRequest{
		MCPURL: c.mcpURL,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal connect request: %w", err)
	}

	rawURL := fmt.Sprintf("https://api.smithery.ai/connect/%s", c.namespace)

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("parse connect URL: %w", err)
	}

	if parsedURL.Scheme != schemeHTTPS {
		return fmt.Errorf("%w: got %q", ErrInvalidURLScheme, parsedURL.Scheme)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, parsedURL.String(), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create connect request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	if c.logger != nil {
		c.logger.DebugContext(ctx, "Creating Smithery connection", "url", parsedURL.String(), "mcpUrl", c.mcpURL)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("connect request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyBytes, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return fmt.Errorf("connect failed with status %d (body unreadable): %w", resp.StatusCode, ErrConnectFailedWithStatus)
		}

		return fmt.Errorf("connect failed with status %d: %s: %w", resp.StatusCode, string(bodyBytes), ErrConnectFailedWithStatus)
	}

	var connectResp smitheryConnectResponse

	err = json.NewDecoder(resp.Body).Decode(&connectResp)
	if err != nil {
		return fmt.Errorf("decode connect response: %w", err)
	}

	c.connectionID = connectResp.ConnectionID

	if c.logger != nil {
		c.logger.DebugContext(ctx, "Smithery connection established", "connectionId", c.connectionID)
	}

	return nil
}

// Initialize initializes the MCP connection.
func (c *SmitheryClient) Initialize(ctx context.Context, request mcpSDK.InitializeRequest) (*mcpSDK.InitializeResult, error) {
	// First ensure we have a connection.
	err := c.Connect(ctx)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}

	result, err := c.rpc(ctx, "initialize", request.Params)
	if err != nil {
		return nil, err
	}

	var initResult mcpSDK.InitializeResult

	err = json.Unmarshal(result, &initResult)
	if err != nil {
		return nil, fmt.Errorf("decode initialize result: %w", err)
	}

	c.mu.Lock()
	c.initialized = true
	c.serverInfo = &initResult.ServerInfo
	c.capabilities = initResult.Capabilities
	c.mu.Unlock()

	return &initResult, nil
}

// ListTools lists available tools from the MCP server.
func (c *SmitheryClient) ListTools(ctx context.Context, request mcpSDK.ListToolsRequest) (*mcpSDK.ListToolsResult, error) {
	result, err := c.rpc(ctx, "tools/list", request.Params)
	if err != nil {
		return nil, err
	}

	var toolsResult mcpSDK.ListToolsResult

	err = json.Unmarshal(result, &toolsResult)
	if err != nil {
		return nil, fmt.Errorf("decode tools/list result: %w", err)
	}

	return &toolsResult, nil
}

// CallTool calls a tool on the MCP server.
func (c *SmitheryClient) CallTool(ctx context.Context, request mcpSDK.CallToolRequest) (*mcpSDK.CallToolResult, error) {
	result, err := c.rpc(ctx, "tools/call", request.Params)
	if err != nil {
		return nil, err
	}

	var callResult mcpSDK.CallToolResult

	err = json.Unmarshal(result, &callResult)
	if err != nil {
		return nil, fmt.Errorf("decode tools/call result: %w", err)
	}

	return &callResult, nil
}

// Close closes the Smithery connection.
func (c *SmitheryClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Smithery connections are stateless on our end, just clear the ID.
	c.connectionID = ""
	c.initialized = false

	return nil
}

// IsInitialized returns true if the client has been initialized.
func (c *SmitheryClient) IsInitialized() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.initialized
}

// rpc makes an RPC call to the Smithery server.
func (c *SmitheryClient) rpc(ctx context.Context, method string, params any) (json.RawMessage, error) {
	c.mu.RLock()
	connectionID := c.connectionID
	c.mu.RUnlock()

	if connectionID == "" {
		return nil, ErrNotConnected
	}

	reqBody := smitheryRPCRequest{
		Method: method,
		Params: params,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal rpc request: %w", err)
	}

	rawURL := fmt.Sprintf("https://api.smithery.ai/connect/%s/%s/rpc", c.namespace, connectionID)

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("parse rpc URL: %w", err)
	}

	if parsedURL.Scheme != schemeHTTPS {
		return nil, fmt.Errorf("%w: got %q", ErrInvalidURLScheme, parsedURL.Scheme)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, parsedURL.String(), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create rpc request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	if c.logger != nil {
		c.logger.DebugContext(ctx, "Smithery RPC call", "method", method, "url", parsedURL.String())
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("rpc request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, fmt.Errorf("rpc failed with status %d (body unreadable): %w", resp.StatusCode, ErrRPCFailedWithStatus)
		}

		return nil, fmt.Errorf("rpc failed with status %d: %s: %w", resp.StatusCode, string(bodyBytes), ErrRPCFailedWithStatus)
	}

	var rpcResp smitheryRPCResponse

	err = json.NewDecoder(resp.Body).Decode(&rpcResp)
	if err != nil {
		return nil, fmt.Errorf("decode rpc response: %w", err)
	}

	if rpcResp.Error != nil {
		return nil, fmt.Errorf("rpc error %d: %s: %w", rpcResp.Error.Code, rpcResp.Error.Message, ErrRPCError)
	}

	return rpcResp.Result, nil
}
