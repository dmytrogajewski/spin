package fuzzy

import (
	"strings"

	"github.com/dmytrogajewski/spin/pkg/alg/search"
	"github.com/dmytrogajewski/spin/pkg/alg/stringsx"
)

// AnchorFind uses the first and last non-blank lines of oldContent as anchors
// to locate the matching region in fileContent.
func AnchorFind(fileContent, oldContent string) []MatchResult {
	firstAnchor, lastAnchor := extractAnchors(oldContent)
	if firstAnchor == "" {
		return nil
	}

	fileLines := strings.Split(fileContent, "\n")
	oldLineCount := stringsx.CountLines(oldContent, func(line string) bool {
		return strings.TrimSpace(line) != ""
	})

	var results []MatchResult

	for idx, line := range fileLines {
		trimmed := strings.TrimSpace(line)
		if trimmed != firstAnchor {
			continue
		}

		// Look for last anchor within a reasonable window.
		endIdx := findLastAnchor(fileLines, lastAnchor, idx, oldLineCount)
		if endIdx < 0 {
			continue
		}

		start := search.LineOffset(fileLines, idx)
		end := search.LineOffsetEnd(fileLines, endIdx)

		results = append(results, MatchResult{
			Start:    start,
			End:      end,
			Original: fileContent[start:end],
		})
	}

	return results
}

// extractAnchors returns the first and last non-blank lines trimmed.
func extractAnchors(content string) (first, last string) {
	for line := range strings.SplitSeq(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		if first == "" {
			first = trimmed
		}

		last = trimmed
	}

	return first, last
}

// anchorSearchWindowMultiplier defines how much larger than the original
// the search window can be when looking for the last anchor.
const anchorSearchWindowMultiplier = 2

// findLastAnchor searches for the last anchor line within a reasonable window.
func findLastAnchor(lines []string, lastAnchor string, startIdx, expectedLines int) int {
	maxEnd := min(startIdx+expectedLines*anchorSearchWindowMultiplier, len(lines))

	for idx := maxEnd - 1; idx > startIdx; idx-- {
		if strings.TrimSpace(lines[idx]) == lastAnchor {
			return idx
		}
	}

	return -1
}
