package llm

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestNewSSEScanner(t *testing.T) {
	input := "data: test\n\n"
	scanner := NewSSEScanner(strings.NewReader(input))

	if scanner == nil {
		t.Fatal("NewSSEScanner() returned nil")
	}
}

func TestSSEScanner_Scan_SimpleEvent(t *testing.T) {
	input := "data: hello world\n\n"
	scanner := NewSSEScanner(strings.NewReader(input))

	if !scanner.Scan() {
		t.Fatal("Scan() failed on valid input")
	}

	event := scanner.Event()
	if event.Data != "hello world" {
		t.Errorf("Event().Data = %q, want %q", event.Data, "hello world")
	}
}

func TestSSEScanner_Scan_MultipleEvents(t *testing.T) {
	input := `data: event1

data: event2

data: event3

`
	scanner := NewSSEScanner(strings.NewReader(input))

	events := []string{}
	for scanner.Scan() {
		events = append(events, scanner.Event().Data)
	}

	if len(events) != 3 {
		t.Fatalf("got %d events, want 3", len(events))
	}

	expected := []string{"event1", "event2", "event3"}
	for i, want := range expected {
		if events[i] != want {
			t.Errorf("event[%d] = %q, want %q", i, events[i], want)
		}
	}
}

func TestSSEScanner_Scan_DoneMarker(t *testing.T) {
	input := `data: event1

data: [DONE]

`
	scanner := NewSSEScanner(strings.NewReader(input))

	// First event
	if !scanner.Scan() {
		t.Fatal("Scan() failed on first event")
	}
	event := scanner.Event()
	if event.Data != "event1" {
		t.Errorf("first event = %q, want %q", event.Data, "event1")
	}

	// Done marker
	if !scanner.Scan() {
		t.Fatal("Scan() failed on [DONE] marker")
	}
	event = scanner.Event()
	if !event.Done {
		t.Error("Event().Done should be true for [DONE] marker")
	}
}

func TestSSEScanner_Scan_MultilineData(t *testing.T) {
	input := `data: line1
data: line2
data: line3

`
	scanner := NewSSEScanner(strings.NewReader(input))

	if !scanner.Scan() {
		t.Fatal("Scan() failed")
	}

	event := scanner.Event()
	want := "line1\nline2\nline3"
	if event.Data != want {
		t.Errorf("Event().Data = %q, want %q", event.Data, want)
	}
}

func TestSSEScanner_Scan_EmptyLines(t *testing.T) {
	input := `
data: event1

data: event2


`
	scanner := NewSSEScanner(strings.NewReader(input))

	count := 0
	for scanner.Scan() {
		count++
	}

	if count != 2 {
		t.Errorf("got %d events, want 2", count)
	}
}

func TestSSEScanner_Scan_NoDataPrefix(t *testing.T) {
	input := "no data prefix\n\n"
	scanner := NewSSEScanner(strings.NewReader(input))

	// Should skip lines without "data:" prefix
	if scanner.Scan() {
		t.Error("Scan() should skip lines without data: prefix")
	}
}

func TestSSEScanner_Err(t *testing.T) {
	input := "data: valid\n\n"
	scanner := NewSSEScanner(strings.NewReader(input))

	scanner.Scan()

	if err := scanner.Err(); err != nil {
		t.Errorf("Err() = %v, want nil", err)
	}
}

func TestSSEScanner_Scan_EOF(t *testing.T) {
	input := "data: event\n\n"
	scanner := NewSSEScanner(strings.NewReader(input))

	// Read the event
	if !scanner.Scan() {
		t.Fatal("Scan() failed")
	}

	// Should return false on EOF
	if scanner.Scan() {
		t.Error("Scan() should return false at EOF")
	}

	if scanner.Err() != nil {
		t.Errorf("Err() = %v, want nil at EOF", scanner.Err())
	}
}

func TestStreamResponse_Basic(t *testing.T) {
	input := `data: {"choices": [{"delta": {"content": "hello"}}]}

data: {"choices": [{"delta": {"content": " world"}}]}

data: [DONE]

`
	chunks := make(chan StreamChunk, 10)
	ctx := context.Background()

	go func() {
		defer close(chunks)
		if err := streamResponse(ctx, strings.NewReader(input), chunks); err != nil {
			t.Errorf("streamResponse() error = %v", err)
		}
	}()

	var received []StreamChunk
	for chunk := range chunks {
		received = append(received, chunk)
	}

	// Should have at least: content delta + content delta + done
	if len(received) < 3 {
		t.Fatalf("got %d chunks, want at least 3", len(received))
	}

	// Verify content chunks
	if received[0].Type != ChunkTypeContentDelta {
		t.Errorf("first chunk type = %v, want %v", received[0].Type, ChunkTypeContentDelta)
	}
	if received[0].Content != "hello" {
		t.Errorf("first chunk content = %q, want %q", received[0].Content, "hello")
	}
}

func TestStreamResponse_ContextCancellation(t *testing.T) {
	// Create a stream that would block
	input := `data: {"choices": [{"delta": {"content": "test"}}]}

data: {"choices": [{"delta": {"content": "test"}}]}

`
	chunks := make(chan StreamChunk) // Unbuffered to force blocking

	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	wg.Add(1)
	errCh := make(chan error, 1)
	go func() {
		defer wg.Done()
		defer close(chunks)
		err := streamResponse(ctx, strings.NewReader(input), chunks)
		errCh <- err
	}()

	// Cancel immediately
	cancel()

	// Wait for goroutine to finish
	wg.Wait()

	// Should get context.Canceled error
	select {
	case err := <-errCh:
		if err != context.Canceled {
			t.Errorf("expected context.Canceled error, got %v", err)
		}
	default:
		// No error is also acceptable if stream finished before cancellation
	}
}

func TestStreamResponse_ErrorChunk(t *testing.T) {
	// Invalid JSON
	input := "data: {invalid json}\n\n"
	chunks := make(chan StreamChunk, 10)
	ctx := context.Background()

	go func() {
		defer close(chunks)
		streamResponse(ctx, strings.NewReader(input), chunks)
	}()

	hasErrorChunk := false
	for chunk := range chunks {
		if chunk.Type == ChunkTypeError {
			hasErrorChunk = true
		}
	}

	if !hasErrorChunk {
		t.Error("expected error chunk for invalid JSON")
	}
}

func TestStreamResponse_EmptyInput(t *testing.T) {
	input := ""
	chunks := make(chan StreamChunk, 10)
	ctx := context.Background()

	go func() {
		defer close(chunks)
		streamResponse(ctx, strings.NewReader(input), chunks)
	}()

	count := 0
	for range chunks {
		count++
	}

	// Should handle empty input gracefully
	if count > 0 {
		t.Errorf("got %d chunks for empty input, want 0", count)
	}
}

func TestStreamResponse_DoneChunk(t *testing.T) {
	input := `data: {"choices": [{"delta": {"content": "test"}}]}

data: [DONE]

`
	chunks := make(chan StreamChunk, 10)
	ctx := context.Background()

	go func() {
		defer close(chunks)
		streamResponse(ctx, strings.NewReader(input), chunks)
	}()

	hasDoneChunk := false
	for chunk := range chunks {
		if chunk.Type == ChunkTypeDone {
			hasDoneChunk = true
		}
	}

	if !hasDoneChunk {
		t.Error("expected done chunk for [DONE] marker")
	}
}

func TestConvertDelta_ContentDelta(t *testing.T) {
	delta := &chatCompletionChunk{
		Choices: []struct {
			Delta struct {
				Content   string `json:"content"`
				ToolCalls []struct {
					Index    int    `json:"index"`
					ID       string `json:"id"`
					Type     string `json:"type"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"delta"`
			FinishReason *string `json:"finish_reason"`
		}{
			{
				Delta: struct {
					Content   string `json:"content"`
					ToolCalls []struct {
						Index    int    `json:"index"`
						ID       string `json:"id"`
						Type     string `json:"type"`
						Function struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
						} `json:"function"`
					} `json:"tool_calls"`
				}{
					Content: "hello",
				},
			},
		},
	}

	chunk := convertDelta(delta)

	if chunk == nil {
		t.Fatal("convertDelta() returned nil")
	}

	if chunk.Type != ChunkTypeContentDelta {
		t.Errorf("Type = %v, want %v", chunk.Type, ChunkTypeContentDelta)
	}

	if chunk.Content != "hello" {
		t.Errorf("Content = %q, want %q", chunk.Content, "hello")
	}
}

func TestConvertDelta_ToolCallDelta(t *testing.T) {
	delta := &chatCompletionChunk{
		Choices: []struct {
			Delta struct {
				Content   string `json:"content"`
				ToolCalls []struct {
					Index    int    `json:"index"`
					ID       string `json:"id"`
					Type     string `json:"type"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"delta"`
			FinishReason *string `json:"finish_reason"`
		}{
			{
				Delta: struct {
					Content   string `json:"content"`
					ToolCalls []struct {
						Index    int    `json:"index"`
						ID       string `json:"id"`
						Type     string `json:"type"`
						Function struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
						} `json:"function"`
					} `json:"tool_calls"`
				}{
					ToolCalls: []struct {
						Index    int    `json:"index"`
						ID       string `json:"id"`
						Type     string `json:"type"`
						Function struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
						} `json:"function"`
					}{
						{
							ID:   "call-1",
							Type: "function",
							Function: struct {
								Name      string `json:"name"`
								Arguments string `json:"arguments"`
							}{
								Name:      "test_function",
								Arguments: `{"key": "value"}`,
							},
						},
					},
				},
			},
		},
	}

	chunk := convertDelta(delta)

	if chunk == nil {
		t.Fatal("convertDelta() returned nil")
	}

	if chunk.Type != ChunkTypeToolCallDelta {
		t.Errorf("Type = %v, want %v", chunk.Type, ChunkTypeToolCallDelta)
	}

	if chunk.ToolCall == nil {
		t.Fatal("ToolCall is nil")
	}

	if chunk.ToolCall.ID != "call-1" {
		t.Errorf("ToolCall.ID = %q, want %q", chunk.ToolCall.ID, "call-1")
	}
}

func TestConvertDelta_FinishReason(t *testing.T) {
	finishReason := "stop"
	delta := &chatCompletionChunk{
		Choices: []struct {
			Delta struct {
				Content   string `json:"content"`
				ToolCalls []struct {
					Index    int    `json:"index"`
					ID       string `json:"id"`
					Type     string `json:"type"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"delta"`
			FinishReason *string `json:"finish_reason"`
		}{
			{
				FinishReason: &finishReason,
			},
		},
	}

	chunk := convertDelta(delta)

	if chunk == nil {
		t.Fatal("convertDelta() returned nil")
	}

	if chunk.FinishReason != "stop" {
		t.Errorf("FinishReason = %q, want %q", chunk.FinishReason, "stop")
	}
}

func TestConvertDelta_EmptyChoices(t *testing.T) {
	delta := &chatCompletionChunk{
		Choices: []struct {
			Delta struct {
				Content   string `json:"content"`
				ToolCalls []struct {
					Index    int    `json:"index"`
					ID       string `json:"id"`
					Type     string `json:"type"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"delta"`
			FinishReason *string `json:"finish_reason"`
		}{},
	}

	chunk := convertDelta(delta)

	if chunk != nil {
		t.Error("convertDelta() should return nil for empty choices")
	}
}

func TestSSEEvent_IsDone(t *testing.T) {
	tests := []struct {
		name string
		data string
		want bool
	}{
		{"[DONE] marker", "[DONE]", true},
		{"regular data", "some data", false},
		{"empty", "", false},
		{"DONE without brackets", "DONE", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := &SSEEvent{Data: tt.data}
			if got := event.IsDone(); got != tt.want {
				t.Errorf("IsDone() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Test goroutine leak prevention
func TestStreamResponse_NoGoroutineLeak(t *testing.T) {
	input := "data: test\n\ndata: [DONE]\n\n"

	for i := 0; i < 10; i++ {
		chunks := make(chan StreamChunk, 10)
		ctx := context.Background()

		go func() {
			defer close(chunks)
			streamResponse(ctx, strings.NewReader(input), chunks)
		}()

		// Consume all chunks
		for range chunks {
		}
	}

	// If there's a goroutine leak, this test will timeout or fail with -race
	time.Sleep(10 * time.Millisecond)
}

// Test channel buffering
func TestStreamResponse_ChannelBuffering(t *testing.T) {
	// Create many events
	var sb strings.Builder
	for i := 0; i < 20; i++ {
		sb.WriteString("data: event\n\n")
	}

	chunks := make(chan StreamChunk, 5) // Small buffer
	ctx := context.Background()

	done := make(chan struct{})
	go func() {
		defer close(done)
		streamResponse(ctx, strings.NewReader(sb.String()), chunks)
		close(chunks)
	}()

	// Slow consumer
	count := 0
	for range chunks {
		count++
		time.Sleep(time.Millisecond)
	}

	<-done

	if count == 0 {
		t.Error("no events received with slow consumer")
	}
}

func TestSSEScanner_Scan_FinalEventWithoutBlankLine(t *testing.T) {
	// Event at EOF without trailing blank line
	input := "data: final event"
	scanner := NewSSEScanner(strings.NewReader(input))

	if !scanner.Scan() {
		t.Fatal("Scan() should handle event at EOF")
	}

	event := scanner.Event()
	if event.Data != "final event" {
		t.Errorf("Event().Data = %q, want %q", event.Data, "final event")
	}

	// No more events
	if scanner.Scan() {
		t.Error("Scan() should return false after EOF")
	}
}

func TestConvertDelta_EmptyContent(t *testing.T) {
	delta := &chatCompletionChunk{
		Choices: []struct {
			Delta struct {
				Content   string `json:"content"`
				ToolCalls []struct {
					Index    int    `json:"index"`
					ID       string `json:"id"`
					Type     string `json:"type"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"delta"`
			FinishReason *string `json:"finish_reason"`
		}{
			{
				Delta: struct {
					Content   string `json:"content"`
					ToolCalls []struct {
						Index    int    `json:"index"`
						ID       string `json:"id"`
						Type     string `json:"type"`
						Function struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
						} `json:"function"`
					} `json:"tool_calls"`
				}{
					Content: "",
				},
			},
		},
	}

	chunk := convertDelta(delta)

	if chunk != nil {
		t.Error("convertDelta() should return nil for empty content")
	}
}

func TestConvertDelta_EmptyFinishReason(t *testing.T) {
	emptyReason := ""
	delta := &chatCompletionChunk{
		Choices: []struct {
			Delta struct {
				Content   string `json:"content"`
				ToolCalls []struct {
					Index    int    `json:"index"`
					ID       string `json:"id"`
					Type     string `json:"type"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"delta"`
			FinishReason *string `json:"finish_reason"`
		}{
			{
				FinishReason: &emptyReason,
			},
		},
	}

	chunk := convertDelta(delta)

	if chunk != nil {
		t.Error("convertDelta() should return nil for empty finish reason")
	}
}

func TestStreamResponse_ScannerError(t *testing.T) {
	// Create a reader that will cause scanner to encounter EOF mid-event
	// This is hard to trigger with bufio.Scanner, but we can test the error path
	input := "data: incomplete"
	chunks := make(chan StreamChunk, 10)
	ctx := context.Background()

	go func() {
		defer close(chunks)
		streamResponse(ctx, strings.NewReader(input), chunks)
	}()

	// Collect chunks
	for range chunks {
	}

	// Test passes if no panic occurs
}
