package tools

import (
	"encoding/json"
	"errors"
	"testing"
)

var (
	errTestError          = errors.New("test error")
	errSomethingWentWrong = errors.New("something went wrong")
	errOperationFailed    = errors.New("operation failed")
	errFail               = errors.New("fail")
	errSomethingBroke     = errors.New("something broke")
)

// TestBuiltinTools_Count verifies that we have exactly 8 builtin tools.
func TestBuiltinTools_Count(t *testing.T) {
	t.Parallel()
	if len(BuiltinTools) != 8 {
		t.Errorf("BuiltinTools count = %d, want 8", len(BuiltinTools))
	}
}

// TestBuiltinTools_Names verifies that all expected builtin tools are present.
func TestBuiltinTools_Names(t *testing.T) {
	t.Parallel()
	expected := map[string]bool{
		"read_file":      false,
		"write_file":     false,
		"list_directory": false,
		"shell_command":  false,
		"get_context":    false,
		"apply_patch":    false,
		"file_search":    false,
		"git_context":    false,
	}

	for _, tool := range BuiltinTools {
		name := tool.Name()
		if _, exists := expected[name]; !exists {
			t.Errorf("unexpected builtin tool: %s", name)

			continue
		}

		expected[name] = true
	}

	for name, found := range expected {
		if !found {
			t.Errorf("missing builtin tool: %s", name)
		}
	}
}

// TestBuiltinTools_NonNil verifies that all builtin tools are non-nil.
func TestBuiltinTools_NonNil(t *testing.T) {
	t.Parallel()
	for i, tool := range BuiltinTools {
		if tool == nil {
			t.Errorf("BuiltinTools[%d] is nil", i)
		}
	}
}

// TestToolResult_ID verifies that ToolResult has ID field.
func TestToolResult_ID(t *testing.T) {
	t.Parallel()
	const testID = "call_abc123"

	result := ToolResult{
		ID:      testID,
		Success: true,
		Output:  "test output",
	}

	if result.ID != testID {
		t.Errorf("ToolResult.ID = %q, want %q", result.ID, testID)
	}

	if !result.Success {
		t.Error("ToolResult.Success = false, want true")
	}

	if result.Output != "test output" {
		t.Errorf("ToolResult.Output = %q, want %q", result.Output, "test output")
	}
}

// TestToolResult_ExitCode verifies that ToolResult has ExitCode field.
func TestToolResult_ExitCode(t *testing.T) {
	t.Parallel()
	const testExitCode = 1

	result := ToolResult{
		Success:  false,
		ExitCode: testExitCode,
	}

	if result.Success {
		t.Error("ToolResult.Success = true, want false")
	}

	if result.ExitCode != testExitCode {
		t.Errorf("ToolResult.ExitCode = %d, want %d", result.ExitCode, testExitCode)
	}
}

// TestToolResult_ErrorAsError verifies that ToolResult has Err field of error type.
func TestToolResult_ErrorAsError(t *testing.T) {
	t.Parallel()
	testErr := errTestError
	result := ToolResult{
		Success: false,
		Err:     testErr,
	}

	if result.Success {
		t.Error("ToolResult.Success = true, want false")
	}

	if !errors.Is(result.Err, testErr) {
		t.Errorf("ToolResult.Err = %v, want %v", result.Err, testErr)
	}
}

// TestNewToolResult creates a success result with output.
func TestNewToolResult(t *testing.T) {
	t.Parallel()
	const testOutput = "test output"

	result := NewToolResult(testOutput)

	if !result.Success {
		t.Error("NewToolResult().Success = false, want true")
	}

	if result.Output != testOutput {
		t.Errorf("NewToolResult().Output = %q, want %q", result.Output, testOutput)
	}
}

// TestNewToolError creates a failed result from error.
func TestNewToolError(t *testing.T) {
	t.Parallel()
	testErr := errSomethingWentWrong
	result := NewToolError(testErr)

	if result.Success {
		t.Error("NewToolError().Success = true, want false")
	}

	if !errors.Is(result.Err, testErr) {
		t.Errorf("NewToolError().Err = %v, want %v", result.Err, testErr)
	}

	if result.Error != testErr.Error() {
		t.Errorf("NewToolError().Error = %q, want %q", result.Error, testErr.Error())
	}
}

// TestNewToolErrorWithID creates a failed result with ID from error.
func TestNewToolErrorWithID(t *testing.T) {
	t.Parallel()
	const testID = "call_err123"

	testErr := errOperationFailed
	result := NewToolErrorWithID(testID, testErr)

	if result.ID != testID {
		t.Errorf("NewToolErrorWithID().ID = %q, want %q", result.ID, testID)
	}

	if result.Success {
		t.Error("NewToolErrorWithID().Success = true, want false")
	}

	if !errors.Is(result.Err, testErr) {
		t.Errorf("NewToolErrorWithID().Err = %v, want %v", result.Err, testErr)
	}
}

// TestToolResult_WithID returns copy with ID set.
func TestToolResult_WithID(t *testing.T) {
	t.Parallel()
	const testID = "call_new456"

	original := ToolResult{Success: true, Output: "test"}
	result := original.WithID(testID)

	if result.ID != testID {
		t.Errorf("WithID().ID = %q, want %q", result.ID, testID)
	}
	// Verify original is unchanged.
	if original.ID != "" {
		t.Errorf("original.ID = %q, want empty", original.ID)
	}
}

// TestToolResult_WithExitCode returns copy with exit code set.
func TestToolResult_WithExitCode(t *testing.T) {
	t.Parallel()
	const testCode = 42

	original := ToolResult{Success: false}
	result := original.WithExitCode(testCode)

	if result.ExitCode != testCode {
		t.Errorf("WithExitCode().ExitCode = %d, want %d", result.ExitCode, testCode)
	}
	// Verify original is unchanged.
	if original.ExitCode != 0 {
		t.Errorf("original.ExitCode = %d, want 0", original.ExitCode)
	}
}

// TestToolResult_WithMetadata returns copy with metadata set.
func TestToolResult_WithMetadata(t *testing.T) {
	t.Parallel()
	testMeta := map[string]any{"key": "value"}
	original := ToolResult{Success: true}
	result := original.WithMetadata(testMeta)

	if result.Metadata["key"] != "value" {
		t.Errorf("WithMetadata().Metadata[key] = %v, want value", result.Metadata["key"])
	}
	// Verify original is unchanged.
	if original.Metadata != nil {
		t.Error("original.Metadata should be nil")
	}
}

// TestToolResult_GetErr returns error or nil.
func TestToolResult_GetErr(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		result  ToolResult
		wantNil bool
	}{
		{
			name:    "no error",
			result:  ToolResult{Success: true},
			wantNil: true,
		},
		{
			name:    "with error",
			result:  ToolResult{Success: false, Err: errFail},
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.result.GetErr()
			if tt.wantNil && err != nil {
				t.Errorf("GetErr() = %v, want nil", err)
			}

			if !tt.wantNil && err == nil {
				t.Error("GetErr() = nil, want error")
			}
		})
	}
}

// TestToolResult_String returns appropriate content.
func TestToolResult_String(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		result ToolResult
		want   string
	}{
		{
			name:   "success returns output",
			result: ToolResult{Success: true, Output: "hello world"},
			want:   "hello world",
		},
		{
			name:   "failure with Err returns error string",
			result: ToolResult{Success: false, Err: errSomethingBroke},
			want:   "something broke",
		},
		{
			name:   "failure with Error string returns error string",
			result: ToolResult{Success: false, Error: "legacy error"},
			want:   "legacy error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.result.String()
			if got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

// marshalToolResult marshals a ToolResult to JSON and back to a map for verification.
func marshalToolResult(t *testing.T, result ToolResult) map[string]any {
	t.Helper()

	jsonBytes, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var data map[string]any
	if err = json.Unmarshal(jsonBytes, &data); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	return data
}

// TestToolResult_JSONSerialization verifies JSON marshaling.
func TestToolResult_JSONSerialization(t *testing.T) {
	t.Parallel()

	t.Run("success result", func(t *testing.T) {
		t.Parallel()

		data := marshalToolResult(t, ToolResult{ID: "call_123", Success: true, Output: "result data"})
		assertJSONField(t, data, "id", "call_123")
		assertJSONField(t, data, "output", "result data")

		if successVal, _ := data["success"].(bool); !successVal {
			t.Errorf("JSON success = %v, want true", data["success"])
		}
	})

	t.Run("error result", func(t *testing.T) {
		t.Parallel()

		data := marshalToolResult(t, ToolResult{Success: false, Error: "operation failed"})
		assertJSONField(t, data, "error", "operation failed")
	})

	t.Run("with exit code", func(t *testing.T) {
		t.Parallel()

		data := marshalToolResult(t, ToolResult{Success: false, ExitCode: 1})
		if code, ok := data["exit_code"]; ok && code != float64(1) {
			t.Errorf("JSON exit_code = %v, want 1", code)
		}
	})
}

// assertJSONField checks that a JSON field has the expected string value.
func assertJSONField(t *testing.T, data map[string]any, key string, want string) {
	t.Helper()

	if data[key] != want {
		t.Errorf("JSON %s = %v, want %s", key, data[key], want)
	}
}
