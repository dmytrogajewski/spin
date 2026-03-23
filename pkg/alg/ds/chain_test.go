package ds

// Journey: specs/journeys/JOURNEY-R20.md.

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// resultPassName is a constant for the expected pass name in test assertions.
const resultPassName = "double"

func TestChain_Find_first_match_wins(t *testing.T) {
	t.Parallel()

	chain := NewChain(
		Pass[int, string]{Name: "never", Find: func(_ int) []string { return nil }},
		Pass[int, string]{Name: resultPassName, Find: func(_ int) []string { return []string{"found"} }},
		Pass[int, string]{Name: "skip", Find: func(_ int) []string { return []string{"also"} }},
	)

	result, passName := chain.Find(42)
	require.NotNil(t, result)
	require.Equal(t, "found", *result)
	require.Equal(t, resultPassName, passName)
}

func TestChain_Find_no_match(t *testing.T) {
	t.Parallel()

	chain := NewChain(
		Pass[int, string]{Name: "nope", Find: func(_ int) []string { return nil }},
	)

	result, passName := chain.Find(1)
	require.Nil(t, result)
	require.Empty(t, passName)
}

func TestChain_FindAll(t *testing.T) {
	t.Parallel()

	chain := NewChain(
		Pass[int, string]{Name: "nope", Find: func(_ int) []string { return nil }},
		Pass[int, string]{Name: "multi", Find: func(_ int) []string { return []string{"a", "b"} }},
	)

	results, passName := chain.FindAll(1)
	require.Equal(t, []string{"a", "b"}, results)
	require.Equal(t, "multi", passName)
}

func TestChain_FindAll_no_match(t *testing.T) {
	t.Parallel()

	chain := NewChain[int, string]()

	results, passName := chain.FindAll(1)
	require.Nil(t, results)
	require.Empty(t, passName)
}

func TestChain_empty_passes(t *testing.T) {
	t.Parallel()

	chain := NewChain[int, string]()

	result, passName := chain.Find(1)
	require.Nil(t, result)
	require.Empty(t, passName)
}

func TestChain_struct_input(t *testing.T) {
	t.Parallel()

	type pair struct {
		a string
		b string
	}

	chain := NewChain(
		Pass[pair, string]{
			Name: "concat",
			Find: func(input pair) []string {
				if input.a == input.b {
					return []string{input.a}
				}

				return nil
			},
		},
	)

	result, passName := chain.Find(pair{a: "hello", b: "hello"})
	require.NotNil(t, result)
	require.Equal(t, "hello", *result)
	require.Equal(t, "concat", passName)

	// No match when different.
	result, _ = chain.Find(pair{a: "x", b: "y"})
	require.Nil(t, result)
}
