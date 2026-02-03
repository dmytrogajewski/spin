package tools

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/google/shlex"
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
	GetMetadata() map[string]interface{}
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
						Description: "Operation type: 'execute' to run a shell command (most common), 'get_environment' to list env vars, 'get_shell_info' for shell info, 'detect_shell' to check if command needs shell, 'validate' to validate a command",
						Enum:        []string{"execute", "get_environment", "get_shell_info", "detect_shell", "validate"},
					},
					"command": {
						Type:        "string",
						Description: "The shell command to execute (e.g., 'ls -la', 'uname -a', 'df -h'). REQUIRED for 'execute', 'detect_shell', and 'validate' operations. This is the actual command string you want to run.",
					},
					"working_directory": {
						Type:        "string",
						Description: "Working directory for command execution (optional)",
					},
					"timeout": {
						Type:        "number",
						Description: "Timeout in seconds (optional, defaults to 30s)",
					},
				},
				Required: []string{"operation", "command"},
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

	// Parse working directory
	workDir, _ := params.GetString("working_directory")
	if workDir == "" {
		if t.shellCtx != nil {
			workDir = t.shellCtx.GetWorkingDirectory()
		} else {
			workDir, _ = os.Getwd()
		}
	}

	// Check if this is a shell command before parsing
	isShellCommand := false
	if t.shellCtx != nil {
		isShellCommand = t.shellCtx.IsShellCommand(cmdStr)
	} else {
		// Fallback detection
		isShellCommand = strings.Contains(cmdStr, "|") ||
			strings.Contains(cmdStr, ">") ||
			strings.Contains(cmdStr, "<") ||
			strings.Contains(cmdStr, "$") ||
			strings.Contains(cmdStr, "&&") ||
			strings.Contains(cmdStr, "||") ||
			strings.HasPrefix(cmdStr, "cd ") ||
			strings.HasPrefix(cmdStr, "export ") ||
			strings.HasPrefix(cmdStr, "source ")
	}

	var cmd *simpleCommand
	if isShellCommand {
		// For shell commands, use raw command string
		// Pass it via shell execution (sh -c "command")
		cmd = &simpleCommand{
			program: "/bin/sh",
			args:    []string{"-c", cmdStr},
			raw:     cmdStr,
			workDir: workDir,
		}
	} else {
		// Parse command with proper quote handling using shlex
		parts, err := shlex.Split(cmdStr)
		if err != nil {
			return ToolResult{
				Success: false,
				Error:   fmt.Sprintf("failed to parse command: %v", err),
			}, nil
		}
		if len(parts) == 0 {
			return ToolResult{
				Success: false,
				Error:   "command cannot be empty",
			}, nil
		}

		cmd = &simpleCommand{
			program: parts[0],
			args:    parts[1:],
			raw:     cmdStr,
			workDir: workDir,
		}
	}

	// Execute through executor (which handles validation and approval)
	result, err := t.executor.Execute(ctx, cmd, nil)
	if err != nil {
		// Include both stdout and stderr when executor returns error
		stdout := ""
		stderr := ""
		if result != nil {
			stdout = result.GetStdout()
			stderr = result.GetStderr()
		}
		// Combine stdout and stderr
		output := stdout
		if stderr != "" {
			if output != "" {
				output += "\n"
			}
			output += stderr
		}
		return ToolResult{
			Success: false,
			Output:  output,
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
	metadata := result.GetMetadata()

	// Combine stdout and stderr
	output := stdout
	if stderr != "" {
		if output != "" {
			output += "\n"
		}
		output += stderr
	}

	// Set error message if command failed
	var errorMsg string
	if exitCode != 0 {
		errorMsg = fmt.Sprintf("command failed with exit code %d", exitCode)
	}

	return ToolResult{
		Success:  exitCode == 0,
		Output:   output,
		Error:    errorMsg,
		Metadata: metadata,
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
		strings.Contains(command, "&&") ||
		strings.Contains(command, "||") ||
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

	// Convert classification to string (matches security.CommandClass.String())
	// CommandClass constants: 0=safe, 1=interactive, 2=dangerous, 3=forbidden, 4=unverified
	var classStr string
	switch classification {
	case 0:
		classStr = "safe"
	case 1:
		classStr = "interactive"
	case 2:
		classStr = "dangerous"
	case 3:
		classStr = "forbidden"
	case 4:
		classStr = "unverified"
	default:
		classStr = "unknown"
	}

	// Check if approval needed (matches security.CommandClass.NeedsApproval())
	// Interactive, Dangerous, Forbidden, Unverified all need approval
	needsApproval := classification >= 1 && classification <= 4

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
