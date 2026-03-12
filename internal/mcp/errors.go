package mcp

import "errors"

// Shared sentinel errors for the mcp package.
var (
	// Registry errors.
	ErrRegistryNameRequired              = errors.New("registry name is required")
	ErrToolNotFound                      = errors.New("tool not found")
	ErrCommandRequiredForLocalRegistry   = errors.New("command is required for local registry")
	ErrURLRequiredForRemoteRegistry      = errors.New("URL is required for remote registry")
	ErrTransportMustBeSseOrStreamable    = errors.New("transport must be 'sse' or 'streamable-http'")
	ErrUnsupportedTransport              = errors.New("unsupported transport type")
	ErrAPIKeyRequiredForSmithery         = errors.New("API key is required for smithery registry")
	ErrNamespaceRequiredForSmithery      = errors.New("namespace is required for static smithery registry")
	ErrAPIClientNotInitialized           = errors.New("API client not initialized")

	// Smithery client errors.
	ErrSmitheryAPIKeyRequired            = errors.New("smithery API key is required")
	ErrSmitheryMcpURLRequired            = errors.New("smithery MCP URL is required")
	ErrSmitheryNamespaceRequired         = errors.New("smithery namespace is required")
	ErrConnectFailedWithStatus           = errors.New("connect failed with status")
	ErrNotConnected                      = errors.New("not connected")
	ErrRPCFailedWithStatus               = errors.New("rpc failed with status")
	ErrRPCError                          = errors.New("rpc error")
	ErrAPIErrorStatus                    = errors.New("API error")
)
