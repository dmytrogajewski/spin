package tools

import (
	"context"
	"fmt"
)

const (
	getProcessOutputName        = "get_process_output"
	getProcessOutputDescription = "Get the output of a background task"
	defaultMaxOutputLines       = 100
)

// GetProcessOutputTool retrieves output from a background task.
type GetProcessOutputTool struct {
	manager TaskManager
}

// NewGetProcessOutputTool creates a new get_process_output tool.
func NewGetProcessOutputTool(manager TaskManager) *GetProcessOutputTool {
	return &GetProcessOutputTool{manager: manager}
}

// Name returns the tool name.
func (t *GetProcessOutputTool) Name() string {
	return getProcessOutputName
}

// Description returns the tool description.
func (t *GetProcessOutputTool) Description() string {
	return getProcessOutputDescription
}

// Schema returns the tool schema.
func (t *GetProcessOutputTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "function",
		Function: FunctionSchema{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters: ParameterSchema{
				Type: "object",
				Properties: map[string]PropertyDefinition{
					"task_id": {
						Type:        "string",
						Description: "The ID of the background task",
					},
					"max_lines": {
						Type:        "integer",
						Description: "Maximum number of output lines to return (default: 100)",
					},
				},
				Required: []string{"task_id"},
			},
		},
	}
}

// Execute retrieves the output of a background task.
func (t *GetProcessOutputTool) Execute(ctx context.Context, params ToolParameters) (ToolResult, error) {
	if t.manager == nil {
		return NewToolResult(errTaskManagerNotAvailable), nil
	}

	taskID := params.GetStringOr("task_id", "")
	if taskID == "" {
		return NewToolError(errTaskIDParameterRequired), nil
	}

	maxLines := params.GetIntOr("max_lines", defaultMaxOutputLines)

	output, err := t.manager.GetOutput(ctx, taskID, maxLines)
	if err != nil {
		return NewToolError(fmt.Errorf("failed to get output: %w", err)), nil
	}

	if output == "" {
		return NewToolResult("No output available."), nil
	}

	return NewToolResult(output), nil
}
