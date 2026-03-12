package mcp

import "errors"

// Shared sentinel errors for the mcp package.
var (
	// Registry errors.
	ErrRegistryNameRequired              = errors.New("registry name is required")
	ErrToolNotFound                      = errors.New("tool not found")
	ErrCommandRequiredForLocalRegistry   = errors.New("command is required for local registry")
	ErrUrlRequiredForRemoteRegistry      = errors.New("URL is required for remote registry")
	ErrTransportMustBeSseOrStreamable    = errors.New("transport must be 'sse' or 'streamable-http'")
	ErrUnsupportedTransport              = errors.New("unsupported transport type")
	ErrApiKeyRequiredForSmithery         = errors.New("API key is required for smithery registry")
	ErrNamespaceRequiredForSmithery      = errors.New("namespace is required for static smithery registry")
	ErrApiClientNotInitialized           = errors.New("API client not initialized")

	// Smithery client errors.
	ErrSmitheryApiKeyRequired            = errors.New("smithery API key is required")
	ErrSmitheryMcpUrlRequired            = errors.New("smithery MCP URL is required")
	ErrSmitheryNamespaceRequired         = errors.New("smithery namespace is required")
	ErrConnectFailedWithStatus           = errors.New("connect failed with status")
	ErrNotConnected                      = errors.New("not connected")
	ErrRpcFailedWithStatus               = errors.New("rpc failed with status")
	ErrRpcError                          = errors.New("rpc error")
	ErrApiErrorStatus                    = errors.New("API error")
)
