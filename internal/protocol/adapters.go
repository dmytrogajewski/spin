package protocol

import (
	"encoding/json"

	"github.com/dmytrogajewski/spin/internal/core"
)

// Adapters convert between protocol types and core types

// FromCoreEvent converts a core.Event to a protocol.Message
func FromCoreEvent(event core.Event) (Message, bool) {
	switch event.Type {
	case core.EventContentDelta:
		if data, ok := event.Data.(core.ContentDeltaData); ok {
			return NewAssistantDeltaMessage(AssistantDelta{
				Delta: data.Content,
			}), true
		}

	case core.EventToolCallStart:
		if data, ok := event.Data.(core.ToolCallData); ok {
			argsJSON, _ := json.Marshal(data.Parameters)
			return NewToolCallProposedMessage(ToolCallProposed{
				ToolCallID:       data.ToolID,
				ToolName:         data.ToolName,
				Arguments:        json.RawMessage(argsJSON),
				RequiresApproval: false, // TODO: Add to core.ToolCallData
			}), true
		}

	case core.EventToolCallProgress:
		if data, ok := event.Data.(core.ToolProgressData); ok {
			return NewToolCallExecutingMessage(ToolCallExecuting{
				ToolCallID: data.ToolID,
			}), true
		}

	case core.EventToolCallComplete:
		if data, ok := event.Data.(core.ToolResultData); ok {
			var result ToolResult
			if data.Error != "" {
				result = NewErrorResult(data.Error)
			} else {
				result = NewSuccessResult(data.Result)
			}
			return NewToolCallResultMessage(ToolCallResult{
				ToolCallID: data.ToolID,
				Result:     result,
			}), true
		}

	case core.EventError:
		// Try both string and ErrorData
		if str, ok := event.Data.(string); ok {
			return NewStatusUpdateMessage(StatusUpdate{
				Message: str,
				Level:   StatusLevelError,
			}), true
		}
		if data, ok := event.Data.(core.ErrorData); ok {
			return NewStatusUpdateMessage(StatusUpdate{
				Message: data.Message + " " + data.Details,
				Level:   StatusLevelError,
			}), true
		}

	case core.EventWarning:
		if data, ok := event.Data.(string); ok {
			return NewStatusUpdateMessage(StatusUpdate{
				Message: data,
				Level:   StatusLevelWarning,
			}), true
		}

	case core.EventInfo:
		if data, ok := event.Data.(string); ok {
			return NewStatusUpdateMessage(StatusUpdate{
				Message: data,
				Level:   StatusLevelInfo,
			}), true
		}
	}

	// Event type not mapped or data type incorrect
	return Message{}, false
}
