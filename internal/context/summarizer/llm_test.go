package summarizer

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/message"
	"github.com/dmytrogajewski/spin/internal/tokenizer"
)

func TestDefaultLLMSummarizerConfig(t *testing.T) {
	config := DefaultLLMSummarizerConfig()

	const expectedModel = "gpt-4o-mini"
	const expectedTimeout = 10 * time.Second
	const expectedMaxTokens = 500
	const expectedTargetRatio = 0.3

	if config.Model != expectedModel {
		t.Errorf("Model = %q, want %q", config.Model, expectedModel)
	}
	if config.Timeout != expectedTimeout {
		t.Errorf("Timeout = %v, want %v", config.Timeout, expectedTimeout)
	}
	if config.DefaultMaxTokens != expectedMaxTokens {
		t.Errorf("DefaultMaxTokens = %d, want %d", config.DefaultMaxTokens, expectedMaxTokens)
	}
	if config.DefaultTargetRatio != expectedTargetRatio {
		t.Errorf("DefaultTargetRatio = %f, want %f", config.DefaultTargetRatio, expectedTargetRatio)
	}
	if config.DefaultStyle != StyleNarrative {
		t.Errorf("DefaultStyle = %q, want %q", config.DefaultStyle, StyleNarrative)
	}
}

func TestNewLLMSummarizer(t *testing.T) {
	provider := llm.NewMockProvider("test-summarizer")
	tok := &tokenizer.SimpleTokenizer{}
	config := DefaultLLMSummarizerConfig()

	summarizer := NewLLMSummarizer(provider, tok, config)

	if summarizer == nil {
		t.Fatal("NewLLMSummarizer returned nil")
	}

	// Verify it implements Summarizer interface
	var _ Summarizer = summarizer
}

func TestNewLLMSummarizerWithNilTokenizer(t *testing.T) {
	provider := llm.NewMockProvider("test-summarizer")
	config := DefaultLLMSummarizerConfig()

	// Should not panic, should use default tokenizer
	summarizer := NewLLMSummarizer(provider, nil, config)

	if summarizer == nil {
		t.Fatal("NewLLMSummarizer returned nil")
	}
}

func TestLLMSummarizer_Summarize_EmptyContent(t *testing.T) {
	provider := llm.NewMockProvider("test-summarizer")
	config := DefaultLLMSummarizerConfig()
	s := NewLLMSummarizer(provider, nil, config)

	ctx := context.Background()
	result, err := s.Summarize(ctx, "", Options{})

	if err != nil {
		t.Fatalf("Summarize error: %v", err)
	}
	if result.Summary != "" {
		t.Errorf("Summary = %q, want empty", result.Summary)
	}
	if result.OriginalTokens != 0 {
		t.Errorf("OriginalTokens = %d, want 0", result.OriginalTokens)
	}
	if result.SummaryTokens != 0 {
		t.Errorf("SummaryTokens = %d, want 0", result.SummaryTokens)
	}
	const expectedRatio = 1.0
	if result.CompressionRatio != expectedRatio {
		t.Errorf("CompressionRatio = %f, want %f", result.CompressionRatio, expectedRatio)
	}
}

func TestLLMSummarizer_Summarize_BasicContent(t *testing.T) {
	const mockResponse = "This is a summary of the content."
	provider := llm.NewMockProvider("test-summarizer", llm.WithResponse(mockResponse))
	config := DefaultLLMSummarizerConfig()
	s := NewLLMSummarizer(provider, nil, config)

	ctx := context.Background()
	content := "This is a long piece of content that needs to be summarized. It contains many words and sentences that could be condensed."
	result, err := s.Summarize(ctx, content, Options{MaxTokens: 100})

	if err != nil {
		t.Fatalf("Summarize error: %v", err)
	}
	if result.Summary != mockResponse {
		t.Errorf("Summary = %q, want %q", result.Summary, mockResponse)
	}
	if result.OriginalTokens <= 0 {
		t.Errorf("OriginalTokens = %d, want > 0", result.OriginalTokens)
	}
	if result.SummaryTokens <= 0 {
		t.Errorf("SummaryTokens = %d, want > 0", result.SummaryTokens)
	}
}

func TestLLMSummarizer_SummarizeMessages_EmptyMessages(t *testing.T) {
	provider := llm.NewMockProvider("test-summarizer")
	config := DefaultLLMSummarizerConfig()
	s := NewLLMSummarizer(provider, nil, config)

	ctx := context.Background()
	result, err := s.SummarizeMessages(ctx, nil, Options{})

	if err != nil {
		t.Fatalf("SummarizeMessages error: %v", err)
	}
	if result.Summary.Content != "" {
		t.Errorf("Summary.Content = %q, want empty", result.Summary.Content)
	}
	if result.OriginalCount != 0 {
		t.Errorf("OriginalCount = %d, want 0", result.OriginalCount)
	}
}

func TestLLMSummarizer_SummarizeMessages_BasicMessages(t *testing.T) {
	const mockResponse = "User asked about authentication. Assistant explained the auth flow."
	provider := llm.NewMockProvider("test-summarizer", llm.WithResponse(mockResponse))
	config := DefaultLLMSummarizerConfig()
	s := NewLLMSummarizer(provider, nil, config)

	ctx := context.Background()
	messages := []message.Message{
		{Role: message.RoleUser, Content: "How does authentication work?"},
		{Role: message.RoleAssistant, Content: "Authentication uses JWT tokens..."},
		{Role: message.RoleUser, Content: "Can you show an example?"},
	}
	result, err := s.SummarizeMessages(ctx, messages, Options{MaxTokens: 100})

	if err != nil {
		t.Fatalf("SummarizeMessages error: %v", err)
	}
	if !strings.Contains(result.Summary.Content, mockResponse) {
		t.Errorf("Summary.Content = %q, want to contain %q", result.Summary.Content, mockResponse)
	}
	if result.Summary.Role != message.RoleAssistant {
		t.Errorf("Summary.Role = %q, want %q", result.Summary.Role, message.RoleAssistant)
	}
	const expectedCount = 3
	if result.OriginalCount != expectedCount {
		t.Errorf("OriginalCount = %d, want %d", result.OriginalCount, expectedCount)
	}
	if result.SummarizedRange[0] != 0 {
		t.Errorf("SummarizedRange[0] = %d, want 0", result.SummarizedRange[0])
	}
	const expectedRangeEnd = 2
	if result.SummarizedRange[1] != expectedRangeEnd {
		t.Errorf("SummarizedRange[1] = %d, want %d", result.SummarizedRange[1], expectedRangeEnd)
	}
}
