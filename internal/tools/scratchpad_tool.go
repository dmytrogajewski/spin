package tools

import (
	"context"
	"errors"
	"fmt"

	"github.com/dmytrogajewski/spin/internal/memory"
)

const (
	defaultScratchpadSearchLimit = 10
	maxScratchpadPreviewLen      = 100
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
	operation := params.GetStringOr("operation", "")
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
	case "pin":
		return t.executePin(ctx, params)
	case "unpin":
		return t.executeUnpin(ctx, params)
	case "clear":
		return t.executeClear(ctx, params)
	default:
		return NewToolError(unknownOperationError(operation)), nil
	}
}

func (t *ScratchpadTool) executePut(ctx context.Context, params ToolParameters) (ToolResult, error) {
	return storePut(ctx, t.scratchpad, params, "")
}

func (t *ScratchpadTool) executeGet(ctx context.Context, params ToolParameters) (ToolResult, error) {
	return storeGet(ctx, t.scratchpad, params, "scratchpad", nil)
}

func (t *ScratchpadTool) executeDelete(ctx context.Context, params ToolParameters) (ToolResult, error) {
	return storeDelete(ctx, t.scratchpad, params, "")
}

func (t *ScratchpadTool) executeList(ctx context.Context, params ToolParameters) (ToolResult, error) {
	return storeList(ctx, t.scratchpad, params, "scratchpad")
}

func (t *ScratchpadTool) executeSearch(ctx context.Context, params ToolParameters) (ToolResult, error) {
	return storeSearch(ctx, t.scratchpad, params, defaultScratchpadSearchLimit, maxScratchpadPreviewLen, "scratchpad")
}

func (t *ScratchpadTool) executePin(_ context.Context, params ToolParameters) (ToolResult, error) {
	key := params.GetStringOr("key", "")
	if key == "" {
		return NewToolError(errKeyParameterRequiredForPin), nil
	}

	pinErr := t.scratchpad.Pin(key)
	if pinErr != nil {
		if errors.Is(pinErr, memory.ErrNotFound) {
			return ErrToResultf("key '%s' not found in scratchpad", fmt.Errorf("%s: %w", key, errKeyNotFound))
		}

		return ErrToResultf("failed to pin entry: %v", pinErr)
	}

	return NewToolResult(fmt.Sprintf("Entry '%s' pinned (protected from eviction)", key)), nil
}

func (t *ScratchpadTool) executeUnpin(_ context.Context, params ToolParameters) (ToolResult, error) {
	key := params.GetStringOr("key", "")
	if key == "" {
		return NewToolError(errKeyParameterRequiredForUnpin), nil
	}

	unpinErr := t.scratchpad.Unpin(key)
	if unpinErr != nil {
		if errors.Is(unpinErr, memory.ErrNotFound) {
			return ErrToResultf("key '%s' not found in scratchpad", fmt.Errorf("%s: %w", key, errKeyNotFound))
		}

		return ErrToResultf("failed to unpin entry: %v", unpinErr)
	}

	return NewToolResult(fmt.Sprintf("Entry '%s' unpinned (can be evicted)", key)), nil
}

func (t *ScratchpadTool) executeClear(_ context.Context, _ ToolParameters) (ToolResult, error) {
	t.scratchpad.Clear()

	return NewToolResult("Scratchpad cleared"), nil
}
