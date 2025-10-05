package format

import (
	"errors"
	"testing"
	"time"
)

func TestExecResult_Creation(t *testing.T) {
	tests := []struct {
		name string
		result *ExecResult
		wantValid bool
	}{
		{
			name: "valid complete result",
			result: &ExecResult{
				Status:        "complete",
				Messages:      []Message{{Role: "user", Content: "test"}},
				FilesModified: []string{"file.go"},
				CommandsRun:   []CommandLog{{Command: "test", ExitCode: 0}},
				TokensUsed:    100,
				Duration:      time.Second,
			},
			wantValid: true,
		},
		{
			name: "valid failed result with error",
			result: &ExecResult{
				Status:     "failed",
				Messages:   []Message{},
				TokensUsed: 50,
				Duration:   500 * time.Millisecond,
				Error:      errors.New("task failed"),
			},
			wantValid: true,
		},
		{
			name: "empty result",
			result: &ExecResult{
				Status:   "complete",
				Messages: []Message{},
			},
			wantValid: true,
		},
		{
			name: "timeout result",
			result: &ExecResult{
				Status:     "timeout",
				Messages:   []Message{{Role: "user", Content: "test"}},
				TokensUsed: 200,
				Duration:   5 * time.Minute,
			},
			wantValid: true,
		},
		{
			name: "cancelled result",
			result: &ExecResult{
				Status:     "cancelled",
				Messages:   []Message{{Role: "user", Content: "test"}},
				TokensUsed: 75,
				Duration:   2 * time.Second,
			},
			wantValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify all fields are accessible
			_ = tt.result.Status
			_ = tt.result.Messages
			_ = tt.result.FilesModified
			_ = tt.result.CommandsRun
			_ = tt.result.TokensUsed
			_ = tt.result.Duration
			_ = tt.result.Error
		})
	}
}

func TestMessage_Creation(t *testing.T) {
	tests := []struct {
		name    string
		role    string
		content string
	}{
		{
			name:    "user message",
			role:    "user",
			content: "Run tests",
		},
		{
			name:    "assistant message",
			role:    "assistant",
			content: "I'll run the tests for you",
		},
		{
			name:    "system message",
			role:    "system",
			content: "Task started",
		},
		{
			name:    "empty content",
			role:    "user",
			content: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Now()
			msg := Message{
				Role:      tt.role,
				Content:   tt.content,
				Timestamp: now,
			}

			if msg.Role != tt.role {
				t.Errorf("Role = %v, want %v", msg.Role, tt.role)
			}
			if msg.Content != tt.content {
				t.Errorf("Content = %v, want %v", msg.Content, tt.content)
			}
			if msg.Timestamp != now {
				t.Errorf("Timestamp = %v, want %v", msg.Timestamp, now)
			}
		})
	}
}

func TestCommandLog_Creation(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		exitCode int
		output   string
	}{
		{
			name:     "successful command",
			command:  "go test ./...",
			exitCode: 0,
			output:   "PASS",
		},
		{
			name:     "failed command",
			command:  "make build",
			exitCode: 1,
			output:   "Error: build failed",
		},
		{
			name:     "command with no output",
			command:  "echo hello",
			exitCode: 0,
			output:   "",
		},
		{
			name:     "command with long output",
			command:  "cat large-file.txt",
			exitCode: 0,
			output:   string(make([]byte, 2000)), // Large output
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := CommandLog{
				Command:  tt.command,
				ExitCode: tt.exitCode,
				Output:   tt.output,
			}

			if log.Command != tt.command {
				t.Errorf("Command = %v, want %v", log.Command, tt.command)
			}
			if log.ExitCode != tt.exitCode {
				t.Errorf("ExitCode = %v, want %v", log.ExitCode, tt.exitCode)
			}
			if log.Output != tt.output {
				t.Errorf("Output length = %v, want %v", len(log.Output), len(tt.output))
			}
		})
	}
}

func TestOutputFormat_Constants(t *testing.T) {
	if FormatText != "text" {
		t.Errorf("FormatText = %v, want text", FormatText)
	}
	if FormatJSON != "json" {
		t.Errorf("FormatJSON = %v, want json", FormatJSON)
	}
}

func TestExecResult_NilFields(t *testing.T) {
	// Verify we can create result with nil/zero fields
	result := &ExecResult{
		Status:        "complete",
		Messages:      nil, // nil slice
		FilesModified: nil,
		CommandsRun:   nil,
		TokensUsed:    0,
		Duration:      0,
		Error:         nil,
	}

	if result.Messages == nil {
		result.Messages = []Message{}
	}
	if result.FilesModified == nil {
		result.FilesModified = []string{}
	}
	if result.CommandsRun == nil {
		result.CommandsRun = []CommandLog{}
	}

	// Should not panic
	_ = len(result.Messages)
	_ = len(result.FilesModified)
	_ = len(result.CommandsRun)
}

func TestMessage_Timestamp(t *testing.T) {
	before := time.Now()
	time.Sleep(time.Millisecond)

	msg := Message{
		Role:      "user",
		Content:   "test",
		Timestamp: time.Now(),
	}

	time.Sleep(time.Millisecond)
	after := time.Now()

	if msg.Timestamp.Before(before) {
		t.Error("Timestamp is before creation time")
	}
	if msg.Timestamp.After(after) {
		t.Error("Timestamp is after creation time")
	}
}

func TestCommandLog_ExitCodes(t *testing.T) {
	testCodes := []int{0, 1, 2, 127, 255, -1}

	for _, code := range testCodes {
		log := CommandLog{
			Command:  "test",
			ExitCode: code,
		}

		if log.ExitCode != code {
			t.Errorf("ExitCode = %v, want %v", log.ExitCode, code)
		}
	}
}
