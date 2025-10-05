package filesearch

import (
	"sort"
	"strings"
)

// Match represents a fuzzy match result.
type Match struct {
	Path    string
	Score   int
	Indices []int // Matched character indices
}

// Matcher performs fuzzy matching on file paths.
type Matcher struct {
	caseSensitive bool
}

// NewMatcher creates a new fuzzy matcher.
func NewMatcher(caseSensitive bool) *Matcher {
	return &Matcher{
		caseSensitive: caseSensitive,
	}
}

// Score calculates the fuzzy match score for a path.
// Returns -1 if no match, otherwise returns score (higher is better) and matched indices.
func (m *Matcher) Score(query, path string) (int, []int) {
	if query == "" {
		return 0, nil
	}

	// Convert to lowercase for case-insensitive matching
	queryLower := strings.ToLower(query)
	pathLower := strings.ToLower(path)

	score, indices := m.matchCharacters(queryLower, pathLower)
	if score == -1 {
		return -1, nil
	}

	// Add path length bonus
	score += m.pathLengthBonus(path)

	if score < 1 {
		score = 1 // Ensure positive score
	}

	return score, indices
}

// matchCharacters finds matching characters and calculates base score.
func (m *Matcher) matchCharacters(query, path string) (int, []int) {
	score := 0
	indices := []int{}
	queryIdx := 0

	for pathIdx := 0; pathIdx < len(path); pathIdx++ {
		if queryIdx >= len(query) {
			break
		}

		if query[queryIdx] == path[pathIdx] {
			// Match found
			indices = append(indices, pathIdx)
			queryIdx++

			// Bonus for consecutive matches
			if len(indices) > 1 && indices[len(indices)-1] == indices[len(indices)-2]+1 {
				score += 15
			}

			// Bonus for match after separator
			score += m.separatorBonus(path, pathIdx)

			// Base score for match
			score++
		}
	}

	// All query chars must match
	if queryIdx != len(query) {
		return -1, nil
	}

	return score, indices
}

// separatorBonus returns bonus points for matches after separators.
func (m *Matcher) separatorBonus(path string, idx int) int {
	if idx > 0 {
		prevChar := path[idx-1]
		if prevChar == '/' || prevChar == '_' || prevChar == '-' || prevChar == '.' {
			return 10
		}
	}
	return 0
}

// pathLengthBonus returns bonus points for shorter paths.
func (m *Matcher) pathLengthBonus(path string) int {
	pathLen := len(path)
	if pathLen < 20 {
		return 50
	} else if pathLen < 40 {
		return 25
	}
	return 10
}

// Match finds all fuzzy matches for the query in the given paths.
// Returns matches sorted by score (highest first).
func (m *Matcher) Match(query string, paths []string) []Match {
	if query == "" {
		return []Match{}
	}

	matches := make([]Match, 0, len(paths))

	for _, path := range paths {
		score, indices := m.Score(query, path)
		if score > 0 {
			matches = append(matches, Match{
				Path:    path,
				Score:   score,
				Indices: indices,
			})
		}
	}

	// Sort by score (descending)
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Score > matches[j].Score
	})

	return matches
}
