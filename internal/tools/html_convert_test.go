package tools_test

// Journey: specs/journeys/JOURNEY-R8.3.md.

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/tools"
)

func TestConvertHTML_Paragraph(t *testing.T) {
	t.Parallel()

	input := []byte("<html><body><p>Hello world</p></body></html>")
	result := tools.ConvertHTML(input)

	require.Contains(t, result, "Hello world")
}

func TestConvertHTML_Headings(t *testing.T) {
	t.Parallel()

	input := []byte("<h1>Title</h1><h2>Subtitle</h2><h3>Section</h3>")
	result := tools.ConvertHTML(input)

	require.Contains(t, result, "# Title")
	require.Contains(t, result, "## Subtitle")
	require.Contains(t, result, "### Section")
}

func TestConvertHTML_Links(t *testing.T) {
	t.Parallel()

	input := []byte(`<p>Visit <a href="https://example.com">Example</a></p>`)
	result := tools.ConvertHTML(input)

	require.Contains(t, result, "[Example](https://example.com)")
}

func TestConvertHTML_List(t *testing.T) {
	t.Parallel()

	input := []byte("<ul><li>First</li><li>Second</li></ul>")
	result := tools.ConvertHTML(input)

	require.Contains(t, result, "- First")
	require.Contains(t, result, "- Second")
}

func TestConvertHTML_CodeBlock(t *testing.T) {
	t.Parallel()

	input := []byte("<pre><code>func main() {}</code></pre>")
	result := tools.ConvertHTML(input)

	require.Contains(t, result, "```")
	require.Contains(t, result, "func main() {}")
}

func TestConvertHTML_InlineCode(t *testing.T) {
	t.Parallel()

	input := []byte("<p>Use the <code>fmt</code> package</p>")
	result := tools.ConvertHTML(input)

	require.Contains(t, result, "`fmt`")
}

func TestConvertHTML_StripsScript(t *testing.T) {
	t.Parallel()

	input := []byte("<p>Hello</p><script>alert('xss')</script><p>World</p>")
	result := tools.ConvertHTML(input)

	require.Contains(t, result, "Hello")
	require.Contains(t, result, "World")
	require.NotContains(t, result, "alert")
	require.NotContains(t, result, "script")
}

func TestConvertHTML_StripsStyle(t *testing.T) {
	t.Parallel()

	input := []byte("<style>body { color: red; }</style><p>Content</p>")
	result := tools.ConvertHTML(input)

	require.Contains(t, result, "Content")
	require.NotContains(t, result, "color")
}

func TestConvertHTML_StripsNav(t *testing.T) {
	t.Parallel()

	input := []byte("<nav>Menu</nav><main>Content</main>")
	result := tools.ConvertHTML(input)

	require.Contains(t, result, "Content")
	require.NotContains(t, result, "Menu")
}

func TestConvertHTML_Bold(t *testing.T) {
	t.Parallel()

	input := []byte("<p>This is <strong>important</strong></p>")
	result := tools.ConvertHTML(input)

	require.Contains(t, result, "**important**")
}

func TestConvertHTML_Italic(t *testing.T) {
	t.Parallel()

	input := []byte("<p>This is <em>emphasized</em></p>")
	result := tools.ConvertHTML(input)

	require.Contains(t, result, "*emphasized*")
}

func TestConvertHTML_HorizontalRule(t *testing.T) {
	t.Parallel()

	input := []byte("<p>Above</p><hr><p>Below</p>")
	result := tools.ConvertHTML(input)

	require.Contains(t, result, "---")
}

func TestConvertHTML_EmptyInput(t *testing.T) {
	t.Parallel()

	result := tools.ConvertHTML(nil)

	require.Empty(t, result)
}

func TestConvertHTML_PlainText(t *testing.T) {
	t.Parallel()

	input := []byte("Just plain text")
	result := tools.ConvertHTML(input)

	require.Contains(t, result, "Just plain text")
}

func TestConvertHTML_CollapsesBlankLines(t *testing.T) {
	t.Parallel()

	input := []byte("<p>One</p><p></p><p></p><p>Two</p>")
	result := tools.ConvertHTML(input)

	require.Contains(t, result, "One")
	require.Contains(t, result, "Two")
	require.NotContains(t, result, "\n\n\n")
}
