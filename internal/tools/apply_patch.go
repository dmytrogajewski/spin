package tools

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/dmytrogajewski/spin/internal/patchapply"
)

const minPatchLines = 3

var (
	ErrEmptyPatch = errors.New("empty patch")
	ErrPatchMustBeInStandardDiff = errors.New(
		"patch must be in standard diff format. Expected to start with '*** filename' or '--- filename'",
	)
	ErrDiffFormatTooShort = errors.New("diff format too short")
	ErrCouldNotExtractFilenameFromFirst = errors.New("could not extract filename from first line")
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
						Type:        "string",
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
func (t *ApplyPatchTool) Execute(_ context.Context, params ToolParameters) (ToolResult, error) {
	// Extract patch_text parameter.
	patchText, _ := params.GetString("patch_text")
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

	result, err := t.applyPatch(workspaceRoot, patch, dryRun, force)
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
func (t *ApplyPatchTool) applyPatch(workspaceRoot string, patch *patchapply.Patch, dryRun, force bool) (*patchapply.ApplyResult, error) {
	applier, err := patchapply.NewApplier(workspaceRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to create applier: %v", err)
	}

	applier.SetDryRun(dryRun)
	applier.SetForceOverwrite(force)

	return applier.Apply(patch)
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

// parseDiffFormat parses a patch in standard diff format directly.
func (t *ApplyPatchTool) parseDiffFormat(diffText string) (*patchapply.Patch, error) {
	lines := strings.Split(diffText, "\n")
	if len(lines) < minPatchLines {
		return nil, ErrDiffFormatTooShort
	}

	// Extract filename from the first line.
	firstLine := strings.TrimSpace(lines[0])

	var filename string
	if after, ok := strings.CutPrefix(firstLine, "*** "); ok {
		filename = strings.TrimSpace(after)
	} else if afterDash, hasDash := strings.CutPrefix(firstLine, "--- "); hasDash {
		filename = strings.TrimSpace(afterDash)
	} else {
return nil, fmt.Errorf("could not extract filename from first line: %q: %w", firstLine, ErrCouldNotExtractFilenameFromFirst)
	}

	// Create patch with update file operation.
	patch := &patchapply.Patch{
		Operations: []patchapply.FileOperation{
			&patchapply.UpdateFile{
				FilePath: filename,
				Hunks:    []patchapply.Hunk{},
			},
		},
	}

	// Parse hunks.
	updateOp := patch.Operations[0].(*patchapply.UpdateFile)
	var currentHunk *patchapply.Hunk

	for i := 2; i < len(lines); i++ {
		line := lines[i]

		if strings.HasPrefix(line, "@@") {
			if currentHunk != nil {
				updateOp.Hunks = append(updateOp.Hunks, *currentHunk)
			}

			currentHunk = &patchapply.Hunk{
				Header:  strings.TrimSpace(strings.TrimPrefix(line, "@@")),
				Changes: []patchapply.LineChange{},
			}

			continue
		}

		if currentHunk == nil {
			continue
		}

		change, ok := parseDiffLine(line)
		if ok {
			currentHunk.Changes = append(currentHunk.Changes, change)
		}
	}

	// Add the last hunk.
	if currentHunk != nil {
		updateOp.Hunks = append(updateOp.Hunks, *currentHunk)
	}

	return patch, nil
}

// parseDiffLine parses a single diff line into a LineChange.
// Returns the change and true if the line is valid, or false to skip.
func parseDiffLine(line string) (patchapply.LineChange, bool) {
	if line == "" {
		return patchapply.LineChange{Type: patchapply.LineContext, Text: ""}, true
	}

	text := ""
	if len(line) > 1 {
		text = line[1:]
	}

	switch line[0] {
	case ' ':
		return patchapply.LineChange{Type: patchapply.LineContext, Text: text}, true
	case '-':
		return patchapply.LineChange{Type: patchapply.LineDelete, Text: text}, true
	case '+':
		return patchapply.LineChange{Type: patchapply.LineInsert, Text: text}, true
	default:
		return patchapply.LineChange{}, false
	}
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
