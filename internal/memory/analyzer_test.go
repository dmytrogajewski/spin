package memory

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultContextAnalyzer_Analyze_EmptyMessages(t *testing.T) {
	analyzer := NewDefaultContextAnalyzer()
	candidates := analyzer.Analyze([]AnalyzableMessage{})
	assert.Empty(t, candidates)
}

func TestDefaultContextAnalyzer_Analyze_NoOffloadableContent(t *testing.T) {
	analyzer := NewDefaultContextAnalyzer()
	messages := []AnalyzableMessage{
		{Role: "user", Content: "Hello, how are you?"},
		{Role: "assistant", Content: "I'm doing well, thanks!"},
	}

	candidates := analyzer.Analyze(messages)
	assert.Empty(t, candidates)
}

func TestDefaultContextAnalyzer_Analyze_LargeCodeBlock(t *testing.T) {
	analyzer := NewDefaultContextAnalyzer()
	analyzer.CodeBlockThreshold = 50 // Lower threshold for testing.

	largeCode := "```go\n" + strings.Repeat("func example() {\n\treturn nil\n}\n", 20) + "```"
	messages := []AnalyzableMessage{
		{Role: "assistant", Content: "Here's the code:\n" + largeCode},
	}

	candidates := analyzer.Analyze(messages)
	assert.Len(t, candidates, 1)
	assert.Equal(t, "Large code block", candidates[0].Reason)
	assert.Equal(t, ScopeSession, candidates[0].Destination)
	assert.Equal(t, 100, candidates[0].Priority)
}

func TestDefaultContextAnalyzer_Analyze_LargeToolOutput(t *testing.T) {
	analyzer := NewDefaultContextAnalyzer()
	analyzer.ToolOutputThreshold = 100 // Lower threshold for testing.

	largeOutput := strings.Repeat("Output line\n", 50)
	messages := []AnalyzableMessage{
		{Role: "tool", Content: largeOutput},
	}

	candidates := analyzer.Analyze(messages)
	assert.Len(t, candidates, 1)
	assert.Equal(t, "Large tool output", candidates[0].Reason)
	assert.Equal(t, ScopeSession, candidates[0].Destination)
	assert.Equal(t, 80, candidates[0].Priority)
}

func TestDefaultContextAnalyzer_Analyze_Decision(t *testing.T) {
	analyzer := NewDefaultContextAnalyzer()
	messages := []AnalyzableMessage{
		{Role: "assistant", Content: "After reviewing the options, I decided to use JWT for authentication."},
	}

	candidates := analyzer.Analyze(messages)
	assert.Len(t, candidates, 1)
	assert.Equal(t, "Important decision", candidates[0].Reason)
	assert.Equal(t, ScopePersistent, candidates[0].Destination)
	assert.Equal(t, 50, candidates[0].Priority)
	assert.Contains(t, candidates[0].Content, "decided to use JWT")
}

func TestDefaultContextAnalyzer_Analyze_MultipleCodeBlocks(t *testing.T) {
	analyzer := NewDefaultContextAnalyzer()
	analyzer.CodeBlockThreshold = 20 // Low threshold.

	content := "First block:\n```go\n" + strings.Repeat("x", 100) + "\n```\n" +
		"Second block:\n```python\n" + strings.Repeat("y", 100) + "\n```"
	messages := []AnalyzableMessage{
		{Role: "assistant", Content: content},
	}

	candidates := analyzer.Analyze(messages)
	assert.Len(t, candidates, 2)
}

func TestExtractCodeBlocks(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected int
	}{
		{
			name:     "no code blocks",
			content:  "Just regular text",
			expected: 0,
		},
		{
			name:     "single code block",
			content:  "```go\nfunc main() {}\n```",
			expected: 1,
		},
		{
			name:     "multiple code blocks",
			content:  "```go\ncode1\n```\ntext\n```python\ncode2\n```",
			expected: 2,
		},
		{
			name:     "code block without language",
			content:  "```\nplain code\n```",
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocks := extractCodeBlocks(tt.content)
			assert.Len(t, blocks, tt.expected)
		})
	}
}

func TestExtractDecision(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		hasMatch bool
	}{
		{
			name:     "decided to",
			content:  "I decided to use Redis for caching.",
			hasMatch: true,
		},
		{
			name:     "decision colon",
			content:  "Decision: We will use PostgreSQL.",
			hasMatch: true,
		},
		{
			name:     "will use",
			content:  "We will use TypeScript for the frontend.",
			hasMatch: true,
		},
		{
			name:     "chose to",
			content:  "The team chose to implement REST over GraphQL.",
			hasMatch: true,
		},
		{
			name:     "no decision",
			content:  "This is just a regular message.",
			hasMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision := extractDecision(tt.content)
			if tt.hasMatch {
				assert.NotEmpty(t, decision)
			} else {
				assert.Empty(t, decision)
			}
		})
	}
}

func TestEstimateTokens(t *testing.T) {
	// Simple heuristic: ~4 chars per token.
	assert.Equal(t, 25, estimateTokens(strings.Repeat("a", 100)))
	assert.Equal(t, 0, estimateTokens(""))
	assert.Equal(t, 1, estimateTokens("hello")) // 5/4 = 1
}

func TestGenerateKey(t *testing.T) {
	key := generateKey("code", 1, 2)
	assert.NotEmpty(t, key)
	assert.Contains(t, key, "code")
}
