package a2a

import "encoding/json"

// Role identifies the sender of a Message.
type Role string

const (
	// RoleUnspecified is an unset role.
	RoleUnspecified Role = "ROLE_UNSPECIFIED"
	// RoleUser is a client-to-server message.
	RoleUser Role = "ROLE_USER"
	// RoleAgent is a server-to-client message.
	RoleAgent Role = "ROLE_AGENT"
)

// Message is one A2A communication turn.
type Message struct {
	ContextID string `json:"contextId,omitempty"`
	MessageID string `json:"messageId"`
	Parts     []Part `json:"parts"`
	Role      Role   `json:"role"`
	TaskID    string `json:"taskId,omitempty"`
}

// Part is a content fragment inside a Message or Artifact.
// A Part should carry exactly one of Text, Raw, URL, or Data.
type Part struct {
	Data      json.RawMessage `json:"data,omitempty"`
	Filename  string          `json:"filename,omitempty"`
	MediaType string          `json:"mediaType,omitempty"`
	Raw       []byte          `json:"raw,omitempty"`
	Text      string          `json:"text,omitempty"`
	URL       string          `json:"url,omitempty"`
}

// Artifact is a task output composed of Parts.
type Artifact struct {
	ArtifactID  string `json:"artifactId"`
	Description string `json:"description,omitempty"`
	Name        string `json:"name,omitempty"`
	Parts       []Part `json:"parts"`
}
