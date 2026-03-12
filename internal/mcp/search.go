package mcp

import (
	"sort"
	"strings"

	"github.com/dmytrogajewski/spin/internal/tools"
)

// SearchOptions configures search behavior.
type SearchOptions struct {
	// FuzzyMatch enables fuzzy string matching (default: true).
	FuzzyMatch bool

	// MatchDescription searches tool descriptions (default: true).
	MatchDescription bool

	// MinScore is the minimum relevance score (0.0-1.0, default: 0.3).
	MinScore float64
}

// DefaultSearchOptions returns the default search options.
func DefaultSearchOptions() SearchOptions {
	return SearchOptions{
		FuzzyMatch:       true,
		MatchDescription: true,
		MinScore:         0.3,
	}
}

// searchResult pairs a tool with its relevance score.
type searchResult struct {
	tool  tools.Tool
	score float64
}

// SearchTools searches through tools and returns matches sorted by relevance.
func SearchTools(toolList []tools.Tool, query string, maxResults int, opts SearchOptions) []tools.Tool {
	if query == "" {
		if maxResults > 0 && len(toolList) > maxResults {
			return toolList[:maxResults]
		}

		return toolList
	}

	query = strings.ToLower(strings.TrimSpace(query))

	var results []searchResult

	for _, t := range toolList {
		score := scoreTool(t, query, opts)
		if score >= opts.MinScore {
			results = append(results, searchResult{tool: t, score: score})
		}
	}

	// Sort by score descending.
	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	// Apply limit.
	if maxResults > 0 && len(results) > maxResults {
		results = results[:maxResults]
	}

	// Extract tools.
	matched := make([]tools.Tool, len(results))
	for i, r := range results {
		matched[i] = r.tool
	}

	return matched
}

// scoreTool calculates a relevance score for a tool given a query.
func scoreTool(t tools.Tool, query string, opts SearchOptions) float64 {
	name := strings.ToLower(t.Name())
	desc := strings.ToLower(t.Description())

	var maxScore float64

	// Score against name.
	nameScore := scoreString(name, query, opts.FuzzyMatch)
	if nameScore > maxScore {
		maxScore = nameScore
	}

	// Score against description (with penalty).
	if opts.MatchDescription && desc != "" {
		descScore := scoreString(desc, query, opts.FuzzyMatch) * 0.6 // Description matches worth less.
		if descScore > maxScore {
			maxScore = descScore
		}
	}

	return maxScore
}

// scoreString calculates a relevance score for a string given a query.
func scoreString(s, query string, fuzzy bool) float64 {
	// Exact match.
	if s == query {
		return 1.0
	}

	// Prefix match.
	if strings.HasPrefix(s, query) {
		return 0.9
	}

	// Contains match.
	if strings.Contains(s, query) {
		return 0.7
	}

	// Word boundary match (query matches start of a word).
	words := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-' || r == ' ' || r == '.'
	})
	for _, word := range words {
		if strings.HasPrefix(word, query) {
			return 0.75
		}
	}

	// Fuzzy match using Levenshtein distance.
	if fuzzy {
		distance := levenshteinDistance(s, query)

		maxLen := max(len(s), len(query))
		if maxLen > 0 {
			similarity := 1.0 - float64(distance)/float64(maxLen)
			if similarity >= 0.5 {
				return similarity * 0.6 // Fuzzy matches worth less.
			}
		}
	}

	return 0.0
}

// levenshteinDistance calculates the edit distance between two strings.
func levenshteinDistance(s1, s2 string) int {
	if len(s1) == 0 {
		return len(s2)
	}

	if len(s2) == 0 {
		return len(s1)
	}

	// Create matrix.
	matrix := make([][]int, len(s1)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(s2)+1)
		matrix[i][0] = i
	}

	for j := range matrix[0] {
		matrix[0][j] = j
	}

	// Fill matrix.
	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			cost := 1
			if s1[i-1] == s2[j-1] {
				cost = 0
			}

			matrix[i][j] = min(
				matrix[i-1][j]+1,      // deletion.
				matrix[i][j-1]+1,      // insertion.
				matrix[i-1][j-1]+cost, // substitution.
			)
		}
	}

	return matrix[len(s1)][len(s2)]
}
