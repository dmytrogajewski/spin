package compress

import (
	"context"

	"github.com/dmytrogajewski/spin/internal/message"
	"github.com/dmytrogajewski/spin/pkg/tokenizer"
)

const defaultMinRetention = 0.3

// Compressor compresses conversation history to fit within token budget.
type Compressor interface {
	// Compress reduces messages to fit target token count.
	// Returns compressed message slice or error.
	Compress(
		ctx context.Context,
		messages []message.Message,
		targetTokens int,
		tok tokenizer.Tokenizer,
	) ([]message.Message, error)

	// Name returns the compressor strategy name.
	Name() string
}

// CompressorConfig configures compression behavior.
type CompressorConfig struct {
	// PreserveCritical ensures critical messages are always kept,
	// even if that exceeds the token budget.
	PreserveCritical bool

	// MinRetention is the minimum fraction of messages to keep (0.0-1.0).
	// For example, 0.3 means at least 30% of messages are preserved.
	MinRetention float64
}

// DefaultCompressorConfig returns sensible default configuration.
func DefaultCompressorConfig() CompressorConfig {
	return CompressorConfig{
		PreserveCritical: true,
		MinRetention:     defaultMinRetention,
	}
}

// Stats contains compression statistics.
type Stats struct {
	// OriginalCount is the number of messages before compression.
	OriginalCount int

	// CompressedCount is the number of messages after compression.
	CompressedCount int

	// OriginalTokens is the token count before compression.
	OriginalTokens int

	// CompressedTokens is the token count after compression.
	CompressedTokens int

	// Summarized indicates if LLM summarization was used.
	Summarized bool

	// Strategy is the compression strategy name.
	Strategy string
}

// CompressionRatio returns the ratio of tokens removed (0.0-1.0).
func (s Stats) CompressionRatio() float64 {
	if s.OriginalTokens == 0 {
		return 0
	}

	return 1.0 - float64(s.CompressedTokens)/float64(s.OriginalTokens)
}

// MessageReduction returns the fraction of messages removed (0.0-1.0).
func (s Stats) MessageReduction() float64 {
	if s.OriginalCount == 0 {
		return 0
	}

	return 1.0 - float64(s.CompressedCount)/float64(s.OriginalCount)
}
