package tools

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dmytrogajewski/spin/internal/undo"
	"github.com/dmytrogajewski/spin/pkg/alg/collections"
	"github.com/dmytrogajewski/spin/pkg/alg/pathx"
	"github.com/dmytrogajewski/spin/pkg/alg/stringsx"
)

// errContentTruncated is returned when content appears truncated.
var errContentTruncated = errors.New("content appears truncated")

// systemPaths lists path prefixes that require critical approval.
var systemPaths = []string{"/etc/", "/sys/", "/usr/"}

// executableExts lists file extensions that require high approval.
var executableExts = []string{".sh", ".go", ".py", ".rb", ".pl", ".js", ".ts"}

// WriteFileTool implements file writing functionality.
type WriteFileTool struct {
	workDir string
	tracker *pathx.FileTracker
	opLog   *undo.OperationLog
}

// NewWriteFileTool creates a new write file tool.
func NewWriteFileTool(workDir ...string) *WriteFileTool {
	return &WriteFileTool{workDir: collections.FirstNonZero(workDir...)}
}

// Name implements the Name operation.
func (t *WriteFileTool) Name() string {
	return "write_file"
}

// Description implements the Description operation.
func (t *WriteFileTool) Description() string {
	return "Write content to a file"
}

// Schema implements the Schema operation.
func (t *WriteFileTool) Schema() ToolSchema {
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
						Description: "The path to the file to write",
					},
					"content": {
						Type:        "string",
						Description: "The content to write to the file",
					},
				},
				Required: []string{"path", "content"},
			},
		},
	}
}

// Execute implements the Execute operation.
func (t *WriteFileTool) Execute(ctx context.Context, params ToolParameters) (ToolResult, error) {
	if err := ctx.Err(); err != nil {
		return NewToolError(err), nil
	}

	path := params.GetStringOr("path", "")
	if path == "" {
		return NewToolError(errPathParameterRequired), nil
	}

	path = pathx.ResolvePath(t.workDir, path)

	content, contentErr := params.GetString("content")
	if contentErr != nil {
		return NewToolError(fmt.Errorf("content parameter required: %w", contentErr)), nil
	}

	if t.tracker != nil {
		if freshErr := t.tracker.AssertFresh(path); freshErr != nil {
			return NewToolError(freshErr), nil
		}
	}

	// Record before-state for undo.
	if t.opLog != nil {
		if logErr := t.opLog.RecordFileChange(path); logErr != nil {
			return ErrToResultf("failed to record operation: %v", logErr)
		}
	}

	// Create parent directories if they don't exist.
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		err := os.MkdirAll(dir, 0o750)
		if err != nil {
			return NewToolError(fmt.Errorf("failed to create parent directories: %w", err)), nil
		}
	}

	if reason := stringsx.DetectTruncation(content); reason != "" {
		return NewToolError(fmt.Errorf("%w (%s) — refusing to write broken file to %s", errContentTruncated, reason, path)), nil
	}

	writeErr := os.WriteFile(path, []byte(content), 0o600)
	if writeErr != nil {
		return NewToolError(fmt.Errorf("failed to write file: %w", writeErr)), nil
	}

	return NewToolResult(fmt.Sprintf("Successfully wrote %d bytes to %s", len(content), path)), nil
}

// SetTracker sets the file tracker for stale-read detection.
func (t *WriteFileTool) SetTracker(tracker *pathx.FileTracker) {
	t.tracker = tracker
}

// SetOperationLog sets the operation log for undo support.
func (t *WriteFileTool) SetOperationLog(log *undo.OperationLog) {
	t.opLog = log
}

// CheckApproval assesses whether the write operation requires approval.
func (t *WriteFileTool) CheckApproval(params ToolParameters) ApprovalNeeds {
	path, err := params.GetString("path")
	if err != nil || path == "" {
		return ApprovalNeeds{
			Required: true,
			Risk:     RiskMedium,
			Reason:   "Writing file: unknown path",
		}
	}

	// System paths require critical approval.
	for _, sysPath := range systemPaths {
		if strings.HasPrefix(path, sysPath) {
			return ApprovalNeeds{
				Required: true,
				Risk:     RiskCritical,
				Reason:   fmt.Sprintf("Writing to system path: %s", path),
			}
		}
	}

	// Executable file extensions require high approval.
	for _, ext := range executableExts {
		if strings.HasSuffix(path, ext) {
			return ApprovalNeeds{
				Required: true,
				Risk:     RiskHigh,
				Reason:   fmt.Sprintf("Writing executable/source code file: %s", path),
			}
		}
	}

	return ApprovalNeeds{
		Required: true,
		Risk:     RiskMedium,
		Reason:   fmt.Sprintf("Writing file: %s", path),
	}
}
