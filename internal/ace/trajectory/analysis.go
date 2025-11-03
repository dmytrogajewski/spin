package trajectory

import (
	"strings"

	"github.com/dmytrogajewski/spin/internal/ace/generator"
)

// HasRecentError checks if any step in the last 'lookback' steps contains an error.
// Returns true if error detected, false otherwise.
// Lookback of 0 or negative value checks all steps.
func (tc *TrajectoryContext) HasRecentError(lookback int) bool {
	steps := getRecentSteps(tc.Steps, lookback)
	for _, step := range steps {
		if containsError(step.Content) {
			return true
		}
	}
	return false
}

// getRecentSteps returns the last 'lookback' steps.
// If lookback <= 0 or >= len(steps), returns all steps.
func getRecentSteps(steps []generator.TrajectoryStep, lookback int) []generator.TrajectoryStep {
	if lookback <= 0 || lookback >= len(steps) {
		return steps
	}
	return steps[len(steps)-lookback:]
}

// ExtractErrorPatterns extracts error-related text from recent steps.
// Returns slice of error descriptions (sentences containing error keywords).
// Lookback of 0 or negative value checks all steps.
func ExtractErrorPatterns(steps []generator.TrajectoryStep, lookback int) []string {
	recentSteps := getRecentSteps(steps, lookback)
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
func (tc *TrajectoryContext) GetRecentTools(lookback int) []string {
	steps := getRecentSteps(tc.Steps, lookback)
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

	// Extract text after "tool:"
	after := strings.TrimSpace(content[idx+5:])
	if after == "" {
		return ""
	}

	// Get first word (tool name)
	fields := strings.Fields(after)
	if len(fields) == 0 {
		return ""
	}

	return fields[0]
}

// ExtractConcepts extracts key concepts from step content.
// Returns unique concepts (words) that appear significant.
// Lookback of 0 or negative value checks all steps.
// Simple implementation: extract capitalized words and technical terms.
func ExtractConcepts(steps []generator.TrajectoryStep, lookback int) []string {
	recentSteps := getRecentSteps(steps, lookback)
	seen := make(map[string]bool)
	concepts := make([]string, 0)

	// Common stopwords to filter out
	stopwords := map[string]bool{
		"the": true, "and": true, "or": true, "but": true, "in": true,
		"on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "a": true, "an": true, "is": true, "are": true,
		"was": true, "were": true, "be": true, "been": true, "being": true,
	}

	for _, step := range recentSteps {
		// Split content into words
		words := strings.Fields(step.Content)
		for _, word := range words {
			// Clean up punctuation
			word = strings.Trim(word, ".,!?:;\"'()[]{}")

			if word == "" {
				continue
			}

			// Check if word is capitalized (potential concept)
			if len(word) > 0 && word[0] >= 'A' && word[0] <= 'Z' {
				lower := strings.ToLower(word)
				if !stopwords[lower] && !seen[word] {
					seen[word] = true
					concepts = append(concepts, word)
				}
			}

			// Also check for technical terms (contains _ or .)
			if strings.Contains(word, "_") || strings.Contains(word, ".") {
				if !seen[word] {
					seen[word] = true
					concepts = append(concepts, word)
				}
			}
		}
	}

	return concepts
}

// containsError checks if content contains error indicators.
// Case-insensitive check for common error keywords.
func containsError(content string) bool {
	lower := strings.ToLower(content)
	keywords := []string{"error", "failed", "exception", "panic", "fatal"}
	for _, keyword := range keywords {
		if strings.Contains(lower, keyword) {
			return true
		}
	}
	return false
}
