package tokenizer

import "strings"

// Tokenizer provides token counting for messages.
//
// Different LLM providers may use different tokenization schemes.
// This interface allows pluggable token counting implementations.
type Tokenizer interface {
	// Count returns the estimated token count for the given text
	Count(text string) int

	// CountMessages returns the total token count for a slice of messages
	CountMessages(messages []interface{}) int
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
func (t *SimpleTokenizer) CountMessages(messages []interface{}) int {
	total := 0

	for _, msg := range messages {
		// Content tokens - assuming message has Content field
		// This is a simplified implementation for interface{}
		contentTokens := 0
		if msgMap, ok := msg.(map[string]interface{}); ok {
			if content, ok := msgMap["content"].(string); ok {
				contentTokens = t.Count(content)
			}
		}
		total += contentTokens

		// Message overhead (role, formatting, etc.)
		total += 4

		// Tool call overhead if present
		if msgMap, ok := msg.(map[string]interface{}); ok {
			if toolCalls, ok := msgMap["tool_calls"].([]interface{}); ok && len(toolCalls) > 0 {
				for _, tc := range toolCalls {
					if tcMap, ok := tc.(map[string]interface{}); ok {
						if function, ok := tcMap["function"].(map[string]interface{}); ok {
							if name, ok := function["name"].(string); ok {
								total += t.Count(name)
							}
							if args, ok := function["arguments"].(string); ok {
								total += t.Count(args)
							}
						}
					}
					total += 8 // Tool call formatting overhead
				}
			}
		}
	}

	return total
}
