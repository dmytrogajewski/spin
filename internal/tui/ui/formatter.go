package ui

import (
	"bytes"
	"regexp"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/glamour"
)

// Formatter handles content formatting for chat messages.
// It provides markdown rendering and code syntax highlighting.
type Formatter struct {
	mdRenderer *glamour.TermRenderer
	width      int
}

// CodeBlock represents an extracted code block from markdown.
type CodeBlock struct {
	Raw      string // Original markdown (```lang\ncode\n```)
	Language string // Programming language
	Code     string // Code content without markers
}

// NewFormatter creates a new content formatter.
func NewFormatter(width int) (*Formatter, error) {
	renderer, err := createMarkdownRenderer(width)
	if err != nil {
		return nil, err
	}

	return &Formatter{
		mdRenderer: renderer,
		width:      width,
	}, nil
}

// SetWidth updates the formatter width and recreates the markdown renderer.
func (f *Formatter) SetWidth(width int) error {
	renderer, err := createMarkdownRenderer(width)
	if err != nil {
		return err
	}

	f.width = width
	f.mdRenderer = renderer
	return nil
}

// RenderMarkdown renders markdown content to formatted terminal output.
func (f *Formatter) RenderMarkdown(content string) (string, error) {
	rendered, err := f.mdRenderer.Render(content)
	if err != nil {
		// Fallback to original content if rendering fails
		return content, err
	}

	return rendered, nil
}

// HighlightCode applies syntax highlighting to code.
func (f *Formatter) HighlightCode(code, language string) string {
	// Get lexer for the language
	var lexer chroma.Lexer
	if language != "" {
		lexer = lexers.Get(language)
	}
	if lexer == nil {
		lexer = lexers.Fallback
	}

	// Use monokai style (good for both dark and light terminals)
	style := styles.Get("monokai")
	if style == nil {
		style = styles.Fallback
	}

	// Format for 256-color terminal
	formatter := formatters.Get("terminal256")
	if formatter == nil {
		formatter = formatters.Fallback
	}

	// Tokenize the code
	iterator, err := lexer.Tokenise(nil, code)
	if err != nil {
		return code // Fallback to plain code
	}

	// Format with syntax highlighting
	var buf bytes.Buffer
	err = formatter.Format(&buf, style, iterator)
	if err != nil {
		return code // Fallback to plain code
	}

	return buf.String()
}

// RenderContent renders message content with markdown and code highlighting.
// It detects code blocks, highlights them separately, then renders markdown.
func (f *Formatter) RenderContent(content string) (string, error) {
	// Extract code blocks first
	codeBlocks := ExtractCodeBlocks(content)

	if len(codeBlocks) == 0 {
		// No code blocks, just render as markdown
		return f.RenderMarkdown(content)
	}

	// For content with code blocks, render without the code blocks first
	// Then inject highlighted code after
	result := content
	highlighted := make([]string, len(codeBlocks))

	for i, block := range codeBlocks {
		// Highlight the code
		highlighted[i] = f.HighlightCode(block.Code, block.Language)

		// Remove code block from content (will add back highlighted version)
		result = strings.Replace(result, block.Raw, "", 1)
	}

	// Render the markdown text (without code blocks)
	rendered, err := f.RenderMarkdown(result)
	if err != nil {
		return content, err
	}

	// Since we can't reliably inject into rendered markdown,
	// for simplicity just concatenate: text + highlighted code blocks
	// This works well for most cases
	parts := []string{rendered}
	parts = append(parts, highlighted...)

	return strings.Join(parts, "\n"), nil
}

// ExtractCodeBlocks extracts code blocks from markdown content.
// It handles both ```lang and ``` (no language) formats.
func ExtractCodeBlocks(content string) []CodeBlock {
	// Regex to match ```lang\ncode\n``` or ```\ncode\n```
	re := regexp.MustCompile("```([a-z]*)\n([\\s\\S]*?)```")
	matches := re.FindAllStringSubmatch(content, -1)

	blocks := make([]CodeBlock, 0, len(matches))
	for _, match := range matches {
		if len(match) >= 3 {
			blocks = append(blocks, CodeBlock{
				Raw:      match[0],                    // Full match: ```lang\ncode\n```
				Language: match[1],                    // Language (may be empty)
				Code:     strings.TrimSpace(match[2]), // Code content
			})
		}
	}

	return blocks
}

// createMarkdownRenderer creates a glamour renderer for terminal output.
func createMarkdownRenderer(width int) (*glamour.TermRenderer, error) {
	return glamour.NewTermRenderer(
		glamour.WithAutoStyle(),         // Auto-detect light/dark terminal
		glamour.WithWordWrap(width-4),   // Wrap text with margin
		glamour.WithPreservedNewLines(), // Preserve line breaks
	)
}
