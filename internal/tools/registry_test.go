package tools

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
)

// contains checks if s contains substr.
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// TestNewRegistryWithBuiltins_ContainsAllBuiltins verifies that the constructor
// pre-populates the registry with all builtin tools.
func TestNewRegistryWithBuiltins_ContainsAllBuiltins(t *testing.T) {
	registry := NewRegistryWithBuiltins()

	for _, tool := range BuiltinTools {
		name := tool.Name()
		retrieved, err := registry.Get(name)
		if err != nil {
			t.Errorf("NewRegistryWithBuiltins() missing builtin tool %q: %v", name, err)
			continue
		}
		if retrieved.Name() != name {
			t.Errorf("retrieved tool name = %q, want %q", retrieved.Name(), name)
		}
	}
}

// TestNewRegistryWithBuiltins_CountMatches verifies the count matches BuiltinTools.
func TestNewRegistryWithBuiltins_CountMatches(t *testing.T) {
	registry := NewRegistryWithBuiltins()

	tools := registry.List()
	if len(tools) != len(BuiltinTools) {
		t.Errorf("NewRegistryWithBuiltins() tool count = %d, want %d", len(tools), len(BuiltinTools))
	}
}

// mockTool is a simple mock implementation for testing.
type mockTool struct {
	name        string
	description string
	schema      ToolSchema
	executeFunc func(context.Context, ToolParameters) (ToolResult, error)
}

func (m *mockTool) Name() string        { return m.name }
func (m *mockTool) Description() string { return m.description }
func (m *mockTool) Schema() ToolSchema  { return m.schema }
func (m *mockTool) Execute(ctx context.Context, params ToolParameters) (ToolResult, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, params)
	}
	return ToolResult{Success: true, Output: "mock result"}, nil
}

func newMockTool(name string) *mockTool {
	return &mockTool{
		name:        name,
		description: "Mock tool for testing",
		schema: ToolSchema{
			Type: "function",
			Function: FunctionSchema{
				Name:        name,
				Description: "Mock tool for testing",
				Parameters: ParameterSchema{
					Type: "object",
					Properties: map[string]PropertyDefinition{
						"param1": {
							Type:        "string",
							Description: "Test parameter",
						},
					},
					Required: []string{"param1"},
				},
			},
		},
	}
}

func TestNewRegistry(t *testing.T) {
	reg := NewRegistry()
	if reg == nil {
		t.Fatal("NewRegistry() returned nil")
	}

	// Should be empty initially
	tools := reg.List()
	if len(tools) != 0 {
		t.Errorf("expected empty registry, got %d tools", len(tools))
	}
}

func TestRegistryRegisterOrReplace(t *testing.T) {
	t.Run("register new tool", func(t *testing.T) {
		reg := NewRegistry()
		tool1 := newMockTool("tool1")

		err := reg.RegisterOrReplace(tool1)
		if err != nil {
			t.Fatalf("RegisterOrReplace() unexpected error: %v", err)
		}

		// Verify tool was registered
		retrieved, err := reg.Get("tool1")
		if err != nil {
			t.Fatalf("Get() error: %v", err)
		}
		if retrieved.Name() != "tool1" {
			t.Errorf("expected tool name 'tool1', got %q", retrieved.Name())
		}
	})

	t.Run("replace existing tool", func(t *testing.T) {
		reg := NewRegistry()

		// Register initial tool
		tool1 := newMockTool("tool1")
		tool1.description = "Original description"
		_ = reg.Register(tool1)

		// Replace with new tool
		tool1Updated := newMockTool("tool1")
		tool1Updated.description = "Updated description"
		err := reg.RegisterOrReplace(tool1Updated)
		if err != nil {
			t.Fatalf("RegisterOrReplace() unexpected error: %v", err)
		}

		// Verify tool was replaced
		retrieved, err := reg.Get("tool1")
		if err != nil {
			t.Fatalf("Get() error: %v", err)
		}
		if retrieved.Description() != "Updated description" {
			t.Errorf("expected description 'Updated description', got %q", retrieved.Description())
		}
	})

	t.Run("replace does not affect other tools", func(t *testing.T) {
		reg := NewRegistry()

		// Register multiple tools
		_ = reg.Register(newMockTool("tool1"))
		_ = reg.Register(newMockTool("tool2"))
		_ = reg.Register(newMockTool("tool3"))

		// Replace tool2
		tool2Updated := newMockTool("tool2")
		tool2Updated.description = "Updated tool2"
		_ = reg.RegisterOrReplace(tool2Updated)

		// Verify all tools are still present
		tools := reg.List()
		if len(tools) != 3 {
			t.Errorf("expected 3 tools, got %d", len(tools))
		}

		// Verify tool1 and tool3 are unchanged
		tool1, _ := reg.Get("tool1")
		if tool1.Description() != "Mock tool for testing" {
			t.Error("tool1 should not have been modified")
		}

		tool3, _ := reg.Get("tool3")
		if tool3.Description() != "Mock tool for testing" {
			t.Error("tool3 should not have been modified")
		}
	})
}

func TestRegistryRegister(t *testing.T) {
	tests := []struct {
		name      string
		tools     []Tool
		wantErr   error
		wantCount int
	}{
		{
			name:      "register single tool",
			tools:     []Tool{newMockTool("tool1")},
			wantCount: 1,
		},
		{
			name:      "register multiple tools",
			tools:     []Tool{newMockTool("tool1"), newMockTool("tool2"), newMockTool("tool3")},
			wantCount: 3,
		},
		{
			name:    "register duplicate tool",
			tools:   []Tool{newMockTool("tool1"), newMockTool("tool1")},
			wantErr: ErrDuplicateTool,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := NewRegistry()
			var lastErr error

			for _, tool := range tt.tools {
				err := reg.Register(tool)
				if err != nil {
					lastErr = err
				}
			}

			if tt.wantErr != nil {
				if !errors.Is(lastErr, tt.wantErr) {
					t.Errorf("expected error %v, got %v", tt.wantErr, lastErr)
				}
				return
			}

			if lastErr != nil {
				t.Errorf("unexpected error: %v", lastErr)
			}

			if len(reg.List()) != tt.wantCount {
				t.Errorf("expected %d tools, got %d", tt.wantCount, len(reg.List()))
			}
		})
	}
}

func TestRegistryGet(t *testing.T) {
	reg := NewRegistry()
	tool1 := newMockTool("tool1")
	_ = reg.Register(tool1)

	tests := []struct {
		name     string
		toolName string
		wantErr  error
		wantTool Tool
	}{
		{
			name:     "get existing tool",
			toolName: "tool1",
			wantTool: tool1,
		},
		{
			name:     "get non-existent tool",
			toolName: "nonexistent",
			wantErr:  ErrToolNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool, err := reg.Get(tt.toolName)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if tool != tt.wantTool {
				t.Errorf("expected tool %v, got %v", tt.wantTool, tool)
			}
		})
	}
}

func TestRegistryList(t *testing.T) {
	reg := NewRegistry()

	// Empty list
	if len(reg.List()) != 0 {
		t.Error("expected empty list")
	}

	// Add tools
	_ = reg.Register(newMockTool("tool1"))
	_ = reg.Register(newMockTool("tool2"))
	_ = reg.Register(newMockTool("tool3"))

	tools := reg.List()
	if len(tools) != 3 {
		t.Errorf("expected 3 tools, got %d", len(tools))
	}

	// Verify all tools present
	names := make(map[string]bool)
	for _, tool := range tools {
		names[tool.Name()] = true
	}

	for _, want := range []string{"tool1", "tool2", "tool3"} {
		if !names[want] {
			t.Errorf("missing tool %s", want)
		}
	}
}

func TestRegistryListSchemas(t *testing.T) {
	reg := NewRegistry()

	// Empty schemas
	if len(reg.ListSchemas()) != 0 {
		t.Error("expected empty schemas")
	}

	// Add tools
	_ = reg.Register(newMockTool("tool1"))
	_ = reg.Register(newMockTool("tool2"))

	schemas := reg.ListSchemas()
	if len(schemas) != 2 {
		t.Errorf("expected 2 schemas, got %d", len(schemas))
	}

	// Verify schema structure
	for _, schema := range schemas {
		if schema.Type != "function" {
			t.Errorf("expected type 'function', got %s", schema.Type)
		}
		if schema.Function.Name == "" {
			t.Error("expected non-empty function name")
		}
		if schema.Function.Parameters.Type != "object" {
			t.Errorf("expected parameters type 'object', got %s", schema.Function.Parameters.Type)
		}
	}
}

func TestRegistryExecute(t *testing.T) {
	reg := NewRegistry()

	// Tool that returns success
	successTool := &mockTool{
		name: "success_tool",
		executeFunc: func(_ context.Context, params ToolParameters) (ToolResult, error) {
			return ToolResult{
				Success: true,
				Output:  "success output",
			}, nil
		},
	}

	// Tool that returns error
	errorTool := &mockTool{
		name: "error_tool",
		executeFunc: func(_ context.Context, params ToolParameters) (ToolResult, error) {
			return ToolResult{}, errors.New("execution failed")
		},
	}

	// Tool that checks parameters
	paramTool := &mockTool{
		name: "param_tool",
		schema: ToolSchema{
			Type: "function",
			Function: FunctionSchema{
				Name: "param_tool",
				Parameters: ParameterSchema{
					Type: "object",
					Properties: map[string]PropertyDefinition{
						"required_param": {Type: "string", Description: "Required parameter"},
						"optional_param": {Type: "string", Description: "Optional parameter"},
					},
					Required: []string{"required_param"},
				},
			},
		},
		executeFunc: func(_ context.Context, params ToolParameters) (ToolResult, error) {
			val, err := params.GetString("required_param")
			if err == nil {
				return ToolResult{Success: true, Output: "received: " + val}, nil
			}
			return ToolResult{Success: false, Error: "missing parameter"}, nil
		},
	}

	_ = reg.Register(successTool)
	_ = reg.Register(errorTool)
	_ = reg.Register(paramTool)

	tests := []struct {
		name       string
		toolName   string
		params     map[string]interface{}
		wantErr    error
		wantResult ToolResult
	}{
		{
			name:     "execute success tool",
			toolName: "success_tool",
			params:   map[string]interface{}{},
			wantResult: ToolResult{
				Success: true,
				Output:  "success output",
			},
		},
		{
			name:     "execute error tool",
			toolName: "error_tool",
			params:   map[string]interface{}{},
			wantErr:  errors.New("execution failed"),
		},
		{
			name:     "execute non-existent tool",
			toolName: "nonexistent",
			params:   map[string]interface{}{},
			wantErr:  ErrToolNotFound,
		},
		{
			name:     "execute with valid required params",
			toolName: "param_tool",
			params: map[string]interface{}{
				"required_param": "test_value",
			},
			wantResult: ToolResult{
				Success: true,
				Output:  "received: test_value",
			},
		},
		{
			name:     "execute with missing required params",
			toolName: "param_tool",
			params: map[string]interface{}{
				"optional_param": "test",
			},
			wantErr: ErrInvalidParameters,
		},
		{
			name:     "execute with wrong param type",
			toolName: "param_tool",
			params: map[string]interface{}{
				"required_param": 123, // should be string
			},
			wantErr: ErrInvalidParameters,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			params, _ := FromMap(tt.params)
			result, err := reg.Execute(ctx, tt.toolName, params)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.wantErr)
					return
				}
				// Check if error matches expected (wrapped or direct)
				if !errors.Is(err, tt.wantErr) {
					// Also check if the wanted error message is contained
					if tt.wantErr.Error() != "" && !contains(err.Error(), tt.wantErr.Error()) {
						t.Errorf("expected error containing %q, got %q", tt.wantErr.Error(), err.Error())
					}
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result.Success != tt.wantResult.Success {
				t.Errorf("expected success %v, got %v", tt.wantResult.Success, result.Success)
			}

			if result.Output != tt.wantResult.Output {
				t.Errorf("expected output %q, got %q", tt.wantResult.Output, result.Output)
			}
		})
	}
}

func TestRegistryExecuteContextCancellation(t *testing.T) {
	reg := NewRegistry()

	// Tool that checks context
	ctxTool := &mockTool{
		name: "ctx_tool",
		executeFunc: func(ctx context.Context, params ToolParameters) (ToolResult, error) {
			select {
			case <-ctx.Done():
				return ToolResult{}, ctx.Err()
			default:
				return ToolResult{Success: true, Output: "completed"}, nil
			}
		},
	}

	_ = reg.Register(ctxTool)

	// Test with cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	params, _ := FromMap(map[string]interface{}{})
	_, err := reg.Execute(ctx, "ctx_tool", params)
	if err == nil {
		t.Error("expected context cancellation error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestRegistryConcurrentAccess(t *testing.T) {
	reg := NewRegistry()

	// Register initial tools
	for i := 0; i < 5; i++ {
		_ = reg.Register(newMockTool(fmt.Sprintf("tool-%d", i)))
	}

	// Concurrent reads and writes
	var wg sync.WaitGroup

	// Concurrent reads
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = reg.List()
			_ = reg.ListSchemas()
			_, _ = reg.Get("tool1")
		}()
	}

	// Concurrent executions
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			params, _ := FromMap(map[string]interface{}{"param1": "test"})
			_, _ = reg.Execute(context.Background(), "tool1", params)
		}()
	}

	// Concurrent registrations (should fail on duplicates, but shouldn't panic)
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_ = reg.Register(newMockTool(fmt.Sprintf("concurrent-%d", idx)))
		}(i)
	}

	wg.Wait()

	// Verify registry is still functional
	tools := reg.List()
	if len(tools) < 5 {
		t.Errorf("expected at least 5 tools after concurrent operations, got %d", len(tools))
	}
}

func TestRegistryTypeValidation(t *testing.T) {
	reg := NewRegistry()

	// Tool with various parameter types
	typeTool := &mockTool{
		name: "type_tool",
		schema: ToolSchema{
			Type: "function",
			Function: FunctionSchema{
				Name: "type_tool",
				Parameters: ParameterSchema{
					Type: "object",
					Properties: map[string]PropertyDefinition{
						"str_param":   {Type: "string"},
						"int_param":   {Type: "integer"},
						"num_param":   {Type: "number"},
						"bool_param":  {Type: "boolean"},
						"array_param": {Type: "array"},
						"obj_param":   {Type: "object"},
					},
					Required: []string{},
				},
			},
		},
		executeFunc: func(_ context.Context, params ToolParameters) (ToolResult, error) {
			return ToolResult{Success: true, Output: "ok"}, nil
		},
	}

	_ = reg.Register(typeTool)

	tests := []struct {
		name    string
		params  map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid string",
			params: map[string]interface{}{
				"str_param": "test",
			},
		},
		{
			name: "valid integer",
			params: map[string]interface{}{
				"int_param": 42,
			},
		},
		{
			name: "valid number - int",
			params: map[string]interface{}{
				"num_param": 42,
			},
		},
		{
			name: "valid number - float",
			params: map[string]interface{}{
				"num_param": 3.14,
			},
		},
		{
			name: "valid boolean",
			params: map[string]interface{}{
				"bool_param": true,
			},
		},
		{
			name: "valid array",
			params: map[string]interface{}{
				"array_param": []string{"a", "b", "c"},
			},
		},
		{
			name: "valid object",
			params: map[string]interface{}{
				"obj_param": map[string]interface{}{"key": "value"},
			},
		},
		{
			name: "invalid string type",
			params: map[string]interface{}{
				"str_param": 123,
			},
			wantErr: true,
		},
		{
			name: "invalid integer type",
			params: map[string]interface{}{
				"int_param": "not an int",
			},
			wantErr: true,
		},
		{
			name: "invalid boolean type",
			params: map[string]interface{}{
				"bool_param": "not a bool",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params, _ := FromMap(tt.params)
			_, err := reg.Execute(context.Background(), "type_tool", params)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestRegistryEnumValidation(t *testing.T) {
	reg := NewRegistry()

	// Tool with enum parameter
	enumTool := &mockTool{
		name: "enum_tool",
		schema: ToolSchema{
			Type: "function",
			Function: FunctionSchema{
				Name: "enum_tool",
				Parameters: ParameterSchema{
					Type: "object",
					Properties: map[string]PropertyDefinition{
						"mode": {
							Type:        "string",
							Description: "Mode parameter",
							Enum:        []string{"read", "write", "execute"},
						},
					},
					Required: []string{"mode"},
				},
			},
		},
		executeFunc: func(_ context.Context, params ToolParameters) (ToolResult, error) {
			return ToolResult{Success: true, Output: "ok"}, nil
		},
	}

	_ = reg.Register(enumTool)

	tests := []struct {
		name    string
		params  map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid enum value - read",
			params: map[string]interface{}{
				"mode": "read",
			},
		},
		{
			name: "valid enum value - write",
			params: map[string]interface{}{
				"mode": "write",
			},
		},
		{
			name: "invalid enum value",
			params: map[string]interface{}{
				"mode": "delete",
			},
			wantErr: true,
		},
		{
			name: "non-string enum value",
			params: map[string]interface{}{
				"mode": 123,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params, _ := FromMap(tt.params)
			_, err := reg.Execute(context.Background(), "enum_tool", params)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// TestRegistryExecute_UnknownParameter verifies that unknown parameters are rejected.
// See FRD-8.12 for specification.
func TestRegistryExecute_UnknownParameter(t *testing.T) {
	reg := NewRegistry()

	// Tool with defined parameters
	tool := &mockTool{
		name: "test_tool",
		schema: ToolSchema{
			Type: "function",
			Function: FunctionSchema{
				Name: "test_tool",
				Parameters: ParameterSchema{
					Type: "object",
					Properties: map[string]PropertyDefinition{
						"param1": {Type: "string", Description: "First parameter"},
						"param2": {Type: "string", Description: "Second parameter"},
					},
					Required: []string{}, // No required params
				},
			},
		},
		executeFunc: func(_ context.Context, params ToolParameters) (ToolResult, error) {
			return ToolResult{Success: true, Output: "ok"}, nil
		},
	}

	_ = reg.Register(tool)

	tests := []struct {
		name    string
		params  map[string]interface{}
		wantErr bool
		errMsg  string
	}{
		{
			name: "all known parameters",
			params: map[string]interface{}{
				"param1": "value1",
				"param2": "value2",
			},
			wantErr: false,
		},
		{
			name: "subset of known parameters",
			params: map[string]interface{}{
				"param1": "value1",
			},
			wantErr: false,
		},
		{
			name:    "empty parameters",
			params:  map[string]interface{}{},
			wantErr: false,
		},
		{
			name: "single unknown parameter",
			params: map[string]interface{}{
				"unknown_param": "value",
			},
			wantErr: true,
			errMsg:  "unknown parameter \"unknown_param\"",
		},
		{
			name: "known and unknown parameters",
			params: map[string]interface{}{
				"param1":        "value1",
				"unknown_param": "value",
			},
			wantErr: true,
			errMsg:  "unknown parameter \"unknown_param\"",
		},
		{
			name: "typo in parameter name",
			params: map[string]interface{}{
				"parm1": "value", // typo: parm1 instead of param1
			},
			wantErr: true,
			errMsg:  "unknown parameter \"parm1\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params, _ := FromMap(tt.params)
			_, err := reg.Execute(context.Background(), "test_tool", params)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
					return
				}
				if !errors.Is(err, ErrInvalidParameters) {
					t.Errorf("expected ErrInvalidParameters, got %v", err)
				}
				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error to contain %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// TestRegistryExecute_UnknownParameter_ErrorMessage verifies error messages are helpful.
// See FRD-8.12 TC-8.12.3 for specification.
func TestRegistryExecute_UnknownParameter_ErrorMessage(t *testing.T) {
	reg := NewRegistry()

	// Tool with multiple defined parameters
	tool := &mockTool{
		name: "file_tool",
		schema: ToolSchema{
			Type: "function",
			Function: FunctionSchema{
				Name: "file_tool",
				Parameters: ParameterSchema{
					Type: "object",
					Properties: map[string]PropertyDefinition{
						"filename": {Type: "string", Description: "File name"},
						"path":     {Type: "string", Description: "File path"},
						"mode":     {Type: "string", Description: "Access mode"},
					},
					Required: []string{"filename"},
				},
			},
		},
		executeFunc: func(_ context.Context, params ToolParameters) (ToolResult, error) {
			return ToolResult{Success: true, Output: "ok"}, nil
		},
	}

	_ = reg.Register(tool)

	// Execute with typo in parameter name
	params, _ := FromMap(map[string]interface{}{
		"filename": "test.txt",
		"fliename": "typo.txt", // typo
	})
	_, err := reg.Execute(context.Background(), "file_tool", params)

	if err == nil {
		t.Fatal("expected error but got none")
	}

	errMsg := err.Error()

	// Error message should contain the unknown parameter name
	if !contains(errMsg, "fliename") {
		t.Errorf("error message should mention unknown parameter 'fliename', got: %s", errMsg)
	}

	// Error message should contain indication of valid parameters
	// It should mention at least some of the valid parameter names
	validParams := []string{"filename", "path", "mode"}
	foundValid := false
	for _, param := range validParams {
		if contains(errMsg, param) {
			foundValid = true
			break
		}
	}

	if !foundValid {
		t.Errorf("error message should list valid parameters, got: %s", errMsg)
	}
}
