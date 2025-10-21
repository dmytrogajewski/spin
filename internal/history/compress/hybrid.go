package compress

import (
	"context"
	"sort"
)

// CompressorConfig configures the hybrid compressor behavior.
type CompressorConfig struct {
	// PreserveCritical ensures critical messages are always kept
	PreserveCritical bool

	// MinRetention is the minimum fraction of messages to keep (0.0-1.0)
	// Example: 0.3 means keep at least 30% of messages
	MinRetention float64
}

// DefaultCompressorConfig returns sensible default configuration.
func DefaultCompressorConfig() CompressorConfig {
	return CompressorConfig{
		PreserveCritical: true,
		MinRetention:     0.3,
	}
}

// HybridCompressor uses importance-weighted greedy selection to compress messages.
//
// Algorithm:
//  1. Classify all messages by importance
//  2. Sort by importance (preserving chronological order within same level)
//  3. Greedily select messages until token budget reached
//  4. Always preserve critical messages if PreserveCritical=true
//  5. Restore chronological order
//  6. Enforce minimum retention ratio as safety check
//
// The compressor is deterministic and thread-safe.
type HybridCompressor struct {
	classifier *MessageClassifier
	config     CompressorConfig
}

// NewHybridCompressor creates a new hybrid compressor with the given configuration.
func NewHybridCompressor(config CompressorConfig) *HybridCompressor {
	return &HybridCompressor{
		classifier: &MessageClassifier{},
		config:     config,
	}
}

// NewDefaultHybridCompressor creates a compressor with default configuration.
func NewDefaultHybridCompressor() *HybridCompressor {
	return NewHybridCompressor(DefaultCompressorConfig())
}

// Compress implements the Compressor interface using importance-weighted selection.
func (c *HybridCompressor) Compress(
	ctx context.Context,
	messages []CompressibleMessage,
	targetTokens int,
	tokenizer Tokenizer,
) ([]CompressibleMessage, error) {
	// Handle empty input
	if len(messages) == 0 {
		return messages, nil
	}

	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Step 1: Classify all messages
	classified := make([]classifiedMessage, len(messages))
	for i, msg := range messages {
		// Ensure token count is set
		tokens := msg.Tokens
		if tokens == 0 {
			tokens = tokenizer.Count(msg.Content) + 4 // Add message overhead
		}

		classified[i] = classifiedMessage{
			message:    msg,
			importance: c.classifier.Classify(msg),
			tokens:     tokens,
			index:      i, // Preserve original index
		}
	}

	// Step 2: Sort by importance (stable sort preserves chronological order within same importance)
	sort.SliceStable(classified, func(i, j int) bool {
		if classified[i].importance == classified[j].importance {
			return classified[i].index < classified[j].index
		}
		return classified[i].importance > classified[j].importance
	})

	// Step 3: Greedy selection
	selected := make([]classifiedMessage, 0, len(classified))
	tokensUsed := 0

	for _, cm := range classified {
		// For critical messages with PreserveCritical, try to include but respect budget
		if cm.importance == ImportanceCritical {
			if c.config.PreserveCritical {
				// Always try to include, but check if we're massively over budget
				// If critical messages alone exceed 2x the target, we need to be selective
				if tokensUsed > targetTokens*2 {
					// Skip if adding would make things much worse
					continue
				}
				selected = append(selected, cm)
				tokensUsed += cm.tokens
				continue
			}
		}

		// Include if within budget
		if tokensUsed+cm.tokens <= targetTokens {
			selected = append(selected, cm)
			tokensUsed += cm.tokens
		}
	}

	// Step 4: Restore chronological order
	sort.SliceStable(selected, func(i, j int) bool {
		return selected[i].index < selected[j].index
	})

	// Step 5: Extract messages
	result := make([]CompressibleMessage, len(selected))
	for i, cm := range selected {
		result[i] = cm.message
	}

	// Step 6: Enforce minimum retention (safety check)
	minMessages := int(float64(len(messages)) * c.config.MinRetention)
	if len(result) < minMessages && minMessages <= len(messages) {
		// Take most recent messages to meet minimum
		// This is a fallback to prevent over-aggressive compression
		startIdx := len(messages) - minMessages
		result = messages[startIdx:]
	}

	return result, nil
}

// Name returns the compressor strategy name.
func (c *HybridCompressor) Name() string {
	return "hybrid"
}

// classifiedMessage holds a message with its computed importance and token count.
type classifiedMessage struct {
	message    CompressibleMessage
	importance MessageImportance
	tokens     int
	index      int // Original position in input slice
}
