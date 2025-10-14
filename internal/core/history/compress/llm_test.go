package compress

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// mockLLMProvider for testing
type mockLLMProvider struct {
	response string
	err      error
	calls    int
}

func (m *mockLLMProvider) Complete(ctx context.Context, req interface{}) (string, error) {
	m.calls++
	if m.err != nil {
		return "", m.err
	}
	return m.response, nil
}

func TestLLMSummarizer_Compress(t *testing.T) {
	mock := &mockLLMProvider{
		response: "Summary of the conversation segment",
	}

	// Use smaller recent window to ensure summarization happens
	config := LLMSummarizerConfig{
		ChunkSize:    10,
		RecentWindow: 5, // Keep last 5, summarize the rest
		Temperature:  0.3,
		MaxTokens:    200,
	}

	summarizer := NewLLMSummarizer(mock, config)
	tokenizer := &simpleTokenizer{}
	ctx := context.Background()

	// Create 30 messages: 2 critical + 28 non-critical
	messages := []CompressibleMessage{
		{Role: RoleUser, Content: "Critical question 1", Tokens: 50},
	}

	// Add 28 non-critical verbose messages (will be split into old + recent)
	for i := 0; i < 28; i++ {
		messages = append(messages, CompressibleMessage{
			Role:    RoleAssistant,
			Content: fmt.Sprintf("Verbose response %d: %s", i, strings.Repeat("detail ", 50)),
			Tokens:  300,
		})
	}

	messages = append(messages, CompressibleMessage{
		Role: RoleUser, Content: "Critical question 2", Tokens: 50,
	})

	// Target: 500 tokens (should trigger summarization)
	compressed, err := summarizer.Compress(ctx, messages, 500, tokenizer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify critical messages preserved
	criticalCount := 0
	for _, msg := range compressed {
		if msg.Role == RoleUser {
			criticalCount++
		}
	}

	if criticalCount != 2 {
		t.Errorf("expected 2 critical user messages, got %d", criticalCount)
	}

	// Verify LLM was called for summarization
	// With 28 non-critical, recent window 5, we should summarize 23 messages
	// That's at least 2 chunks (23/10 = 2.3), so at least 2 calls
	if mock.calls == 0 {
		t.Errorf("expected LLM to be called for summarization, got %d calls", mock.calls)
	}
}

func TestLLMSummarizer_NoCompressionNeeded(t *testing.T) {
	mock := &mockLLMProvider{
		response: "Summary",
	}

	summarizer := NewDefaultLLMSummarizer(mock)
	tokenizer := &simpleTokenizer{}
	ctx := context.Background()

	messages := []CompressibleMessage{
		{Role: RoleUser, Content: "Short message", Tokens: 50},
	}

	// Target: 1000 tokens (no compression needed)
	compressed, err := summarizer.Compress(ctx, messages, 1000, tokenizer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(compressed) != len(messages) {
		t.Errorf("expected no compression, got %d messages (original: %d)", len(compressed), len(messages))
	}

	// Verify LLM was NOT called
	if mock.calls > 0 {
		t.Errorf("expected LLM not to be called when no compression needed, got %d calls", mock.calls)
	}
}

func TestLLMSummarizer_PreserveCritical(t *testing.T) {
	mock := &mockLLMProvider{
		response: "Summarized content",
	}

	summarizer := NewDefaultLLMSummarizer(mock)
	tokenizer := &simpleTokenizer{}
	ctx := context.Background()

	// All critical messages
	messages := []CompressibleMessage{
		{ID: "1", Role: RoleUser, Content: "Question 1", Tokens: 50},
		{ID: "2", Role: RoleTool, Content: "Tool result 1", Tokens: 50},
		{ID: "3", Role: RoleUser, Content: "Question 2", Tokens: 50},
		{ID: "4", Role: RoleTool, Content: "Tool result 2", Tokens: 50},
	}

	compressed, err := summarizer.Compress(ctx, messages, 100, tokenizer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// All critical should be preserved (even if over budget)
	if len(compressed) != 4 {
		t.Errorf("expected all 4 critical messages preserved, got %d", len(compressed))
	}

	// Verify LLM was NOT called (no non-critical messages to summarize)
	if mock.calls > 0 {
		t.Errorf("expected LLM not to be called for critical-only messages")
	}
}

func TestLLMSummarizer_RecentWindow(t *testing.T) {
	mock := &mockLLMProvider{
		response: "Summary of old messages",
	}

	config := LLMSummarizerConfig{
		ChunkSize:    5,
		RecentWindow: 3, // Keep last 3 messages verbatim
		Temperature:  0.3,
		MaxTokens:    200,
	}

	summarizer := NewLLMSummarizer(mock, config)
	tokenizer := &simpleTokenizer{}
	ctx := context.Background()

	// Create 10 non-critical messages
	messages := make([]CompressibleMessage, 10)
	for i := 0; i < 10; i++ {
		messages[i] = CompressibleMessage{
			Role:    RoleAssistant,
			Content: fmt.Sprintf("Message %d", i),
			Tokens:  50,
		}
	}

	compressed, err := summarizer.Compress(ctx, messages, 200, tokenizer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have: 1-2 summaries (for first 7 messages) + 3 recent
	// Exact count depends on token budget and summary size
	if len(compressed) >= len(messages) {
		t.Errorf("expected compression, got %d messages (original: %d)", len(compressed), len(messages))
	}

	// Verify LLM was called
	if mock.calls == 0 {
		t.Errorf("expected LLM to be called for summarization")
	}
}

func TestLLMSummarizer_LLMError(t *testing.T) {
	mock := &mockLLMProvider{
		err: fmt.Errorf("LLM API error"),
	}

	summarizer := NewDefaultLLMSummarizer(mock)
	tokenizer := &simpleTokenizer{}
	ctx := context.Background()

	messages := []CompressibleMessage{
		{Role: RoleAssistant, Content: "Message 1", Tokens: 100},
		{Role: RoleAssistant, Content: "Message 2", Tokens: 100},
		{Role: RoleAssistant, Content: "Message 3", Tokens: 100},
	}

	compressed, err := summarizer.Compress(ctx, messages, 150, tokenizer)
	if err != nil {
		t.Fatalf("unexpected error (should fallback on LLM error): %v", err)
	}

	// Should fallback to keeping original messages or using hybrid
	if len(compressed) == 0 {
		t.Errorf("expected fallback to preserve some messages")
	}
}

func TestLLMSummarizer_ChunkSizeRespected(t *testing.T) {
	mock := &mockLLMProvider{
		response: "Chunk summary",
	}

	config := LLMSummarizerConfig{
		ChunkSize:    3, // Summarize 3 messages at a time
		RecentWindow: 0, // No recent window
		Temperature:  0.3,
		MaxTokens:    200,
	}

	summarizer := NewLLMSummarizer(mock, config)
	tokenizer := &simpleTokenizer{}
	ctx := context.Background()

	// Create 10 non-critical messages
	messages := make([]CompressibleMessage, 10)
	for i := 0; i < 10; i++ {
		messages[i] = CompressibleMessage{
			Role:    RoleAssistant,
			Content: fmt.Sprintf("Message %d", i),
			Tokens:  100,
		}
	}

	_, err := summarizer.Compress(ctx, messages, 200, tokenizer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// With 10 messages and chunk size 3, should have: 3 + 3 + 3 + 1 = 4 chunks
	// So 4 LLM calls expected
	expectedCalls := 4
	if mock.calls != expectedCalls {
		t.Errorf("expected %d LLM calls (chunkSize=3, 10 messages), got %d", expectedCalls, mock.calls)
	}
}

func TestLLMSummarizer_EmptyMessages(t *testing.T) {
	mock := &mockLLMProvider{
		response: "Summary",
	}

	summarizer := NewDefaultLLMSummarizer(mock)
	tokenizer := &simpleTokenizer{}
	ctx := context.Background()

	compressed, err := summarizer.Compress(ctx, []CompressibleMessage{}, 1000, tokenizer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(compressed) != 0 {
		t.Errorf("expected empty result, got %d messages", len(compressed))
	}

	// Verify LLM was NOT called
	if mock.calls > 0 {
		t.Errorf("expected no LLM calls for empty input")
	}
}

func TestLLMSummarizer_ContextCancellation(t *testing.T) {
	mock := &mockLLMProvider{
		response: "Summary",
	}

	summarizer := NewDefaultLLMSummarizer(mock)
	tokenizer := &simpleTokenizer{}

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	messages := []CompressibleMessage{
		{Role: RoleAssistant, Content: "Message", Tokens: 100},
	}

	_, err := summarizer.Compress(ctx, messages, 50, tokenizer)
	if err == nil {
		t.Errorf("expected context cancellation error")
	}

	if err != context.Canceled {
		t.Errorf("expected context.Canceled error, got: %v", err)
	}
}

// Benchmark LLM summarization
func BenchmarkLLMSummarizer_100Messages(b *testing.B) {
	mock := &mockLLMProvider{
		response: "Summary of conversation",
	}

	summarizer := NewDefaultLLMSummarizer(mock)
	tokenizer := &simpleTokenizer{}
	ctx := context.Background()

	messages := generateTestMessages(100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = summarizer.Compress(ctx, messages, 3000, tokenizer)
	}
}
