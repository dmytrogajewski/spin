package history

import (
	"context"
	"testing"

	"github.com/dmytrogajewski/spin/internal/history/compress"
	"github.com/dmytrogajewski/spin/internal/message"
	"github.com/dmytrogajewski/spin/internal/tokenizer"
)

func TestNewHistory(t *testing.T) {
	t.Parallel()

	h := NewHistory(4096, &tokenizer.SimpleTokenizer{})
	if h == nil {
		t.Fatal("NewHistory returned nil")
	}

	if h.maxTokens != 4096 {
		t.Errorf("expected maxTokens 4096, got %d", h.maxTokens)
	}
}

func TestNewHistory_NilTokenizer(t *testing.T) {
	t.Parallel()

	h := NewHistory(4096, nil)
	if h == nil {
		t.Fatal("NewHistory returned nil")
	}

	if h.tokenizer == nil {
		t.Error("tokenizer should default to SimpleTokenizer")
	}
}

func TestNewHistoryWithDefaults(t *testing.T) {
	t.Parallel()

	h := NewHistoryWithDefaults()
	if h == nil {
		t.Fatal("NewHistoryWithDefaults returned nil")
	}

	if h.maxTokens != 4096 {
		t.Errorf("expected default maxTokens 4096, got %d", h.maxTokens)
	}
}

func TestHistory_AddMessage(t *testing.T) {
	t.Parallel()

	h := NewHistory(4096, &tokenizer.SimpleTokenizer{})

	err := h.AddMessage(context.Background(), message.Message{
		Role:    message.RoleUser,
		Content: "Hello",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	msgs := h.Messages()
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}

	if msgs[0].Role != message.RoleUser {
		t.Errorf("expected role user, got %s", msgs[0].Role)
	}

	if msgs[0].ID == "" {
		t.Error("message ID should be generated")
	}

	if msgs[0].Tokens == 0 {
		t.Error("message tokens should be calculated")
	}
}

func TestHistory_AddMessage_EmptyRole(t *testing.T) {
	t.Parallel()

	h := NewHistory(4096, &tokenizer.SimpleTokenizer{})

	err := h.AddMessage(context.Background(), message.Message{
		Content: "Hello",
	})
	if err == nil {
		t.Fatal("expected error for empty role")
	}
}

func TestHistory_AddSystemMessage(t *testing.T) {
	t.Parallel()

	h := NewHistory(4096, &tokenizer.SimpleTokenizer{})

	err := h.AddSystemMessage(context.Background(), "You are helpful.")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	msgs := h.Messages()
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}

	if msgs[0].Role != message.RoleSystem {
		t.Errorf("expected role system, got %s", msgs[0].Role)
	}
}

func TestHistory_AddUserMessage(t *testing.T) {
	t.Parallel()

	h := NewHistory(4096, &tokenizer.SimpleTokenizer{})

	err := h.AddUserMessage(context.Background(), "Hello!")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	msgs := h.Messages()
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}

	if msgs[0].Role != message.RoleUser {
		t.Errorf("expected role user, got %s", msgs[0].Role)
	}
}

func TestHistory_TokenCount(t *testing.T) {
	t.Parallel()

	h := NewHistory(4096, &tokenizer.SimpleTokenizer{})

	_ = h.AddMessage(context.Background(), message.Message{
		Role:    message.RoleUser,
		Content: "Hello",
		Tokens:  10,
	})
	_ = h.AddMessage(context.Background(), message.Message{
		Role:    message.RoleAssistant,
		Content: "Hi there!",
		Tokens:  20,
	})

	count := h.TokenCount()
	if count != 30 {
		t.Errorf("expected token count 30, got %d", count)
	}
}

func TestHistory_Messages_DefensiveCopy(t *testing.T) {
	t.Parallel()

	h := NewHistory(4096, &tokenizer.SimpleTokenizer{})
	_ = h.AddUserMessage(context.Background(), "Hello")

	msgs := h.Messages()
	msgs[0].Content = "Modified"

	// Original should be unchanged.
	original := h.Messages()
	if original[0].Content == "Modified" {
		t.Error("Messages() should return a defensive copy")
	}
}

func TestHistory_SetCompressor(t *testing.T) {
	t.Parallel()

	h := NewHistory(4096, &tokenizer.SimpleTokenizer{})
	classifier := compress.NewClassifier()
	compressor := compress.NewHybridCompressor(classifier, compress.DefaultCompressorConfig())

	h.SetCompressor(compressor)
	// No way to verify directly, but should not panic.
}

func TestHistory_SetCompressionConfig(t *testing.T) {
	t.Parallel()

	h := NewHistory(4096, &tokenizer.SimpleTokenizer{})
	cfg := CompressionConfig{
		Enabled:     true,
		Threshold:   0.9,
		TargetRatio: 0.8,
	}

	h.SetCompressionConfig(cfg)
	// No way to verify directly, but should not panic.
}

func TestDefaultCompressionConfig(t *testing.T) {
	t.Parallel()

	cfg := DefaultCompressionConfig()
	if !cfg.Enabled {
		t.Error("expected Enabled to be true by default")
	}

	if cfg.Threshold != 0.8 {
		t.Errorf("expected Threshold 0.8, got %f", cfg.Threshold)
	}

	if cfg.TargetRatio != 0.7 {
		t.Errorf("expected TargetRatio 0.7, got %f", cfg.TargetRatio)
	}
}

func TestHistory_CompressionTrigger(t *testing.T) {
	t.Parallel()

	// Create history with small token limit.
	h := NewHistory(100, &tokenizer.SimpleTokenizer{})

	// Set up compressor.
	classifier := compress.NewClassifier()
	compressor := compress.NewHybridCompressor(classifier, compress.CompressorConfig{
		PreserveCritical: true,
		MinRetention:     0.3,
	})
	h.SetCompressor(compressor)
	h.SetCompressionConfig(CompressionConfig{
		Enabled:     true,
		Threshold:   0.5, // 50% = 50 tokens.
		TargetRatio: 0.3, // 30% = 30 tokens.
	})

	// Add messages that will exceed threshold.
	for range 5 {
			_ = h.AddUserMessage(context.Background(), "User message content here")
	}

	// Should have compressed.
	msgs := h.Messages()
	tokens := h.TokenCount()

	// After compression, should have fewer tokens
	// The exact number depends on message sizes and compression logic.
	if tokens > 100 {
		t.Errorf("expected tokens <= 100 after compression, got %d", tokens)
	}

	// Should still have user messages (critical).
	hasUser := false

	for _, msg := range msgs {
		if msg.Role == message.RoleUser {
			hasUser = true

			break
		}
	}

	if !hasUser {
		t.Error("expected at least one user message to be preserved")
	}
}

func TestHistory_CompressionDisabled(t *testing.T) {
	t.Parallel()

	h := NewHistory(100, &tokenizer.SimpleTokenizer{})

	// Set up compressor but disable compression.
	classifier := compress.NewClassifier()
	compressor := compress.NewHybridCompressor(classifier, compress.DefaultCompressorConfig())
	h.SetCompressor(compressor)
	h.SetCompressionConfig(CompressionConfig{
		Enabled:     false, // Disabled.
		Threshold:   0.5,
		TargetRatio: 0.3,
	})

	// Add messages that would exceed threshold.
	for range 10 {
		_ = h.AddUserMessage(context.Background(), "User message content")
	}

	// Should NOT have compressed.
	msgs := h.Messages()
	if len(msgs) != 10 {
		t.Errorf("expected 10 messages (no compression), got %d", len(msgs))
	}
}

func TestHistory_NoCompressorSet(t *testing.T) {
	t.Parallel()

	h := NewHistory(100, &tokenizer.SimpleTokenizer{})

	// Enable compression but no compressor set.
	h.SetCompressionConfig(CompressionConfig{
		Enabled:     true,
		Threshold:   0.5,
		TargetRatio: 0.3,
	})

	// Add messages.
	for range 10 {
		_ = h.AddUserMessage(context.Background(), "User message")
	}

	// Should NOT have compressed (no compressor).
	msgs := h.Messages()
	if len(msgs) != 10 {
		t.Errorf("expected 10 messages (no compressor), got %d", len(msgs))
	}
}

// mockCompressor is a test compressor that tracks calls.
type mockCompressor struct {
	callCount int
	messages  []message.Message
}

func (m *mockCompressor) Compress(_ context.Context, msgs []message.Message, _ int, _ tokenizer.Tokenizer) ([]message.Message, error) {
	m.callCount++
	m.messages = msgs
	// Return all messages (no actual compression).
	return msgs, nil
}

func (m *mockCompressor) Name() string {
	return "mock"
}

func TestHistory_CompressionCallsCompressor(t *testing.T) {
	t.Parallel()

	h := NewHistory(100, &tokenizer.SimpleTokenizer{})

	mock := &mockCompressor{}
	h.SetCompressor(mock)
	h.SetCompressionConfig(CompressionConfig{
		Enabled:     true,
		Threshold:   0.1, // Very low threshold to trigger compression.
		TargetRatio: 0.05,
	})

	// Add messages to trigger compression.
	_ = h.AddUserMessage(context.Background(), "Message 1")
	_ = h.AddUserMessage(context.Background(), "Message 2")
	_ = h.AddUserMessage(context.Background(), "Message 3")

	// Compressor should have been called.
	if mock.callCount == 0 {
		t.Error("expected compressor to be called")
	}
}

func TestHistory_MessagesForLLM(t *testing.T) {
	t.Parallel()

	h := NewHistory(4096, &tokenizer.SimpleTokenizer{})
	_ = h.AddSystemMessage(context.Background(), "System")
	_ = h.AddUserMessage(context.Background(), "User")

	msgs := h.MessagesForLLM()
	if len(msgs) != 2 {
		t.Errorf("expected 2 messages, got %d", len(msgs))
	}
}

func TestHistory_ToolCallTokenCounting(t *testing.T) {
	t.Parallel()

	h := NewHistory(4096, &tokenizer.SimpleTokenizer{})

	err := h.AddMessage(context.Background(), message.Message{
		Role:    message.RoleAssistant,
		Content: "Let me check that file.",
		ToolCalls: []message.ToolCall{
			{
				ID:   "call_123",
				Type: "function",
				Function: message.ToolCallFunction{
					Name:      "read_file",
					Arguments: `{"path": "/tmp/test.txt"}`,
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	msgs := h.Messages()
	// Token count should include tool call overhead.
	if msgs[0].Tokens < 10 {
		t.Errorf("expected tokens to include tool call overhead, got %d", msgs[0].Tokens)
	}
}
