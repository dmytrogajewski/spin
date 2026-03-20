package executor

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/google/shlex"

	"github.com/dmytrogajewski/spin/internal/tools"
)

var (
	// ErrSessionWorkingDirectoryNotSet is a sentinel error.
	ErrSessionWorkingDirectoryNotSet = errors.New("session working directory not set")
	// ErrPathIsOutsideTheAllowedWorkspace is a sentinel error.
	ErrPathIsOutsideTheAllowedWorkspace = errors.New(
		"path '' is outside the allowed workspace (). " +
			"Use relative paths or absolute paths within the workspace",
	)
)

// ACPTerminalTool exposes terminal/create as a tool to the LLM.
// When executed, it uses ACP terminal protocol via TerminalExecutor.
type ACPTerminalTool struct {
	runtime *ACPRuntime
}

// NewACPTerminalTool creates a new ACP terminal tool.
func NewACPTerminalTool(runtime *ACPRuntime) *ACPTerminalTool {
	return &ACPTerminalTool{
		runtime: runtime,
	}
}

// Name implements the Name operation.
func (t *ACPTerminalTool) Name() string {
	return "shell_command"
}

// Description implements the Description operation.
func (t *ACPTerminalTool) Description() string {
	return "Execute shell commands using ACP terminal protocol"
}

// Schema implements the Schema operation.
func (t *ACPTerminalTool) Schema() tools.ToolSchema {
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
						Description: "Operation type. REQUIRED. Must be 'execute' for ACP mode.",
						Enum:        []string{"execute"},
					},
					"command": {
						Type:        "string",
						Description: "Shell command string. REQUIRED for execute operation.",
					},
					"working_directory": {
						Type:        "string",
						Description: "Working directory (optional, defaults to session workDir)",
					},
				},
				Required: []string{"operation", "command"},
			},
		},
	}
}

// Execute implements the Execute operation.
func (t *ACPTerminalTool) Execute(ctx context.Context, params tools.ToolParameters) (tools.ToolResult, error) {
	if err := t.validateParams(params); err != "" {
		return tools.ToolResult{Success: false, Error: err}, nil
	}

	cmdStr, _ := params.GetString("command")
	workDir, _ := params.GetString("working_directory")

	if workDir == "" {
		// Check context for session-specific workDir (set by ACP agent per-session).
		workDir = GetWorkDirFromContext(ctx)
	}

	if workDir == "" {
		workDir = t.runtime.workDir
	}

	if !t.runtime.SupportsTerminals() || t.runtime.terminalClient == nil {
		return tools.ToolResult{Success: false, Error: "ACP terminal protocol not available"}, nil
	}

	cmd, parseErr := t.parseCommand(cmdStr, workDir)
	if parseErr != "" {
		return tools.ToolResult{Success: false, Error: parseErr}, nil
	}

	terminalExec := NewTerminalExecutor(t.runtime.terminalClient, t.runtime.sessionID, workDir)

	result, err := terminalExec.Execute(ctx, cmd, nil)
	if err != nil {
		return tools.ToolResult{Success: false, Error: fmt.Sprintf("execution failed: %v", err)}, nil
	}

	return t.buildResult(result), nil
}

// validateParams validates the required parameters for the ACP terminal tool.
func (t *ACPTerminalTool) validateParams(params tools.ToolParameters) string {
	operation, _ := params.GetString("operation")
	if operation != "execute" {
		return "operation must be 'execute' for ACP mode"
	}

	cmdStr, _ := params.GetString("command")
	if cmdStr == "" {
		return "command parameter is required"
	}

	return ""
}

// parseCommand parses a command string into a CommandInfo.
func (t *ACPTerminalTool) parseCommand(cmdStr, workDir string) (cmdInfo tools.CommandInfo, parsedCmd string) {
	if isShellCommand(cmdStr) {
		return &simpleCommand{
			program: "/bin/sh",
			args:    []string{"-c", cmdStr},
			raw:     cmdStr,
			workDir: workDir,
		}, ""
	}

	parts, err := shlex.Split(cmdStr)
	if err != nil {
		return nil, fmt.Sprintf("failed to parse command: %v", err)
	}

	if len(parts) == 0 {
		return nil, "command cannot be empty"
	}

	return &simpleCommand{
		program: parts[0],
		args:    parts[1:],
		raw:     cmdStr,
		workDir: workDir,
	}, ""
}

// isShellCommand checks if a command string requires shell interpretation.
func isShellCommand(cmdStr string) bool {
	shellChars := []string{"|", ">", "<", "$", "&&", "||"}
	for _, c := range shellChars {
		if strings.Contains(cmdStr, c) {
			return true
		}
	}

	shellPrefixes := []string{"cd ", "export ", "source "}
	for _, p := range shellPrefixes {
		if strings.HasPrefix(cmdStr, p) {
			return true
		}
	}

	return false
}

// buildResult converts an ExecutionResult into a ToolResult.
func (t *ACPTerminalTool) buildResult(result tools.ExecutionResult) tools.ToolResult {
	output := result.GetStdout()
	if stderr := result.GetStderr(); stderr != "" {
		if output != "" {
			output += "\n"
		}

		output += stderr
	}

	return tools.ToolResult{
		Success:  result.GetExitCode() == 0,
		Output:   output,
		Error:    "",
		Metadata: result.GetMetadata(),
	}
}

// simpleCommand implements tools.CommandInfo.
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

// ACPReadFileTool exposes fs/read_text_file as a tool to the LLM.
type ACPReadFileTool struct {
	runtime *ACPRuntime
}

// NewACPReadFileTool creates a new ACP read file tool.
func NewACPReadFileTool(runtime *ACPRuntime) *ACPReadFileTool {
	return &ACPReadFileTool{
		runtime: runtime,
	}
}

// Name implements the Name operation.
func (t *ACPReadFileTool) Name() string {
	return "read_file"
}

// Description implements the Description operation.
func (t *ACPReadFileTool) Description() string {
	return "Read the contents of a file. Use paths relative to the session working directory, or absolute paths within the workspace."
}

// Schema implements the Schema operation.
func (t *ACPReadFileTool) Schema() tools.ToolSchema {
	return tools.ToolSchema{
		Type: "function",
		Function: tools.FunctionSchema{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters: tools.ParameterSchema{
				Type: "object",
				Properties: map[string]tools.PropertyDefinition{
					"path": {
						Type: "string",
						Description: "Path to the file. Use relative paths (e.g., 'src/main.py') or absolute paths " +
							"within the session workspace. Paths outside the workspace are rejected.",
					},
				},
				Required: []string{"path"},
			},
		},
	}
}

// Execute implements the Execute operation.
func (t *ACPReadFileTool) Execute(ctx context.Context, params tools.ToolParameters) (tools.ToolResult, error) {
	path, _ := params.GetString("path")
	if path == "" {
		return tools.ToolResult{
			Success: false,
			Error:   "path parameter must be a non-empty string",
		}, nil
	}

	if t.runtime.filesystemClient == nil {
		return tools.ToolResult{
			Success: false,
			Error:   "ACP filesystem protocol not available",
		}, nil
	}

	// Resolve and validate the path.
	resolvedPath, resolveErr := t.resolvePathWithContext(ctx, path)
	if resolveErr != nil {
		return tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("path resolution failed: %v", resolveErr),
		}, nil
	}

	content, err := t.runtime.filesystemClient.ReadTextFile(ctx, resolvedPath, nil, nil)
	if err != nil {
		// Provide helpful error message for path-related errors.
		errMsg := err.Error()

		workDir := GetWorkDirFromContext(ctx)
		if workDir == "" {
			workDir = t.runtime.workDir
		}

		if strings.Contains(errMsg, "invalid path") {
			return tools.ToolResult{
				Success: false,
				Error: fmt.Sprintf(
					"failed to read file: path '%s' is outside the allowed workspace. "+
						"Use a path within the session directory: %s", path, workDir,
				),
			}, nil
		}

		return tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("failed to read file: %v", err),
		}, nil
	}

	return tools.ToolResult{
		Success: true,
		Output:  content,
	}, nil
}

// resolvePath resolves a path relative to the session working directory.
// It handles both relative and absolute paths, ensuring the result is within the workspace.
func (t *ACPReadFileTool) resolvePathWithContext(ctx context.Context, path string) (string, error) {
	// Get workDir from context first (session-specific), fall back to runtime.
	workDir := GetWorkDirFromContext(ctx)
	if workDir == "" {
		workDir = t.runtime.workDir
	}

	if workDir == "" {
		return "", ErrSessionWorkingDirectoryNotSet
	}

	// If path is relative, resolve it against workDir.
	if !filepath.IsAbs(path) {
		path = filepath.Join(workDir, path)
	}

	// Clean the path to resolve any ".." components.
	cleanPath := filepath.Clean(path)

	// Ensure the path is within the workspace.
	if !isPathWithinWorkspace(cleanPath, workDir) {
		return "", fmt.Errorf(
			"path '%s' is outside the allowed workspace (%s). "+
				"Use relative paths or absolute paths within the workspace: %w",
			path, workDir, ErrPathIsOutsideTheAllowedWorkspace,
		)
	}

	return cleanPath, nil
}

// ACPWriteFileTool exposes fs/write_text_file as a tool to the LLM.
type ACPWriteFileTool struct {
	runtime *ACPRuntime
}

// NewACPWriteFileTool creates a new ACP write file tool.
func NewACPWriteFileTool(runtime *ACPRuntime) *ACPWriteFileTool {
	return &ACPWriteFileTool{
		runtime: runtime,
	}
}

// Name implements the Name operation.
func (t *ACPWriteFileTool) Name() string {
	return "write_file"
}

// Description implements the Description operation.
func (t *ACPWriteFileTool) Description() string {
	return "Write content to a file. Use paths relative to the session working directory, or absolute paths within the workspace."
}

// Schema implements the Schema operation.
func (t *ACPWriteFileTool) Schema() tools.ToolSchema {
	return tools.ToolSchema{
		Type: "function",
		Function: tools.FunctionSchema{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters: tools.ParameterSchema{
				Type: "object",
				Properties: map[string]tools.PropertyDefinition{
					"path": {
						Type: "string",
						Description: "Path to the file. Use relative paths (e.g., 'src/main.py') or absolute paths " +
							"within the session workspace. Paths outside the workspace (like /tmp) are rejected.",
					},
					"content": {
						Type:        "string",
						Description: "The content to write to the file",
					},
				},
				Required: []string{"path", "content"},
			},
		},
	}
}

// Execute implements the Execute operation.
func (t *ACPWriteFileTool) Execute(ctx context.Context, params tools.ToolParameters) (tools.ToolResult, error) {
	path, _ := params.GetString("path")
	if path == "" {
		return tools.ToolResult{
			Success: false,
			Error:   "path parameter must be a non-empty string",
		}, nil
	}

	content, contentErr := params.GetString("content")
	if contentErr != nil {
		return tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("content parameter must be a string: %v", contentErr),
		}, nil
	}

	if t.runtime.filesystemClient == nil {
		return tools.ToolResult{
			Success: false,
			Error:   "ACP filesystem protocol not available",
		}, nil
	}

	// Resolve and validate the path.
	resolvedPath, resolveErr := t.resolvePathWithContext(ctx, path)
	if resolveErr != nil {
		return tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("path resolution failed: %v", resolveErr),
		}, nil
	}

	err := t.runtime.filesystemClient.WriteTextFile(ctx, resolvedPath, content)
	if err != nil {
		// Provide helpful error message for path-related errors.
		errMsg := err.Error()

		workDir := GetWorkDirFromContext(ctx)
		if workDir == "" {
			workDir = t.runtime.workDir
		}

		if strings.Contains(errMsg, "invalid path") {
			return tools.ToolResult{
				Success: false,
				Error: fmt.Sprintf(
					"failed to write file: path '%s' is outside the allowed workspace. "+
						"Use a path within the session directory: %s", path, workDir,
				),
			}, nil
		}

		return tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("failed to write file: %v", err),
		}, nil
	}

	return tools.ToolResult{
		Success: true,
		Output:  fmt.Sprintf("Successfully wrote %d bytes to %s", len(content), resolvedPath),
	}, nil
}

// resolvePathWithContext resolves a path relative to the session working directory.
// It handles both relative and absolute paths, ensuring the result is within the workspace.
func (t *ACPWriteFileTool) resolvePathWithContext(ctx context.Context, path string) (string, error) {
	// Get workDir from context first (session-specific), fall back to runtime.
	workDir := GetWorkDirFromContext(ctx)
	if workDir == "" {
		workDir = t.runtime.workDir
	}

	if workDir == "" {
		return "", ErrSessionWorkingDirectoryNotSet
	}

	// If path is relative, resolve it against workDir.
	if !filepath.IsAbs(path) {
		path = filepath.Join(workDir, path)
	}

	// Clean the path to resolve any ".." components.
	cleanPath := filepath.Clean(path)

	// Ensure the path is within the workspace.
	if !isPathWithinWorkspace(cleanPath, workDir) {
		return "", fmt.Errorf(
			"path '%s' is outside the allowed workspace (%s). "+
				"Use relative paths or absolute paths within the workspace: %w",
			path, workDir, ErrPathIsOutsideTheAllowedWorkspace,
		)
	}

	return cleanPath, nil
}

// isPathWithinWorkspace checks if a path is within the workspace directory.
// It handles symlinks and ".." path components.
func isPathWithinWorkspace(path, workDir string) bool {
	// Clean both paths.
	cleanPath := filepath.Clean(path)
	cleanWorkDir := filepath.Clean(workDir)

	// Check if path starts with workDir
	// We need to ensure it's a proper prefix (not just string prefix)
	// e.g., /home/user/workspace should match /home/user/workspace/file
	// but not /home/user/workspace2/file.
	if !strings.HasPrefix(cleanPath, cleanWorkDir) {
		return false
	}

	// Ensure it's a proper directory boundary
	// Path must be exactly workDir or have a separator after workDir.
	if len(cleanPath) > len(cleanWorkDir) {
		// Check that the next character is a path separator.
		return cleanPath[len(cleanWorkDir)] == filepath.Separator
	}

	// Path is exactly workDir.
	return true
}
