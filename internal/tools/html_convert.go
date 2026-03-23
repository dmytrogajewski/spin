package tools

import (
	"bytes"
	"fmt"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"github.com/dmytrogajewski/spin/pkg/alg/stringsx"
)

// HTMLConverter converts HTML content to markdown-formatted text.
type HTMLConverter func(htmlContent []byte) string

// shouldSkipElement returns true for HTML elements whose content should be stripped.
func shouldSkipElement(a atom.Atom) bool {
	switch a {
	case atom.Script, atom.Style, atom.Nav, atom.Footer, atom.Header, atom.Noscript:
		return true
	default:
		return false
	}
}

// headingPrefix returns the markdown heading prefix for heading atoms, or empty string.
func headingPrefix(a atom.Atom) string {
	switch a {
	case atom.H1:
		return "# "
	case atom.H2:
		return "## "
	case atom.H3:
		return "### "
	case atom.H4:
		return "#### "
	case atom.H5:
		return "##### "
	case atom.H6:
		return "###### "
	default:
		return ""
	}
}

// ConvertHTML converts HTML content to markdown-formatted text.
// It preserves document structure: headings, paragraphs, lists, links, and code blocks.
// Script, style, nav, footer, and header elements are stripped.
func ConvertHTML(htmlContent []byte) string {
	doc, parseErr := html.Parse(bytes.NewReader(htmlContent))
	if parseErr != nil {
		return string(htmlContent)
	}

	var builder strings.Builder

	walkNode(&builder, doc)

	return stringsx.CollapseBlankLines(strings.TrimSpace(builder.String()))
}

// walkNode recursively walks the HTML tree and writes markdown to the builder.
func walkNode(builder *strings.Builder, node *html.Node) {
	if node == nil {
		return
	}

	switch node.Type {
	case html.TextNode:
		writeTextNode(builder, node)
	case html.ElementNode:
		writeElementNode(builder, node)
	default:
		walkChildren(builder, node)
	}
}

// writeTextNode writes a text node's content, collapsing whitespace.
func writeTextNode(builder *strings.Builder, node *html.Node) {
	text := strings.TrimSpace(node.Data)
	if text != "" {
		builder.WriteString(text)
	}
}

// writeElementNode handles an HTML element node, writing markdown equivalents.
func writeElementNode(builder *strings.Builder, node *html.Node) {
	if shouldSkipElement(node.DataAtom) {
		return
	}

	if prefix := headingPrefix(node.DataAtom); prefix != "" {
		writeHeading(builder, node, prefix)

		return
	}

	writeNonHeadingElement(builder, node)
}

// writeNonHeadingElement handles non-heading, non-skip elements.
func writeNonHeadingElement(builder *strings.Builder, node *html.Node) {
	switch node.DataAtom {
	case atom.P, atom.Div, atom.Section, atom.Article, atom.Main:
		writeBlock(builder, node)
	case atom.A:
		writeLink(builder, node)
	case atom.Li:
		writeListItem(builder, node)
	case atom.Pre:
		writePreformatted(builder, node)
	case atom.Code:
		writeInlineCode(builder, node)
	case atom.Br:
		builder.WriteString("\n")
	case atom.Hr:
		builder.WriteString("\n---\n\n")
	case atom.Strong, atom.B:
		writeBold(builder, node)
	case atom.Em, atom.I:
		writeItalic(builder, node)
	default:
		walkChildren(builder, node)
	}
}

// writeHeading writes a markdown heading.
func writeHeading(builder *strings.Builder, node *html.Node, prefix string) {
	fmt.Fprintf(builder, "\n%s", prefix)

	walkChildren(builder, node)

	builder.WriteString("\n\n")
}

// writeBlock writes a block element with surrounding newlines.
func writeBlock(builder *strings.Builder, node *html.Node) {
	builder.WriteString("\n")

	walkChildren(builder, node)

	builder.WriteString("\n")
}

// writeLink writes a markdown link [text](href).
func writeLink(builder *strings.Builder, node *html.Node) {
	href := getAttr(node, "href")

	builder.WriteString("[")

	walkChildren(builder, node)

	fmt.Fprintf(builder, "](%s)", href)
}

// writeListItem writes a markdown list item.
func writeListItem(builder *strings.Builder, node *html.Node) {
	builder.WriteString("\n- ")

	walkChildren(builder, node)
}

// writePreformatted writes a markdown code block.
func writePreformatted(builder *strings.Builder, node *html.Node) {
	builder.WriteString("\n```\n")

	writeRawText(builder, node)

	builder.WriteString("\n```\n")
}

// writeInlineCode writes inline code with backticks.
func writeInlineCode(builder *strings.Builder, node *html.Node) {
	builder.WriteString("`")

	walkChildren(builder, node)

	builder.WriteString("`")
}

// writeBold writes bold text with **.
func writeBold(builder *strings.Builder, node *html.Node) {
	builder.WriteString("**")

	walkChildren(builder, node)

	builder.WriteString("**")
}

// writeItalic writes italic text with *.
func writeItalic(builder *strings.Builder, node *html.Node) {
	builder.WriteString("*")

	walkChildren(builder, node)

	builder.WriteString("*")
}

// walkChildren walks all child nodes.
func walkChildren(builder *strings.Builder, node *html.Node) {
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		walkNode(builder, child)
	}
}

// writeRawText extracts raw text from a node tree without any formatting.
func writeRawText(builder *strings.Builder, node *html.Node) {
	if node.Type == html.TextNode {
		builder.WriteString(node.Data)

		return
	}

	for child := node.FirstChild; child != nil; child = child.NextSibling {
		writeRawText(builder, child)
	}
}

// getAttr returns the value of the named attribute, or empty string if not found.
func getAttr(node *html.Node, name string) string {
	for _, attr := range node.Attr {
		if attr.Key == name {
			return attr.Val
		}
	}

	return ""
}
