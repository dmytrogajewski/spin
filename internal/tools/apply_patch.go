package tools

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/dmytrogajewski/spin/internal/patchapply"
	"github.com/dmytrogajewski/spin/pkg/alg/diff"
)

var (
	// ErrEmptyPatch is a sentinel error.
	ErrEmptyPatch = errors.New("empty patch")
	// ErrPatchMustBeInStandardDiff is a sentinel error.
	ErrPatchMustBeInStandardDiff = errors.New(
		"patch must be in standard diff format. Expected to start with '*** filename' or '--- filename'",
	)
)

// ApplyPatchTool implements structured patch application functionality.
type ApplyPatchTool struct {
	workspaceRoot string
}

// NewApplyPatchTool creates a new apply patch tool.
func NewApplyPatchTool(workspaceRoot string) *ApplyPatchTool {
	return &ApplyPatchTool{
		workspaceRoot: workspaceRoot,
	}
}

// Name implements the Name operation.
func (t *ApplyPatchTool) Name() string {
	return "apply_patch"
}

// Description implements the Description operation.
func (t *ApplyPatchTool) Description() string {
	return "Apply a patch to modify files in the workspace using standard diff format.\n" +
		"Format: *** filename\n--- filename\n@@ -start,count +start,count @@\n+new line\n-old line\n" +
		"Example:\n" +
		"*** a/file.go\n" +
		"--- b/file.go\n" +
		"@@ -1,3 +1,4 @@\n" +
		" package main\n" +
		" \n" +
		" import \"fmt\"\n" +
		"+// Added comment\n" +
		" func main() {\n"
}

// Schema implements the Schema operation.
func (t *ApplyPatchTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "function",
		Function: FunctionSchema{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters: ParameterSchema{
				Type: "object",
				Properties: map[string]PropertyDefinition{
					"patch_text": {
						Type: "string",
						Description: "The patch text in standard diff format. Must start with " +
							"'*** filename' or '--- filename' and contain " +
							"'@@ -start,count +start,count @@' hunks with '+', '-', or ' ' prefixed lines.",
					},
					"workspace_root": {
						Type:        "string",
						Description: "The workspace root directory (optional, defaults to tool's workspace)",
					},
					"dry_run": {
						Type:        "boolean",
						Description: "If true, validate without applying changes",
					},
					"force": {
						Type:        "boolean",
						Description: "If true, allow overwriting existing files on Add operations",
					},
				},
				Required: []string{"patch_text"},
			},
		},
	}
}

// Execute implements the Execute operation.
func (t *ApplyPatchTool) Execute(ctx context.Context, params ToolParameters) (ToolResult, error) {
	if err := ctx.Err(); err != nil {
		return NewToolError(err), nil
	}
	// Extract patch_text parameter. Accept "patch" as alias since LLMs frequently
	// use the shorter name despite the schema specifying "patch_text".
	patchText := params.GetStringOr("patch_text", "")
	if patchText == "" {
		patchText = params.GetStringOr("patch", "")
	}

	if patchText == "" {
		return ToolResult{
			Success: false,
			Error:   "patch_text parameter must be a non-empty string",
		}, nil
	}

	// Extract workspace_root parameter (optional).
	workspaceRoot := t.workspaceRoot

	customRoot, err := params.GetString("workspace_root")
	if err == nil && customRoot != "" {
		workspaceRoot = customRoot
	}

	// Extract dry_run parameter (optional).
	dryRun := params.GetBoolOr("dry_run", false)

	// Extract force parameter (optional).
	force := params.GetBoolOr("force", false)

	// Detect patch format and parse accordingly.
	patch, err := t.parsePatch(patchText)
	if err != nil {
		return ToolResult{
			Success: false,
			Error:   fmt.Sprintf("parse error: %v", err),
		}, nil
	}

	result, err := t.applyPatch(ctx, workspaceRoot, patch, dryRun, force)
	if err != nil {
		return ToolResult{
			Success: false,
			Error:   fmt.Sprintf("failed to apply patch: %v", err),
		}, nil
	}

	return ToolResult{
		Success: true,
		Output:  formatApplyResult(result, dryRun),
	}, nil
}

// applyPatch creates an applier and applies the patch.
func (t *ApplyPatchTool) applyPatch(
	ctx context.Context, workspaceRoot string, patch *patchapply.Patch, dryRun, force bool,
) (*patchapply.ApplyResult, error) {
	applier, err := patchapply.NewApplier(workspaceRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to create applier: %w", err)
	}

	applier.SetDryRun(dryRun)
	applier.SetForceOverwrite(force)

	return applier.Apply(ctx, patch)
}

// formatApplyResult formats the apply result into a human-readable string.
func formatApplyResult(result *patchapply.ApplyResult, dryRun bool) string {
	var output strings.Builder
	if dryRun {
		output.WriteString("Dry run completed successfully. No files were modified.\n\n")
	} else {
		output.WriteString("Patch applied successfully.\n\n")
	}

	if len(result.FilesCreated) > 0 {
		fmt.Fprintf(&output, "Created %d file(s):\n", len(result.FilesCreated))

		for _, file := range result.FilesCreated {
			fmt.Fprintf(&output, "  + %s\n", file)
		}
	}

	if len(result.FilesDeleted) > 0 {
		fmt.Fprintf(&output, "Deleted %d file(s):\n", len(result.FilesDeleted))

		for _, file := range result.FilesDeleted {
			fmt.Fprintf(&output, "  - %s\n", file)
		}
	}

	if len(result.FilesUpdated) > 0 {
		fmt.Fprintf(&output, "Updated %d file(s):\n", len(result.FilesUpdated))

		for _, file := range result.FilesUpdated {
			fmt.Fprintf(&output, "  ~ %s\n", file)
		}
	}

	if len(result.FilesMoved) > 0 {
		fmt.Fprintf(&output, "Moved %d file(s):\n", len(result.FilesMoved))

		for oldPath, newPath := range result.FilesMoved {
			fmt.Fprintf(&output, "  %s → %s\n", oldPath, newPath)
		}
	}

	return output.String()
}

// parsePatch parses a patch in standard diff format.
func (t *ApplyPatchTool) parsePatch(patchText string) (*patchapply.Patch, error) {
	lines := strings.Split(patchText, "\n")
	if len(lines) == 0 {
		return nil, ErrEmptyPatch
	}

	// Check if it's a proper patchapply format (starts with "*** Begin Patch").
	firstLine := strings.TrimSpace(lines[0])
	if firstLine == "*** Begin Patch" {
		// Use the proper patchapply parser.
		parser := patchapply.NewParser(patchText)

		return parser.Parse()
	}

	// Check if it's a diff format (starts with "*** filename" or "--- filename").
	if !strings.HasPrefix(firstLine, "*** ") && !strings.HasPrefix(firstLine, "--- ") {
		return nil, fmt.Errorf(
			"patch must be in standard diff format. Expected '*** filename' or '--- filename', got: %q: %w",
			firstLine, ErrPatchMustBeInStandardDiff)
	}

	// Parse diff format directly.
	return t.parseDiffFormat(patchText)
}

// parseDiffFormat parses a patch in standard diff format using pkg/alg/diff.
func (t *ApplyPatchTool) parseDiffFormat(diffText string) (*patchapply.Patch, error) {
	filename, hunks, err := diff.Parse(diffText)
	if err != nil {
		return nil, fmt.Errorf("parse diff: %w", err)
	}

	return &patchapply.Patch{
		Operations: []patchapply.FileOperation{
			&patchapply.UpdateFile{
				FilePath: filename,
				Hunks:    convertHunks(hunks),
			},
		},
	}, nil
}

// convertHunks converts diff.Hunk slices to patchapply.Hunk slices.
func convertHunks(hunks []diff.Hunk) []patchapply.Hunk {
	result := make([]patchapply.Hunk, len(hunks))

	for idx, hunk := range hunks {
		result[idx] = patchapply.Hunk{
			Changes: convertChanges(hunk.Changes),
		}
	}

	return result
}

// lineTypeMap maps diff.LineType to patchapply.LineChangeType.
var lineTypeMap = map[diff.LineType]patchapply.LineChangeType{
	diff.LineContext: patchapply.LineContext,
	diff.LineInsert:  patchapply.LineInsert,
	diff.LineDelete:  patchapply.LineDelete,
}

// convertChanges converts diff.LineChange slices to patchapply.LineChange slices.
func convertChanges(changes []diff.LineChange) []patchapply.LineChange {
	result := make([]patchapply.LineChange, len(changes))

	for idx, change := range changes {
		result[idx] = patchapply.LineChange{
			Type: lineTypeMap[change.Type],
			Text: change.Text,
		}
	}

	return result
}

// CheckApproval assesses whether the patch operation requires approval.
func (t *ApplyPatchTool) CheckApproval(params ToolParameters) ApprovalNeeds {
	patchText, err := params.GetString("patch_text")
	if err != nil || patchText == "" {
		return ApprovalNeeds{Required: false, Risk: RiskSafe}
	}

	return ApprovalNeeds{
		Required: true,
		Risk:     RiskHigh,
		Reason:   "Applying patch can modify multiple files",
	}
}
