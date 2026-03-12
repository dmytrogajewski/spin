package tools

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/dmytrogajewski/spin/internal/patchapply"
)

var (
	ErrEmptyPatch = errors.New("empty patch")
	ErrPatchMustBeInStandardDiff = errors.New("patch must be in standard diff format. Expected to start with '*** filename' or '--- filename'")
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
						Description: "The patch text in standard diff format. Must start with '*** filename' or '--- filename' and contain '@@ -start,count +start,count @@' hunks with '+', '-', or ' ' prefixed lines.",
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

	// Create applier.
	applier, err := patchapply.NewApplier(workspaceRoot)
	if err != nil {
		return ToolResult{
			Success: false,
			Error:   fmt.Sprintf("failed to create applier: %v", err),
		}, nil
	}

	// Configure applier.
	applier.SetDryRun(dryRun)
	applier.SetForceOverwrite(force)

	// Apply the patch.
	result, err := applier.Apply(patch)
	if err != nil {
		// Extract error message.
		errMsg := err.Error()

		return ToolResult{
			Success: false,
			Error:   fmt.Sprintf("failed to apply patch: %v", errMsg),
		}, nil
	}

	// Format output.
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

	return ToolResult{
		Success: true,
		Output:  output.String(),
	}, nil
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
return nil, fmt.Errorf("patch must be in standard diff format. Expected to start with '*** filename' or '--- filename', got: %q: %w", firstLine, ErrPatchMustBeInStandardDiff)
	}

	// Parse diff format directly.
	return t.parseDiffFormat(patchText)
}

// parseDiffFormat parses a patch in standard diff format directly.
func (t *ApplyPatchTool) parseDiffFormat(diffText string) (*patchapply.Patch, error) {
	lines := strings.Split(diffText, "\n")
	if len(lines) < 3 {
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
	var currentHunk *patchapply.Hunk

	for i := 2; i < len(lines); i++ {
		line := lines[i]

		if strings.HasPrefix(line, "@@") {
			// Start of a new hunk.
			if currentHunk != nil {
				updateOp, ok := patch.Operations[0].(*patchapply.UpdateFile)
				if ok {
					updateOp.Hunks = append(updateOp.Hunks, *currentHunk)
				}
			}

			currentHunk = &patchapply.Hunk{
				Header:  strings.TrimSpace(strings.TrimPrefix(line, "@@")),
				Changes: []patchapply.LineChange{},
			}
		} else if currentHunk != nil {
			// Parse line change.
			if len(line) == 0 {
				currentHunk.Changes = append(currentHunk.Changes, patchapply.LineChange{
					Type: patchapply.LineContext,
					Text: "",
				})
			} else {
				prefix := line[0]

				text := ""
				if len(line) > 1 {
					text = line[1:]
				}

				var changeType patchapply.LineChangeType

				switch prefix {
				case ' ':
					changeType = patchapply.LineContext
				case '-':
					changeType = patchapply.LineDelete
				case '+':
					changeType = patchapply.LineInsert
				default:
					// Skip lines without proper prefixes.
					continue
				}

				currentHunk.Changes = append(currentHunk.Changes, patchapply.LineChange{
					Type: changeType,
					Text: text,
				})
			}
		}
	}

	// Add the last hunk.
	if currentHunk != nil {
		if updateOp, ok := patch.Operations[0].(*patchapply.UpdateFile); ok {
			updateOp.Hunks = append(updateOp.Hunks, *currentHunk)
		}
	}

	return patch, nil
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
