package adapter_test

import (
	"testing"

	"github.com/dmytrogajewski/spin/internal/contexteng/adapter"
	"github.com/dmytrogajewski/spin/internal/contexteng/observation"
	"github.com/dmytrogajewski/spin/internal/message"
)

func TestObservationAdapter_SummarizeToolResults_Empty(t *testing.T) {
	t.Parallel()

	sum := observation.NewSummarizer()
	adapt := adapter.NewObservationAdapter(sum)

	var msgs []message.Message

	result := adapt.SummarizeToolResults(msgs)

	if len(result) != 0 {
		t.Errorf("expected 0 messages, got %d", len(result))
	}
}

func TestObservationAdapter_SummarizeToolResults_NonToolUntouched(t *testing.T) {
	t.Parallel()

	sum := observation.NewSummarizer()
	adapt := adapter.NewObservationAdapter(sum)

	msgs := []message.Message{
		{Role: message.RoleUser, Content: "hello"},
		{Role: message.RoleAssistant, Content: "hi there"},
		{Role: message.RoleSystem, Content: "system prompt"},
	}

	result := adapt.SummarizeToolResults(msgs)
	if len(result) != len(msgs) {
		t.Fatalf("expected %d messages, got %d", len(msgs), len(result))
	}

	for i, msg := range result {
		if msg.Content != msgs[i].Content {
			t.Errorf("message %d: expected %q, got %q", i, msgs[i].Content, msg.Content)
		}
	}
}

func TestObservationAdapter_SummarizeToolResults_ToolMessageSummarized(t *testing.T) {
	t.Parallel()

	sum := observation.NewSummarizer()
	adapt := adapter.NewObservationAdapter(sum)

	// Long tool output that exceeds ShortOutputMax (100 chars).
	longOutput := make([]byte, 200)
	for i := range longOutput {
		longOutput[i] = 'a'
	}

	msgs := []message.Message{
		{Role: message.RoleUser, Content: "do something"},
		{Role: message.RoleTool, Content: string(longOutput), Name: "read_file"},
		{Role: message.RoleAssistant, Content: "done"},
	}

	result := adapt.SummarizeToolResults(msgs)
	if len(result) != len(msgs) {
		t.Fatalf("expected %d messages, got %d", len(msgs), len(result))
	}

	// User and assistant unchanged.
	if result[0].Content != msgs[0].Content {
		t.Errorf("user message changed: %q", result[0].Content)
	}

	if result[2].Content != msgs[2].Content {
		t.Errorf("assistant message changed: %q", result[2].Content)
	}

	// Tool message should be summarized (not equal to original).
	if result[1].Content == string(longOutput) {
		t.Error("expected tool message to be summarized")
	}

	if result[1].Role != message.RoleTool {
		t.Errorf("expected role tool, got %s", result[1].Role)
	}
}

func TestObservationAdapter_SummarizeToolResults_ShortToolVerbatim(t *testing.T) {
	t.Parallel()

	sum := observation.NewSummarizer()
	adapt := adapter.NewObservationAdapter(sum)

	msgs := []message.Message{
		{Role: message.RoleTool, Content: "ok", Name: "shell_command"},
	}

	result := adapt.SummarizeToolResults(msgs)
	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result))
	}

	// Short output should pass through verbatim for shell_command.
	if result[0].Content != "ok" {
		t.Errorf("expected verbatim %q, got %q", "ok", result[0].Content)
	}
}

func TestObservationAdapter_SummarizeToolResults_EmptyContent(t *testing.T) {
	t.Parallel()

	sum := observation.NewSummarizer()
	adapt := adapter.NewObservationAdapter(sum)

	msgs := []message.Message{
		{Role: message.RoleTool, Content: "", Name: "read_file"},
	}

	result := adapt.SummarizeToolResults(msgs)
	if result[0].Content != "" {
		t.Errorf("expected empty content preserved, got %q", result[0].Content)
	}
}

func TestObservationAdapter_NilSummarizer(t *testing.T) {
	t.Parallel()

	adapt := adapter.NewObservationAdapter(nil)

	msgs := []message.Message{
		{Role: message.RoleTool, Content: "output"},
	}

	result := adapt.SummarizeToolResults(msgs)
	if len(result) != len(msgs) {
		t.Fatalf("expected %d messages, got %d", len(msgs), len(result))
	}

	if result[0].Content != msgs[0].Content {
		t.Error("nil summarizer should return messages unchanged")
	}
}

func TestObservationAdapter_PreservesMessageFields(t *testing.T) {
	t.Parallel()

	sum := observation.NewSummarizer()
	adapt := adapter.NewObservationAdapter(sum)

	msgs := []message.Message{
		{
			Role:       message.RoleTool,
			Content:    "short",
			Name:       "shell_command",
			ToolCallID: "call_123",
		},
	}

	result := adapt.SummarizeToolResults(msgs)

	if result[0].ToolCallID != "call_123" {
		t.Errorf("expected ToolCallID preserved, got %q", result[0].ToolCallID)
	}

	if result[0].Name != "shell_command" {
		t.Errorf("expected Name preserved, got %q", result[0].Name)
	}
}
