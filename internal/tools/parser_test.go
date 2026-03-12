package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewArgumentParser(t *testing.T) {
	t.Parallel()
	parser := &ArgumentParser{AllowEmpty: true}
	assert.NotNil(t, parser)
	assert.True(t, parser.AllowEmpty, "default parser should allow empty arguments")
}

func TestNewStrictArgumentParser(t *testing.T) {
	t.Parallel()
	parser := NewStrictArgumentParser()
	assert.NotNil(t, parser)
	assert.False(t, parser.AllowEmpty, "strict parser should not allow empty arguments")
}

func TestArgumentParser_Parse_Valid(t *testing.T) {
	t.Parallel()

	parser := &ArgumentParser{AllowEmpty: true}

	tests := []struct {
		name  string
		input string
		want  map[string]any
	}{
		{name: "valid_json_arguments", input: `{"path": "/tmp", "recursive": true}`, want: map[string]any{"path": "/tmp", "recursive": true}},
		{name: "empty_json_object", input: `{}`, want: map[string]any{}},
		{name: "empty_string_with_allow_empty", input: "", want: map[string]any{}},
		{name: "complex_nested_json", input: `{"config": {"host": "localhost", "port": 8080}, "enabled": true}`, want: map[string]any{"config": map[string]any{"host": "localhost", "port": float64(8080)}, "enabled": true}},
		{name: "json_with_null_values", input: `{"path": null, "count": 0}`, want: map[string]any{"path": nil, "count": float64(0)}},
		{name: "json_with_array_value", input: `{"files": ["a.txt", "b.txt"], "verbose": true}`, want: map[string]any{"files": []any{"a.txt", "b.txt"}, "verbose": true}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := parser.Parse(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got.ToMap())
		})
	}
}

func TestArgumentParser_Parse_Errors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		parser     *ArgumentParser
		input      string
		errContain string
	}{
		{name: "empty_string_with_strict_parser", parser: NewStrictArgumentParser(), input: "", errContain: "cannot be empty"},
		{name: "invalid_json", parser: &ArgumentParser{AllowEmpty: true}, input: `{"path": "/tmp"`, errContain: "failed to parse"},
		{name: "not_json_object", parser: &ArgumentParser{AllowEmpty: true}, input: `["not", "an", "object"]`, errContain: "failed to parse"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := tt.parser.Parse(tt.input)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errContain)
			assert.Equal(t, ToolParameters{}, got)
		})
	}
}

func TestArgumentParser_CustomConfiguration(t *testing.T) {
	t.Parallel()
	t.Run("can_modify_allow_empty", func(t *testing.T) {
		t.Parallel()
		parser := &ArgumentParser{AllowEmpty: false}

		// Should error on empty.
		_, err := parser.Parse("")
		require.Error(t, err)

		// Change to allow empty.
		parser.AllowEmpty = true
		result, err := parser.Parse("")
		require.NoError(t, err)
		assert.Equal(t, ToolParameters{}, result)
	})
}

// Benchmark tests.
func BenchmarkArgumentParser_Parse(b *testing.B) {
	parser := &ArgumentParser{AllowEmpty: true}
	input := `{"path": "/tmp/file.txt", "mode": 0o600, "recursive": true}`

	b.ResetTimer()

	for range b.N {
		_, _ = parser.Parse(input)
	}
}

func BenchmarkArgumentParser_ParseEmpty(b *testing.B) {
	parser := &ArgumentParser{AllowEmpty: true}

	b.ResetTimer()

	for range b.N {
		_, _ = parser.Parse("")
	}
}
