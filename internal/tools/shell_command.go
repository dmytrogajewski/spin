package tools

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/google/shlex"
)

const (
	classificationLow    = 2
	classificationMedium = 3
	classificationHigh   = 4
	shellCommandName     = "shell_command"
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
	Execute(ctx context.Context, cmd CommandInfo, opts any) (ExecutionResult, error)
}

// ExecutionResult represents command execution result.
type ExecutionResult interface {
	GetStdout() string
	GetStderr() string
	GetExitCode() int
	GetMetadata() map[string]any
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

// GetProgram implements the GetProgram operation.
func (c *simpleCommand) GetProgram() string { return c.program }

// GetArgs implements the GetArgs operation.
func (c *simpleCommand) GetArgs() []string { return c.args }

// GetRaw implements the GetRaw operation.
func (c *simpleCommand) GetRaw() string { return c.raw }

// GetWorkDir implements the GetWorkDir operation.
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
	return shellCommandName
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
						Type: "string",
						Description: "Operation type: 'execute' to run a shell command (most common), " +
							"'get_environment' to list env vars, 'get_shell_info' for shell info, " +
							"'detect_shell' to check if command needs shell, 'validate' to validate a command",
						Enum: []string{"execute", "get_environment", "get_shell_info", "detect_shell", "validate"},
					},
					"command": {
						Type: "string",
						Description: "The shell command to execute (e.g., 'ls -la', 'uname -a', 'df -h'). " +
							"REQUIRED for 'execute', 'detect_shell', and 'validate' operations. " +
							"This is the actual command string you want to run.",
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
	operation, _ := params.GetString("operation")
	if operation == "" {
		return NewToolError(errOperationParameterRequired), nil
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
		return NewToolError(fmt.Errorf("unknown operation: %s: %w", operation, errUnknownOperation)), nil
	}
}

// executeCommand executes a shell command through the executor (which handles approval).
func (t *ShellCommandTool) executeCommand(ctx context.Context, params ToolParameters) (ToolResult, error) {
	if t.executor == nil {
		return NewToolError(errExecutorNotConfigured), nil
	}

	cmdStr, _ := params.GetString("command")
	if cmdStr == "" {
		return NewToolError(fmt.Errorf("execute operation: %w", errCommandParameterRequired)), nil
	}

	workDir := t.resolveWorkDir(params)

	cmd, err := t.buildCommand(cmdStr, workDir)
	if err != nil {
		return NewToolError(err), nil
	}

	// Execute through executor (which handles validation and approval).
	result, execErr := t.executor.Execute(ctx, cmd, nil)
	if execErr != nil {
		return t.buildErrorResult(result, execErr), nil
	}

	return t.buildSuccessResult(result), nil
}

// resolveWorkDir resolves the working directory from params, shell context, or os.
func (t *ShellCommandTool) resolveWorkDir(params ToolParameters) string {
	workDir, _ := params.GetString("working_directory")
	if workDir != "" {
		return workDir
	}

	if t.shellCtx != nil {
		return t.shellCtx.GetWorkingDirectory()
	}

	workDir, _ = os.Getwd()

	return workDir
}

// isShellCmd detects whether a command string requires shell execution.
func (t *ShellCommandTool) isShellCmd(cmdStr string) bool {
	if t.shellCtx != nil {
		return t.shellCtx.IsShellCommand(cmdStr)
	}

	return strings.Contains(cmdStr, "|") ||
		strings.Contains(cmdStr, ">") ||
		strings.Contains(cmdStr, "<") ||
		strings.Contains(cmdStr, "$") ||
		strings.Contains(cmdStr, "&&") ||
		strings.Contains(cmdStr, "||") ||
		strings.HasPrefix(cmdStr, "cd ") ||
		strings.HasPrefix(cmdStr, "export ") ||
		strings.HasPrefix(cmdStr, "source ")
}

// buildCommand builds a simpleCommand from the command string and working directory.
func (t *ShellCommandTool) buildCommand(cmdStr, workDir string) (*simpleCommand, error) {
	if t.isShellCmd(cmdStr) {
		return &simpleCommand{
			program: "/bin/sh",
			args:    []string{"-c", cmdStr},
			raw:     cmdStr,
			workDir: workDir,
		}, nil
	}

	parts, err := shlex.Split(cmdStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse command: %w", err)
	}

	if len(parts) == 0 {
		return nil, errCommandCannotBeEmpty
	}

	return &simpleCommand{
		program: parts[0],
		args:    parts[1:],
		raw:     cmdStr,
		workDir: workDir,
	}, nil
}

// combineOutput merges stdout and stderr into a single string.
func combineOutput(stdout, stderr string) string {
	if stderr == "" {
		return stdout
	}

	if stdout != "" {
		return stdout + "\n" + stderr
	}

	return stderr
}

// buildErrorResult constructs a ToolResult from a failed execution.
func (t *ShellCommandTool) buildErrorResult(result ExecutionResult, execErr error) ToolResult {
	stdout, stderr := "", ""
	if result != nil {
		stdout = result.GetStdout()
		stderr = result.GetStderr()
	}

	return ToolResult{
		Success: false,
		Output:  TruncateOutput(combineOutput(stdout, stderr)),
		Error:   execErr.Error(),
	}
}

// buildSuccessResult constructs a ToolResult from a successful execution.
func (t *ShellCommandTool) buildSuccessResult(result ExecutionResult) ToolResult {
	if result == nil {
		return NewToolResult("")
	}

	exitCode := result.GetExitCode()
	output := combineOutput(result.GetStdout(), result.GetStderr())

	var errorMsg string
	if exitCode != 0 {
		errorMsg = fmt.Sprintf("command failed with exit code %d", exitCode)
	}

	return ToolResult{
		Success:  exitCode == 0,
		Output:   TruncateOutput(output),
		Error:    errorMsg,
		Metadata: result.GetMetadata(),
	}
}

// getEnvironment returns environment variables.
func (t *ShellCommandTool) getEnvironment() (ToolResult, error) {
	// Try to use shell context if available.
	if t.shellCtx != nil {
		envMap := t.shellCtx.GetEnvironmentVars()

		var output strings.Builder
		output.WriteString("Environment Variables:\n")

		for key, value := range envMap {
			fmt.Fprintf(&output, "%s=%s\n", key, value)
		}

		return NewToolResult(output.String()), nil
	}

	// Fallback to os.Environ().
	var output strings.Builder
	output.WriteString("Environment Variables:\n")

	for _, env := range os.Environ() {
		output.WriteString(env + "\n")
	}

	return NewToolResult(output.String()), nil
}

// getShellInfo returns shell information.
func (t *ShellCommandTool) getShellInfo() (ToolResult, error) {
	if t.shellCtx != nil {
		return t.getShellInfoFromContext(), nil
	}

	return t.getShellInfoFallback(), nil
}

// getShellInfoFromContext builds shell info from the shell context.
func (t *ShellCommandTool) getShellInfoFromContext() ToolResult {
	contextInfo := t.shellCtx.GetContextInfo()

	var output strings.Builder
	output.WriteString("Shell Information:\n")
	fmt.Fprintf(&output, "shell_enabled: %t\n", contextInfo.IsShellEnabled())

	if contextInfo.IsShellEnabled() {
		writeShellDetails(&output, contextInfo)
	}

	return NewToolResult(output.String())
}

// writeShellDetails writes shell path and env details to the output.
func writeShellDetails(output *strings.Builder, info ShellContextInfo) {
	fmt.Fprintf(output, "shell: %s\n", info.GetShell())

	if shellPath := info.GetShellPath(); shellPath != "" {
		fmt.Fprintf(output, "shell_path: %s\n", shellPath)
	}

	if shellEnv := info.GetShellEnv(); len(shellEnv) > 0 {
		output.WriteString("shell_env:\n")

		for k, v := range shellEnv {
			fmt.Fprintf(output, "  %s=%s\n", k, v)
		}
	}
}

// getShellInfoFallback builds shell info from environment variables.
func (t *ShellCommandTool) getShellInfoFallback() ToolResult {
	shell := os.Getenv("SHELL")

	var output strings.Builder
	output.WriteString("Shell Information:\n")

	if shell != "" {
		output.WriteString("shell_enabled: true\n")
		fmt.Fprintf(&output, "shell: %s\n", shell)
	} else {
		output.WriteString("shell_enabled: false\n")
	}

	return NewToolResult(output.String())
}

// detectShell checks if a command requires shell execution.
func (t *ShellCommandTool) detectShell(params ToolParameters) (ToolResult, error) {
	command, _ := params.GetString("command")
	if command == "" {
		return NewToolError(fmt.Errorf("detect_shell operation: %w", errCommandParameterRequired)), nil
	}

	// Try to use shell context if available.
	if t.shellCtx != nil {
		isShell := t.shellCtx.IsShellCommand(command)

		return NewToolResult(fmt.Sprintf("Is shell command: %t", isShell)), nil
	}

	// Fallback to simple heuristics.
	isShell := t.isShellCmd(command)

	return NewToolResult(fmt.Sprintf("Is shell command: %t", isShell)), nil
}

// validateCommand validates a command and returns its classification.
func (t *ShellCommandTool) validateCommand(params ToolParameters) (ToolResult, error) {
	command, _ := params.GetString("command")
	if command == "" {
		return NewToolError(fmt.Errorf("validate operation: %w", errCommandParameterRequired)), nil
	}

	if t.validator == nil {
		return NewToolError(errValidatorNotConfigured), nil
	}

	parts := strings.Fields(command)
	if len(parts) == 0 {
		return NewToolError(errCommandCannotBeEmpty), nil
	}

	workDir := t.resolveWorkDir(params)

	cmd := &simpleCommand{
		program: parts[0],
		args:    parts[1:],
		raw:     command,
		workDir: workDir,
	}

	result, classifyErr := t.validator.Classify(cmd)
	if classifyErr != nil {
		return ErrToResultf("classification failed: %v", classifyErr)
	}

	return t.formatClassification(result), nil
}

// classificationToString converts a numeric classification to its string name.
func classificationToString(classification int) string {
	switch classification {
	case 0:
		return "safe"
	case 1:
		return "interactive"
	case classificationLow:
		return "dangerous"
	case classificationMedium:
		return "forbidden"
	case classificationHigh:
		return "unverified"
	default:
		return unknownStatus
	}
}

// formatClassification formats a validation result into a ToolResult.
func (t *ShellCommandTool) formatClassification(result ValidationResult) ToolResult {
	classification := result.GetClassification()
	reason := result.GetReason()
	classStr := classificationToString(classification)
	needsApproval := classification >= 1 && classification <= classificationHigh

	var output strings.Builder
	output.WriteString("Command Validation Result:\n")
	fmt.Fprintf(&output, "Classification: %s\n", classStr)
	fmt.Fprintf(&output, "Needs Approval: %t\n", needsApproval)

	if reason != "" {
		fmt.Fprintf(&output, "Reason: %s\n", reason)
	}

	return NewToolResult(output.String())
}
