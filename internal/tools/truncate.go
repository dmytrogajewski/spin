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

// TruncateOutput applies both line truncation and head-tail truncation
// using the default constants.
func TruncateOutput(input string) string {
	lined := stringsx.TruncateLines(input, MaxLineChars)

	return stringsx.TruncateHeadTail(lined, MaxOutputChars, HeadChars, TailChars)
}
