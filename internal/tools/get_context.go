package tools

import (
	"context"
	"reflect"
)

// GetContextTool implements environment context retrieval.
type GetContextTool struct {
	context any // agent.Environment - using interface{} to avoid circular import.
}

// NewGetContextTool creates a new get context tool.
func NewGetContextTool(env any) *GetContextTool {
	return &GetContextTool{
		context: env,
	}
}

// Name implements the Name operation.
func (t *GetContextTool) Name() string {
	return "get_context"
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
	if t.context == nil {
		return ToolResult{
			Success: false,
			Error:   "context not available",
		}, nil
	}

	// Use reflection to call String() method to avoid circular import
	// The context is agent.Environment which implements String() string.
	val := reflect.ValueOf(t.context)

	// Check if the context has a String() method.
	stringMethod := val.MethodByName("String")
	if !stringMethod.IsValid() {
		return ToolResult{
			Success: false,
			Error:   "context does not implement String() method",
		}, nil
	}

	// Call String() method.
	results := stringMethod.Call(nil)
	if len(results) != 1 {
		return ToolResult{
			Success: false,
			Error:   "invalid String() method signature",
		}, nil
	}

	// Extract string result.
	output, ok := results[0].Interface().(string)
	if !ok {
		return ToolResult{
			Success: false,
			Error:   "String() method did not return a string",
		}, nil
	}

	return ToolResult{
		Success: true,
		Output:  output,
	}, nil
}
