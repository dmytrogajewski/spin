package core

import "strings"

// Tokenizer provides token counting for messages.
//
// Different LLM providers may use different tokenization schemes.
// This interface allows pluggable token counting implementations.
type Tokenizer interface {
	// Count returns the estimated token count for the given text
	Count(text string) int

	// CountMessages returns the total token count for a slice of messages
	CountMessages(messages []Message) int
}

// SimpleTokenizer provides a basic word-based token estimation.
//
// This is a fallback tokenizer that estimates tokens based on word count.
// It uses a rough approximation of ~1.3 tokens per word plus overhead per message.
//
// For production use with specific LLM providers, implement proper tokenizers
// (e.g., tiktoken for OpenAI models).
type SimpleTokenizer struct{}

// Count estimates token count based on word count.
//
// This uses a simple heuristic: words * 1.3, which is a rough approximation
// of English text tokenization in many LLM models.
func (t *SimpleTokenizer) Count(text string) int {
	if text == "" {
		return 0
	}

	// Count words
	words := strings.Fields(text)
	wordCount := len(words)

	// Approximate: 1.3 tokens per word
	tokens := int(float64(wordCount) * 1.3)

	// Minimum 1 token for non-empty text
	if tokens == 0 && text != "" {
		tokens = 1
	}

	return tokens
}

// CountMessages estimates total token count for a slice of messages.
//
// This adds per-message overhead (approximately 4 tokens) in addition to
// the content tokens, to account for message formatting overhead in the API.
func (t *SimpleTokenizer) CountMessages(messages []Message) int {
	total := 0

	for _, msg := range messages {
		// Content tokens
		contentTokens := t.Count(msg.Content)
		total += contentTokens

		// Message overhead (role, formatting, etc.)
		total += 4

		// Tool call overhead if present
		if len(msg.ToolCalls) > 0 {
			for _, tc := range msg.ToolCalls {
				total += t.Count(tc.Function.Name)
				total += t.Count(tc.Function.Arguments)
				total += 8 // Tool call formatting overhead
			}
		}
	}

	return total
}
