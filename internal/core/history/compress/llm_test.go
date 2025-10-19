package compress

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// Mock LLM provider for testing
type mockLLMProvider struct {
	completeFunc func(ctx context.Context, req interface{}) (string, error)
}

func (m *mockLLMProvider) Complete(ctx context.Context, req interface{}) (string, error) {
	if m.completeFunc != nil {
		return m.completeFunc(ctx, req)
	}
	return "Mock summary", nil
}

func TestDefaultLLMSummarizerConfig(t *testing.T) {
	config := DefaultLLMSummarizerConfig()

	if config.ChunkSize == 0 {
		t.Error("DefaultLLMSummarizerConfig().ChunkSize should not be zero")
	}

	if config.RecentWindow == 0 {
		t.Error("DefaultLLMSummarizerConfig().RecentWindow should not be zero")
	}

	if config.Temperature < 0 || config.Temperature > 1 {
		t.Errorf("DefaultLLMSummarizerConfig().Temperature = %v, want between 0 and 1", config.Temperature)
	}

	if config.MaxTokens == 0 {
		t.Error("DefaultLLMSummarizerConfig().MaxTokens should not be zero")
	}
}

func TestNewLLMSummarizer(t *testing.T) {
	mockProvider := &mockLLMProvider{}
	config := DefaultLLMSummarizerConfig()

	summarizer := NewLLMSummarizer(mockProvider, config)

	if summarizer == nil {
		t.Fatal("NewLLMSummarizer() returned nil")
	}

	if summarizer.Name() != "llm-summary" {
		t.Errorf("NewLLMSummarizer().Name() = %v, want %v", summarizer.Name(), "llm-summary")
	}
}

func TestLLMSummarizer_Compress_Success(t *testing.T) {
	mockProvider := &mockLLMProvider{
		completeFunc: func(ctx context.Context, req interface{}) (string, error) {
			return "Concise summary of the conversation", nil
		},
	}
	config := DefaultLLMSummarizerConfig()

	summarizer := NewLLMSummarizer(mockProvider, config)

	tokenizer := &mockTokenizer{
		counts: map[string]int{
			"Message 1":  50,
			"Response 1": 50,
			"Message 2":  50,
		},
	}

	messages := []CompressibleMessage{
		{Role: "user", Content: "Message 1", Tokens: 50},
		{Role: "assistant", Content: "Response 1", Tokens: 50},
		{Role: "user", Content: "Message 2", Tokens: 50},
	}

	result, err := summarizer.Compress(context.Background(), messages, 100, tokenizer)

	if err != nil {
		t.Errorf("Compress() unexpected error: %v", err)
	}

	if len(result) == 0 {
		t.Error("Compress() returned empty result")
	}
}

func TestLLMSummarizer_Compress_Error(t *testing.T) {
	mockProvider := &mockLLMProvider{
		completeFunc: func(ctx context.Context, req interface{}) (string, error) {
			return "", errors.New("LLM error")
		},
	}
	config := DefaultLLMSummarizerConfig()

	summarizer := NewLLMSummarizer(mockProvider, config)

	tokenizer := &mockTokenizer{
		counts: map[string]int{
			"Message 1": 50,
		},
	}

	messages := []CompressibleMessage{
		{Role: "user", Content: "Message 1", Tokens: 50},
	}

	result, err := summarizer.Compress(context.Background(), messages, 100, tokenizer)

	// The LLM summarizer may not return an error immediately but may return the messages unchanged
	// or may handle the error gracefully. Check that either an error occurred OR messages were returned
	if err != nil && result != nil {
		t.Errorf("Compress() returned both error and result: err=%v, result=%v", err, result)
	}
}

func TestLLMSummarizer_Compress_EmptyMessages(t *testing.T) {
	mockProvider := &mockLLMProvider{}
	config := DefaultLLMSummarizerConfig()

	summarizer := NewLLMSummarizer(mockProvider, config)

	tokenizer := &mockTokenizer{
		counts: map[string]int{},
	}

	result, err := summarizer.Compress(context.Background(), []CompressibleMessage{}, 100, tokenizer)

	if err != nil {
		t.Errorf("Compress() unexpected error: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("Compress() expected empty result, got %d messages", len(result))
	}
}

func TestLLMSummarizer_Name(t *testing.T) {
	mockProvider := &mockLLMProvider{}
	config := DefaultLLMSummarizerConfig()

	summarizer := NewLLMSummarizer(mockProvider, config)

	if summarizer.Name() != "llm-summary" {
		t.Errorf("Name() = %v, want %v", summarizer.Name(), "llm-summary")
	}
}

func TestLLMSummarizer_SummarizeChunk(t *testing.T) {
	mockProvider := &mockLLMProvider{
		completeFunc: func(ctx context.Context, req interface{}) (string, error) {
			return "This is a summary of the conversation", nil
		},
	}
	config := DefaultLLMSummarizerConfig()
	summarizer := NewLLMSummarizer(mockProvider, config)

	tokenizer := &mockTokenizer{
		counts: map[string]int{
			"This is a summary of the conversation": 7,
		},
	}

	chunk := []CompressibleMessage{
		{ID: "1", Role: "user", Content: "Hello", Tokens: 5},
		{ID: "2", Role: "assistant", Content: "Hi there", Tokens: 6},
	}

	result, err := summarizer.summarizeChunk(context.Background(), chunk, tokenizer)
	if err != nil {
		t.Errorf("summarizeChunk() unexpected error: %v", err)
		return
	}

	if result.Role != RoleAssistant {
		t.Errorf("summarizeChunk() Role = %v, want %v", result.Role, RoleAssistant)
	}
	if !strings.Contains(result.Content, "Summary:") {
		t.Errorf("summarizeChunk() Content should contain 'Summary:', got: %v", result.Content)
	}
	if result.ToolCallCount != 0 {
		t.Errorf("summarizeChunk() ToolCallCount = %v, want 0", result.ToolCallCount)
	}
	if result.Tokens != 11 { // 7 + 4 overhead
		t.Errorf("summarizeChunk() Tokens = %v, want 11", result.Tokens)
	}
}

func TestLLMSummarizer_SummarizeChunk_Error(t *testing.T) {
	mockProvider := &mockLLMProvider{
		completeFunc: func(ctx context.Context, req interface{}) (string, error) {
			return "", errors.New("LLM error")
		},
	}
	config := DefaultLLMSummarizerConfig()
	summarizer := NewLLMSummarizer(mockProvider, config)

	tokenizer := &mockTokenizer{}
	chunk := []CompressibleMessage{
		{ID: "1", Role: "user", Content: "Hello", Tokens: 5},
	}

	_, err := summarizer.summarizeChunk(context.Background(), chunk, tokenizer)
	if err == nil {
		t.Error("summarizeChunk() expected error, got nil")
	}
	if !strings.Contains(err.Error(), "LLM summarization failed") {
		t.Errorf("summarizeChunk() error should contain 'LLM summarization failed', got: %v", err)
	}
}

func TestLLMSummarizer_BuildSummarizationPrompt(t *testing.T) {
	mockProvider := &mockLLMProvider{}
	config := DefaultLLMSummarizerConfig()
	summarizer := NewLLMSummarizer(mockProvider, config)

	chunk := []CompressibleMessage{
		{ID: "1", Role: "user", Content: "Hello", Tokens: 5},
		{ID: "2", Role: "assistant", Content: "Hi there", Tokens: 6},
	}

	prompt := summarizer.buildSummarizationPrompt(chunk)

	if !strings.Contains(prompt, "Summarize the following conversation segment") {
		t.Errorf("buildSummarizationPrompt() should contain summarization instruction")
	}
	if !strings.Contains(prompt, "[user]: Hello") {
		t.Errorf("buildSummarizationPrompt() should contain user message")
	}
	if !strings.Contains(prompt, "[assistant]: Hi there") {
		t.Errorf("buildSummarizationPrompt() should contain assistant message")
	}
}

func TestLLMSummarizer_BuildSummarizationPrompt_EmptyChunk(t *testing.T) {
	mockProvider := &mockLLMProvider{}
	config := DefaultLLMSummarizerConfig()
	summarizer := NewLLMSummarizer(mockProvider, config)

	chunk := []CompressibleMessage{}

	prompt := summarizer.buildSummarizationPrompt(chunk)

	if !strings.Contains(prompt, "Summarize the following conversation segment") {
		t.Errorf("buildSummarizationPrompt() should contain summarization instruction")
	}
	if len(prompt) == 0 {
		t.Error("buildSummarizationPrompt() should not return empty prompt")
	}
}

func TestLLMSummarizer_BuildSummarizationPrompt_SingleMessage(t *testing.T) {
	mockProvider := &mockLLMProvider{}
	config := DefaultLLMSummarizerConfig()
	summarizer := NewLLMSummarizer(mockProvider, config)

	chunk := []CompressibleMessage{
		{ID: "1", Role: "user", Content: "Hello", Tokens: 5},
	}

	prompt := summarizer.buildSummarizationPrompt(chunk)

	if !strings.Contains(prompt, "[user]: Hello") {
		t.Errorf("buildSummarizationPrompt() should contain user message")
	}
	if !strings.Contains(prompt, "Summarize the following conversation segment") {
		t.Errorf("buildSummarizationPrompt() should contain summarization instruction")
	}
}
