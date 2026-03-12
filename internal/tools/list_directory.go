package tools

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
)

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

func (t *ListDirectoryTool) Execute(ctx context.Context, params ToolParameters) (ToolResult, error) {
	path, err := params.GetString("path")
	if err != nil || path == "" {
		return NewToolError(errors.New("path parameter must be a non-empty string")), nil
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return NewToolError(fmt.Errorf("failed to read directory: %w", err)), nil
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

	return NewToolResult(output.String()), nil
}
