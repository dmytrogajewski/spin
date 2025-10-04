package tools

import (
	"context"
	"fmt"
	"os"
	"reflect"
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
	// Validate executor dependency
	if t.executor == nil {
		return ToolResult{
			Success: false,
			Error:   "executor not configured",
		}, nil
	}

	// Extract command string
	cmdStr, ok := params["command"].(string)
	if !ok || cmdStr == "" {
		return ToolResult{
			Success: false,
			Error:   "command parameter must be a non-empty string",
		}, nil
	}

	// Parse command into parts
	parts := strings.Fields(cmdStr)
	if len(parts) == 0 {
		return ToolResult{
			Success: false,
			Error:   "command cannot be empty",
		}, nil
	}

	// Use reflection to create a Command struct dynamically
	// This avoids circular import with internal/core
	executorVal := reflect.ValueOf(t.executor)

	// Find the Execute method
	executeMethod := executorVal.MethodByName("Execute")
	if !executeMethod.IsValid() {
		return ToolResult{
			Success: false,
			Error:   "executor does not have Execute method",
		}, nil
	}

	// Get the method signature to find Command type
	methodType := executeMethod.Type()
	if methodType.NumIn() < 3 {
		return ToolResult{
			Success: false,
			Error:   "invalid Execute method signature",
		}, nil
	}

	// Get the command type (second parameter, first is context.Context)
	cmdType := methodType.In(1)

	// Create command value based on the parameter type
	var cmdValue reflect.Value

	if cmdType.Kind() == reflect.Interface {
		// For interface{} parameters (used by mocks), create a dynamic struct
		// Define a struct type with the fields we need
		type dynamicCommand struct {
			Program string
			Args    []string
			WorkDir string
			Raw     string
		}

		cmd := &dynamicCommand{
			Program: parts[0],
			Args:    parts[1:],
			Raw:     cmdStr,
		}

		// Set working directory if provided
		if workDir, ok := params["workdir"].(string); ok && workDir != "" {
			cmd.WorkDir = workDir
		}

		cmdValue = reflect.ValueOf(cmd)
	} else if cmdType.Kind() == reflect.Ptr {
		// For typed parameters (real executor), create instance of the actual type
		cmdStructType := cmdType.Elem()
		cmdValue = reflect.New(cmdStructType)
		cmdElem := cmdValue.Elem()

		// Set Program field
		if programField := cmdElem.FieldByName("Program"); programField.IsValid() && programField.CanSet() {
			programField.SetString(parts[0])
		}

		// Set Args field
		if argsField := cmdElem.FieldByName("Args"); argsField.IsValid() && argsField.CanSet() {
			argsSlice := reflect.MakeSlice(reflect.TypeOf([]string{}), len(parts)-1, len(parts)-1)
			for i := 1; i < len(parts); i++ {
				argsSlice.Index(i - 1).SetString(parts[i])
			}
			argsField.Set(argsSlice)
		}

		// Set Raw field
		if rawField := cmdElem.FieldByName("Raw"); rawField.IsValid() && rawField.CanSet() {
			rawField.SetString(cmdStr)
		}

		// Set WorkDir field if provided
		if workDir, ok := params["workdir"].(string); ok && workDir != "" {
			if workDirField := cmdElem.FieldByName("WorkDir"); workDirField.IsValid() && workDirField.CanSet() {
				workDirField.SetString(workDir)
			}
		}
	} else {
		return ToolResult{
			Success: false,
			Error:   "unexpected Execute command parameter type",
		}, nil
	}

	// Call Execute method
	// Execute(ctx context.Context, cmd *Command, opts *ExecuteOptions) (*Result, error)
	args := []reflect.Value{
		reflect.ValueOf(ctx),
		cmdValue,
		reflect.Zero(methodType.In(2)), // nil for ExecuteOptions
	}

	results := executeMethod.Call(args)
	if len(results) != 2 {
		return ToolResult{
			Success: false,
			Error:   "unexpected Execute return values",
		}, nil
	}

	// Check error (second return value)
	errVal := results[1]
	if !errVal.IsNil() {
		// Get the result (first return value) for error details
		resultVal := results[0]
		var stderr string

		if !resultVal.IsNil() {
			// Unwrap interface{} if needed
			if resultVal.Kind() == reflect.Interface {
				resultVal = resultVal.Elem()
			}
			// Dereference pointer if needed
			var resultElem reflect.Value
			if resultVal.Kind() == reflect.Ptr {
				resultElem = resultVal.Elem()
			} else {
				resultElem = resultVal
			}

			if stderrField := resultElem.FieldByName("Stderr"); stderrField.IsValid() {
				stderr = stderrField.String()
			}
		}

		return ToolResult{
			Success: false,
			Output:  stderr,
			Error:   errVal.Interface().(error).Error(),
		}, nil
	}

	// Extract result fields
	resultVal := results[0]
	if resultVal.Kind() == reflect.Ptr && resultVal.IsNil() {
		return ToolResult{
			Success: false,
			Error:   "nil result from Execute",
		}, nil
	}

	// Get the struct value (dereference if it's a pointer or interface)
	var resultElem reflect.Value
	if resultVal.Kind() == reflect.Interface {
		// Unwrap interface{}
		resultVal = resultVal.Elem()
	}
	if resultVal.Kind() == reflect.Ptr {
		resultElem = resultVal.Elem()
	} else {
		resultElem = resultVal
	}

	var stdout, stderr string
	var exitCode int

	if stdoutField := resultElem.FieldByName("Stdout"); stdoutField.IsValid() {
		stdout = stdoutField.String()
	}
	if stderrField := resultElem.FieldByName("Stderr"); stderrField.IsValid() {
		stderr = stderrField.String()
	}
	if exitCodeField := resultElem.FieldByName("ExitCode"); exitCodeField.IsValid() {
		exitCode = int(exitCodeField.Int())
	}

	// Combine stdout and stderr for output
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
	if t.context == nil {
		return ToolResult{
			Success: false,
			Error:   "context not available",
		}, nil
	}

	// Use reflection to call String() method to avoid circular import
	// The context is core.Environment which implements String() string
	val := reflect.ValueOf(t.context)

	// Check if the context has a String() method
	stringMethod := val.MethodByName("String")
	if !stringMethod.IsValid() {
		return ToolResult{
			Success: false,
			Error:   "context does not implement String() method",
		}, nil
	}

	// Call String() method
	results := stringMethod.Call(nil)
	if len(results) != 1 {
		return ToolResult{
			Success: false,
			Error:   "invalid String() method signature",
		}, nil
	}

	// Extract string result
	output, ok := results[0].Interface().(string)
	if !ok {
		return ToolResult{
			Success: false,
			Error:   "String() method did not return a string",
		}, nil
	}

	return ToolResult{
		Success: true,
		Output:  output,
	}, nil
}
