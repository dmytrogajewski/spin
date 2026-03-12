package tools

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/dmytrogajewski/spin/internal/memory"
)

// ScratchpadTool provides LLM access to session-scoped ephemeral memory.
//
// The scratchpad allows the agent to store information outside the immediate
// context window, enabling efficient context management during long sessions.
type ScratchpadTool struct {
	scratchpad *memory.Scratchpad
}

// NewScratchpadTool creates a new scratchpad tool.
// Returns nil if scratchpad is nil.
func NewScratchpadTool(scratchpad *memory.Scratchpad) *ScratchpadTool {
	if scratchpad == nil {
		return nil
	}

	return &ScratchpadTool{
		scratchpad: scratchpad,
	}
}

func (t *ScratchpadTool) Name() string {
	return "scratchpad"
}

func (t *ScratchpadTool) Description() string {
	return "Store and retrieve session-scoped ephemeral memory. Use this to offload " +
		"information from the context window that you may need to reference in the future."
}

func (t *ScratchpadTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "function",
		Function: FunctionSchema{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters: ParameterSchema{
				Type: "object",
				Properties: map[string]PropertyDefinition{
					"operation": {
						Type:        "string",
						Description: "The operation to perform: put, get, delete, list, search, pin, unpin, clear",
						Enum:        []string{"put", "get", "delete", "list", "search", "pin", "unpin", "clear"},
					},
					"key": {
						Type:        "string",
						Description: "The key for the entry (required for put, get, delete, pin, unpin)",
					},
					"value": {
						Type:        "string",
						Description: "The value to store (required for put)",
					},
					"pattern": {
						Type:        "string",
						Description: "Glob pattern for list operation (e.g., 'prefix*'). Defaults to '*' (all keys)",
					},
					"query": {
						Type:        "string",
						Description: "Search query for search operation",
					},
					"namespace": {
						Type:        "string",
						Description: "Logical grouping for the entry (optional, defaults to 'default')",
					},
					"tags": {
						Type:        "array",
						Description: "Labels for filtering (optional)",
					},
				},
				Required: []string{"operation"},
			},
		},
	}
}

func (t *ScratchpadTool) Execute(ctx context.Context, params ToolParameters) (ToolResult, error) {
	operation, err := params.GetString("operation")
	if err != nil {
		return NewToolError(errors.New("operation parameter is required")), nil
	}

	switch operation {
	case "put":
		return t.executePut(ctx, params)
	case "get":
		return t.executeGet(ctx, params)
	case "delete":
		return t.executeDelete(ctx, params)
	case "list":
		return t.executeList(ctx, params)
	case "search":
		return t.executeSearch(ctx, params)
	case "pin":
		return t.executePin(ctx, params)
	case "unpin":
		return t.executeUnpin(ctx, params)
	case "clear":
		return t.executeClear(ctx, params)
	default:
		return NewToolError(fmt.Errorf("unknown operation: %s", operation)), nil
	}
}

func (t *ScratchpadTool) executePut(ctx context.Context, params ToolParameters) (ToolResult, error) {
	key, err := params.GetString("key")
	if err != nil || key == "" {
		return NewToolError(errors.New("key parameter is required for put operation")), nil
	}

	value, err := params.GetString("value")
	if err != nil || value == "" {
		return NewToolError(errors.New("value parameter is required for put operation")), nil
	}

	opts := memory.PutOptions{
		Overwrite: true,
	}

	ns, err := params.GetString("namespace")
	if err == nil && ns != "" {
		opts.Namespace = ns
	}

	// Handle tags - they come as an array.
	if params.Has("tags") {
		var tags []string
		err := params.GetObject("tags", &tags)
		if err == nil {
			opts.Tags = tags
		}
	}

	err = t.scratchpad.Put(ctx, key, value, opts)
	if err != nil {
		return NewToolError(fmt.Errorf("failed to store entry: %w", err)), nil
	}

	return NewToolResult(fmt.Sprintf("Stored entry with key '%s' (%d bytes)", key, len(value))), nil
}

func (t *ScratchpadTool) executeGet(ctx context.Context, params ToolParameters) (ToolResult, error) {
	key, err := params.GetString("key")
	if err != nil || key == "" {
		return NewToolError(errors.New("key parameter is required for get operation")), nil
	}

	entry, err := t.scratchpad.Get(ctx, key)
	if err != nil {
		if err == memory.ErrNotFound {
			return NewToolError(fmt.Errorf("key '%s' not found in scratchpad", key)), nil
		}

		return NewToolError(fmt.Errorf("failed to get entry: %w", err)), nil
	}

	// Format output with metadata.
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Key: %s\n", entry.Key))
	sb.WriteString(fmt.Sprintf("Namespace: %s\n", entry.Namespace))

	if len(entry.Tags) > 0 {
		sb.WriteString(fmt.Sprintf("Tags: %s\n", strings.Join(entry.Tags, ", ")))
	}

	sb.WriteString(fmt.Sprintf("Value:\n%s", entry.Value))

	return NewToolResult(sb.String()), nil
}

func (t *ScratchpadTool) executeDelete(ctx context.Context, params ToolParameters) (ToolResult, error) {
	key, err := params.GetString("key")
	if err != nil || key == "" {
		return NewToolError(errors.New("key parameter is required for delete operation")), nil
	}

	err = t.scratchpad.Delete(ctx, key)
	if err != nil {
		return NewToolError(fmt.Errorf("failed to delete entry: %w", err)), nil
	}

	return NewToolResult(fmt.Sprintf("Entry '%s' deleted", key)), nil
}

func (t *ScratchpadTool) executeList(ctx context.Context, params ToolParameters) (ToolResult, error) {
	pattern := params.GetStringOr("pattern", "*")

	keys, err := t.scratchpad.List(ctx, pattern)
	if err != nil {
		return NewToolError(fmt.Errorf("failed to list entries: %w", err)), nil
	}

	if len(keys) == 0 {
		return NewToolResult("No entries found"), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d entries:\n", len(keys)))

	for _, key := range keys {
		sb.WriteString(fmt.Sprintf("  - %s\n", key))
	}

	return NewToolResult(sb.String()), nil
}

func (t *ScratchpadTool) executeSearch(ctx context.Context, params ToolParameters) (ToolResult, error) {
	query, err := params.GetString("query")
	if err != nil || query == "" {
		return NewToolError(errors.New("query parameter is required for search operation")), nil
	}

	entries, err := t.scratchpad.Search(ctx, query, 10)
	if err != nil {
		return NewToolError(fmt.Errorf("failed to search entries: %w", err)), nil
	}

	if len(entries) == 0 {
		return NewToolResult(fmt.Sprintf("No entries found matching '%s'", query)), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d entries matching '%s':\n", len(entries), query))

	for _, entry := range entries {
		// Show preview of value (first 100 chars).
		preview := entry.Value
		if len(preview) > 100 {
			preview = preview[:100] + "..."
		}

		sb.WriteString(fmt.Sprintf("  - %s: %s\n", entry.Key, preview))
	}

	return NewToolResult(sb.String()), nil
}

func (t *ScratchpadTool) executePin(ctx context.Context, params ToolParameters) (ToolResult, error) {
	key, err := params.GetString("key")
	if err != nil || key == "" {
		return NewToolError(errors.New("key parameter is required for pin operation")), nil
	}

	err = t.scratchpad.Pin(key)
	if err != nil {
		if err == memory.ErrNotFound {
			return NewToolError(fmt.Errorf("key '%s' not found in scratchpad", key)), nil
		}

		return NewToolError(fmt.Errorf("failed to pin entry: %w", err)), nil
	}

	return NewToolResult(fmt.Sprintf("Entry '%s' pinned (protected from eviction)", key)), nil
}

func (t *ScratchpadTool) executeUnpin(ctx context.Context, params ToolParameters) (ToolResult, error) {
	key, err := params.GetString("key")
	if err != nil || key == "" {
		return NewToolError(errors.New("key parameter is required for unpin operation")), nil
	}

	err = t.scratchpad.Unpin(key)
	if err != nil {
		if err == memory.ErrNotFound {
			return NewToolError(fmt.Errorf("key '%s' not found in scratchpad", key)), nil
		}

		return NewToolError(fmt.Errorf("failed to unpin entry: %w", err)), nil
	}

	return NewToolResult(fmt.Sprintf("Entry '%s' unpinned (can be evicted)", key)), nil
}

func (t *ScratchpadTool) executeClear(ctx context.Context, params ToolParameters) (ToolResult, error) {
	t.scratchpad.Clear()

	return NewToolResult("Scratchpad cleared"), nil
}
