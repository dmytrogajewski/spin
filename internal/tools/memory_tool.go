package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/dmytrogajewski/spin/internal/memory"
)

// MemoryTool provides LLM access to persistent cross-session memory.
//
// The memory tool allows the agent to store information that persists
// across sessions, enabling long-term learning and preference storage.
type MemoryTool struct {
	store *memory.PersistentStore
}

// NewMemoryTool creates a new memory tool.
// Returns nil if store is nil.
func NewMemoryTool(store *memory.PersistentStore) *MemoryTool {
	if store == nil {
		return nil
	}
	return &MemoryTool{
		store: store,
	}
}

func (t *MemoryTool) Name() string {
	return "memory"
}

func (t *MemoryTool) Description() string {
	return "Store and retrieve persistent cross-session memory. Use this for information " +
		"that should persist across sessions like user preferences and learned patterns."
}

func (t *MemoryTool) Schema() ToolSchema {
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
						Description: "The operation to perform: put, get, delete, list, search",
						Enum:        []string{"put", "get", "delete", "list", "search"},
					},
					"key": {
						Type:        "string",
						Description: "The key for the entry (required for put, get, delete)",
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

func (t *MemoryTool) Execute(ctx context.Context, params ToolParameters) (ToolResult, error) {
	operation, err := params.GetString("operation")
	if err != nil {
		return NewToolError(fmt.Errorf("operation parameter is required")), nil
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
	default:
		return NewToolError(fmt.Errorf("unknown operation: %s", operation)), nil
	}
}

func (t *MemoryTool) executePut(ctx context.Context, params ToolParameters) (ToolResult, error) {
	key, err := params.GetString("key")
	if err != nil || key == "" {
		return NewToolError(fmt.Errorf("key parameter is required for put operation")), nil
	}

	value, err := params.GetString("value")
	if err != nil || value == "" {
		return NewToolError(fmt.Errorf("value parameter is required for put operation")), nil
	}

	opts := memory.PutOptions{
		Overwrite: true,
	}

	if ns, err := params.GetString("namespace"); err == nil && ns != "" {
		opts.Namespace = ns
	}

	// Handle tags - they come as an array
	if params.Has("tags") {
		var tags []string
		if err := params.GetObject("tags", &tags); err == nil {
			opts.Tags = tags
		}
	}

	if err := t.store.Put(ctx, key, value, opts); err != nil {
		return NewToolError(fmt.Errorf("failed to store entry: %w", err)), nil
	}

	return NewToolResult(fmt.Sprintf("Stored entry with key '%s' (%d bytes, persistent)", key, len(value))), nil
}

func (t *MemoryTool) executeGet(ctx context.Context, params ToolParameters) (ToolResult, error) {
	key, err := params.GetString("key")
	if err != nil || key == "" {
		return NewToolError(fmt.Errorf("key parameter is required for get operation")), nil
	}

	entry, err := t.store.Get(ctx, key)
	if err != nil {
		if err == memory.ErrNotFound {
			return NewToolError(fmt.Errorf("key '%s' not found in persistent memory", key)), nil
		}
		return NewToolError(fmt.Errorf("failed to get entry: %w", err)), nil
	}

	// Format output with metadata
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Key: %s\n", entry.Key))
	sb.WriteString(fmt.Sprintf("Namespace: %s\n", entry.Namespace))
	if len(entry.Tags) > 0 {
		sb.WriteString(fmt.Sprintf("Tags: %s\n", strings.Join(entry.Tags, ", ")))
	}
	sb.WriteString(fmt.Sprintf("Created: %s\n", entry.CreatedAt.Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("Updated: %s\n", entry.UpdatedAt.Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("Value:\n%s", entry.Value))

	return NewToolResult(sb.String()), nil
}

func (t *MemoryTool) executeDelete(ctx context.Context, params ToolParameters) (ToolResult, error) {
	key, err := params.GetString("key")
	if err != nil || key == "" {
		return NewToolError(fmt.Errorf("key parameter is required for delete operation")), nil
	}

	if err := t.store.Delete(ctx, key); err != nil {
		return NewToolError(fmt.Errorf("failed to delete entry: %w", err)), nil
	}

	return NewToolResult(fmt.Sprintf("Entry '%s' deleted from persistent memory", key)), nil
}

func (t *MemoryTool) executeList(ctx context.Context, params ToolParameters) (ToolResult, error) {
	pattern := params.GetStringOr("pattern", "*")

	keys, err := t.store.List(ctx, pattern)
	if err != nil {
		return NewToolError(fmt.Errorf("failed to list entries: %w", err)), nil
	}

	if len(keys) == 0 {
		return NewToolResult("No entries found in persistent memory"), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d entries in persistent memory:\n", len(keys)))
	for _, key := range keys {
		sb.WriteString(fmt.Sprintf("  - %s\n", key))
	}

	return NewToolResult(sb.String()), nil
}

func (t *MemoryTool) executeSearch(ctx context.Context, params ToolParameters) (ToolResult, error) {
	query, err := params.GetString("query")
	if err != nil || query == "" {
		return NewToolError(fmt.Errorf("query parameter is required for search operation")), nil
	}

	entries, err := t.store.Search(ctx, query, 10)
	if err != nil {
		return NewToolError(fmt.Errorf("failed to search entries: %w", err)), nil
	}

	if len(entries) == 0 {
		return NewToolResult(fmt.Sprintf("No entries found matching '%s' in persistent memory", query)), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d entries matching '%s':\n", len(entries), query))
	for _, entry := range entries {
		// Show preview of value (first 100 chars)
		preview := entry.Value
		if len(preview) > 100 {
			preview = preview[:100] + "..."
		}
		sb.WriteString(fmt.Sprintf("  - %s: %s\n", entry.Key, preview))
	}

	return NewToolResult(sb.String()), nil
}
