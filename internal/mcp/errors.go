package mcp

import "errors"

// Shared sentinel errors for the mcp package.
var (
	// ErrRegistryNameRequired is returned when a registry name is missing.
	ErrRegistryNameRequired = errors.New("registry name is required")
	// ErrCommandRequiredForLocalRegistry is a sentinel error.
	ErrCommandRequiredForLocalRegistry = errors.New("command is required for local registry")
	// ErrURLRequiredForRemoteRegistry is a sentinel error.
	ErrURLRequiredForRemoteRegistry = errors.New("URL is required for remote registry")
	// ErrTransportMustBeSseOrStreamable is a sentinel error.
	ErrTransportMustBeSseOrStreamable = errors.New("transport must be 'sse' or 'streamable-http'")
	// ErrUnsupportedTransport is a sentinel error.
	ErrUnsupportedTransport = errors.New("unsupported transport type")
	// ErrAPIKeyRequiredForSmithery is a sentinel error.
	ErrAPIKeyRequiredForSmithery = errors.New("API key is required for smithery registry")
	// ErrNamespaceRequiredForSmithery is a sentinel error.
	ErrNamespaceRequiredForSmithery = errors.New("namespace is required for static smithery registry")
	// ErrAPIClientNotInitialized is a sentinel error.
	ErrAPIClientNotInitialized = errors.New("API client not initialized")

	// ErrSmitheryAPIKeyRequired is returned when the Smithery API key is missing.
	ErrSmitheryAPIKeyRequired = errors.New("smithery API key is required")
	// ErrSmitheryMcpURLRequired is a sentinel error.
	ErrSmitheryMcpURLRequired = errors.New("smithery MCP URL is required")
	// ErrSmitheryNamespaceRequired is a sentinel error.
	ErrSmitheryNamespaceRequired = errors.New("smithery namespace is required")
	// ErrConnectFailedWithStatus is a sentinel error.
	ErrConnectFailedWithStatus = errors.New("connect failed with status")
	// ErrNotConnected is a sentinel error.
	ErrNotConnected = errors.New("not connected")
	// ErrRPCFailedWithStatus is a sentinel error.
	ErrRPCFailedWithStatus = errors.New("rpc failed with status")
	// ErrRPCError is a sentinel error.
	ErrRPCError = errors.New("rpc error")
	// ErrAPIErrorStatus is a sentinel error.
	ErrAPIErrorStatus = errors.New("API error")
	// ErrInvalidURLScheme is returned when a URL has an unexpected scheme.
	ErrInvalidURLScheme = errors.New("invalid URL scheme, expected https")
)
