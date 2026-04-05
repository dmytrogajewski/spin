package observation_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dmytrogajewski/spin/internal/contexteng/observation"
)

// Journey: specs/journeys/JOURNEY-2.5.md.

// TestSummarize_FileRead verifies compact summary for file reads.
// Kills mutant: passing through full file content would waste context.
func TestSummarize_FileRead(t *testing.T) {
	t.Parallel()

	sum := observation.NewSummarizer()
	output := "line1\nline2\nline3"

	result := sum.Summarize(observation.ToolReadFile, output)

	assert.Contains(t, result, "Read file")
	assert.Contains(t, result, "3 lines")
	assert.Contains(t, result, "chars")
}

// TestSummarize_SearchResults verifies compact summary for searches.
// Kills mutant: full search output would waste context.
func TestSummarize_SearchResults(t *testing.T) {
	t.Parallel()

	sum := observation.NewSummarizer()
	output := "match1\nmatch2\nmatch3\nmatch4"

	result := sum.Summarize(observation.ToolFileSearch, output)

	assert.Contains(t, result, "Search completed")
	assert.Contains(t, result, "4 matches")
}

// TestSummarize_DirectoryListing verifies compact summary for listings.
// Kills mutant: full listing would waste context.
func TestSummarize_DirectoryListing(t *testing.T) {
	t.Parallel()

	sum := observation.NewSummarizer()
	output := "file1\nfile2\nfile3"

	result := sum.Summarize(observation.ToolListDirectory, output)

	assert.Contains(t, result, "Listed directory")
	assert.Contains(t, result, "3 items")
}

// TestSummarize_ShortCommand verifies short command output passes through.
// Kills mutant: summarizing short output would lose useful detail.
func TestSummarize_ShortCommand(t *testing.T) {
	t.Parallel()

	sum := observation.NewSummarizer()
	output := "OK"

	result := sum.Summarize(observation.ToolShellCommand, output)

	assert.Equal(t, "OK", result)
}

// TestSummarize_LongCommand verifies long command output gets summary.
// Kills mutant: passing through long output would waste context.
func TestSummarize_LongCommand(t *testing.T) {
	t.Parallel()

	sum := observation.NewSummarizer()
	output := strings.Repeat("x", observation.ShortOutputMax+1) + "\nline2\nline3"

	result := sum.Summarize(observation.ToolShellCommand, output)

	assert.Contains(t, result, "Command executed")
	assert.Contains(t, result, "3 lines")
}

// TestSummarize_EmptyOutput verifies empty output returns empty.
// Kills mutant: producing summary for empty output would confuse agent.
func TestSummarize_EmptyOutput(t *testing.T) {
	t.Parallel()

	sum := observation.NewSummarizer()

	result := sum.Summarize(observation.ToolReadFile, "")

	assert.Empty(t, result)
}

// TestSummarize_UnknownTool verifies fallback strategy for unknown tools.
// Kills mutant: panicking on unknown tool would crash harness.
func TestSummarize_UnknownTool(t *testing.T) {
	t.Parallel()

	sum := observation.NewSummarizer()
	output := strings.Repeat("y", observation.ShortOutputMax+1)

	result := sum.Summarize("custom_tool", output)

	assert.Contains(t, result, "Tool output")
}

// TestSummarize_UnknownToolShortOutput verifies short unknown tool output is verbatim.
// Kills mutant: summarizing short output from unknown tool would lose detail.
func TestSummarize_UnknownToolShortOutput(t *testing.T) {
	t.Parallel()

	sum := observation.NewSummarizer()

	result := sum.Summarize("custom_tool", "short result")

	assert.Equal(t, "short result", result)
}

// TestSummarizeError_Short verifies short error gets prefix.
// Kills mutant: missing error prefix would lose error classification.
func TestSummarizeError_Short(t *testing.T) {
	t.Parallel()

	result := observation.SummarizeError("permission denied")

	assert.Equal(t, "Error: permission denied", result)
}

// TestSummarizeError_Long verifies long error gets truncated.
// Kills mutant: unbounded error would waste context.
func TestSummarizeError_Long(t *testing.T) {
	t.Parallel()

	longErr := strings.Repeat("a", observation.ErrorTruncateMax+50)

	result := observation.SummarizeError(longErr)

	assert.True(t, strings.HasPrefix(result, "Error: "))
	assert.True(t, strings.HasSuffix(result, "..."))
	assert.Less(t, len(result), len(longErr))
}

// TestSummarizeError_NoCascade reproduces the bug where repeated summarization
// of error tool results caused cascading error prefix duplication,
// making the actual error unreadable to the LLM.
func TestSummarizeError_NoCascade(t *testing.T) {
	t.Parallel()

	// Simulate what happens across multiple turns of observation summarization.
	original := "execution failed: exit status 101"

	once := observation.SummarizeError(original)
	assert.Equal(t, "Error: execution failed: exit status 101", once)

	// Second summarization should NOT double-prefix.
	twice := observation.SummarizeError(once)
	assert.Equal(t, once, twice, "re-summarizing should not add another Error: prefix")

	// Even after many rounds, it should stay the same.
	result := original
	for range 10 {
		result = observation.SummarizeError(result)
	}

	assert.Equal(t, once, result, "10 rounds of summarization should produce same result as 1")
}

// TestSummarize_EmptySearchResults verifies zero matches for empty search.
// Kills mutant: counting 1 match for empty output would be wrong.
func TestSummarize_EmptySearchResults(t *testing.T) {
	t.Parallel()

	sum := observation.NewSummarizer()

	// Empty output returns "" from Summarize (early return).
	result := sum.Summarize(observation.ToolFileSearch, "")

	assert.Empty(t, result)
}
