// Package feedback provides feedback parsing and processing.
package feedback

import (
	"regexp"
	"strings"
)

// BulletFeedback contains utility annotations for bullets.
type BulletFeedback struct {
	// HelpfulBullets are bullets marked as helpful.
	HelpfulBullets []string // Bullet IDs.

	// HarmfulBullets are bullets marked as harmful.
	HarmfulBullets []string // Bullet IDs.

	// Explanation is optional reasoning for feedback.
	Explanation string
}

// Parser extracts bullet feedback from LLM responses.
type Parser interface {
	// Parse extracts BulletFeedback from response text.
	Parse(response string) (*BulletFeedback, error)
}

// RegexParser uses regex patterns to extract feedback.
type RegexParser struct {
	helpfulPattern     *regexp.Regexp
	harmfulPattern     *regexp.Regexp
	explanationPattern *regexp.Regexp
}

// NewRegexParser creates a regex-based parser.
func NewRegexParser() *RegexParser {
	return &RegexParser{
		helpfulPattern:     regexp.MustCompile(`HELPFUL:\s*\[(.*?)\]`),
		harmfulPattern:     regexp.MustCompile(`HARMFUL:\s*\[(.*?)\]`),
		explanationPattern: regexp.MustCompile(`EXPLANATION:\s*(.+?)(?:\n\n|$)`),
	}
}

// Parse implements Parser interface.
func (p *RegexParser) Parse(response string) (*BulletFeedback, error) {
	feedback := &BulletFeedback{
		HelpfulBullets: []string{},
		HarmfulBullets: []string{},
	}

	// Extract HELPFUL markers.
	helpfulMatches := p.helpfulPattern.FindStringSubmatch(response)
	if len(helpfulMatches) > 1 {
		markers := parseBulletMarkers(helpfulMatches[1])
		feedback.HelpfulBullets = markers
	}

	// Extract HARMFUL markers.
	harmfulMatches := p.harmfulPattern.FindStringSubmatch(response)
	if len(harmfulMatches) > 1 {
		markers := parseBulletMarkers(harmfulMatches[1])
		feedback.HarmfulBullets = markers
	}

	// Extract EXPLANATION (optional).
	explMatches := p.explanationPattern.FindStringSubmatch(response)
	if len(explMatches) > 1 {
		feedback.Explanation = strings.TrimSpace(explMatches[1])
	}

	return feedback, nil
}

// parseBulletMarkers parses "[B1, B3, B5]" -> ["B1", "B3", "B5"].
func parseBulletMarkers(s string) []string {
	if s == "" {
		return []string{}
	}

	// Split by comma.
	parts := strings.Split(s, ",")
	markers := make([]string, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			markers = append(markers, part)
		}
	}

	return markers
}
