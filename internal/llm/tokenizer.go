package llm

import (
	"strings"
	"unicode"
)

// Tokenizer provides token counting functionality for LLM inputs.
// Token counting is essential for managing context windows and costs.
type Tokenizer interface {
	// Count returns the approximate number of tokens in the given text.
	Count(text string) int

	// CountMessages returns the approximate number of tokens for a slice of messages,
	// including message formatting overhead.
	CountMessages(msgs []Message) int
}

// approximateTokenizer implements a simple heuristic-based token counter.
// It uses character-based estimation which tends to overcount slightly,
// providing a conservative estimate suitable for preventing context overflow.
type approximateTokenizer struct {
	// charsPerToken is the average characters per token ratio
	charsPerToken float64
}

// NewApproximateTokenizer creates a tokenizer using character-based heuristics.
// The default ratio of 4 characters per token works well for English text and code.
func NewApproximateTokenizer() Tokenizer {
	return &approximateTokenizer{
		charsPerToken: 4.0,
	}
}

// Count estimates tokens using a character-based heuristic.
// This provides a conservative estimate that slightly overcounts,
// which is safer for context window management.
func (t *approximateTokenizer) Count(text string) int {
	if text == "" {
		return 0
	}

	// Count meaningful characters (exclude some whitespace)
	chars := 0
	for _, r := range text {
		// Count all non-space characters
		if !unicode.IsSpace(r) {
			chars++
		} else {
			// Count spaces but give them less weight
			chars++
		}
	}

	// Adjust for code vs prose
	// Code tends to have more tokens per character due to syntax
	if isLikelyCode(text) {
		chars = int(float64(chars) * 1.2) // 20% adjustment for code
	}

	tokens := int(float64(chars) / t.charsPerToken)

	// Ensure at least 1 token for non-empty text
	if tokens == 0 && chars > 0 {
		tokens = 1
	}

	return tokens
}

// CountMessages calculates total tokens for a message array,
// including message formatting overhead from the chat format.
func (t *approximateTokenizer) CountMessages(msgs []Message) int {
	if len(msgs) == 0 {
		return 0
	}

	total := 0

	// Add overhead for each message (role markers, formatting)
	for _, msg := range msgs {
		// Role overhead: each message has role metadata
		total += getRoleOverhead(msg.Role)

		// Content tokens
		total += t.Count(msg.Content)

		// Tool calls overhead
		for _, tc := range msg.ToolCalls {
			total += t.countToolCall(&tc)
		}

		// Tool response overhead
		if msg.ToolCallID != "" {
			total += 3 // tool_call_id overhead
		}
	}

	// Add conversation-level overhead (varies by provider, conservative estimate)
	total += 3

	return total
}

// countToolCall estimates tokens for a tool call.
func (t *approximateTokenizer) countToolCall(tc *ToolCall) int {
	tokens := 0

	// Tool call structure overhead
	tokens += 5 // {type, id, function} structure

	// Function name
	tokens += len(tc.Function.Name) / 4

	// Arguments (JSON)
	tokens += t.Count(tc.Function.Arguments)

	return tokens
}

// getRoleOverhead returns token overhead for message role formatting.
func getRoleOverhead(role string) int {
	switch role {
	case "system":
		return 4 // <|im_start|>system<|im_sep|>...<|im_end|>
	case "user":
		return 4
	case "assistant":
		return 4
	case "tool":
		return 5 // tool role has additional metadata
	default:
		return 4
	}
}

// isLikelyCode detects if text is likely code vs natural language.
func isLikelyCode(text string) bool {
	if len(text) < 10 {
		return false
	}

	// Sample first 200 chars for efficiency
	sample := text
	if len(text) > 200 {
		sample = text[:200]
	}

	indicators := 0
	total := 0

	// Check for code indicators
	codeChars := []rune{'(', ')', '{', '}', '[', ']', ';', '=', '<', '>'}

	// Strong code keywords (must be whole words, not just substrings)
	keywords := []string{
		"func ", "def ", "class ", "import ", "const ", "var ", "let ", "return ",
		"package ", "export ", "async ", "await ", "function ",
	}

	for _, ch := range sample {
		for _, code := range codeChars {
			if ch == code {
				indicators++
				break
			}
		}
		total++
	}

	// Check for keywords (whole words only to avoid false positives)
	lowerSample := strings.ToLower(sample)
	for _, kw := range keywords {
		if strings.Contains(lowerSample, kw) {
			indicators += 5 // keywords are strong indicators
		}
	}

	// Colon in JSON/YAML is a strong indicator
	if strings.Contains(sample, `":`) || strings.Contains(sample, `": `) {
		indicators += 10 // Strong indicator for JSON
	}

	// Check for common code patterns
	if strings.Contains(sample, "=>") || strings.Contains(sample, "->") {
		indicators += 5
	}

	// If more than 12% of chars are code-like, consider it code
	// Lowered threshold to catch more code patterns
	return float64(indicators)/float64(total) > 0.12
}

// EstimateTokens estimates total tokens for a completion request,
// including messages, tools, and reserved response budget.
func EstimateTokens(req CompletionRequest) int {
	tokenizer := NewApproximateTokenizer()

	total := 0

	// Message tokens
	total += tokenizer.CountMessages(req.Messages)

	// Tool definitions
	for _, tool := range req.Tools {
		total += estimateToolDefinition(&tool)
	}

	// Response budget (maxTokens or default)
	if req.MaxTokens > 0 {
		total += req.MaxTokens
	} else {
		total += 512 // conservative default response budget
	}

	return total
}

// estimateToolDefinition estimates tokens for a tool definition.
func estimateToolDefinition(tool *Tool) int {
	tokens := 0

	// Tool structure overhead
	tokens += 3 // {type, function}

	// Function definition
	tokens += len(tool.Function.Name) / 4
	tokens += len(tool.Function.Description) / 4

	// Parameters schema (JSON Schema is verbose)
	// Rough estimate: 20 tokens per parameter
	// This is a heuristic since we'd need to parse the schema
	tokens += 20

	return tokens
}
