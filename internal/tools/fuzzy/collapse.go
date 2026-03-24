package fuzzy

import "github.com/dmytrogajewski/spin/pkg/alg/stringsx"

// CollapseFind collapses consecutive blank lines and finds matches.
func CollapseFind(fileContent, oldContent string) []MatchResult {
	normalizedOld := stringsx.CollapseBlankLines(oldContent)
	normalizedFile := stringsx.CollapseBlankLines(fileContent)

	matches := findByNormalized(fileContent, normalizedFile, normalizedOld)
	if len(matches) > 0 {
		return matches
	}

	// Also try with stripped trailing whitespace per line + collapsed blanks.
	normalizedOld = stringsx.CollapseBlankLines(stringsx.TrimTrailingPerLine(oldContent))
	normalizedFile = stringsx.CollapseBlankLines(stringsx.TrimTrailingPerLine(fileContent))

	return findByNormalized(fileContent, normalizedFile, normalizedOld)
}
