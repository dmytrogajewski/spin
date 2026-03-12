package patchapply

import (
	"bufio"
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrLine is a sentinel error.
	ErrLine = errors.New("line")
	// ErrUnexpectedEOFMissingEndPatch is a sentinel error.
	ErrUnexpectedEOFMissingEndPatch = errors.New("unexpected EOF: missing '*** End Patch'")
	// ErrUnknownOperation is a sentinel error.
	ErrUnknownOperation = errors.New("unknown operation")
	// ErrInvalidPath is a sentinel error.
	ErrInvalidPath = errors.New("invalid path")
	// ErrInvalidPath2 is a sentinel error.
	ErrInvalidPath2 = errors.New("invalid path")
	// ErrInvalidLineFormat is a sentinel error.
	ErrInvalidLineFormat = errors.New("invalid line format")
	// ErrInvalidPath3 is a sentinel error.
	ErrInvalidPath3 = errors.New("invalid path")
	// ErrInvalidPath4 is a sentinel error.
	ErrInvalidPath4 = errors.New("invalid path")
	// ErrInvalidPath5 is a sentinel error.
	ErrInvalidPath5 = errors.New("invalid path")
	// ErrInvalidPath6 is a sentinel error.
	ErrInvalidPath6 = errors.New("invalid path")
	// ErrInvalidNewPath is a sentinel error.
	ErrInvalidNewPath = errors.New("invalid new path")
	// ErrInvalidNewPath2 is a sentinel error.
	ErrInvalidNewPath2 = errors.New("invalid new path")
	// ErrInvalidLinePrefix is a sentinel error.
	ErrInvalidLinePrefix = errors.New("invalid line prefix")
)

// Parser parses patch text into structured Patch AST.
// It uses a streaming approach with [bufio.Scanner] for memory efficiency.
type Parser struct {
	scanner  *bufio.Scanner
	lineNum  int
	peeked   *string // Peeked line for lookahead.
	peekLine int     // Line number of peeked line.
}

// NewParser creates a new patch parser for the given text.
//
// The parser uses a streaming approach with [bufio.Scanner] for memory efficiency.
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
		return nil, fmt.Errorf("line %d: expected '*** Begin Patch': %w", p.lineNum, ErrLine)
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

	err := p.scanner.Err()
	if err != nil {
		return nil, fmt.Errorf("scan error at line %d: %w", p.lineNum, err)
	}

	return nil, ErrUnexpectedEOFMissingEndPatch
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
		return nil, fmt.Errorf("unknown operation: %q: %w", line, ErrUnknownOperation)
	}
}

// parseAddFile parses an add file operation.
func (p *Parser) parseAddFile(path string) (*AddFile, error) {
	// Validate path.
	if strings.HasPrefix(path, "/") {
		return nil, fmt.Errorf("invalid path %q: absolute paths not allowed: %w", path, ErrInvalidPath)
	}

	if strings.Contains(path, "..") {
		return nil, fmt.Errorf("invalid path %q: path traversal not allowed: %w", path, ErrInvalidPath2)
	}

	lines := make([]string, 0)

	for {
		line, ok := p.peek()
		if !ok {
			break
		}

		// Check for next operation or end.
		if strings.HasPrefix(strings.TrimSpace(line), "***") {
			break
		}

		// Consume the line.
		line, _ = p.nextLine()

		if !strings.HasPrefix(line, "+") {
			return nil, fmt.Errorf("invalid line format: expected '+' prefix, got: %q: %w", line, ErrInvalidLineFormat)
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
	// Validate path.
	if strings.HasPrefix(path, "/") {
		return nil, fmt.Errorf("invalid path %q: absolute paths not allowed: %w", path, ErrInvalidPath3)
	}

	if strings.Contains(path, "..") {
		return nil, fmt.Errorf("invalid path %q: path traversal not allowed: %w", path, ErrInvalidPath4)
	}

	return &DeleteFile{
		FilePath: path,
	}, nil
}

// parseUpdateFile parses an update file operation.
func (p *Parser) parseUpdateFile(path string) (*UpdateFile, error) {
	if err := validatePath(path, ErrInvalidPath5, ErrInvalidPath6); err != nil {
		return nil, err
	}

	update := &UpdateFile{
		FilePath: path,
		Hunks:    make([]Hunk, 0),
	}

	if err := p.parseMoveDirective(update); err != nil {
		return nil, err
	}

	if err := p.parseHunks(update); err != nil {
		return nil, err
	}

	return update, nil
}

// validatePath checks a path for absolute paths and path traversal.
func validatePath(path string, absErr, traversalErr error) error {
	if strings.HasPrefix(path, "/") {
		return fmt.Errorf("invalid path %q: absolute paths not allowed: %w", path, absErr)
	}

	if strings.Contains(path, "..") {
		return fmt.Errorf("invalid path %q: path traversal not allowed: %w", path, traversalErr)
	}

	return nil
}

// parseMoveDirective parses an optional "*** Move to:" directive.
func (p *Parser) parseMoveDirective(update *UpdateFile) error {
	line, ok := p.peek()
	if !ok {
		return nil
	}

	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "*** Move to: ") {
		return nil
	}

	p.nextLine()

	newPath := strings.TrimSpace(strings.TrimPrefix(line, "*** Move to: "))
	if err := validatePath(newPath, ErrInvalidNewPath, ErrInvalidNewPath2); err != nil {
		return err
	}

	update.NewPath = newPath

	return nil
}

// isOperationBoundary returns true if the line starts a new operation or ends the patch.
func isOperationBoundary(line string) bool {
	return strings.HasPrefix(line, "*** End Patch") ||
		strings.HasPrefix(line, "*** Add File") ||
		strings.HasPrefix(line, "*** Delete File") ||
		strings.HasPrefix(line, "*** Update File")
}

// parseHunks parses all hunks for an update operation.
func (p *Parser) parseHunks(update *UpdateFile) error {
	for {
		line, ok := p.peek()
		if !ok {
			break
		}

		line = strings.TrimSpace(line)
		if isOperationBoundary(line) || !strings.HasPrefix(line, "@@") {
			break
		}

		p.nextLine()

		hunk, err := p.parseHunk(line)
		if err != nil {
			return err
		}

		update.Hunks = append(update.Hunks, *hunk)
	}

	return nil
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

		err := p.parseHunkLine(hunk, line)
		if err != nil {
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
	if line == "" {
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
		return fmt.Errorf("invalid line prefix: expected ' ', '-', or '+', got %q: %w", prefix, ErrInvalidLinePrefix)
	}

	return nil
}
