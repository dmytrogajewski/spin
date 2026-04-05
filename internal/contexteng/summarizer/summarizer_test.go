package summarizer

import (
	"context"
	"testing"

	"github.com/dmytrogajewski/spin/internal/message"
)

func TestContentTypeValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		ct       ContentType
		expected string
	}{
		{"conversation", ContentTypeConversation, "conversation"},
		{"tool_output", ContentTypeToolOutput, "tool_output"},
		{"document", ContentTypeDocument, "document"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if string(tt.ct) != tt.expected {
				t.Errorf("ContentType %s = %q, want %q", tt.name, string(tt.ct), tt.expected)
			}
		})
	}
}

func TestSummaryStyleValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		style    SummaryStyle
		expected string
	}{
		{"brief", StyleBrief, "brief"},
		{"detailed", StyleDetailed, "detailed"},
		{"bullet", StyleBullet, "bullet"},
		{"narrative", StyleNarrative, "narrative"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if string(tt.style) != tt.expected {
				t.Errorf("SummaryStyle %s = %q, want %q", tt.name, string(tt.style), tt.expected)
			}
		})
	}
}

func TestOptionsDefaults(t *testing.T) {
	t.Parallel()

	opts := Options{}

	// Zero values should be the defaults.
	if opts.MaxTokens != 0 {
		t.Errorf("MaxTokens = %d, want 0", opts.MaxTokens)
	}

	if opts.TargetRatio != 0 {
		t.Errorf("TargetRatio = %f, want 0", opts.TargetRatio)
	}

	if opts.ContentType != "" {
		t.Errorf("ContentType = %q, want empty", opts.ContentType)
	}

	if opts.Style != "" {
		t.Errorf("Style = %q, want empty", opts.Style)
	}

	if opts.PreserveList != nil {
		t.Errorf("PreserveList = %v, want nil", opts.PreserveList)
	}
}

func TestOptionsWithValues(t *testing.T) {
	t.Parallel()

	const (
		maxTokens   = 500
		targetRatio = 0.3
	)

	opts := Options{
		MaxTokens:    maxTokens,
		TargetRatio:  targetRatio,
		PreserveList: []string{"error", "warning"},
		ContentType:  ContentTypeConversation,
		Style:        StyleNarrative,
	}

	if opts.MaxTokens != maxTokens {
		t.Errorf("MaxTokens = %d, want %d", opts.MaxTokens, maxTokens)
	}

	if opts.TargetRatio != targetRatio {
		t.Errorf("TargetRatio = %f, want %f", opts.TargetRatio, targetRatio)
	}

	if len(opts.PreserveList) != 2 {
		t.Errorf("PreserveList len = %d, want 2", len(opts.PreserveList))
	}

	if opts.ContentType != ContentTypeConversation {
		t.Errorf("ContentType = %q, want %q", opts.ContentType, ContentTypeConversation)
	}

	if opts.Style != StyleNarrative {
		t.Errorf("Style = %q, want %q", opts.Style, StyleNarrative)
	}
}

func TestResultFields(t *testing.T) {
	t.Parallel()

	const (
		originalTokens   = 1000
		summaryTokens    = 300
		compressionRatio = 0.3
	)

	result := Result{
		Summary:          "This is a summary",
		OriginalTokens:   originalTokens,
		SummaryTokens:    summaryTokens,
		CompressionRatio: compressionRatio,
		PreservedItems:   []string{"error"},
		KeyPoints:        []string{"point1", "point2"},
	}

	if result.Summary != "This is a summary" {
		t.Errorf("Summary = %q, want %q", result.Summary, "This is a summary")
	}

	if result.OriginalTokens != originalTokens {
		t.Errorf("OriginalTokens = %d, want %d", result.OriginalTokens, originalTokens)
	}

	if result.SummaryTokens != summaryTokens {
		t.Errorf("SummaryTokens = %d, want %d", result.SummaryTokens, summaryTokens)
	}

	if result.CompressionRatio != compressionRatio {
		t.Errorf("CompressionRatio = %f, want %f", result.CompressionRatio, compressionRatio)
	}

	if len(result.PreservedItems) != 1 {
		t.Errorf("PreservedItems len = %d, want 1", len(result.PreservedItems))
	}

	if len(result.KeyPoints) != 2 {
		t.Errorf("KeyPoints len = %d, want 2", len(result.KeyPoints))
	}
}

func TestMessageResultFields(t *testing.T) {
	t.Parallel()

	const (
		originalCount = 10
		rangeStart    = 0
		rangeEnd      = 9
	)

	result := MessageResult{
		Summary: message.Message{
			Role:    message.RoleAssistant,
			Content: "Summary of conversation",
		},
		OriginalCount:   originalCount,
		SummarizedRange: [2]int{rangeStart, rangeEnd},
		KeyDecisions:    []string{"decision1"},
		KeyActions:      []string{"action1", "action2"},
	}

	if result.Summary.Role != message.RoleAssistant {
		t.Errorf("Summary.Role = %q, want %q", result.Summary.Role, message.RoleAssistant)
	}

	if result.Summary.Content != "Summary of conversation" {
		t.Errorf("Summary.Content = %q, want %q", result.Summary.Content, "Summary of conversation")
	}

	if result.OriginalCount != originalCount {
		t.Errorf("OriginalCount = %d, want %d", result.OriginalCount, originalCount)
	}

	if result.SummarizedRange[0] != rangeStart {
		t.Errorf("SummarizedRange[0] = %d, want %d", result.SummarizedRange[0], rangeStart)
	}

	if result.SummarizedRange[1] != rangeEnd {
		t.Errorf("SummarizedRange[1] = %d, want %d", result.SummarizedRange[1], rangeEnd)
	}

	if len(result.KeyDecisions) != 1 {
		t.Errorf("KeyDecisions len = %d, want 1", len(result.KeyDecisions))
	}

	if len(result.KeyActions) != 2 {
		t.Errorf("KeyActions len = %d, want 2", len(result.KeyActions))
	}
}

// mockSummarizer is a test implementation of Summarizer interface.
type mockSummarizer struct {
	summarizeFunc         func(ctx context.Context, content string, opts Options) (*Result, error)
	summarizeMessagesFunc func(ctx context.Context, messages []message.Message, opts Options) (*MessageResult, error)
}

func (m *mockSummarizer) Summarize(ctx context.Context, content string, opts Options) (*Result, error) {
	if m.summarizeFunc != nil {
		return m.summarizeFunc(ctx, content, opts)
	}

	return &Result{Summary: "mock summary"}, nil
}

func (m *mockSummarizer) SummarizeMessages(ctx context.Context, messages []message.Message, opts Options) (*MessageResult, error) {
	if m.summarizeMessagesFunc != nil {
		return m.summarizeMessagesFunc(ctx, messages, opts)
	}

	return &MessageResult{
		Summary: message.Message{Role: message.RoleAssistant, Content: "mock message summary"},
	}, nil
}

func TestSummarizerInterface(t *testing.T) {
	t.Parallel()

	// Verify mockSummarizer implements Summarizer interface.
	var _ Summarizer = (*mockSummarizer)(nil)

	mock := &mockSummarizer{}
	ctx := context.Background()

	t.Run("Summarize", func(t *testing.T) {
		t.Parallel()

		result, err := mock.Summarize(ctx, "test content", Options{})
		if err != nil {
			t.Fatalf("Summarize error: %v", err)
		}

		if result.Summary != "mock summary" {
			t.Errorf("Summary = %q, want %q", result.Summary, "mock summary")
		}
	})

	t.Run("SummarizeMessages", func(t *testing.T) {
		t.Parallel()

		messages := []message.Message{{Role: message.RoleUser, Content: "test"}}

		result, err := mock.SummarizeMessages(ctx, messages, Options{})
		if err != nil {
			t.Fatalf("SummarizeMessages error: %v", err)
		}

		if result.Summary.Content != "mock message summary" {
			t.Errorf("Summary.Content = %q, want %q", result.Summary.Content, "mock message summary")
		}
	})
}
