package compress

import (
	"context"
)

// Tokenizer is the minimal interface needed for token counting.
// This avoids circular imports with core package.
type Tokenizer interface {
	Count(text string) int
}

// CompressibleMessage is a simplified message type for compression.
// This avoids circular imports with the core package.
type CompressibleMessage struct {
	ID            string
	Role          string
	Content       string
	ToolCallCount int
	Tokens        int
}

// GetRole implements Message interface
func (m CompressibleMessage) GetRole() string {
	return m.Role
}

// GetContent implements Message interface
func (m CompressibleMessage) GetContent() string {
	return m.Content
}

// GetToolCallCount implements Message interface
func (m CompressibleMessage) GetToolCallCount() int {
	return m.ToolCallCount
}

// Compressor compresses conversation history to fit within token budgets.
//
// Different compression strategies can be implemented by satisfying this interface.
// The compressor receives a slice of messages and a target token count, and returns
// a compressed slice that fits within the budget while preserving important information.
type Compressor interface {
	// Compress reduces the message slice to fit within targetTokens.
	//
	// Parameters:
	//   - ctx: Context for cancellation
	//   - messages: Input messages to compress
	//   - targetTokens: Maximum token count for output
	//   - tokenizer: Token counting implementation
	//
	// Returns:
	//   - Compressed message slice (may be shorter than input)
	//   - Error if compression fails
	//
	// The returned slice must maintain chronological order.
	Compress(ctx context.Context, messages []CompressibleMessage, targetTokens int, tokenizer Tokenizer) ([]CompressibleMessage, error)

	// Name returns the compressor strategy name for logging and configuration.
	Name() string
}
