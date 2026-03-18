package tools

import (
	"context"
	"fmt"
	"os"
	"strings"
)

// ListDirectoryTool implements directory listing functionality.
type ListDirectoryTool struct {
	workDir string
}

// NewListDirectoryTool creates a new list directory tool.
func NewListDirectoryTool(workDir ...string) *ListDirectoryTool {
	var wd string
	if len(workDir) > 0 {
		wd = workDir[0]
	}

	return &ListDirectoryTool{workDir: wd}
}

// Name implements the Name operation.
func (t *ListDirectoryTool) Name() string {
	return "list_directory"
}

// Description implements the Description operation.
func (t *ListDirectoryTool) Description() string {
	return "List the contents of a directory"
}

// Schema implements the Schema operation.
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

// Execute implements the Execute operation.
func (t *ListDirectoryTool) Execute(_ context.Context, params ToolParameters) (ToolResult, error) {
	path, _ := params.GetString("path")
	if path == "" {
		return NewToolError(ErrPathParameterRequired), nil
	}

	path = resolvePath(path, t.workDir)

	entries, err := os.ReadDir(path)
	if err != nil {
		return NewToolError(fmt.Errorf("failed to read directory: %w", err)), nil
	}

	var output strings.Builder

	for _, entry := range entries {
		info, infoErr := entry.Info()
		if infoErr != nil {
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
