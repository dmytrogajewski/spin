package fuzzy

import (
	"strings"

	"github.com/dmytrogajewski/spin/pkg/alg/stringsx"
)

// CollapseFind collapses consecutive blank lines and finds matches.
func CollapseFind(fileContent, oldContent string) []MatchResult {
	normalizedOld := stringsx.CollapseBlankLines(oldContent)
	normalizedFile := stringsx.CollapseBlankLines(fileContent)

	matches := findByNormalized(fileContent, normalizedFile, normalizedOld)
	if len(matches) > 0 {
		return matches
	}

	// Also try with stripped trailing whitespace per line + collapsed blanks.
	normalizedOld = stringsx.CollapseBlankLines(trimLines(oldContent))
	normalizedFile = stringsx.CollapseBlankLines(trimLines(fileContent))

	return findByNormalized(fileContent, normalizedFile, normalizedOld)
}

// trimLines trims trailing whitespace from each line.
func trimLines(str string) string {
	lines := strings.Split(str, "\n")
	for idx, line := range lines {
		lines[idx] = strings.TrimRight(line, " \t")
	}

	return strings.Join(lines, "\n")
}
