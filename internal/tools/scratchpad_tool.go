package tools

import (
	"errors"
	"context"
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

// Name implements the Name operation.
func (t *ScratchpadTool) Name() string {
	return "scratchpad"
}

// Description implements the Description operation.
func (t *ScratchpadTool) Description() string {
	return "Store and retrieve session-scoped ephemeral memory. Use this to offload " +
		"information from the context window that you may need to reference in the future."
}

// Schema implements the Schema operation.
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

// Execute implements the Execute operation.
func (t *ScratchpadTool) Execute(ctx context.Context, params ToolParameters) (ToolResult, error) {
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
	case "pin":
		return t.executePin(ctx, params)
	case "unpin":
		return t.executeUnpin(ctx, params)
	case "clear":
		return t.executeClear(ctx, params)
	default:
		return NewToolError(fmt.Errorf("unknown operation: %s: %w", operation, ErrUnknownOperation)), nil
	}
}

func (t *ScratchpadTool) executePut(ctx context.Context, params ToolParameters) (ToolResult, error) {
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

	putErr := t.scratchpad.Put(ctx, key, value, opts)
	if putErr != nil {
		return ErrToResultf("failed to store entry: %v", putErr)
	}

	return NewToolResult(fmt.Sprintf("Stored entry with key '%s' (%d bytes)", key, len(value))), nil
}

func (t *ScratchpadTool) executeGet(ctx context.Context, params ToolParameters) (ToolResult, error) {
	key, _ := params.GetString("key")
	if key == "" {
		return NewToolError(ErrKeyParameterRequiredForGet), nil
	}

	entry, getErr := t.scratchpad.Get(ctx, key)
	if getErr != nil {
		if errors.Is(getErr, memory.ErrNotFound) {
			return ErrToResultf("key '%s' not found in scratchpad", fmt.Errorf("%s: %w", key, ErrKeyNotFound))
		}

		return ErrToResultf("failed to get entry: %v", getErr)
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
	key, _ := params.GetString("key")
	if key == "" {
		return NewToolError(ErrKeyParameterRequiredForDelete), nil
	}

	delErr := t.scratchpad.Delete(ctx, key)
	if delErr != nil {
		return ErrToResultf("failed to delete entry: %v", delErr)
	}

	return NewToolResult(fmt.Sprintf("Entry '%s' deleted", key)), nil
}

func (t *ScratchpadTool) executeList(ctx context.Context, params ToolParameters) (ToolResult, error) {
	pattern := params.GetStringOr("pattern", "*")

	keys, listErr := t.scratchpad.List(ctx, pattern)
	if listErr != nil {
		return ErrToResultf("failed to list entries: %v", listErr)
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
	query, _ := params.GetString("query")
	if query == "" {
		return NewToolError(ErrQueryParameterRequiredForSearch), nil
	}

	entries, searchErr := t.scratchpad.Search(ctx, query, 10)
	if searchErr != nil {
		return ErrToResultf("failed to search entries: %v", searchErr)
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

func (t *ScratchpadTool) executePin(_ context.Context, params ToolParameters) (ToolResult, error) {
	key, _ := params.GetString("key")
	if key == "" {
		return NewToolError(ErrKeyParameterRequiredForPin), nil
	}

	pinErr := t.scratchpad.Pin(key)
	if pinErr != nil {
		if errors.Is(pinErr, memory.ErrNotFound) {
			return ErrToResultf("key '%s' not found in scratchpad", fmt.Errorf("%s: %w", key, ErrKeyNotFound))
		}

		return ErrToResultf("failed to pin entry: %v", pinErr)
	}

	return NewToolResult(fmt.Sprintf("Entry '%s' pinned (protected from eviction)", key)), nil
}

func (t *ScratchpadTool) executeUnpin(_ context.Context, params ToolParameters) (ToolResult, error) {
	key, _ := params.GetString("key")
	if key == "" {
		return NewToolError(ErrKeyParameterRequiredForUnpin), nil
	}

	unpinErr := t.scratchpad.Unpin(key)
	if unpinErr != nil {
		if errors.Is(unpinErr, memory.ErrNotFound) {
			return ErrToResultf("key '%s' not found in scratchpad", fmt.Errorf("%s: %w", key, ErrKeyNotFound))
		}

		return ErrToResultf("failed to unpin entry: %v", unpinErr)
	}

	return NewToolResult(fmt.Sprintf("Entry '%s' unpinned (can be evicted)", key)), nil
}

func (t *ScratchpadTool) executeClear(_ context.Context, _ ToolParameters) (ToolResult, error) {
	t.scratchpad.Clear()

	return NewToolResult("Scratchpad cleared"), nil
}
