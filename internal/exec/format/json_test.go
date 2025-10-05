package format

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

)

func TestJSONFormatter_FormatStart(t *testing.T) {
	tests := []struct {
		name   string
		prompt string
	}{
		{
			name:   "basic prompt",
			prompt: "Run tests",
		},
		{
			name:   "empty prompt",
			prompt: "",
		},
		{
			name:   "prompt with special characters",
			prompt: "Test \"quotes\" and \\backslashes\\",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewJSONFormatter()
			output := f.FormatStart(tt.prompt)

			// Verify it's valid JSON
			var decoded map[string]interface{}
			if err := json.Unmarshal([]byte(output), &decoded); err != nil {
				t.Fatalf("FormatStart() produced invalid JSON: %v", err)
			}

			// Verify structure
			if decoded["type"] != "start" {
				t.Errorf("type = %v, want start", decoded["type"])
			}
			if tt.prompt != "" && decoded["prompt"] != tt.prompt {
				t.Errorf("prompt = %v, want %v", decoded["prompt"], tt.prompt)
			}
		})
	}
}

func TestJSONFormatter_FormatDelta(t *testing.T) {
	tests := []struct {
		name  string
		delta string
	}{
		{
			name:  "simple delta",
			delta: "Reading files...",
		},
		{
			name:  "empty delta",
			delta: "",
		},
		{
			name:  "delta with newlines",
			delta: "Line 1\nLine 2\nLine 3",
		},
		{
			name:  "delta with unicode",
			delta: "✓ Tests passed 日本語",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewJSONFormatter()
			output := f.FormatDelta(tt.delta)

			if tt.delta == "" {
				if output != "" {
					t.Errorf("FormatDelta() = %q, want empty string for empty delta", output)
				}
				return
			}

			// Verify it's valid JSON
			var decoded map[string]interface{}
			if err := json.Unmarshal([]byte(output), &decoded); err != nil {
				t.Fatalf("FormatDelta() produced invalid JSON: %v", err)
			}

			// Verify structure
			if decoded["type"] != "delta" {
				t.Errorf("type = %v, want delta", decoded["type"])
			}
			if decoded["content"] != tt.delta {
				t.Errorf("content = %v, want %v", decoded["content"], tt.delta)
			}
		})
	}
}

func TestJSONFormatter_FormatComplete(t *testing.T) {
	tests := []struct {
		name   string
		result *ExecResult
	}{
		{
			name: "complete with all fields",
			result: &ExecResult{
				Status:        "complete",
				TokensUsed:    1234,
				Duration:      2 * time.Second,
				FilesModified: []string{"file1.go", "file2.go"},
				CommandsRun: []CommandLog{
					{Command: "go test", ExitCode: 0, Output: "PASS"},
					{Command: "make build", ExitCode: 1, Output: "Error"},
				},
				Messages: []Message{
					{Role: "user", Content: "Run tests", Timestamp: time.Now()},
					{Role: "assistant", Content: "Running...", Timestamp: time.Now()},
				},
			},
		},
		{
			name: "failed with error",
			result: &ExecResult{
				Status:     "failed",
				TokensUsed: 500,
				Duration:   1 * time.Second,
				Error:      errors.New("task failed"),
			},
		},
		{
			name: "empty result",
			result: &ExecResult{
				Status: "complete",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewJSONFormatter()
			output := f.FormatComplete(tt.result)

			// Verify it's valid JSON
			var decoded map[string]interface{}
			if err := json.Unmarshal([]byte(output), &decoded); err != nil {
				t.Fatalf("FormatComplete() produced invalid JSON: %v\nOutput: %s", err, output)
			}

			// Verify required fields
			if decoded["status"] != tt.result.Status {
				t.Errorf("status = %v, want %v", decoded["status"], tt.result.Status)
			}

			if decoded["tokens_used"] != float64(tt.result.TokensUsed) {
				t.Errorf("tokens_used = %v, want %v", decoded["tokens_used"], tt.result.TokensUsed)
			}

			// Verify duration_ms
			if tt.result.Duration > 0 {
				expectedMS := tt.result.Duration.Milliseconds()
				if decoded["duration_ms"] != float64(expectedMS) {
					t.Errorf("duration_ms = %v, want %v", decoded["duration_ms"], expectedMS)
				}
			}

			// Verify files_modified
			if len(tt.result.FilesModified) > 0 {
				files, ok := decoded["files_modified"].([]interface{})
				if !ok {
					t.Error("files_modified is not an array")
				} else if len(files) != len(tt.result.FilesModified) {
					t.Errorf("files_modified length = %v, want %v", len(files), len(tt.result.FilesModified))
				}
			}

			// Verify commands_executed
			if len(tt.result.CommandsRun) > 0 {
				commands, ok := decoded["commands_executed"].([]interface{})
				if !ok {
					t.Error("commands_executed is not an array")
				} else if len(commands) != len(tt.result.CommandsRun) {
					t.Errorf("commands_executed length = %v, want %v", len(commands), len(tt.result.CommandsRun))
				}
			}

			// Verify error handling
			if tt.result.Error != nil {
				if decoded["error"] == nil {
					t.Error("error field is nil when result has error")
				}
			}
		})
	}
}

func TestJSONFormatter_FormatError(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{
			name: "simple error",
			err:  errors.New("something went wrong"),
		},
		{
			name: "nil error",
			err:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewJSONFormatter()
			output := f.FormatError(tt.err)

			// Verify it's valid JSON
			var decoded map[string]interface{}
			if err := json.Unmarshal([]byte(output), &decoded); err != nil {
				t.Fatalf("FormatError() produced invalid JSON: %v", err)
			}

			// Verify structure
			if decoded["type"] != "error" {
				t.Errorf("type = %v, want error", decoded["type"])
			}

			if tt.err != nil {
				errorMsg, ok := decoded["error"].(string)
				if !ok {
					t.Error("error field is not a string")
				}
				if errorMsg != tt.err.Error() {
					t.Errorf("error = %v, want %v", errorMsg, tt.err.Error())
				}
			}
		})
	}
}

func TestJSONFormatter_ValidJSON(t *testing.T) {
	f := NewJSONFormatter()

	tests := []struct {
		name   string
		format func() string
	}{
		{
			name:   "FormatStart",
			format: func() string { return f.FormatStart("test") },
		},
		{
			name:   "FormatDelta",
			format: func() string { return f.FormatDelta("delta") },
		},
		{
			name: "FormatComplete",
			format: func() string {
				return f.FormatComplete(&ExecResult{Status: "complete"})
			},
		},
		{
			name:   "FormatError",
			format: func() string { return f.FormatError(errors.New("error")) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := tt.format()
			if output == "" {
				t.Skip("Empty output")
			}

			var decoded map[string]interface{}
			if err := json.Unmarshal([]byte(output), &decoded); err != nil {
				t.Errorf("%s produced invalid JSON: %v\nOutput: %s", tt.name, err, output)
			}
		})
	}
}

func TestJSONFormatter_Streaming(t *testing.T) {
	f := NewJSONFormatter()

	// Simulate streaming output
	outputs := []string{
		f.FormatStart("Run tests"),
		f.FormatDelta("Reading files..."),
		f.FormatDelta("Running tests..."),
		f.FormatComplete(&ExecResult{
			Status:     "complete",
			TokensUsed: 100,
		}),
	}

	// Each output should be valid JSON
	for i, output := range outputs {
		var decoded map[string]interface{}
		if err := json.Unmarshal([]byte(output), &decoded); err != nil {
			t.Errorf("Output %d is invalid JSON: %v\nOutput: %s", i, err, output)
		}
	}

	// Concatenated output should be JSON Lines format (newline-separated)
	combined := strings.Join(outputs, "")

	// Each line should be parseable
	for i, line := range outputs {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var decoded map[string]interface{}
		if err := json.Unmarshal([]byte(line), &decoded); err != nil {
			t.Errorf("Line %d is invalid JSON: %v", i, err)
		}
	}

	_ = combined
}

func TestJSONFormatter_SpecialCharacters(t *testing.T) {
	f := NewJSONFormatter()

	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "quotes",
			input: `Test "quotes" here`,
		},
		{
			name:  "backslashes",
			input: `C:\Windows\Path`,
		},
		{
			name:  "newlines",
			input: "Line 1\nLine 2\nLine 3",
		},
		{
			name:  "unicode",
			input: "Hello 世界 🚀",
		},
		{
			name:  "control characters",
			input: "Tab\there",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := f.FormatDelta(tt.input)

			var decoded map[string]interface{}
			if err := json.Unmarshal([]byte(output), &decoded); err != nil {
				t.Fatalf("Failed to parse JSON with special characters: %v\nInput: %s\nOutput: %s", err, tt.input, output)
			}

			// Verify the input was preserved
			content, ok := decoded["content"].(string)
			if !ok {
				t.Fatal("content field is not a string")
			}
			if content != tt.input {
				t.Errorf("content = %q, want %q", content, tt.input)
			}
		})
	}
}

func TestJSONFormatter_EmptySlices(t *testing.T) {
	f := NewJSONFormatter()

	result := &ExecResult{
		Status:        "complete",
		Messages:      []Message{},
		FilesModified: []string{},
		CommandsRun:   []CommandLog{},
	}

	output := f.FormatComplete(result)

	var decoded map[string]interface{}
	if err := json.Unmarshal([]byte(output), &decoded); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Empty slices should be arrays, not null
	files, ok := decoded["files_modified"].([]interface{})
	if !ok || files == nil {
		t.Error("files_modified should be an empty array, not null")
	}

	commands, ok := decoded["commands_executed"].([]interface{})
	if !ok || commands == nil {
		t.Error("commands_executed should be an empty array, not null")
	}
}

func TestJSONFormatter_Interface(t *testing.T) {
	// Verify JSONFormatter implements Formatter interface
	var _ Formatter = (*JSONFormatter)(nil)
}
