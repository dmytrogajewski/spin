package ui

import (
	"strings"
)

// ThinkingParser extracts <think>...</think> tags from streaming content.
// Returns only NEW content deltas on each Parse() call.
type ThinkingParser struct {
	buffer         string // Buffered unparsed content
	thinkingTotal  string // Total accumulated thinking content
	regularTotal   string // Total accumulated regular content
	inThinking     bool   // Currently inside <think> tag
	lastThinkingLen int   // Length of thinking returned last time
	lastRegularLen  int   // Length of regular returned last time
}

// NewThinkingParser creates a new thinking parser.
func NewThinkingParser() *ThinkingParser {
	return &ThinkingParser{}
}

// Parse processes a content delta and separates thinking from regular content.
// Returns (regularDelta, thinkingDelta) - only the NEW content since last call.
func (tp *ThinkingParser) Parse(delta string) (string, string) {
	tp.buffer += delta

	// Process buffer to update totals
	for len(tp.buffer) > 0 {
		if tp.inThinking {
			// Look for closing tag
			if idx := strings.Index(tp.buffer, "</think>"); idx != -1 {
				// Found closing tag
				thinkingPart := tp.buffer[:idx]
				tp.thinkingTotal += thinkingPart
				tp.inThinking = false
				tp.buffer = tp.buffer[idx+8:] // Skip "</think>"
			} else {
				// No closing tag yet - add to thinking total for streaming
				// But keep in buffer in case we get more
				if tp.endsWithPartialTag(tp.buffer) {
					// Don't consume partial tag
					break
				}
				tp.thinkingTotal += tp.buffer
				tp.buffer = ""
			}
		} else {
			// Look for opening tag
			if idx := strings.Index(tp.buffer, "<think>"); idx != -1 {
				// Found opening tag
				tp.regularTotal += tp.buffer[:idx]
				tp.inThinking = true
				tp.buffer = tp.buffer[idx+7:] // Skip "<think>"
			} else {
				// Check if buffer ends with partial tag to avoid splitting it
				if tp.endsWithPartialTag(tp.buffer) {
					// Keep partial tag in buffer
					break
				}
				// No opening tag - all is regular content
				tp.regularTotal += tp.buffer
				tp.buffer = ""
			}
		}
	}

	// Calculate deltas (what's new since last call)
	regularDelta := tp.regularTotal[tp.lastRegularLen:]
	thinkingDelta := tp.thinkingTotal[tp.lastThinkingLen:]

	// Update last lengths
	tp.lastRegularLen = len(tp.regularTotal)
	tp.lastThinkingLen = len(tp.thinkingTotal)

	return regularDelta, thinkingDelta
}

// endsWithPartialTag checks if the string ends with a partial <think> or </think> tag.
func (tp *ThinkingParser) endsWithPartialTag(s string) bool {
	// Check for partial opening tag: "<", "<t", "<th", "<thi", "<thin", "<think"
	openingPrefixes := []string{"<", "<t", "<th", "<thi", "<thin", "<think"}
	for _, prefix := range openingPrefixes {
		if strings.HasSuffix(s, prefix) {
			return true
		}
	}

	// Check for partial closing tag: "<", "</", "</t", "</th", "</thi", "</thin", "</think"
	closingPrefixes := []string{"</", "</t", "</th", "</thi", "</thin", "</think"}
	for _, prefix := range closingPrefixes {
		if strings.HasSuffix(s, prefix) {
			return true
		}
	}

	return false
}

// Reset clears the parser state.
func (tp *ThinkingParser) Reset() {
	tp.buffer = ""
	tp.thinkingTotal = ""
	tp.regularTotal = ""
	tp.inThinking = false
	tp.lastThinkingLen = 0
	tp.lastRegularLen = 0
}

// IsInThinking returns true if currently parsing thinking content.
func (tp *ThinkingParser) IsInThinking() bool {
	return tp.inThinking
}
