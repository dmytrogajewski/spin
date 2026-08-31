package tools

import (
	"context"
	"os"

	"github.com/dmytrogajewski/spin/internal/contexteng/compact"
	"github.com/dmytrogajewski/spin/pkg/alg/collections"
	"github.com/dmytrogajewski/spin/pkg/alg/pathx"
)

// ReadFileTool implements file reading functionality.
type ReadFileTool struct {
	compactControl

	workDir   string
	tracker   *pathx.FileTracker
	readLevel string
}

// NewReadFileTool creates a new read file tool.
func NewReadFileTool(workDir ...string) *ReadFileTool {
	return &ReadFileTool{workDir: collections.FirstNonZero(workDir...)}
}

// Name implements the Name operation.
func (t *ReadFileTool) Name() string {
	return "read_file"
}

// Description implements the Description operation.
func (t *ReadFileTool) Description() string {
	return "Read the contents of a file"
}

// Schema implements the Schema operation.
func (t *ReadFileTool) Schema() ToolSchema {
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
						Description: "The path to the file to read",
					},
					"level": {
						Type:        "string",
						Description: "Compact read level: none, minimal (default), aggressive",
						Enum:        []string{compact.LevelNone, compact.LevelMinimal, compact.LevelAggressive},
					},
				},
				Required: []string{"path"},
			},
		},
	}
}

// Execute implements the Execute operation.
func (t *ReadFileTool) Execute(ctx context.Context, params ToolParameters) (ToolResult, error) {
	if err := ctx.Err(); err != nil {
		return NewToolError(err), nil
	}

	path := params.GetStringOr("path", "")
	if path == "" {
		return NewToolError(errPathParameterRequired), nil
	}

	path = pathx.ResolvePath(t.workDir, path)

	content, readErr := os.ReadFile(path)
	if readErr != nil {
		return ErrToResultf("failed to read file: %v", readErr)
	}

	if t.tracker != nil {
		if recordErr := t.tracker.RecordRead(path); recordErr != nil {
			return ErrToResultf("failed to record file read: %v", recordErr)
		}
	}

	level := params.GetStringOr("level", t.readLevel)
	if level == "" {
		level = compact.LevelMinimal
	}

	return applyBuiltinCompact(t.compactOn(), "read -l "+level, string(content)), nil
}

// SetTracker sets the file tracker for stale-read detection.
func (t *ReadFileTool) SetTracker(tracker *pathx.FileTracker) {
	t.tracker = tracker
}

// SetReadLevel sets the default R8 level when the tool arg is omitted.
func (t *ReadFileTool) SetReadLevel(level string) {
	t.readLevel = level
}
