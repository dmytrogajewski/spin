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

func TestArgumentParser_Parse(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		parser     *ArgumentParser
		input      string
		want       map[string]any // Keep for comparison via ToMap().
		wantErr    bool
		errContain string
	}{
		{
			name:   "valid_json_arguments",
			parser: &ArgumentParser{AllowEmpty: true},
			input:  `{"path": "/tmp", "recursive": true}`,
			want: map[string]any{
				"path":      "/tmp",
				"recursive": true,
			},
			wantErr: false,
		},
		{
			name:    "empty_json_object",
			parser:  &ArgumentParser{AllowEmpty: true},
			input:   `{}`,
			want:    map[string]any{},
			wantErr: false,
		},
		{
			name:    "empty_string_with_allow_empty",
			parser:  &ArgumentParser{AllowEmpty: true},
			input:   "",
			want:    map[string]any{},
			wantErr: false,
		},
		{
			name:       "empty_string_with_strict_parser",
			parser:     NewStrictArgumentParser(),
			input:      "",
			want:       nil,
			wantErr:    true,
			errContain: "cannot be empty",
		},
		{
			name:       "invalid_json",
			parser:     &ArgumentParser{AllowEmpty: true},
			input:      `{"path": "/tmp"`,
			want:       nil,
			wantErr:    true,
			errContain: "failed to parse",
		},
		{
			name:       "not_json_object",
			parser:     &ArgumentParser{AllowEmpty: true},
			input:      `["not", "an", "object"]`,
			want:       nil,
			wantErr:    true,
			errContain: "failed to parse",
		},
		{
			name:   "complex_nested_json",
			parser: &ArgumentParser{AllowEmpty: true},
			input:  `{"config": {"host": "localhost", "port": 8080}, "enabled": true}`,
			want: map[string]any{
				"config": map[string]any{
					"host": "localhost",
					"port": float64(8080), // JSON numbers unmarshal to float64.
				},
				"enabled": true,
			},
			wantErr: false,
		},
		{
			name:   "json_with_null_values",
			parser: &ArgumentParser{AllowEmpty: true},
			input:  `{"path": null, "count": 0}`,
			want: map[string]any{
				"path":  nil,
				"count": float64(0),
			},
			wantErr: false,
		},
		{
			name:   "json_with_array_value",
			parser: &ArgumentParser{AllowEmpty: true},
			input:  `{"files": ["a.txt", "b.txt"], "verbose": true}`,
			want: map[string]any{
				"files":   []any{"a.txt", "b.txt"},
				"verbose": true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := tt.parser.Parse(tt.input)

			if tt.wantErr {
				require.Error(t, err)

				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}

				assert.Equal(t, ToolParameters{}, got)
			} else {
				require.NoError(t, err)
				// Compare via ToMap() for easier assertion.
				assert.Equal(t, tt.want, got.ToMap())
			}
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
		assert.Error(t, err)

		// Change to allow empty.
		parser.AllowEmpty = true
		result, err := parser.Parse("")
		assert.NoError(t, err)
		assert.Equal(t, ToolParameters{}, result)
	})
}

// Benchmark tests.
func BenchmarkArgumentParser_Parse(b *testing.B) {
	parser := &ArgumentParser{AllowEmpty: true}
	input := `{"path": "/tmp/file.txt", "mode": 0644, "recursive": true}`

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
