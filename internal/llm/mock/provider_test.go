package mock

import (
	"context"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/llm"
)

func TestProvider_Complete(t *testing.T) {
	p := NewProvider("Response 1", "Response 2")

	// First call
	resp, err := p.Complete(context.Background(), llm.CompletionRequest{
		Messages: []llm.Message{{Role: "user", Content: "Hello"}},
	})
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}
	if resp.Content != "Response 1" {
		t.Errorf("Expected 'Response 1', got %q", resp.Content)
	}

	// Second call
	resp, err = p.Complete(context.Background(), llm.CompletionRequest{
		Messages: []llm.Message{{Role: "user", Content: "World"}},
	})
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}
	if resp.Content != "Response 2" {
		t.Errorf("Expected 'Response 2', got %q", resp.Content)
	}

	// Third call (queue empty)
	resp, err = p.Complete(context.Background(), llm.CompletionRequest{
		Messages: []llm.Message{{Role: "user", Content: "Fallback"}},
	})
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}
	if resp.Content != "Mock response" {
		t.Errorf("Expected 'Mock response', got %q", resp.Content)
	}
}

func TestProvider_Stream(t *testing.T) {
	p := NewProvider("Hello world!")
	p.StreamChunkSize = 5

	chunks, err := p.Stream(context.Background(), llm.CompletionRequest{
		Messages: []llm.Message{{Role: "user", Content: "Test"}},
	})
	if err != nil {
		t.Fatalf("Stream failed: %v", err)
	}

	var content string
	for chunk := range chunks {
		if chunk.Type == llm.ChunkTypeContentDelta {
			content += chunk.Content
		}
	}

	if content != "Hello world!" {
		t.Errorf("Expected 'Hello world!', got %q", content)
	}
}

func TestProvider_CallHistory(t *testing.T) {
	p := NewProvider("Response")

	p.Complete(context.Background(), llm.CompletionRequest{
		Messages: []llm.Message{{Role: "user", Content: "Prompt 1"}},
	})
	p.Complete(context.Background(), llm.CompletionRequest{
		Messages: []llm.Message{{Role: "user", Content: "Prompt 2"}},
	})
	p.Stream(context.Background(), llm.CompletionRequest{
		Messages: []llm.Message{{Role: "user", Content: "Prompt 3"}},
	})

	if p.PromptCount() != 3 {
		t.Errorf("Expected 3 prompts, got %d", p.PromptCount())
	}

	if p.LastPrompt() != "Prompt 3" {
		t.Errorf("Expected last prompt 'Prompt 3', got %q", p.LastPrompt())
	}
}

func TestProvider_Delay(t *testing.T) {
	p := NewProvider("Response")
	p.Delay = 50 * time.Millisecond

	start := time.Now()
	_, err := p.Complete(context.Background(), llm.CompletionRequest{
		Messages: []llm.Message{{Role: "user", Content: "Test"}},
	})
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}
	elapsed := time.Since(start)

	if elapsed < 50*time.Millisecond {
		t.Errorf("Expected delay of at least 50ms, got %v", elapsed)
	}
}

func TestProvider_Cancel(t *testing.T) {
	p := NewProvider("Response")
	p.Delay = 1 * time.Second

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := p.Complete(ctx, llm.CompletionRequest{
		Messages: []llm.Message{{Role: "user", Content: "Test"}},
	})
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
}

func TestProvider_Reset(t *testing.T) {
	p := NewProvider("Response 1")
	p.Complete(context.Background(), llm.CompletionRequest{
		Messages: []llm.Message{{Role: "user", Content: "Test"}},
	})

	if p.PromptCount() != 1 {
		t.Fatalf("Expected 1 prompt before reset, got %d", p.PromptCount())
	}

	p.Reset("Response 2", "Response 3")

	if p.PromptCount() != 0 {
		t.Errorf("Expected 0 prompts after reset, got %d", p.PromptCount())
	}

	resp, _ := p.Complete(context.Background(), llm.CompletionRequest{
		Messages: []llm.Message{{Role: "user", Content: "New"}},
	})
	if resp.Content != "Response 2" {
		t.Errorf("Expected 'Response 2' after reset, got %q", resp.Content)
	}
}
