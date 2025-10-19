package protocol

// AI Models - Types matching OpenAI-compatible API response formats

// ResponseItem represents different types of AI responses
type ResponseItem struct {
	Message   *MessageItem   `json:"message,omitempty"`
	ToolCall  *ToolCallItem  `json:"tool_call,omitempty"`
	Reasoning *ReasoningItem `json:"reasoning,omitempty"`
}

// MessageItem contains message content
type MessageItem struct {
	Content []ContentItem `json:"content"`
}

// ContentItem represents different content types
type ContentItem struct {
	Type        string              `json:"type"` // "text", "image", "file_pointer"
	Text        *TextContent        `json:"text,omitempty"`
	Image       *ImageContent       `json:"image,omitempty"`
	FilePointer *FilePointerContent `json:"file_pointer,omitempty"`
}

// TextContent contains plain text
type TextContent struct {
	Text string `json:"text"`
}

// ImageContent contains image data
type ImageContent struct {
	URL      *string `json:"url,omitempty"`
	Data     *string `json:"data,omitempty"` // Base64
	MimeType string  `json:"mime_type,omitempty"`
}

// FilePointerContent references a file
type FilePointerContent struct {
	Path     string  `json:"path"`
	MimeType *string `json:"mime_type,omitempty"`
}

// ToolCallItem represents a tool invocation
type ToolCallItem struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON-encoded
}

// ReasoningItem contains AI's internal reasoning (for models like o1/o3)
type ReasoningItem struct {
	Reasoning string `json:"reasoning"`
}

// LocalShellAction represents shell command execution
type LocalShellAction struct {
	Command string           `json:"command"`
	Status  LocalShellStatus `json:"status"`
}

// LocalShellStatus tracks shell command state
type LocalShellStatus struct {
	Pending   *struct{}       `json:"pending,omitempty"`
	Running   *struct{}       `json:"running,omitempty"`
	Completed *ShellCompleted `json:"completed,omitempty"`
	Failed    *ShellFailed    `json:"failed,omitempty"`
}

// ShellCompleted contains successful execution result
type ShellCompleted struct {
	ExitCode int    `json:"exit_code"`
	Output   string `json:"output"`
}

// ShellFailed contains error information
type ShellFailed struct {
	Error string `json:"error"`
}
