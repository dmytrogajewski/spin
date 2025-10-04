package types

// ClientCapabilities describes the features a client supports.
type ClientCapabilities struct {
	// Tools indicates the client can handle tools
	Tools *ToolsCapability `json:"tools,omitempty"`

	// Resources indicates the client can handle resources
	Resources *ResourcesCapability `json:"resources,omitempty"`

	// Prompts indicates the client can handle prompts
	Prompts *PromptsCapability `json:"prompts,omitempty"`
}

// ServerCapabilities describes the features a server provides.
type ServerCapabilities struct {
	// Tools indicates the server provides tools
	Tools *ToolsCapability `json:"tools,omitempty"`

	// Resources indicates the server provides resources
	Resources *ResourcesCapability `json:"resources,omitempty"`

	// Prompts indicates the server provides prompts
	Prompts *PromptsCapability `json:"prompts,omitempty"`
}

// ToolsCapability describes tool-related capabilities.
type ToolsCapability struct {
	// ListChanged indicates support for tool list change notifications
	ListChanged bool `json:"listChanged,omitempty"`
}

// ResourcesCapability describes resource-related capabilities.
type ResourcesCapability struct {
	// Subscribe indicates support for resource subscriptions
	Subscribe bool `json:"subscribe,omitempty"`

	// ListChanged indicates support for resource list change notifications
	ListChanged bool `json:"listChanged,omitempty"`
}

// PromptsCapability describes prompt-related capabilities.
type PromptsCapability struct {
	// ListChanged indicates support for prompt list change notifications
	ListChanged bool `json:"listChanged,omitempty"`
}
