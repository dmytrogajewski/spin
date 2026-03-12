package tools

import (
	"errors"
	"context"
	"fmt"
	"strings"

	"github.com/dmytrogajewski/spin/internal/memory"
)

const (
	defaultMemorySearchLimit = 10
	maxMemoryPreviewLen      = 100
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

// Name implements the Name operation.
func (t *MemoryTool) Name() string {
	return "memory"
}

// Description implements the Description operation.
func (t *MemoryTool) Description() string {
	return "Store and retrieve persistent cross-session memory. Use this for information " +
		"that should persist across sessions like user preferences and learned patterns."
}

// Schema implements the Schema operation.
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

// Execute implements the Execute operation.
func (t *MemoryTool) Execute(ctx context.Context, params ToolParameters) (ToolResult, error) {
	operation, _ := params.GetString("operation")
	if operation == "" {
		return NewToolError(ErrOperationParameterRequired), nil
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
		return NewToolError(fmt.Errorf("unknown operation: %s: %w", operation, ErrUnknownOperation)), nil
	}
}

func (t *MemoryTool) executePut(ctx context.Context, params ToolParameters) (ToolResult, error) {
	key, _ := params.GetString("key")
	if key == "" {
		return NewToolError(ErrKeyParameterRequiredForPut), nil
	}

	value, _ := params.GetString("value")
	if value == "" {
		return NewToolError(ErrValueParameterRequiredForPut), nil
	}

	opts := memory.PutOptions{
		Overwrite: true,
	}

	ns, _ := params.GetString("namespace")
	if ns != "" {
		opts.Namespace = ns
	}

	// Handle tags - they come as an array.
	if params.Has("tags") {
		var tags []string
		tagsErr := params.GetObject("tags", &tags)
		if tagsErr == nil {
			opts.Tags = tags
		}
	}

	putErr := t.store.Put(ctx, key, value, opts)
	if putErr != nil {
		return NewToolError(fmt.Errorf("failed to store entry: %w", putErr)), nil
	}

	return NewToolResult(fmt.Sprintf("Stored entry with key '%s' (%d bytes, persistent)", key, len(value))), nil
}

func (t *MemoryTool) executeGet(ctx context.Context, params ToolParameters) (ToolResult, error) {
	key, _ := params.GetString("key")
	if key == "" {
		return NewToolError(ErrKeyParameterRequiredForGet), nil
	}

	entry, getErr := t.store.Get(ctx, key)
	if getErr != nil {
		if errors.Is(getErr, memory.ErrNotFound) {
			return ErrToResultf("key '%s' not found in persistent memory", fmt.Errorf("%s: %w", key, ErrKeyNotFound))
		}

		return ErrToResultf("failed to get entry: %v", getErr)
	}

	// Format output with metadata.
	var sb strings.Builder
	fmt.Fprintf(&sb, "Key: %s\n", entry.Key)
	fmt.Fprintf(&sb, "Namespace: %s\n", entry.Namespace)

	if len(entry.Tags) > 0 {
		fmt.Fprintf(&sb, "Tags: %s\n", strings.Join(entry.Tags, ", "))
	}

	fmt.Fprintf(&sb, "Created: %s\n", entry.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(&sb, "Updated: %s\n", entry.UpdatedAt.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(&sb, "Value:\n%s", entry.Value)

	return NewToolResult(sb.String()), nil
}

func (t *MemoryTool) executeDelete(ctx context.Context, params ToolParameters) (ToolResult, error) {
	key, _ := params.GetString("key")
	if key == "" {
		return NewToolError(ErrKeyParameterRequiredForDelete), nil
	}

	delErr := t.store.Delete(ctx, key)
	if delErr != nil {
		return ErrToResultf("failed to delete entry: %v", delErr)
	}

	return NewToolResult(fmt.Sprintf("Entry '%s' deleted from persistent memory", key)), nil
}

func (t *MemoryTool) executeList(ctx context.Context, params ToolParameters) (ToolResult, error) {
	pattern := params.GetStringOr("pattern", "*")

	keys, listErr := t.store.List(ctx, pattern)
	if listErr != nil {
		return ErrToResultf("failed to list entries: %v", listErr)
	}

	if len(keys) == 0 {
		return NewToolResult("No entries found in persistent memory"), nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Found %d entries in persistent memory:\n", len(keys))

	for _, key := range keys {
		fmt.Fprintf(&sb, "  - %s\n", key)
	}

	return NewToolResult(sb.String()), nil
}

func (t *MemoryTool) executeSearch(ctx context.Context, params ToolParameters) (ToolResult, error) {
	query, _ := params.GetString("query")
	if query == "" {
		return NewToolError(ErrQueryParameterRequiredForSearch), nil
	}

	entries, searchErr := t.store.Search(ctx, query, defaultMemorySearchLimit)
	if searchErr != nil {
		return ErrToResultf("failed to search entries: %v", searchErr)
	}

	if len(entries) == 0 {
		return NewToolResult(fmt.Sprintf("No entries found matching '%s' in persistent memory", query)), nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Found %d entries matching '%s':\n", len(entries), query)

	for _, entry := range entries {
		// Show preview of value (first 100 chars).
		preview := entry.Value
		if len(preview) > maxMemoryPreviewLen {
			preview = preview[:100] + "..."
		}

		fmt.Fprintf(&sb, "  - %s: %s\n", entry.Key, preview)
	}

	return NewToolResult(sb.String()), nil
}
