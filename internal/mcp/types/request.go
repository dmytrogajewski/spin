package types

import "encoding/json"

// InitializeRequest initializes an MCP connection.
type InitializeRequest struct {
	// ProtocolVersion is the MCP protocol version (e.g., "2024-11-05")
	ProtocolVersion string `json:"protocolVersion"`

	// Capabilities describes the client's capabilities
	Capabilities ClientCapabilities `json:"capabilities"`

	// ClientInfo provides information about the client
	ClientInfo Implementation `json:"clientInfo"`
}

// ListToolsRequest lists available tools from a server.
type ListToolsRequest struct {
	// Cursor for pagination (optional)
	Cursor *string `json:"cursor,omitempty"`
}

// CallToolRequest invokes a tool.
type CallToolRequest struct {
	// Name is the tool identifier
	Name string `json:"name"`

	// Arguments contains the tool's input parameters as JSON
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

// ListResourcesRequest lists available resources from a server.
type ListResourcesRequest struct {
	// Cursor for pagination (optional)
	Cursor *string `json:"cursor,omitempty"`
}

// ReadResourceRequest reads a specific resource.
type ReadResourceRequest struct {
	// URI is the resource identifier
	URI string `json:"uri"`
}

// ListPromptsRequest lists available prompts from a server.
type ListPromptsRequest struct {
	// Cursor for pagination (optional)
	Cursor *string `json:"cursor,omitempty"`
}

// GetPromptRequest retrieves a specific prompt.
type GetPromptRequest struct {
	// Name is the prompt identifier
	Name string `json:"name"`

	// Arguments provides values for prompt parameters
	Arguments map[string]string `json:"arguments,omitempty"`
}
