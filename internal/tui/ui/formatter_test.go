package ui

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFormatter(t *testing.T) {
	f, err := NewFormatter(80)

	require.NoError(t, err)
	assert.NotNil(t, f)
	assert.NotNil(t, f.mdRenderer)
}

func TestFormatter_RenderMarkdown(t *testing.T) {
	f, err := NewFormatter(80)
	require.NoError(t, err)

	tests := []struct {
		name     string
		input    string
		contains []string
	}{
		{
			name:     "simple text",
			input:    "Hello world",
			contains: []string{"Hello world"},
		},
		{
			name:     "bold text",
			input:    "This is **bold** text",
			contains: []string{"bold"},
		},
		{
			name:     "heading",
			input:    "# Title\n\nContent",
			contains: []string{"Title", "Content"},
		},
		{
			name:     "list",
			input:    "- Item 1\n- Item 2",
			contains: []string{"Item 1", "Item 2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := f.RenderMarkdown(tt.input)
			require.NoError(t, err)

			for _, substr := range tt.contains {
				assert.Contains(t, result, substr)
			}
		})
	}
}

func TestFormatter_RenderMarkdown_Error(t *testing.T) {
	f, err := NewFormatter(80)
	require.NoError(t, err)

	// Empty input should still work
	result, err := f.RenderMarkdown("")
	assert.NoError(t, err)
	assert.NotEmpty(t, result) // glamour may add formatting even for empty
}

func TestFormatter_HighlightCode(t *testing.T) {
	f, err := NewFormatter(80)
	require.NoError(t, err)

	tests := []struct {
		name     string
		code     string
		lang     string
		contains []string
	}{
		{
			name:     "go code",
			code:     "func main() {\n\tfmt.Println(\"hello\")\n}",
			lang:     "go",
			contains: []string{"main", "Println"},
		},
		{
			name:     "python code",
			code:     "def hello():\n    print('world')",
			lang:     "python",
			contains: []string{"hello", "print"},
		},
		{
			name:     "bash code",
			code:     "echo 'hello world'",
			lang:     "bash",
			contains: []string{"echo", "hello"},
		},
		{
			name:     "unknown language",
			code:     "some code",
			lang:     "unknown123",
			contains: []string{"some code"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := f.HighlightCode(tt.code, tt.lang)

			for _, substr := range tt.contains {
				assert.Contains(t, result, substr)
			}
		})
	}
}

func TestFormatter_HighlightCode_WithANSI(t *testing.T) {
	f, err := NewFormatter(80)
	require.NoError(t, err)

	code := "func test() {}"
	result := f.HighlightCode(code, "go")

	// Should contain ANSI escape codes for coloring
	// Terminal256 formatter uses \x1b[ sequences
	hasANSI := strings.Contains(result, "\x1b[") || strings.Contains(result, "\033[")
	assert.True(t, hasANSI, "highlighted code should contain ANSI escape codes")
}

func TestFormatter_ExtractCodeBlocks(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		wantCount int
		wantLang  string
		wantCode  string
	}{
		{
			name:      "single code block",
			content:   "Here's code:\n```go\nfunc main() {}\n```\nDone",
			wantCount: 1,
			wantLang:  "go",
			wantCode:  "func main() {}",
		},
		{
			name:      "multiple code blocks",
			content:   "```python\nprint('hi')\n```\nText\n```bash\necho ok\n```",
			wantCount: 2,
		},
		{
			name:      "no code blocks",
			content:   "Just plain text",
			wantCount: 0,
		},
		{
			name:      "code block without language",
			content:   "```\ncode here\n```",
			wantCount: 1,
			wantLang:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocks := ExtractCodeBlocks(tt.content)
			assert.Len(t, blocks, tt.wantCount)

			if tt.wantCount > 0 {
				if tt.wantLang != "" {
					assert.Equal(t, tt.wantLang, blocks[0].Language)
				}
				if tt.wantCode != "" {
					assert.Equal(t, tt.wantCode, blocks[0].Code)
				}
			}
		})
	}
}

func TestFormatter_RenderContent(t *testing.T) {
	f, err := NewFormatter(80)
	require.NoError(t, err)

	tests := []struct {
		name     string
		content  string
		contains []string
	}{
		{
			name:     "plain text",
			content:  "Hello world",
			contains: []string{"Hello world"},
		},
		{
			name:     "markdown with code",
			content:  "# Title\n\n```go\nfunc main() {}\n```\n\nText",
			contains: []string{"Title", "main", "Text"},
		},
		{
			name:     "multiple paragraphs",
			content:  "Para 1\n\nPara 2\n\nPara 3",
			contains: []string{"Para 1", "Para 2", "Para 3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := f.RenderContent(tt.content)
			require.NoError(t, err)

			for _, substr := range tt.contains {
				assert.Contains(t, result, substr)
			}
		})
	}
}

func TestCodeBlock_Fields(t *testing.T) {
	cb := CodeBlock{
		Raw:      "```go\ncode\n```",
		Language: "go",
		Code:     "code",
	}

	assert.Equal(t, "```go\ncode\n```", cb.Raw)
	assert.Equal(t, "go", cb.Language)
	assert.Equal(t, "code", cb.Code)
}

func TestFormatter_ResizeWidth(t *testing.T) {
	f, err := NewFormatter(80)
	require.NoError(t, err)

	// Change width
	err = f.SetWidth(120)
	require.NoError(t, err)

	// Renderer should be recreated with new width
	result, err := f.RenderMarkdown("# Title")
	require.NoError(t, err)
	assert.Contains(t, result, "Title")
}
