// Package toolparams provides type-safe parameter handling for tool execution.
//
// This package replaces the use of map[string]interface{} with a structured
// ToolParameters type that provides type-safe accessors and better error handling.
//
// # Basic Usage
//
//	params, err := toolparams.FromMap(rawParams)
//	if err != nil {
//	    return err
//	}
//
// Type-safe access with error handling
//
//	filePath, err := params.GetString("file_path")
//	if err != nil {
//	    return fmt.Errorf("missing file_path: %w", err)
//	}
//
// Optional parameters with defaults
//
//	timeout := params.GetIntOr("timeout", 30)
//
// # Migration from map[string]interface{}
//
// Before:
//
//	func Execute(ctx context.Context, params map[string]interface{}) (ToolResult, error) {
//	    filePath, ok := params["file_path"].(string)
//	    if !ok {
//	        return ToolResult{}, errors.New("file_path must be a string")
//	    }
//	}
//
// After:
//
//	func Execute(ctx context.Context, params toolparams.ToolParameters) (ToolResult, error) {
//	    filePath, err := params.GetString("file_path")
//	    if err != nil {
//	        return ToolResult{}, fmt.Errorf("invalid file_path: %w", err)
//	    }
//	}
package tools
