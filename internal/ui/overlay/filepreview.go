package overlay

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FilePreview represents a readonly file preview overlay
type FilePreview struct {
	// File path being previewed
	FilePath string
	// File content (lines)
	Lines []string
	// Target line to jump to (1-indexed)
	TargetLine int
	// Current scroll position (0-indexed line number)
	ScrollPos int
	// Viewport dimensions
	Width  int
	Height int
	// Search state
	SearchQuery string
	SearchPos   int // current match index
}

// NewFilePreview creates a new file preview from a file path and optional target line
func NewFilePreview(filePath string, targetLine int, width, height int) (*FilePreview, error) {
	// Read file content
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Calculate initial scroll position to center target line
	scrollPos := 0
	if targetLine > 0 && targetLine <= len(lines) {
		// Center the target line in the viewport
		scrollPos = targetLine - 1 - (height / 2)
		if scrollPos < 0 {
			scrollPos = 0
		}
	}

	return &FilePreview{
		FilePath:   filePath,
		Lines:      lines,
		TargetLine: targetLine,
		ScrollPos:  scrollPos,
		Width:      width,
		Height:     height,
	}, nil
}

// ScrollUp scrolls up by n lines
func (fp *FilePreview) ScrollUp(n int) {
	fp.ScrollPos -= n
	if fp.ScrollPos < 0 {
		fp.ScrollPos = 0
	}
}

// ScrollDown scrolls down by n lines
func (fp *FilePreview) ScrollDown(n int) {
	maxScroll := len(fp.Lines) - fp.Height
	if maxScroll < 0 {
		maxScroll = 0
	}
	fp.ScrollPos += n
	if fp.ScrollPos > maxScroll {
		fp.ScrollPos = maxScroll
	}
}

// ScrollToTop scrolls to the top of the file
func (fp *FilePreview) ScrollToTop() {
	fp.ScrollPos = 0
}

// ScrollToBottom scrolls to the bottom of the file
func (fp *FilePreview) ScrollToBottom() {
	maxScroll := len(fp.Lines) - fp.Height
	if maxScroll < 0 {
		maxScroll = 0
	}
	fp.ScrollPos = maxScroll
}

// GetVisibleLines returns the lines visible in the current viewport
func (fp *FilePreview) GetVisibleLines() []string {
	if fp.ScrollPos >= len(fp.Lines) {
		return []string{}
	}
	end := fp.ScrollPos + fp.Height
	if end > len(fp.Lines) {
		end = len(fp.Lines)
	}
	return fp.Lines[fp.ScrollPos:end]
}

// IsTargetLineVisible returns true if the target line is in the current viewport
func (fp *FilePreview) IsTargetLineVisible() bool {
	if fp.TargetLine <= 0 {
		return false
	}
	lineIdx := fp.TargetLine - 1
	return lineIdx >= fp.ScrollPos && lineIdx < fp.ScrollPos+fp.Height
}

// Search finds all occurrences of a query string in the file
func (fp *FilePreview) Search(query string) []int {
	if query == "" {
		return nil
	}
	fp.SearchQuery = query
	fp.SearchPos = 0

	var matches []int
	for i, line := range fp.Lines {
		if strings.Contains(strings.ToLower(line), strings.ToLower(query)) {
			matches = append(matches, i)
		}
	}
	return matches
}

// NextMatch jumps to the next search match
func (fp *FilePreview) NextMatch(matches []int) {
	if len(matches) == 0 {
		return
	}
	fp.SearchPos = (fp.SearchPos + 1) % len(matches)
	// Scroll to center the match
	targetLine := matches[fp.SearchPos]
	fp.ScrollPos = targetLine - (fp.Height / 2)
	if fp.ScrollPos < 0 {
		fp.ScrollPos = 0
	}
	maxScroll := len(fp.Lines) - fp.Height
	if fp.ScrollPos > maxScroll {
		fp.ScrollPos = maxScroll
	}
}

// PrevMatch jumps to the previous search match
func (fp *FilePreview) PrevMatch(matches []int) {
	if len(matches) == 0 {
		return
	}
	fp.SearchPos = (fp.SearchPos - 1 + len(matches)) % len(matches)
	// Scroll to center the match
	targetLine := matches[fp.SearchPos]
	fp.ScrollPos = targetLine - (fp.Height / 2)
	if fp.ScrollPos < 0 {
		fp.ScrollPos = 0
	}
	maxScroll := len(fp.Lines) - fp.Height
	if fp.ScrollPos > maxScroll {
		fp.ScrollPos = maxScroll
	}
}

// CalculatePopupDimensions calculates optimal popup size based on terminal dimensions
// Spec: Width: min(100ch, W - 2*s4); Height: min(30 rows, H - 6)
func CalculatePopupDimensions(termWidth, termHeight int) (width, height int) {
	const s4 = 4 // spacing constant from design tokens

	width = termWidth - 2*s4
	if width > 100 {
		width = 100
	}
	if width < 40 { // minimum usable width
		width = 40
	}

	height = termHeight - 6
	if height > 30 {
		height = 30
	}
	if height < 10 { // minimum usable height
		height = 10
	}

	return width, height
}

// Anchor represents a filename:line anchor detected in text
type Anchor struct {
	FilePath string
	Line     int
	Start    int // start position in text
	End      int // end position in text
}

// DetectAnchors finds all filename:line anchors in a string
// Matches patterns like: internal/ui/term/tty.go:42 or src/main.go:123
func DetectAnchors(text string) []Anchor {
	var anchors []Anchor

	// Simple pattern: word characters, slashes, dots, then :digits
	// Example: internal/ui/term/tty.go:42
	i := 0
	for i < len(text) {
		// Find potential start of file path (alphanumeric, slash, dot, underscore, hyphen)
		if !isFilePathChar(text[i]) {
			i++
			continue
		}

		start := i
		// Scan forward to collect file path
		for i < len(text) && isFilePathChar(text[i]) {
			i++
		}

		// Check if followed by :digits
		if i < len(text) && text[i] == ':' {
			i++ // skip ':'
			lineStart := i
			// Collect digits
			for i < len(text) && text[i] >= '0' && text[i] <= '9' {
				i++
			}

			if i > lineStart { // found at least one digit
				filePath := text[start : lineStart-1] // -1 to exclude ':'
				lineNum := 0
				fmt.Sscanf(text[lineStart:i], "%d", &lineNum)

				// Only consider paths with extension (must contain '.')
				if strings.Contains(filePath, ".") {
					anchors = append(anchors, Anchor{
						FilePath: filePath,
						Line:     lineNum,
						Start:    start,
						End:      i,
					})
				}
			}
		}
	}

	return anchors
}

func isFilePathChar(c byte) bool {
	return (c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') ||
		c == '/' || c == '.' || c == '_' || c == '-'
}

// FindAnchorAtPosition finds an anchor at a specific position in text
func FindAnchorAtPosition(text string, pos int) *Anchor {
	anchors := DetectAnchors(text)
	for _, anchor := range anchors {
		if pos >= anchor.Start && pos < anchor.End {
			return &anchor
		}
	}
	return nil
}

// GetAbsolutePath resolves a file path relative to a base directory
func GetAbsolutePath(basePath, filePath string) string {
	if filepath.IsAbs(filePath) {
		return filePath
	}
	// Try relative to base path
	absPath := filepath.Join(basePath, filePath)
	if _, err := os.Stat(absPath); err == nil {
		return absPath
	}
	// Try current working directory
	if cwd, err := os.Getwd(); err == nil {
		absPath = filepath.Join(cwd, filePath)
		if _, err := os.Stat(absPath); err == nil {
			return absPath
		}
	}
	// Return as-is if nothing works
	return filePath
}
