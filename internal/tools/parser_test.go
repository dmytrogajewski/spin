package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewArgumentParser(t *testing.T) {
	parser := NewArgumentParser()
	assert.NotNil(t, parser)
	assert.True(t, parser.AllowEmpty, "default parser should allow empty arguments")
}

func TestNewStrictArgumentParser(t *testing.T) {
	parser := NewStrictArgumentParser()
	assert.NotNil(t, parser)
	assert.False(t, parser.AllowEmpty, "strict parser should not allow empty arguments")
}

func TestArgumentParser_Parse(t *testing.T) {
	tests := []struct {
		name       string
		parser     *ArgumentParser
		input      string
		want       map[string]interface{}
		wantErr    bool
		errContain string
	}{
		{
			name:   "valid_json_arguments",
			parser: NewArgumentParser(),
			input:  `{"path": "/tmp", "recursive": true}`,
			want: map[string]interface{}{
				"path":      "/tmp",
				"recursive": true,
			},
			wantErr: false,
		},
		{
			name:    "empty_json_object",
			parser:  NewArgumentParser(),
			input:   `{}`,
			want:    map[string]interface{}{},
			wantErr: false,
		},
		{
			name:    "empty_string_with_allow_empty",
			parser:  NewArgumentParser(),
			input:   "",
			want:    map[string]interface{}{},
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
			parser:     NewArgumentParser(),
			input:      `{"path": "/tmp"`,
			want:       nil,
			wantErr:    true,
			errContain: "failed to parse",
		},
		{
			name:       "not_json_object",
			parser:     NewArgumentParser(),
			input:      `["not", "an", "object"]`,
			want:       nil,
			wantErr:    true,
			errContain: "failed to parse",
		},
		{
			name:   "complex_nested_json",
			parser: NewArgumentParser(),
			input:  `{"config": {"host": "localhost", "port": 8080}, "enabled": true}`,
			want: map[string]interface{}{
				"config": map[string]interface{}{
					"host": "localhost",
					"port": float64(8080), // JSON numbers unmarshal to float64
				},
				"enabled": true,
			},
			wantErr: false,
		},
		{
			name:   "json_with_null_values",
			parser: NewArgumentParser(),
			input:  `{"path": null, "count": 0}`,
			want: map[string]interface{}{
				"path":  nil,
				"count": float64(0),
			},
			wantErr: false,
		},
		{
			name:   "json_with_array_value",
			parser: NewArgumentParser(),
			input:  `{"files": ["a.txt", "b.txt"], "verbose": true}`,
			want: map[string]interface{}{
				"files":   []interface{}{"a.txt", "b.txt"},
				"verbose": true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.parser.Parse(tt.input)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				assert.Nil(t, got)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestArgumentParser_ParseBytes(t *testing.T) {
	tests := []struct {
		name       string
		parser     *ArgumentParser
		input      []byte
		want       map[string]interface{}
		wantErr    bool
		errContain string
	}{
		{
			name:   "valid_json_bytes",
			parser: NewArgumentParser(),
			input:  []byte(`{"key": "value"}`),
			want: map[string]interface{}{
				"key": "value",
			},
			wantErr: false,
		},
		{
			name:    "empty_bytes_with_allow_empty",
			parser:  NewArgumentParser(),
			input:   []byte{},
			want:    map[string]interface{}{},
			wantErr: false,
		},
		{
			name:    "nil_bytes_with_allow_empty",
			parser:  NewArgumentParser(),
			input:   nil,
			want:    map[string]interface{}{},
			wantErr: false,
		},
		{
			name:       "empty_bytes_with_strict_parser",
			parser:     NewStrictArgumentParser(),
			input:      []byte{},
			want:       nil,
			wantErr:    true,
			errContain: "cannot be empty",
		},
		{
			name:       "invalid_json_bytes",
			parser:     NewArgumentParser(),
			input:      []byte(`{"invalid": json}`),
			want:       nil,
			wantErr:    true,
			errContain: "failed to parse",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.parser.ParseBytes(tt.input)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				assert.Nil(t, got)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestArgumentParser_MustParse(t *testing.T) {
	t.Run("valid_json_does_not_panic", func(t *testing.T) {
		parser := NewArgumentParser()
		assert.NotPanics(t, func() {
			result := parser.MustParse(`{"key": "value"}`)
			assert.Equal(t, "value", result["key"])
		})
	})

	t.Run("empty_with_allow_empty_does_not_panic", func(t *testing.T) {
		parser := NewArgumentParser()
		assert.NotPanics(t, func() {
			result := parser.MustParse("")
			assert.Empty(t, result)
		})
	})

	t.Run("invalid_json_panics", func(t *testing.T) {
		parser := NewArgumentParser()
		assert.Panics(t, func() {
			parser.MustParse(`{"invalid"`)
		})
	})

	t.Run("empty_with_strict_parser_panics", func(t *testing.T) {
		parser := NewStrictArgumentParser()
		assert.Panics(t, func() {
			parser.MustParse("")
		})
	})
}

func TestArgumentParser_CustomConfiguration(t *testing.T) {
	t.Run("can_modify_allow_empty", func(t *testing.T) {
		parser := &ArgumentParser{AllowEmpty: false}

		// Should error on empty
		_, err := parser.Parse("")
		assert.Error(t, err)

		// Change to allow empty
		parser.AllowEmpty = true
		result, err := parser.Parse("")
		assert.NoError(t, err)
		assert.Empty(t, result)
	})
}

// Benchmark tests
func BenchmarkArgumentParser_Parse(b *testing.B) {
	parser := NewArgumentParser()
	input := `{"path": "/tmp/file.txt", "mode": 0644, "recursive": true}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parser.Parse(input)
	}
}

func BenchmarkArgumentParser_ParseEmpty(b *testing.B) {
	parser := NewArgumentParser()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parser.Parse("")
	}
}

func BenchmarkArgumentParser_ParseBytes(b *testing.B) {
	parser := NewArgumentParser()
	input := []byte(`{"path": "/tmp/file.txt", "mode": 0644, "recursive": true}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parser.ParseBytes(input)
	}
}
