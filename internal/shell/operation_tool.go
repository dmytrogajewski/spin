package shell

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/dmytrogajewski/spin/internal/tools"
)

// ShellOperationTool implements Shell operations using ShellIntegration.
type ShellOperationTool struct {
	shellIntegration *ShellIntegration
}

// NewShellOperationTool creates a new shell operation tool.
func NewShellOperationTool(shellIntegration *ShellIntegration) *ShellOperationTool {
	return &ShellOperationTool{
		shellIntegration: shellIntegration,
	}
}

func (t *ShellOperationTool) Name() string {
	return "shell_operation"
}

func (t *ShellOperationTool) Description() string {
	return "Perform Shell operations like command execution, environment management"
}

func (t *ShellOperationTool) Schema() tools.ToolSchema {
	return tools.ToolSchema{
		Type: "function",
		Function: tools.FunctionSchema{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters: tools.ParameterSchema{
				Type: "object",
				Properties: map[string]tools.PropertyDefinition{
					"operation": {
						Type:        "string",
						Description: "Shell operation: execute_command, get_environment, get_shell_info, is_shell_command",
						Enum:        []string{"execute_command", "get_environment", "get_shell_info", "is_shell_command"},
					},
					"command": {
						Type:        "string",
						Description: "Command to execute for execute_command operation (optional)",
					},
					"args": {
						Type:        "array",
						Description: "Command arguments for execute_command operation (optional)",
					},
					"working_directory": {
						Type:        "string",
						Description: "Working directory for execute_command operation (optional)",
					},
					"timeout": {
						Type:        "number",
						Description: "Timeout in seconds for execute_command operation (optional, defaults to 30s)",
					},
				},
				Required: []string{"operation"},
			},
		},
	}
}

func (t *ShellOperationTool) Execute(ctx context.Context, params map[string]interface{}) (tools.ToolResult, error) {
	if t.shellIntegration == nil || !t.shellIntegration.IsEnabled() {
		return tools.ToolResult{
			Success: false,
			Error:   "Shell integration not available",
		}, nil
	}

	operation, ok := params["operation"].(string)
	if !ok {
		return tools.ToolResult{
			Success: false,
			Error:   "operation parameter is required",
		}, nil
	}

	switch operation {
	case "execute_command":
		command, _ := params["command"].(string)
		if command == "" {
			return tools.ToolResult{
				Success: false,
				Error:   "command is required for execute_command operation",
			}, nil
		}

		// Parse timeout parameter (optional, defaults to 30s)
		timeout := 30 * time.Second
		if timeoutParam, exists := params["timeout"]; exists {
			if timeoutFloat, ok := timeoutParam.(float64); ok {
				timeout = time.Duration(timeoutFloat) * time.Second
			}
		}

		// Create a context with the specified timeout
		cmdCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		result, err := t.shellIntegration.ExecuteShellCommand(cmdCtx, command)
		if err != nil {
			return tools.ToolResult{
				Success: false,
				Error:   fmt.Sprintf("Failed to execute command '%s': %v", command, err),
			}, nil
		}

		return tools.ToolResult{
			Success: true,
			Output:  fmt.Sprintf("Command executed successfully: %s", result),
		}, nil

	case "get_environment":
		envVars := t.shellIntegration.GetEnvironmentVars()
		var output strings.Builder
		output.WriteString("Environment Variables:\n")
		for key, value := range envVars {
			output.WriteString(fmt.Sprintf("%s=%s\n", key, value))
		}
		return tools.ToolResult{
			Success: true,
			Output:  output.String(),
		}, nil

	case "get_shell_info":
		shellInfo := t.shellIntegration.GetContextInfo()
		var output strings.Builder
		output.WriteString("Shell Information:\n")
		for key, value := range shellInfo {
			output.WriteString(fmt.Sprintf("%s: %v\n", key, value))
		}
		return tools.ToolResult{
			Success: true,
			Output:  output.String(),
		}, nil

	case "is_shell_command":
		command, _ := params["command"].(string)
		if command == "" {
			return tools.ToolResult{
				Success: false,
				Error:   "command is required for is_shell_command operation",
			}, nil
		}

		isShell := t.shellIntegration.IsShellCommand(command)
		return tools.ToolResult{
			Success: true,
			Output:  fmt.Sprintf("Is shell command: %t", isShell),
		}, nil

	default:
		return tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("Unknown operation: %s", operation),
		}, nil
	}
}
