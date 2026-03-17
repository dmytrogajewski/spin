package tools

import (
	"context"
	"fmt"
	"strings"
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
func (t *ListProcessesTool) Execute(_ context.Context, _ ToolParameters) (ToolResult, error) {
	if t.manager == nil {
		return NewToolResult("task manager not available"), nil
	}

	tasks := t.manager.List()
	if len(tasks) == 0 {
		return NewToolResult("No background tasks."), nil
	}

	return NewToolResult(formatTaskTable(tasks)), nil
}

// formatTaskTable formats task snapshots as a human-readable table.
func formatTaskTable(tasks []TaskSnapshot) string {
	var buf strings.Builder

	buf.WriteString("ID       | Command              | State     | Exit Code\n")
	buf.WriteString("---------+----------------------+-----------+----------\n")

	for _, task := range tasks {
		fmt.Fprintf(&buf, "%-8s | %-20s | %-9s | %d\n",
			task.ID,
			truncateCommand(task.Command),
			task.Status.String(),
			task.ExitCode,
		)
	}

	return buf.String()
}

const maxCommandDisplayLen = 20

// truncateCommand shortens a command string for table display.
func truncateCommand(cmd string) string {
	if len(cmd) <= maxCommandDisplayLen {
		return cmd
	}

	const ellipsisSuffix = "..."

	return cmd[:maxCommandDisplayLen-len(ellipsisSuffix)] + ellipsisSuffix
}
