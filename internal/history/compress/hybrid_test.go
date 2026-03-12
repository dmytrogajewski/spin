package compress

import (
	"context"
	"testing"

	"github.com/dmytrogajewski/spin/internal/context/summarizer"
	"github.com/dmytrogajewski/spin/internal/message"
	"github.com/dmytrogajewski/spin/internal/tokenizer"
)

// mockSummarizer is a test summarizer.
type mockSummarizer struct {
	summarizeMessagesCalled bool
	returnError             error
}

func (m *mockSummarizer) Summarize(_ context.Context, _ string, _ summarizer.Options) (*summarizer.Result, error) {
	return &summarizer.Result{
		Summary:        "Summary of content",
		OriginalTokens: 100,
		SummaryTokens:  20,
	}, m.returnError
}

func (m *mockSummarizer) SummarizeMessages(_ context.Context, msgs []message.Message, _ summarizer.Options) (*summarizer.MessageResult, error) {
	m.summarizeMessagesCalled = true
	if m.returnError != nil {
		return nil, m.returnError
	}

	return &summarizer.MessageResult{
		Summary: message.Message{
			Role:    message.RoleAssistant,
			Content: "[Summary of previous messages]",
		},
		OriginalCount:   len(msgs),
		SummarizedRange: [2]int{0, len(msgs) - 1},
	}, nil
}

func TestNewHybridCompressor(t *testing.T) {
	t.Parallel()

	c := NewHybridCompressor(nil, DefaultCompressorConfig())
	if c == nil {
		t.Fatal("NewHybridCompressor returned nil")
	}

	if c.classifier == nil {
		t.Error("classifier should be set to default")
	}
}

func TestHybridCompressor_Name(t *testing.T) {
	t.Parallel()

	c := NewHybridCompressor(nil, DefaultCompressorConfig())
	if c.Name() != "hybrid" {
		t.Errorf("expected 'hybrid', got %q", c.Name())
	}
}

func TestHybridCompressor_EmptyMessages(t *testing.T) {
	t.Parallel()

	c := NewHybridCompressor(nil, DefaultCompressorConfig())
	tok := &tokenizer.SimpleTokenizer{}

	result, err := c.Compress(context.Background(), []message.Message{}, 1000, tok)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("expected empty result, got %d messages", len(result))
	}
}

func TestHybridCompressor_PreserveCritical(t *testing.T) {
	t.Parallel()

	c := NewHybridCompressor(nil, CompressorConfig{
		PreserveCritical: true,
		MinRetention:     0,
	})
	tok := &tokenizer.SimpleTokenizer{}

	messages := []message.Message{
		{Role: message.RoleUser, Content: "Help me", Tokens: 100},
		{Role: message.RoleAssistant, Content: "Sure, I can help.", Tokens: 100},
		{Role: message.RoleUser, Content: "Thanks", Tokens: 100},
	}

	// Very small budget - should still keep all user messages.
	result, err := c.Compress(context.Background(), messages, 50, tok)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// All user messages (critical) should be preserved.
	userCount := 0

	for _, msg := range result {
		if msg.Role == message.RoleUser {
			userCount++
		}
	}

	if userCount != 2 {
		t.Errorf("expected 2 user messages preserved, got %d", userCount)
	}
}

func TestHybridCompressor_GreedySelection(t *testing.T) {
	t.Parallel()

	c := NewHybridCompressor(nil, CompressorConfig{
		PreserveCritical: false, // Disable to test greedy selection.
		MinRetention:     0,
	})
	tok := &tokenizer.SimpleTokenizer{}

	messages := []message.Message{
		{Role: message.RoleAssistant, Content: "Low priority verbose content here", Tokens: 100},
		{Role: message.RoleAssistant, Content: "```go\ncode block\n```", Tokens: 50}, // High (code).
		{Role: message.RoleAssistant, Content: "Medium priority", Tokens: 30},
	}

	// Budget for ~80 tokens - should prioritize high importance.
	result, err := c.Compress(context.Background(), messages, 80, tok)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should include high importance (code block) message.
	hasCodeBlock := false

	for _, msg := range result {
		if msg.Content == "```go\ncode block\n```" {
			hasCodeBlock = true

			break
		}
	}

	if !hasCodeBlock {
		t.Error("expected high importance code block message to be included")
	}
}

func TestHybridCompressor_ChronologicalOrder(t *testing.T) {
	t.Parallel()

	c := NewHybridCompressor(nil, DefaultCompressorConfig())
	tok := &tokenizer.SimpleTokenizer{}

	messages := []message.Message{
		{Role: message.RoleUser, Content: "First", Tokens: 10},
		{Role: message.RoleUser, Content: "Second", Tokens: 10},
		{Role: message.RoleUser, Content: "Third", Tokens: 10},
	}

	result, err := c.Compress(context.Background(), messages, 1000, tok)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(result))
	}

	// Verify chronological order.
	if result[0].Content != "First" {
		t.Errorf("expected 'First' at index 0, got %q", result[0].Content)
	}

	if result[1].Content != "Second" {
		t.Errorf("expected 'Second' at index 1, got %q", result[1].Content)
	}

	if result[2].Content != "Third" {
		t.Errorf("expected 'Third' at index 2, got %q", result[2].Content)
	}
}

func TestHybridCompressor_MinRetention(t *testing.T) {
	t.Parallel()

	c := NewHybridCompressor(nil, CompressorConfig{
		PreserveCritical: false,
		MinRetention:     0.5, // Keep at least 50%.
	})
	tok := &tokenizer.SimpleTokenizer{}

	// Create verbose messages that would all be low priority.
	messages := make([]message.Message, 10)
	for i := range messages {
		messages[i] = message.Message{
			Role:    message.RoleAssistant,
			Content: "This is message content",
			Tokens:  100,
		}
	}

	// Very small budget.
	result, err := c.Compress(context.Background(), messages, 10, tok)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should keep at least 50% = 5 messages.
	if len(result) < 5 {
		t.Errorf("expected at least 5 messages (50%% retention), got %d", len(result))
	}
}

func TestHybridCompressor_AllCritical(t *testing.T) {
	t.Parallel()

	c := NewHybridCompressor(nil, CompressorConfig{
		PreserveCritical: true,
		MinRetention:     0,
	})
	tok := &tokenizer.SimpleTokenizer{}

	// All critical messages.
	messages := []message.Message{
		{Role: message.RoleUser, Content: "User 1", Tokens: 100},
		{Role: message.RoleUser, Content: "User 2", Tokens: 100},
		{Role: message.RoleUser, Content: "User 3", Tokens: 100},
	}

	// Budget too small for all.
	result, err := c.Compress(context.Background(), messages, 50, tok)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// All critical should be preserved regardless of budget.
	if len(result) != 3 {
		t.Errorf("expected all 3 critical messages preserved, got %d", len(result))
	}
}

func TestHybridCompressor_MixedImportance(t *testing.T) {
	t.Parallel()

	c := NewHybridCompressor(nil, CompressorConfig{
		PreserveCritical: true,
		MinRetention:     0,
	})
	tok := &tokenizer.SimpleTokenizer{}

	messages := []message.Message{
		{Role: message.RoleSystem, Content: "System prompt", Tokens: 50},       // Critical.
		{Role: message.RoleUser, Content: "User question", Tokens: 30},         // Critical.
		{Role: message.RoleAssistant, Content: "Regular response", Tokens: 40}, // Medium.
		{Role: message.RoleAssistant, Content: "```code```", Tokens: 30},       // High.
		{Role: message.RoleUser, Content: "Follow up", Tokens: 20},             // Critical.
	}

	// Budget for critical + some others.
	result, err := c.Compress(context.Background(), messages, 150, tok)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// All critical (system, users) must be present.
	criticalCount := 0

	for _, msg := range result {
		if msg.Role == message.RoleSystem || msg.Role == message.RoleUser {
			criticalCount++
		}
	}

	if criticalCount != 3 {
		t.Errorf("expected 3 critical messages, got %d", criticalCount)
	}
}

func TestHybridCompressor_CompressWithStats(t *testing.T) {
	t.Parallel()

	c := NewHybridCompressor(nil, DefaultCompressorConfig())
	tok := &tokenizer.SimpleTokenizer{}

	messages := []message.Message{
		{Role: message.RoleUser, Content: "Hello", Tokens: 10},
		{Role: message.RoleAssistant, Content: "Hi there! How can I help?", Tokens: 20},
	}

	result, stats, err := c.CompressWithStats(context.Background(), messages, 1000, tok)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("expected 2 messages, got %d", len(result))
	}

	if stats.OriginalCount != 2 {
		t.Errorf("expected OriginalCount 2, got %d", stats.OriginalCount)
	}

	if stats.CompressedCount != 2 {
		t.Errorf("expected CompressedCount 2, got %d", stats.CompressedCount)
	}

	if stats.Strategy != "hybrid" {
		t.Errorf("expected strategy 'hybrid', got %q", stats.Strategy)
	}
}

func TestHybridCompressor_TokenCalculation(t *testing.T) {
	t.Parallel()

	c := NewHybridCompressor(nil, DefaultCompressorConfig())
	tok := &tokenizer.SimpleTokenizer{}

	// Messages without pre-set tokens.
	messages := []message.Message{
		{Role: message.RoleUser, Content: "Short"},
		{Role: message.RoleAssistant, Content: "Also short"},
	}

	_, stats, err := c.CompressWithStats(context.Background(), messages, 1000, tok)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if stats.OriginalTokens == 0 {
		t.Error("expected non-zero OriginalTokens")
	}
}

func TestStats_CompressionRatio(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		original   int
		compressed int
		expected   float64
	}{
		{"no compression", 100, 100, 0.0},
		{"50% compression", 100, 50, 0.5},
		{"zero original", 0, 0, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := Stats{
				OriginalTokens:   tt.original,
				CompressedTokens: tt.compressed,
			}

			ratio := s.CompressionRatio()
			if ratio != tt.expected {
				t.Errorf("expected ratio %f, got %f", tt.expected, ratio)
			}
		})
	}
}

func TestStats_MessageReduction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		original   int
		compressed int
		expected   float64
	}{
		{"no reduction", 10, 10, 0.0},
		{"50% reduction", 10, 5, 0.5},
		{"zero original", 0, 0, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := Stats{
				OriginalCount:   tt.original,
				CompressedCount: tt.compressed,
			}

			reduction := s.MessageReduction()
			if reduction != tt.expected {
				t.Errorf("expected reduction %f, got %f", tt.expected, reduction)
			}
		})
	}
}

func TestDefaultCompressorConfig(t *testing.T) {
	t.Parallel()

	cfg := DefaultCompressorConfig()
	if !cfg.PreserveCritical {
		t.Error("expected PreserveCritical to be true by default")
	}

	if cfg.MinRetention != 0.3 {
		t.Errorf("expected MinRetention 0.3, got %f", cfg.MinRetention)
	}
}

func TestHybridCompressor_ToolResults(t *testing.T) {
	t.Parallel()

	c := NewHybridCompressor(nil, DefaultCompressorConfig())
	tok := &tokenizer.SimpleTokenizer{}

	messages := []message.Message{
		{Role: message.RoleUser, Content: "Read file", Tokens: 10},
		{
			Role:    message.RoleAssistant,
			Content: "Reading file...",
			Tokens:  10,
			ToolCalls: []message.ToolCall{
				{ID: "call_1", Type: "function", Function: message.ToolCallFunction{Name: "read_file"}},
			},
		},
		{Role: message.RoleTool, Content: "File contents...", Tokens: 100, ToolCallID: "call_1"},
	}

	result, err := c.Compress(context.Background(), messages, 50, tok)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// All should be preserved (all critical).
	if len(result) != 3 {
		t.Errorf("expected all 3 messages (all critical), got %d", len(result))
	}
}

func TestHybridCompressor_ErrorMessages(t *testing.T) {
	t.Parallel()

	c := NewHybridCompressor(nil, DefaultCompressorConfig())
	tok := &tokenizer.SimpleTokenizer{}

	messages := []message.Message{
		{Role: message.RoleAssistant, Content: "Error: file not found", Tokens: 20},
		{Role: message.RoleAssistant, Content: "Normal response", Tokens: 20},
	}

	result, err := c.Compress(context.Background(), messages, 30, tok)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Error message should be preserved.
	hasError := false

	for _, msg := range result {
		if msg.Content == "Error: file not found" {
			hasError = true

			break
		}
	}

	if !hasError {
		t.Error("expected error message to be preserved")
	}
}

func TestHybridCompressor_WithSummarizer(t *testing.T) {
	t.Parallel()

	c := NewHybridCompressor(nil, CompressorConfig{
		PreserveCritical: false,
		MinRetention:     0,
	})
	mock := &mockSummarizer{}
	c.WithSummarizer(mock)

	if c.Name() != "hybrid-summarizing" {
		t.Errorf("expected name 'hybrid-summarizing', got %q", c.Name())
	}

	tok := &tokenizer.SimpleTokenizer{}

	// Create messages where some will be removed.
	messages := []message.Message{
		{Role: message.RoleAssistant, Content: "Message 1", Tokens: 50},
		{Role: message.RoleAssistant, Content: "Message 2", Tokens: 50},
		{Role: message.RoleAssistant, Content: "Message 3", Tokens: 50},
	}

	// Small budget to force removal.
	result, err := c.Compress(context.Background(), messages, 60, tok)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have called summarizer for removed messages.
	if !mock.summarizeMessagesCalled {
		t.Error("expected summarizer to be called for removed messages")
	}

	// Result should include summary.
	hasSummary := false

	for _, msg := range result {
		if msg.Content == "[Summary of previous messages]" {
			hasSummary = true

			break
		}
	}

	if !hasSummary {
		t.Error("expected summary message in result")
	}
}

func TestHybridCompressor_WithSummarizer_Error(t *testing.T) {
	t.Parallel()

	c := NewHybridCompressor(nil, CompressorConfig{
		PreserveCritical: false,
		MinRetention:     0,
	})
	mock := &mockSummarizer{returnError: context.DeadlineExceeded}
	c.WithSummarizer(mock)

	tok := &tokenizer.SimpleTokenizer{}

	messages := []message.Message{
		{Role: message.RoleAssistant, Content: "Message 1", Tokens: 50},
		{Role: message.RoleAssistant, Content: "Message 2", Tokens: 50},
	}

	// Should not fail even if summarizer errors.
	result, err := c.Compress(context.Background(), messages, 60, tok)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should still have some result (just without summary).
	if len(result) == 0 {
		t.Error("expected non-empty result even on summarizer error")
	}
}

func TestHybridCompressor_WithSummarizer_NoRemoved(t *testing.T) {
	t.Parallel()

	c := NewHybridCompressor(nil, DefaultCompressorConfig())
	mock := &mockSummarizer{}
	c.WithSummarizer(mock)

	tok := &tokenizer.SimpleTokenizer{}

	// All critical messages - none will be removed.
	messages := []message.Message{
		{Role: message.RoleUser, Content: "Hello", Tokens: 10},
	}

	_, err := c.Compress(context.Background(), messages, 1000, tok)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Summarizer should NOT be called when nothing removed.
	if mock.summarizeMessagesCalled {
		t.Error("summarizer should not be called when no messages removed")
	}
}

func TestHybridCompressor_EnforceMinRetention_AlreadySufficient(t *testing.T) {
	t.Parallel()

	c := NewHybridCompressor(nil, CompressorConfig{
		PreserveCritical: true,
		MinRetention:     0.3,
	})
	tok := &tokenizer.SimpleTokenizer{}

	// All critical messages.
	messages := []message.Message{
		{Role: message.RoleUser, Content: "1", Tokens: 10},
		{Role: message.RoleUser, Content: "2", Tokens: 10},
		{Role: message.RoleUser, Content: "3", Tokens: 10},
	}

	result, err := c.Compress(context.Background(), messages, 1000, tok)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// All should be kept (already > min retention).
	if len(result) != 3 {
		t.Errorf("expected 3 messages, got %d", len(result))
	}
}

func TestHybridCompressor_EnforceMinRetention_TooManyInClassified(t *testing.T) {
	t.Parallel()

	c := NewHybridCompressor(nil, CompressorConfig{
		PreserveCritical: false,
		MinRetention:     0.9, // Very high min retention.
	})
	tok := &tokenizer.SimpleTokenizer{}

	// 2 messages, min retention is 0.9 = need at least 1.8 -> 1.
	messages := []message.Message{
		{Role: message.RoleAssistant, Content: "1", Tokens: 10},
		{Role: message.RoleAssistant, Content: "2", Tokens: 10},
	}

	result, err := c.Compress(context.Background(), messages, 15, tok)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have at least 1 message.
	if len(result) < 1 {
		t.Errorf("expected at least 1 message, got %d", len(result))
	}
}
