package tools

import (
	"context"
	"errors"
	"fmt"
	"os"
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

func (t *ReadFileTool) Execute(ctx context.Context, params ToolParameters) (ToolResult, error) {
	path, err := params.GetString("path")
	if err != nil {
		return NewToolError(errors.New("path parameter must be a non-empty string")), nil
	}

	if path == "" {
		return NewToolError(errors.New("path parameter must be a non-empty string")), nil
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return NewToolError(fmt.Errorf("failed to read file: %w", err)), nil
	}

	return NewToolResult(string(content)), nil
}
