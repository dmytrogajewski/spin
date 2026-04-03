package tools

import (
	"context"
	"fmt"
	"go/token"
	"strings"

	"github.com/dmytrogajewski/spin/internal/lsp"
)

const (
	renameSymbolName = "rename_symbol"
	paramNewName     = "new_name"
)

// SymbolRenamer renames a symbol at a given position across the workspace.
type SymbolRenamer func(ctx context.Context, filePath string, line, character int, newName string) (*lsp.WorkspaceEdit, error)

// RenameSymbolTool performs workspace-wide semantic rename via LSP.
// Implements both [Tool] and [ToolWithApproval].
type RenameSymbolTool struct {
	rename SymbolRenamer
}

// NewRenameSymbolTool creates a rename_symbol tool backed by the given renamer.
func NewRenameSymbolTool(rename SymbolRenamer) *RenameSymbolTool {
	return &RenameSymbolTool{rename: rename}
}

// Name returns the tool name.
func (t *RenameSymbolTool) Name() string {
	return renameSymbolName
}

// Description returns a human-readable description.
func (t *RenameSymbolTool) Description() string {
	return "Rename a symbol across the entire workspace using the language server"
}

// Schema returns the parameter schema.
func (t *RenameSymbolTool) Schema() ToolSchema {
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
						Description: "File path containing the symbol to rename",
					},
					paramLine: {
						Type:        "integer",
						Description: "Zero-based line number of the symbol",
					},
					paramCharacter: {
						Type:        "integer",
						Description: "Zero-based character offset of the symbol",
					},
					paramNewName: {
						Type:        "string",
						Description: "New name for the symbol (must be a valid identifier)",
					},
				},
				Required: []string{paramFilePath, paramLine, paramCharacter, paramNewName},
			},
		},
	}
}

// Execute performs the rename operation.
func (t *RenameSymbolTool) Execute(ctx context.Context, params ToolParameters) (ToolResult, error) {
	filePath := params.GetStringOr(paramFilePath, "")
	if filePath == "" {
		return NewToolError(ErrInvalidParameters), nil
	}

	if !params.Has(paramLine) || !params.Has(paramCharacter) {
		return NewToolError(ErrInvalidParameters), nil
	}

	line := params.GetIntOr(paramLine, 0)
	char := params.GetIntOr(paramCharacter, 0)

	newName := params.GetStringOr(paramNewName, "")
	if newName == "" {
		return NewToolError(ErrInvalidParameters), nil
	}

	if !isValidIdentifier(newName) {
		return NewToolError(fmt.Errorf("%w: %q is not a valid identifier", ErrInvalidParameters, newName)), nil
	}

	if t.rename == nil {
		return NewToolError(lsp.ErrServerNotFound), nil
	}

	edit, renameErr := t.rename(ctx, filePath, line, char, newName)
	if renameErr != nil {
		return ErrToResultf("rename: %s", renameErr)
	}

	if edit == nil || len(edit.Changes) == 0 {
		return NewToolResult("No changes produced by rename"), nil
	}

	return NewToolResult(formatWorkspaceEdit(edit)), nil
}

// CheckApproval assesses whether the rename operation requires approval.
func (t *RenameSymbolTool) CheckApproval(params ToolParameters) ApprovalNeeds {
	newName := params.GetStringOr(paramNewName, "")
	filePath := params.GetStringOr(paramFilePath, "")

	return ApprovalNeeds{
		Required: true,
		Risk:     RiskHigh,
		Reason:   fmt.Sprintf("rename symbol in %s to %q (affects multiple files)", filePath, newName),
	}
}

// isValidIdentifier checks if the name is a valid Go identifier.
// Uses go/token for correct Unicode support.
func isValidIdentifier(name string) bool {
	return token.IsIdentifier(name)
}

// formatWorkspaceEdit formats a workspace edit into a human-readable summary.
func formatWorkspaceEdit(edit *lsp.WorkspaceEdit) string {
	var builder strings.Builder

	totalEdits := 0

	for _, edits := range edit.Changes {
		totalEdits += len(edits)
	}

	fmt.Fprintf(&builder, "Renamed across %d file(s), %d edit(s):\n", len(edit.Changes), totalEdits)

	for uri, edits := range edit.Changes {
		filePath := lsp.URIToPath(uri)

		fmt.Fprintf(&builder, "\n%s: %d edit(s)\n", filePath, len(edits))

		for _, ed := range edits {
			fmt.Fprintf(&builder, "  line %d:%d → %q\n",
				ed.Range.Start.Line+1,
				ed.Range.Start.Character+1,
				ed.NewText,
			)
		}
	}

	return builder.String()
}
