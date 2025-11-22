package runtime

import (
	"context"
	"fmt"
	"strings"

	"github.com/dmytrogajewski/spin/internal/tools"
	"github.com/google/shlex"
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

func (t *ACPTerminalTool) Name() string {
	return "shell_command"
}

func (t *ACPTerminalTool) Description() string {
	return "Execute shell commands using ACP terminal protocol"
}

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

func (t *ACPTerminalTool) Execute(ctx context.Context, params tools.ToolParameters) (tools.ToolResult, error) {
	operation, err := params.GetString("operation")
	if err != nil || operation != "execute" {
		return tools.ToolResult{
			Success: false,
			Error:   "operation must be 'execute' for ACP mode",
		}, nil
	}

	cmdStr, err := params.GetString("command")
	if err != nil || cmdStr == "" {
		return tools.ToolResult{
			Success: false,
			Error:   "command parameter is required",
		}, nil
	}

	// Get working directory
	workDir, _ := params.GetString("working_directory")
	if workDir == "" {
		workDir = t.runtime.workDir
	}

	// Check if terminal client is available
	if !t.runtime.SupportsTerminals() || t.runtime.terminalClient == nil {
		return tools.ToolResult{
			Success: false,
			Error:   "ACP terminal protocol not available",
		}, nil
	}

	// Create terminal executor
	terminalExec := NewTerminalExecutor(t.runtime.terminalClient, t.runtime.sessionID, t.runtime.workDir)

	// Parse command
	var cmd tools.CommandInfo
	isShellCommand := strings.Contains(cmdStr, "|") ||
		strings.Contains(cmdStr, ">") ||
		strings.Contains(cmdStr, "<") ||
		strings.Contains(cmdStr, "$") ||
		strings.Contains(cmdStr, "&&") ||
		strings.Contains(cmdStr, "||") ||
		strings.HasPrefix(cmdStr, "cd ") ||
		strings.HasPrefix(cmdStr, "export ") ||
		strings.HasPrefix(cmdStr, "source ")

	if isShellCommand {
		cmd = &simpleCommand{
			program: "/bin/sh",
			args:    []string{"-c", cmdStr},
			raw:     cmdStr,
			workDir: workDir,
		}
	} else {
		parts, err := shlex.Split(cmdStr)
		if err != nil {
			return tools.ToolResult{
				Success: false,
				Error:   fmt.Sprintf("failed to parse command: %v", err),
			}, nil
		}
		if len(parts) == 0 {
			return tools.ToolResult{
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

	// Execute via terminal executor
	result, err := terminalExec.Execute(ctx, cmd, nil)
	if err != nil {
		return tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("execution failed: %v", err),
		}, nil
	}

	// Build output
	output := result.GetStdout()
	if stderr := result.GetStderr(); stderr != "" {
		if output != "" {
			output += "\n"
		}
		output += stderr
	}

	// Get metadata (includes terminal_id)
	metadata := result.GetMetadata()

	return tools.ToolResult{
		Success:  result.GetExitCode() == 0,
		Output:   output,
		Error:    "",
		Metadata: metadata,
	}, nil
}

// simpleCommand implements tools.CommandInfo
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

func (t *ACPReadFileTool) Name() string {
	return "read_file"
}

func (t *ACPReadFileTool) Description() string {
	return "Read the contents of a file using ACP fs/read_text_file protocol"
}

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
						Type:        "string",
						Description: "The absolute path to the file to read",
					},
				},
				Required: []string{"path"},
			},
		},
	}
}

func (t *ACPReadFileTool) Execute(ctx context.Context, params tools.ToolParameters) (tools.ToolResult, error) {
	path, err := params.GetString("path")
	if err != nil || path == "" {
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

	content, err := t.runtime.filesystemClient.ReadTextFile(ctx, path, nil, nil)
	if err != nil {
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

func (t *ACPWriteFileTool) Name() string {
	return "write_file"
}

func (t *ACPWriteFileTool) Description() string {
	return "Write content to a file using ACP fs/write_text_file protocol"
}

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
						Type:        "string",
						Description: "The absolute path to the file to write",
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

func (t *ACPWriteFileTool) Execute(ctx context.Context, params tools.ToolParameters) (tools.ToolResult, error) {
	path, err := params.GetString("path")
	if err != nil || path == "" {
		return tools.ToolResult{
			Success: false,
			Error:   "path parameter must be a non-empty string",
		}, nil
	}

	content, err := params.GetString("content")
	if err != nil {
		return tools.ToolResult{
			Success: false,
			Error:   "content parameter must be a string",
		}, nil
	}

	if t.runtime.filesystemClient == nil {
		return tools.ToolResult{
			Success: false,
			Error:   "ACP filesystem protocol not available",
		}, nil
	}

	err = t.runtime.filesystemClient.WriteTextFile(ctx, path, content)
	if err != nil {
		return tools.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("failed to write file: %v", err),
		}, nil
	}

	return tools.ToolResult{
		Success: true,
		Output:  fmt.Sprintf("Successfully wrote %d bytes to %s", len(content), path),
	}, nil
}
