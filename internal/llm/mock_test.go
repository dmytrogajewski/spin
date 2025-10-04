package llm

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNewMockProvider(t *testing.T) {
	t.Run("default configuration", func(t *testing.T) {
		p := NewMockProvider("test")

		if p.Name() != "test" {
			t.Errorf("Name() = %s, want test", p.Name())
		}

		caps := p.Capabilities()
		if !caps.Streaming {
			t.Error("Expected streaming capability by default")
		}
		if !caps.FunctionCalling {
			t.Error("Expected function calling capability by default")
		}
	})

	t.Run("with custom response", func(t *testing.T) {
		p := NewMockProvider("test", WithResponse("custom response"))

		resp, err := p.Complete(context.Background(), CompletionRequest{})
		if err != nil {
			t.Fatalf("Complete() error = %v", err)
		}

		if resp.Content != "custom response" {
			t.Errorf("Content = %s, want 'custom response'", resp.Content)
		}
	})

	t.Run("with tool calls", func(t *testing.T) {
		toolCalls := []ToolCall{
			{
				ID:   "call_1",
				Type: "function",
				Function: FunctionCall{
					Name:      "test_func",
					Arguments: `{"arg":"value"}`,
				},
			},
		}

		p := NewMockProvider("test", WithToolCalls(toolCalls))

		resp, err := p.Complete(context.Background(), CompletionRequest{})
		if err != nil {
			t.Fatalf("Complete() error = %v", err)
		}

		if len(resp.ToolCalls) != 1 {
			t.Fatalf("ToolCalls length = %d, want 1", len(resp.ToolCalls))
		}

		if resp.ToolCalls[0].ID != "call_1" {
			t.Errorf("ToolCall ID = %s, want call_1", resp.ToolCalls[0].ID)
		}

		if resp.FinishReason != "tool_calls" {
			t.Errorf("FinishReason = %s, want tool_calls", resp.FinishReason)
		}
	})

	t.Run("with error", func(t *testing.T) {
		expectedErr := errors.New("mock error")
		p := NewMockProvider("test", WithError(expectedErr))

		_, err := p.Complete(context.Background(), CompletionRequest{})
		if !errors.Is(err, expectedErr) {
			t.Errorf("Complete() error = %v, want %v", err, expectedErr)
		}
	})

	t.Run("with streaming chunks", func(t *testing.T) {
		chunks := []string{"Hello", " ", "World"}
		p := NewMockProvider("test", WithStreaming(chunks))

		// Complete should return concatenated chunks
		resp, err := p.Complete(context.Background(), CompletionRequest{})
		if err != nil {
			t.Fatalf("Complete() error = %v", err)
		}

		if resp.Content != "Hello World" {
			t.Errorf("Content = %s, want 'Hello World'", resp.Content)
		}
	})

	t.Run("with capabilities", func(t *testing.T) {
		caps := Capabilities{
			Streaming:       false,
			FunctionCalling: true,
			Vision:          true,
		}

		p := NewMockProvider("test", WithCapabilities(caps))

		got := p.Capabilities()
		if got.Streaming != false {
			t.Error("Expected Streaming = false")
		}
		if got.FunctionCalling != true {
			t.Error("Expected FunctionCalling = true")
		}
		if got.Vision != true {
			t.Error("Expected Vision = true")
		}
	})

	t.Run("with models", func(t *testing.T) {
		models := []Model{
			{ID: "model-1", Name: "Model 1", ContextSize: 4096},
			{ID: "model-2", Name: "Model 2", ContextSize: 8192},
		}

		p := NewMockProvider("test", WithModels(models))

		got, err := p.Models(context.Background())
		if err != nil {
			t.Fatalf("Models() error = %v", err)
		}

		if len(got) != 2 {
			t.Fatalf("Models length = %d, want 2", len(got))
		}

		if got[0].ID != "model-1" {
			t.Errorf("Model[0].ID = %s, want model-1", got[0].ID)
		}
	})
}

func TestMockProvider_Complete(t *testing.T) {
	t.Run("basic completion", func(t *testing.T) {
		p := NewMockProvider("test", WithResponse("Hello, World!"))

		req := CompletionRequest{
			Messages: []Message{
				{Role: "user", Content: "Hi"},
			},
			Model:     "mock-model",
			MaxTokens: 100,
		}

		resp, err := p.Complete(context.Background(), req)
		if err != nil {
			t.Fatalf("Complete() error = %v", err)
		}

		if resp.Content != "Hello, World!" {
			t.Errorf("Content = %s, want 'Hello, World!'", resp.Content)
		}

		if resp.Model != "mock-model" {
			t.Errorf("Model = %s, want mock-model", resp.Model)
		}

		if resp.FinishReason != "stop" {
			t.Errorf("FinishReason = %s, want stop", resp.FinishReason)
		}

		if resp.Usage.TotalTokens == 0 {
			t.Error("Expected non-zero token usage")
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		p := NewMockProvider("test", WithDelay(100*time.Millisecond))

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err := p.Complete(ctx, CompletionRequest{})
		if err == nil {
			t.Error("Expected context cancellation error")
		}

		if !errors.Is(err, context.Canceled) {
			t.Errorf("Error = %v, want context.Canceled", err)
		}
	})

	t.Run("timeout", func(t *testing.T) {
		p := NewMockProvider("test", WithDelay(100*time.Millisecond))

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		_, err := p.Complete(ctx, CompletionRequest{})
		if err == nil {
			t.Error("Expected timeout error")
		}
	})

	t.Run("with error", func(t *testing.T) {
		expectedErr := errors.New("test error")
		p := NewMockProvider("test", WithError(expectedErr))

		_, err := p.Complete(context.Background(), CompletionRequest{})
		if !errors.Is(err, expectedErr) {
			t.Errorf("Error = %v, want %v", err, expectedErr)
		}
	})
}

func TestMockProvider_Stream(t *testing.T) {
	t.Run("stream chunks", func(t *testing.T) {
		chunks := []string{"Hello", " ", "World", "!"}
		p := NewMockProvider("test", WithStreaming(chunks))

		stream, err := p.Stream(context.Background(), CompletionRequest{})
		if err != nil {
			t.Fatalf("Stream() error = %v", err)
		}

		var received []string
		var doneReceived bool

		for chunk := range stream {
			if chunk.Error != nil {
				t.Errorf("Chunk error: %v", chunk.Error)
				continue
			}

			if chunk.Type == ChunkTypeContentDelta {
				received = append(received, chunk.Content)
			} else if chunk.Type == ChunkTypeDone {
				doneReceived = true
			}
		}

		if !doneReceived {
			t.Error("Did not receive done chunk")
		}

		if len(received) != len(chunks) {
			t.Fatalf("Received %d chunks, want %d", len(received), len(chunks))
		}

		for i, chunk := range received {
			if chunk != chunks[i] {
				t.Errorf("Chunk[%d] = %s, want %s", i, chunk, chunks[i])
			}
		}
	})

	t.Run("stream tool calls", func(t *testing.T) {
		toolCalls := []ToolCall{
			{ID: "call_1", Type: "function"},
			{ID: "call_2", Type: "function"},
		}
		p := NewMockProvider("test", WithToolCalls(toolCalls))

		stream, err := p.Stream(context.Background(), CompletionRequest{})
		if err != nil {
			t.Fatalf("Stream() error = %v", err)
		}

		var received []ToolCall
		var finishReason string

		for chunk := range stream {
			if chunk.Error != nil {
				t.Errorf("Chunk error: %v", chunk.Error)
				continue
			}

			if chunk.Type == ChunkTypeToolCallStart && chunk.ToolCall != nil {
				received = append(received, *chunk.ToolCall)
			} else if chunk.Type == ChunkTypeDone {
				finishReason = chunk.FinishReason
			}
		}

		if len(received) != 2 {
			t.Fatalf("Received %d tool calls, want 2", len(received))
		}

		if finishReason != "tool_calls" {
			t.Errorf("FinishReason = %s, want tool_calls", finishReason)
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		p := NewMockProvider("test", WithDelay(50*time.Millisecond), WithStreaming([]string{"a", "b", "c"}))

		ctx, cancel := context.WithCancel(context.Background())

		stream, err := p.Stream(ctx, CompletionRequest{})
		if err != nil {
			t.Fatalf("Stream() error = %v", err)
		}

		// Cancel after first chunk
		time.AfterFunc(60*time.Millisecond, cancel)

		errorReceived := false
		for chunk := range stream {
			if chunk.Error != nil {
				errorReceived = true
				if !errors.Is(chunk.Error, context.Canceled) {
					t.Errorf("Chunk error = %v, want context.Canceled", chunk.Error)
				}
			}
		}

		if !errorReceived {
			t.Error("Expected error chunk from cancellation")
		}
	})

	t.Run("error before streaming", func(t *testing.T) {
		expectedErr := errors.New("test error")
		p := NewMockProvider("test", WithError(expectedErr))

		_, err := p.Stream(context.Background(), CompletionRequest{})
		if !errors.Is(err, expectedErr) {
			t.Errorf("Stream() error = %v, want %v", err, expectedErr)
		}
	})
}

func TestMockProvider_Models(t *testing.T) {
	t.Run("default models", func(t *testing.T) {
		p := NewMockProvider("test")

		models, err := p.Models(context.Background())
		if err != nil {
			t.Fatalf("Models() error = %v", err)
		}

		if len(models) == 0 {
			t.Error("Expected at least one default model")
		}
	})

	t.Run("custom models", func(t *testing.T) {
		customModels := []Model{
			{ID: "custom-1", Name: "Custom 1"},
		}

		p := NewMockProvider("test", WithModels(customModels))

		models, err := p.Models(context.Background())
		if err != nil {
			t.Fatalf("Models() error = %v", err)
		}

		if len(models) != 1 {
			t.Fatalf("Models length = %d, want 1", len(models))
		}

		if models[0].ID != "custom-1" {
			t.Errorf("Model ID = %s, want custom-1", models[0].ID)
		}
	})

	t.Run("error", func(t *testing.T) {
		expectedErr := errors.New("models error")
		p := NewMockProvider("test", WithError(expectedErr))

		_, err := p.Models(context.Background())
		if !errors.Is(err, expectedErr) {
			t.Errorf("Models() error = %v, want %v", err, expectedErr)
		}
	})
}

func TestMockProvider_ThreadSafety(t *testing.T) {
	p := NewMockProvider("test")

	// Concurrent reads
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				_ = p.Name()
				_ = p.Capabilities()
				_, _ = p.Complete(context.Background(), CompletionRequest{})
			}
			done <- true
		}()
	}

	// Concurrent writes
	for i := 0; i < 10; i++ {
		go func(i int) {
			for j := 0; j < 100; j++ {
				p.SetResponse("response" + string(rune(i)))
				p.SetError(nil)
				p.SetToolCalls(nil)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 20; i++ {
		<-done
	}
}

func TestMockProvider_Close(t *testing.T) {
	p := NewMockProvider("test")

	err := p.Close()
	if err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}

	// Should be idempotent
	err = p.Close()
	if err != nil {
		t.Errorf("Close() second call error = %v, want nil", err)
	}
}

func TestChunkType_String(t *testing.T) {
	tests := []struct {
		typ  ChunkType
		want string
	}{
		{ChunkTypeContentDelta, "content_delta"},
		{ChunkTypeToolCallStart, "tool_call_start"},
		{ChunkTypeToolCallDelta, "tool_call_delta"},
		{ChunkTypeToolCallComplete, "tool_call_complete"},
		{ChunkTypeDone, "done"},
		{ChunkTypeError, "error"},
		{ChunkType(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.typ.String()
			if got != tt.want {
				t.Errorf("String() = %s, want %s", got, tt.want)
			}
		})
	}
}
