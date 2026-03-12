package runtime

import (
	"context"

	"github.com/dmytrogajewski/spin/internal/tools"
)

// NotificationSender converts internal events to runtime-specific notifications.
// ACP runtime sends ACP protocol notifications, builtin runtime sends TUI events.
type NotificationSender interface {
	// SendToolCallStart notifies about a tool call starting.
	SendToolCallStart(ctx context.Context, toolID, toolName string, params tools.ToolParameters) error

	// SendToolCallUpdate notifies about a tool call progress update.
	// status: "pending", "in_progress", "completed", "failed"
	// content: runtime-specific content (ACP ToolCallContent, TUI blocks, etc.)
	SendToolCallUpdate(ctx context.Context, toolID string, status string, content any) error

	// SendToolCallComplete notifies about a tool call completion.
	SendToolCallComplete(ctx context.Context, toolID string, success bool, output string, err error) error

	// SendMessageChunk sends a chunk of agent message content.
	SendMessageChunk(ctx context.Context, content string) error

	// SendPlanUpdate sends plan entries update.
	SendPlanUpdate(ctx context.Context, entries []PlanEntry) error
}

// PlanEntry represents a plan entry for runtime notifications.
type PlanEntry struct {
	Content  string
	Priority string // "high", "medium", "low".
	Status   string // "pending", "in_progress", "completed".
}
