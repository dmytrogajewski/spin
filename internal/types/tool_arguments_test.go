package types

import (
	"encoding/json"
	"testing"
)

func TestToolCallArguments_Get(t *testing.T) {
	tests := []struct {
		name      string
		args      ToolCallArguments
		key       string
		dest      any
		wantError bool
		wantValue any
	}{
		{
			name: "get string value",
			args: ToolCallArguments{
				"name": json.RawMessage(`"test"`),
			},
			key:       "name",
			dest:      &[]string{""}[0],
			wantError: false,
			wantValue: "test",
		},
		{
			name: "get int value",
			args: ToolCallArguments{
				"count": json.RawMessage(`42`),
			},
			key:       "count",
			dest:      &[]int{0}[0],
			wantError: false,
			wantValue: 42,
		},
		{
			name: "get bool value",
			args: ToolCallArguments{
				"enabled": json.RawMessage(`true`),
			},
			key:       "enabled",
			dest:      &[]bool{false}[0],
			wantError: false,
			wantValue: true,
		},
		{
			name: "parameter not found",
			args: ToolCallArguments{
				"other": json.RawMessage(`"value"`),
			},
			key:       "missing",
			dest:      &[]string{""}[0],
			wantError: true,
		},
		{
			name: "invalid JSON",
			args: ToolCallArguments{
				"invalid": json.RawMessage(`{invalid json}`),
			},
			key:       "invalid",
			dest:      &[]string{""}[0],
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.args.Get(tt.key, tt.dest)
			if (err != nil) != tt.wantError {
				t.Errorf("ToolCallArguments.Get() error = %v, wantError %v", err, tt.wantError)
			}
			if !tt.wantError {
				// Check if the value was set correctly
				switch v := tt.dest.(type) {
				case *string:
					if *v != tt.wantValue {
						t.Errorf("ToolCallArguments.Get() string value = %v, want %v", *v, tt.wantValue)
					}
				case *int:
					if *v != tt.wantValue {
						t.Errorf("ToolCallArguments.Get() int value = %v, want %v", *v, tt.wantValue)
					}
				case *bool:
					if *v != tt.wantValue {
						t.Errorf("ToolCallArguments.Get() bool value = %v, want %v", *v, tt.wantValue)
					}
				}
			}
		})
	}
}

func TestToolCallArguments_GetString(t *testing.T) {
	tests := []struct {
		name      string
		args      ToolCallArguments
		key       string
		wantValue string
		wantError bool
	}{
		{
			name: "get string value",
			args: ToolCallArguments{
				"name": json.RawMessage(`"test"`),
			},
			key:       "name",
			wantValue: "test",
			wantError: false,
		},
		{
			name: "parameter not found",
			args: ToolCallArguments{
				"other": json.RawMessage(`"value"`),
			},
			key:       "missing",
			wantValue: "",
			wantError: true,
		},
		{
			name: "invalid JSON",
			args: ToolCallArguments{
				"invalid": json.RawMessage(`{invalid json}`),
			},
			key:       "invalid",
			wantValue: "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.args.GetString(tt.key)
			if (err != nil) != tt.wantError {
				t.Errorf("ToolCallArguments.GetString() error = %v, wantError %v", err, tt.wantError)
			}
			if result != tt.wantValue {
				t.Errorf("ToolCallArguments.GetString() = %v, want %v", result, tt.wantValue)
			}
		})
	}
}

func TestToolCallArguments_GetInt(t *testing.T) {
	tests := []struct {
		name      string
		args      ToolCallArguments
		key       string
		wantValue int
		wantError bool
	}{
		{
			name: "get int value",
			args: ToolCallArguments{
				"count": json.RawMessage(`42`),
			},
			key:       "count",
			wantValue: 42,
			wantError: false,
		},
		{
			name: "parameter not found",
			args: ToolCallArguments{
				"other": json.RawMessage(`"value"`),
			},
			key:       "missing",
			wantValue: 0,
			wantError: true,
		},
		{
			name: "invalid JSON",
			args: ToolCallArguments{
				"invalid": json.RawMessage(`"not a number"`),
			},
			key:       "invalid",
			wantValue: 0,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.args.GetInt(tt.key)
			if (err != nil) != tt.wantError {
				t.Errorf("ToolCallArguments.GetInt() error = %v, wantError %v", err, tt.wantError)
			}
			if result != tt.wantValue {
				t.Errorf("ToolCallArguments.GetInt() = %v, want %v", result, tt.wantValue)
			}
		})
	}
}

func TestToolCallArguments_GetBool(t *testing.T) {
	tests := []struct {
		name      string
		args      ToolCallArguments
		key       string
		wantValue bool
		wantError bool
	}{
		{
			name: "get bool value true",
			args: ToolCallArguments{
				"enabled": json.RawMessage(`true`),
			},
			key:       "enabled",
			wantValue: true,
			wantError: false,
		},
		{
			name: "get bool value false",
			args: ToolCallArguments{
				"disabled": json.RawMessage(`false`),
			},
			key:       "disabled",
			wantValue: false,
			wantError: false,
		},
		{
			name: "parameter not found",
			args: ToolCallArguments{
				"other": json.RawMessage(`"value"`),
			},
			key:       "missing",
			wantValue: false,
			wantError: true,
		},
		{
			name: "invalid JSON",
			args: ToolCallArguments{
				"invalid": json.RawMessage(`"not a bool"`),
			},
			key:       "invalid",
			wantValue: false,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.args.GetBool(tt.key)
			if (err != nil) != tt.wantError {
				t.Errorf("ToolCallArguments.GetBool() error = %v, wantError %v", err, tt.wantError)
			}
			if result != tt.wantValue {
				t.Errorf("ToolCallArguments.GetBool() = %v, want %v", result, tt.wantValue)
			}
		})
	}
}

func TestToolCallArguments_ToMap(t *testing.T) {
	tests := []struct {
		name     string
		args     ToolCallArguments
		expected map[string]any
	}{
		{
			name: "convert to map",
			args: ToolCallArguments{
				"name":    json.RawMessage(`"test"`),
				"count":   json.RawMessage(`42`),
				"enabled": json.RawMessage(`true`),
			},
			expected: map[string]any{
				"name":    "test",
				"count":   float64(42), // JSON numbers are float64
				"enabled": true,
			},
		},
		{
			name:     "empty arguments",
			args:     ToolCallArguments{},
			expected: map[string]any{},
		},
		{
			name: "with invalid JSON",
			args: ToolCallArguments{
				"valid":   json.RawMessage(`"test"`),
				"invalid": json.RawMessage(`{invalid json}`),
			},
			expected: map[string]any{
				"valid": "test",
				// invalid JSON should be skipped
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.args.ToMap()

			// Check length
			if len(result) != len(tt.expected) {
				t.Errorf("ToolCallArguments.ToMap() length = %v, want %v", len(result), len(tt.expected))
			}

			// Check values
			for k, v := range tt.expected {
				if result[k] != v {
					t.Errorf("ToolCallArguments.ToMap()[%s] = %v, want %v", k, result[k], v)
				}
			}
		})
	}
}

func TestFromMap(t *testing.T) {
	tests := []struct {
		name      string
		input     map[string]any
		wantError bool
	}{
		{
			name: "convert from map",
			input: map[string]any{
				"name":    "test",
				"count":   42,
				"enabled": true,
			},
			wantError: false,
		},
		{
			name:      "empty map",
			input:     map[string]any{},
			wantError: false,
		},
		{
			name: "complex types",
			input: map[string]any{
				"list":   []string{"a", "b", "c"},
				"nested": map[string]any{"key": "value"},
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := FromMap(tt.input)
			if (err != nil) != tt.wantError {
				t.Errorf("FromMap() error = %v, wantError %v", err, tt.wantError)
			}
			if !tt.wantError {
				if len(result) != len(tt.input) {
					t.Errorf("FromMap() length = %v, want %v", len(result), len(tt.input))
				}

				// Verify we can retrieve values
				for k := range tt.input {
					var dest any
					err := result.Get(k, &dest)
					if err != nil {
						t.Errorf("FromMap() result.Get(%s) error = %v", k, err)
					}
				}
			}
		})
	}
}

func TestErrParameterNotFound(t *testing.T) {
	if ErrParameterNotFound == nil {
		t.Error("ErrParameterNotFound should not be nil")
	}

	if ErrParameterNotFound.Error() != "parameter not found" {
		t.Errorf("ErrParameterNotFound.Error() = %v, want %v", ErrParameterNotFound.Error(), "parameter not found")
	}
}

func TestToolCallArguments_Concurrency(t *testing.T) {
	args := ToolCallArguments{
		"name":    json.RawMessage(`"test"`),
		"count":   json.RawMessage(`42`),
		"enabled": json.RawMessage(`true`),
	}

	// Test concurrent access
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			_, _ = args.GetString("name")
			_, _ = args.GetInt("count")
			_, _ = args.GetBool("enabled")
			_ = args.ToMap()
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}
