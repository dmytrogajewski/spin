// Package sanitizer provides input and output sanitization.
package sanitizer

import (
	"strings"
)

// State represents the current state of the sanitizer.
type State int

// StateNormal defines a StateNormal constant.
const (
	StateNormal State = iota
	StateInThink
	StateInDrop
)

// Sanitizer filters the LLM stream to separate content, thoughts, and protocol artifacts.
type Sanitizer struct {
	state     State
	buffer    strings.Builder
	dropUntil string // The closing tag we are waiting for when in StateInDrop.
}

// New creates a new Sanitizer instance.
func New() *Sanitizer {
	return &Sanitizer{
		state: StateNormal,
	}
}

// Process feeds a chunk of text into the sanitizer and returns clean content and thoughts.
// It buffers potential tags internally.
func (s *Sanitizer) Process(chunk string) (content string, thought string) {
	// Prepend any buffered text from previous call.
	if s.buffer.Len() > 0 {
		chunk = s.buffer.String() + chunk
		s.buffer.Reset()
	}

	var (
		contentBuilder strings.Builder
		thoughtBuilder strings.Builder
	)

	i := 0
	for i < len(chunk) {
		// If we have a potential start of a tag.
		if chunk[i] == '<' {
			// Look ahead to see if we match any known tags or prefixes
			// We need to find the longest match or determine if it's not a tag.
			remaining := chunk[i:]

			// Check state transitions.
			if s.state == StateNormal {
				// Check for start tags.
				if strings.HasPrefix(remaining, "<think>") {
					s.state = StateInThink
					i += len("<think>")

					continue
				}

				if strings.HasPrefix(remaining, "<function=") {
					s.state = StateInDrop
					s.dropUntil = "</function>"
					i += len("<function=")

					continue
				}

				if strings.HasPrefix(remaining, "<parameter=") {
					s.state = StateInDrop
					s.dropUntil = "</parameter>"
					i += len("<parameter=")

					continue
				}
				// Check for standalone tags to drop.
				if strings.HasPrefix(remaining, "</tool_call>") {
					i += len("</tool_call>")

					continue
				}

				// Check partial matches (if we are at the end of the chunk).
				if isPartialMatch(remaining, []string{"<think>", "<function=", "<parameter=", "</tool_call>"}) {
					s.buffer.WriteString(remaining)

					return contentBuilder.String(), thoughtBuilder.String()
				}
			} else if s.state == StateInThink {
				// Check for end tag.
				if strings.HasPrefix(remaining, "</think>") {
					s.state = StateNormal
					i += len("</think>")

					continue
				}
				// Check partial match.
				if isPartialMatch(remaining, []string{"</think>"}) {
					s.buffer.WriteString(remaining)

					return contentBuilder.String(), thoughtBuilder.String()
				}
			} else if s.state == StateInDrop {
				// Check for end tag.
				if strings.HasPrefix(remaining, s.dropUntil) {
					matchLen := len(s.dropUntil)
					s.state = StateNormal
					s.dropUntil = ""
					i += matchLen

					continue
				}
				// Check partial match.
				if isPartialMatch(remaining, []string{s.dropUntil}) {
					s.buffer.WriteString(remaining)

					return contentBuilder.String(), thoughtBuilder.String()
				}
			}
		}

		// Process character based on state.
		char := chunk[i]

		switch s.state {
		case StateNormal:
			contentBuilder.WriteByte(char)
		case StateInThink:
			thoughtBuilder.WriteByte(char)
		case StateInDrop:
			// Drop character.
		}

		i++
	}

	return contentBuilder.String(), thoughtBuilder.String()
}

// isPartialMatch checks if s is a prefix of any candidate, but not a full match yet.
// It assumes s is potentially shorter than candidates.
func isPartialMatch(s string, candidates []string) bool {
	for _, c := range candidates {
		if strings.HasPrefix(c, s) && len(s) < len(c) {
			return true
		}
	}

	return false
}
