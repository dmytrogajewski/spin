package tools

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/dmytrogajewski/spin/internal/tools/fuzzy"
	"github.com/dmytrogajewski/spin/internal/undo"
	"github.com/dmytrogajewski/spin/pkg/alg/diff"
	"github.com/dmytrogajewski/spin/pkg/alg/pathx"
)

// Sentinel errors for edit file tool.
var (
	// ErrOldContentRequired is returned when old_content parameter is missing.
	ErrOldContentRequired = errors.New("old_content parameter must be a non-empty string")
	// ErrNoMatchFound is returned when old_content cannot be found in the file.
	ErrNoMatchFound = errors.New("no match found for old_content")
	// ErrAmbiguousMatch is returned when old_content matches multiple regions.
	ErrAmbiguousMatch = errors.New("ambiguous match")
)

// EditFileTool implements fuzzy file editing functionality.
type EditFileTool struct {
	workDir string
	tracker *FileTracker
	chain   *fuzzy.Chain
	opLog   *undo.OperationLog
}

// NewEditFileTool creates a new edit file tool.
func NewEditFileTool(workDir ...string) *EditFileTool {
	var wd string
	if len(workDir) > 0 {
		wd = workDir[0]
	}

	return &EditFileTool{
		workDir: wd,
		chain:   fuzzy.DefaultChain(),
	}
}

// SetTracker sets the file tracker for stale-read detection.
func (t *EditFileTool) SetTracker(tracker *FileTracker) {
	t.tracker = tracker
}

// SetOperationLog sets the operation log for undo support.
func (t *EditFileTool) SetOperationLog(log *undo.OperationLog) {
	t.opLog = log
}

// Name implements the Name operation.
func (t *EditFileTool) Name() string {
	return "edit_file"
}

// Description implements the Description operation.
func (t *EditFileTool) Description() string {
	return "Edit a file by replacing old_content with new_content using fuzzy matching"
}

// Schema implements the Schema operation.
func (t *EditFileTool) Schema() ToolSchema {
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
						Description: "The path to the file to edit",
					},
					"old_content": {
						Type:        "string",
						Description: "The content to find and replace",
					},
					"new_content": {
						Type:        "string",
						Description: "The replacement content",
					},
				},
				Required: []string{"path", "old_content", "new_content"},
			},
		},
	}
}

// Execute implements the Execute operation.
func (t *EditFileTool) Execute(ctx context.Context, params ToolParameters) (ToolResult, error) {
	if err := ctx.Err(); err != nil {
		return NewToolError(err), nil
	}

	path, oldContent, newContent, extractErr := t.extractParams(params)
	if extractErr != nil {
		return NewToolError(extractErr), nil
	}

	if err := t.assertFresh(path); err != nil {
		return NewToolError(err), nil
	}

	fileContent, readErr := t.readFile(path)
	if readErr != nil {
		return ErrToResultf("failed to read file: %v", readErr)
	}

	match, matchErr := t.findUniqueMatch(fileContent, oldContent, path)
	if matchErr != nil {
		return NewToolError(matchErr), nil
	}

	// Record before-state for undo.
	if t.opLog != nil {
		if logErr := t.opLog.RecordFileChange(path); logErr != nil {
			return ErrToResultf("failed to record operation: %v", logErr)
		}
	}

	if writeErr := t.applyEdit(path, fileContent, newContent, match); writeErr != nil {
		return NewToolError(fmt.Errorf("failed to write file: %w", writeErr)), nil
	}

	if err := t.recordRead(path); err != nil {
		return ErrToResultf("edit succeeded but failed to record read: %v", err)
	}

	diffOutput := diff.Generate(path, match.Original, newContent)
	output := fmt.Sprintf("Edit applied via %q pass.\n\n%s", match.PassName, diffOutput)

	return NewToolResult(output), nil
}

// extractParams validates and extracts the required parameters.
func (t *EditFileTool) extractParams(params ToolParameters) (path, oldContent, newContent string, err error) {
	path = params.GetStringOr("path", "")
	if path == "" {
		return "", "", "", errPathParameterRequired
	}

	path = pathx.ResolvePath(t.workDir, path)

	oldContent = params.GetStringOr("old_content", "")
	if oldContent == "" {
		return "", "", "", ErrOldContentRequired
	}

	newContent = params.GetStringOr("new_content", "")

	return path, oldContent, newContent, nil
}

// assertFresh checks file freshness via tracker if available.
func (t *EditFileTool) assertFresh(path string) error {
	if t.tracker == nil {
		return nil
	}

	return t.tracker.AssertFresh(path)
}

// readFile reads the file content.
func (t *EditFileTool) readFile(path string) (string, error) {
	fileBytes, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}

	return string(fileBytes), nil
}

// findUniqueMatch finds exactly one match, returning an error on zero or multiple matches.
func (t *EditFileTool) findUniqueMatch(fileContent, oldContent, path string) (*fuzzy.MatchResult, error) {
	allMatches := t.chain.FindAll(fileContent, oldContent)
	if len(allMatches) == 0 {
		return nil, fmt.Errorf("%w in %s", ErrNoMatchFound, path)
	}

	if len(allMatches) > 1 {
		return nil, fmt.Errorf(
			"%w — %d occurrences found in %s; provide more context to uniquely identify the region",
			ErrAmbiguousMatch, len(allMatches), path,
		)
	}

	return &allMatches[0], nil
}

// applyEdit writes the edited content to the file.
func (t *EditFileTool) applyEdit(path, fileContent, newContent string, match *fuzzy.MatchResult) error {
	result := fileContent[:match.Start] + newContent + fileContent[match.End:]

	if err := os.WriteFile(path, []byte(result), 0o600); err != nil {
		return fmt.Errorf("write edited file: %w", err)
	}

	return nil
}

// recordRead records the file read via tracker if available.
func (t *EditFileTool) recordRead(path string) error {
	if t.tracker == nil {
		return nil
	}

	return t.tracker.RecordRead(path)
}

// CheckApproval assesses whether the edit operation requires approval.
func (t *EditFileTool) CheckApproval(params ToolParameters) ApprovalNeeds {
	path, err := params.GetString("path")
	if err != nil || path == "" {
		return ApprovalNeeds{
			Required: true,
			Risk:     RiskMedium,
			Reason:   fmt.Sprintf("Editing file: %s", path),
		}
	}

	return ApprovalNeeds{
		Required: true,
		Risk:     RiskHigh,
		Reason:   fmt.Sprintf("Editing file: %s", path),
	}
}
