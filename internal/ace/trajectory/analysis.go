// Package trajectory provides trajectory analysis for agent execution.
package trajectory

import (
	"strings"

	"github.com/dmytrogajewski/spin/internal/ace/generator"
	"github.com/dmytrogajewski/spin/pkg/alg/collections"
	"github.com/dmytrogajewski/spin/pkg/alg/stringsx"
)

// HasRecentError checks if any step in the last 'lookback' steps contains an error.
// Returns true if error detected, false otherwise.
// Lookback of 0 or negative value checks all steps.
func (tc *Context) HasRecentError(lookback int) bool {
	steps := collections.TailNOrAll(tc.Steps, lookback)
	for _, step := range steps {
		if containsError(step.Content) {
			return true
		}
	}

	return false
}

// ExtractErrorPatterns extracts error-related text from recent steps.
// Returns slice of error descriptions (sentences containing error keywords).
// Lookback of 0 or negative value checks all steps.
func ExtractErrorPatterns(steps []generator.TrajectoryStep, lookback int) []string {
	recentSteps := collections.TailNOrAll(steps, lookback)
	patterns := make([]string, 0)

	for _, step := range recentSteps {
		if containsError(step.Content) {
			patterns = append(patterns, step.Content)
		}
	}

	return patterns
}

// GetRecentTools extracts tool names from recent steps.
// Returns unique tool names in order of first appearance.
// Lookback of 0 or negative value checks all steps.
func (tc *Context) GetRecentTools(lookback int) []string {
	steps := collections.TailNOrAll(tc.Steps, lookback)
	seen := make(map[string]bool)
	tools := make([]string, 0)

	for _, step := range steps {
		toolName := extractToolName(step.Content)
		if toolName != "" && !seen[toolName] {
			seen[toolName] = true
			tools = append(tools, toolName)
		}
	}

	return tools
}

// extractToolName extracts tool name from step content.
// Looks for patterns like "Tool: name" or "Calling tool: name".
// Returns empty string if no tool found.
func extractToolName(content string) string {
	lower := strings.ToLower(content)

	idx := strings.Index(lower, "tool:")
	if idx == -1 {
		return ""
	}

	// Extract text after "tool:".
	after := strings.TrimSpace(content[idx+5:])
	if after == "" {
		return ""
	}

	// Get first word (tool name).
	fields := strings.Fields(after)
	if len(fields) == 0 {
		return ""
	}

	return fields[0]
}

// Common stopwords to filter out in concept extraction.
var stopwords = map[string]bool{
	"the": true, "and": true, "or": true, "but": true, "in": true,
	"on": true, "at": true, "to": true, "for": true, "of": true,
	"with": true, "a": true, "an": true, "is": true, "are": true,
	"was": true, "were": true, "be": true, "been": true, "being": true,
}

// ExtractConcepts extracts key concepts from step content.
// Returns unique concepts (words) that appear significant.
// Lookback of 0 or negative value checks all steps.
// Simple implementation: extract capitalized words and technical terms.
func ExtractConcepts(steps []generator.TrajectoryStep, lookback int) []string {
	recentSteps := collections.TailNOrAll(steps, lookback)
	seen := make(map[string]bool)
	concepts := make([]string, 0)

	for _, step := range recentSteps {
		words := strings.FieldsSeq(step.Content)
		for word := range words {
			word = strings.Trim(word, ".,!?:;\"'()[]{}")
			if concept, ok := extractConcept(word, seen); ok {
				concepts = append(concepts, concept)
			}
		}
	}

	return concepts
}

// extractConcept checks if a word is a concept (capitalized or technical term)
// and returns it if not already seen.
func extractConcept(word string, seen map[string]bool) (string, bool) {
	if word == "" || seen[word] {
		return "", false
	}

	isCapitalized := word[0] >= 'A' && word[0] <= 'Z'
	isTechnical := strings.Contains(word, "_") || strings.Contains(word, ".")

	if isCapitalized && !stopwords[strings.ToLower(word)] {
		seen[word] = true

		return word, true
	}

	if isTechnical {
		seen[word] = true

		return word, true
	}

	return "", false
}

// errorKeywords are case-insensitive indicators of execution errors.
var errorKeywords = []string{"error", "failed", "exception", "panic", "fatal"}

// containsError checks if content contains error indicators.
// Case-insensitive check for common error keywords.
func containsError(content string) bool {
	return stringsx.ContainsAnyKeyword(content, errorKeywords)
}
