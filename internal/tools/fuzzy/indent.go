package fuzzy

import (
	"strings"

	"github.com/dmytrogajewski/spin/pkg/alg/search"
)

// IndentFind strips leading whitespace from each line and compares line-by-line.
func IndentFind(fileContent, oldContent string) []MatchResult {
	return findByLineTransform(fileContent, oldContent, func(line string) string {
		return strings.TrimLeft(line, " \t")
	})
}

// findByLineTransform applies a per-line transform to both inputs and finds
// matching line sequences. Shared by passes that differ only in line normalization.
func findByLineTransform(fileContent, oldContent string, transform func(string) string) []MatchResult {
	oldLines := strings.Split(oldContent, "\n")
	transformedOld := make([]string, len(oldLines))

	for idx, line := range oldLines {
		transformedOld[idx] = transform(line)
	}

	fileLines := strings.Split(fileContent, "\n")
	transformedFile := make([]string, len(fileLines))

	for idx, line := range fileLines {
		transformedFile[idx] = transform(line)
	}

	return findLineSequence(fileContent, fileLines, transformedFile, transformedOld)
}

// findLineSequence finds where strippedTarget appears as a consecutive subsequence in strippedFile.
func findLineSequence(
	originalContent string,
	originalLines, strippedFile, strippedTarget []string,
) []MatchResult {
	if len(strippedTarget) == 0 {
		return nil
	}

	var results []MatchResult

	targetLen := len(strippedTarget)

	strEq := func(a, b string) bool { return a == b }

	for idx := 0; idx <= len(strippedFile)-targetLen; idx++ {
		if search.MatchesAt(strippedFile, strippedTarget, idx, strEq) {
			start := search.LineOffset(originalLines, idx)
			end := search.LineOffsetEnd(originalLines, idx+targetLen-1)

			results = append(results, MatchResult{
				Start:    start,
				End:      end,
				Original: originalContent[start:end],
			})
		}
	}

	return results
}
