package llm

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/openai/openai-go"
)

var (
	errMockError   = errors.New("mock error")
	errTestError   = errors.New("test error")
	errTestError2  = errors.New("test error")
	errModelsError = errors.New("models error")
)

// testParams creates standard test params for mock provider tests.
func testParams() openai.ChatCompletionNewParams {
	return openai.ChatCompletionNewParams{
		Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
			openai.UserMessage("test"),
		}),
	}
}

// getResponseContent extracts content from completion response safely.
func getResponseContent(resp *openai.ChatCompletion) string {
	if len(resp.Choices) > 0 {
		return resp.Choices[0].Message.Content
	}

	return ""
}

func TestNewMockProvider(t *testing.T) {
	t.Parallel()

	t.Run("default configuration", func(t *testing.T) {
		t.Parallel()
		testNewMockProviderDefaults(t)
	})

	t.Run("with custom response", func(t *testing.T) {
		t.Parallel()
		testNewMockProviderResponse(t)
	})

	t.Run("with tool calls", func(t *testing.T) {
		t.Parallel()
		testNewMockProviderToolCalls(t)
	})

	t.Run("with error", func(t *testing.T) {
		t.Parallel()
		testNewMockProviderError(t)
	})

	t.Run("with streaming chunks", func(t *testing.T) {
		t.Parallel()
		testNewMockProviderStreaming(t)
	})

	t.Run("with capabilities", func(t *testing.T) {
		t.Parallel()
		testNewMockProviderCapabilities(t)
	})

	t.Run("with models", func(t *testing.T) {
		t.Parallel()
		testNewMockProviderModels(t)
	})
}

func testNewMockProviderDefaults(t *testing.T) {
	t.Helper()

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
}

func testNewMockProviderResponse(t *testing.T) {
	t.Helper()

	p := NewMockProvider("test", WithResponse("custom response"))

	resp, err := p.Complete(context.Background(), testParams())
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	if content := getResponseContent(resp); content != "custom response" {
		t.Errorf("Content = %s, want 'custom response'", content)
	}
}

func testNewMockProviderToolCalls(t *testing.T) {
	t.Helper()

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

	resp, err := p.Complete(context.Background(), testParams())
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
}

func testNewMockProviderError(t *testing.T) {
	t.Helper()

	p := NewMockProvider("test", WithError(errMockError))

	_, err := p.Complete(context.Background(), testParams())
	if !errors.Is(err, errMockError) {
		t.Errorf("Complete() error = %v, want %v", err, errMockError)
	}
}

func testNewMockProviderStreaming(t *testing.T) {
	t.Helper()

	chunks := []string{"Hello", " ", "World"}
	p := NewMockProvider("test", WithStreaming(chunks))

	resp, err := p.Complete(context.Background(), testParams())
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	if content := getResponseContent(resp); content != "Hello World" {
		t.Errorf("Content = %s, want 'Hello World'", content)
	}
}

func testNewMockProviderCapabilities(t *testing.T) {
	t.Helper()

	caps := Capabilities{Streaming: false, FunctionCalling: true, Vision: true}
	p := NewMockProvider("test", WithCapabilities(caps))

	got := p.Capabilities()
	if got.Streaming {
		t.Error("Expected Streaming = false")
	}

	if !got.FunctionCalling {
		t.Error("Expected FunctionCalling = true")
	}

	if !got.Vision {
		t.Error("Expected Vision = true")
	}
}

func testNewMockProviderModels(t *testing.T) {
	t.Helper()

	models := []openai.Model{{ID: "model-1"}, {ID: "model-2"}}
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
}

func TestMockProvider_Complete(t *testing.T) {
	t.Parallel()

	t.Run("basic completion", func(t *testing.T) {
		t.Parallel()
		testCompleteBasic(t)
	})

	testContextErrorCases(t, "Complete", func(p *MockProvider, ctx context.Context) error {
		_, err := p.Complete(ctx, testParams())

		return err
	})

	t.Run("with error", func(t *testing.T) {
		t.Parallel()

		p := NewMockProvider("test", WithError(errTestError))

		_, err := p.Complete(context.Background(), testParams())
		if !errors.Is(err, errTestError) {
			t.Errorf("Error = %v, want %v", err, errTestError)
		}
	})
}

func testCompleteBasic(t *testing.T) {
	t.Helper()

	p := NewMockProvider("test", WithResponse("Hello, World!"))
	params := openai.ChatCompletionNewParams{
		Messages: openai.F([]openai.ChatCompletionMessageParamUnion{openai.UserMessage("Hi")}),
		Model:    openai.F("mock-model"),
	}

	resp, err := p.Complete(context.Background(), params)
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	if content := getResponseContent(resp); content != "Hello, World!" {
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
}

// testContextErrorCases runs context cancellation and timeout subtests for a MockProvider operation.
func testContextErrorCases(t *testing.T, opName string, op func(p *MockProvider, ctx context.Context) error) {
	t.Helper()

	t.Run("context cancellation", func(t *testing.T) {
		t.Parallel()

		p := NewMockProvider("test", WithDelay(100*time.Millisecond))
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := op(p, ctx)
		if err == nil {
			t.Errorf("%s: Expected context cancellation error", opName)
		}

		if !errors.Is(err, context.Canceled) {
			t.Errorf("%s: Error = %v, want context.Canceled", opName, err)
		}
	})

	t.Run("timeout", func(t *testing.T) {
		t.Parallel()

		p := NewMockProvider("test", WithDelay(100*time.Millisecond))

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		err := op(p, ctx)
		if err == nil {
			t.Errorf("%s: Expected timeout error", opName)
		}
	})
}

func TestMockProvider_Stream(t *testing.T) {
	t.Parallel()

	t.Run("stream chunks", func(t *testing.T) {
		t.Parallel()
		testStreamChunks(t)
	})

	t.Run("stream tool calls", func(t *testing.T) {
		t.Parallel()
		testStreamToolCalls(t)
	})

	t.Run("stream context cancellation", func(t *testing.T) {
		t.Parallel()
		testStreamCancellation(t)
	})

	t.Run("error before streaming", func(t *testing.T) {
		t.Parallel()

		p := NewMockProvider("test", WithError(errTestError2))

		_, err := p.Stream(context.Background(), testParams())
		if !errors.Is(err, errTestError2) {
			t.Errorf("Stream() error = %v, want %v", err, errTestError2)
		}
	})
}

// collectStreamContent drains a stream and returns content chunks and whether a done signal was received.
func collectStreamContent(stream <-chan openai.ChatCompletionChunk) (contents []string, doneReceived bool) {
	for chunk := range stream {
		if len(chunk.Choices) == 0 {
			continue
		}

		if chunk.Choices[0].Delta.Content != "" {
			contents = append(contents, chunk.Choices[0].Delta.Content)
		}

		if chunk.Choices[0].FinishReason != "" {
			doneReceived = true
		}
	}

	return contents, doneReceived
}

func testStreamChunks(t *testing.T) {
	t.Helper()

	chunks := []string{"Hello", " ", "World", "!"}
	p := NewMockProvider("test", WithStreaming(chunks))

	stream, err := p.Stream(context.Background(), testParams())
	if err != nil {
		t.Fatalf("Stream() error = %v", err)
	}

	received, doneReceived := collectStreamContent(stream)

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
}

func testStreamToolCalls(t *testing.T) {
	t.Helper()

	toolCalls := []openai.ChatCompletionMessageToolCall{
		{ID: "call_1", Type: openai.ChatCompletionMessageToolCallTypeFunction},
		{ID: "call_2", Type: openai.ChatCompletionMessageToolCallTypeFunction},
	}
	p := NewMockProvider("test", WithToolCalls(toolCalls))

	stream, err := p.Stream(context.Background(), testParams())
	if err != nil {
		t.Fatalf("Stream() error = %v", err)
	}

	var (
		receivedToolCalls []openai.ChatCompletionChunkChoicesDeltaToolCall
		finishReason      openai.ChatCompletionChunkChoicesFinishReason
	)

	for chunk := range stream {
		if len(chunk.Choices) == 0 {
			continue
		}

		receivedToolCalls = append(receivedToolCalls, chunk.Choices[0].Delta.ToolCalls...)
		if chunk.Choices[0].FinishReason != "" {
			finishReason = chunk.Choices[0].FinishReason
		}
	}

	if len(receivedToolCalls) != 2 {
		t.Fatalf("Received %d tool calls, want 2", len(receivedToolCalls))
	}

	if finishReason != openai.ChatCompletionChunkChoicesFinishReasonToolCalls {
		t.Errorf("FinishReason = %s, want tool_calls", finishReason)
	}
}

func testStreamCancellation(t *testing.T) {
	t.Helper()

	p := NewMockProvider("test", WithDelay(50*time.Millisecond), WithStreaming([]string{"a", "b", "c"}))
	ctx, cancel := context.WithCancel(context.Background())

	stream, err := p.Stream(ctx, testParams())
	if err != nil {
		t.Fatalf("Stream() error = %v", err)
	}

	time.AfterFunc(60*time.Millisecond, cancel)

	chunkCount := 0
	for range stream {
		chunkCount++
	}

	if chunkCount == 0 {
		t.Error("Expected to receive at least one chunk")
	}
}

func TestMockProvider_Models(t *testing.T) {
	t.Parallel()

	t.Run("default models", func(t *testing.T) {
		t.Parallel()

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
		t.Parallel()

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
		t.Parallel()

		expectedErr := errModelsError
		p := NewMockProvider("test", WithError(expectedErr))

		_, err := p.Models(context.Background())
		if !errors.Is(err, expectedErr) {
			t.Errorf("Models() error = %v, want %v", err, expectedErr)
		}
	})
}

func TestMockProvider_ThreadSafety(t *testing.T) {
	t.Parallel()

	p := NewMockProvider("test")

	// Concurrent reads.
	done := make(chan bool)

	for range 10 {
		go func() {
			params := openai.ChatCompletionNewParams{
				Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
					openai.UserMessage("test"),
				}),
			}

			for range 100 {
				_ = p.Name()
				_ = p.Capabilities()
				_, _ = p.Complete(context.Background(), params)
			}

			done <- true
		}()
	}

	// Concurrent writes.
	for i := range 10 {
		go func(i int) {
			for range 100 {
				p.SetResponse("response" + strconv.Itoa(i))
				p.SetError(nil)
				p.SetToolCalls(nil)
			}

			done <- true
		}(i)
	}

	// Wait for all goroutines.
	for range 20 {
		<-done
	}
}

func TestMockProvider_Close(t *testing.T) {
	t.Parallel()

	p := NewMockProvider("test")

	err := p.Close()
	if err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}

	// Should be idempotent.
	err = p.Close()
	if err != nil {
		t.Errorf("Close() second call error = %v, want nil", err)
	}
}
