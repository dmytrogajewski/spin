package compress

import (
	"testing"

	"github.com/dmytrogajewski/spin/internal/message"
)

func TestNewClassifier(t *testing.T) {
	c := NewClassifier()
	if c == nil {
		t.Fatal("NewClassifier returned nil")
	}
	if c.verboseThreshold != 1000 {
		t.Errorf("expected default verboseThreshold 1000, got %d", c.verboseThreshold)
	}
}

func TestNewClassifierWithOptions(t *testing.T) {
	c := NewClassifierWithOptions(WithVerboseThreshold(500))
	if c.verboseThreshold != 500 {
		t.Errorf("expected verboseThreshold 500, got %d", c.verboseThreshold)
	}
}

func TestClassifier_SystemMessage(t *testing.T) {
	c := NewClassifier()
	msg := message.Message{
		Role:    message.RoleSystem,
		Content: "You are a helpful assistant.",
	}

	importance := c.Classify(msg)
	if importance != ImportanceCritical {
		t.Errorf("expected ImportanceCritical for system message, got %s", importance)
	}
}

func TestClassifier_UserMessage(t *testing.T) {
	c := NewClassifier()
	msg := message.Message{
		Role:    message.RoleUser,
		Content: "Help me fix this bug.",
	}

	importance := c.Classify(msg)
	if importance != ImportanceCritical {
		t.Errorf("expected ImportanceCritical for user message, got %s", importance)
	}
}

func TestClassifier_ToolResult(t *testing.T) {
	c := NewClassifier()
	msg := message.Message{
		Role:       message.RoleTool,
		Content:    "File contents here...",
		ToolCallID: "call_123",
	}

	importance := c.Classify(msg)
	if importance != ImportanceCritical {
		t.Errorf("expected ImportanceCritical for tool result, got %s", importance)
	}
}

func TestClassifier_ToolCall(t *testing.T) {
	c := NewClassifier()
	msg := message.Message{
		Role:    message.RoleAssistant,
		Content: "Let me read that file.",
		ToolCalls: []message.ToolCall{
			{
				ID:   "call_123",
				Type: "function",
				Function: message.ToolCallFunction{
					Name:      "read_file",
					Arguments: `{"path": "test.go"}`,
				},
			},
		},
	}

	importance := c.Classify(msg)
	if importance != ImportanceCritical {
		t.Errorf("expected ImportanceCritical for message with tool calls, got %s", importance)
	}
}

func TestClassifier_ErrorMessage(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{"error prefix", "Error: file not found"},
		{"failed prefix", "Failed: authentication failed"},
		{"exception", "Exception: null pointer"},
		{"panic", "Panic: runtime error"},
		{"fatal", "Fatal: cannot continue"},
		{"cannot", "Cannot connect to server"},
		{"could not", "Could not find module"},
		{"unable to", "Unable to parse JSON"},
	}

	c := NewClassifier()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := message.Message{
				Role:    message.RoleAssistant,
				Content: tt.content,
			}

			importance := c.Classify(msg)
			if importance != ImportanceCritical {
				t.Errorf("expected ImportanceCritical for error message %q, got %s", tt.content, importance)
			}
		})
	}
}

func TestClassifier_ErrorMetadata(t *testing.T) {
	c := NewClassifier()

	// Metadata is map[string]string, so we only test string "true"
	msg := message.Message{
		Role:     message.RoleAssistant,
		Content:  "Something happened",
		Metadata: message.Metadata{"is_error": "true"},
	}

	importance := c.Classify(msg)
	if importance != ImportanceCritical {
		t.Errorf("expected ImportanceCritical for error metadata, got %s", importance)
	}
}

func TestClassifier_CodeBlock(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{
			"fenced code block",
			"Here's the code:\n```go\nfunc main() {}\n```",
		},
		{
			"diff markers",
			"The patch:\n@@ -1,3 +1,4 @@\n+new line",
		},
		{
			"indented code",
			"Example:\n    line 1\n    line 2\n    line 3\n    line 4",
		},
	}

	c := NewClassifier()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := message.Message{
				Role:    message.RoleAssistant,
				Content: tt.content,
			}

			importance := c.Classify(msg)
			if importance != ImportanceHigh {
				t.Errorf("expected ImportanceHigh for code block, got %s", importance)
			}
		})
	}
}

func TestClassifier_RegularAssistant(t *testing.T) {
	c := NewClassifier()
	msg := message.Message{
		Role:    message.RoleAssistant,
		Content: "I'll help you with that task.",
	}

	importance := c.Classify(msg)
	if importance != ImportanceMedium {
		t.Errorf("expected ImportanceMedium for regular assistant message, got %s", importance)
	}
}

func TestClassifier_VerboseAssistant(t *testing.T) {
	c := NewClassifierWithOptions(WithVerboseThreshold(100))

	// Create a long message without code
	longContent := "This is a very long explanation. "
	for i := 0; i < 10; i++ {
		longContent += "Let me explain in more detail how this works and why it's important. "
	}

	msg := message.Message{
		Role:    message.RoleAssistant,
		Content: longContent,
	}

	importance := c.Classify(msg)
	if importance != ImportanceLow {
		t.Errorf("expected ImportanceLow for verbose assistant message, got %s", importance)
	}
}

func TestClassifier_VerboseWithCode(t *testing.T) {
	c := NewClassifierWithOptions(WithVerboseThreshold(100))

	// Create a long message WITH code
	longContent := "This is a very long explanation. "
	for i := 0; i < 10; i++ {
		longContent += "Let me explain in more detail how this works and why it's important. "
	}
	longContent += "\n```go\nfunc main() {}\n```"

	msg := message.Message{
		Role:    message.RoleAssistant,
		Content: longContent,
	}

	importance := c.Classify(msg)
	if importance != ImportanceHigh {
		t.Errorf("expected ImportanceHigh for long message with code, got %s", importance)
	}
}

func TestClassifier_UnknownRole(t *testing.T) {
	c := NewClassifier()
	msg := message.Message{
		Role:    "unknown",
		Content: "Some content",
	}

	importance := c.Classify(msg)
	if importance != ImportanceLow {
		t.Errorf("expected ImportanceLow for unknown role, got %s", importance)
	}
}

func TestImportance_String(t *testing.T) {
	tests := []struct {
		importance Importance
		expected   string
	}{
		{ImportanceLow, "low"},
		{ImportanceMedium, "medium"},
		{ImportanceHigh, "high"},
		{ImportanceCritical, "critical"},
		{Importance(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.importance.String(); got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestClassifier_TabIndentedCode(t *testing.T) {
	c := NewClassifier()
	msg := message.Message{
		Role:    message.RoleAssistant,
		Content: "Example:\n\tline 1\n\tline 2\n\tline 3",
	}

	importance := c.Classify(msg)
	if importance != ImportanceHigh {
		t.Errorf("expected ImportanceHigh for tab-indented code, got %s", importance)
	}
}

func TestClassifier_NilMetadata(t *testing.T) {
	c := NewClassifier()
	msg := message.Message{
		Role:     message.RoleAssistant,
		Content:  "Normal message",
		Metadata: nil,
	}

	// Should not panic
	importance := c.Classify(msg)
	if importance != ImportanceMedium {
		t.Errorf("expected ImportanceMedium, got %s", importance)
	}
}
