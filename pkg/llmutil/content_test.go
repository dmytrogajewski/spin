package llmutil

// Journey: specs/journeys/JOURNEY-R21.md.

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractContent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input json.RawMessage
		want  string
	}{
		{name: "plain_string", input: json.RawMessage(`"hello world"`), want: "hello world"},
		{
			name:  "array_of_text",
			input: json.RawMessage(`[{"type":"text","text":"hello"},{"type":"text","text":" world"}]`),
			want:  "hello world",
		},
		{
			name:  "array_mixed_types",
			input: json.RawMessage(`[{"type":"image","url":"x"},{"type":"text","text":"caption"}]`),
			want:  "caption",
		},
		{name: "empty_string", input: json.RawMessage(`""`), want: ""},
		{name: "empty_array", input: json.RawMessage(`[]`), want: ""},
		{name: "null", input: json.RawMessage(`null`), want: ""},
		{name: "malformed", input: json.RawMessage(`{invalid`), want: ""},
		{name: "nil_input", input: nil, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := ExtractContent(tt.input)
			require.Equal(t, tt.want, got)
		})
	}
}
