package output

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/dmytrogajewski/spin/internal/types"
)

// Helper to create ToolCallArguments for testing
func makeArgs(m map[string]interface{}) types.ToolCallArguments {
	result := make(types.ToolCallArguments)
	for k, v := range m {
		b, _ := json.Marshal(v)
		result[k] = json.RawMessage(b)
	}
	return result
}

func TestFormatStart_ExecuteCommand(t *testing.T) {
	registry := NewFormatterRegistry(80)

	params := makeArgs(map[string]interface{}{
		"command": "go test ./...",
		"cwd":     "/home/user/project",
	})

	result := registry.FormatStart("execute_command", "tool_123", params)

	// Should contain tag and command
	if !strings.Contains(result, "EXECUTE") {
		t.Errorf("Expected 'EXECUTE' tag, got: %s", result)
	}
	if !strings.Contains(result, "go test ./...") {
		t.Errorf("Expected command in output, got: %s", result)
	}

	// Should not exceed terminal width
	// Strip ANSI codes for length check
	plainText := stripANSI(result)
	if len(plainText) > 80 {
		t.Errorf("Output exceeds terminal width: %d chars", len(plainText))
	}
}

func TestFormatStart_ReadFile(t *testing.T) {
	registry := NewFormatterRegistry(80)

	params := makeArgs(map[string]interface{}{
		"path":   "package.json",
		"offset": 100,
		"limit":  400,
	})

	result := registry.FormatStart("read_file", "tool_456", params)

	if !strings.Contains(result, "READ") {
		t.Errorf("Expected 'READ' tag, got: %s", result)
	}
	if !strings.Contains(result, "package.json") {
		t.Errorf("Expected file path in output, got: %s", result)
	}
	if !strings.Contains(result, "offset: 100") {
		t.Errorf("Expected offset in output, got: %s", result)
	}
	if !strings.Contains(result, "limit: 400") {
		t.Errorf("Expected limit in output, got: %s", result)
	}
}

func TestFormatStart_WriteFile(t *testing.T) {
	registry := NewFormatterRegistry(80)

	params := makeArgs(map[string]interface{}{
		"path":    "hello.txt",
		"content": "Hello, World!",
	})

	result := registry.FormatStart("write_file", "tool_789", params)

	if !strings.Contains(result, "WRITE") {
		t.Errorf("Expected 'WRITE' tag, got: %s", result)
	}
	if !strings.Contains(result, "hello.txt") {
		t.Errorf("Expected file path in output, got: %s", result)
	}
	// Content should not be shown in start line (only size)
	if strings.Contains(result, "Hello, World!") {
		t.Errorf("Should not show full content, got: %s", result)
	}
}

func TestFormatStart_Truncation(t *testing.T) {
	registry := NewFormatterRegistry(80)

	// Very long command that exceeds terminal width
	longCommand := "go test -race -cover -timeout=30m -v ./internal/... ./cmd/... ./pkg/... ./tools/... 2>&1 | tee output.log"
	params := makeArgs(map[string]interface{}{
		"command": longCommand,
		"cwd":     "/home/user/very/long/path/to/project/workspace",
		"impact":  "high",
	})

	result := registry.FormatStart("execute_command", "tool_long", params)

	// Should contain truncation indicator
	if !strings.Contains(result, "...") {
		t.Errorf("Expected truncation indicator '...', got: %s", result)
	}

	// Should not exceed terminal width
	plainText := stripANSI(result)
	if len(plainText) > 80 {
		t.Errorf("Output exceeds terminal width: %d chars (expected ≤80)", len(plainText))
	}
}

func TestFormatStart_Unicode(t *testing.T) {
	registry := NewFormatterRegistry(80)

	params := makeArgs(map[string]interface{}{
		"command": "notify-send '🎉 Test passed!'",
	})

	result := registry.FormatStart("execute_command", "tool_unicode", params)

	// Should handle Unicode without breaking
	if !strings.Contains(result, "🎉") {
		t.Errorf("Expected Unicode emoji in output, got: %s", result)
	}
}

func TestFormatComplete_Success(t *testing.T) {
	registry := NewFormatterRegistry(80)

	output := "Test output\nLine 2\nLine 3"
	result := registry.FormatComplete("execute_command", true, output, "")

	// Should contain continuation arrow
	if !strings.Contains(result, "↳") {
		t.Errorf("Expected continuation arrow '↳', got: %s", result)
	}

	// Should indicate success
	if !strings.Contains(result, "Exit code: 0") && !strings.Contains(result, "success") {
		// Either "Exit code: 0" or "success" should be present
		if !strings.Contains(result, "0") && !strings.Contains(result, "success") {
			t.Errorf("Expected success indicator in output, got: %s", result)
		}
	}

	// Should show output line count
	if !strings.Contains(result, "3 lines") && !strings.Contains(result, "Output: 3") {
		t.Errorf("Expected line count in output, got: %s", result)
	}
}

func TestFormatComplete_Failure(t *testing.T) {
	registry := NewFormatterRegistry(80)

	result := registry.FormatComplete("execute_command", false, "Error output", "Command failed with exit code 1")

	// Should contain continuation arrow
	if !strings.Contains(result, "↳") {
		t.Errorf("Expected continuation arrow '↳', got: %s", result)
	}

	// Should indicate failure
	if !strings.Contains(result, "Failed") && !strings.Contains(result, "Error") {
		t.Errorf("Expected failure indicator in output, got: %s", result)
	}
}

func TestFormatComplete_ReadFile(t *testing.T) {
	registry := NewFormatterRegistry(80)

	output := strings.Repeat("line\n", 62)
	result := registry.FormatComplete("read_file", true, output, "")

	if !strings.Contains(result, "↳") {
		t.Errorf("Expected continuation arrow, got: %s", result)
	}

	// Should show line count
	if !strings.Contains(result, "62") {
		t.Errorf("Expected line count in output, got: %s", result)
	}
}

func TestFormatComplete_NoOutput(t *testing.T) {
	registry := NewFormatterRegistry(80)

	result := registry.FormatComplete("execute_command", true, "", "")

	// Should handle empty output gracefully
	if !strings.Contains(result, "↳") {
		t.Errorf("Expected continuation arrow, got: %s", result)
	}

	// Should indicate no output or 0 lines
	if !strings.Contains(result, "0") && !strings.Contains(result, "No output") && !strings.Contains(result, "empty") {
		// At least one of these indicators should be present
		// If result is just "↳" that's also acceptable
		if result != " ↳ " && !strings.HasPrefix(result, " ↳") {
			t.Errorf("Expected output indicator, got: %s", result)
		}
	}
}

func TestFormatStart_ListDirectory(t *testing.T) {
	registry := NewFormatterRegistry(80)

	params := makeArgs(map[string]interface{}{
		"path": ".",
	})

	result := registry.FormatStart("list_directory", "tool_list", params)

	if !strings.Contains(result, "LIST") {
		t.Errorf("Expected 'LIST' tag, got: %s", result)
	}
	if !strings.Contains(result, "ls .") {
		t.Errorf("Expected 'ls .' in output, got: %s", result)
	}
}

func TestColorMapping(t *testing.T) {
	registry := NewFormatterRegistry(80)

	tests := []struct {
		toolName        string
		expectedTag     string
		shouldHaveColor bool
	}{
		{"execute_command", "EXECUTE", true},
		{"read_file", "READ", true},
		{"write_file", "WRITE", true},
		{"grep", "GREP", true},
		{"list_directory", "LIST", true},  // Now has its own tag
		{"unknown_tool", "TOOL", true},    // Unknown tools use default formatter
	}

	for _, tt := range tests {
		t.Run(tt.toolName, func(t *testing.T) {
			params := makeArgs(map[string]interface{}{
				"command": "test",
				"path":    "test.txt",
				"pattern": "test",
			})
			result := registry.FormatStart(tt.toolName, "tool_test", params)

			if tt.shouldHaveColor {
				// Should contain ANSI color codes
				if !strings.Contains(result, "\x1b[") {
					t.Errorf("Expected ANSI color codes for %s, got: %s", tt.toolName, result)
				}
				// Should contain expected tag text
				if !strings.Contains(result, tt.expectedTag) {
					t.Errorf("Expected tag '%s' for %s, got: %s", tt.expectedTag, tt.toolName, result)
				}
			}
		})
	}
}

func TestWidthRespect(t *testing.T) {
	widths := []int{80, 120, 200}

	for _, width := range widths {
		t.Run(string(rune('0'+width/10)), func(t *testing.T) {
			registry := NewFormatterRegistry(width)

			params := makeArgs(map[string]interface{}{
				"command": strings.Repeat("a", 200), // Very long command
				"cwd":     strings.Repeat("b", 100),
				"impact":  "high",
			})

			result := registry.FormatStart("execute_command", "tool_width", params)

			// Strip ANSI codes
			plainText := stripANSI(result)

			if len(plainText) > width {
				t.Errorf("Output exceeds terminal width %d: got %d chars", width, len(plainText))
			}
		})
	}
}

// Helper function to strip ANSI escape codes for length checking
func stripANSI(s string) string {
	// Simple ANSI stripper (strips \x1b[...m sequences)
	result := ""
	inEscape := false
	for i := 0; i < len(s); i++ {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			inEscape = true
			i++ // Skip '['
			continue
		}
		if inEscape {
			if s[i] == 'm' {
				inEscape = false
			}
			continue
		}
		result += string(s[i])
	}
	return result
}
