package agent

import (
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/ace/bullet"
	"github.com/dmytrogajewski/spin/internal/ace/generator"
	"github.com/dmytrogajewski/spin/internal/message"
)

func TestExtractInitialQuery(t *testing.T) {
	t.Run("returns first user message content", func(t *testing.T) {
		messages := []message.Message{
			{Role: message.RoleUser, Content: "install nodejs", Timestamp: time.Now()},
		}

		got := extractInitialQuery(messages)
		want := "install nodejs"

		if got != want {
			t.Errorf("extractInitialQuery() = %q, want %q", got, want)
		}
	})

	t.Run("returns empty string when no user messages", func(t *testing.T) {
		messages := []message.Message{
			{Role: message.RoleAssistant, Content: "I can help", Timestamp: time.Now()},
			{Role: message.RoleSystem, Content: "System prompt", Timestamp: time.Now()},
		}

		got := extractInitialQuery(messages)
		want := ""

		if got != want {
			t.Errorf("extractInitialQuery() = %q, want %q", got, want)
		}
	})

	t.Run("returns user message after system message", func(t *testing.T) {
		messages := []message.Message{
			{Role: message.RoleSystem, Content: "System prompt", Timestamp: time.Now()},
			{Role: message.RoleUser, Content: "debug the app", Timestamp: time.Now()},
		}

		got := extractInitialQuery(messages)
		want := "debug the app"

		if got != want {
			t.Errorf("extractInitialQuery() = %q, want %q", got, want)
		}
	})
}

func TestExtractNewSteps(t *testing.T) {
	t.Run("extracts single assistant reasoning message", func(t *testing.T) {
		ts := time.Now()
		messages := []message.Message{
			{Role: message.RoleAssistant, Content: "I'll check the file", Timestamp: ts},
		}

		steps := extractNewSteps(messages, 0)

		if len(steps) != 1 {
			t.Fatalf("expected 1 step, got %d", len(steps))
		}

		want := generator.TrajectoryStep{
			StepNumber: 0,
			Type:       "reasoning",
			Content:    "I'll check the file",
			Timestamp:  ts,
		}

		got := steps[0]
		if got.StepNumber != want.StepNumber {
			t.Errorf("StepNumber = %d, want %d", got.StepNumber, want.StepNumber)
		}
		if got.Type != want.Type {
			t.Errorf("Type = %q, want %q", got.Type, want.Type)
		}
		if got.Content != want.Content {
			t.Errorf("Content = %q, want %q", got.Content, want.Content)
		}
		if !got.Timestamp.Equal(want.Timestamp) {
			t.Errorf("Timestamp = %v, want %v", got.Timestamp, want.Timestamp)
		}
	})

	t.Run("extracts tool call from assistant message", func(t *testing.T) {
		ts := time.Now()
		messages := []message.Message{
			{
				Role:      message.RoleAssistant,
				Timestamp: ts,
				ToolCalls: []message.ToolCall{
					{
						ID:   "call_1",
						Type: "function",
						Function: message.ToolCallFunction{
							Name:      "read_file",
							Arguments: `{"path": "main.go"}`,
						},
					},
				},
			},
		}

		steps := extractNewSteps(messages, 0)

		if len(steps) != 1 {
			t.Fatalf("expected 1 step, got %d", len(steps))
		}

		got := steps[0]
		if got.StepNumber != 0 {
			t.Errorf("StepNumber = %d, want 0", got.StepNumber)
		}
		if got.Type != "tool_call" {
			t.Errorf("Type = %q, want %q", got.Type, "tool_call")
		}
		wantContent := "Tool: read_file\nArguments: {\"path\": \"main.go\"}"
		if got.Content != wantContent {
			t.Errorf("Content = %q, want %q", got.Content, wantContent)
		}
	})

	t.Run("extracts tool result message", func(t *testing.T) {
		ts := time.Now()
		messages := []message.Message{
			{
				Role:       message.RoleTool,
				ToolCallID: "call_1",
				Content:    "package main\n\nfunc main() {}",
				Timestamp:  ts,
			},
		}

		steps := extractNewSteps(messages, 0)

		if len(steps) != 1 {
			t.Fatalf("expected 1 step, got %d", len(steps))
		}

		got := steps[0]
		if got.StepNumber != 0 {
			t.Errorf("StepNumber = %d, want 0", got.StepNumber)
		}
		if got.Type != "tool_result" {
			t.Errorf("Type = %q, want %q", got.Type, "tool_result")
		}
		wantContent := "Tool Result (ID: call_1):\npackage main\n\nfunc main() {}"
		if got.Content != wantContent {
			t.Errorf("Content = %q, want %q", got.Content, wantContent)
		}
	})
}

func TestExtractBulletIDs(t *testing.T) {
	t.Run("extracts IDs from bullets", func(t *testing.T) {
		bullets := []*bullet.Bullet{
			{ID: "b1", Content: "First bullet"},
			{ID: "b2", Content: "Second bullet"},
			{ID: "b3", Content: "Third bullet"},
		}

		ids := extractBulletIDs(bullets)

		if len(ids) != 3 {
			t.Fatalf("expected 3 IDs, got %d", len(ids))
		}

		want := []string{"b1", "b2", "b3"}
		for i, id := range ids {
			if id != want[i] {
				t.Errorf("ids[%d] = %q, want %q", i, id, want[i])
			}
		}
	})

	t.Run("returns empty slice for nil bullets", func(t *testing.T) {
		ids := extractBulletIDs(nil)

		if len(ids) != 0 {
			t.Errorf("expected empty slice, got %d IDs", len(ids))
		}
	})
}
