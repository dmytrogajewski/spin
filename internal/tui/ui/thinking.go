package ui

import (
	"strings"
)

// ThinkingParser extracts <think>...</think> tags from streaming content.
type ThinkingParser struct {
	buffer         string // Accumulated content
	thinkingBuffer string // Accumulated thinking content
	inThinking     bool   // Currently inside <think> tag
}

// NewThinkingParser creates a new thinking parser.
func NewThinkingParser() *ThinkingParser {
	return &ThinkingParser{}
}

// Parse processes a content delta and separates thinking from regular content.
// Returns (regularContent, thinkingContent).
func (tp *ThinkingParser) Parse(delta string) (string, string) {
	tp.buffer += delta

	regularContent := ""
	thinkingContent := ""

	// Process buffer character by character to handle partial tags
	for len(tp.buffer) > 0 {
		if tp.inThinking {
			// Look for closing tag
			if idx := strings.Index(tp.buffer, "</think>"); idx != -1 {
				// Found closing tag
				thinkingPart := tp.buffer[:idx]
				tp.thinkingBuffer += thinkingPart
				thinkingContent += tp.thinkingBuffer
				tp.thinkingBuffer = ""
				tp.inThinking = false
				tp.buffer = tp.buffer[idx+8:] // Skip "</think>"
			} else {
				// No closing tag yet - accumulate in thinking buffer
				tp.thinkingBuffer += tp.buffer
				tp.buffer = ""
			}
		} else {
			// Look for opening tag
			if idx := strings.Index(tp.buffer, "<think>"); idx != -1 {
				// Found opening tag
				regularContent += tp.buffer[:idx]
				tp.inThinking = true
				tp.buffer = tp.buffer[idx+7:] // Skip "<think>"
			} else {
				// Check if buffer ends with partial tag to avoid splitting it
				if tp.endsWithPartialTag(tp.buffer) {
					// Keep partial tag in buffer
					break
				}
				// No opening tag - all is regular content
				regularContent += tp.buffer
				tp.buffer = ""
			}
		}
	}

	return regularContent, thinkingContent
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
	tp.thinkingBuffer = ""
	tp.inThinking = false
}

// IsInThinking returns true if currently parsing thinking content.
func (tp *ThinkingParser) IsInThinking() bool {
	return tp.inThinking
}
