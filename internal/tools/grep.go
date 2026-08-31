package tools

import (
	"context"
	"fmt"

	"github.com/dmytrogajewski/spin/internal/filesearch"
	"github.com/dmytrogajewski/spin/pkg/alg/collections"
	"github.com/dmytrogajewski/spin/pkg/alg/pathx"
)

const grepToolName = "grep"

// GrepTool searches file contents under a workspace root.
type GrepTool struct {
	compactControl

	workDir string
}

// NewGrepTool creates a content-search tool.
func NewGrepTool(workDir ...string) *GrepTool {
	return &GrepTool{workDir: collections.FirstNonZero(workDir...)}
}

// Name implements the Name operation.
func (t *GrepTool) Name() string {
	return grepToolName
}

// Description implements the Description operation.
func (t *GrepTool) Description() string {
	return "Search file contents for a pattern"
}

// Schema implements the Schema operation.
func (t *GrepTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "function",
		Function: FunctionSchema{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters: ParameterSchema{
				Type: "object",
				Properties: map[string]PropertyDefinition{
					"pattern": {
						Type:        "string",
						Description: "The text pattern to search for",
					},
					"path": {
						Type:        "string",
						Description: "Directory to search (optional, defaults to workspace)",
					},
				},
				Required: []string{"pattern"},
			},
		},
	}
}

// Execute implements the Execute operation.
func (t *GrepTool) Execute(ctx context.Context, params ToolParameters) (ToolResult, error) {
	if err := ctx.Err(); err != nil {
		return NewToolError(err), nil
	}

	pattern := params.GetStringOr("pattern", "")
	if pattern == "" {
		return NewToolError(errQueryParameterRequired), nil
	}

	root := params.GetStringOr("path", t.workDir)
	root = pathx.ResolvePath(t.workDir, root)

	raw, err := filesearch.Grep(ctx, root, pattern)
	if err != nil {
		return NewToolError(fmt.Errorf("grep: %w", err)), nil
	}

	if raw == "" {
		return NewToolResult(fmt.Sprintf("No matches for %q", pattern)), nil
	}

	return applyBuiltinCompact(t.compactOn(), "grep", raw), nil
}
