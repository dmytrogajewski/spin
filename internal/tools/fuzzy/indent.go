package fuzzy

import "strings"

// IndentFind strips leading whitespace from each line and compares line-by-line.
func IndentFind(fileContent, oldContent string) []MatchResult {
	oldLines := strings.Split(oldContent, "\n")
	strippedOld := make([]string, len(oldLines))

	for idx, line := range oldLines {
		strippedOld[idx] = strings.TrimLeft(line, " \t")
	}

	fileLines := strings.Split(fileContent, "\n")
	strippedFile := make([]string, len(fileLines))

	for idx, line := range fileLines {
		strippedFile[idx] = strings.TrimLeft(line, " \t")
	}

	return findLineSequence(fileContent, fileLines, strippedFile, strippedOld)
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

	for idx := 0; idx <= len(strippedFile)-targetLen; idx++ {
		if matchesAt(strippedFile, strippedTarget, idx) {
			start := lineOffset(originalLines, idx)
			end := lineOffsetEnd(originalLines, idx+targetLen-1)

			results = append(results, MatchResult{
				Start:    start,
				End:      end,
				Original: originalContent[start:end],
			})
		}
	}

	return results
}

// matchesAt checks if target matches file lines starting at position.
func matchesAt(fileLines, targetLines []string, pos int) bool {
	for idx, target := range targetLines {
		if fileLines[pos+idx] != target {
			return false
		}
	}

	return true
}

// lineOffset returns byte offset of line at given index.
func lineOffset(lines []string, lineIdx int) int {
	offset := 0

	for idx := range lineIdx {
		offset += len(lines[idx]) + 1 // +1 for newline.
	}

	return offset
}

// lineOffsetEnd returns byte offset at end of line at given index.
func lineOffsetEnd(lines []string, lineIdx int) int {
	return lineOffset(lines, lineIdx) + len(lines[lineIdx])
}
