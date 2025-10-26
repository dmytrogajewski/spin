package llm

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/openai/openai-go"
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

		params := openai.ChatCompletionNewParams{
			Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
				openai.UserMessage("test"),
			}),
		}
		resp, err := p.Complete(context.Background(), params)
		if err != nil {
			t.Fatalf("Complete() error = %v", err)
		}

		if len(resp.Choices) == 0 || resp.Choices[0].Message.Content != "custom response" {
			content := ""
			if len(resp.Choices) > 0 {
				content = resp.Choices[0].Message.Content
			}
			t.Errorf("Content = %s, want 'custom response'", content)
		}
	})

	t.Run("with tool calls", func(t *testing.T) {
		toolCalls := []openai.ChatCompletionMessageToolCall{
			{
				ID:   "call_1",
				Type: openai.ChatCompletionMessageToolCallTypeFunction,
				Function: openai.ChatCompletionMessageToolCallFunction{
					Name:      "test_func",
					Arguments: `{"arg":"value"}`,
				},
			},
		}

		p := NewMockProvider("test", WithToolCalls(toolCalls))

		params := openai.ChatCompletionNewParams{
			Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
				openai.UserMessage("test"),
			}),
		}
		resp, err := p.Complete(context.Background(), params)
		if err != nil {
			t.Fatalf("Complete() error = %v", err)
		}

		if len(resp.Choices) == 0 || len(resp.Choices[0].Message.ToolCalls) != 1 {
			t.Fatalf("ToolCalls length = %d, want 1", len(resp.Choices[0].Message.ToolCalls))
		}

		if resp.Choices[0].Message.ToolCalls[0].ID != "call_1" {
			t.Errorf("ToolCall ID = %s, want call_1", resp.Choices[0].Message.ToolCalls[0].ID)
		}

		if resp.Choices[0].FinishReason != openai.ChatCompletionChoicesFinishReasonToolCalls {
			t.Errorf("FinishReason = %s, want tool_calls", resp.Choices[0].FinishReason)
		}
	})

	t.Run("with error", func(t *testing.T) {
		expectedErr := errors.New("mock error")
		p := NewMockProvider("test", WithError(expectedErr))

		params := openai.ChatCompletionNewParams{
			Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
				openai.UserMessage("test"),
			}),
		}
		_, err := p.Complete(context.Background(), params)
		if !errors.Is(err, expectedErr) {
			t.Errorf("Complete() error = %v, want %v", err, expectedErr)
		}
	})

	t.Run("with streaming chunks", func(t *testing.T) {
		chunks := []string{"Hello", " ", "World"}
		p := NewMockProvider("test", WithStreaming(chunks))

		// Complete should return concatenated chunks
		params := openai.ChatCompletionNewParams{
			Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
				openai.UserMessage("test"),
			}),
		}
		resp, err := p.Complete(context.Background(), params)
		if err != nil {
			t.Fatalf("Complete() error = %v", err)
		}

		if len(resp.Choices) == 0 || resp.Choices[0].Message.Content != "Hello World" {
			content := ""
			if len(resp.Choices) > 0 {
				content = resp.Choices[0].Message.Content
			}
			t.Errorf("Content = %s, want 'Hello World'", content)
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
		models := []openai.Model{
			{ID: "model-1"},
			{ID: "model-2"},
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

		params := openai.ChatCompletionNewParams{
			Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
				openai.UserMessage("Hi"),
			}),
			Model: openai.F("mock-model"),
		}

		resp, err := p.Complete(context.Background(), params)
		if err != nil {
			t.Fatalf("Complete() error = %v", err)
		}

		if len(resp.Choices) == 0 || resp.Choices[0].Message.Content != "Hello, World!" {
			content := ""
			if len(resp.Choices) > 0 {
				content = resp.Choices[0].Message.Content
			}
			t.Errorf("Content = %s, want 'Hello, World!'", content)
		}

		if resp.Model != "mock-model" {
			t.Errorf("Model = %s, want mock-model", resp.Model)
		}

		if len(resp.Choices) > 0 && resp.Choices[0].FinishReason != openai.ChatCompletionChoicesFinishReasonStop {
			t.Errorf("FinishReason = %s, want stop", resp.Choices[0].FinishReason)
		}

		if resp.Usage.TotalTokens == 0 {
			t.Error("Expected non-zero token usage")
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		p := NewMockProvider("test", WithDelay(100*time.Millisecond))

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		params := openai.ChatCompletionNewParams{
			Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
				openai.UserMessage("test"),
			}),
		}
		_, err := p.Complete(ctx, params)
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

		params := openai.ChatCompletionNewParams{
			Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
				openai.UserMessage("test"),
			}),
		}
		_, err := p.Complete(ctx, params)
		if err == nil {
			t.Error("Expected timeout error")
		}
	})

	t.Run("with error", func(t *testing.T) {
		expectedErr := errors.New("test error")
		p := NewMockProvider("test", WithError(expectedErr))

		params := openai.ChatCompletionNewParams{
			Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
				openai.UserMessage("test"),
			}),
		}
		_, err := p.Complete(context.Background(), params)
		if !errors.Is(err, expectedErr) {
			t.Errorf("Error = %v, want %v", err, expectedErr)
		}
	})
}

func TestMockProvider_Stream(t *testing.T) {
	t.Run("stream chunks", func(t *testing.T) {
		chunks := []string{"Hello", " ", "World", "!"}
		p := NewMockProvider("test", WithStreaming(chunks))

		params := openai.ChatCompletionNewParams{
			Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
				openai.UserMessage("test"),
			}),
		}
		stream, err := p.Stream(context.Background(), params)
		if err != nil {
			t.Fatalf("Stream() error = %v", err)
		}

		var received []string
		var doneReceived bool

		for chunk := range stream {
			if len(chunk.Choices) > 0 {
				delta := chunk.Choices[0].Delta
				if delta.Content != "" {
					received = append(received, delta.Content)
				}
				if chunk.Choices[0].FinishReason != "" {
					doneReceived = true
				}
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
		toolCalls := []openai.ChatCompletionMessageToolCall{
			{ID: "call_1", Type: openai.ChatCompletionMessageToolCallTypeFunction},
			{ID: "call_2", Type: openai.ChatCompletionMessageToolCallTypeFunction},
		}
		p := NewMockProvider("test", WithToolCalls(toolCalls))

		params := openai.ChatCompletionNewParams{
			Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
				openai.UserMessage("test"),
			}),
		}
		stream, err := p.Stream(context.Background(), params)
		if err != nil {
			t.Fatalf("Stream() error = %v", err)
		}

		var receivedToolCalls []openai.ChatCompletionChunkChoicesDeltaToolCall
		var finishReason openai.ChatCompletionChunkChoicesFinishReason

		for chunk := range stream {
			if len(chunk.Choices) > 0 {
				delta := chunk.Choices[0].Delta
				if len(delta.ToolCalls) > 0 {
					receivedToolCalls = append(receivedToolCalls, delta.ToolCalls...)
				}
				if chunk.Choices[0].FinishReason != "" {
					finishReason = chunk.Choices[0].FinishReason
				}
			}
		}

		if len(receivedToolCalls) != 2 {
			t.Fatalf("Received %d tool calls, want 2", len(receivedToolCalls))
		}

		if finishReason != openai.ChatCompletionChunkChoicesFinishReasonToolCalls {
			t.Errorf("FinishReason = %s, want tool_calls", finishReason)
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		p := NewMockProvider("test", WithDelay(50*time.Millisecond), WithStreaming([]string{"a", "b", "c"}))

		ctx, cancel := context.WithCancel(context.Background())

		params := openai.ChatCompletionNewParams{
			Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
				openai.UserMessage("test"),
			}),
		}
		stream, err := p.Stream(ctx, params)
		if err != nil {
			t.Fatalf("Stream() error = %v", err)
		}

		// Cancel after first chunk
		time.AfterFunc(60*time.Millisecond, cancel)

		chunkCount := 0
		for range stream {
			chunkCount++
		}

		// Should receive at least one chunk before cancellation
		if chunkCount == 0 {
			t.Error("Expected to receive at least one chunk")
		}
	})

	t.Run("error before streaming", func(t *testing.T) {
		expectedErr := errors.New("test error")
		p := NewMockProvider("test", WithError(expectedErr))

		params := openai.ChatCompletionNewParams{
			Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
				openai.UserMessage("test"),
			}),
		}
		_, err := p.Stream(context.Background(), params)
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
		customModels := []openai.Model{
			{ID: "custom-1"},
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
			params := openai.ChatCompletionNewParams{
				Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
					openai.UserMessage("test"),
				}),
			}
			for j := 0; j < 100; j++ {
				_ = p.Name()
				_ = p.Capabilities()
				_, _ = p.Complete(context.Background(), params)
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
