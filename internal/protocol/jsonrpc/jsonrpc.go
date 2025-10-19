package jsonrpc

import (
	"encoding/json"
	"fmt"
)

// Request represents a JSON-RPC 2.0 request
type Request struct {
	JSONRPC string          `json:"jsonrpc"` // Always "2.0"
	ID      *RequestID      `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// RequestID can be string or number
type RequestID struct {
	Str *string
	Num *int64
}

// MarshalJSON implements custom marshaling for RequestID
func (r RequestID) MarshalJSON() ([]byte, error) {
	if r.Str != nil {
		return json.Marshal(*r.Str)
	}
	if r.Num != nil {
		return json.Marshal(*r.Num)
	}
	return []byte("null"), nil
}

// UnmarshalJSON implements custom unmarshaling for RequestID
func (r *RequestID) UnmarshalJSON(data []byte) error {
	// Try string first
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		r.Str = &s
		return nil
	}

	// Try number
	var n int64
	if err := json.Unmarshal(data, &n); err == nil {
		r.Num = &n
		return nil
	}

	return fmt.Errorf("request id must be string or number")
}

// String returns string representation
func (r RequestID) String() string {
	if r.Str != nil {
		return *r.Str
	}
	if r.Num != nil {
		return fmt.Sprintf("%d", *r.Num)
	}
	return ""
}

// Response represents a JSON-RPC 2.0 response
type Response struct {
	JSONRPC string          `json:"jsonrpc"` // Always "2.0"
	ID      RequestID       `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
}

// Error represents a JSON-RPC error
type Error struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// Error implements the error interface
func (e *Error) Error() string {
	return fmt.Sprintf("JSON-RPC error %d: %s", e.Code, e.Message)
}

// Standard error codes
const (
	ParseError     = -32700 // Invalid JSON
	InvalidRequest = -32600 // Invalid Request object
	MethodNotFound = -32601 // Method not found
	InvalidParams  = -32602 // Invalid method parameters
	InternalError  = -32603 // Internal JSON-RPC error
)

// Application error codes (custom)
const (
	ConversationNotFound = -32000 // Conversation not found
	InvalidState         = -32001 // Invalid conversation state
	ExecutionError       = -32002 // Execution error
)

// NewError creates a new error
func NewError(code int, message string) *Error {
	return &Error{Code: code, Message: message}
}

// Notification represents a JSON-RPC notification (no response expected)
type Notification struct {
	JSONRPC string          `json:"jsonrpc"` // Always "2.0"
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// App-Server Protocol Messages

// InitializeParams contains initialization parameters
type InitializeParams struct {
	WorkspacePath string                 `json:"workspace_path"`
	Config        map[string]interface{} `json:"config,omitempty"`
}

// InitializeResult is the response to initialize
type InitializeResult struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

// SendMessageParams contains message sending parameters
type SendMessageParams struct {
	// ConversationID identifies the conversation (nil = new conversation)
	ConversationID *string `json:"conversation_id,omitempty"`

	// Message is the user's input
	Message string `json:"message"`

	// TaskMode optionally specifies the task mode to use for this turn.
	// Valid values: "regular", "review", "compact", "planning"
	// If nil, uses the conversation's current mode (default: "regular")
	TaskMode *string `json:"task_mode,omitempty"`
}

// SendMessageResult is the response to send_message
type SendMessageResult struct {
	// ConversationID uniquely identifies the conversation
	ConversationID string `json:"conversation_id"`

	// TurnID uniquely identifies this turn
	TurnID string `json:"turn_id"`

	// TaskMode is the current task mode for the conversation
	TaskMode string `json:"task_mode"`
}

// ApproveToolParams contains tool approval parameters
type ApproveToolParams struct {
	ToolCallID   string           `json:"tool_call_id"`
	Approved     bool             `json:"approved"`
	ModifiedArgs *json.RawMessage `json:"modified_args,omitempty"`
}

// ApproveToolResult is the response to approve_tool
type ApproveToolResult struct {
	Status string `json:"status"`
}

// CancelTurnParams contains turn cancellation parameters
type CancelTurnParams struct {
	TurnID string `json:"turn_id"`
}

// CancelTurnResult is the response to cancel_turn
type CancelTurnResult struct {
	Status string `json:"status"`
}

// SearchFilesParams contains file search parameters
type SearchFilesParams struct {
	Query string `json:"query"`
	Limit int    `json:"limit,omitempty"`
}

// SearchFilesResult is the response to search_files
type SearchFilesResult struct {
	Files []FileMatch `json:"files"`
}

// FileMatch represents a matched file
type FileMatch struct {
	Path  string  `json:"path"`
	Score float64 `json:"score,omitempty"`
}

// ValidTaskModes are the allowed task mode names
var ValidTaskModes = map[string]bool{
	"regular":  true,
	"review":   true,
	"compact":  true,
	"planning": true,
}

// ValidateTaskMode checks if a task mode name is valid.
// Empty string is valid (means use default).
func ValidateTaskMode(mode string) error {
	if mode == "" {
		return nil
	}
	if !ValidTaskModes[mode] {
		return fmt.Errorf("invalid task mode: %s (valid: regular, review, compact, planning)", mode)
	}
	return nil
}
