package tools

import (
	"context"
	"fmt"
)

const (
	startProcessName        = "start_process"
	startProcessDescription = "Start a command as a background process managed by the agent. " +
		"Returns a task ID that can be used with list_processes, get_process_output, and kill_process."
)

// StartProcessTool launches a command in the background.
type StartProcessTool struct {
	manager TaskStarter
}

// NewStartProcessTool creates a new start_process tool.
func NewStartProcessTool(manager TaskStarter) *StartProcessTool {
	return &StartProcessTool{manager: manager}
}

// Name returns the tool name.
func (t *StartProcessTool) Name() string {
	return startProcessName
}

// Description returns the tool description.
func (t *StartProcessTool) Description() string {
	return startProcessDescription
}

// Schema returns the tool schema.
func (t *StartProcessTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "function",
		Function: FunctionSchema{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters: ParameterSchema{
				Type: "object",
				Properties: map[string]PropertyDefinition{
					"command": {
						Type:        "string",
						Description: "The command to run in the background (e.g., 'python3 server.py')",
					},
					"working_directory": {
						Type:        "string",
						Description: "Working directory for command execution (optional, defaults to current)",
					},
				},
				Required: []string{"command"},
			},
		},
	}
}

// Execute starts a command as a background process.
func (t *StartProcessTool) Execute(ctx context.Context, params ToolParameters) (ToolResult, error) {
	if t.manager == nil {
		return NewToolResult("task manager not available"), nil
	}

	command, _ := params.GetString("command")
	if command == "" {
		return NewToolError(errCommandParameterRequired), nil
	}

	workDir, _ := params.GetString("working_directory")

	taskID, initialOutput, err := t.manager.Start(ctx, command, workDir)
	if err != nil {
		return NewToolError(fmt.Errorf("failed to start background process: %w", err)), nil
	}

	result := fmt.Sprintf("Background process started.\nTask ID: %s", taskID)
	if initialOutput != "" {
		result += fmt.Sprintf("\nInitial output:\n%s", initialOutput)
	}

	return NewToolResult(result), nil
}
