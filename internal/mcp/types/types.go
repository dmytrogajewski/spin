// Package types provides Go type definitions for the Model Context Protocol (MCP).
//
// The types in this package correspond to the MCP specification and use standard
// encoding/json for serialization.
//
// Specification: https://modelcontextprotocol.io/specification
package types

import "encoding/json"

// Implementation describes client or server information.
type Implementation struct {
	// Name is the name of the implementation (e.g., "spin", "claude-desktop")
	Name string `json:"name"`

	// Version is the version of the implementation (e.g., "0.1.0")
	Version string `json:"version"`
}

// Tool represents an MCP tool that can be invoked.
type Tool struct {
	// Name is the unique identifier for the tool
	Name string `json:"name"`

	// Description provides a human-readable description of the tool's purpose
	Description *string `json:"description,omitempty"`

	// InputSchema is a JSON Schema describing the tool's parameters
	InputSchema json.RawMessage `json:"inputSchema"`
}

// Resource represents an MCP resource that can be read.
type Resource struct {
	// URI is the unique identifier for the resource
	URI string `json:"uri"`

	// Name is a human-readable name for the resource
	Name string `json:"name"`

	// Description provides additional context about the resource
	Description *string `json:"description,omitempty"`

	// MimeType indicates the resource's content type
	MimeType *string `json:"mimeType,omitempty"`
}

// ResourceContents contains the actual data of a resource.
type ResourceContents struct {
	// URI is the identifier of the resource
	URI string `json:"uri"`

	// MimeType indicates the content type
	MimeType *string `json:"mimeType,omitempty"`

	// Text contains text content (mutually exclusive with Blob)
	Text *string `json:"text,omitempty"`

	// Blob contains base64-encoded binary content
	Blob *string `json:"blob,omitempty"`
}

// Prompt represents an MCP prompt template.
type Prompt struct {
	// Name is the unique identifier for the prompt
	Name string `json:"name"`

	// Description provides context about the prompt's purpose
	Description *string `json:"description,omitempty"`

	// Arguments defines the parameters this prompt accepts
	Arguments []PromptArgument `json:"arguments,omitempty"`
}

// PromptArgument defines a parameter for a prompt.
type PromptArgument struct {
	// Name is the parameter name
	Name string `json:"name"`

	// Description explains the parameter's purpose
	Description *string `json:"description,omitempty"`

	// Required indicates if this parameter must be provided
	Required bool `json:"required,omitempty"`
}

// PromptMessage is a message in a prompt template.
type PromptMessage struct {
	// Role is either "user" or "assistant"
	Role string `json:"role"`

	// Content is the message content
	Content []Content `json:"content"`
}

// Content represents different types of content in messages.
type Content struct {
	// Type is one of: "text", "image", "resource"
	Type string `json:"type"`

	// Text contains text content (for type="text")
	Text *string `json:"text,omitempty"`

	// Data contains base64-encoded data (for type="image")
	Data *string `json:"data,omitempty"`

	// MimeType specifies the content type (for type="image" or type="resource")
	MimeType *string `json:"mimeType,omitempty"`

	// URI references a resource (for type="resource")
	URI *string `json:"uri,omitempty"`
}

// TextContent creates a text content object.
func TextContent(text string) Content {
	return Content{
		Type: "text",
		Text: &text,
	}
}

// ImageContent creates an image content object with base64-encoded data.
func ImageContent(base64Data, mimeType string) Content {
	return Content{
		Type:     "image",
		Data:     &base64Data,
		MimeType: &mimeType,
	}
}

// ResourceContent creates a resource content object.
func ResourceContent(uri string, mimeType *string) Content {
	return Content{
		Type:     "resource",
		URI:      &uri,
		MimeType: mimeType,
	}
}
