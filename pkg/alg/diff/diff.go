// Package diff provides unified diff generation and parsing.
package diff

import (
	"errors"
	"fmt"
	"strings"
)

// Sentinel errors for diff parsing.
var (
	// ErrDiffTooShort is returned when the diff text has fewer lines than required.
	ErrDiffTooShort = errors.New("diff too short")
	// ErrBadHeader is returned when the first line doesn't match a diff header format.
	ErrBadHeader = errors.New("cannot extract filename from header")
)

// LineType classifies a diff line.
type LineType int

const (
	// LineContext is an unchanged context line.
	LineContext LineType = iota
	// LineInsert is an added line.
	LineInsert
	// LineDelete is a removed line.
	LineDelete
)

// LineChange represents a single line in a diff hunk.
type LineChange struct {
	Type LineType
	Text string
}

// Hunk represents a contiguous block of changes.
type Hunk struct {
	Changes []LineChange
}

// minParseLines is the minimum number of lines for a valid diff (header + content).
const minParseLines = 3

// Generate produces a simple unified diff showing all old lines as deletions
// and all new lines as insertions.
func Generate(filePath, oldText, newText string) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "--- %s\n+++ %s\n@@ -0,0 +0,0 @@\n", filePath, filePath)

	for line := range strings.SplitSeq(oldText, "\n") {
		sb.WriteString("-")
		sb.WriteString(line)
		sb.WriteString("\n")
	}

	for line := range strings.SplitSeq(newText, "\n") {
		sb.WriteString("+")
		sb.WriteString(line)
		sb.WriteString("\n")
	}

	return sb.String()
}

// Parse parses a unified diff into a filename and list of hunks.
// Supports --- and *** file headers, @@ hunk headers, and +/-/space line prefixes.
func Parse(diffText string) (filename string, hunks []Hunk, err error) {
	lines := strings.Split(diffText, "\n")
	if len(lines) < minParseLines {
		return "", nil, fmt.Errorf("%w: need at least %d lines, got %d", ErrDiffTooShort, minParseLines, len(lines))
	}

	filename, err = extractFilename(lines[0])
	if err != nil {
		return "", nil, err
	}

	hunks = parseHunks(lines)

	return filename, hunks, nil
}

// extractFilename extracts the filename from a diff header line.
func extractFilename(line string) (string, error) {
	trimmed := strings.TrimSpace(line)

	if after, ok := strings.CutPrefix(trimmed, "*** "); ok {
		return stripDiffPrefix(strings.TrimSpace(after)), nil
	}

	if after, ok := strings.CutPrefix(trimmed, "--- "); ok {
		return stripDiffPrefix(strings.TrimSpace(after)), nil
	}

	return "", fmt.Errorf("%w: %q", ErrBadHeader, trimmed)
}

// stripDiffPrefix removes standard a/ or b/ prefix from git diff filenames.
func stripDiffPrefix(name string) string {
	if after, ok := strings.CutPrefix(name, "a/"); ok {
		return after
	}

	if after, ok := strings.CutPrefix(name, "b/"); ok {
		return after
	}

	return name
}

// parseHunks parses hunk sections from diff lines (starting after the header).
func parseHunks(lines []string) []Hunk {
	var hunks []Hunk

	var current *Hunk

	// Start from line 2 (skip --- and +++ headers).
	for i := 2; i < len(lines); i++ {
		line := lines[i]

		if strings.HasPrefix(line, "@@") {
			if current != nil {
				hunks = append(hunks, *current)
			}

			current = &Hunk{Changes: []LineChange{}}

			continue
		}

		if current == nil {
			continue
		}

		if change, ok := parseLine(line); ok {
			current.Changes = append(current.Changes, change)
		}
	}

	if current != nil {
		hunks = append(hunks, *current)
	}

	return hunks
}

// parseLine parses a single diff line into a LineChange.
func parseLine(line string) (LineChange, bool) {
	if line == "" {
		return LineChange{Type: LineContext, Text: ""}, true
	}

	text := ""
	if len(line) > 1 {
		text = line[1:]
	}

	switch line[0] {
	case ' ':
		return LineChange{Type: LineContext, Text: text}, true
	case '-':
		return LineChange{Type: LineDelete, Text: text}, true
	case '+':
		return LineChange{Type: LineInsert, Text: text}, true
	default:
		return LineChange{}, false
	}
}
