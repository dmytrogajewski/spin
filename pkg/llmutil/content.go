package llmutil

import (
	"encoding/json"
	"strings"
)

// contentPart represents a single part in the OpenAI content array format.
type contentPart struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// ExtractContent decodes content from an OpenAI-compatible JSON value that
// is either a plain string or an array of content part objects.
// Returns an empty string for null, malformed, or empty input.
func ExtractContent(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	// Try plain string first (most common case).
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return text
	}

	// Try array of content parts.
	var parts []contentPart
	if err := json.Unmarshal(raw, &parts); err == nil {
		return joinTextParts(parts)
	}

	return ""
}

// joinTextParts concatenates text from content parts of type "text".
func joinTextParts(parts []contentPart) string {
	var builder strings.Builder

	for _, part := range parts {
		if part.Type == "text" {
			builder.WriteString(part.Text)
		}
	}

	return builder.String()
}
