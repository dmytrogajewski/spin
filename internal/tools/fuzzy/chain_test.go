package fuzzy_test

// Journey: specs/journeys/JOURNEY-R4.2.md.

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/tools/fuzzy"
)

// Exact pass tests.

func TestExactPass_Match(t *testing.T) {
	t.Parallel()

	results := fuzzy.ExactFind("hello world", "world")

	require.Len(t, results, 1)
	require.Equal(t, "world", results[0].Original)
	require.Equal(t, 6, results[0].Start)
	require.Equal(t, 11, results[0].End)
}

func TestExactPass_NoMatch(t *testing.T) {
	t.Parallel()

	results := fuzzy.ExactFind("hello world", "xyz")
	require.Empty(t, results)
}

func TestExactPass_AmbiguousMatch(t *testing.T) {
	t.Parallel()

	results := fuzzy.ExactFind("foo bar foo", "foo")

	require.Len(t, results, 2)
	require.Equal(t, 0, results[0].Start)
	require.Equal(t, 8, results[1].Start)
}

// Whitespace pass tests.

func TestWhitespacePass_ExtraSpaces(t *testing.T) {
	t.Parallel()

	fileContent := "if  (x  ==  y) {"
	oldContent := "if (x == y) {"

	results := fuzzy.WhitespaceFind(fileContent, oldContent)

	require.Len(t, results, 1)
	require.Equal(t, fileContent, results[0].Original)
}

func TestWhitespacePass_TabsAndSpaces(t *testing.T) {
	t.Parallel()

	fileContent := "a\t\tb"
	oldContent := "a b"

	results := fuzzy.WhitespaceFind(fileContent, oldContent)

	require.Len(t, results, 1)
	require.Equal(t, fileContent, results[0].Original)
}

// Indent pass tests.

func TestIndentPass_DifferentIndentation(t *testing.T) {
	t.Parallel()

	fileContent := "\t\treturn nil\n\t}"
	oldContent := "    return nil\n}"

	results := fuzzy.IndentFind(fileContent, oldContent)

	require.Len(t, results, 1)
	require.Equal(t, fileContent, results[0].Original)
}

func TestIndentPass_NoMatch(t *testing.T) {
	t.Parallel()

	results := fuzzy.IndentFind("return nil\n}", "return err\n}")
	require.Empty(t, results)
}

// Escape pass tests.

func TestEscapePass_EscapeDifferences(t *testing.T) {
	t.Parallel()

	fileContent := "fmt.Println(\"hello\tworld\")"
	oldContent := `fmt.Println("hello\tworld")`

	results := fuzzy.EscapeFind(fileContent, oldContent)

	require.Len(t, results, 1)
	require.Equal(t, fileContent, results[0].Original)
}

// LineEnd pass tests.

func TestLineEndPass_CRLFvsLF(t *testing.T) {
	t.Parallel()

	fileContent := "line1\r\nline2\r\nline3"
	oldContent := "line1\nline2\nline3"

	results := fuzzy.LineEndFind(fileContent, oldContent)

	require.Len(t, results, 1)
	require.Equal(t, fileContent, results[0].Original)
}

// Trim pass tests.

func TestTrimPass_TrailingSpaces(t *testing.T) {
	t.Parallel()

	fileContent := "hello   \nworld   "
	oldContent := "hello\nworld"

	results := fuzzy.TrimFind(fileContent, oldContent)

	require.Len(t, results, 1)
	require.Equal(t, fileContent, results[0].Original)
}

// Collapse pass tests.

func TestCollapsePass_ExtraBlankLines(t *testing.T) {
	t.Parallel()

	fileContent := "func a() {\n\n\n\n\treturn\n}"
	oldContent := "func a() {\n\n\treturn\n}"

	results := fuzzy.CollapseFind(fileContent, oldContent)

	require.Len(t, results, 1)
	require.Equal(t, fileContent, results[0].Original)
}

// Anchor pass tests.

func TestAnchorPass_ContextAnchors(t *testing.T) {
	t.Parallel()

	fileContent := "package main\n\nfunc hello() {\n\tfmt.Println(\"hi\")\n}\n\nfunc bye() {"
	oldContent := "func hello() {\n  println(\"hi\")\n}"

	results := fuzzy.AnchorFind(fileContent, oldContent)

	require.Len(t, results, 1)
	require.Contains(t, results[0].Original, "func hello()")
	require.Contains(t, results[0].Original, "}")
}

func TestAnchorPass_NoAnchors(t *testing.T) {
	t.Parallel()

	results := fuzzy.AnchorFind("hello world", "   \n  \n  ")
	require.Empty(t, results)
}

// Partial pass tests.

func TestPartialPass_SubstringMatch(t *testing.T) {
	t.Parallel()

	fileContent := "func processData(input string) error {"
	// Matches at least 60% of oldContent.
	oldContent := "func processData(input string)"

	results := fuzzy.PartialFind(fileContent, oldContent)

	require.Len(t, results, 1)
	require.Equal(t, oldContent, results[0].Original)
}

func TestPartialPass_BelowThreshold(t *testing.T) {
	t.Parallel()

	results := fuzzy.PartialFind("completely different text here", "xyz abc 123 456 789")
	require.Empty(t, results)
}

func TestPartialPass_EmptyInput(t *testing.T) {
	t.Parallel()

	require.Empty(t, fuzzy.PartialFind("", "test"))
	require.Empty(t, fuzzy.PartialFind("test", ""))
}

// Chain tests.

func TestChain_ShortCircuitsOnExact(t *testing.T) {
	t.Parallel()

	chain := fuzzy.DefaultChain()

	result := chain.Find("hello world", "world")

	require.NotNil(t, result)
	require.Equal(t, "exact", result.PassName)
	require.Equal(t, "world", result.Original)
}

func TestChain_FallsThroughToFuzzy(t *testing.T) {
	t.Parallel()

	chain := fuzzy.DefaultChain()

	// Indentation differs, so exact won't match but indent will.
	fileContent := "\t\treturn nil\n\t}"
	oldContent := "    return nil\n}"

	result := chain.Find(fileContent, oldContent)

	require.NotNil(t, result)
	require.Equal(t, "indent", result.PassName)
	require.Equal(t, fileContent, result.Original)
}

func TestChain_AllPassesFail(t *testing.T) {
	t.Parallel()

	chain := fuzzy.DefaultChain()

	result := chain.Find("hello world", "completely unrelated and different text content that has no overlap whatsoever")
	require.Nil(t, result)
}

func TestChain_FindAll_ReturnsAllMatches(t *testing.T) {
	t.Parallel()

	chain := fuzzy.DefaultChain()

	results := chain.FindAll("foo bar foo baz foo", "foo")

	require.Len(t, results, 3)

	for _, res := range results {
		require.Equal(t, "exact", res.PassName)
	}
}

func TestChain_FindAll_NoMatch(t *testing.T) {
	t.Parallel()

	chain := fuzzy.DefaultChain()

	results := chain.FindAll("hello", "xyz")
	require.Nil(t, results)
}
