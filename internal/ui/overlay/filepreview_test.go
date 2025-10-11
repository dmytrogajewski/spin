package overlay

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewFilePreview(t *testing.T) {
	// Create a temporary file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.go")
	content := `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
`
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	tests := []struct {
		name       string
		targetLine int
		wantScroll int
	}{
		{"no target line", 0, 0},
		{"target line 1", 1, 0},
		{"target line 4", 4, 0}, // too close to start to center
		{"target line 100", 100, 0}, // beyond file length
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fp, err := NewFilePreview(tmpFile, tt.targetLine, 80, 20)
			if err != nil {
				t.Fatalf("NewFilePreview() error = %v", err)
			}

			if fp.FilePath != tmpFile {
				t.Errorf("FilePath = %v, want %v", fp.FilePath, tmpFile)
			}
			if len(fp.Lines) != 7 { // 7 lines in content
				t.Errorf("Lines count = %v, want 7", len(fp.Lines))
			}
			if fp.TargetLine != tt.targetLine {
				t.Errorf("TargetLine = %v, want %v", fp.TargetLine, tt.targetLine)
			}
			if fp.ScrollPos < 0 {
				t.Errorf("ScrollPos = %v, should not be negative", fp.ScrollPos)
			}
		})
	}
}

func TestNewFilePreview_NonExistentFile(t *testing.T) {
	_, err := NewFilePreview("/nonexistent/file.go", 0, 80, 20)
	if err == nil {
		t.Error("NewFilePreview() expected error for nonexistent file, got nil")
	}
}

func TestFilePreview_ScrollUp(t *testing.T) {
	fp := &FilePreview{
		Lines:     make([]string, 100),
		ScrollPos: 10,
		Height:    20,
	}

	fp.ScrollUp(5)
	if fp.ScrollPos != 5 {
		t.Errorf("ScrollPos = %v, want 5", fp.ScrollPos)
	}

	// Can't scroll below 0
	fp.ScrollUp(10)
	if fp.ScrollPos != 0 {
		t.Errorf("ScrollPos = %v, want 0 (clamped)", fp.ScrollPos)
	}
}

func TestFilePreview_ScrollDown(t *testing.T) {
	fp := &FilePreview{
		Lines:     make([]string, 100),
		ScrollPos: 10,
		Height:    20,
	}

	fp.ScrollDown(5)
	if fp.ScrollPos != 15 {
		t.Errorf("ScrollPos = %v, want 15", fp.ScrollPos)
	}

	// Can't scroll past end
	fp.ScrollDown(100)
	maxScroll := 100 - 20 // len(lines) - height
	if fp.ScrollPos != maxScroll {
		t.Errorf("ScrollPos = %v, want %v (clamped)", fp.ScrollPos, maxScroll)
	}
}

func TestFilePreview_ScrollToTop(t *testing.T) {
	fp := &FilePreview{
		Lines:     make([]string, 100),
		ScrollPos: 50,
		Height:    20,
	}

	fp.ScrollToTop()
	if fp.ScrollPos != 0 {
		t.Errorf("ScrollPos = %v, want 0", fp.ScrollPos)
	}
}

func TestFilePreview_ScrollToBottom(t *testing.T) {
	fp := &FilePreview{
		Lines:     make([]string, 100),
		ScrollPos: 0,
		Height:    20,
	}

	fp.ScrollToBottom()
	want := 100 - 20
	if fp.ScrollPos != want {
		t.Errorf("ScrollPos = %v, want %v", fp.ScrollPos, want)
	}
}

func TestFilePreview_GetVisibleLines(t *testing.T) {
	lines := []string{"line1", "line2", "line3", "line4", "line5"}
	fp := &FilePreview{
		Lines:     lines,
		ScrollPos: 1,
		Height:    3,
	}

	visible := fp.GetVisibleLines()
	want := []string{"line2", "line3", "line4"}
	if len(visible) != len(want) {
		t.Fatalf("GetVisibleLines() length = %v, want %v", len(visible), len(want))
	}
	for i := range visible {
		if visible[i] != want[i] {
			t.Errorf("GetVisibleLines()[%d] = %v, want %v", i, visible[i], want[i])
		}
	}
}

func TestFilePreview_GetVisibleLines_AtEnd(t *testing.T) {
	lines := []string{"line1", "line2", "line3"}
	fp := &FilePreview{
		Lines:     lines,
		ScrollPos: 1,
		Height:    10, // viewport larger than remaining lines
	}

	visible := fp.GetVisibleLines()
	want := []string{"line2", "line3"}
	if len(visible) != len(want) {
		t.Fatalf("GetVisibleLines() length = %v, want %v", len(visible), len(want))
	}
}

func TestFilePreview_IsTargetLineVisible(t *testing.T) {
	tests := []struct {
		name       string
		targetLine int
		scrollPos  int
		height     int
		want       bool
	}{
		{"target visible in middle", 5, 0, 10, true},
		{"target at top", 1, 0, 10, true},
		{"target above viewport", 1, 5, 10, false},
		{"target below viewport", 20, 0, 10, false},
		{"no target line", 0, 0, 10, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fp := &FilePreview{
				Lines:      make([]string, 50),
				TargetLine: tt.targetLine,
				ScrollPos:  tt.scrollPos,
				Height:     tt.height,
			}
			if got := fp.IsTargetLineVisible(); got != tt.want {
				t.Errorf("IsTargetLineVisible() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilePreview_Search(t *testing.T) {
	lines := []string{
		"package main",
		"import fmt",
		"func main() {",
		"  fmt.Println(\"hello\")",
		"}",
	}
	fp := &FilePreview{
		Lines:  lines,
		Height: 10,
	}

	matches := fp.Search("fmt")
	if len(matches) != 2 {
		t.Errorf("Search() found %v matches, want 2", len(matches))
	}
	if matches[0] != 1 || matches[1] != 3 {
		t.Errorf("Search() matches = %v, want [1 3]", matches)
	}
}

func TestFilePreview_Search_CaseInsensitive(t *testing.T) {
	lines := []string{"Hello World", "hello world"}
	fp := &FilePreview{Lines: lines, Height: 10}

	matches := fp.Search("HELLO")
	if len(matches) != 2 {
		t.Errorf("Search() found %v matches, want 2 (case insensitive)", len(matches))
	}
}

func TestFilePreview_NextMatch(t *testing.T) {
	lines := make([]string, 100)
	for i := range lines {
		if i%10 == 0 {
			lines[i] = "match"
		} else {
			lines[i] = "other"
		}
	}

	fp := &FilePreview{
		Lines:  lines,
		Height: 10,
	}

	matches := fp.Search("match")
	if len(matches) != 10 {
		t.Fatalf("Search() found %v matches, want 10", len(matches))
	}

	// First call should go to match 1 (wraps around since SearchPos starts at 0)
	fp.NextMatch(matches)
	if fp.SearchPos != 1 {
		t.Errorf("SearchPos = %v, want 1", fp.SearchPos)
	}

	// Last call should wrap to 0
	fp.SearchPos = 9
	fp.NextMatch(matches)
	if fp.SearchPos != 0 {
		t.Errorf("SearchPos = %v, want 0 (wrapped)", fp.SearchPos)
	}
}

func TestFilePreview_PrevMatch(t *testing.T) {
	lines := make([]string, 100)
	for i := range lines {
		if i%10 == 0 {
			lines[i] = "match"
		} else {
			lines[i] = "other"
		}
	}

	fp := &FilePreview{
		Lines:     lines,
		Height:    10,
		SearchPos: 0, // Start at first match
	}

	matches := fp.Search("match")
	if len(matches) != 10 {
		t.Fatalf("Search() found %v matches, want 10", len(matches))
	}

	// From pos 0, should wrap to end (pos 9)
	fp.PrevMatch(matches)
	if fp.SearchPos != 9 {
		t.Errorf("SearchPos = %v, want 9 (wrapped from 0)", fp.SearchPos)
	}

	// From pos 9, should go to 8
	fp.PrevMatch(matches)
	if fp.SearchPos != 8 {
		t.Errorf("SearchPos = %v, want 8", fp.SearchPos)
	}
}

func TestCalculatePopupDimensions(t *testing.T) {
	tests := []struct {
		name       string
		termWidth  int
		termHeight int
		wantWidth  int
		wantHeight int
	}{
		{"large terminal", 200, 60, 100, 30},  // max width/height
		{"medium terminal", 80, 24, 72, 18},   // W-8, H-6
		{"small terminal", 50, 20, 42, 14},    // W-8, H-6
		{"tiny terminal", 30, 10, 40, 10},     // min width, clamped height
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotWidth, gotHeight := CalculatePopupDimensions(tt.termWidth, tt.termHeight)
			if gotWidth != tt.wantWidth {
				t.Errorf("width = %v, want %v", gotWidth, tt.wantWidth)
			}
			if gotHeight != tt.wantHeight {
				t.Errorf("height = %v, want %v", gotHeight, tt.wantHeight)
			}
		})
	}
}

func TestDetectAnchors(t *testing.T) {
	tests := []struct {
		name  string
		text  string
		want  int // number of anchors
		check func(*testing.T, []Anchor)
	}{
		{
			name: "simple anchor",
			text: "See internal/ui/term/tty.go:42 for details",
			want: 1,
			check: func(t *testing.T, anchors []Anchor) {
				if anchors[0].FilePath != "internal/ui/term/tty.go" {
					t.Errorf("FilePath = %v, want internal/ui/term/tty.go", anchors[0].FilePath)
				}
				if anchors[0].Line != 42 {
					t.Errorf("Line = %v, want 42", anchors[0].Line)
				}
			},
		},
		{
			name: "multiple anchors",
			text: "Check main.go:10 and utils.go:25",
			want: 2,
		},
		{
			name: "no anchors",
			text: "No file references here",
			want: 0,
		},
		{
			name: "path without extension",
			text: "Check somedir/file:42 (no extension, should be ignored)",
			want: 0,
		},
		{
			name: "anchor at start",
			text: "main.go:1 is the first line",
			want: 1,
		},
		{
			name: "anchor at end",
			text: "End of file: main.go:100",
			want: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			anchors := DetectAnchors(tt.text)
			if len(anchors) != tt.want {
				t.Errorf("DetectAnchors() found %v anchors, want %v", len(anchors), tt.want)
			}
			if tt.check != nil && len(anchors) > 0 {
				tt.check(t, anchors)
			}
		})
	}
}

func TestFindAnchorAtPosition(t *testing.T) {
	text := "See main.go:42 for details"
	// Positions:  0123456789...

	tests := []struct {
		name string
		pos  int
		want bool // whether anchor should be found
	}{
		{"before anchor", 2, false},
		{"at anchor start", 4, true},
		{"in anchor", 10, true},
		{"at anchor end", 13, true},
		{"after anchor", 15, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			anchor := FindAnchorAtPosition(text, tt.pos)
			found := anchor != nil
			if found != tt.want {
				t.Errorf("FindAnchorAtPosition(%d) found=%v, want=%v", tt.pos, found, tt.want)
			}
			if found && anchor.FilePath != "main.go" {
				t.Errorf("FilePath = %v, want main.go", anchor.FilePath)
			}
		})
	}
}

func TestGetAbsolutePath(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name     string
		basePath string
		filePath string
		wantAbs  bool // should return absolute path
	}{
		{"absolute path", "/tmp", "/etc/hosts", true},
		{"relative to base", tmpDir, "test.go", true},
		{"nonexistent relative", tmpDir, "nonexistent.go", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetAbsolutePath(tt.basePath, tt.filePath)
			isAbs := filepath.IsAbs(result)
			if tt.wantAbs && !isAbs {
				t.Errorf("GetAbsolutePath() = %v, want absolute path", result)
			}
			// If the file should exist relative to base, verify it exists
			if tt.name == "relative to base" {
				if _, err := os.Stat(result); err != nil {
					t.Errorf("GetAbsolutePath() = %v, file should exist", result)
				}
			}
		})
	}
}

func TestIsFilePathChar(t *testing.T) {
	tests := []struct {
		char byte
		want bool
	}{
		{'a', true},
		{'Z', true},
		{'0', true},
		{'/', true},
		{'.', true},
		{'_', true},
		{'-', true},
		{' ', false},
		{':', false},
		{'@', false},
	}

	for _, tt := range tests {
		t.Run(string(tt.char), func(t *testing.T) {
			if got := isFilePathChar(tt.char); got != tt.want {
				t.Errorf("isFilePathChar(%c) = %v, want %v", tt.char, got, tt.want)
			}
		})
	}
}

func TestDetectAnchors_ComplexPaths(t *testing.T) {
	tests := []struct {
		name string
		text string
		want []Anchor
	}{
		{
			name: "underscore in filename",
			text: "Check test_file.go:10",
			want: []Anchor{{FilePath: "test_file.go", Line: 10}},
		},
		{
			name: "hyphen in filename",
			text: "See my-module.go:5",
			want: []Anchor{{FilePath: "my-module.go", Line: 5}},
		},
		{
			name: "nested path",
			text: "Error in src/internal/core/agent.go:123",
			want: []Anchor{{FilePath: "src/internal/core/agent.go", Line: 123}},
		},
		{
			name: "multiple dots",
			text: "Check file.test.go:1",
			want: []Anchor{{FilePath: "file.test.go", Line: 1}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectAnchors(tt.text)
			if len(got) != len(tt.want) {
				t.Fatalf("DetectAnchors() = %v anchors, want %v", len(got), len(tt.want))
			}
			for i := range got {
				if got[i].FilePath != tt.want[i].FilePath {
					t.Errorf("anchor[%d].FilePath = %v, want %v", i, got[i].FilePath, tt.want[i].FilePath)
				}
				if got[i].Line != tt.want[i].Line {
					t.Errorf("anchor[%d].Line = %v, want %v", i, got[i].Line, tt.want[i].Line)
				}
			}
		})
	}
}

// Benchmark anchor detection
func BenchmarkDetectAnchors(b *testing.B) {
	text := strings.Repeat("Check internal/ui/term/tty.go:42 and src/main.go:100 for details. ", 10)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DetectAnchors(text)
	}
}
