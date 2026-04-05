package llmutil

// Journey: specs/journeys/JOURNEY-R21.md.

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestModelContextWindow(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		model string
		want  int
	}{
		{name: "gpt4o", model: "gpt-4o", want: contextWindow128K},
		{name: "claude_opus_4", model: "claude-opus-4", want: contextWindow200K},
		{name: "gemini_2_flash", model: "gemini-2.0-flash", want: contextWindow1M},
		{name: "unknown_returns_zero", model: "totally-unknown-model", want: 0},
		{name: "empty_string", model: "", want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := ModelContextWindow(tt.model)
			require.Equal(t, tt.want, got)
		})
	}
}
