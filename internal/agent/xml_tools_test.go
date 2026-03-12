package agent

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseToolCallsFromXML(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		content string
		want    int // number of tool calls.
	}{
		{
			name:    "no tool calls",
			content: "Just some text.",
			want:    0,
		},
		{
			name: "single tool call",
			content: `I will read the file.
<tool_call>
<function=read_file>
<parameter=path>test.txt</parameter>
</function>
</tool_call>`,
			want: 1,
		},
		{
			name: "tool call without wrapper",
			content: `<function=list_directory>
<parameter=path>.</parameter>
</function>`,
			want: 1,
		},
		{
			name: "multiple tool calls",
			content: `<function=func1><parameter=a>1</parameter></function>
<function=func2><parameter=b>2</parameter></function>`,
			want: 2,
		},
		{
			name: "tool call with multiple parameters",
			content: `<function=write_file>
<parameter=path>file.txt</parameter>
<parameter=content>hello world</parameter>
</function>`,
			want: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := parseToolCallsFromXML(tt.content)
			assert.Equal(t, tt.want, len(got))

			if tt.want > 0 {
				for _, call := range got {
					assert.NotEmpty(t, call.ID)
					assert.Equal(t, "function", string(call.Type))
					assert.NotEmpty(t, call.Function.Name)

					// Verify args are valid JSON.
					var args map[string]any

					err := json.Unmarshal([]byte(call.Function.Arguments), &args)
					require.NoError(t, err)
				}
			}
		})
	}
}
