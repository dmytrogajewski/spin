package appserver

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/dmytrogajewski/spin/internal/protocol/jsonrpc"
)

// SearchFiles performs fuzzy file search in workspace
func SearchFiles(workspacePath, query string, limit int) ([]jsonrpc.FileMatch, error) {
	if limit <= 0 {
		limit = 50
	}

	var matches []jsonrpc.FileMatch
	query = strings.ToLower(query)

	err := filepath.Walk(workspacePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Skip hidden files and directories
		if strings.HasPrefix(info.Name(), ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Fuzzy match
		filename := strings.ToLower(info.Name())
		if fuzzyMatch(filename, query) {
			relPath, _ := filepath.Rel(workspacePath, path)
			matches = append(matches, jsonrpc.FileMatch{
				Path:  relPath,
				Score: calculateScore(filename, query),
			})

			if len(matches) >= limit {
				return filepath.SkipAll
			}
		}

		return nil
	})

	return matches, err
}

// fuzzyMatch checks if query fuzzy matches target
func fuzzyMatch(target, query string) bool {
	if strings.Contains(target, query) {
		return true
	}

	// Subsequence matching
	queryIdx := 0
	for i := 0; i < len(target) && queryIdx < len(query); i++ {
		if target[i] == query[queryIdx] {
			queryIdx++
		}
	}

	return queryIdx == len(query)
}

// calculateScore computes relevance score
func calculateScore(target, query string) float64 {
	// Exact match
	if target == query {
		return 1.0
	}

	// Contains match
	if strings.Contains(target, query) {
		// Higher score for matches at start
		idx := strings.Index(target, query)
		if idx == 0 {
			return 0.9
		}
		return 0.8
	}

	// Subsequence match
	return 0.5
}
