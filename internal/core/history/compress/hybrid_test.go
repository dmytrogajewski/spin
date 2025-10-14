package compress

import (
	"context"
	"testing"
)

func TestHybridCompressor_PreserveCritical(t *testing.T) {
	classifier := &MessageClassifier{}
	compressor := &HybridCompressor{
		classifier: classifier,
		config: CompressorConfig{
			PreserveCritical: true,
			MinRetention:     0.3,
		},
	}

	tokenizer := &simpleTokenizer{}
	ctx := context.Background()

	// Create messages: 3 critical (user) + 2 low (verbose assistant)
	messages := []CompressibleMessage{
		{Role: RoleUser, Content: "First question", Tokens: 10},
		{Role: RoleAssistant, Content: generateLongContent(), Tokens: 500},
		{Role: RoleUser, Content: "Second question", Tokens: 10},
		{Role: RoleAssistant, Content: generateLongContent(), Tokens: 500},
		{Role: RoleUser, Content: "Third question", Tokens: 10},
	}

	// Target: 50 tokens (only fit user messages)
	compressed, err := compressor.Compress(ctx, messages, 50, tokenizer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// All 3 user messages should be preserved
	userCount := 0
	for _, msg := range compressed {
		if msg.Role == RoleUser {
			userCount++
		}
	}

	if userCount != 3 {
		t.Errorf("expected 3 user messages preserved, got %d", userCount)
	}

	// Should have removed low-importance messages
	if len(compressed) >= len(messages) {
		t.Errorf("expected compression, got %d messages (original: %d)", len(compressed), len(messages))
	}
}

func TestHybridCompressor_ChronologicalOrder(t *testing.T) {
	classifier := &MessageClassifier{}
	compressor := &HybridCompressor{
		classifier: classifier,
		config: CompressorConfig{
			PreserveCritical: true,
			MinRetention:     0.3,
		},
	}

	tokenizer := &simpleTokenizer{}
	ctx := context.Background()

	// Create messages with IDs to track order
	messages := []CompressibleMessage{
		{ID: "1", Role: RoleUser, Content: "First", Tokens: 10},
		{ID: "2", Role: RoleAssistant, Content: "Response 1", Tokens: 20},
		{ID: "3", Role: RoleUser, Content: "Second", Tokens: 10},
		{ID: "4", Role: RoleAssistant, Content: "Response 2", Tokens: 20},
		{ID: "5", Role: RoleUser, Content: "Third", Tokens: 10},
	}

	compressed, err := compressor.Compress(ctx, messages, 100, tokenizer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check chronological order is preserved
	for i := 1; i < len(compressed); i++ {
		prev := compressed[i-1]
		curr := compressed[i]

		// Find original indices
		prevIdx := findMessageIndex(messages, prev.ID)
		currIdx := findMessageIndex(messages, curr.ID)

		if prevIdx > currIdx {
			t.Errorf("chronological order violated: message %s (idx %d) before %s (idx %d)",
				prev.ID, prevIdx, curr.ID, currIdx)
		}
	}
}

func TestHybridCompressor_GreedySelection(t *testing.T) {
	classifier := &MessageClassifier{}
	compressor := &HybridCompressor{
		classifier: classifier,
		config: CompressorConfig{
			PreserveCritical: true,
			MinRetention:     0.3,
		},
	}

	tokenizer := &simpleTokenizer{}
	ctx := context.Background()

	// Create mix of importance levels
	messages := []CompressibleMessage{
		{Role: RoleUser, Content: "Critical 1", Tokens: 10},                // Critical
		{Role: RoleAssistant, Content: "```code```", Tokens: 20},           // High
		{Role: RoleAssistant, Content: "Regular", Tokens: 15},              // Medium
		{Role: RoleAssistant, Content: generateLongContent(), Tokens: 500}, // Low
		{Role: RoleUser, Content: "Critical 2", Tokens: 10},                // Critical
	}

	// Target: 60 tokens (fits critical + high + medium, excludes low)
	compressed, err := compressor.Compress(ctx, messages, 60, tokenizer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should include: 2 critical (20) + 1 high (20) + 1 medium (15) = 55 tokens
	if len(compressed) < 4 {
		t.Errorf("expected at least 4 messages, got %d", len(compressed))
	}

	// Should exclude the verbose (low importance) message
	hasVerbose := false
	for _, msg := range compressed {
		if msg.Tokens == 500 {
			hasVerbose = true
			break
		}
	}

	if hasVerbose {
		t.Errorf("expected verbose message to be excluded")
	}
}

func TestHybridCompressor_EmptyMessages(t *testing.T) {
	classifier := &MessageClassifier{}
	compressor := &HybridCompressor{
		classifier: classifier,
		config: CompressorConfig{
			PreserveCritical: true,
			MinRetention:     0.3,
		},
	}

	tokenizer := &simpleTokenizer{}
	ctx := context.Background()

	compressed, err := compressor.Compress(ctx, []CompressibleMessage{}, 1000, tokenizer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(compressed) != 0 {
		t.Errorf("expected empty result, got %d messages", len(compressed))
	}
}

func TestHybridCompressor_AllCritical(t *testing.T) {
	classifier := &MessageClassifier{}
	compressor := &HybridCompressor{
		classifier: classifier,
		config: CompressorConfig{
			PreserveCritical: true,
			MinRetention:     0.3,
		},
	}

	tokenizer := &simpleTokenizer{}
	ctx := context.Background()

	// All messages are critical (user messages)
	messages := []CompressibleMessage{
		{Role: RoleUser, Content: "Question 1", Tokens: 100},
		{Role: RoleUser, Content: "Question 2", Tokens: 100},
		{Role: RoleUser, Content: "Question 3", Tokens: 100},
		{Role: RoleUser, Content: "Question 4", Tokens: 100},
	}

	// Target: 250 tokens (can't fit all 400)
	compressed, err := compressor.Compress(ctx, messages, 250, tokenizer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// With PreserveCritical=true, should keep all critical even if exceeds budget
	// OR implement smart handling (keep most recent critical)
	if len(compressed) == 0 {
		t.Errorf("expected some messages to be preserved")
	}

	// All preserved messages should be user messages
	for _, msg := range compressed {
		if msg.Role != RoleUser {
			t.Errorf("expected only user messages, got role: %s", msg.Role)
		}
	}
}

func TestHybridCompressor_MinRetention(t *testing.T) {
	classifier := &MessageClassifier{}
	compressor := &HybridCompressor{
		classifier: classifier,
		config: CompressorConfig{
			PreserveCritical: true,
			MinRetention:     0.5, // Keep at least 50%
		},
	}

	tokenizer := &simpleTokenizer{}
	ctx := context.Background()

	// Create 10 messages, all low importance
	messages := make([]CompressibleMessage, 10)
	for i := 0; i < 10; i++ {
		messages[i] = CompressibleMessage{
			Role:    RoleAssistant,
			Content: generateLongContent(),
			Tokens:  100,
		}
	}

	// Target: very low (10 tokens), but MinRetention should enforce 50%
	compressed, err := compressor.Compress(ctx, messages, 10, tokenizer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	minExpected := int(float64(len(messages)) * 0.5)
	if len(compressed) < minExpected {
		t.Errorf("expected at least %d messages (50%%), got %d", minExpected, len(compressed))
	}
}

func TestHybridCompressor_ZeroTarget(t *testing.T) {
	classifier := &MessageClassifier{}
	compressor := &HybridCompressor{
		classifier: classifier,
		config: CompressorConfig{
			PreserveCritical: true,
			MinRetention:     0.3,
		},
	}

	tokenizer := &simpleTokenizer{}
	ctx := context.Background()

	messages := []CompressibleMessage{
		{Role: RoleUser, Content: "Test", Tokens: 10},
	}

	// Target: 0 tokens (edge case)
	compressed, err := compressor.Compress(ctx, messages, 0, tokenizer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should still preserve critical with PreserveCritical=true
	// OR return minimum based on MinRetention
	if len(compressed) == 0 {
		t.Errorf("expected at least some messages preserved with zero target")
	}
}

// Benchmark compression performance
func BenchmarkHybridCompressor_100Messages(b *testing.B) {
	classifier := &MessageClassifier{}
	compressor := &HybridCompressor{
		classifier: classifier,
		config: CompressorConfig{
			PreserveCritical: true,
			MinRetention:     0.3,
		},
	}

	tokenizer := &simpleTokenizer{}
	ctx := context.Background()

	// Generate 100 messages
	messages := generateTestMessages(100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = compressor.Compress(ctx, messages, 5000, tokenizer)
	}
}

func BenchmarkHybridCompressor_500Messages(b *testing.B) {
	classifier := &MessageClassifier{}
	compressor := &HybridCompressor{
		classifier: classifier,
		config: CompressorConfig{
			PreserveCritical: true,
			MinRetention:     0.3,
		},
	}

	tokenizer := &simpleTokenizer{}
	ctx := context.Background()

	messages := generateTestMessages(500)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = compressor.Compress(ctx, messages, 20000, tokenizer)
	}
}

func BenchmarkHybridCompressor_1000Messages(b *testing.B) {
	classifier := &MessageClassifier{}
	compressor := &HybridCompressor{
		classifier: classifier,
		config: CompressorConfig{
			PreserveCritical: true,
			MinRetention:     0.3,
		},
	}

	tokenizer := &simpleTokenizer{}
	ctx := context.Background()

	messages := generateTestMessages(1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = compressor.Compress(ctx, messages, 40000, tokenizer)
	}
}

// Helper functions

func generateLongContent() string {
	content := "This is a very verbose response. "
	for i := 0; i < 30; i++ {
		content += "Let me elaborate on this point in great detail. "
	}
	return content
}

func findMessageIndex(messages []CompressibleMessage, id string) int {
	for i, msg := range messages {
		if msg.ID == id {
			return i
		}
	}
	return -1
}

func generateTestMessages(count int) []CompressibleMessage {
	messages := make([]CompressibleMessage, count)
	for i := 0; i < count; i++ {
		if i%3 == 0 {
			messages[i] = CompressibleMessage{
				Role:    RoleUser,
				Content: "User question",
				Tokens:  50,
			}
		} else if i%3 == 1 {
			messages[i] = CompressibleMessage{
				Role:    RoleAssistant,
				Content: "Regular response",
				Tokens:  80,
			}
		} else {
			messages[i] = CompressibleMessage{
				Role:    RoleAssistant,
				Content: generateLongContent(),
				Tokens:  200,
			}
		}
	}
	return messages
}

// simpleTokenizer for tests
type simpleTokenizer struct{}

func (t *simpleTokenizer) Count(text string) int {
	// Simple approximation: ~4 chars per token
	return len(text) / 4
}
