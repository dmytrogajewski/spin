package patchapply

import (
	"bufio"
	"fmt"
	"strings"
)

// Parser parses patch text into structured Patch AST.
// It uses a streaming approach with bufio.Scanner for memory efficiency.
type Parser struct {
	scanner  *bufio.Scanner
	lineNum  int
	peeked   *string // Peeked line for lookahead
	peekLine int     // Line number of peeked line
}

// NewParser creates a new patch parser for the given text.
//
// The parser uses a streaming approach with bufio.Scanner for memory efficiency.
// It can handle large patches (>10k lines) without loading everything into memory.
func NewParser(text string) *Parser {
	return &Parser{
		scanner:  bufio.NewScanner(strings.NewReader(text)),
		lineNum:  0,
		peeked:   nil,
		peekLine: 0,
	}
}

// Parse parses the complete patch and returns a structured Patch AST.
//
// Returns an error if:
//   - Begin/End markers are missing or malformed
//   - Any file paths are invalid (absolute, traversal, etc.)
//   - Syntax is incorrect (invalid prefixes, unknown operations)
//
// Errors include line numbers for precise debugging.
func (p *Parser) Parse() (*Patch, error) {
	if !p.expectLine("*** Begin Patch") {
		return nil, fmt.Errorf("line %d: expected '*** Begin Patch'", p.lineNum)
	}

	var ops []FileOperation
	for {
		line, ok := p.nextLine()
		if !ok {
			break
		}

		line = strings.TrimSpace(line)

		if line == "*** End Patch" {
			return &Patch{Operations: ops}, nil
		}

		op, err := p.parseOperation(line)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", p.lineNum, err)
		}
		ops = append(ops, op)
	}

	if err := p.scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan error at line %d: %w", p.lineNum, err)
	}

	return nil, fmt.Errorf("unexpected EOF: missing '*** End Patch'")
}

// nextLine returns the next line, either from peek buffer or scanner.
func (p *Parser) nextLine() (string, bool) {
	if p.peeked != nil {
		line := *p.peeked
		p.lineNum = p.peekLine
		p.peeked = nil
		p.peekLine = 0
		return line, true
	}

	if !p.scanner.Scan() {
		return "", false
	}
	p.lineNum++
	return p.scanner.Text(), true
}

// peekLine looks ahead at the next line without consuming it.
func (p *Parser) peek() (string, bool) {
	if p.peeked != nil {
		return *p.peeked, true
	}

	if !p.scanner.Scan() {
		return "", false
	}

	line := p.scanner.Text()
	p.peeked = &line
	p.peekLine = p.lineNum + 1
	return line, true
}

// expectLine checks if the next line matches expected text.
func (p *Parser) expectLine(expected string) bool {
	line, ok := p.nextLine()
	if !ok {
		return false
	}
	return strings.TrimSpace(line) == expected
}

// parseOperation parses a single file operation.
func (p *Parser) parseOperation(line string) (FileOperation, error) {
	switch {
	case strings.HasPrefix(line, "*** Add File: "):
		path := strings.TrimSpace(strings.TrimPrefix(line, "*** Add File: "))
		return p.parseAddFile(path)
	case strings.HasPrefix(line, "*** Delete File: "):
		path := strings.TrimSpace(strings.TrimPrefix(line, "*** Delete File: "))
		return p.parseDeleteFile(path)
	case strings.HasPrefix(line, "*** Update File: "):
		path := strings.TrimSpace(strings.TrimPrefix(line, "*** Update File: "))
		return p.parseUpdateFile(path)
	default:
		return nil, fmt.Errorf("unknown operation: %q", line)
	}
}

// parseAddFile parses an add file operation.
func (p *Parser) parseAddFile(path string) (*AddFile, error) {
	// Validate path
	if strings.HasPrefix(path, "/") {
		return nil, fmt.Errorf("invalid path %q: absolute paths not allowed", path)
	}
	if strings.Contains(path, "..") {
		return nil, fmt.Errorf("invalid path %q: path traversal not allowed", path)
	}

	lines := make([]string, 0)
	for {
		line, ok := p.peek()
		if !ok {
			break
		}

		// Check for next operation or end
		if strings.HasPrefix(strings.TrimSpace(line), "***") {
			break
		}

		// Consume the line
		line, _ = p.nextLine()

		if !strings.HasPrefix(line, "+") {
			return nil, fmt.Errorf("invalid line format: expected '+' prefix, got: %q", line)
		}

		lines = append(lines, strings.TrimPrefix(line, "+"))
	}

	return &AddFile{
		FilePath: path,
		Lines:    lines,
	}, nil
}

// parseDeleteFile parses a delete file operation.
func (p *Parser) parseDeleteFile(path string) (*DeleteFile, error) {
	// Validate path
	if strings.HasPrefix(path, "/") {
		return nil, fmt.Errorf("invalid path %q: absolute paths not allowed", path)
	}
	if strings.Contains(path, "..") {
		return nil, fmt.Errorf("invalid path %q: path traversal not allowed", path)
	}

	return &DeleteFile{
		FilePath: path,
	}, nil
}

// parseUpdateFile parses an update file operation.
func (p *Parser) parseUpdateFile(path string) (*UpdateFile, error) {
	// Validate path
	if strings.HasPrefix(path, "/") {
		return nil, fmt.Errorf("invalid path %q: absolute paths not allowed", path)
	}
	if strings.Contains(path, "..") {
		return nil, fmt.Errorf("invalid path %q: path traversal not allowed", path)
	}

	update := &UpdateFile{
		FilePath: path,
		Hunks:    make([]Hunk, 0),
	}

	// Check for optional move operation or first hunk
	line, ok := p.peek()
	if !ok {
		return update, nil
	}

	line = strings.TrimSpace(line)

	// Check for move operation
	if strings.HasPrefix(line, "*** Move to: ") {
		p.nextLine() // consume the peeked line
		newPath := strings.TrimSpace(strings.TrimPrefix(line, "*** Move to: "))
		if strings.HasPrefix(newPath, "/") {
			return nil, fmt.Errorf("invalid new path %q: absolute paths not allowed", newPath)
		}
		if strings.Contains(newPath, "..") {
			return nil, fmt.Errorf("invalid new path %q: path traversal not allowed", newPath)
		}
		update.NewPath = newPath

		// After move, check for hunks
		// (checked in the parse hunks loop below)
	}

	// Parse hunks
	for {
		line, ok := p.peek()
		if !ok {
			break
		}

		line = strings.TrimSpace(line)

		// Check for next operation or end
		if strings.HasPrefix(line, "*** End Patch") || strings.HasPrefix(line, "*** Add File") ||
			strings.HasPrefix(line, "*** Delete File") || strings.HasPrefix(line, "*** Update File") {
			break
		}

		if strings.HasPrefix(line, "@@") {
			p.nextLine() // consume the peeked line
			hunk, err := p.parseHunk(line)
			if err != nil {
				return nil, err
			}
			update.Hunks = append(update.Hunks, *hunk)
		} else {
			// Unexpected line
			break
		}
	}

	return update, nil
}

// parseHunk parses a single hunk starting with @@.
func (p *Parser) parseHunk(firstLine string) (*Hunk, error) {
	hunk := p.createHunk(firstLine)

	for {
		line, ok := p.peek()
		if !ok {
			break
		}

		if p.isHunkEnd(line) {
			break
		}

		line, _ = p.nextLine()
		if err := p.parseHunkLine(hunk, line); err != nil {
			return nil, err
		}
	}

	return hunk, nil
}

// createHunk creates a new hunk with the given header.
func (p *Parser) createHunk(firstLine string) *Hunk {
	return &Hunk{
		Header:  strings.TrimSpace(strings.TrimPrefix(firstLine, "@@")),
		Changes: make([]LineChange, 0),
	}
}

// isHunkEnd checks if the line indicates the end of a hunk.
func (p *Parser) isHunkEnd(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "@@") || strings.HasPrefix(trimmed, "***")
}

// parseHunkLine parses a single line within a hunk.
func (p *Parser) parseHunkLine(hunk *Hunk, line string) error {
	if len(line) == 0 {
		hunk.Changes = append(hunk.Changes, LineChange{
			Type: LineContext,
			Text: "",
		})
		return nil
	}

	prefix := line[0]
	text := ""
	if len(line) > 1 {
		text = line[1:]
	}

	return p.addLineChange(hunk, prefix, text)
}

// addLineChange adds a line change to the hunk based on the prefix.
func (p *Parser) addLineChange(hunk *Hunk, prefix byte, text string) error {
	switch prefix {
	case ' ':
		hunk.Changes = append(hunk.Changes, LineChange{
			Type: LineContext,
			Text: text,
		})
	case '-':
		hunk.Changes = append(hunk.Changes, LineChange{
			Type: LineDelete,
			Text: text,
		})
	case '+':
		hunk.Changes = append(hunk.Changes, LineChange{
			Type: LineInsert,
			Text: text,
		})
	default:
		return fmt.Errorf("invalid line prefix: expected ' ', '-', or '+', got %q", prefix)
	}
	return nil
}
