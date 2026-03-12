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
func NewGetContextTool(context any) *GetContextTool {
	return &GetContextTool{
		context: context,
	}
}

func (t *GetContextTool) Name() string {
	return "get_context"
}

func (t *GetContextTool) Description() string {
	return "Get environment context information"
}

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

func (t *GetContextTool) Execute(ctx context.Context, params ToolParameters) (ToolResult, error) {
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
