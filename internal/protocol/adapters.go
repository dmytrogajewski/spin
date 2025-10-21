package protocol

import (
	"encoding/json"

	"github.com/dmytrogajewski/spin/internal/events"
)

// Adapters convert between protocol types and core types

// FromCoreEvent converts an events.Event to a protocol.Message
func FromCoreEvent(event events.Event) (Message, bool) {
	switch event.Type {
	case events.EventContentDelta:
		if data, ok := event.Data.(events.ContentDeltaData); ok {
			return NewAssistantDeltaMessage(AssistantDelta{
				Delta: data.Content,
			}), true
		}

	case events.EventToolCallStart:
		if data, ok := event.Data.(events.ToolCallStartData); ok {
			argsJSON, _ := json.Marshal(data.Parameters.ToMap())
			return NewToolCallProposedMessage(ToolCallProposed{
				ToolCallID:       data.ToolID,
				ToolName:         data.ToolName,
				Arguments:        json.RawMessage(argsJSON),
				RequiresApproval: false, // TODO: Add to events.ToolCallData
			}), true
		}

	case events.EventToolCallProgress:
		if data, ok := event.Data.(events.ToolProgressData); ok {
			return NewToolCallExecutingMessage(ToolCallExecuting{
				ToolCallID: data.ToolID,
			}), true
		}

	case events.EventToolCallComplete:
		if data, ok := event.Data.(events.ToolCallCompleteData); ok {
			var result ToolResult
			if !data.Success && data.Error != "" {
				result = NewErrorResult(data.Error)
			} else {
				result = NewSuccessResult(data.Output)
			}
			return NewToolCallResultMessage(ToolCallResult{
				ToolCallID: data.ToolID,
				Result:     result,
			}), true
		}

	case events.EventError:
		// Try both string and ErrorData
		if str, ok := event.Data.(string); ok {
			return NewStatusUpdateMessage(StatusUpdate{
				Message: str,
				Level:   StatusLevelError,
			}), true
		}
		if data, ok := event.Data.(events.ErrorData); ok {
			return NewStatusUpdateMessage(StatusUpdate{
				Message: data.Message + " " + data.Details,
				Level:   StatusLevelError,
			}), true
		}

	case events.EventWarning:
		if data, ok := event.Data.(string); ok {
			return NewStatusUpdateMessage(StatusUpdate{
				Message: data,
				Level:   StatusLevelWarning,
			}), true
		}

	case events.EventInfo:
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
