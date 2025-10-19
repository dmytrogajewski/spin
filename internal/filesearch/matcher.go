package filesearch

import (
	"path/filepath"
	"sort"
	"strings"
)

// Match represents a fuzzy match result.
type Match struct {
	Path    string
	Score   int
	Indices []int // Matched character indices
}

// Matcher performs fuzzy matching on file paths with advanced scoring.
type Matcher struct {
	caseSensitive bool
}

// NewMatcher creates a new fuzzy matcher.
func NewMatcher(caseSensitive bool) *Matcher {
	return &Matcher{
		caseSensitive: caseSensitive,
	}
}

// Score calculates the advanced fuzzy match score for a path.
// Returns -1 if no match, otherwise returns score (higher is better) and matched indices.
//
// Scoring algorithm (priority order):
//  1. Exact filename match: 100
//  2. Filename starts with query: 90
//  3. Filename contains query (early): 80-70 (position-weighted)
//  4. Path segment exact match: 60
//  5. Path segment prefix: 50
//  6. Fuzzy match (consecutive): 40+
//  7. Fuzzy match (scattered): 20+
func (m *Matcher) Score(query, path string) (int, []int) {
	if query == "" {
		return 0, nil
	}

	queryLower := strings.ToLower(query)
	pathLower := strings.ToLower(path)
	filename := filepath.Base(pathLower)

	// Try filename matches first (highest priority)
	if score, indices := m.scoreFilenameMatch(queryLower, filename, path); score > 0 {
		return score, indices
	}

	// Try path segment matches
	if score, indices := m.scorePathSegmentMatch(queryLower, pathLower, path); score > 0 {
		return score, indices
	}

	// Fall back to fuzzy matching
	return m.scoreFuzzyMatch(queryLower, pathLower, path)
}

// scoreFilenameMatch scores filename-based matches.
func (m *Matcher) scoreFilenameMatch(queryLower, filename, path string) (int, []int) {
	// 1. Exact filename match - highest priority
	if filename == queryLower {
		return 100, m.allIndicesInPath(path, len(path)-len(filename), len(filename))
	}

	// 2. Filename starts with query
	if strings.HasPrefix(filename, queryLower) {
		startIdx := len(path) - len(filename)
		return 90, m.prefixIndices(startIdx, len(queryLower))
	}

	// 3. Filename contains query (position-weighted)
	if idx := strings.Index(filename, queryLower); idx >= 0 {
		score := m.calculatePositionScore(idx, len(filename))
		startIdx := len(path) - len(filename) + idx
		return score, m.prefixIndices(startIdx, len(queryLower))
	}

	return 0, nil
}

// calculatePositionScore calculates score based on position in filename.
func (m *Matcher) calculatePositionScore(idx, filenameLen int) int {
	score := 80 - (idx * 10 / filenameLen)
	if score < 70 {
		score = 70
	}
	return score
}

// scorePathSegmentMatch scores path segment-based matches.
func (m *Matcher) scorePathSegmentMatch(queryLower, pathLower, path string) (int, []int) {
	dir := filepath.Dir(pathLower)
	if dir == "." || dir == "/" {
		return 0, nil
	}

	segments := strings.Split(dir, string(filepath.Separator))
	for _, seg := range segments {
		// Exact segment match
		if seg == queryLower {
			return 60, m.findSegmentIndices(path, seg)
		}
		// Segment prefix match
		if strings.HasPrefix(seg, queryLower) {
			return 50, m.findSegmentIndices(path, seg[:len(queryLower)])
		}
	}

	return 0, nil
}

// scoreFuzzyMatch scores fuzzy matches with bonuses.
func (m *Matcher) scoreFuzzyMatch(queryLower, pathLower, path string) (int, []int) {
	score, indices := m.matchCharacters(queryLower, pathLower)
	if score == -1 {
		return -1, nil
	}

	// Add filename bonus if match is primarily in filename
	if m.isMatchInFilename(path, indices) {
		score += 30
	}

	// Add path length bonus
	score += m.pathLengthBonus(path)

	// Ensure positive score
	if score < 1 {
		score = 1
	}

	return score, indices
}

// allIndicesInPath returns indices for all characters in a substring.
func (m *Matcher) allIndicesInPath(path string, start, length int) []int {
	indices := make([]int, length)
	for i := 0; i < length; i++ {
		indices[i] = start + i
	}
	return indices
}

// prefixIndices returns indices for prefix match starting at given position.
func (m *Matcher) prefixIndices(start, length int) []int {
	indices := make([]int, length)
	for i := 0; i < length; i++ {
		indices[i] = start + i
	}
	return indices
}

// findSegmentIndices finds indices of segment in path.
func (m *Matcher) findSegmentIndices(path, segment string) []int {
	pathLower := strings.ToLower(path)
	segLower := strings.ToLower(segment)

	idx := strings.Index(pathLower, segLower)
	if idx == -1 {
		return nil
	}

	indices := make([]int, len(segment))
	for i := 0; i < len(segment); i++ {
		indices[i] = idx + i
	}
	return indices
}

// isMatchInFilename checks if most of the matched indices are in the filename.
func (m *Matcher) isMatchInFilename(path string, indices []int) bool {
	if len(indices) == 0 {
		return false
	}

	filename := filepath.Base(path)
	filenameStart := len(path) - len(filename)

	matchesInFilename := 0
	for _, idx := range indices {
		if idx >= filenameStart {
			matchesInFilename++
		}
	}

	// More than 50% of matches in filename
	return matchesInFilename > len(indices)/2
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
