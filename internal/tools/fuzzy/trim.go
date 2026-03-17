package fuzzy

import "strings"

// TrimFind trims trailing whitespace from each line and compares.
func TrimFind(fileContent, oldContent string) []MatchResult {
	oldLines := strings.Split(oldContent, "\n")
	trimmedOld := make([]string, len(oldLines))

	for idx, line := range oldLines {
		trimmedOld[idx] = strings.TrimRight(line, " \t")
	}

	fileLines := strings.Split(fileContent, "\n")
	trimmedFile := make([]string, len(fileLines))

	for idx, line := range fileLines {
		trimmedFile[idx] = strings.TrimRight(line, " \t")
	}

	return findLineSequence(fileContent, fileLines, trimmedFile, trimmedOld)
}
