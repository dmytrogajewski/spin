package core

import (
	"strings"
	"testing"
)

func TestSimpleTokenizer_Count(t *testing.T) {
	tokenizer := &SimpleTokenizer{}

	tests := []struct {
		name     string
		text     string
		expected int
	}{
		{
			name:     "empty string",
			text:     "",
			expected: 0,
		},
		{
			name:     "single word",
			text:     "hello",
			expected: 1, // 1 * 1.3 = 1.3 -> 1
		},
		{
			name:     "two words",
			text:     "hello world",
			expected: 2, // 2 * 1.3 = 2.6 -> 2
		},
		{
			name:     "three words",
			text:     "hello world test",
			expected: 3, // 3 * 1.3 = 3.9 -> 3
		},
		{
			name:     "multiple words",
			text:     "this is a longer sentence with more words",
			expected: 10, // 8 * 1.3 = 10.4 -> 10
		},
		{
			name:     "whitespace only",
			text:     "   \t\n  ",
			expected: 1, // Minimum 1 token for non-empty text
		},
		{
			name:     "punctuation",
			text:     "hello, world!",
			expected: 2, // 2 * 1.3 = 2.6 -> 2
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tokenizer.Count(tt.text)
			if result != tt.expected {
				t.Errorf("Count() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSimpleTokenizer_CountMessages(t *testing.T) {
	tokenizer := &SimpleTokenizer{}

	tests := []struct {
		name     string
		messages []Message
		expected int
	}{
		{
			name:     "empty messages",
			messages: []Message{},
			expected: 0,
		},
		{
			name: "single message",
			messages: []Message{
				{
					Content: "hello world",
					Role:    RoleUser,
				},
			},
			expected: 6, // 2 content tokens + 4 message overhead
		},
		{
			name: "multiple messages",
			messages: []Message{
				{
					Content: "hello",
					Role:    RoleUser,
				},
				{
					Content: "hi there",
					Role:    RoleAssistant,
				},
			},
			expected: 11, // (1 + 4) + (2 + 4) = 11
		},
		{
			name: "message with tool calls",
			messages: []Message{
				{
					Content: "execute command",
					Role:    RoleUser,
					ToolCalls: []ToolCall{
						{
							ID: "call_1",
							Function: ToolCallFunction{
								Name:      "execute_command",
								Arguments: `{"command": "ls"}`,
							},
						},
					},
				},
			},
			expected: 17, // 2 content + 4 message + 11 tool call + 8 tool overhead
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tokenizer.CountMessages(tt.messages)
			if result != tt.expected {
				t.Errorf("CountMessages() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSimpleTokenizer_CountMessages_ToolCallOverhead(t *testing.T) {
	tokenizer := &SimpleTokenizer{}

	message := Message{
		Content: "test",
		Role:    RoleAssistant,
		ToolCalls: []ToolCall{
			{
				ID: "call_1",
				Function: ToolCallFunction{
					Name:      "tool_name",
					Arguments: `{"arg": "value"}`,
				},
			},
			{
				ID: "call_2",
				Function: ToolCallFunction{
					Name:      "another_tool",
					Arguments: `{"param": "data"}`,
				},
			},
		},
	}

	result := tokenizer.CountMessages([]Message{message})

	// Content: 1 token
	// Message overhead: 4 tokens
	// Tool call 1: 1 (name) + 3 (args) + 8 (overhead) = 12
	// Tool call 2: 2 (name) + 3 (args) + 8 (overhead) = 13
	// Total: 1 + 4 + 10 + 12 = 27
	expected := 27

	if result != expected {
		t.Errorf("CountMessages() with tool calls = %v, want %v", result, expected)
	}
}

func TestSimpleTokenizer_EdgeCases(t *testing.T) {
	tokenizer := &SimpleTokenizer{}

	t.Run("very long text", func(t *testing.T) {
		longText := "word " + strings.Repeat("word ", 1000)
		result := tokenizer.Count(longText)
		if result <= 0 {
			t.Errorf("Count() for long text should be positive, got %v", result)
		}
	})

	t.Run("unicode text", func(t *testing.T) {
		unicodeText := "こんにちは 世界"
		result := tokenizer.Count(unicodeText)
		if result <= 0 {
			t.Errorf("Count() for unicode text should be positive, got %v", result)
		}
	})

	t.Run("mixed content", func(t *testing.T) {
		mixedText := "Hello 123 world! @#$%"
		result := tokenizer.Count(mixedText)
		if result <= 0 {
			t.Errorf("Count() for mixed content should be positive, got %v", result)
		}
	})
}
