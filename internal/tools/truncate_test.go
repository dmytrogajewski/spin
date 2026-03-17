// Journey: specs/journeys/JOURNEY-R1.1.md.
package tools_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/tools"
)

func TestTruncateHeadTail_UnderLimit(t *testing.T) {
	t.Parallel()

	// Mutant killed: "always truncate".
	input := "short output"
	got := tools.TruncateHeadTail(input, tools.MaxOutputChars, tools.HeadChars, tools.TailChars)
	require.Equal(t, input, got)
}

func TestTruncateHeadTail_ExactLimit(t *testing.T) {
	t.Parallel()

	// Mutant killed: "off-by-one at boundary".
	input := strings.Repeat("x", tools.MaxOutputChars)
	got := tools.TruncateHeadTail(input, tools.MaxOutputChars, tools.HeadChars, tools.TailChars)
	require.Equal(t, input, got)
}

func TestTruncateHeadTail_OverLimit(t *testing.T) {
	t.Parallel()

	// Mutant killed: "missing head or tail".
	overSize := tools.MaxOutputChars + 5000
	input := strings.Repeat("A", tools.HeadChars) +
		strings.Repeat("B", overSize-tools.HeadChars-tools.TailChars) +
		strings.Repeat("C", tools.TailChars)

	got := tools.TruncateHeadTail(input, tools.MaxOutputChars, tools.HeadChars, tools.TailChars)

	require.LessOrEqual(t, len(got), tools.MaxOutputChars)
	require.True(t, strings.HasPrefix(got, strings.Repeat("A", tools.HeadChars)))
	require.True(t, strings.HasSuffix(got, strings.Repeat("C", tools.TailChars)))
}

func TestTruncateHeadTail_OmissionMarker(t *testing.T) {
	t.Parallel()

	// Mutant killed: "wrong count".
	overSize := tools.MaxOutputChars + 5000
	input := strings.Repeat("x", overSize)
	omitted := overSize - tools.HeadChars - tools.TailChars

	got := tools.TruncateHeadTail(input, tools.MaxOutputChars, tools.HeadChars, tools.TailChars)
	marker := fmt.Sprintf("\n... [%d characters omitted] ...\n", omitted)

	require.Contains(t, got, marker)
}

func TestTruncateHeadTail_EmptyInput(t *testing.T) {
	t.Parallel()

	// Mutant killed: "nil panic".
	got := tools.TruncateHeadTail("", tools.MaxOutputChars, tools.HeadChars, tools.TailChars)
	require.Empty(t, got)
}

func TestTruncateLines_UnderLimit(t *testing.T) {
	t.Parallel()

	// Mutant killed: "always truncate lines".
	input := "short line\nanother short line"
	got := tools.TruncateLines(input, tools.MaxLineChars)
	require.Equal(t, input, got)
}

func TestTruncateLines_LongLine(t *testing.T) {
	t.Parallel()

	// Mutant killed: "missing suffix".
	longLine := strings.Repeat("x", tools.MaxLineChars+500)
	got := tools.TruncateLines(longLine, tools.MaxLineChars)

	require.Len(t, got, tools.MaxLineChars)
	require.True(t, strings.HasSuffix(got, tools.TruncatedSuffix))
}

func TestTruncateLines_MultipleLines(t *testing.T) {
	t.Parallel()

	// Mutant killed: "only first line".
	long := strings.Repeat("x", tools.MaxLineChars+100)
	input := long + "\nshort\n" + long

	got := tools.TruncateLines(input, tools.MaxLineChars)
	lines := strings.Split(got, "\n")

	require.Len(t, lines, 3)
	require.True(t, strings.HasSuffix(lines[0], tools.TruncatedSuffix))
	require.Equal(t, "short", lines[1])
	require.True(t, strings.HasSuffix(lines[2], tools.TruncatedSuffix))
}

func TestTruncateLines_EmptyInput(t *testing.T) {
	t.Parallel()

	// Mutant killed: "nil panic".
	got := tools.TruncateLines("", tools.MaxLineChars)
	require.Empty(t, got)
}

func TestTruncateOutput_Integration(t *testing.T) {
	t.Parallel()

	// Mutant killed: "stages not composed".
	// Build input with a long line that should be truncated first,
	// then the overall output should be head-tail truncated.
	longLine := strings.Repeat("z", tools.MaxLineChars+500)
	bigBody := strings.Repeat("normal line\n", 3000)
	input := longLine + "\n" + bigBody

	got := tools.TruncateOutput(input)

	require.LessOrEqual(t, len(got), tools.MaxOutputChars)
	// The long line should have been truncated (suffix present in head).
	require.Contains(t, got[:tools.HeadChars], tools.TruncatedSuffix)
}

func TestTruncateOutput_ShortInput(t *testing.T) {
	t.Parallel()

	// Short input passes through unchanged.
	input := "hello world"
	got := tools.TruncateOutput(input)
	require.Equal(t, input, got)
}
