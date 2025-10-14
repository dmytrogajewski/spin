package compress

import (
	"testing"
)

func TestClassifier_UserMessage(t *testing.T) {
	classifier := &MessageClassifier{}

	msg := &CompressibleMessage{
		Role:    RoleUser,
		Content: "Please help me debug this issue",
	}

	importance := classifier.Classify(msg)

	if importance != ImportanceCritical {
		t.Errorf("expected ImportanceCritical, got %v", importance)
	}
}

func TestClassifier_ToolResult(t *testing.T) {
	classifier := &MessageClassifier{}

	msg := &CompressibleMessage{
		Role:    RoleTool,
		Content: "Command output: test passed",
	}

	importance := classifier.Classify(msg)

	if importance != ImportanceCritical {
		t.Errorf("expected ImportanceCritical for tool result, got %v", importance)
	}
}

func TestClassifier_MessageWithToolCalls(t *testing.T) {
	classifier := &MessageClassifier{}

	msg := &CompressibleMessage{
		Role:          RoleAssistant,
		Content:       "Let me read that file for you",
		ToolCallCount: 1, // Has tool calls
	}

	importance := classifier.Classify(msg)

	if importance != ImportanceCritical {
		t.Errorf("expected ImportanceCritical for message with tool calls, got %v", importance)
	}
}

func TestClassifier_ErrorMessage(t *testing.T) {
	classifier := &MessageClassifier{}

	tests := []struct {
		name    string
		content string
	}{
		{
			name:    "error in content (lowercase)",
			content: "error: file not found",
		},
		{
			name:    "error in content (uppercase)",
			content: "ERROR: permission denied",
		},
		{
			name:    "error in content (mixed)",
			content: "An Error occurred during processing",
		},
		{
			name:    "failed in content",
			content: "failed: command execution failed",
		},
		{
			name:    "exception in content",
			content: "exception: null pointer exception",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &CompressibleMessage{
				Role:    RoleAssistant,
				Content: tt.content,
			}

			importance := classifier.Classify(msg)

			if importance != ImportanceCritical {
				t.Errorf("expected ImportanceCritical for error message, got %v", importance)
			}
		})
	}
}

func TestClassifier_CodeBlock(t *testing.T) {
	classifier := &MessageClassifier{}

	msg := &CompressibleMessage{
		Role: RoleAssistant,
		Content: `Let me show you the implementation:

` + "```go" + `
func main() {
    fmt.Println("Hello")
}
` + "```",
	}

	importance := classifier.Classify(msg)

	if importance != ImportanceHigh {
		t.Errorf("expected ImportanceHigh for code block, got %v", importance)
	}
}

func TestClassifier_DiffBlock(t *testing.T) {
	classifier := &MessageClassifier{}

	msg := &CompressibleMessage{
		Role: RoleAssistant,
		Content: `Here's the patch:

@@ -1,3 +1,4 @@
 package main
+import "log"
 
 func main() {`,
	}

	importance := classifier.Classify(msg)

	if importance != ImportanceHigh {
		t.Errorf("expected ImportanceHigh for diff block, got %v", importance)
	}
}

func TestClassifier_VerboseResponse(t *testing.T) {
	classifier := &MessageClassifier{}

	// Generate a long response (> 1000 chars) without code
	longContent := `I understand you need help with this issue. Let me think through this carefully. `
	for i := 0; i < 20; i++ {
		longContent += `First, we need to consider the architecture and how the components interact. `
	}

	msg := &CompressibleMessage{
		Role:    RoleAssistant,
		Content: longContent,
	}

	importance := classifier.Classify(msg)

	if importance != ImportanceLow {
		t.Errorf("expected ImportanceLow for verbose response, got %v", importance)
	}
}

func TestClassifier_RegularResponse(t *testing.T) {
	classifier := &MessageClassifier{}

	msg := &CompressibleMessage{
		Role:    RoleAssistant,
		Content: "I'll help you with that task.",
	}

	importance := classifier.Classify(msg)

	if importance != ImportanceMedium {
		t.Errorf("expected ImportanceMedium for regular response, got %v", importance)
	}
}

func TestClassifier_SystemMessage(t *testing.T) {
	classifier := &MessageClassifier{}

	msg := &CompressibleMessage{
		Role:    RoleSystem,
		Content: "You are a helpful coding assistant",
	}

	importance := classifier.Classify(msg)

	// System messages should be critical (always preserved)
	if importance != ImportanceCritical {
		t.Errorf("expected ImportanceCritical for system message, got %v", importance)
	}
}

func TestClassifier_EmptyMessage(t *testing.T) {
	classifier := &MessageClassifier{}

	msg := &CompressibleMessage{
		Role:    RoleAssistant,
		Content: "",
	}

	importance := classifier.Classify(msg)

	// Empty messages should be low importance
	if importance != ImportanceLow {
		t.Errorf("expected ImportanceLow for empty message, got %v", importance)
	}
}

// Benchmark classifier performance
func BenchmarkClassifier(b *testing.B) {
	classifier := &MessageClassifier{}
	msg := &CompressibleMessage{
		Role:    RoleAssistant,
		Content: "This is a test message for benchmarking",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = classifier.Classify(msg)
	}
}
