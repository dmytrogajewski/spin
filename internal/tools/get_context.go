package tools

import (
	"context"
	"fmt"
)

// GetContextTool implements environment context retrieval.
type GetContextTool struct {
	stringer fmt.Stringer
}

// NewGetContextTool creates a new get context tool.
// Accepts [fmt.Stringer] to avoid circular import with agent package.
func NewGetContextTool(env fmt.Stringer) *GetContextTool {
	return &GetContextTool{
		stringer: env,
	}
}

const getContextName = "get_context"

// Name implements the Name operation.
func (t *GetContextTool) Name() string {
	return getContextName
}

// Description implements the Description operation.
func (t *GetContextTool) Description() string {
	return "Get environment context information"
}

// Schema implements the Schema operation.
func (t *GetContextTool) Schema() ToolSchema {
	return ToolSchema{
		Type: "function",
		Function: FunctionSchema{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters: ParameterSchema{
				Type:       "object",
				Properties: map[string]PropertyDefinition{},
				Required:   []string{},
			},
		},
	}
}

// Execute implements the Execute operation.
func (t *GetContextTool) Execute(_ context.Context, _ ToolParameters) (ToolResult, error) {
	if t.stringer == nil {
		return ToolResult{
			Success: false,
			Error:   "context not available",
		}, nil
	}

	return ToolResult{
		Success: true,
		Output:  t.stringer.String(),
	}, nil
}
