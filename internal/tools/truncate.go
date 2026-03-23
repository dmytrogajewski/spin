package tools

import "github.com/dmytrogajewski/spin/pkg/alg/stringsx"

// Output truncation constants.
const (
	// MaxOutputChars is the maximum total characters in truncated output.
	MaxOutputChars = 30_000

	// HeadChars is the number of characters preserved from the start.
	HeadChars = 10_000

	// TailChars is the number of characters preserved from the end.
	TailChars = 10_000

	// MaxLineChars is the maximum length of a single line.
	MaxLineChars = 2_000

	// TruncatedSuffix is appended to lines that exceed MaxLineChars.
	TruncatedSuffix = "... [truncated]"
)

// TruncateHeadTail truncates s to at most maxTotal characters,
// preserving the first head and last tail characters with an
// omission marker between them.
func TruncateHeadTail(input string, maxTotal, head, tail int) string {
	return stringsx.TruncateHeadTail(input, maxTotal, head, tail)
}

// TruncateLines truncates individual lines exceeding maxLen.
func TruncateLines(input string, maxLen int) string {
	return stringsx.TruncateLines(input, maxLen)
}

// TruncateOutput applies both line truncation and head-tail truncation
// using the default constants.
func TruncateOutput(input string) string {
	lined := stringsx.TruncateLines(input, MaxLineChars)

	return stringsx.TruncateHeadTail(lined, MaxOutputChars, HeadChars, TailChars)
}
