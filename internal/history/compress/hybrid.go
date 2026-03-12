package compress

import (
	"context"
	"sort"

	"github.com/dmytrogajewski/spin/internal/context/summarizer"
	"github.com/dmytrogajewski/spin/internal/message"
	"github.com/dmytrogajewski/spin/internal/tokenizer"
)

// HybridCompressor uses importance-weighted greedy selection
// with optional LLM summarization for removed content.
type HybridCompressor struct {
	classifier *Classifier
	summarizer summarizer.Summarizer
	config     CompressorConfig
}

// NewHybridCompressor creates a new hybrid compressor.
func NewHybridCompressor(classifier *Classifier, config CompressorConfig) *HybridCompressor {
	if classifier == nil {
		classifier = NewClassifier()
	}

	return &HybridCompressor{
		classifier: classifier,
		config:     config,
	}
}

// WithSummarizer adds an optional summarizer for semantic compression.
// When set, removed messages are summarized instead of discarded.
func (c *HybridCompressor) WithSummarizer(s summarizer.Summarizer) *HybridCompressor {
	c.summarizer = s

	return c
}

// Name returns the compressor strategy name.
func (c *HybridCompressor) Name() string {
	if c.summarizer != nil {
		return "hybrid-summarizing"
	}

	return "hybrid"
}

// Compress implements importance-weighted greedy selection.
func (c *HybridCompressor) Compress(
	ctx context.Context,
	messages []message.Message,
	targetTokens int,
	tok tokenizer.Tokenizer,
) ([]message.Message, error) {
	if len(messages) == 0 {
		return messages, nil
	}

	// 1. Classify all messages.
	classified := c.classifyMessages(messages, tok)

	// 2. Sort by importance (stable sort preserves chronological order within same importance).
	sort.SliceStable(classified, func(i, j int) bool {
		if classified[i].Importance == classified[j].Importance {
			return classified[i].Index < classified[j].Index
		}

		return classified[i].Importance > classified[j].Importance
	})

	// 3. Greedy selection.
	selected, removed := c.selectMessages(classified, targetTokens)

	// 4. Optionally summarize removed messages.
	if c.summarizer != nil && len(removed) > 0 {
		summaryMsg, err := c.summarizeRemoved(ctx, removed)
		if err == nil && summaryMsg != nil {
			// Count summary tokens.
			summaryMsg.Tokens = tok.Count(summaryMsg.Content) + 4
			selected = append(selected, ClassifiedMessage{
				Message:    *summaryMsg,
				Importance: ImportanceMedium,
				Tokens:     summaryMsg.Tokens,
				Index:      -1, // Placed at the beginning during chronological sort.
			})
		}
		// On error, just proceed without summary (best-effort).
	}

	// 5. Enforce minimum retention.
	selected = c.enforceMinRetention(selected, classified, len(messages))

	// 6. Restore chronological order.
	sort.SliceStable(selected, func(i, j int) bool {
		// Summary messages (Index -1) go first.
		if selected[i].Index == -1 {
			return true
		}

		if selected[j].Index == -1 {
			return false
		}

		return selected[i].Index < selected[j].Index
	})

	// 7. Extract messages.
	result := make([]message.Message, len(selected))
	for i, cm := range selected {
		result[i] = cm.Message
	}

	return result, nil
}

// classifyMessages assigns importance to each message.
func (c *HybridCompressor) classifyMessages(messages []message.Message, tok tokenizer.Tokenizer) []ClassifiedMessage {
	classified := make([]ClassifiedMessage, len(messages))
	for i, msg := range messages {
		tokens := msg.Tokens
		if tokens == 0 {
			tokens = tok.Count(msg.Content) + 4
		}

		classified[i] = ClassifiedMessage{
			Message:    msg,
			Importance: c.classifier.Classify(msg),
			Tokens:     tokens,
			Index:      i,
		}
	}

	return classified
}

// selectMessages performs greedy selection based on importance.
func (c *HybridCompressor) selectMessages(
	classified []ClassifiedMessage,
	targetTokens int,
) (selected, removed []ClassifiedMessage) {
	selected = make([]ClassifiedMessage, 0, len(classified))
	removed = make([]ClassifiedMessage, 0)
	tokensUsed := 0

	for _, cm := range classified {
		// Always include critical if config says so.
		if c.config.PreserveCritical && cm.Importance == ImportanceCritical {
			selected = append(selected, cm)
			tokensUsed += cm.Tokens

			continue
		}

		// Include if within budget.
		if tokensUsed+cm.Tokens <= targetTokens {
			selected = append(selected, cm)
			tokensUsed += cm.Tokens
		} else {
			removed = append(removed, cm)
		}
	}

	return selected, removed
}

// summarizeRemoved creates a summary of removed messages.
func (c *HybridCompressor) summarizeRemoved(
	ctx context.Context,
	removed []ClassifiedMessage,
) (*message.Message, error) {
	// Sort removed by original index for coherent summary.
	sort.SliceStable(removed, func(i, j int) bool {
		return removed[i].Index < removed[j].Index
	})

	// Extract messages.
	msgs := make([]message.Message, len(removed))
	for i, cm := range removed {
		msgs[i] = cm.Message
	}

	// Calculate target tokens (compress to ~20% of original).
	totalTokens := 0
	for _, cm := range removed {
		totalTokens += cm.Tokens
	}

	targetTokens := max(totalTokens/5, 100)

	// Use summarizer.
	result, err := c.summarizer.SummarizeMessages(ctx, msgs, summarizer.Options{
		MaxTokens:   targetTokens,
		TargetRatio: 0.2,
		Style:       summarizer.StyleNarrative,
		ContentType: summarizer.ContentTypeConversation,
	})
	if err != nil {
		return nil, err
	}

	return &result.Summary, nil
}

// enforceMinRetention ensures minimum message retention.
func (c *HybridCompressor) enforceMinRetention(
	selected, classified []ClassifiedMessage,
	originalCount int,
) []ClassifiedMessage {
	minMessages := int(float64(originalCount) * c.config.MinRetention)

	if len(selected) >= minMessages || minMessages >= len(classified) {
		return selected
	}

	// Need to add more messages - take most recent ones not already selected.
	selectedIndices := make(map[int]bool)

	for _, cm := range selected {
		if cm.Index >= 0 {
			selectedIndices[cm.Index] = true
		}
	}

	// Sort classified by index (most recent last).
	sortedByIndex := make([]ClassifiedMessage, len(classified))
	copy(sortedByIndex, classified)
	sort.SliceStable(sortedByIndex, func(i, j int) bool {
		return sortedByIndex[i].Index > sortedByIndex[j].Index
	})

	// Add recent messages until we reach minimum.
	for _, cm := range sortedByIndex {
		if len(selected) >= minMessages {
			break
		}

		if !selectedIndices[cm.Index] {
			selected = append(selected, cm)
			selectedIndices[cm.Index] = true
		}
	}

	return selected
}

// CompressWithStats returns compression statistics along with results.
func (c *HybridCompressor) CompressWithStats(
	ctx context.Context,
	messages []message.Message,
	targetTokens int,
	tok tokenizer.Tokenizer,
) ([]message.Message, *Stats, error) {
	// Calculate original stats.
	originalTokens := 0

	for _, msg := range messages {
		tokens := msg.Tokens
		if tokens == 0 {
			tokens = tok.Count(msg.Content) + 4
		}

		originalTokens += tokens
	}

	// Perform compression.
	result, err := c.Compress(ctx, messages, targetTokens, tok)
	if err != nil {
		return nil, nil, err
	}

	// Calculate compressed stats.
	compressedTokens := 0

	for _, msg := range result {
		tokens := msg.Tokens
		if tokens == 0 {
			tokens = tok.Count(msg.Content) + 4
		}

		compressedTokens += tokens
	}

	stats := &Stats{
		OriginalCount:    len(messages),
		CompressedCount:  len(result),
		OriginalTokens:   originalTokens,
		CompressedTokens: compressedTokens,
		Summarized:       c.summarizer != nil,
		Strategy:         c.Name(),
	}

	return result, stats, nil
}
