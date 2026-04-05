package acp

// Journey: specs/journeys/JOURNEY-R3.1-acp-session-info-update.md.

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestGenerateSessionTitle tests title generation from agent response content.
func TestGenerateSessionTitle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "empty content",
			content: "",
			want:    "",
		},
		{
			name:    "whitespace only",
			content: "   \n\t  ",
			want:    "",
		},
		{
			name:    "short sentence",
			content: "Hello world.",
			want:    "Hello world.",
		},
		{
			name:    "sentence with exclamation",
			content: "Done! The file has been updated.",
			want:    "Done!",
		},
		{
			name:    "sentence with question",
			content: "What would you like? I can help.",
			want:    "What would you like?",
		},
		{
			name:    "first sentence extraction",
			content: "Fix the bug. Then deploy the changes.",
			want:    "Fix the bug.",
		},
		{
			name:    "no sentence boundary uses newline",
			content: "This is a title\nAnd more content here",
			want:    "This is a title",
		},
		{
			name:    "long content truncated at word boundary",
			content: strings.Repeat("word ", 20),
			want:    strings.TrimSpace(strings.Repeat("word ", 16)) + "...",
		},
		{
			name:    "markdown heading stripped",
			content: "## Summary\nThe project is complete.",
			want:    "Summary",
		},
		{
			name:    "no punctuation no newline",
			content: "Short title",
			want:    "Short title",
		},
		{
			name:    "period not followed by space",
			content: "file.txt is ready. Done.",
			want:    "file.txt is ready.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := generateSessionTitle(tt.content)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestGenerateSessionTitle_MaxLength verifies title never exceeds maxTitleLength + ellipsis.
func TestGenerateSessionTitle_MaxLength(t *testing.T) {
	t.Parallel()

	longContent := strings.Repeat("a ", 100)
	title := generateSessionTitle(longContent)

	// maxTitleLength (80) + "..." (3) = 83 max, but truncation at word boundary means it could be less.
	assert.LessOrEqual(t, len(title), maxTitleLength+3)
	assert.NotEmpty(t, title)
}
