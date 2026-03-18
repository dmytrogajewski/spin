package tools

import (
	"context"
	"fmt"
)

const (
	killProcessName        = "kill_process"
	killProcessDescription = "Kill a running background task"
)

// KillProcessTool terminates a running background task.
type KillProcessTool struct {
	manager TaskManager
}

// NewKillProcessTool creates a new kill_process tool.
func NewKillProcessTool(manager TaskManager) *KillProcessTool {
	return &KillProcessTool{manager: manager}
}

// Name returns the tool name.
func (t *KillProcessTool) Name() string {
	return killProcessName
}

// Description returns the tool description.
func (t *KillProcessTool) Description() string {
	return killProcessDescription
}

// Schema returns the tool schema.
func (t *KillProcessTool) Schema() ToolSchema {
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
						Description: "The ID of the background task to kill",
					},
				},
				Required: []string{"task_id"},
			},
		},
	}
}

// Execute kills a running background task.
func (t *KillProcessTool) Execute(ctx context.Context, params ToolParameters) (ToolResult, error) {
	if t.manager == nil {
		return NewToolResult("task manager not available"), nil
	}

	taskID, _ := params.GetString("task_id")
	if taskID == "" {
		return NewToolError(errTaskIDParameterRequired), nil
	}

	err := t.manager.Kill(ctx, taskID)
	if err != nil {
		return NewToolError(fmt.Errorf("failed to kill task: %w", err)), nil
	}

	return NewToolResult(fmt.Sprintf("Task %s killed successfully.", taskID)), nil
}
