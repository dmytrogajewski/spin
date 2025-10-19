package compress

import (
	"testing"
)

func TestMessageClassifier_Classify(t *testing.T) {
	classifier := &MessageClassifier{}

	tests := []struct {
		name     string
		msg      Message
		expected MessageImportance
	}{
		{
			name:     "system message",
			msg:      &mockMessage{role: RoleSystem, content: "System prompt"},
			expected: ImportanceCritical,
		},
		{
			name:     "user message",
			msg:      &mockMessage{role: RoleUser, content: "User input"},
			expected: ImportanceCritical,
		},
		{
			name:     "tool message",
			msg:      &mockMessage{role: RoleTool, content: "Tool result"},
			expected: ImportanceCritical,
		},
		{
			name:     "message with tool calls",
			msg:      &mockMessage{role: RoleAssistant, content: "Response", toolCallCount: 1},
			expected: ImportanceCritical,
		},
		{
			name:     "error message",
			msg:      &mockMessage{role: RoleAssistant, content: "Error: something failed"},
			expected: ImportanceCritical,
		},
		{
			name:     "assistant with code block",
			msg:      &mockMessage{role: RoleAssistant, content: "Here's the code:\n```go\nfunc main() {}\n```"},
			expected: ImportanceHigh,
		},
		{
			name:     "assistant with diff",
			msg:      &mockMessage{role: RoleAssistant, content: "Here's the diff:\n@@ -1,3 +1,3 @@"},
			expected: ImportanceHigh,
		},
		{
			name:     "verbose assistant response",
			msg:      &mockMessage{role: RoleAssistant, content: string(make([]byte, 1001))},
			expected: ImportanceLow,
		},
		{
			name:     "empty assistant response",
			msg:      &mockMessage{role: RoleAssistant, content: "   "},
			expected: ImportanceLow,
		},
		{
			name:     "regular assistant response",
			msg:      &mockMessage{role: RoleAssistant, content: "Regular response"},
			expected: ImportanceMedium,
		},
		{
			name:     "unknown role",
			msg:      &mockMessage{role: "unknown", content: "Unknown role message"},
			expected: ImportanceLow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifier.Classify(tt.msg)
			if result != tt.expected {
				t.Errorf("MessageClassifier.Classify() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestMessageClassifier_IsError(t *testing.T) {
	classifier := &MessageClassifier{}

	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{"no error", "This is a normal message", false},
		{"error keyword", "Error: something went wrong", true},
		{"error space", "Error something went wrong", true},
		{"failed keyword", "Failed: operation failed", true},
		{"failed space", "Failed operation failed", true},
		{"exception keyword", "Exception: something happened", true},
		{"exception space", "Exception something happened", true},
		{"case insensitive", "ERROR: something went wrong", true},
		{"mixed case", "Failed: operation failed", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifier.isError(tt.content)
			if result != tt.expected {
				t.Errorf("MessageClassifier.isError() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestMessageClassifier_ClassifyAssistantMessage(t *testing.T) {
	classifier := &MessageClassifier{}

	tests := []struct {
		name     string
		content  string
		expected MessageImportance
	}{
		{"empty", "", ImportanceLow},
		{"whitespace", "   \n\t  ", ImportanceLow},
		{"with code", "```go\nfunc main() {}\n```", ImportanceHigh},
		{"with diff", "@@ -1,3 +1,3 @@", ImportanceHigh},
		{"verbose", string(make([]byte, 1001)), ImportanceLow},
		{"regular", "Regular response", ImportanceMedium},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifier.classifyAssistantMessage(tt.content)
			if result != tt.expected {
				t.Errorf("MessageClassifier.classifyAssistantMessage() = %v, want %v", result, tt.expected)
			}
		})
	}
}

type mockMessage struct {
	role          string
	content       string
	toolCallCount int
}

func (m *mockMessage) GetRole() string {
	return m.role
}

func (m *mockMessage) GetContent() string {
	return m.content
}

func (m *mockMessage) GetToolCallCount() int {
	return m.toolCallCount
}
