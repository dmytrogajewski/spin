package types

// InitializeResponse contains server initialization information.
type InitializeResponse struct {
	// ProtocolVersion is the MCP protocol version the server supports
	ProtocolVersion string `json:"protocolVersion"`

	// Capabilities describes the server's capabilities
	Capabilities ServerCapabilities `json:"capabilities"`

	// ServerInfo provides information about the server
	ServerInfo Implementation `json:"serverInfo"`
}

// ListToolsResponse contains a list of available tools.
type ListToolsResponse struct {
	// Tools is the list of available tools
	Tools []Tool `json:"tools"`

	// NextCursor for pagination (optional)
	NextCursor *string `json:"nextCursor,omitempty"`
}

// CallToolResponse contains the result of a tool invocation.
type CallToolResponse struct {
	// Content is the tool's output
	Content []Content `json:"content"`

	// IsError indicates if the tool execution failed
	IsError bool `json:"isError,omitempty"`
}

// ListResourcesResponse contains a list of available resources.
type ListResourcesResponse struct {
	// Resources is the list of available resources
	Resources []Resource `json:"resources"`

	// NextCursor for pagination (optional)
	NextCursor *string `json:"nextCursor,omitempty"`
}

// ReadResourceResponse contains resource contents.
type ReadResourceResponse struct {
	// Contents contains the resource data
	Contents []ResourceContents `json:"contents"`
}

// ListPromptsResponse contains a list of available prompts.
type ListPromptsResponse struct {
	// Prompts is the list of available prompts
	Prompts []Prompt `json:"prompts"`

	// NextCursor for pagination (optional)
	NextCursor *string `json:"nextCursor,omitempty"`
}

// GetPromptResponse contains prompt details.
type GetPromptResponse struct {
	// Description provides context about the prompt
	Description *string `json:"description,omitempty"`

	// Messages contains the prompt template messages
	Messages []PromptMessage `json:"messages"`
}
