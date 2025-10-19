package core

import (
	"context"
	"testing"

	"github.com/dmytrogajewski/spin/internal/core/history/compress"
)

// mockTokenizer is a simple tokenizer for testing
type mockTokenizer struct{}

func (m *mockTokenizer) Count(text string) int {
	return len(text) / 4 // Simple approximation: 4 chars = 1 token
}

func (m *mockTokenizer) CountMessages(messages []Message) int {
	total := 0
	for _, msg := range messages {
		total += m.Count(msg.Content)
	}
	return total
}

func TestHistory_CompressLocked(t *testing.T) {
	tests := []struct {
		name      string
		messages  []Message
		maxTokens int
		wantErr   bool
	}{
		{
			name:      "empty history",
			messages:  []Message{},
			maxTokens: 1000,
			wantErr:   false,
		},
		{
			name: "single message",
			messages: []Message{
				{ID: "1", Role: RoleUser, Content: "test", Tokens: 10},
			},
			maxTokens: 1000,
			wantErr:   false,
		},
		{
			name: "multiple messages",
			messages: []Message{
				{ID: "1", Role: RoleUser, Content: "test1", Tokens: 10},
				{ID: "2", Role: RoleAssistant, Content: "response1", Tokens: 15},
				{ID: "3", Role: RoleUser, Content: "test2", Tokens: 10},
			},
			maxTokens: 1000,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock compressor
			mockCompressor := &compress.HybridCompressor{}

			h := &History{
				messages:   tt.messages,
				maxTokens:  tt.maxTokens,
				compressor: mockCompressor,
				tokenizer:  &mockTokenizer{},
				config: &HistoryConfig{
					CompressionThreshold: 0.8,
				},
			}

			ctx := context.Background()
			err := h.compressLocked(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("compressLocked() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHistory_EmitCompressionEvent(t *testing.T) {
	tests := []struct {
		name         string
		beforeCount  int
		beforeTokens int
		afterCount   int
		afterTokens  int
		hasEmitter   bool
	}{
		{
			name:         "with emitter",
			beforeCount:  10,
			beforeTokens: 1000,
			afterCount:   5,
			afterTokens:  500,
			hasEmitter:   true,
		},
		{
			name:         "without emitter",
			beforeCount:  10,
			beforeTokens: 1000,
			afterCount:   5,
			afterTokens:  500,
			hasEmitter:   false,
		},
		{
			name:         "zero before count",
			beforeCount:  0,
			beforeTokens: 0,
			afterCount:   0,
			afterTokens:  0,
			hasEmitter:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var emitter *EventEmitter
			if tt.hasEmitter {
				emitter = NewEventEmitter(10)
			}

			h := &History{
				emitter: emitter,
			}

			// Run emitCompressionEvent in goroutine to avoid blocking
			done := make(chan bool, 1)
			go func() {
				h.emitCompressionEvent(tt.beforeCount, tt.beforeTokens, tt.afterCount, tt.afterTokens)
				done <- true
			}()

			if tt.hasEmitter {
				// Check that event was emitted
				select {
				case event := <-emitter.Events():
					if event.Type != EventInfo {
						t.Errorf("emitCompressionEvent() event type = %v, want %v", event.Type, EventInfo)
					}
				case <-done:
					// Completed without emitting (shouldn't happen)
					t.Error("emitCompressionEvent() completed without emitting event")
				}
			} else {
				// Should complete immediately when no emitter
				<-done
			}
		})
	}
}

func TestHistory_ToCompressibleMessages(t *testing.T) {
	messages := []Message{
		{
			ID:      "1",
			Role:    RoleUser,
			Content: "test message",
			Tokens:  10,
			ToolCalls: []ToolCall{
				{ID: "tool1", Function: ToolCallFunction{Name: "test_tool"}},
			},
		},
		{
			ID:      "2",
			Role:    RoleAssistant,
			Content: "response",
			Tokens:  15,
		},
	}

	h := &History{}
	result := h.toCompressibleMessages(messages)

	if len(result) != len(messages) {
		t.Errorf("toCompressibleMessages() returned %d messages, want %d", len(result), len(messages))
	}

	// Check first message
	if result[0].ID != "1" {
		t.Errorf("toCompressibleMessages() message[0].ID = %v, want 1", result[0].ID)
	}
	if result[0].Role != string(RoleUser) {
		t.Errorf("toCompressibleMessages() message[0].Role = %v, want %v", result[0].Role, RoleUser)
	}
	if result[0].ToolCallCount != 1 {
		t.Errorf("toCompressibleMessages() message[0].ToolCallCount = %v, want 1", result[0].ToolCallCount)
	}

	// Check second message
	if result[1].ID != "2" {
		t.Errorf("toCompressibleMessages() message[1].ID = %v, want 2", result[1].ID)
	}
	if result[1].Tokens != 15 {
		t.Errorf("toCompressibleMessages() message[1].Tokens = %v, want 15", result[1].Tokens)
	}
}

func TestHistory_FromCompressibleMessages(t *testing.T) {
	compressible := []compress.CompressibleMessage{
		{
			ID:            "1",
			Role:          string(RoleUser),
			Content:       "test",
			ToolCallCount: 2,
			Tokens:        10,
		},
		{
			ID:            "2",
			Role:          string(RoleAssistant),
			Content:       "response",
			ToolCallCount: 0,
			Tokens:        15,
		},
	}

	h := &History{}
	result := h.fromCompressibleMessages(compressible)

	if len(result) != len(compressible) {
		t.Errorf("fromCompressibleMessages() returned %d messages, want %d", len(result), len(compressible))
	}

	// Check first message
	if result[0].ID != "1" {
		t.Errorf("fromCompressibleMessages() message[0].ID = %v, want 1", result[0].ID)
	}
	if result[0].Role != RoleUser {
		t.Errorf("fromCompressibleMessages() message[0].Role = %v, want %v", result[0].Role, RoleUser)
	}
	if result[0].Tokens != 10 {
		t.Errorf("fromCompressibleMessages() message[0].Tokens = %v, want 10", result[0].Tokens)
	}

	// Check second message
	if result[1].ID != "2" {
		t.Errorf("fromCompressibleMessages() message[1].ID = %v, want 2", result[1].ID)
	}
	if result[1].Content != "response" {
		t.Errorf("fromCompressibleMessages() message[1].Content = %v, want response", result[1].Content)
	}
}

func TestHistory_FindMessageByID(t *testing.T) {
	messages := []Message{
		{ID: "1", Content: "first"},
		{ID: "2", Content: "second"},
		{ID: "3", Content: "third"},
	}

	h := &History{
		messages: messages,
	}

	tests := []struct {
		name    string
		id      string
		want    *Message
		wantNil bool
	}{
		{
			name:    "found first",
			id:      "1",
			want:    &messages[0],
			wantNil: false,
		},
		{
			name:    "found middle",
			id:      "2",
			want:    &messages[1],
			wantNil: false,
		},
		{
			name:    "found last",
			id:      "3",
			want:    &messages[2],
			wantNil: false,
		},
		{
			name:    "not found",
			id:      "999",
			want:    nil,
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := h.findMessageByID(tt.id)
			if tt.wantNil {
				if got != nil {
					t.Errorf("findMessageByID() = %v, want nil", got)
				}
			} else {
				if got == nil {
					t.Errorf("findMessageByID() = nil, want non-nil")
					return
				}
				if got.ID != tt.want.ID {
					t.Errorf("findMessageByID().ID = %v, want %v", got.ID, tt.want.ID)
				}
			}
		})
	}
}

func TestHistory_MessageCount(t *testing.T) {
	tests := []struct {
		name     string
		messages []Message
		want     int
	}{
		{
			name:     "empty",
			messages: []Message{},
			want:     0,
		},
		{
			name: "single message",
			messages: []Message{
				{ID: "1", Content: "test"},
			},
			want: 1,
		},
		{
			name: "multiple messages",
			messages: []Message{
				{ID: "1", Content: "test1"},
				{ID: "2", Content: "test2"},
				{ID: "3", Content: "test3"},
			},
			want: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &History{
				messages: tt.messages,
			}

			got := len(h.messages)
			if got != tt.want {
				t.Errorf("message count = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHistory_ToCompressibleMessages_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		messages []Message
	}{
		{
			name:     "empty messages",
			messages: []Message{},
		},
		{
			name: "message with multiple tool calls",
			messages: []Message{
				{
					ID:      "1",
					Role:    RoleUser,
					Content: "test",
					ToolCalls: []ToolCall{
						{ID: "tool1", Function: ToolCallFunction{Name: "t1"}},
						{ID: "tool2", Function: ToolCallFunction{Name: "t2"}},
						{ID: "tool3", Function: ToolCallFunction{Name: "t3"}},
					},
				},
			},
		},
		{
			name: "message with zero tokens",
			messages: []Message{
				{ID: "1", Role: RoleUser, Content: "", Tokens: 0},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &History{}
			result := h.toCompressibleMessages(tt.messages)

			if len(result) != len(tt.messages) {
				t.Errorf("toCompressibleMessages() length = %v, want %v", len(result), len(tt.messages))
			}

			// Verify tool call count matches
			for i, msg := range tt.messages {
				if result[i].ToolCallCount != len(msg.ToolCalls) {
					t.Errorf("message[%d].ToolCallCount = %v, want %v", i, result[i].ToolCallCount, len(msg.ToolCalls))
				}
			}
		})
	}
}

func TestHistory_FromCompressibleMessages_EdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		compressible []compress.CompressibleMessage
	}{
		{
			name:         "empty",
			compressible: []compress.CompressibleMessage{},
		},
		{
			name: "with tool call count",
			compressible: []compress.CompressibleMessage{
				{ID: "1", Role: "user", Content: "test", ToolCallCount: 5, Tokens: 10},
			},
		},
		{
			name: "system role",
			compressible: []compress.CompressibleMessage{
				{ID: "1", Role: "system", Content: "system msg", ToolCallCount: 0, Tokens: 20},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &History{}
			result := h.fromCompressibleMessages(tt.compressible)

			if len(result) != len(tt.compressible) {
				t.Errorf("fromCompressibleMessages() length = %v, want %v", len(result), len(tt.compressible))
			}

			// Verify basic fields are preserved
			for i, msg := range tt.compressible {
				if result[i].ID != msg.ID {
					t.Errorf("message[%d].ID = %v, want %v", i, result[i].ID, msg.ID)
				}
				if result[i].Content != msg.Content {
					t.Errorf("message[%d].Content = %v, want %v", i, result[i].Content, msg.Content)
				}
			}
		})
	}
}

func TestHistory_CompressLocked_WithEmitter(t *testing.T) {
	emitter := NewEventEmitter(10)

	messages := []Message{
		{ID: "1", Role: RoleUser, Content: "test1", Tokens: 10},
		{ID: "2", Role: RoleAssistant, Content: "response1", Tokens: 15},
	}

	h := &History{
		messages:   messages,
		maxTokens:  1000,
		compressor: &compress.HybridCompressor{},
		tokenizer:  &mockTokenizer{},
		emitter:    emitter,
		config: &HistoryConfig{
			CompressionThreshold: 0.8,
		},
	}

	ctx := context.Background()

	// Run compressLocked in goroutine to allow event emission
	errChan := make(chan error, 1)
	go func() {
		errChan <- h.compressLocked(ctx)
	}()

	// Check that compression event was emitted
	select {
	case event := <-emitter.Events():
		if event.Type != EventInfo {
			t.Errorf("compression event type = %v, want %v", event.Type, EventInfo)
		}
		data, ok := event.Data.(SystemEventData)
		if !ok {
			t.Errorf("compression event data type = %T, want SystemEventData", event.Data)
		}
		if data.Message != "Context history compressed" {
			t.Errorf("compression event message = %v, want 'Context history compressed'", data.Message)
		}
	case err := <-errChan:
		if err != nil {
			t.Errorf("compressLocked() error = %v", err)
		}
		t.Error("compressLocked() completed without emitting compression event")
	}
}
