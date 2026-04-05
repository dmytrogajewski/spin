package tools

import (
	"context"
	"strconv"

	"github.com/dmytrogajewski/spin/pkg/alg/stringsx"
)

const (
	listProcessesName        = "list_processes"
	listProcessesDescription = "List all background tasks managed by the agent"
)

// ListProcessesTool lists background tasks and their states.
type ListProcessesTool struct {
	manager TaskManager
}

// NewListProcessesTool creates a new list_processes tool.
func NewListProcessesTool(manager TaskManager) *ListProcessesTool {
	return &ListProcessesTool{manager: manager}
}

// Name returns the tool name.
func (t *ListProcessesTool) Name() string {
	return listProcessesName
}

// Description returns the tool description.
func (t *ListProcessesTool) Description() string {
	return listProcessesDescription
}

// Schema returns the tool schema.
func (t *ListProcessesTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "function",
		Function: FunctionSchema{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters: ParameterSchema{
				Type:       "object",
				Properties: map[string]PropertyDefinition{},
			},
		},
	}
}

// Execute lists all background tasks.
func (t *ListProcessesTool) Execute(ctx context.Context, _ ToolParameters) (ToolResult, error) {
	if t.manager == nil {
		return NewToolResult(errTaskManagerNotAvailable), nil
	}

	tasks := t.manager.List(ctx)
	if len(tasks) == 0 {
		return NewToolResult("No background tasks."), nil
	}

	return NewToolResult(formatTaskTable(tasks)), nil
}

// Column width constants for task table formatting.
const (
	colWidthID       = 8
	colWidthCommand  = 20
	colWidthState    = 9
	colWidthExitCode = 9
)

const maxCommandDisplayLen = 20

// taskTableColumns defines the column layout for task table formatting.
var taskTableColumns = []stringsx.Column{
	{Name: "ID", Width: colWidthID},
	{Name: "Command", Width: colWidthCommand},
	{Name: "State", Width: colWidthState},
	{Name: "Exit Code", Width: colWidthExitCode},
}

// formatTaskTable formats task snapshots as a human-readable table.
func formatTaskTable(tasks []TaskSnapshot) string {
	return stringsx.FormatTable(tasks, taskTableColumns, func(task TaskSnapshot) []string {
		return []string{
			task.ID,
			stringsx.TruncateWithSuffix(task.Command, maxCommandDisplayLen, "..."),
			task.Status.String(),
			strconv.Itoa(task.ExitCode),
		}
	})
}
