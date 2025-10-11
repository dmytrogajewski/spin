package overlay

import (
	"fmt"
	"strings"
	"testing"
)

func TestFilePreviewRenderer_Render(t *testing.T) {
	lines := []string{
		"package main",
		"",
		"import \"fmt\"",
		"",
		"func main() {",
		"	fmt.Println(\"Hello\")",
		"}",
	}

	fp := &FilePreview{
		FilePath:   "main.go",
		Lines:      lines,
		TargetLine: 5,
		ScrollPos:  0,
		Width:      60,
		Height:     10,
	}

	renderer := NewFilePreviewRenderer(60)
	output := renderer.Render(fp)

	// Check that output contains key elements
	if !strings.Contains(output, "main.go") {
		t.Error("Render() output should contain filename")
	}
	if !strings.Contains(output, "[Esc to close]") {
		t.Error("Render() output should contain close hint")
	}
	if !strings.Contains(output, "package main") {
		t.Error("Render() output should contain file content")
	}

	// Count number of lines
	lines_out := strings.Split(output, "\n")
	// Should have: header + border + content lines + bottom border
	// With height=10: 1 header + 1 border + 7 content + 1 border = 10 lines
	if len(lines_out) < 8 {
		t.Errorf("Render() output has %d lines, expected at least 8", len(lines_out))
	}
}

func TestFilePreviewRenderer_RenderHeader(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		width    int
		checks   []string
	}{
		{
			name:     "short filename",
			filePath: "main.go",
			width:    80,
			checks:   []string{"main.go", "[Esc to close]", "┌─", "─┐"},
		},
		{
			name:     "long filename",
			filePath: "/very/long/path/to/some/deeply/nested/directory/structure/file.go",
			width:    80,
			checks:   []string{"...", "file.go", "[Esc to close]"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fp := &FilePreview{
				FilePath: tt.filePath,
				Width:    tt.width,
			}
			r := NewFilePreviewRenderer(tt.width)
			header := r.renderHeader(fp)

			for _, check := range tt.checks {
				if !strings.Contains(header, check) {
					t.Errorf("renderHeader() should contain %q", check)
				}
			}
		})
	}
}

func TestFilePreviewRenderer_RenderBorder(t *testing.T) {
	renderer := NewFilePreviewRenderer(40)
	border := renderer.renderBorder()

	// Border contains ANSI codes, so check for presence of border chars
	if !strings.Contains(border, "│") {
		t.Error("renderBorder() should contain │")
	}
	// Should have two border chars (start and end)
	count := strings.Count(border, "│")
	if count != 2 {
		t.Errorf("renderBorder() should contain 2 × │, got %d", count)
	}
}

func TestFilePreviewRenderer_RenderBottomBorder(t *testing.T) {
	renderer := NewFilePreviewRenderer(60)

	tests := []struct {
		name       string
		totalLines int
		scrollPos  int
		height     int
		wantInfo   bool // should show scroll info
	}{
		{
			name:       "short file, no scroll info",
			totalLines: 5,
			scrollPos:  0,
			height:     10,
			wantInfo:   false,
		},
		{
			name:       "medium file, no scroll info",
			totalLines: 7, // exactly fits in height-3
			scrollPos:  0,
			height:     10,
			wantInfo:   false,
		},
		{
			name:       "file exceeds viewport, show scroll info",
			totalLines: 10, // exceeds height-3
			scrollPos:  0,
			height:     10,
			wantInfo:   true,
		},
		{
			name:       "long file, show scroll info",
			totalLines: 100,
			scrollPos:  10,
			height:     20,
			wantInfo:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fp := &FilePreview{
				Lines:     make([]string, tt.totalLines),
				ScrollPos: tt.scrollPos,
				Height:    tt.height,
			}
			border := renderer.renderBottomBorder(fp)

			// Check for scroll indicator by looking for the actual pattern (not ANSI escape codes)
			// Pattern is " [start-end/total] "
			hasInfo := strings.Contains(border, "/") && strings.Count(border, "[") > 2 // ANSI codes have [, so check for extra brackets
			if hasInfo != tt.wantInfo {
				t.Errorf("renderBottomBorder() hasInfo=%v, want %v", hasInfo, tt.wantInfo)
			}

			// Border contains ANSI codes, so check for presence
			if !strings.Contains(border, "└") {
				t.Error("renderBottomBorder() should contain └")
			}
			if !strings.Contains(border, "┘") {
				t.Error("renderBottomBorder() should contain ┘")
			}
		})
	}
}

func TestFilePreviewRenderer_RenderContent(t *testing.T) {
	lines := []string{
		"line 1",
		"line 2",
		"line 3",
		"line 4",
		"line 5",
	}

	tests := []struct {
		name         string
		scrollPos    int
		targetLine   int
		contentLines []string
	}{
		{
			name:         "scroll at top",
			scrollPos:    0,
			targetLine:   0,
			contentLines: []string{"line 1", "line 2", "line 3"},
		},
		{
			name:         "scroll in middle",
			scrollPos:    1,
			targetLine:   0,
			contentLines: []string{"line 2", "line 3", "line 4"},
		},
		{
			name:         "with target line",
			scrollPos:    0,
			targetLine:   2,
			contentLines: []string{"line 1", "line 2", "line 3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fp := &FilePreview{
				Lines:      lines,
				ScrollPos:  tt.scrollPos,
				TargetLine: tt.targetLine,
				Height:     6, // 6 total height = 3 content lines after header/borders
			}

			renderer := NewFilePreviewRenderer(60)
			content := renderer.renderContent(fp, 3)

			for _, line := range tt.contentLines {
				if !strings.Contains(content, line) {
					t.Errorf("renderContent() should contain %q", line)
				}
			}

			// Should contain line number gutter
			if !strings.Contains(content, "│") {
				t.Error("renderContent() should contain gutter separator │")
			}
		})
	}
}

func TestFilePreviewRenderer_RenderContent_TargetLineHighlight(t *testing.T) {
	lines := []string{
		"line 1",
		"line 2 - TARGET",
		"line 3",
	}

	fp := &FilePreview{
		Lines:      lines,
		ScrollPos:  0,
		TargetLine: 2, // line 2 is the target
		Height:     6,
	}

	renderer := NewFilePreviewRenderer(60)
	content := renderer.renderContent(fp, 3)

	// Target line should be present
	if !strings.Contains(content, "line 2 - TARGET") {
		t.Error("renderContent() should contain target line")
	}

	// Check that yellow color code is used (simplified check)
	// The actual ANSI code for yellow is in the output
	lines_out := strings.Split(content, "\n")
	foundYellow := false
	for _, line := range lines_out {
		if strings.Contains(line, "line 2 - TARGET") && strings.Contains(line, "\x1b[38;5;220m") {
			foundYellow = true
			break
		}
	}
	if !foundYellow {
		t.Error("renderContent() should highlight target line in yellow")
	}
}

func TestFilePreviewRenderer_RenderContent_LineTruncation(t *testing.T) {
	longLine := strings.Repeat("x", 200)
	lines := []string{longLine}

	fp := &FilePreview{
		Lines:     lines,
		ScrollPos: 0,
		Height:    5,
	}

	renderer := NewFilePreviewRenderer(60)
	content := renderer.renderContent(fp, 2)

	// Should contain truncation marker
	if !strings.Contains(content, "…") {
		t.Error("renderContent() should truncate long lines with …")
	}
}

func TestFilePreviewRenderer_RenderContent_EmptyLines(t *testing.T) {
	lines := []string{"line 1", "line 2"}

	fp := &FilePreview{
		Lines:     lines,
		ScrollPos: 0,
		Height:    10, // request more lines than available
	}

	renderer := NewFilePreviewRenderer(60)
	content := renderer.renderContent(fp, 5) // 5 content lines requested

	// Should have 5 lines (2 with content, 3 empty)
	lines_out := strings.Split(strings.TrimRight(content, "\n"), "\n")
	if len(lines_out) != 5 {
		t.Errorf("renderContent() = %d lines, want 5", len(lines_out))
	}
}

func TestFilePreviewRenderer_RenderContent_DynamicGutter(t *testing.T) {
	// Test that gutter width adapts to line number size
	tests := []struct {
		name         string
		totalLines   int
		minGutterLen int
	}{
		{"small file (1-9)", 9, 3},    // at least 3 chars
		{"medium file (10-99)", 99, 3}, // 2 digits + spacing
		{"large file (100-999)", 999, 4}, // 3 digits + spacing
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines := make([]string, tt.totalLines)
			for i := range lines {
				lines[i] = "code"
			}

			fp := &FilePreview{
				Lines:     lines,
				ScrollPos: tt.totalLines - 3, // scroll near end to see large line numbers
				Height:    6,
			}

			renderer := NewFilePreviewRenderer(80)
			content := renderer.renderContent(fp, 3)

			// Check that line numbers are present and properly formatted
			// Line numbers should be right-aligned in gutter
			if !strings.Contains(content, fmt.Sprintf("%d", tt.totalLines)) {
				t.Errorf("renderContent() should contain line number %d", tt.totalLines)
			}
		})
	}
}

// Test ANSI color helpers
func TestANSIHelpers(t *testing.T) {
	tests := []struct {
		name   string
		fn     func(string) string
		input  string
		checks []string // ANSI codes that should be present
	}{
		{"dim", dim, "text", []string{"\x1b[2m", "\x1b[0m"}},
		{"muted", muted, "text", []string{"\x1b[38;5;242m", "\x1b[0m"}},
		{"yellow", yellow, "text", []string{"\x1b[38;5;220m", "\x1b[0m"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := tt.fn(tt.input)
			if !strings.Contains(output, tt.input) {
				t.Errorf("%s() should contain input text", tt.name)
			}
			for _, check := range tt.checks {
				if !strings.Contains(output, check) {
					t.Errorf("%s() should contain ANSI code %q", tt.name, check)
				}
			}
		})
	}
}

// Benchmark rendering
func BenchmarkFilePreviewRenderer_Render(b *testing.B) {
	lines := make([]string, 1000)
	for i := range lines {
		lines[i] = fmt.Sprintf("line %d with some content here", i+1)
	}

	fp := &FilePreview{
		FilePath:   "large_file.go",
		Lines:      lines,
		TargetLine: 500,
		ScrollPos:  450,
		Width:      80,
		Height:     30,
	}

	renderer := NewFilePreviewRenderer(80)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = renderer.Render(fp)
	}
}
