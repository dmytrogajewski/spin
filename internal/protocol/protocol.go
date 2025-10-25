package protocol

import (
	"encoding/json"
	"fmt"
	"time"
)

// Inbound Messages (UI → Core)

// UserMessage represents user's textual input
type UserMessage struct {
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// ToolApproval is user's approval/rejection of a proposed tool call
type ToolApproval struct {
	ToolCallID   string           `json:"tool_call_id"`
	Approved     bool             `json:"approved"`
	ModifiedArgs *json.RawMessage `json:"modified_args,omitempty"`
}

// CancelRequest requests cancellation of an in-progress turn
type CancelRequest struct {
	TurnID string `json:"turn_id"`
}

// Outbound Messages (Core → UI)

// TurnStart signals the beginning of a conversation turn
type TurnStart struct {
	TurnID      string `json:"turn_id"`
	UserMessage string `json:"user_message"`
}

// AssistantDelta contains incremental text from the AI model (streaming)
type AssistantDelta struct {
	Delta     string  `json:"delta"`
	Reasoning *string `json:"reasoning,omitempty"`
}

// ToolCallProposed indicates AI proposes a tool invocation
type ToolCallProposed struct {
	ToolCallID       string          `json:"tool_call_id"`
	ToolName         string          `json:"tool_name"`
	Arguments        json.RawMessage `json:"arguments"`
	RequiresApproval bool            `json:"requires_approval"`
}

// ToolCallExecuting indicates tool execution has started
type ToolCallExecuting struct {
	ToolCallID string `json:"tool_call_id"`
}

// ToolCallResult contains tool execution completion status
type ToolCallResult struct {
	ToolCallID string     `json:"tool_call_id"`
	Result     ToolResult `json:"result"`
}

// ToolResult represents success or failure of tool execution
type ToolResult struct {
	Success *ToolSuccess `json:"success,omitempty"`
	Error   *ToolError   `json:"error,omitempty"`
}

// ToolSuccess contains successful tool output
type ToolSuccess struct {
	Output string `json:"output"`
}

// ToolError contains error message from failed tool
type ToolError struct {
	Message string `json:"message"`
}

// NewSuccessResult creates a successful tool result
func NewSuccessResult(output string) ToolResult {
	return ToolResult{Success: &ToolSuccess{Output: output}}
}

// NewErrorResult creates an error tool result
func NewErrorResult(message string) ToolResult {
	return ToolResult{Error: &ToolError{Message: message}}
}

// TurnComplete signals conversation turn finished
type TurnComplete struct {
	TurnID       string `json:"turn_id"`
	FinalMessage string `json:"final_message"`
}

// StatusUpdate is a status message for display to user
type StatusUpdate struct {
	Message string      `json:"message"`
	Level   StatusLevel `json:"level"`
}

// StatusLevel indicates severity of status message
type StatusLevel string

const (
	StatusLevelInfo    StatusLevel = "info"
	StatusLevelWarning StatusLevel = "warning"
	StatusLevelError   StatusLevel = "error"
)

// Message Envelope

// Message is a tagged union of all protocol messages
type Message struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// NewTurnStartMessage creates a turn_start message

// NewAssistantDeltaMessage creates an assistant_delta message
func NewAssistantDeltaMessage(ad AssistantDelta) Message {
	data, _ := json.Marshal(ad)
	return Message{Type: "assistant_delta", Data: data}
}

// NewToolCallProposedMessage creates a tool_call_proposed message
func NewToolCallProposedMessage(tcp ToolCallProposed) Message {
	data, _ := json.Marshal(tcp)
	return Message{Type: "tool_call_proposed", Data: data}
}

// NewToolCallExecutingMessage creates a tool_call_executing message
func NewToolCallExecutingMessage(tce ToolCallExecuting) Message {
	data, _ := json.Marshal(tce)
	return Message{Type: "tool_call_executing", Data: data}
}

// NewToolCallResultMessage creates a tool_call_result message
func NewToolCallResultMessage(tcr ToolCallResult) Message {
	data, _ := json.Marshal(tcr)
	return Message{Type: "tool_call_result", Data: data}
}

// NewTurnCompleteMessage creates a turn_complete message

// NewStatusUpdateMessage creates a status_update message
func NewStatusUpdateMessage(su StatusUpdate) Message {
	data, _ := json.Marshal(su)
	return Message{Type: "status_update", Data: data}
}

// ParseMessage parses a message by type
func ParseMessage(msg Message) (interface{}, error) {
	parser := getMessageParser(msg.Type)
	if parser == nil {
		return nil, fmt.Errorf("unknown message type: %s", msg.Type)
	}
	return parser(msg.Data)
}

// messageParser is a function that parses message data into a specific type.
type messageParser func([]byte) (interface{}, error)

// getMessageParser returns the appropriate parser for a message type.
func getMessageParser(msgType string) messageParser {
	switch msgType {
	case "turn_start":
		return parseTurnStart
	case "assistant_delta":
		return parseAssistantDelta
	case "tool_call_proposed":
		return parseToolCallProposed
	case "tool_call_executing":
		return parseToolCallExecuting
	case "tool_call_result":
		return parseToolCallResult
	case "turn_complete":
		return parseTurnComplete
	case "status_update":
		return parseStatusUpdate
	default:
		return nil
	}
}

// parseTurnStart parses a TurnStart message.
func parseTurnStart(data []byte) (interface{}, error) {
	var ts TurnStart
	if err := json.Unmarshal(data, &ts); err != nil {
		return nil, err
	}
	return ts, nil
}

// parseAssistantDelta parses an AssistantDelta message.
func parseAssistantDelta(data []byte) (interface{}, error) {
	var ad AssistantDelta
	if err := json.Unmarshal(data, &ad); err != nil {
		return nil, err
	}
	return ad, nil
}

// parseToolCallProposed parses a ToolCallProposed message.
func parseToolCallProposed(data []byte) (interface{}, error) {
	var tcp ToolCallProposed
	if err := json.Unmarshal(data, &tcp); err != nil {
		return nil, err
	}
	return tcp, nil
}

// parseToolCallExecuting parses a ToolCallExecuting message.
func parseToolCallExecuting(data []byte) (interface{}, error) {
	var tce ToolCallExecuting
	if err := json.Unmarshal(data, &tce); err != nil {
		return nil, err
	}
	return tce, nil
}

// parseToolCallResult parses a ToolCallResult message.
func parseToolCallResult(data []byte) (interface{}, error) {
	var tcr ToolCallResult
	if err := json.Unmarshal(data, &tcr); err != nil {
		return nil, err
	}
	return tcr, nil
}

// parseTurnComplete parses a TurnComplete message.
func parseTurnComplete(data []byte) (interface{}, error) {
	var tc TurnComplete
	if err := json.Unmarshal(data, &tc); err != nil {
		return nil, err
	}
	return tc, nil
}

// parseStatusUpdate parses a StatusUpdate message.
func parseStatusUpdate(data []byte) (interface{}, error) {
	var su StatusUpdate
	if err := json.Unmarshal(data, &su); err != nil {
		return nil, err
	}
	return su, nil
}
