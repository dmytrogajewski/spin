// Journey: specs/journeys/JOURNEY-R-REF-1.md.
package tools_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/internal/tools"
)

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
