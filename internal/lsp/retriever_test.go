package lsp_test

// Journey: specs/journeys/JOURNEY-R8.2.md.

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/lsp"
)

func TestMatcher_Exact(t *testing.T) {
	t.Parallel()

	matcher := lsp.ParseMatcher("HandleRequest")

	require.True(t, matcher.Match("HandleRequest"))
	require.False(t, matcher.Match("handleRequest"))
	require.False(t, matcher.Match("HandleRequestFoo"))
}

func TestMatcher_Prefix(t *testing.T) {
	t.Parallel()

	matcher := lsp.ParseMatcher("Handle.")

	require.True(t, matcher.Match("HandleRequest"))
	require.True(t, matcher.Match("HandleResponse"))
	require.False(t, matcher.Match("ProcessRequest"))
	require.False(t, matcher.Match(""))
}

func TestMatcher_Wildcard(t *testing.T) {
	t.Parallel()

	matcher := lsp.ParseMatcher("Handle*")

	require.True(t, matcher.Match("HandleRequest"))
	require.True(t, matcher.Match("HandleResponse"))
	require.True(t, matcher.Match("Handle"))
	require.False(t, matcher.Match("ProcessRequest"))
}

func TestMatcher_QuestionMark(t *testing.T) {
	t.Parallel()

	matcher := lsp.ParseMatcher("Get?")

	require.True(t, matcher.Match("GetX"))
	require.False(t, matcher.Match("GetXY"))
	require.False(t, matcher.Match("Get"))
}

func TestMatcher_EmptyPattern(t *testing.T) {
	t.Parallel()

	matcher := lsp.ParseMatcher("")

	require.True(t, matcher.Match(""))
	require.False(t, matcher.Match("anything"))
}

func TestFilterSymbols(t *testing.T) {
	t.Parallel()

	symbols := []lsp.Symbol{
		{Name: "HandleRequest", Kind: lsp.SymbolFunction},
		{Name: "HandleResponse", Kind: lsp.SymbolFunction},
		{Name: "ProcessRequest", Kind: lsp.SymbolFunction},
		{Name: "main", Kind: lsp.SymbolFunction},
	}

	matcher := lsp.ParseMatcher("Handle*")
	matched := lsp.FilterSymbols(symbols, matcher)

	require.Len(t, matched, 2)
	require.Equal(t, "HandleRequest", matched[0].Name)
	require.Equal(t, "HandleResponse", matched[1].Name)
}

func TestFilterSymbols_NoMatch(t *testing.T) {
	t.Parallel()

	symbols := []lsp.Symbol{
		{Name: "main", Kind: lsp.SymbolFunction},
	}

	matcher := lsp.ParseMatcher("NonExistent")
	matched := lsp.FilterSymbols(symbols, matcher)

	require.Empty(t, matched)
}

func TestFilterSymbols_Empty(t *testing.T) {
	t.Parallel()

	matcher := lsp.ParseMatcher("main")
	matched := lsp.FilterSymbols(nil, matcher)

	require.Empty(t, matched)
}
