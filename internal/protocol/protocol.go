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
func NewTurnStartMessage(ts TurnStart) Message {
	data, _ := json.Marshal(ts)
	return Message{Type: "turn_start", Data: data}
}

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
func NewTurnCompleteMessage(tc TurnComplete) Message {
	data, _ := json.Marshal(tc)
	return Message{Type: "turn_complete", Data: data}
}

// NewStatusUpdateMessage creates a status_update message
func NewStatusUpdateMessage(su StatusUpdate) Message {
	data, _ := json.Marshal(su)
	return Message{Type: "status_update", Data: data}
}

// ParseMessage parses a message by type
func ParseMessage(msg Message) (interface{}, error) {
	switch msg.Type {
	case "turn_start":
		var ts TurnStart
		if err := json.Unmarshal(msg.Data, &ts); err != nil {
			return nil, err
		}
		return ts, nil
	case "assistant_delta":
		var ad AssistantDelta
		if err := json.Unmarshal(msg.Data, &ad); err != nil {
			return nil, err
		}
		return ad, nil
	case "tool_call_proposed":
		var tcp ToolCallProposed
		if err := json.Unmarshal(msg.Data, &tcp); err != nil {
			return nil, err
		}
		return tcp, nil
	case "tool_call_executing":
		var tce ToolCallExecuting
		if err := json.Unmarshal(msg.Data, &tce); err != nil {
			return nil, err
		}
		return tce, nil
	case "tool_call_result":
		var tcr ToolCallResult
		if err := json.Unmarshal(msg.Data, &tcr); err != nil {
			return nil, err
		}
		return tcr, nil
	case "turn_complete":
		var tc TurnComplete
		if err := json.Unmarshal(msg.Data, &tc); err != nil {
			return nil, err
		}
		return tc, nil
	case "status_update":
		var su StatusUpdate
		if err := json.Unmarshal(msg.Data, &su); err != nil {
			return nil, err
		}
		return su, nil
	default:
		return nil, fmt.Errorf("unknown message type: %s", msg.Type)
	}
}
