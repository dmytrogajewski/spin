package compress

import (
	"context"
	"testing"
)

func TestHybridCompressor_Compress(t *testing.T) {
	compressor := &HybridCompressor{
		classifier: &MessageClassifier{},
	}

	messages := []CompressibleMessage{
		{ID: "1", Role: "user", Content: "Hello", Tokens: 10},
		{ID: "2", Role: "assistant", Content: "Hi there", Tokens: 15},
		{ID: "3", Role: "user", Content: "How are you?", Tokens: 20},
		{ID: "4", Role: "assistant", Content: "I'm doing well", Tokens: 25},
	}

	tokenizer := &mockTokenizer{
		counts: map[string]int{
			"Hello":          10,
			"Hi there":       15,
			"How are you?":   20,
			"I'm doing well": 25,
		},
	}

	tests := []struct {
		name         string
		targetTokens int
		wantLength   int
		wantErr      bool
	}{
		{
			name:         "compress to fit target",
			targetTokens: 30,
			wantLength:   2, // Should keep most important messages
			wantErr:      false,
		},
		{
			name:         "no compression needed",
			targetTokens: 100,
			wantLength:   4, // All messages fit
			wantErr:      false,
		},
		{
			name:         "extreme compression",
			targetTokens: 5,
			wantLength:   0, // No messages fit in extreme compression
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := compressor.Compress(context.Background(), messages, tt.targetTokens, tokenizer)

			if tt.wantErr {
				if err == nil {
					t.Errorf("HybridCompressor.Compress() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("HybridCompressor.Compress() unexpected error: %v", err)
				}
				if len(result) != tt.wantLength {
					t.Errorf("HybridCompressor.Compress() result length = %d, want %d", len(result), tt.wantLength)
				}
			}
		})
	}
}

func TestHybridCompressor_Name(t *testing.T) {
	compressor := &HybridCompressor{}
	name := compressor.Name()

	if name != "hybrid" {
		t.Errorf("HybridCompressor.Name() = %s, want 'hybrid'", name)
	}
}

func TestHybridCompressor_Compress_EmptyMessages(t *testing.T) {
	compressor := &HybridCompressor{
		classifier: &MessageClassifier{},
	}

	tokenizer := &mockTokenizer{counts: map[string]int{}}

	result, err := compressor.Compress(context.Background(), []CompressibleMessage{}, 100, tokenizer)

	if err != nil {
		t.Errorf("HybridCompressor.Compress() with empty messages unexpected error: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("HybridCompressor.Compress() with empty messages result length = %d, want 0", len(result))
	}
}

func TestHybridCompressor_Compress_ContextCancellation(t *testing.T) {
	compressor := &HybridCompressor{
		classifier: &MessageClassifier{},
	}

	messages := []CompressibleMessage{
		{ID: "1", Role: "user", Content: "Hello", Tokens: 10},
		{ID: "2", Role: "assistant", Content: "Hi there", Tokens: 15},
	}

	tokenizer := &mockTokenizer{
		counts: map[string]int{
			"Hello":    10,
			"Hi there": 15,
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	result, err := compressor.Compress(ctx, messages, 100, tokenizer)

	if err == nil {
		t.Errorf("HybridCompressor.Compress() expected context cancellation error, got nil")
	}

	if result != nil {
		t.Errorf("HybridCompressor.Compress() expected nil result on context cancellation, got %v", result)
	}
}

func TestHybridCompressor_Compress_PreservesOrder(t *testing.T) {
	compressor := &HybridCompressor{
		classifier: &MessageClassifier{},
	}

	messages := []CompressibleMessage{
		{ID: "1", Role: "user", Content: "First message", Tokens: 10},
		{ID: "2", Role: "assistant", Content: "Second message", Tokens: 15},
		{ID: "3", Role: "user", Content: "Third message", Tokens: 20},
		{ID: "4", Role: "assistant", Content: "Fourth message", Tokens: 25},
	}

	tokenizer := &mockTokenizer{
		counts: map[string]int{
			"First message":  10,
			"Second message": 15,
			"Third message":  20,
			"Fourth message": 25,
		},
	}

	result, err := compressor.Compress(context.Background(), messages, 50, tokenizer)

	if err != nil {
		t.Errorf("HybridCompressor.Compress() unexpected error: %v", err)
	}

	// Check that order is preserved
	for i := 1; i < len(result); i++ {
		if result[i].ID < result[i-1].ID {
			t.Errorf("HybridCompressor.Compress() order not preserved at index %d", i)
		}
	}
}

func TestDefaultCompressorConfig(t *testing.T) {
	config := DefaultCompressorConfig()

	if !config.PreserveCritical {
		t.Error("DefaultCompressorConfig().PreserveCritical should be true")
	}

	if config.MinRetention <= 0 || config.MinRetention > 1 {
		t.Errorf("DefaultCompressorConfig().MinRetention = %v, want between 0 and 1", config.MinRetention)
	}
}

func TestNewHybridCompressor(t *testing.T) {
	config := DefaultCompressorConfig()

	compressor := NewHybridCompressor(config)

	if compressor == nil {
		t.Fatal("NewHybridCompressor() returned nil")
	}

	if compressor.Name() != "hybrid" {
		t.Errorf("NewHybridCompressor().Name() = %v, want %v", compressor.Name(), "hybrid")
	}
}

func TestNewDefaultHybridCompressor(t *testing.T) {
	compressor := NewDefaultHybridCompressor()

	if compressor == nil {
		t.Fatal("NewDefaultHybridCompressor() returned nil")
	}

	if compressor.Name() != "hybrid" {
		t.Errorf("NewDefaultHybridCompressor().Name() = %v, want %v", compressor.Name(), "hybrid")
	}
}
