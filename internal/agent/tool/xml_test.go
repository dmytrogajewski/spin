package tool

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/tools"
)

func TestParseToolCallsFromXML(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		content      string
		want         int
		wantWarnings int
		wantFunc     string
		wantArgs     map[string]any
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
			want:     1,
			wantFunc: "read_file",
			wantArgs: map[string]any{"path": "test.txt"},
		},
		{
			name: "tool call without wrapper",
			content: `<function=list_directory>
<parameter=path>.</parameter>
</function>`,
			want:     1,
			wantFunc: "list_directory",
			wantArgs: map[string]any{"path": "."},
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
			want:     1,
			wantFunc: "write_file",
			wantArgs: map[string]any{"path": "file.txt", "content": "hello world"},
		},
		{
			name: "parameter value containing closing tag",
			content: `<function=write_file>
<parameter=path>out.xml</parameter>
<parameter=content>before </parameter> after</parameter>
</function>`,
			want:     1,
			wantFunc: "write_file",
		},
		{
			name: "nested function-like content in parameter",
			content: `<function=write_file>
<parameter=path>test.xml</parameter>
<parameter=content><function=inner><parameter=x>y</parameter></function></parameter>
</function>`,
			want:     1,
			wantFunc: "write_file",
		},
		{
			name: "function with no parameters",
			content: `<function=get_status>
</function>`,
			want:     1,
			wantFunc: "get_status",
			wantArgs: map[string]any{},
		},
		{
			name:         "unclosed function tag",
			content:      `<function=broken>some content`,
			want:         0,
			wantWarnings: 1,
		},
		{
			name:         "empty function name",
			content:      `<function=>stuff</function>`,
			want:         0,
			wantWarnings: 1,
		},
		{
			name: "type inference for numeric values",
			content: `<function=set_config>
<parameter=count>42</parameter>
<parameter=enabled>true</parameter>
<parameter=name>test</parameter>
</function>`,
			want:     1,
			wantFunc: "set_config",
			wantArgs: map[string]any{"count": float64(42), "enabled": true, "name": "test"},
		},
		{
			name: "function name with hyphens and dots",
			content: `<function=my-tool.v2>
<parameter=input>data</parameter>
</function>`,
			want:     1,
			wantFunc: "my-tool.v2",
		},
		{
			name:    "full UUID in tool call ID",
			content: `<function=test><parameter=x>1</parameter></function>`,
			want:    1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, warnings := ParseToolCallsFromXML(tt.content)
			assert.Len(t, got, tt.want)

			if tt.wantWarnings > 0 {
				assert.Len(t, warnings, tt.wantWarnings)
			}

			if tt.want > 0 {
				for _, call := range got {
					assert.NotEmpty(t, call.ID)
					assert.True(t, strings.HasPrefix(call.ID, "call_"), "ID should have call_ prefix")
					// Full UUID: "call_" + 36 chars = 41.
					assert.Len(t, call.ID, 41, "ID should use full UUID")
					assert.Equal(t, "function", string(call.Type))
					assert.NotEmpty(t, call.Function.Name)

					// Verify args are valid JSON.
					var args map[string]any

					err := json.Unmarshal([]byte(call.Function.Arguments), &args)
					require.NoError(t, err)
				}

				if tt.wantFunc != "" {
					assert.Equal(t, tt.wantFunc, got[0].Function.Name)
				}

				if tt.wantArgs != nil {
					var gotArgs map[string]any

					err := json.Unmarshal([]byte(got[0].Function.Arguments), &gotArgs)
					require.NoError(t, err)
					assert.Equal(t, tt.wantArgs, gotArgs)
				}
			}
		})
	}
}

func TestParseToolCallsFromXML_MaxLimit(t *testing.T) {
	t.Parallel()

	// Build content with more than maxXMLToolCalls function blocks.
	var sb strings.Builder

	for range maxXMLToolCalls + 5 {
		sb.WriteString(`<function=tool><parameter=i>x</parameter></function>`)
	}

	got, warnings := ParseToolCallsFromXML(sb.String())
	assert.Len(t, got, maxXMLToolCalls)
	assert.NotEmpty(t, warnings, "should warn about hitting limit")
}

func TestFindClosingTag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		content  string
		startPos int
		openTag  string
		closeTag string
		want     int
	}{
		{
			name:     "simple close",
			content:  "value</param>",
			startPos: 0,
			openTag:  "<param=",
			closeTag: "</param>",
			want:     5,
		},
		{
			name:     "nested",
			content:  "outer <param=x>inner</param> rest</param>",
			startPos: 0,
			openTag:  "<param=",
			closeTag: "</param>",
			want:     33,
		},
		{
			name:     "no close",
			content:  "value without close",
			startPos: 0,
			openTag:  "<param=",
			closeTag: "</param>",
			want:     -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := findClosingTag(tt.content, tt.startPos, tt.openTag, tt.closeTag)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestInferTypedValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  any
	}{
		{"42", float64(42)},
		{"3.14", float64(3.14)},
		{"true", true},
		{"false", false},
		{"null", nil},
		{`{"key":"val"}`, map[string]any{"key": "val"}},
		{`[1,2,3]`, []any{float64(1), float64(2), float64(3)}},
		{"hello", "hello"},
		{"", ""},
		{"not json {", "not json {"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			got := inferTypedValue(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

// mockTool implements tools.Tool for testing FormatToolsAsXMLPrompt.
type mockTool struct {
	name        string
	description string
	schema      tools.ToolSchema
}

func (m *mockTool) Name() string             { return m.name }
func (m *mockTool) Description() string      { return m.description }
func (m *mockTool) Schema() tools.ToolSchema { return m.schema }

func (m *mockTool) Execute(_ context.Context, _ tools.ToolParameters) (tools.ToolResult, error) {
	return tools.ToolResult{}, nil
}

func TestFormatToolsAsXMLPrompt(t *testing.T) {
	t.Parallel()

	t.Run("empty tools", func(t *testing.T) {
		t.Parallel()

		result := FormatToolsAsXMLPrompt(nil)
		assert.Empty(t, result)
	})

	t.Run("single tool", func(t *testing.T) {
		t.Parallel()

		toolList := []tools.Tool{
			&mockTool{
				name:        "read_file",
				description: "Read a file from disk",
				schema: tools.ToolSchema{
					Type: "function",
					Function: tools.FunctionSchema{
						Name:        "read_file",
						Description: "Read a file from disk",
						Parameters: tools.ParameterSchema{
							Type: "object",
							Properties: map[string]tools.PropertyDefinition{
								"path": {Type: "string", Description: "File path to read"},
							},
							Required: []string{"path"},
						},
					},
				},
			},
		}

		result := FormatToolsAsXMLPrompt(toolList)
		assert.Contains(t, result, "# Tool Calling")
		assert.Contains(t, result, "<function=TOOL_NAME>")
		assert.Contains(t, result, "### read_file")
		assert.Contains(t, result, "Read a file from disk")
		assert.Contains(t, result, "`path`")
		assert.Contains(t, result, "required")
	})

	t.Run("tool with enum", func(t *testing.T) {
		t.Parallel()

		toolList := []tools.Tool{
			&mockTool{
				name: "set_mode",
				schema: tools.ToolSchema{
					Type: "function",
					Function: tools.FunctionSchema{
						Name: "set_mode",
						Parameters: tools.ParameterSchema{
							Type: "object",
							Properties: map[string]tools.PropertyDefinition{
								"mode": {
									Type:        "string",
									Description: "Operating mode",
									Enum:        []string{"fast", "slow"},
								},
							},
						},
					},
				},
			},
		}

		result := FormatToolsAsXMLPrompt(toolList)
		assert.Contains(t, result, "`fast`")
		assert.Contains(t, result, "`slow`")
	})
}
