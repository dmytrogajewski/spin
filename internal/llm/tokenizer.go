package llm

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
