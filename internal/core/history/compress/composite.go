package compress

import (
	"context"
	"fmt"
)

// CompositeCompressor chains multiple compression strategies with fallback.
//
// The composite tries strategies in order until one succeeds:
//  1. Primary strategy (e.g., LLM summarization)
//  2. Fallback strategy (e.g., hybrid compression)
//
// This allows using advanced strategies with graceful degradation when
// they fail (e.g., LLM API errors, timeouts).
type CompositeCompressor struct {
	primary  Compressor
	fallback Compressor
}

// NewLLMWithHybridFallback creates a composite with LLM summarization as primary
// and hybrid compression as fallback.
//
// This is the recommended configuration for production use:
// - LLM summarization provides better semantic preservation
// - Hybrid compression ensures reliability when LLM unavailable
func NewLLMWithHybridFallback(llm LLMProvider, config LLMSummarizerConfig) *CompositeCompressor {
	primary := NewLLMSummarizer(llm, config)
	fallback := NewDefaultHybridCompressor()

	return &CompositeCompressor{
		primary:  primary,
		fallback: fallback,
	}
}

// NewDefaultLLMWithHybridFallback creates composite with default LLM config.
func NewDefaultLLMWithHybridFallback(llm LLMProvider) *CompositeCompressor {
	return NewLLMWithHybridFallback(llm, DefaultLLMSummarizerConfig())
}

// Compress implements the Compressor interface with fallback behavior.
//
// Algorithm:
//  1. Try primary compressor
//  2. If primary succeeds, return result
//  3. If primary fails, log error and try fallback
//  4. Return fallback result (or error if both fail)
func (c *CompositeCompressor) Compress(
	ctx context.Context,
	messages []CompressibleMessage,
	targetTokens int,
	tokenizer Tokenizer,
) ([]CompressibleMessage, error) {
	// Try primary strategy
	result, err := c.primary.Compress(ctx, messages, targetTokens, tokenizer)
	if err == nil {
		// Primary succeeded
		return result, nil
	}

	// Primary failed - use fallback
	// In production, this would log the primary error
	_ = err // Suppress unused error (would be logged)

	// Try fallback strategy
	result, err = c.fallback.Compress(ctx, messages, targetTokens, tokenizer)
	if err != nil {
		return nil, fmt.Errorf("both primary (%s) and fallback (%s) compression failed: %w",
			c.primary.Name(), c.fallback.Name(), err)
	}

	return result, nil
}

// Name returns the compressor strategy name.
func (c *CompositeCompressor) Name() string {
	return fmt.Sprintf("composite(%s→%s)", c.primary.Name(), c.fallback.Name())
}
