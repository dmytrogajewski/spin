package sanitizer

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizer_Process(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		chunks      []string
		wantContent string
		wantThought string
	}{
		{
			name:        "normal text",
			chunks:      []string{"Hello", " world"},
			wantContent: "Hello world",
			wantThought: "",
		},
		{
			name:        "thinking block",
			chunks:      []string{"<think>", "Thinking...", "</think>"},
			wantContent: "",
			wantThought: "Thinking...",
		},
		{
			name:        "mixed content and thinking",
			chunks:      []string{"Start ", "<think>Hmm</think>", " End"},
			wantContent: "Start  End",
			wantThought: "Hmm",
		},
		{
			name:        "tool call function block dropped",
			chunks:      []string{"Text ", "<function=read_file>", "args", "</function>", " More text"},
			wantContent: "Text  More text",
			wantThought: "",
		},
		{
			name:        "tool call parameter block dropped",
			chunks:      []string{"Text ", "<parameter=path>", "/tmp", "</parameter>", " More text"},
			wantContent: "Text  More text",
			wantThought: "",
		},
		{
			name:        "standalone tool_call dropped",
			chunks:      []string{"End ", "</tool_call>"},
			wantContent: "End ",
			wantThought: "",
		},
		{
			name:        "partial tags across chunks",
			chunks:      []string{"<thi", "nk>Thinking</th", "ink>"},
			wantContent: "",
			wantThought: "Thinking",
		},
		{
			name:        "partial function tag",
			chunks:      []string{"<func", "tion=test>", "content", "</func", "tion>"},
			wantContent: "",
			wantThought: "",
		},
		{
			name:        "complex sequence",
			chunks:      []string{"Let's see.\n", "<think>Analyzing...</think>\n", "<function=ls>\npath\n</function>\n", "Done."},
			wantContent: "Let's see.\n\n\nDone.",
			wantThought: "Analyzing...",
		},
		{
			name:        "false alarm partial tag",
			chunks:      []string{"Use < ", "symbol"},
			wantContent: "Use < symbol",
			wantThought: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			content, thought := processAllChunks(tt.chunks)

			assert.Equal(t, tt.wantContent, content, "content mismatch")
			assert.Equal(t, tt.wantThought, thought, "thought mismatch")
		})
	}
}

// processAllChunks runs all chunks through a fresh Sanitizer and returns accumulated content and thought.
func processAllChunks(chunks []string) (string, string) {
	s := New()

	var contentSb, thoughtSb strings.Builder

	for _, chunk := range chunks {
		c, th := s.Process(chunk)
		contentSb.WriteString(c)
		thoughtSb.WriteString(th)
	}

	return contentSb.String(), thoughtSb.String()
}
