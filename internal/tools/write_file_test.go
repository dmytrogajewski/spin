package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/pkg/alg/stringsx"
)

func TestWriteFileTool(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	tool := NewWriteFileTool()

	tests := []struct {
		name        string
		params      map[string]any
		wantErr     bool
		verifyWrite bool
	}{
		{
			name: "write new file",
			params: map[string]any{
				"path":    filepath.Join(tmpDir, "new.txt"),
				"content": "test content",
			},
			verifyWrite: true,
		},
		{
			name: "overwrite existing file",
			params: map[string]any{
				"path":    filepath.Join(tmpDir, "overwrite.txt"),
				"content": "new content",
			},
			verifyWrite: true,
		},
		{
			name: "create parent directories",
			params: map[string]any{
				"path":    filepath.Join(tmpDir, "subdir", "nested", "file.txt"),
				"content": "content in nested directory",
			},
			verifyWrite: true,
		},
		{
			name:    "missing path parameter",
			params:  map[string]any{"content": "test"},
			wantErr: true,
		},
		{
			name:    "missing content parameter",
			params:  map[string]any{"path": filepath.Join(tmpDir, "test.txt")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			runWriteFileSubtest(t, tool, tt.params, tt.wantErr, tt.verifyWrite)
		})
	}
}

func runWriteFileSubtest(t *testing.T, tool Tool, params map[string]any, wantErr, verifyWrite bool) {
	t.Helper()

	p, _ := FromMap(params)
	result, err := tool.Execute(context.Background(), p)

	if wantErr {
		if err == nil && result.Success {
			t.Error("expected error but got success")
		}

		return
	}

	if err != nil {
		t.Errorf("unexpected error: %v", err)

		return
	}

	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)

		return
	}

	if verifyWrite {
		verifyWrittenFile(t, params)
	}
}

func verifyWrittenFile(t *testing.T, params map[string]any) {
	t.Helper()

	path, _ := params["path"].(string)
	expectedContent, _ := params["content"].(string)

	content, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Errorf("failed to read written file: %v", readErr)

		return
	}

	if string(content) != expectedContent {
		t.Errorf("expected content %q, got %q", expectedContent, string(content))
	}
}

func TestWriteFileTool_ErrorCases(t *testing.T) {
	t.Parallel()

	tool := NewWriteFileTool()

	tests := []struct {
		name   string
		params map[string]any
	}{
		{
			name:   "missing path",
			params: map[string]any{"content": "test"},
		},
		{
			name:   "missing content",
			params: map[string]any{"path": "test.txt"},
		},
		{
			name:   "invalid path type",
			params: map[string]any{"path": 123, "content": "test"},
		},
		{
			name:   "invalid content type",
			params: map[string]any{"path": "test.txt", "content": 123},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			params, _ := FromMap(tt.params)

			result, err := tool.Execute(context.Background(), params)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if result.Success {
				t.Error("expected failure result")
			}
		})
	}
}

// writeFileApprovalCase describes a test case for write file approval checking.
type writeFileApprovalCase struct {
	name string
	path string
	want RiskLevel
}

func runWriteFileApprovalTests(t *testing.T, tool *WriteFileTool, cases []writeFileApprovalCase) {
	t.Helper()

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			params, _ := FromMap(map[string]any{
				"path":    tt.path,
				"content": "test",
			})

			needs := tool.CheckApproval(params)

			if !needs.Required {
				t.Errorf("CheckApproval should require approval for %s", tt.path)
			}

			if needs.Risk != tt.want {
				t.Errorf("CheckApproval Risk = %v, want %v", needs.Risk, tt.want)
			}

			if needs.Reason == "" {
				t.Error("CheckApproval should provide a reason")
			}
		})
	}
}

// TestWriteFileTool_DetectsTruncatedContent tests that the write tool warns when
// content appears truncated (e.g., LLM hit max_tokens mid-output).
// Reproduces bug: LLM response was cut at 9990 bytes mid-expression
// ("execute!(stdout, c") and write_file silently wrote broken code.
func TestWriteFileTool_DetectsTruncatedContent(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	tool := NewWriteFileTool()

	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{
			name:    "truncated mid-expression",
			content: "fn main() {\n    execute!(stdout, c",
			wantErr: true,
		},
		{
			name:    "truncated with unclosed brace",
			content: "fn main() {\n    let x = 5;\n    if x > 0 {",
			wantErr: true,
		},
		{
			name:    "truncated with unclosed string",
			content: `fn main() { println!("hello`,
			wantErr: true,
		},
		{
			name:    "valid complete content",
			content: "fn main() {\n    println!(\"hello\");\n}\n",
			wantErr: false,
		},
		{
			name:    "plain text is fine",
			content: "Hello, world!\n",
			wantErr: false,
		},
		{
			name:    "empty content is fine",
			content: "",
			wantErr: false,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(tmpDir, fmt.Sprintf("trunc_%d.txt", i))
			params, _ := FromMap(map[string]any{
				"path":    path,
				"content": tt.content,
			})

			result, err := tool.Execute(context.Background(), params)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantErr {
				if result.Success {
					t.Error("expected failure for truncated content, but got success")
				}
			} else {
				if !result.Success {
					t.Errorf("expected success, got error: %s", result.Error)
				}
			}
		})
	}
}

// TestWriteFileTool_CreatesNewFileWithTracker tests that write_file can create
// a new file that doesn't exist yet, even when a FileTracker is active.
// Reproduces bug: agent tried to WRITE tetris/src/main.rs (new file in empty dir),
// got "file not previously read; read the file before editing" because the tracker
// requires a prior read even for files that don't exist.
func TestWriteFileTool_CreatesNewFileWithTracker(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	newFilePath := filepath.Join(tmpDir, "brand_new_file.rs")

	tool := NewWriteFileTool()
	tracker := NewFileTracker()
	tool.SetTracker(tracker)

	// File does not exist — no prior read is possible.
	params, _ := FromMap(map[string]any{
		"path":    newFilePath,
		"content": "fn main() {}\n",
	})

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Errorf("expected success creating new file with tracker, got error: %s", result.Error)
	}

	// Verify file was actually created.
	content, readErr := os.ReadFile(newFilePath)
	if readErr != nil {
		t.Fatalf("file was not created: %v", readErr)
	}

	if string(content) != "fn main() {}\n" {
		t.Errorf("file content = %q, want %q", string(content), "fn main() {}\n")
	}
}

// TestWriteFileTool_CreatesNewFileInSubdirWithTracker tests creating a new file
// in a subdirectory that needs to be created, with a tracker active.
func TestWriteFileTool_CreatesNewFileInSubdirWithTracker(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	newFilePath := filepath.Join(tmpDir, "src", "main.rs")

	tool := NewWriteFileTool()
	tracker := NewFileTracker()
	tool.SetTracker(tracker)

	params, _ := FromMap(map[string]any{
		"path":    newFilePath,
		"content": "fn main() {}\n",
	})

	result, err := tool.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Errorf("expected success creating new file in subdir with tracker, got error: %s", result.Error)
	}
}

// TestWriteFileTool_OverwritesFileCreatedByShellCommand tests that write_file
// can overwrite a file that exists on disk but was never read through read_file.
// Reproduces bug: agent runs `cargo new`, which creates Cargo.toml on disk,
// then tries to overwrite it via write_file. The tracker blocks it with
// "file not previously read" because the file was never read via read_file —
// even though the agent itself created it via shell_command moments ago.
func TestWriteFileTool_OverwritesFileCreatedByShellCommand(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "Cargo.toml")

	// Simulate: shell_command created this file (e.g. `cargo new`).
	err := os.WriteFile(filePath, []byte("[package]\nname = \"test\"\n"), 0o600)
	if err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	tool := NewWriteFileTool()
	tracker := NewFileTracker()
	tool.SetTracker(tracker)

	// Agent tries to overwrite without having read it first.
	params, _ := FromMap(map[string]any{
		"path":    filePath,
		"content": "[package]\nname = \"test\"\nversion = \"0.1.0\"\n\n[dependencies]\nsdl2 = \"0.35\"\n",
	})

	result, execErr := tool.Execute(context.Background(), params)
	if execErr != nil {
		t.Fatalf("unexpected error: %v", execErr)
	}

	if !result.Success {
		t.Errorf("expected success overwriting file created by shell command, got error: %s", result.Error)
	}

	// Verify the file was actually updated.
	content, readErr := os.ReadFile(filePath)
	if readErr != nil {
		t.Fatalf("failed to read file: %v", readErr)
	}

	if !strings.Contains(string(content), "sdl2") {
		t.Errorf("file was not updated, content = %q", string(content))
	}
}

// TestFileTracker_StillBlocksStaleOverwrite verifies the tracker still protects
// against overwriting files that WERE read and then modified externally.
func TestFileTracker_StillBlocksStaleOverwrite(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "config.toml")

	// Create file, agent reads it.
	err := os.WriteFile(filePath, []byte("original"), 0o600)
	if err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	tracker := NewFileTracker()
	if recordErr := tracker.RecordRead(filePath); recordErr != nil {
		t.Fatalf("RecordRead failed: %v", recordErr)
	}

	// External modification after the read.
	time.Sleep(modTimeTolerance * 2)

	err = os.WriteFile(filePath, []byte("externally modified"), 0o600)
	if err != nil {
		t.Fatalf("failed to modify file: %v", err)
	}

	// Tracker should still block this — file was read AND modified since.
	tool := NewWriteFileTool()
	tool.SetTracker(tracker)

	params, _ := FromMap(map[string]any{
		"path":    filePath,
		"content": "agent wants to overwrite\n",
	})

	result, execErr := tool.Execute(context.Background(), params)
	if execErr != nil {
		t.Fatalf("unexpected error: %v", execErr)
	}

	if result.Success {
		t.Error("expected failure: file was read then modified externally, but write succeeded")
	}
}

func TestWriteFileTool_CheckApproval_SystemPaths(t *testing.T) {
	t.Parallel()

	tool := NewWriteFileTool()
	runWriteFileApprovalTests(t, tool, []writeFileApprovalCase{
		{"etc directory", "/etc/config.conf", RiskCritical},
		{"sys directory", "/sys/kernel/param", RiskCritical},
		{"usr directory", "/usr/bin/script", RiskCritical},
	})
}

func TestWriteFileTool_CheckApproval_RegularFiles(t *testing.T) {
	t.Parallel()

	tool := NewWriteFileTool()
	runWriteFileApprovalTests(t, tool, []writeFileApprovalCase{
		{"text file", "/tmp/notes.txt", RiskMedium},
		{"markdown", "/tmp/README.md", RiskMedium},
		{"json file", "/tmp/config.json", RiskMedium},
	})
}

func TestWriteFileTool_CheckApproval_ExecutableFiles(t *testing.T) {
	t.Parallel()

	tool := NewWriteFileTool()
	runWriteFileApprovalTests(t, tool, []writeFileApprovalCase{
		{"shell script", "/tmp/script.sh", RiskHigh},
		{"go source", "/tmp/main.go", RiskHigh},
		{"python script", "/tmp/script.py", RiskHigh},
	})
}

// Truncation Detection Tests.

func TestDetectTruncation_ValidCode_NoFalsePositive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
	}{
		{
			name:    "rust lifetime annotation",
			content: "fn foo<'a>(x: &'a str) -> &'a str { x }",
		},
		{
			name:    "rust static lifetime",
			content: "const NAME: &'static str = \"hello\";",
		},
		{
			name: "rust complex lifetimes",
			content: `struct Parser<'input> {
    source: &'input str,
    pos: usize,
}

impl<'input> Parser<'input> {
    fn new(source: &'input str) -> Self {
        Parser { source, pos: 0 }
    }
}`,
		},
		{
			name:    "rust char literal",
			content: "let c: char = 'x';",
		},
		{
			name:    "go rune literal",
			content: "var r rune = '\\n'",
		},
		{
			name: "complete rust function",
			content: `fn main() -> Result<(), Box<dyn std::error::Error>> {
    println!("hello");
    Ok(())
}`,
		},
		{
			name:    "python single-quoted string",
			content: "x = 'hello world'",
		},
		{
			name:    "empty content",
			content: "",
		},
		{
			name:    "balanced braces",
			content: "func main() { if true { x() } }",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := stringsx.DetectTruncation(tt.content)
			if result != "" {
				t.Errorf("detectTruncation false positive for %q: got %q, want empty", tt.name, result)
			}
		})
	}
}

func TestDetectTruncation_TruncatedContent_Detected(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
		wantMsg string
	}{
		{
			name:    "unclosed brace",
			content: "func main() {",
			wantMsg: "unclosed delimiter",
		},
		{
			name:    "unclosed double quote",
			content: `x = "hello`,
			wantMsg: "unclosed string literal",
		},
		{
			name:    "unclosed bracket",
			content: "items = [1, 2, 3",
			wantMsg: "unclosed delimiter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := stringsx.DetectTruncation(tt.content)
			if result == "" {
				t.Errorf("detectTruncation missed truncation for %q", tt.name)
			} else if !strings.Contains(result, tt.wantMsg) {
				t.Errorf("stringsx.DetectTruncation(%q) = %q, want containing %q", tt.name, result, tt.wantMsg)
			}
		})
	}
}
