package tools

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
)

var (
	errExecutionFailed = errors.New("execution failed")
	errExecutionFailed2 = errors.New("execution failed")
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

	// Should be empty initially.
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

		// Verify tool was registered.
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

		// Register initial tool.
		tool1 := newMockTool("tool1")
		tool1.description = "Original description"
		_ = reg.Register(tool1)

		// Replace with new tool.
		tool1Updated := newMockTool("tool1")
		tool1Updated.description = "Updated description"

		err := reg.RegisterOrReplace(tool1Updated)
		if err != nil {
			t.Fatalf("RegisterOrReplace() unexpected error: %v", err)
		}

		// Verify tool was replaced.
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

		// Register multiple tools.
		_ = reg.Register(newMockTool("tool1"))
		_ = reg.Register(newMockTool("tool2"))
		_ = reg.Register(newMockTool("tool3"))

		// Replace tool2.
		tool2Updated := newMockTool("tool2")
		tool2Updated.description = "Updated tool2"
		_ = reg.RegisterOrReplace(tool2Updated)

		// Verify all tools are still present.
		tools := reg.List()
		if len(tools) != 3 {
			t.Errorf("expected 3 tools, got %d", len(tools))
		}

		// Verify tool1 and tool3 are unchanged.
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

	// Empty list.
	if len(reg.List()) != 0 {
		t.Error("expected empty list")
	}

	// Add tools.
	_ = reg.Register(newMockTool("tool1"))
	_ = reg.Register(newMockTool("tool2"))
	_ = reg.Register(newMockTool("tool3"))

	tools := reg.List()
	if len(tools) != 3 {
		t.Errorf("expected 3 tools, got %d", len(tools))
	}

	// Verify all tools present.
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

	// Empty schemas.
	if len(reg.ListSchemas()) != 0 {
		t.Error("expected empty schemas")
	}

	// Add tools.
	_ = reg.Register(newMockTool("tool1"))
	_ = reg.Register(newMockTool("tool2"))

	schemas := reg.ListSchemas()
	if len(schemas) != 2 {
		t.Errorf("expected 2 schemas, got %d", len(schemas))
	}

	// Verify schema structure.
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

	// Tool that returns success.
	successTool := &mockTool{
		name: "success_tool",
		executeFunc: func(_ context.Context, _ ToolParameters) (ToolResult, error) {
			return ToolResult{
				Success: true,
				Output:  "success output",
			}, nil
		},
	}

	// Tool that returns error.
	errorTool := &mockTool{
		name: "error_tool",
		executeFunc: func(_ context.Context, _ ToolParameters) (ToolResult, error) {
			return ToolResult{}, errExecutionFailed
		},
	}

	// Tool that checks parameters.
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
		params     map[string]any
		wantErr    error
		wantResult ToolResult
	}{
		{
			name:     "execute success tool",
			toolName: "success_tool",
			params:   map[string]any{},
			wantResult: ToolResult{
				Success: true,
				Output:  "success output",
			},
		},
		{
			name:     "execute error tool",
			toolName: "error_tool",
			params:   map[string]any{},
			wantErr:  errExecutionFailed2,
		},
		{
			name:     "execute non-existent tool",
			toolName: "nonexistent",
			params:   map[string]any{},
			wantErr:  ErrToolNotFound,
		},
		{
			name:     "execute with valid required params",
			toolName: "param_tool",
			params: map[string]any{
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
			params: map[string]any{
				"optional_param": "test",
			},
			wantErr: ErrInvalidParameters,
		},
		{
			name:     "execute with wrong param type",
			toolName: "param_tool",
			params: map[string]any{
				"required_param": 123, // should be string.
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
				// Check if error matches expected (wrapped or direct).
				if !errors.Is(err, tt.wantErr) {
					// Also check if the wanted error message is contained.
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

	// Tool that checks context.
	ctxTool := &mockTool{
		name: "ctx_tool",
		executeFunc: func(ctx context.Context, _ ToolParameters) (ToolResult, error) {
			select {
			case <-ctx.Done():
				return ToolResult{}, ctx.Err()
			default:
				return ToolResult{Success: true, Output: "completed"}, nil
			}
		},
	}

	_ = reg.Register(ctxTool)

	// Test with canceled context.
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	params, _ := FromMap(map[string]any{})

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

	// Register initial tools.
	for i := range 5 {
		_ = reg.Register(newMockTool(fmt.Sprintf("tool-%d", i)))
	}

	// Concurrent reads and writes.
	var wg sync.WaitGroup

	// Concurrent reads.
	for range 10 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			_ = reg.List()
			_ = reg.ListSchemas()
			_, _ = reg.Get("tool1")
		}()
	}

	// Concurrent executions.
	for range 10 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			params, _ := FromMap(map[string]any{"param1": "test"})
			_, _ = reg.Execute(context.Background(), "tool1", params)
		}()
	}

	// Concurrent registrations (should fail on duplicates, but shouldn't panic).
	for i := range 5 {
		wg.Add(1)

		go func(idx int) {
			defer wg.Done()

			_ = reg.Register(newMockTool(fmt.Sprintf("concurrent-%d", idx)))
		}(i)
	}

	wg.Wait()

	// Verify registry is still functional.
	tools := reg.List()
	if len(tools) < 5 {
		t.Errorf("expected at least 5 tools after concurrent operations, got %d", len(tools))
	}
}

func TestRegistryTypeValidation(t *testing.T) {
	reg := NewRegistry()

	// Tool with various parameter types.
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
		executeFunc: func(_ context.Context, _ ToolParameters) (ToolResult, error) {
			return ToolResult{Success: true, Output: "ok"}, nil
		},
	}

	_ = reg.Register(typeTool)

	tests := []struct {
		name    string
		params  map[string]any
		wantErr bool
	}{
		{
			name: "valid string",
			params: map[string]any{
				"str_param": "test",
			},
		},
		{
			name: "valid integer",
			params: map[string]any{
				"int_param": 42,
			},
		},
		{
			name: "valid number - int",
			params: map[string]any{
				"num_param": 42,
			},
		},
		{
			name: "valid number - float",
			params: map[string]any{
				"num_param": 3.14,
			},
		},
		{
			name: "valid boolean",
			params: map[string]any{
				"bool_param": true,
			},
		},
		{
			name: "valid array",
			params: map[string]any{
				"array_param": []string{"a", "b", "c"},
			},
		},
		{
			name: "valid object",
			params: map[string]any{
				"obj_param": map[string]any{"key": "value"},
			},
		},
		{
			name: "invalid string type",
			params: map[string]any{
				"str_param": 123,
			},
			wantErr: true,
		},
		{
			name: "invalid integer type",
			params: map[string]any{
				"int_param": "not an int",
			},
			wantErr: true,
		},
		{
			name: "invalid boolean type",
			params: map[string]any{
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

	// Tool with enum parameter.
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
		executeFunc: func(_ context.Context, _ ToolParameters) (ToolResult, error) {
			return ToolResult{Success: true, Output: "ok"}, nil
		},
	}

	_ = reg.Register(enumTool)

	tests := []struct {
		name    string
		params  map[string]any
		wantErr bool
	}{
		{
			name: "valid enum value - read",
			params: map[string]any{
				"mode": "read",
			},
		},
		{
			name: "valid enum value - write",
			params: map[string]any{
				"mode": "write",
			},
		},
		{
			name: "invalid enum value",
			params: map[string]any{
				"mode": "delete",
			},
			wantErr: true,
		},
		{
			name: "non-string enum value",
			params: map[string]any{
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

	// Tool with defined parameters.
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
					Required: []string{}, // No required params.
				},
			},
		},
		executeFunc: func(_ context.Context, _ ToolParameters) (ToolResult, error) {
			return ToolResult{Success: true, Output: "ok"}, nil
		},
	}

	_ = reg.Register(tool)

	tests := []struct {
		name    string
		params  map[string]any
		wantErr bool
		errMsg  string
	}{
		{
			name: "all known parameters",
			params: map[string]any{
				"param1": "value1",
				"param2": "value2",
			},
			wantErr: false,
		},
		{
			name: "subset of known parameters",
			params: map[string]any{
				"param1": "value1",
			},
			wantErr: false,
		},
		{
			name:    "empty parameters",
			params:  map[string]any{},
			wantErr: false,
		},
		{
			name: "single unknown parameter",
			params: map[string]any{
				"unknown_param": "value",
			},
			wantErr: true,
			errMsg:  "unknown parameter \"unknown_param\"",
		},
		{
			name: "known and unknown parameters",
			params: map[string]any{
				"param1":        "value1",
				"unknown_param": "value",
			},
			wantErr: true,
			errMsg:  "unknown parameter \"unknown_param\"",
		},
		{
			name: "typo in parameter name",
			params: map[string]any{
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

	// Tool with multiple defined parameters.
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
		executeFunc: func(_ context.Context, _ ToolParameters) (ToolResult, error) {
			return ToolResult{Success: true, Output: "ok"}, nil
		},
	}

	_ = reg.Register(tool)

	// Execute with typo in parameter name.
	params, _ := FromMap(map[string]any{
		"filename": "test.txt",
		"fliename": "typo.txt", // typo.
	})

	_, err := reg.Execute(context.Background(), "file_tool", params)
	if err == nil {
		t.Fatal("expected error but got none")
	}

	errMsg := err.Error()

	// Error message should contain the unknown parameter name.
	if !contains(errMsg, "fliename") {
		t.Errorf("error message should mention unknown parameter 'fliename', got: %s", errMsg)
	}

	// Error message should contain indication of valid parameters
	// It should mention at least some of the valid parameter names.
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

func TestNewDefaultRegistry(t *testing.T) {
	workDir := "/tmp"
	// Use a simple struct that implements the interface GetContextTool expects.
	type testEnv struct {
		WorkDir string
	}

	env := &testEnv{WorkDir: workDir}
	registry := NewDefaultRegistry(workDir, env)

	// Verify all tools are registered.
	expectedTools := []string{
		"read_file", "write_file", "list_directory",
		"shell_command", "get_context", "apply_patch",
		"file_search", "git_context",
	}

	for _, toolName := range expectedTools {
		tool, err := registry.Get(toolName)
		if err != nil {
			t.Errorf("tool %s should be registered: %v", toolName, err)

			continue
		}

		if tool == nil {
			t.Errorf("tool %s should not be nil", toolName)
		}
	}
}

func TestNewDefaultRegistry_ToolsConfigured(t *testing.T) {
	workDir := "/tmp/workdir"

	type testEnv struct {
		WorkDir string
	}

	env := &testEnv{WorkDir: workDir}
	registry := NewDefaultRegistry(workDir, env)

	// Verify tools that need WorkDir are configured correctly
	// Note: We can't directly access internal fields, so we verify
	// by ensuring tools can be retrieved and are not nil.
	patchTool, err := registry.Get("apply_patch")
	if err != nil {
		t.Fatalf("apply_patch tool should be registered: %v", err)
	}

	if patchTool == nil {
		t.Error("apply_patch tool should not be nil")
	}

	searchTool, err := registry.Get("file_search")
	if err != nil {
		t.Fatalf("file_search tool should be registered: %v", err)
	}

	if searchTool == nil {
		t.Error("file_search tool should not be nil")
	}

	gitTool, err := registry.Get("git_context")
	if err != nil {
		t.Fatalf("git_context tool should be registered: %v", err)
	}

	if gitTool == nil {
		t.Error("git_context tool should not be nil")
	}

	// Verify get_context is registered with environment.
	contextTool, err := registry.Get("get_context")
	if err != nil {
		t.Fatalf("get_context tool should be registered: %v", err)
	}

	if contextTool == nil {
		t.Error("get_context tool should not be nil")
	}
}

func TestNewDefaultRegistry_NilEnvironment(t *testing.T) {
	// Should handle nil gracefully - create tools with empty WorkDir.
	registry := NewDefaultRegistry("", nil)
	if registry == nil {
		t.Fatal("NewDefaultRegistry should not return nil")
	}

	// Verify all tools are still registered.
	expectedTools := []string{
		"read_file", "write_file", "list_directory",
		"shell_command", "get_context", "apply_patch",
		"file_search", "git_context",
	}

	for _, toolName := range expectedTools {
		tool, err := registry.Get(toolName)
		if err != nil {
			t.Errorf("tool %s should be registered even with nil env: %v", toolName, err)

			continue
		}

		if tool == nil {
			t.Errorf("tool %s should not be nil", toolName)
		}
	}
}

func TestNewDefaultRegistry_EquivalentToManual(t *testing.T) {
	workDir := "/tmp/test"

	type testEnv struct {
		WorkDir string
	}

	env := &testEnv{WorkDir: workDir}

	// Manual construction (old way).
	manual := NewRegistry()
	_ = manual.Register(NewReadFileTool())
	_ = manual.Register(NewWriteFileTool())
	_ = manual.Register(NewListDirectoryTool())
	_ = manual.Register(NewShellCommandTool(nil, nil, nil))
	_ = manual.Register(NewGetContextTool(env))
	_ = manual.Register(NewApplyPatchTool(workDir))
	_ = manual.Register(NewFileSearchTool(workDir))
	_ = manual.Register(NewGitContextTool(workDir))

	// Factory construction (new way).
	factory := NewDefaultRegistry(workDir, env)

	// Verify both registries have same tools.
	manualTools := manual.List()
	factoryTools := factory.List()

	if len(manualTools) != len(factoryTools) {
		t.Errorf("tool counts should match: manual=%d, factory=%d", len(manualTools), len(factoryTools))
	}

	// Verify each tool exists in both.
	manualToolMap := make(map[string]Tool)
	for _, tool := range manualTools {
		manualToolMap[tool.Name()] = tool
	}

	factoryToolMap := make(map[string]Tool)
	for _, tool := range factoryTools {
		factoryToolMap[tool.Name()] = tool
	}

	// Verify all tools are present in both.
	for toolName := range manualToolMap {
		if _, exists := factoryToolMap[toolName]; !exists {
			t.Errorf("factory should contain tool %s", toolName)
		}
	}

	for toolName := range factoryToolMap {
		if _, exists := manualToolMap[toolName]; !exists {
			t.Errorf("manual should contain tool %s", toolName)
		}
	}
}

func TestNewDefaultRegistry_AllToolsRegistered(t *testing.T) {
	workDir := "/tmp"

	type testEnv struct {
		WorkDir string
	}

	env := &testEnv{WorkDir: workDir}
	registry := NewDefaultRegistry(workDir, env)

	// Verify we have exactly 8 tools (matching BuiltinTools count).
	tools := registry.List()
	if len(tools) != 8 {
		t.Errorf("should have exactly 8 builtin tools, got %d", len(tools))
	}

	// Verify each tool has valid name, description, and schema.
	for _, tool := range tools {
		if tool.Name() == "" {
			t.Error("tool should have non-empty name")
		}

		if tool.Description() == "" {
			t.Errorf("tool %s should have description", tool.Name())
		}

		schema := tool.Schema()
		if schema.Type != "function" {
			t.Errorf("tool %s should have type 'function', got %s", tool.Name(), schema.Type)
		}

		if schema.Function.Name == "" {
			t.Errorf("tool %s should have function name", tool.Name())
		}

		if schema.Function.Description == "" {
			t.Errorf("tool %s should have function description", tool.Name())
		}
	}
}

func TestNewDefaultRegistry_UniqueToolNames(t *testing.T) {
	workDir := "/tmp"

	type testEnv struct {
		WorkDir string
	}

	env := &testEnv{WorkDir: workDir}
	registry := NewDefaultRegistry(workDir, env)

	// Verify all tool names are unique.
	tools := registry.List()
	toolNames := make(map[string]bool)

	for _, tool := range tools {
		name := tool.Name()
		if toolNames[name] {
			t.Errorf("duplicate tool name: %s", name)
		}

		toolNames[name] = true
	}
}
