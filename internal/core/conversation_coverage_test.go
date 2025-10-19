package core

import (
	"context"
	"testing"
	"time"
)

func TestConversation_waitForResume(t *testing.T) {
	tests := []struct {
		name        string
		conv        *Conversation
		ctx         context.Context
		signalChan  <-chan ControlSignal
		wantTimeout bool
	}{
		{
			name: "resume signal received",
			conv: &Conversation{},
			ctx:  context.Background(),
			signalChan: func() <-chan ControlSignal {
				ch := make(chan ControlSignal, 1)
				ch <- SignalResume
				return ch
			}(),
			wantTimeout: false,
		},
		{
			name: "cancel signal received",
			conv: &Conversation{},
			ctx:  context.Background(),
			signalChan: func() <-chan ControlSignal {
				ch := make(chan ControlSignal, 1)
				ch <- SignalCancel
				return ch
			}(),
			wantTimeout: true, // Cancel signal causes context cancellation
		},
		{
			name: "context timeout",
			conv: &Conversation{},
			ctx: func() context.Context {
				ctx, _ := context.WithTimeout(context.Background(), 10*time.Millisecond)
				return ctx
			}(),
			signalChan: func() <-chan ControlSignal {
				ch := make(chan ControlSignal)
				return ch // Never sends signal
			}(),
			wantTimeout: true,
		},
		{
			name: "context cancelled",
			conv: &Conversation{},
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			}(),
			signalChan: func() <-chan ControlSignal {
				ch := make(chan ControlSignal)
				return ch // Never sends signal
			}(),
			wantTimeout: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.conv.waitForResume(tt.ctx, tt.signalChan)

			if tt.wantTimeout {
				if err == nil {
					t.Errorf("Conversation.waitForResume() expected timeout error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Conversation.waitForResume() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestConversation_GetMaxTokens(t *testing.T) {
	tests := []struct {
		name       string
		conv       *Conversation
		wantTokens int
	}{
		{
			name:       "default max tokens",
			conv:       &Conversation{},
			wantTokens: 16384, // Default value from DefaultConfig
		},
		{
			name: "with current task",
			conv: func() *Conversation {
				c := &Conversation{}
				// Create a mock task that returns custom tokens
				c.currentTask = &mockTask{maxTokens: 8192}
				return c
			}(),
			wantTokens: 8192,
		},
		{
			name: "with nil current task",
			conv: &Conversation{
				currentTask: nil,
			},
			wantTokens: 16384, // Fallback default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.conv.GetMaxTokens()

			if result != tt.wantTokens {
				t.Errorf("Conversation.GetMaxTokens() = %v, want %v", result, tt.wantTokens)
			}
		})
	}
}

func TestConversation_GetTokenCount(t *testing.T) {
	tests := []struct {
		name      string
		conv      *Conversation
		wantCount int
	}{
		{
			name:      "without history",
			conv:      &Conversation{},
			wantCount: 0,
		},
		{
			name: "with history",
			conv: func() *Conversation {
				c := &Conversation{}
				c.history = &History{}
				// Add some messages to simulate token count
				c.history.AddMessage(Message{
					Role:    RoleUser,
					Content: "Hello",
					Tokens:  5,
				})
				c.history.AddMessage(Message{
					Role:    RoleAssistant,
					Content: "Hi there!",
					Tokens:  3,
				})
				return c
			}(),
			wantCount: 8, // 5 + 3 tokens
		},
		{
			name: "with zero token history",
			conv: func() *Conversation {
				c := &Conversation{}
				c.history = &History{}
				return c
			}(),
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.conv.GetTokenCount()

			if result != tt.wantCount {
				t.Errorf("Conversation.GetTokenCount() = %v, want %v", result, tt.wantCount)
			}
		})
	}
}

// mockTask implements a minimal Task interface for testing
type mockTask struct {
	maxTokens int
}

func (m *mockTask) Name() string {
	return "mock"
}

func (m *mockTask) SystemPrompt() string {
	return "mock prompt"
}

func (m *mockTask) AllowedTools() []string {
	return []string{}
}

func (m *mockTask) MaxTokens() int {
	return m.maxTokens
}

func (m *mockTask) Validate() error {
	return nil
}
