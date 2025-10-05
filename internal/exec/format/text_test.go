package format

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

func TestTextFormatter_FormatStart(t *testing.T) {
	tests := []struct {
		name    string
		prompt  string
		want    string
	}{
		{
			name:   "basic prompt",
			prompt: "Run tests",
			want:   "[Spin] Starting task: Run tests\n",
		},
		{
			name:   "empty prompt",
			prompt: "",
			want:   "[Spin] Starting task\n",
		},
		{
			name:   "long prompt",
			prompt: "This is a very long prompt that should still be displayed correctly",
			want:   "[Spin] Starting task: This is a very long prompt that should still be displayed correctly\n",
		},
		{
			name:   "prompt with special characters",
			prompt: "Run \"tests\" & check $VARS",
			want:   "[Spin] Starting task: Run \"tests\" & check $VARS\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewTextFormatter()
			got := f.FormatStart(tt.prompt)
			if got != tt.want {
				t.Errorf("FormatStart() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTextFormatter_FormatDelta(t *testing.T) {
	tests := []struct {
		name  string
		delta string
		want  string
	}{
		{
			name:  "simple delta",
			delta: "Reading files...",
			want:  "[Spin] Reading files...\n",
		},
		{
			name:  "empty delta",
			delta: "",
			want:  "",
		},
		{
			name:  "multiline delta",
			delta: "Line 1\nLine 2",
			want:  "[Spin] Line 1\nLine 2\n",
		},
		{
			name:  "delta with unicode",
			delta: "✓ Tests passed",
			want:  "[Spin] ✓ Tests passed\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewTextFormatter()
			got := f.FormatDelta(tt.delta)
			if got != tt.want {
				t.Errorf("FormatDelta() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTextFormatter_FormatComplete(t *testing.T) {
	tests := []struct {
		name   string
		result *ExecResult
		want   []string // Substrings that must appear
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
				},
			},
			want: []string{
				"[Spin] Task complete",
				"Summary:",
				"Status: complete",
				"Duration:",
				"Tokens: 1,234",
				"Files modified: 2",
				"file1.go",
				"file2.go",
				"Commands executed: 1",
				"go test (exit 0)",
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
			want: []string{
				"[Spin] Task failed",
				"Summary:",
				"Status: failed",
				"Error: task failed",
			},
		},
		{
			name: "timeout",
			result: &ExecResult{
				Status:     "timeout",
				TokensUsed: 1000,
				Duration:   5 * time.Minute,
			},
			want: []string{
				"[Spin] Task timed out",
				"Status: timeout",
			},
		},
		{
			name: "cancelled",
			result: &ExecResult{
				Status:     "cancelled",
				TokensUsed: 100,
				Duration:   500 * time.Millisecond,
			},
			want: []string{
				"[Spin] Task cancelled",
				"Status: cancelled",
			},
		},
		{
			name: "empty result",
			result: &ExecResult{
				Status: "complete",
			},
			want: []string{
				"[Spin] Task complete",
				"Summary:",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewTextFormatter()
			got := f.FormatComplete(tt.result)

			for _, substr := range tt.want {
				if !strings.Contains(got, substr) {
					t.Errorf("FormatComplete() missing substring %q\nGot: %s", substr, got)
				}
			}
		})
	}
}

func TestTextFormatter_FormatError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "simple error",
			err:  errors.New("something went wrong"),
			want: "[Spin] Error: something went wrong\n",
		},
		{
			name: "nil error",
			err:  nil,
			want: "[Spin] Error: <nil>\n",
		},
		{
			name: "wrapped error",
			err:  errors.New("inner error"),
			want: "[Spin] Error: inner error\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewTextFormatter()
			got := f.FormatError(tt.err)
			if got != tt.want {
				t.Errorf("FormatError() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTextFormatter_NoColor(t *testing.T) {
	// Set NO_COLOR environment variable
	oldNoColor := os.Getenv("NO_COLOR")
	defer func() {
		if oldNoColor == "" {
			os.Unsetenv("NO_COLOR")
		} else {
			os.Setenv("NO_COLOR", oldNoColor)
		}
	}()

	os.Setenv("NO_COLOR", "1")

	f := NewTextFormatter()
	result := &ExecResult{
		Status:     "complete",
		TokensUsed: 100,
		Duration:   time.Second,
	}

	output := f.FormatComplete(result)

	// Should not contain ANSI escape codes
	if strings.Contains(output, "\033[") {
		t.Error("FormatComplete() contains ANSI codes when NO_COLOR is set")
	}
}

func TestTextFormatter_Summary(t *testing.T) {
	result := &ExecResult{
		Status:        "complete",
		TokensUsed:    123456,
		Duration:      2*time.Minute + 30*time.Second,
		FilesModified: []string{"a.go", "b.go", "c.go"},
		CommandsRun: []CommandLog{
			{Command: "cmd1", ExitCode: 0},
			{Command: "cmd2", ExitCode: 1},
		},
	}

	f := NewTextFormatter()
	output := f.FormatComplete(result)

	// Verify number formatting
	if !strings.Contains(output, "123,456") {
		t.Error("FormatComplete() doesn't format large token numbers with commas")
	}

	// Verify duration formatting
	if !strings.Contains(output, "2m30s") {
		t.Error("FormatComplete() doesn't format duration correctly")
	}

	// Verify file count
	if !strings.Contains(output, "Files modified: 3") {
		t.Error("FormatComplete() doesn't show correct file count")
	}

	// Verify command count
	if !strings.Contains(output, "Commands executed: 2") {
		t.Error("FormatComplete() doesn't show correct command count")
	}
}

func TestTextFormatter_LongOutput(t *testing.T) {
	longOutput := strings.Repeat("x", 2000)
	result := &ExecResult{
		Status: "complete",
		CommandsRun: []CommandLog{
			{Command: "long-cmd", ExitCode: 0, Output: longOutput},
		},
	}

	f := NewTextFormatter()
	output := f.FormatComplete(result)

	// Output should be truncated (implementation detail)
	// Just verify it doesn't cause issues
	if output == "" {
		t.Error("FormatComplete() returned empty string for long output")
	}
}

func TestTextFormatter_SpecialCharacters(t *testing.T) {
	result := &ExecResult{
		Status: "complete",
		FilesModified: []string{
			"file with spaces.go",
			"file-with-dash.go",
			"file_with_underscore.go",
		},
	}

	f := NewTextFormatter()
	output := f.FormatComplete(result)

	// All file names should appear
	for _, file := range result.FilesModified {
		if !strings.Contains(output, file) {
			t.Errorf("FormatComplete() missing file %q", file)
		}
	}
}

func TestTextFormatter_Interface(t *testing.T) {
	// Verify TextFormatter implements Formatter interface
	var _ Formatter = (*TextFormatter)(nil)
}
