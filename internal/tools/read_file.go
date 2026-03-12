package tools

import (
	"context"
	"os"
)

// ReadFileTool implements file reading functionality.
type ReadFileTool struct{}

// NewReadFileTool creates a new read file tool.
func NewReadFileTool() *ReadFileTool {
	return &ReadFileTool{}
}

// Name implements the Name operation.
func (t *ReadFileTool) Name() string {
	return "read_file"
}

// Description implements the Description operation.
func (t *ReadFileTool) Description() string {
	return "Read the contents of a file"
}

// Schema implements the Schema operation.
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

// Execute implements the Execute operation.
func (t *ReadFileTool) Execute(_ context.Context, params ToolParameters) (ToolResult, error) {
	path, _ := params.GetString("path")
	if path == "" {
		return NewToolError(ErrPathParameterRequired), nil
	}

	content, readErr := os.ReadFile(path)
	if readErr != nil {
		return ErrToResultf("failed to read file: %v", readErr)
	}

	return NewToolResult(string(content)), nil
}
