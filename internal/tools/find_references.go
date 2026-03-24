package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/dmytrogajewski/spin/internal/lsp"
)

const (
	findReferencesName = "find_references"
	paramLine          = "line"
	paramCharacter     = "character"
)

// ReferenceFinder locates all references to a symbol at a given position.
type ReferenceFinder func(ctx context.Context, filePath string, line, character int) ([]lsp.Location, error)

// FindReferencesTool searches for all references to a symbol at a given position.
type FindReferencesTool struct {
	find ReferenceFinder
}

// NewFindReferencesTool creates a find_references tool backed by the given reference finder.
func NewFindReferencesTool(find ReferenceFinder) *FindReferencesTool {
	return &FindReferencesTool{find: find}
}

// Name returns the tool name.
func (t *FindReferencesTool) Name() string {
	return findReferencesName
}

// Description returns a human-readable description.
func (t *FindReferencesTool) Description() string {
	return "Find all references to the symbol at the given position"
}

// Schema returns the parameter schema.
func (t *FindReferencesTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "function",
		Function: FunctionSchema{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters: ParameterSchema{
				Type: "object",
				Properties: map[string]PropertyDefinition{
					paramFilePath: {
						Type:        "string",
						Description: "File path containing the symbol",
					},
					paramLine: {
						Type:        "integer",
						Description: "Zero-based line number of the symbol",
					},
					paramCharacter: {
						Type:        "integer",
						Description: "Zero-based character offset of the symbol",
					},
				},
				Required: []string{paramFilePath, paramLine, paramCharacter},
			},
		},
	}
}

// Execute runs the references search.
func (t *FindReferencesTool) Execute(ctx context.Context, params ToolParameters) (ToolResult, error) {
	filePath := params.GetStringOr(paramFilePath, "")
	if filePath == "" {
		return NewToolError(ErrInvalidParameters), nil
	}

	if !params.Has(paramLine) || !params.Has(paramCharacter) {
		return NewToolError(ErrInvalidParameters), nil
	}

	line := params.GetIntOr(paramLine, 0)
	char := params.GetIntOr(paramCharacter, 0)

	if t.find == nil {
		return NewToolError(lsp.ErrServerNotFound), nil
	}

	refs, findErr := t.find(ctx, filePath, line, char)
	if findErr != nil {
		return ErrToResultf("find references: %s", findErr)
	}

	if len(refs) == 0 {
		return NewToolResult("No references found"), nil
	}

	return NewToolResult(formatReferencesGrouped(refs)), nil
}

// formatReferencesGrouped groups references by file and formats them.
func formatReferencesGrouped(locations []lsp.Location) string {
	grouped := make(map[string][]lsp.Location)

	for _, loc := range locations {
		grouped[loc.URI] = append(grouped[loc.URI], loc)
	}

	var builder strings.Builder

	fmt.Fprintf(&builder, "Found %d reference(s) across %d file(s):\n", len(locations), len(grouped))

	for uri, locs := range grouped {
		filePath := strings.TrimPrefix(uri, "file://")

		fmt.Fprintf(&builder, "\n%s:\n", filePath)

		for _, loc := range locs {
			fmt.Fprintf(&builder, "  line %d, col %d\n",
				loc.Range.Start.Line+1,
				loc.Range.Start.Character+1,
			)
		}
	}

	return builder.String()
}
