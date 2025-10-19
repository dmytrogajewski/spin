package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/tools"
	"github.com/dmytrogajewski/spin/internal/types"
)

// ShouldApprove determines if a command needs user approval.
//
// Returns:
//   - needsApproval: true if the command requires approval
//   - reason: explanation of why approval is needed
func (a *Agent) ShouldApprove(cmd *Command) (bool, string) {
	// If approval is disabled, never require approval
	if !a.config.RequireApproval {
		return false, ""
	}

	// Classify the command
	result, err := a.validator.Classify(cmd)
	if err != nil {
		// On error, require approval for safety
		return true, fmt.Sprintf("Classification error: %v", err)
	}

	switch result.Classification {
	case CommandSafe:
		return false, ""

	case CommandInteractive:
		return true, "This command may modify files or system state"

	case CommandDangerous:
		return true, fmt.Sprintf("WARNING: Dangerous operation - %s", result.Reason)

	case CommandForbidden:
		// Forbidden commands should never be executed, even with approval
		// This will be handled by the executor
		return false, fmt.Sprintf("BLOCKED: %s", result.Reason)

	case CommandUnverified:
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

		// Convert llm.ToolCall to core.ToolCall
		coreToolCall := &ToolCall{
			ID:   toolCall.ID,
			Type: toolCall.Type,
			Function: ToolCallFunction{
				Name:      toolCall.Function.Name,
				Arguments: toolCall.Function.Arguments,
			},
		}

		// Add to assistant message (note: message already appended above)
		messages[len(messages)-1].ToolCalls = append(messages[len(messages)-1].ToolCalls, *coreToolCall)

		// Process the tool call (ProcessToolCall will emit EventToolCallStart)
		toolResult, err := a.ProcessToolCall(ctx, coreToolCall)
		if err != nil {
			// Emit tool error event
			a.emitter.Emit(Event{
				Type:      EventToolCallComplete,
				Timestamp: time.Now(),
				Data: ToolCallCompleteData{
					ToolID:   coreToolCall.ID,
					ToolName: coreToolCall.Function.Name,
					Success:  false,
					Error:    err.Error(),
				},
			})

			// Add error message to conversation (after assistant message)
			messages = append(messages, Message{
				Role: RoleTool,
				Content: fmt.Sprintf("Tool %s failed: %v",
					coreToolCall.Function.Name, err),
				ToolCallID: coreToolCall.ID,
				Timestamp:  time.Now(),
			})
		} else {
			// Emit tool completion event
			completion := ToolCallCompleteData{
				ToolID:   coreToolCall.ID,
				ToolName: coreToolCall.Function.Name,
				Success:  toolResult.Success,
			}
			if toolResult.Success {
				completion.Output = toolResult.Output
			} else if toolResult.Error != nil {
				completion.Error = toolResult.Error.Error()
			}
			a.emitter.Emit(Event{
				Type:      EventToolCallComplete,
				Timestamp: time.Now(),
				Data:      completion,
			})

			// Add tool result to conversation (after assistant message)
			slog.Debug("Agent tool result", "tool", coreToolCall.Function.Name, "output_len", len(toolResult.Output), "success", toolResult.Success)
			messages = append(messages, Message{
				Role:       RoleTool,
				Content:    toolResult.Output,
				ToolCallID: coreToolCall.ID,
				Timestamp:  time.Now(),
			})

			// Track tool call in response
			resp.ToolCalls = append(resp.ToolCalls, coreToolCall)
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
func (a *Agent) ProcessToolCall(ctx context.Context, call *ToolCall) (*ToolResult, error) {
	// 1. Validate tool call
	if err := a.validateToolCall(call); err != nil {
		return &ToolResult{
			ID:      call.ID,
			Success: false,
			Error:   err,
		}, nil // Return nil error so agent continues
	}

	// 2. Parse arguments
	args, err := a.parseToolArguments(call)
	if err != nil {
		return &ToolResult{
			ID:      call.ID,
			Success: false,
			Error:   err,
		}, nil
	}

	// 3. Emit tool start event
	// Convert args to ToolCallArguments
	toolArgs, _ := types.FromMap(args)

	a.emitter.Emit(Event{
		Type:      EventToolCallStart,
		Timestamp: time.Now(),
		Data: ToolCallStartData{
			ToolID:     call.ID,
			ToolName:   call.Function.Name,
			Parameters: toolArgs,
		},
	})

	// 4. Execute tool
	var result *ToolResult

	// execute_command needs special handling for approval workflow
	if call.Function.Name == "execute_command" {
		result, _ = a.executeCommand(ctx, call.ID, args)
	} else {
		// Use tool registry for other tools
		toolResult, err := a.toolRegistry.Execute(ctx, call.Function.Name, args)
		if err != nil {
			result = &ToolResult{
				ID:      call.ID,
				Success: false,
				Error:   err,
			}
		} else {
			// Convert tools.ToolResult to core.ToolResult
			result = &ToolResult{
				ID:      call.ID,
				Success: toolResult.Success,
				Output:  toolResult.Output,
			}
			if toolResult.Error != "" {
				result.Error = errors.New(toolResult.Error)
			}
		}
	}

	// 5. Emit completion event
	completion := ToolCallCompleteData{
		ToolID:   call.ID,
		ToolName: call.Function.Name,
		Success:  result.Success,
	}
	if result.Success {
		completion.Output = result.Output
	} else if result.Error != nil {
		completion.Error = result.Error.Error()
	}
	a.emitter.Emit(Event{
		Type:      EventToolCallComplete,
		Timestamp: time.Now(),
		Data:      completion,
	})

	return result, nil // Always return nil error so agent continues
}

// validateToolCall validates the tool call structure.
func (a *Agent) validateToolCall(call *ToolCall) error {
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
func (a *Agent) parseToolArguments(call *ToolCall) (map[string]interface{}, error) {
	if call.Function.Arguments == "" {
		return nil, errors.New("tool arguments cannot be empty")
	}

	var args map[string]interface{}
	if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
		return nil, fmt.Errorf("failed to parse tool arguments: %w", err)
	}

	return args, nil
}

// executeCommand executes a shell command with approval workflow.
func (a *Agent) executeCommand(ctx context.Context, id string, args map[string]interface{}) (*ToolResult, error) {
	// Extract command string
	cmdStr, ok := args["command"].(string)
	if !ok {
		return &ToolResult{
			ID:      id,
			Success: false,
			Error:   errors.New("command argument must be a string"),
		}, nil
	}

	// Parse command into parts
	parts := strings.Fields(cmdStr)
	if len(parts) == 0 {
		return &ToolResult{
			ID:      id,
			Success: false,
			Error:   errors.New("command cannot be empty"),
		}, nil
	}

	// Create Command struct
	cmd := &Command{
		Program: parts[0],
		Args:    parts[1:],
		Raw:     cmdStr,
	}

	// Set working directory if provided
	if workDir, ok := args["workdir"].(string); ok {
		cmd.WorkDir = workDir
	} else {
		cmd.WorkDir = a.context.WorkDir
	}

	// Check if command needs approval
	if needsApproval, reason := a.ShouldApprove(cmd); needsApproval {
		// Request approval
		approved := a.requestApproval(ctx, cmd, reason)
		if !approved {
			return &ToolResult{
				ID:      id,
				Success: false,
				Error:   errors.New("command denied by user"),
			}, nil
		}
	}

	// Execute command (use default options)
	result, err := a.executor.Execute(ctx, cmd, nil)
	if err != nil {
		return &ToolResult{
			ID:       id,
			Success:  false,
			Output:   result.Stderr,
			Error:    err,
			ExitCode: result.ExitCode,
		}, nil
	}

	return &ToolResult{
		ID:       id,
		Success:  result.ExitCode == 0,
		Output:   result.Stdout,
		ExitCode: result.ExitCode,
	}, nil
}

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
