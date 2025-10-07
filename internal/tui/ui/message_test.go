package ui

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMessageRole_String(t *testing.T) {
	tests := []struct {
		role MessageRole
		want string
	}{
		{RoleUser, "user"},
		{RoleAssistant, "assistant"},
		{RoleSystem, "system"},
		{RoleTool, "tool"},
	}

	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			got := string(tt.role)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMessage_IsStreaming(t *testing.T) {
	msg := Message{
		Role:      RoleAssistant,
		Content:   "Hello",
		Streaming: true,
	}

	assert.True(t, msg.Streaming)

	msg.Streaming = false
	assert.False(t, msg.Streaming)
}

func TestMessage_WithToolCall(t *testing.T) {
	toolCall := &ToolCall{
		Name: "shell",
		Arguments: map[string]interface{}{
			"command": "ls -la",
		},
		ID: "call_123",
	}

	msg := Message{
		Role:     RoleTool,
		Content:  "Executing shell command",
		ToolCall: toolCall,
	}

	assert.NotNil(t, msg.ToolCall)
	assert.Equal(t, "shell", msg.ToolCall.Name)
	assert.Equal(t, "call_123", msg.ToolCall.ID)
	assert.Equal(t, "ls -la", msg.ToolCall.Arguments["command"])
}

func TestMessage_WithToolResult(t *testing.T) {
	toolResult := &ToolResult{
		ToolCallID: "call_123",
		Output:     "file1.txt\nfile2.txt",
		Error:      "",
	}

	msg := Message{
		Role:       RoleTool,
		Content:    "Command completed",
		ToolResult: toolResult,
	}

	assert.NotNil(t, msg.ToolResult)
	assert.Equal(t, "call_123", msg.ToolResult.ToolCallID)
	assert.Contains(t, msg.ToolResult.Output, "file1.txt")
	assert.Empty(t, msg.ToolResult.Error)
}

func TestMessage_WithThinking(t *testing.T) {
	msg := Message{
		Role:     RoleAssistant,
		Content:  "I'll help with that",
		Thinking: "User needs assistance with file operations",
	}

	assert.Equal(t, "User needs assistance with file operations", msg.Thinking)
}

func TestMessage_Timestamp(t *testing.T) {
	now := time.Now()
	msg := Message{
		Role:      RoleUser,
		Content:   "Hello",
		Timestamp: now,
	}

	assert.Equal(t, now, msg.Timestamp)
}

func TestNewUserMessage(t *testing.T) {
	content := "Hello, AI!"
	msg := NewUserMessage(content)

	assert.Equal(t, RoleUser, msg.Role)
	assert.Equal(t, content, msg.Content)
	assert.False(t, msg.Streaming)
	assert.WithinDuration(t, time.Now(), msg.Timestamp, time.Second)
}

func TestNewAssistantMessage(t *testing.T) {
	content := "Hello, human!"
	msg := NewAssistantMessage(content)

	assert.Equal(t, RoleAssistant, msg.Role)
	assert.Equal(t, content, msg.Content)
	assert.False(t, msg.Streaming)
	assert.WithinDuration(t, time.Now(), msg.Timestamp, time.Second)
}

func TestNewStreamingMessage(t *testing.T) {
	content := "Hel"
	msg := NewStreamingMessage(content)

	assert.Equal(t, RoleAssistant, msg.Role)
	assert.Equal(t, content, msg.Content)
	assert.True(t, msg.Streaming)
	assert.WithinDuration(t, time.Now(), msg.Timestamp, time.Second)
}

func TestToolCall_IsValid(t *testing.T) {
	tests := []struct {
		name    string
		tc      ToolCall
		wantErr bool
	}{
		{
			name: "valid tool call",
			tc: ToolCall{
				Name:      "shell",
				Arguments: map[string]interface{}{"command": "ls"},
				ID:        "call_123",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			tc: ToolCall{
				Arguments: map[string]interface{}{"command": "ls"},
				ID:        "call_123",
			},
			wantErr: true,
		},
		{
			name: "missing ID",
			tc: ToolCall{
				Name:      "shell",
				Arguments: map[string]interface{}{"command": "ls"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.tc.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
