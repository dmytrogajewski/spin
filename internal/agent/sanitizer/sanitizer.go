// Package sanitizer provides input and output sanitization.
package sanitizer

import (
	"strings"

	"github.com/dmytrogajewski/spin/pkg/alg/stringsx"
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
func (s *Sanitizer) Process(chunk string) (content, thought string) {
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
		if chunk[i] == '<' {
			remaining := chunk[i:]
			advance, buffered := s.processTag(remaining, &contentBuilder, &thoughtBuilder)

			if buffered {
				return contentBuilder.String(), thoughtBuilder.String()
			}

			if advance > 0 {
				i += advance

				continue
			}
		}

		// Process character based on state.
		switch s.state {
		case StateNormal:
			contentBuilder.WriteByte(chunk[i])
		case StateInThink:
			thoughtBuilder.WriteByte(chunk[i])
		case StateInDrop:
			// Drop character.
		}

		i++
	}

	return contentBuilder.String(), thoughtBuilder.String()
}

// processTag handles tag detection at the current position.
// Returns the number of characters to advance (0 if no tag matched) and whether remaining was buffered.
func (s *Sanitizer) processTag(remaining string, contentBuilder, thoughtBuilder *strings.Builder) (advance int, buffered bool) {
	switch s.state {
	case StateNormal:
		return s.processTagNormal(remaining, contentBuilder, thoughtBuilder)
	case StateInThink:
		return s.processTagThink(remaining, contentBuilder, thoughtBuilder)
	case StateInDrop:
		return s.processTagDrop(remaining, contentBuilder, thoughtBuilder)
	}

	return 0, false
}

// processTagNormal handles tag detection in normal state.
func (s *Sanitizer) processTagNormal(remaining string, contentBuilder, thoughtBuilder *strings.Builder) (int, bool) {
	if strings.HasPrefix(remaining, "<think>") {
		s.state = StateInThink

		return len("<think>"), false
	}

	if strings.HasPrefix(remaining, "<function=") {
		s.state = StateInDrop
		s.dropUntil = "</function>"

		return len("<function="), false
	}

	if strings.HasPrefix(remaining, "<parameter=") {
		s.state = StateInDrop
		s.dropUntil = "</parameter>"

		return len("<parameter="), false
	}

	if strings.HasPrefix(remaining, "</tool_call>") {
		return len("</tool_call>"), false
	}

	if stringsx.IsPartialPrefix(remaining, []string{"<think>", "<function=", "<parameter=", "</tool_call>"}) {
		s.buffer.WriteString(remaining)

		return 0, true
	}

	_ = contentBuilder
	_ = thoughtBuilder

	return 0, false
}

// processTagThink handles tag detection in thinking state.
func (s *Sanitizer) processTagThink(remaining string, contentBuilder, thoughtBuilder *strings.Builder) (int, bool) {
	if strings.HasPrefix(remaining, "</think>") {
		s.state = StateNormal

		return len("</think>"), false
	}

	if stringsx.IsPartialPrefix(remaining, []string{"</think>"}) {
		s.buffer.WriteString(remaining)

		return 0, true
	}

	_ = contentBuilder
	_ = thoughtBuilder

	return 0, false
}

// processTagDrop handles tag detection in drop state.
func (s *Sanitizer) processTagDrop(remaining string, contentBuilder, thoughtBuilder *strings.Builder) (int, bool) {
	if strings.HasPrefix(remaining, s.dropUntil) {
		matchLen := len(s.dropUntil)
		s.state = StateNormal
		s.dropUntil = ""

		return matchLen, false
	}

	if stringsx.IsPartialPrefix(remaining, []string{s.dropUntil}) {
		s.buffer.WriteString(remaining)

		return 0, true
	}

	_ = contentBuilder
	_ = thoughtBuilder

	return 0, false
}

