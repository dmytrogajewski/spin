package tools

import (
	"context"
	"fmt"
	"os"
	"strings"
)

// CommandValidator validates commands for security (to avoid import cycle with security package).
type CommandValidator interface {
	Classify(cmd CommandInfo) (ValidationResult, error)
}

// CommandInfo represents command information for validation/execution.
type CommandInfo interface {
	GetProgram() string
	GetArgs() []string
	GetRaw() string
	GetWorkDir() string
}

// ValidationResult represents validation result.
type ValidationResult interface {
	GetClassification() int
	GetReason() string
}

// ShellContext provides shell-aware functionality (to avoid import cycle with shell package).
type ShellContext interface {
	GetWorkingDirectory() string
	GetEnvironmentVars() map[string]string
	GetContextInfo() ShellContextInfo
	IsShellCommand(command string) bool
}

// ShellContextInfo represents shell context information.
type ShellContextInfo interface {
	IsShellEnabled() bool
	GetShell() string
	GetShellPath() string
	GetShellEnv() map[string]string
}

// CommandExecutor executes validated commands (to avoid import cycle with agent package).
type CommandExecutor interface {
	Execute(ctx context.Context, cmd CommandInfo, opts interface{}) (ExecutionResult, error)
}

// ExecutionResult represents command execution result.
type ExecutionResult interface {
	GetStdout() string
	GetStderr() string
	GetExitCode() int
}

// ShellCommandTool provides unified shell command execution and introspection.
type ShellCommandTool struct {
	validator CommandValidator
	shellCtx  ShellContext
	executor  CommandExecutor
}

// simpleCommand implements CommandInfo interface.
type simpleCommand struct {
	program string
	args    []string
	raw     string
	workDir string
}

func (c *simpleCommand) GetProgram() string { return c.program }
func (c *simpleCommand) GetArgs() []string  { return c.args }
func (c *simpleCommand) GetRaw() string     { return c.raw }
func (c *simpleCommand) GetWorkDir() string { return c.workDir }

// NewShellCommandTool creates the unified shell command tool.
func NewShellCommandTool(validator CommandValidator, shellCtx ShellContext, executor CommandExecutor) *ShellCommandTool {
	return &ShellCommandTool{
		validator: validator,
		shellCtx:  shellCtx,
		executor:  executor,
	}
}

// Name returns the tool name.
func (t *ShellCommandTool) Name() string {
	return "shell_command"
}

// Description returns the tool description.
func (t *ShellCommandTool) Description() string {
	return "Execute shell commands (with security approval) and inspect shell environment"
}

// Schema returns the tool schema.
func (t *ShellCommandTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "function",
		Function: FunctionSchema{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters: ParameterSchema{
				Type: "object",
				Properties: map[string]PropertyDefinition{
					"operation": {
						Type:        "string",
						Description: "Operation: execute, get_environment, get_shell_info, detect_shell, validate",
						Enum:        []string{"execute", "get_environment", "get_shell_info", "detect_shell", "validate"},
					},
					"command": {
						Type:        "string",
						Description: "Shell command (required for execute, detect_shell, validate)",
					},
					"working_directory": {
						Type:        "string",
						Description: "Working directory (optional, for execute operation)",
					},
					"timeout": {
						Type:        "number",
						Description: "Timeout in seconds (optional, for execute operation, defaults to 30s)",
					},
				},
				Required: []string{"operation"},
			},
		},
	}
}

// Execute handles all shell operations.
func (t *ShellCommandTool) Execute(ctx context.Context, params ToolParameters) (ToolResult, error) {
	operation, err := params.GetString("operation")
	if err != nil {
		return ToolResult{
			Success: false,
			Error:   "operation parameter is required",
		}, nil
	}

	switch operation {
	case "execute":
		return t.executeCommand(ctx, params)
	case "get_environment":
		return t.getEnvironment()
	case "get_shell_info":
		return t.getShellInfo()
	case "detect_shell":
		return t.detectShell(params)
	case "validate":
		return t.validateCommand(params)
	default:
		return ToolResult{
			Success: false,
			Error:   fmt.Sprintf("unknown operation: %s", operation),
		}, nil
	}
}

// executeCommand executes a shell command through the executor (which handles approval).
func (t *ShellCommandTool) executeCommand(ctx context.Context, params ToolParameters) (ToolResult, error) {
	if t.executor == nil {
		return ToolResult{
			Success: false,
			Error:   "executor not configured",
		}, nil
	}

	cmdStr, err := params.GetString("command")
	if err != nil || cmdStr == "" {
		return ToolResult{
			Success: false,
			Error:   "command parameter is required for execute operation",
		}, nil
	}

	parts := strings.Fields(cmdStr)
	if len(parts) == 0 {
		return ToolResult{
			Success: false,
			Error:   "command cannot be empty",
		}, nil
	}

	// Parse working directory
	workDir, _ := params.GetString("working_directory")
	if workDir == "" {
		if t.shellCtx != nil {
			workDir = t.shellCtx.GetWorkingDirectory()
		} else {
			workDir, _ = os.Getwd()
		}
	}

	// Create command
	cmd := &simpleCommand{
		program: parts[0],
		args:    parts[1:],
		raw:     cmdStr,
		workDir: workDir,
	}

	// Execute through executor (which handles validation and approval)
	result, err := t.executor.Execute(ctx, cmd, nil)
	if err != nil {
		stderr := ""
		if result != nil {
			stderr = result.GetStderr()
		}
		return ToolResult{
			Success: false,
			Output:  stderr,
			Error:   err.Error(),
		}, nil
	}

	if result == nil {
		return ToolResult{
			Success: true,
			Output:  "",
		}, nil
	}

	// Get output and exit code
	stdout := result.GetStdout()
	stderr := result.GetStderr()
	exitCode := result.GetExitCode()

	// Combine stdout and stderr
	output := stdout
	if stderr != "" {
		if output != "" {
			output += "\n"
		}
		output += stderr
	}

	return ToolResult{
		Success: exitCode == 0,
		Output:  output,
	}, nil
}

// getEnvironment returns environment variables.
func (t *ShellCommandTool) getEnvironment() (ToolResult, error) {
	// Try to use shell context if available
	if t.shellCtx != nil {
		envMap := t.shellCtx.GetEnvironmentVars()
		var output strings.Builder
		output.WriteString("Environment Variables:\n")
		for key, value := range envMap {
			output.WriteString(fmt.Sprintf("%s=%s\n", key, value))
		}
		return ToolResult{
			Success: true,
			Output:  output.String(),
		}, nil
	}

	// Fallback to os.Environ()
	var output strings.Builder
	output.WriteString("Environment Variables:\n")
	for _, env := range os.Environ() {
		output.WriteString(env + "\n")
	}
	return ToolResult{
		Success: true,
		Output:  output.String(),
	}, nil
}

// getShellInfo returns shell information.
func (t *ShellCommandTool) getShellInfo() (ToolResult, error) {
	// Try to use shell context if available
	if t.shellCtx != nil {
		contextInfo := t.shellCtx.GetContextInfo()

		var output strings.Builder
		output.WriteString("Shell Information:\n")
		output.WriteString(fmt.Sprintf("shell_enabled: %t\n", contextInfo.IsShellEnabled()))

		if contextInfo.IsShellEnabled() {
			output.WriteString(fmt.Sprintf("shell: %s\n", contextInfo.GetShell()))
			if shellPath := contextInfo.GetShellPath(); shellPath != "" {
				output.WriteString(fmt.Sprintf("shell_path: %s\n", shellPath))
			}
			if shellEnv := contextInfo.GetShellEnv(); len(shellEnv) > 0 {
				output.WriteString("shell_env:\n")
				for k, v := range shellEnv {
					output.WriteString(fmt.Sprintf("  %s=%s\n", k, v))
				}
			}
		}

		return ToolResult{
			Success: true,
			Output:  output.String(),
		}, nil
	}

	// Fallback to basic info
	shell := os.Getenv("SHELL")
	var output strings.Builder
	output.WriteString("Shell Information:\n")
	if shell != "" {
		output.WriteString("shell_enabled: true\n")
		output.WriteString(fmt.Sprintf("shell: %s\n", shell))
	} else {
		output.WriteString("shell_enabled: false\n")
	}

	return ToolResult{
		Success: true,
		Output:  output.String(),
	}, nil
}

// detectShell checks if a command requires shell execution.
func (t *ShellCommandTool) detectShell(params ToolParameters) (ToolResult, error) {
	command, err := params.GetString("command")
	if err != nil || command == "" {
		return ToolResult{
			Success: false,
			Error:   "command parameter is required for detect_shell operation",
		}, nil
	}

	// Try to use shell context if available
	if t.shellCtx != nil {
		isShell := t.shellCtx.IsShellCommand(command)
		return ToolResult{
			Success: true,
			Output:  fmt.Sprintf("Is shell command: %t", isShell),
		}, nil
	}

	// Fallback to simple heuristics
	isShell := strings.Contains(command, "|") ||
		strings.Contains(command, ">") ||
		strings.Contains(command, "<") ||
		strings.Contains(command, "$") ||
		strings.HasPrefix(command, "cd ") ||
		strings.HasPrefix(command, "export ") ||
		strings.HasPrefix(command, "source ")

	return ToolResult{
		Success: true,
		Output:  fmt.Sprintf("Is shell command: %t", isShell),
	}, nil
}

// validateCommand validates a command and returns its classification.
func (t *ShellCommandTool) validateCommand(params ToolParameters) (ToolResult, error) {
	command, err := params.GetString("command")
	if err != nil || command == "" {
		return ToolResult{
			Success: false,
			Error:   "command parameter is required for validate operation",
		}, nil
	}

	if t.validator == nil {
		return ToolResult{
			Success: false,
			Error:   "validator not configured",
		}, nil
	}

	// Parse command
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return ToolResult{
			Success: false,
			Error:   "command cannot be empty",
		}, nil
	}

	// Get working directory
	workDir, _ := params.GetString("working_directory")
	if workDir == "" {
		if t.shellCtx != nil {
			workDir = t.shellCtx.GetWorkingDirectory()
		} else {
			workDir, _ = os.Getwd()
		}
	}

	// Create command for validation
	cmd := &simpleCommand{
		program: parts[0],
		args:    parts[1:],
		raw:     command,
		workDir: workDir,
	}

	// Call Classify using validator
	result, err := t.validator.Classify(cmd)
	if err != nil {
		return ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	// Extract classification and reason
	classification := result.GetClassification()
	reason := result.GetReason()

	// Convert classification to string
	classStr := classificationToString(classification)
	needsApproval := classificationNeedsApproval(classification)

	var output strings.Builder
	output.WriteString("Command Validation Result:\n")
	output.WriteString(fmt.Sprintf("Classification: %s\n", classStr))
	output.WriteString(fmt.Sprintf("Needs Approval: %t\n", needsApproval))

	if reason != "" {
		output.WriteString(fmt.Sprintf("Reason: %s\n", reason))
	}

	return ToolResult{
		Success: true,
		Output:  output.String(),
	}, nil
}

// classificationToString converts classification int to string (matches security.CommandClass).
func classificationToString(class int) string {
	switch class {
	case 0:
		return "safe"
	case 1:
		return "interactive"
	case 2:
		return "dangerous"
	case 3:
		return "forbidden"
	case 4:
		return "unverified"
	default:
		return "unknown"
	}
}

// classificationNeedsApproval returns true if classification needs approval.
func classificationNeedsApproval(class int) bool {
	// Interactive, Dangerous, Forbidden, Unverified all need approval
	return class >= 1 && class <= 4
}
