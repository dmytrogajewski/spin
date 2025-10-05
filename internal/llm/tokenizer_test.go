package llm

import (
	"strings"
	"testing"
)

func TestNewApproximateTokenizer(t *testing.T) {
	tok := NewApproximateTokenizer()
	if tok == nil {
		t.Fatal("NewApproximateTokenizer() returned nil")
	}
}

func TestApproximateTokenizer_Count(t *testing.T) {
	tok := NewApproximateTokenizer()

	tests := []struct {
		name     string
		text     string
		wantMin  int
		wantMax  int
		wantZero bool
	}{
		{
			name:     "empty string",
			text:     "",
			wantZero: true,
		},
		{
			name:    "short text",
			text:    "hello",
			wantMin: 1,
			wantMax: 3,
		},
		{
			name:    "simple sentence",
			text:    "The quick brown fox jumps over the lazy dog",
			wantMin: 8,
			wantMax: 15,
		},
		{
			name:    "longer text with punctuation",
			text:    "This is a longer piece of text. It has multiple sentences! And some punctuation?",
			wantMin: 15,
			wantMax: 25,
		},
		{
			name:    "code snippet",
			text:    `func main() { fmt.Println("hello") }`,
			wantMin: 8,
			wantMax: 15,
		},
		{
			name: "multiline code",
			text: `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}`,
			wantMin: 15,
			wantMax: 35,
		},
		{
			name:    "unicode text",
			text:    "Hello 世界 🌍",
			wantMin: 2,
			wantMax: 8,
		},
		{
			name:    "single character",
			text:    "a",
			wantMin: 1,
			wantMax: 1,
		},
		{
			name:    "whitespace only",
			text:    "   \n\t  ",
			wantMin: 1,
			wantMax: 3,
		},
		{
			name:    "special characters",
			text:    "!@#$%^&*()",
			wantMin: 2,
			wantMax: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tok.Count(tt.text)

			if tt.wantZero {
				if got != 0 {
					t.Errorf("Count() = %d, want 0 for empty string", got)
				}
				return
			}

			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("Count() = %d, want between %d and %d for text: %q",
					got, tt.wantMin, tt.wantMax, tt.text)
			}
		})
	}
}

func TestApproximateTokenizer_CountMessages(t *testing.T) {
	tok := NewApproximateTokenizer()

	tests := []struct {
		name     string
		messages []Message
		wantMin  int
		wantMax  int
	}{
		{
			name:     "empty messages",
			messages: []Message{},
			wantMin:  0,
			wantMax:  0,
		},
		{
			name: "single user message",
			messages: []Message{
				{Role: "user", Content: "Hello"},
			},
			wantMin: 5,
			wantMax: 12,
		},
		{
			name: "conversation",
			messages: []Message{
				{Role: "system", Content: "You are a helpful assistant."},
				{Role: "user", Content: "What is the weather?"},
				{Role: "assistant", Content: "I cannot access weather data."},
			},
			wantMin: 15,
			wantMax: 40,
		},
		{
			name: "message with tool calls",
			messages: []Message{
				{
					Role:    "assistant",
					Content: "",
					ToolCalls: []ToolCall{
						{
							ID:   "call_1",
							Type: "function",
							Function: FunctionCall{
								Name:      "get_weather",
								Arguments: `{"location":"London"}`,
							},
						},
					},
				},
			},
			wantMin: 10,
			wantMax: 30,
		},
		{
			name: "tool response",
			messages: []Message{
				{
					Role:       "tool",
					Content:    "Temperature: 20°C",
					ToolCallID: "call_1",
				},
			},
			wantMin: 8,
			wantMax: 20,
		},
		{
			name: "multiple tool calls",
			messages: []Message{
				{
					Role:    "assistant",
					Content: "",
					ToolCalls: []ToolCall{
						{
							ID:   "call_1",
							Type: "function",
							Function: FunctionCall{
								Name:      "function_a",
								Arguments: `{"arg":"value"}`,
							},
						},
						{
							ID:   "call_2",
							Type: "function",
							Function: FunctionCall{
								Name:      "function_b",
								Arguments: `{"arg":"other"}`,
							},
						},
					},
				},
			},
			wantMin: 15,
			wantMax: 50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tok.CountMessages(tt.messages)

			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("CountMessages() = %d, want between %d and %d",
					got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestIsLikelyCode(t *testing.T) {
	tests := []struct {
		name string
		text string
		want bool
	}{
		{
			name: "empty string",
			text: "",
			want: false,
		},
		{
			name: "short string",
			text: "hello",
			want: false,
		},
		{
			name: "natural language",
			text: "This is a normal sentence without any code.",
			want: false,
		},
		{
			name: "go function",
			text: "func main() { fmt.Println(\"hello\") }",
			want: true,
		},
		{
			name: "python code",
			text: "def calculate(x, y): return x + y",
			want: true,
		},
		{
			name: "javascript code",
			text: "const result = array.map(item => item * 2);",
			want: true,
		},
		{
			name: "json",
			text: `{"name": "John", "age": 30, "city": "New York"}`,
			want: true,
		},
		{
			name: "import statement",
			text: "import numpy as np\nimport pandas as pd",
			want: true,
		},
		{
			name: "class definition",
			text: "class Calculator { constructor() { } }",
			want: true,
		},
		{
			name: "brackets only",
			text: "((()))[[[{{{",
			want: true,
		},
		{
			name: "natural language with technical terms",
			text: "The function returns a value that represents the sum.",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isLikelyCode(tt.text)
			if got != tt.want {
				t.Errorf("isLikelyCode() = %v, want %v for text: %q",
					got, tt.want, tt.text)
			}
		})
	}
}

func TestGetRoleOverhead(t *testing.T) {
	tests := []struct {
		role     string
		wantMin  int
		wantZero bool
	}{
		{role: "system", wantMin: 1},
		{role: "user", wantMin: 1},
		{role: "assistant", wantMin: 1},
		{role: "tool", wantMin: 1},
		{role: "unknown", wantMin: 1},
		{role: "", wantMin: 1},
	}

	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			got := getRoleOverhead(tt.role)

			if tt.wantZero && got != 0 {
				t.Errorf("getRoleOverhead(%q) = %d, want 0", tt.role, got)
			}

			if !tt.wantZero && got < tt.wantMin {
				t.Errorf("getRoleOverhead(%q) = %d, want >= %d", tt.role, got, tt.wantMin)
			}
		})
	}
}

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		name    string
		req     CompletionRequest
		wantMin int
		wantMax int
	}{
		{
			name: "simple request",
			req: CompletionRequest{
				Messages: []Message{
					{Role: "user", Content: "Hello"},
				},
			},
			wantMin: 10,
			wantMax: 600,
		},
		{
			name: "request with max tokens",
			req: CompletionRequest{
				Messages: []Message{
					{Role: "user", Content: "Hello"},
				},
				MaxTokens: 100,
			},
			wantMin: 100,
			wantMax: 200,
		},
		{
			name: "request with tools",
			req: CompletionRequest{
				Messages: []Message{
					{Role: "user", Content: "What's the weather?"},
				},
				Tools: []Tool{
					{
						Type: "function",
						Function: Function{
							Name:        "get_weather",
							Description: "Get current weather for a location",
						},
					},
				},
			},
			wantMin: 20,
			wantMax: 700,
		},
		{
			name: "complex request",
			req: CompletionRequest{
				Messages: []Message{
					{Role: "system", Content: "You are a helpful coding assistant."},
					{Role: "user", Content: "Write a function to calculate fibonacci numbers"},
				},
				Tools: []Tool{
					{
						Type: "function",
						Function: Function{
							Name:        "execute_code",
							Description: "Execute Python code",
						},
					},
					{
						Type: "function",
						Function: Function{
							Name:        "read_file",
							Description: "Read file contents",
						},
					},
				},
				MaxTokens: 1000,
			},
			wantMin: 1000,
			wantMax: 1200,
		},
		{
			name: "empty request",
			req: CompletionRequest{
				Messages: []Message{},
			},
			wantMin: 500,
			wantMax: 600,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EstimateTokens(tt.req)

			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("EstimateTokens() = %d, want between %d and %d",
					got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestEstimateTokens_Accuracy(t *testing.T) {
	// Test that estimates are conservative (slightly overcount)
	// This is important for context window management

	req := CompletionRequest{
		Messages: []Message{
			{
				Role:    "user",
				Content: strings.Repeat("word ", 100), // ~400 chars = ~100 tokens
			},
		},
		MaxTokens: 200,
	}

	estimate := EstimateTokens(req)

	// Should be roughly 100 (content) + overhead + 200 (response) = ~310
	// Allow range 250-400 for conservative estimate
	if estimate < 250 || estimate > 400 {
		t.Errorf("EstimateTokens() = %d, expected conservative estimate in range 250-400", estimate)
	}
}

// BenchmarkCount benchmarks token counting performance
func BenchmarkCount(b *testing.B) {
	tok := NewApproximateTokenizer()
	text := "The quick brown fox jumps over the lazy dog. " +
		"This is a longer piece of text to benchmark token counting performance."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tok.Count(text)
	}
}

// BenchmarkCountMessages benchmarks message counting performance
func BenchmarkCountMessages(b *testing.B) {
	tok := NewApproximateTokenizer()
	messages := []Message{
		{Role: "system", Content: "You are a helpful assistant."},
		{Role: "user", Content: "What is the weather like today?"},
		{Role: "assistant", Content: "I don't have access to current weather data."},
		{Role: "user", Content: "Can you help me write some code?"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tok.CountMessages(messages)
	}
}

// BenchmarkEstimateTokens benchmarks full request estimation
func BenchmarkEstimateTokens(b *testing.B) {
	req := CompletionRequest{
		Messages: []Message{
			{Role: "system", Content: "You are a helpful coding assistant."},
			{Role: "user", Content: "Write a function to calculate fibonacci numbers"},
		},
		Tools: []Tool{
			{
				Type: "function",
				Function: Function{
					Name:        "execute_code",
					Description: "Execute Python code",
				},
			},
		},
		MaxTokens: 1000,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		EstimateTokens(req)
	}
}

// BenchmarkIsLikelyCode benchmarks code detection
func BenchmarkIsLikelyCode(b *testing.B) {
	text := `func main() {
		fmt.Println("Hello, World!")
	}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		isLikelyCode(text)
	}
}
