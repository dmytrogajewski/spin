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

// SymbolSearcher searches for symbols by name in the given file.
type SymbolSearcher func(ctx context.Context, filePath string, pattern string) ([]lsp.Symbol, error)

// FindSymbolTool looks up symbol definitions by name and file context.
type FindSymbolTool struct {
	find   DefinitionFinder
	search SymbolSearcher
}

// NewFindSymbolTool creates a find_symbol tool backed by the given definition finder.
func NewFindSymbolTool(find DefinitionFinder) *FindSymbolTool {
	return &FindSymbolTool{find: find}
}

// NewFindSymbolToolWithSearch creates a find_symbol tool that uses symbol search.
func NewFindSymbolToolWithSearch(find DefinitionFinder, search SymbolSearcher) *FindSymbolTool {
	return &FindSymbolTool{find: find, search: search}
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

	// Prefer symbol search by name when available (uses textDocument/documentSymbol).
	if t.search != nil {
		return t.executeSearch(ctx, filePath, name)
	}

	// Fallback to definition finder at position (0,0) — less accurate.
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

// executeSearch uses the SymbolSearcher to find symbols by name.
func (t *FindSymbolTool) executeSearch(ctx context.Context, filePath, name string) (ToolResult, error) {
	symbols, err := t.search(ctx, filePath, name)
	if err != nil {
		return ErrToResultf("search symbols: %s", err)
	}

	if len(symbols) == 0 {
		return NewToolResult("No symbols found matching: " + name), nil
	}

	return NewToolResult(formatSymbols(symbols)), nil
}

// formatSymbols formats symbol search results.
func formatSymbols(symbols []lsp.Symbol) string {
	var builder strings.Builder

	fmt.Fprintf(&builder, "Found %d symbol(s):\n", len(symbols))

	for _, sym := range symbols {
		fmt.Fprintf(&builder, "  %s %s (line %d)\n",
			sym.Kind, sym.Name, sym.Location.Range.Start.Line+1)
	}

	return builder.String()
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
