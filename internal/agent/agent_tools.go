package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/orchestration"
	"github.com/dmytrogajewski/spin/internal/security"
	"github.com/dmytrogajewski/spin/internal/tools"
	"github.com/dmytrogajewski/spin/internal/types"
)

// ShouldApprove determines if a command needs user approval.
//
// Returns:
//   - needsApproval: true if the command requires approval
//   - reason: explanation of why approval is needed
func (a *Agent) ShouldApprove(cmd *security.Command) (bool, string) {
	// If approval is disabled, never require approval
	if !a.config.RequireApproval {
		return false, ""
	}

	// Classify the command via security service
	result, err := a.security.ValidateCommand(cmd)
	if err != nil {
		// On error, require approval for safety
		return true, fmt.Sprintf("Classification error: %v", err)
	}

	switch result.Classification {
	case security.CommandSafe:
		return false, ""

	case security.CommandInteractive:
		return true, "This command may modify files or system state"

	case security.CommandDangerous:
		return true, fmt.Sprintf("WARNING: Dangerous operation - %s", result.Reason)

	case security.CommandForbidden:
		// Forbidden commands should never be executed, even with approval
		// This will be handled by the executor
		return false, fmt.Sprintf("BLOCKED: %s", result.Reason)

	case security.CommandUnverified:
		return true, "Unknown command, approval required for safety"

	default:
		return true, "Unknown command classification, approval required"
	}
}

// processToolCalls handles all tool calls from an LLM response.
// It adds the assistant message with tool calls, executes each tool,
// and adds tool result messages to the conversation.
func (a *Agent) processToolCalls(ctx context.Context, messages []Message, llmResp *llm.CompletionResponse, resp *AgentResponse) []Message {
	// Create assistant message with tool calls
	assistantMsg := Message{
		Role:      RoleAssistant,
		Content:   llmResp.Content,
		Timestamp: time.Now(),
	}

	// Add assistant message FIRST (before tool results)
	messages = append(messages, assistantMsg)

	// Convert and process each tool call
	for i := range llmResp.ToolCalls {
		toolCall := &llmResp.ToolCalls[i]

		// Convert llm.ToolCall to orchestration.ToolCall
		coreToolCall := &orchestration.ToolCall{
			ID:   toolCall.ID,
			Type: toolCall.Type,
			Function: orchestration.ToolCallFunction{
				Name:      toolCall.Function.Name,
				Arguments: toolCall.Function.Arguments,
			},
		}

		// Add to assistant message (note: message already appended above)
		messages[len(messages)-1].ToolCalls = append(messages[len(messages)-1].ToolCalls, *coreToolCall)

		// Process the tool call (ProcessToolCall will emit EventToolCallStart)
		toolResult, err := a.ProcessToolCall(ctx, coreToolCall)
		if err != nil {

			// Add error message to conversation (after assistant message)
			messages = append(messages, Message{
				Role: RoleTool,
				Content: fmt.Sprintf("Tool %s failed: %v",
					coreToolCall.Function.Name, err),
				ToolCallID: coreToolCall.ID,
				Timestamp:  time.Now(),
			})
		} else {

			// Add tool result to conversation (after assistant message)
			slog.Debug("Agent tool result", "tool", coreToolCall.Function.Name, "output_len", len(toolResult.Output), "success", toolResult.Success)
			messages = append(messages, Message{
				Role:       RoleTool,
				Content:    getToolResultContent(coreToolCall, toolResult),
				ToolCallID: coreToolCall.ID,
				Timestamp:  time.Now(),
			})

			// Track tool call in response
			resp.ToolCalls = append(resp.ToolCalls, *coreToolCall)
		}
	}

	return messages
}

// ProcessToolCall processes a single tool call from the LLM.
//
// This method validates the tool call, parses arguments, executes the appropriate
// tool based on the function name, and returns the result. It handles:
// - Command execution with approval workflow
// - File operations (read, write, list)
// - Event emission for tool lifecycle
// - Error handling and recovery
func (a *Agent) ProcessToolCall(ctx context.Context, call *orchestration.ToolCall) (*orchestration.ToolResult, error) {
	// 1. Validate tool call
	if err := a.validateToolCall(call); err != nil {
		return &orchestration.ToolResult{
			ID:      call.ID,
			Success: false,
			Error:   err,
		}, nil // Return nil error so agent continues
	}

	// 2. Parse arguments
	args, err := a.parseToolArguments(call)
	if err != nil {
		return &orchestration.ToolResult{
			ID:      call.ID,
			Success: false,
			Error:   err,
		}, nil
	}

	// 3. Emit tool start event
	// Convert args to ToolCallArguments
	toolArgs, _ := types.FromMap(args)

	a.emitter.Emit(events.Event{
		Type:      events.EventToolCallStart,
		Timestamp: time.Now(),
		Data: events.ToolCallStartData{
			ToolID:     call.ID,
			ToolName:   call.Function.Name,
			Parameters: toolArgs,
		},
	})

	// 4. Execute tool via orchestration service
	result, err := a.orchestration.ExecuteTool(ctx, call)
	if err != nil {
		slog.Error("tool execution failed via orchestration", "tool", call.Function.Name, "error", err)
		// If orchestration returns an error, create error result
		result = &orchestration.ToolResult{
			ID:      call.ID,
			Success: false,
			Error:   err,
		}
	}

	// 5. Emit completion event
	completion := events.ToolCallCompleteData{
		ToolID:   call.ID,
		ToolName: call.Function.Name,
		Success:  result.Success,
	}
	if result.Success {
		completion.Output = result.Output
		slog.Debug("tool execution succeeded", "tool", call.Function.Name, "output_len", len(result.Output))
	} else if result.Error != nil {
		completion.Error = result.Error.Error()
		slog.Warn("tool execution failed", "tool", call.Function.Name, "error", result.Error.Error())
	}
	a.emitter.Emit(events.Event{
		Type:      events.EventToolCallComplete,
		Timestamp: time.Now(),
		Data:      completion,
	})

	return result, nil // Always return nil error so agent continues
}

// validateToolCall validates the tool call structure.
func (a *Agent) validateToolCall(call *orchestration.ToolCall) error {
	if call == nil {
		return errors.New("tool call cannot be nil")
	}
	if call.ID == "" {
		return errors.New("tool call ID cannot be empty")
	}
	if call.Function.Name == "" {
		return errors.New("tool function name cannot be empty")
	}
	return nil
}

// parseToolArguments extracts and parses JSON arguments from tool call.
func (a *Agent) parseToolArguments(call *orchestration.ToolCall) (map[string]interface{}, error) {
	if call.Function.Arguments == "" {
		return nil, errors.New("tool arguments cannot be empty")
	}

	var args map[string]interface{}
	if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
		return nil, fmt.Errorf("failed to parse tool arguments: %w", err)
	}

	return args, nil
}

// REMOVED: executeCommand method - now handled by ToolExecutor via OrchestrationService

// convertParameterSchemaToMap converts a ParameterSchema struct to map[string]interface{}.
// This is needed because LLM providers expect parameters as a JSON-compatible map.
func convertParameterSchemaToMap(params tools.ParameterSchema) map[string]interface{} {
	// Convert struct to map via JSON marshaling
	data, err := json.Marshal(params)
	if err != nil {
		// Fallback to empty object if marshaling fails
		return map[string]interface{}{"type": "object"}
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		// Fallback to empty object if unmarshaling fails
		return map[string]interface{}{"type": "object"}
	}

	return result
}

// getToolResultContent returns the appropriate content to send to LLM based on tool result.
// If tool succeeded, returns output. If failed, returns error message.
func getToolResultContent(toolCall *orchestration.ToolCall, result *orchestration.ToolResult) string {
	if result.Success {
		return result.Output
	}

	// Tool failed - send error message to LLM so it knows what went wrong
	if result.Error != nil {
		errorMsg := fmt.Sprintf("Tool %s failed: %v", toolCall.Function.Name, result.Error)
		slog.Debug("Tool failed, sending error to LLM", "tool", toolCall.Function.Name, "error", result.Error)
		return errorMsg
	}

	// Edge case: not successful but no error message
	return fmt.Sprintf("Tool %s failed with no error message", toolCall.Function.Name)
}
