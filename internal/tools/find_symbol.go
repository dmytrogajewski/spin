package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/dmytrogajewski/spin/internal/lsp"
)

const (
	findSymbolName = "find_symbol"
	paramName      = "name"
	paramFilePath  = "file_path"
)

// DefinitionFinder locates symbol definitions in the given file at a position.
type DefinitionFinder func(ctx context.Context, filePath string, line, character int) ([]lsp.Location, error)

// FindSymbolTool looks up symbol definitions by name and file context.
type FindSymbolTool struct {
	find DefinitionFinder
}

// NewFindSymbolTool creates a find_symbol tool backed by the given definition finder.
func NewFindSymbolTool(find DefinitionFinder) *FindSymbolTool {
	return &FindSymbolTool{find: find}
}

// Name returns the tool name.
func (t *FindSymbolTool) Name() string {
	return findSymbolName
}

// Description returns a human-readable description.
func (t *FindSymbolTool) Description() string {
	return "Find symbol definitions by name using the language server"
}

// Schema returns the parameter schema.
func (t *FindSymbolTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "function",
		Function: FunctionSchema{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters: ParameterSchema{
				Type: "object",
				Properties: map[string]PropertyDefinition{
					paramName: {
						Type:        "string",
						Description: "Symbol name to search for (exact, prefix with '.', or wildcard with '*')",
					},
					paramFilePath: {
						Type:        "string",
						Description: "File path providing language context for the search",
					},
				},
				Required: []string{paramName, paramFilePath},
			},
		},
	}
}

// Execute runs the symbol lookup.
func (t *FindSymbolTool) Execute(ctx context.Context, params ToolParameters) (ToolResult, error) {
	name, _ := params.GetString(paramName)
	if name == "" {
		return NewToolError(ErrInvalidParameters), nil
	}

	filePath, _ := params.GetString(paramFilePath)
	if filePath == "" {
		return NewToolError(ErrInvalidParameters), nil
	}

	if t.find == nil {
		return NewToolError(lsp.ErrServerNotFound), nil
	}

	locations, findErr := t.find(ctx, filePath, 0, 0)
	if findErr != nil {
		return ErrToResultf("find definition: %s", findErr)
	}

	if len(locations) == 0 {
		return NewToolResult("No symbols found matching: " + name), nil
	}

	return NewToolResult(formatLocations(locations)), nil
}

// formatLocations formats locations into a human-readable string.
func formatLocations(locations []lsp.Location) string {
	var builder strings.Builder

	fmt.Fprintf(&builder, "Found %d location(s):\n", len(locations))

	for _, loc := range locations {
		uri := strings.TrimPrefix(loc.URI, "file://")

		fmt.Fprintf(&builder, "  %s:%d:%d\n",
			uri,
			loc.Range.Start.Line+1,
			loc.Range.Start.Character+1,
		)
	}

	return builder.String()
}
