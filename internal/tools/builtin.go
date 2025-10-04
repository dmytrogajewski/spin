package tools

import (
	"context"
	"fmt"
	"os"
	"strings"
)

// ReadFileTool implements file reading functionality.
type ReadFileTool struct{}

// NewReadFileTool creates a new read file tool.
func NewReadFileTool() *ReadFileTool {
	return &ReadFileTool{}
}

func (t *ReadFileTool) Name() string {
	return "read_file"
}

func (t *ReadFileTool) Description() string {
	return "Read the contents of a file"
}

func (t *ReadFileTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "function",
		Function: FunctionSchema{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters: ParameterSchema{
				Type: "object",
				Properties: map[string]PropertyDefinition{
					"path": {
						Type:        "string",
						Description: "The path to the file to read",
					},
				},
				Required: []string{"path"},
			},
		},
	}
}

func (t *ReadFileTool) Execute(ctx context.Context, params map[string]interface{}) (ToolResult, error) {
	path, ok := params["path"].(string)
	if !ok || path == "" {
		return ToolResult{
			Success: false,
			Error:   "path parameter must be a non-empty string",
		}, nil
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return ToolResult{
			Success: false,
			Error:   fmt.Sprintf("failed to read file: %v", err),
		}, nil
	}

	return ToolResult{
		Success: true,
		Output:  string(content),
	}, nil
}

// WriteFileTool implements file writing functionality.
type WriteFileTool struct{}

// NewWriteFileTool creates a new write file tool.
func NewWriteFileTool() *WriteFileTool {
	return &WriteFileTool{}
}

func (t *WriteFileTool) Name() string {
	return "write_file"
}

func (t *WriteFileTool) Description() string {
	return "Write content to a file"
}

func (t *WriteFileTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "function",
		Function: FunctionSchema{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters: ParameterSchema{
				Type: "object",
				Properties: map[string]PropertyDefinition{
					"path": {
						Type:        "string",
						Description: "The path to the file to write",
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

func (t *WriteFileTool) Execute(ctx context.Context, params map[string]interface{}) (ToolResult, error) {
	path, ok := params["path"].(string)
	if !ok || path == "" {
		return ToolResult{
			Success: false,
			Error:   "path parameter must be a non-empty string",
		}, nil
	}

	content, ok := params["content"].(string)
	if !ok {
		return ToolResult{
			Success: false,
			Error:   "content parameter must be a string",
		}, nil
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return ToolResult{
			Success: false,
			Error:   fmt.Sprintf("failed to write file: %v", err),
		}, nil
	}

	return ToolResult{
		Success: true,
		Output:  fmt.Sprintf("Successfully wrote %d bytes to %s", len(content), path),
	}, nil
}

// ListDirectoryTool implements directory listing functionality.
type ListDirectoryTool struct{}

// NewListDirectoryTool creates a new list directory tool.
func NewListDirectoryTool() *ListDirectoryTool {
	return &ListDirectoryTool{}
}

func (t *ListDirectoryTool) Name() string {
	return "list_directory"
}

func (t *ListDirectoryTool) Description() string {
	return "List the contents of a directory"
}

func (t *ListDirectoryTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "function",
		Function: FunctionSchema{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters: ParameterSchema{
				Type: "object",
				Properties: map[string]PropertyDefinition{
					"path": {
						Type:        "string",
						Description: "The path to the directory to list",
					},
				},
				Required: []string{"path"},
			},
		},
	}
}

func (t *ListDirectoryTool) Execute(ctx context.Context, params map[string]interface{}) (ToolResult, error) {
	path, ok := params["path"].(string)
	if !ok || path == "" {
		return ToolResult{
			Success: false,
			Error:   "path parameter must be a non-empty string",
		}, nil
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return ToolResult{
			Success: false,
			Error:   fmt.Sprintf("failed to read directory: %v", err),
		}, nil
	}

	var output strings.Builder
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		typeStr := "file"
		if entry.IsDir() {
			typeStr = "dir"
		}

		fmt.Fprintf(&output, "%s\t%s\t%d bytes\n", entry.Name(), typeStr, info.Size())
	}

	return ToolResult{
		Success: true,
		Output:  output.String(),
	}, nil
}

// ExecuteCommandTool implements command execution functionality.
// It requires an Executor and Validator from the core package.
type ExecuteCommandTool struct {
	executor  interface{} // core.Executor - using interface{} to avoid circular import
	validator interface{} // core.Validator - using interface{} to avoid circular import
}

// NewExecuteCommandTool creates a new execute command tool.
func NewExecuteCommandTool(executor, validator interface{}) *ExecuteCommandTool {
	return &ExecuteCommandTool{
		executor:  executor,
		validator: validator,
	}
}

func (t *ExecuteCommandTool) Name() string {
	return "execute_command"
}

func (t *ExecuteCommandTool) Description() string {
	return "Execute a shell command"
}

func (t *ExecuteCommandTool) Schema() ToolSchema {
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
						Description: "The command to execute",
					},
					"workdir": {
						Type:        "string",
						Description: "The working directory for the command (optional)",
					},
				},
				Required: []string{"command"},
			},
		},
	}
}

func (t *ExecuteCommandTool) Execute(ctx context.Context, params map[string]interface{}) (ToolResult, error) {
	// Note: This is a stub implementation
	// The actual implementation will delegate to core.Executor
	// This is implemented in agent.go as executeCommand()
	return ToolResult{
		Success: false,
		Error:   "execute_command must be called through Agent.ProcessToolCall",
	}, nil
}

// GetContextTool implements environment context retrieval.
type GetContextTool struct {
	context interface{} // core.Context - using interface{} to avoid circular import
}

// NewGetContextTool creates a new get context tool.
func NewGetContextTool(context interface{}) *GetContextTool {
	return &GetContextTool{
		context: context,
	}
}

func (t *GetContextTool) Name() string {
	return "get_context"
}

func (t *GetContextTool) Description() string {
	return "Get environment context information"
}

func (t *GetContextTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "function",
		Function: FunctionSchema{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters: ParameterSchema{
				Type:       "object",
				Properties: map[string]PropertyDefinition{},
				Required:   []string{},
			},
		},
	}
}

func (t *GetContextTool) Execute(ctx context.Context, params map[string]interface{}) (ToolResult, error) {
	// Note: This is a stub implementation
	// The actual implementation would serialize the Context
	if t.context == nil {
		return ToolResult{
			Success: false,
			Error:   "context not available",
		}, nil
	}

	// For now, return a simple message
	return ToolResult{
		Success: true,
		Output:  "Context information available",
	}, nil
}
