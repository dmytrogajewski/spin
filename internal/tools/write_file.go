package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dmytrogajewski/spin/internal/undo"
)

// WriteFileTool implements file writing functionality.
type WriteFileTool struct {
	workDir string
	tracker *FileTracker
	opLog   *undo.OperationLog
}

// NewWriteFileTool creates a new write file tool.
func NewWriteFileTool(workDir ...string) *WriteFileTool {
	var wd string
	if len(workDir) > 0 {
		wd = workDir[0]
	}

	return &WriteFileTool{workDir: wd}
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
func (t *WriteFileTool) Execute(_ context.Context, params ToolParameters) (ToolResult, error) {
	path, _ := params.GetString("path")
	if path == "" {
		return NewToolError(ErrPathParameterRequired), nil
	}

	path = resolvePath(path, t.workDir)

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

	writeErr := os.WriteFile(path, []byte(content), 0o600)
	if writeErr != nil {
		return NewToolError(fmt.Errorf("failed to write file: %w", writeErr)), nil
	}

	return NewToolResult(fmt.Sprintf("Successfully wrote %d bytes to %s", len(content), path)), nil
}

// SetTracker sets the file tracker for stale-read detection.
func (t *WriteFileTool) SetTracker(tracker *FileTracker) {
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
			Reason:   fmt.Sprintf("Writing file: %s", path),
		}
	}

	// System paths require critical approval.
	systemPaths := []string{"/etc/", "/sys/", "/usr/"}
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
	executableExts := []string{".sh", ".go", ".py", ".rb", ".pl", ".js", ".ts"}
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
