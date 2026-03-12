// Package tokenizer provides text tokenization utilities.
package tokenizer

import "strings"

const tokensPerWordRatio = 1.3

// Tokenizer provides token counting for text.
//
// Different LLM providers may use different tokenization schemes.
// This interface allows pluggable token counting implementations.
type Tokenizer interface {
	// Count returns the estimated token count for the given text.
	Count(text string) int
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

	// Count words.
	words := strings.Fields(text)
	wordCount := len(words)

	// Approximate: 1.3 tokens per word.
	tokens := int(float64(wordCount) * tokensPerWordRatio)

	// Minimum 1 token for non-empty text.
	if tokens == 0 && text != "" {
		tokens = 1
	}

	return tokens
}
