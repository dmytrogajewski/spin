package adapter

import (
	"github.com/dmytrogajewski/spin/internal/contexteng/observation"
	"github.com/dmytrogajewski/spin/internal/message"
)

// ObservationAdapter adapts observation.Summarizer to the harness.ObservationSummarizer
// interface. It walks messages and summarizes tool-role message content.
type ObservationAdapter struct {
	inner *observation.Summarizer
}

// NewObservationAdapter creates an ObservationAdapter wrapping the given summarizer.
// A nil summarizer produces a no-op adapter that returns messages unchanged.
func NewObservationAdapter(s *observation.Summarizer) *ObservationAdapter {
	return &ObservationAdapter{inner: s}
}

// isErrorContent checks if tool output content indicates an error result.
// Delegates to the shared message.IsErrorContent detector.
func isErrorContent(content string) bool {
	return message.IsErrorContent(content)
}

// SummarizeToolResults walks messages and applies per-tool summarization to
// tool-role messages. Non-tool messages are returned unchanged. All message
// fields (ToolCallID, Name, Metadata, etc.) are preserved.
func (a *ObservationAdapter) SummarizeToolResults(
	messages []message.Message,
) []message.Message {
	if a.inner == nil || len(messages) == 0 {
		return messages
	}

	result := make([]message.Message, len(messages))
	copy(result, messages)

	for idx := range result {
		if result[idx].Role != message.RoleTool {
			continue
		}

		// Use dedicated error summarization for failed tool results.
		if isErrorContent(result[idx].Content) {
			result[idx].Content = observation.SummarizeError(result[idx].Content)
		} else {
			result[idx].Content = a.inner.Summarize(
				result[idx].Name, result[idx].Content,
			)
		}
	}

	return result
}
