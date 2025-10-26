package patchapply

import (
	"strings"
)

// Matcher finds hunk context in file content using fuzzy matching algorithms.
//
// The matcher uses a multi-strategy approach to locate context:
//  1. Exact match (fast path): Direct string comparison
//  2. Fuzzy match: Whitespace-normalized similarity ≥ threshold (default 85%)
//  3. Header hints: Use @@ context to disambiguate multiple occurrences
//
// Example usage:
//
//	fileLines := strings.Split(fileContent, "\n")
//	m := NewMatcher(fileLines)
//	pos := m.FindContext(contextLines, "func Process")
//	if pos < 0 {
//	    log.Fatalf("context not found")
//	}
type Matcher struct {
	fileLines       []string
	normalizedLines []string
	threshold       float64
}

// NewMatcher creates a new fuzzy matcher for the given file content.
//
// The file content is provided as a slice of lines (without newline characters).
// The matcher pre-normalizes whitespace for all file lines during initialization
// for performance. Subsequent calls to FindContext reuse this normalization.
//
// The default similarity threshold is 0.85 (85%). This can be changed using SetThreshold.
//
// Example:
//
//	fileLines := strings.Split(fileContent, "\n")
//	m := NewMatcher(fileLines)
//	pos := m.FindContext(contextLines, "")
func NewMatcher(fileLines []string) *Matcher {
	m := &Matcher{
		fileLines: fileLines,
		threshold: 0.85, // Default 85% similarity
	}
	m.normalizedLines = m.normalizeLines(fileLines)
	return m
}

// FindContext finds the line index where the context lines match in the file.
//
// Returns the starting line index (0-based) where the context was found,
// or -1 if no match was found above the similarity threshold.
//
// The search uses a multi-strategy approach:
//  1. If header is provided, find closest match to header location
//  2. Try exact match (fast path)
//  3. Try fuzzy match with whitespace normalization
//  4. Return best match if similarity ≥ threshold
//
// Parameters:
//   - contextLines: The context lines from the hunk (without +/- prefixes)
//   - header: Optional hunk header (e.g., "func Process") for disambiguation
//
// Example:
//
//	pos := m.FindContext(contextLines, "@@ func Process")
//	if pos < 0 {
//	    log.Fatalf("context not found")
//	}
//	log.Printf("Found context at line %d", pos)
func (m *Matcher) FindContext(contextLines []string, header string) int {
	if len(contextLines) == 0 {
		return 0 // Empty context matches at start
	}

	// If header provided, find closest match to header
	if header != "" {
		headerPos := m.findHeader(header)
		if headerPos >= 0 {
			// Search in window around header (±50 lines) and return closest match
			start := max(0, headerPos-50)
			end := min(len(m.fileLines), headerPos+50)
			if pos := m.findInRangeClosest(start, end, headerPos, contextLines); pos >= 0 {
				return pos
			}
		}
	}

	// Fallback: search entire file
	return m.findInRange(0, len(m.fileLines), contextLines)
}

// findInRange searches for context within a specific range of file lines.
// Returns the starting line index or -1 if not found.
func (m *Matcher) findInRange(start, end int, contextLines []string) int {
	// Try exact match first (fast path)
	if pos := m.findExact(start, end, contextLines); pos >= 0 {
		return pos
	}

	// Try fuzzy match
	return m.findFuzzy(start, end, contextLines)
}

// findInRangeClosest searches for context within a specific range and returns
// the match closest to the reference position (headerPos).
// This is used when a header is present to disambiguate multiple occurrences.
// Returns the starting line index or -1 if not found.
func (m *Matcher) findInRangeClosest(start, end, headerPos int, contextLines []string) int {
	// Try to find all matches (both exact and fuzzy) in the range
	var matches []int

	// Check for exact matches
	contextLen := len(contextLines)
	if contextLen == 0 {
		return start
	}

	searchEnd := end - contextLen
	if searchEnd < start {
		return -1
	}

	// Find all exact matches
	for i := start; i <= searchEnd; i++ {
		match := true
		for j := 0; j < contextLen; j++ {
			if m.fileLines[i+j] != contextLines[j] {
				match = false
				break
			}
		}
		if match {
			matches = append(matches, i)
		}
	}

	// If exact matches found, return the one closest to header
	if len(matches) > 0 {
		return m.findClosestToHeader(matches, headerPos)
	}

	// Try fuzzy matching and collect all matches above threshold
	normalizedContext := m.normalizeLines(contextLines)
	type fuzzyMatch struct {
		pos   int
		score float64
	}
	var fuzzyMatches []fuzzyMatch

	for i := start; i <= searchEnd; i++ {
		window := m.normalizedLines[i : i+contextLen]
		score := m.computeSimilarity(normalizedContext, window)
		if score >= m.threshold {
			fuzzyMatches = append(fuzzyMatches, fuzzyMatch{pos: i, score: score})
		}
	}

	// If fuzzy matches found, return the one closest to header
	if len(fuzzyMatches) > 0 {
		closestMatch := fuzzyMatches[0]
		minDistance := abs(fuzzyMatches[0].pos - headerPos)

		for _, fm := range fuzzyMatches[1:] {
			distance := abs(fm.pos - headerPos)
			// Prefer closer matches, but if distance is similar, prefer higher score
			if distance < minDistance || (distance == minDistance && fm.score > closestMatch.score) {
				closestMatch = fm
				minDistance = distance
			}
		}
		return closestMatch.pos
	}

	return -1
}

// findClosestToHeader returns the position closest to the header from a list of positions.
func (m *Matcher) findClosestToHeader(positions []int, headerPos int) int {
	if len(positions) == 0 {
		return -1
	}

	closest := positions[0]
	minDistance := abs(positions[0] - headerPos)

	for _, pos := range positions[1:] {
		distance := abs(pos - headerPos)
		if distance < minDistance {
			closest = pos
			minDistance = distance
		}
	}

	return closest
}

// findExact performs exact string matching for context lines.
// Returns the starting line index or -1 if not found.
func (m *Matcher) findExact(start, end int, contextLines []string) int {
	contextLen := len(contextLines)
	if contextLen == 0 {
		return start
	}

	// Ensure we don't search past valid range
	searchEnd := end - contextLen
	if searchEnd < start {
		return -1
	}

	for i := start; i <= searchEnd; i++ {
		match := true
		for j := 0; j < contextLen; j++ {
			if m.fileLines[i+j] != contextLines[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

// findFuzzy performs fuzzy matching with whitespace normalization.
// Returns the starting line index of the best match above threshold, or -1 if none found.
func (m *Matcher) findFuzzy(start, end int, contextLines []string) int {
	normalizedContext := m.normalizeLines(contextLines)
	contextLen := len(contextLines)

	if contextLen == 0 {
		return start
	}

	// Ensure we don't search past valid range
	searchEnd := end - contextLen
	if searchEnd < start {
		return -1
	}

	bestScore := 0.0
	bestPos := -1

	for i := start; i <= searchEnd; i++ {
		// Extract window
		window := m.normalizedLines[i : i+contextLen]

		// Compute similarity
		score := m.computeSimilarity(normalizedContext, window)

		if score > bestScore {
			bestScore = score
			bestPos = i
		}

		// Early exit if perfect match
		if score >= 1.0 {
			return bestPos
		}
	}

	// Return best match if above threshold
	if bestScore >= m.threshold {
		return bestPos
	}

	return -1
}

// findHeader finds the first line containing the header text (case-insensitive).
// Returns the line index or -1 if not found.
func (m *Matcher) findHeader(header string) int {
	headerLower := strings.ToLower(strings.TrimSpace(header))
	for i, line := range m.fileLines {
		if strings.Contains(strings.ToLower(line), headerLower) {
			return i
		}
	}
	return -1
}

// computeSimilarity computes the average similarity between context and window lines.
// Returns a value between 0.0 (completely different) and 1.0 (identical).
func (m *Matcher) computeSimilarity(contextLines, windowLines []string) float64 {
	if len(contextLines) != len(windowLines) {
		return 0.0
	}

	if len(contextLines) == 0 {
		return 1.0 // Empty contexts are considered identical
	}

	totalSimilarity := 0.0
	for i := range contextLines {
		similarity := calculateSimilarity(contextLines[i], windowLines[i])
		totalSimilarity += similarity
	}

	return totalSimilarity / float64(len(contextLines))
}

// normalizeLines normalizes whitespace in all lines for fuzzy comparison.
// Returns a new slice of normalized lines.
func (m *Matcher) normalizeLines(lines []string) []string {
	normalized := make([]string, len(lines))
	for i, line := range lines {
		normalized[i] = normalizeWhitespace(line)
	}
	return normalized
}

// calculateSimilarity calculates similarity between two strings using simple character matching.
func calculateSimilarity(s1, s2 string) float64 {
	if s1 == s2 {
		return 1.0
	}
	if len(s1) == 0 || len(s2) == 0 {
		return 0.0
	}

	// Simple similarity based on common characters
	common := 0
	maxLen := len(s1)
	if len(s2) > maxLen {
		maxLen = len(s2)
	}

	for i := 0; i < len(s1) && i < len(s2); i++ {
		if s1[i] == s2[i] {
			common++
		}
	}

	return float64(common) / float64(maxLen)
}

// normalizeWhitespace normalizes whitespace in a string.
func normalizeWhitespace(s string) string {
	// Replace multiple spaces with single space and trim
	result := strings.ReplaceAll(s, "\t", " ")
	for strings.Contains(result, "  ") {
		result = strings.ReplaceAll(result, "  ", " ")
	}
	return strings.TrimSpace(result)
}

// Helper functions for min/max/abs
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}
