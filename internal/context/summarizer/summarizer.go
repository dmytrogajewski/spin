package summarizer

import (
	"context"

	"github.com/dmytrogajewski/spin/internal/message"
)

// ContentType represents the type of content being summarized.
type ContentType string

// Content types for summarization.
const (
	ContentTypeConversation ContentType = "conversation"
	ContentTypeToolOutput   ContentType = "tool_output"
	ContentTypeDocument     ContentType = "document"
)

// SummaryStyle represents the style of summary output.
type SummaryStyle string

// Summary styles.
const (
	StyleBrief     SummaryStyle = "brief"     // Minimal, key points only
	StyleDetailed  SummaryStyle = "detailed"  // More context preserved
	StyleBullet    SummaryStyle = "bullet"    // Bullet point format
	StyleNarrative SummaryStyle = "narrative" // Flowing narrative
)

// Options configures summarization behavior.
type Options struct {
	// MaxTokens is the target output size in tokens.
	MaxTokens int

	// TargetRatio is the target compression ratio (e.g., 0.3 = 30% of original).
	TargetRatio float64

	// PreserveList contains items to preserve verbatim in the summary.
	PreserveList []string

	// ContentType indicates the type of content being summarized.
	ContentType ContentType

	// Style is the desired summary style.
	Style SummaryStyle
}

// Result contains the summarization output and metadata.
type Result struct {
	// Summary is the condensed content.
	Summary string

	// OriginalTokens is the token count of the original content.
	OriginalTokens int

	// SummaryTokens is the token count of the summary.
	SummaryTokens int

	// CompressionRatio is the ratio of summary to original tokens.
	CompressionRatio float64

	// PreservedItems contains items that were preserved verbatim.
	PreservedItems []string

	// KeyPoints contains extracted key points from the content.
	KeyPoints []string
}

// MessageResult contains the result of summarizing messages.
type MessageResult struct {
	// Summary is the single summary message.
	Summary message.Message

	// OriginalCount is the number of messages summarized.
	OriginalCount int

	// SummarizedRange contains the indices of summarized messages [start, end].
	SummarizedRange [2]int

	// KeyDecisions contains important decisions preserved from the conversation.
	KeyDecisions []string

	// KeyActions contains actions taken during the conversation.
	KeyActions []string
}

// Summarizer defines the interface for content summarization.
type Summarizer interface {
	// Summarize compresses content while preserving key information.
	Summarize(ctx context.Context, content string, opts Options) (*Result, error)

	// SummarizeMessages summarizes a sequence of conversation messages.
	SummarizeMessages(ctx context.Context, messages []message.Message, opts Options) (*MessageResult, error)
}
