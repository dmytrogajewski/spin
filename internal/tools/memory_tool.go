package tools

import (
	"context"
	"fmt"
	"strings"
	"time"

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
		return NewToolError(errOperationParameterRequired), nil
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
		return NewToolError(fmt.Errorf("unknown operation: %s: %w", operation, errUnknownOperation)), nil
	}
}

// memoryEntryFormatter adds timestamps to the get output for persistent memory entries.
func memoryEntryFormatter(entry *memory.Entry, sb *strings.Builder) {
	fmt.Fprintf(sb, "Created: %s\n", entry.CreatedAt.Format(time.DateTime))
	fmt.Fprintf(sb, "Updated: %s\n", entry.UpdatedAt.Format(time.DateTime))
}

func (t *MemoryTool) executePut(ctx context.Context, params ToolParameters) (ToolResult, error) {
	return storePut(ctx, t.store, params, "persistent")
}

func (t *MemoryTool) executeGet(ctx context.Context, params ToolParameters) (ToolResult, error) {
	return storeGet(ctx, t.store, params, "persistent memory", memoryEntryFormatter)
}

func (t *MemoryTool) executeDelete(ctx context.Context, params ToolParameters) (ToolResult, error) {
	return storeDelete(ctx, t.store, params, "persistent memory")
}

func (t *MemoryTool) executeList(ctx context.Context, params ToolParameters) (ToolResult, error) {
	return storeList(ctx, t.store, params, "persistent memory")
}

func (t *MemoryTool) executeSearch(ctx context.Context, params ToolParameters) (ToolResult, error) {
	return storeSearch(ctx, t.store, params, defaultMemorySearchLimit, maxMemoryPreviewLen, "persistent memory")
}
