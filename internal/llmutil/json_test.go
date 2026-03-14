package llmutil

// Journey: specs/journeys/JOURNEY-extract-llmutil.md.

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCleanJSONResponse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain JSON passes through unchanged",
			input:    `[{"content": "test"}]`,
			expected: `[{"content": "test"}]`,
		},
		{
			name:     "JSON with markdown json block",
			input:    "```json\n[{\"content\": \"test\"}]\n```",
			expected: `[{"content": "test"}]`,
		},
		{
			name:     "JSON with markdown generic block",
			input:    "```\n[{\"content\": \"test\"}]\n```",
			expected: `[{"content": "test"}]`,
		},
		{
			name:     "JSON with extra whitespace",
			input:    "  \n```json\n[{\"content\": \"test\"}]\n```\n  ",
			expected: `[{"content": "test"}]`,
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "whitespace only",
			input:    "   \n  ",
			expected: "",
		},
		{
			name:     "trailing backticks only",
			input:    `{"key": "value"}` + "\n```",
			expected: `{"key": "value"}`,
		},
		{
			name:     "no trailing backticks",
			input:    "```json\n{\"key\": \"value\"}",
			expected: `{"key": "value"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := CleanJSONResponse(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
