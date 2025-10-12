# FRD-20251012030000: Patch Parser for internal/patchapply

**Feature:** Patch Parser (Feature 2.1 from tools-modules ROADMAP)
**Priority:** P0 (Blocker for all file modifications)
**Status:** Planning
**Created:** 2025-10-12 03:00:00
**Updated:** 2025-10-12 03:00:00
**Related:** [tools-modules.md](../tools-modules/tools-modules.md), [ROADMAP.md](../tools-modules/ROADMAP.md)

---

## Overview

The Patch Parser is the foundation of Spin's file modification system. It parses Spin's custom patch format (designed for AI model generation) into a structured AST that can be safely applied to the filesystem. The parser must be robust, secure, and provide excellent error messages to help AI models generate correct patches.

### Goals

1. **Parse Spin Patch Format:** Convert text patches into structured AST
2. **Validate Syntax:** Catch malformed patches with detailed error messages
3. **Security:** Validate all paths to prevent path traversal attacks
4. **Clarity:** Provide line-accurate error reporting for AI model feedback
5. **Performance:** Handle large patches (>10k lines) efficiently

### Non-Goals

- Fuzzy matching (handled by matcher.go)
- File I/O (handled by applier.go)
- Git patch format support (handled by internal/gitpatch)

---

## Background

### Problem Statement

AI coding agents need a way to modify files that is:
- **Simple to generate:** Reduces AI model errors
- **Unambiguous:** Clear intent (add/delete/update)
- **Safe:** Cannot escape workspace or cause security issues
- **Debuggable:** Clear error messages when patches fail

Standard diff formats (unified diff, context diff) are complex and error-prone for AI models to generate. Spin's custom format simplifies this while maintaining safety.

### Current State

- ✅ `pkg/pathutil` exists for path validation
- ✅ `pkg/strutil` exists for string manipulation
- ❌ No patch parsing implementation exists yet

---

## Requirements

### Functional Requirements

#### FR1: Parse Complete Patches

**Description:** Parse a complete patch from text into structured AST

**Format:**
```
*** Begin Patch
[file operations]
*** End Patch
```

**Acceptance Criteria:**
- ✅ Recognizes `*** Begin Patch` marker
- ✅ Parses all file operations between markers
- ✅ Recognizes `*** End Patch` marker
- ✅ Returns `*Patch` struct with all operations
- ✅ Returns error if markers are missing or malformed

**Test Cases:**
```go
// TC1: Valid empty patch
input := "*** Begin Patch\n*** End Patch\n"
expected := &Patch{Operations: []FileOperation{}}

// TC2: Missing begin marker
input := "*** End Patch\n"
expectError := "expected '*** Begin Patch'"

// TC3: Missing end marker
input := "*** Begin Patch\n"
expectError := "missing '*** End Patch'"

// TC4: Invalid begin marker
input := "*** Begin Patchh\n*** End Patch\n"
expectError := "expected '*** Begin Patch'"
```

---

#### FR2: Parse Add File Operations

**Description:** Parse file addition operations

**Format:**
```
*** Add File: path/to/file.txt
+Line 1 content
+Line 2 content
+Line 3 content
```

**Acceptance Criteria:**
- ✅ Recognizes `*** Add File:` marker
- ✅ Extracts file path after marker
- ✅ Collects all lines starting with `+`
- ✅ Stops at next operation or end marker
- ✅ Validates path is relative (using `pkg/pathutil`)
- ✅ Returns `*AddFile` with path and lines

**Test Cases:**
```go
// TC1: Valid add file
input := `*** Begin Patch
*** Add File: src/new.go
+package main
+
+func main() {}
*** End Patch`
expected := &AddFile{
    FilePath: "src/new.go",
    Lines: []string{"package main", "", "func main() {}"},
}

// TC2: Add empty file
input := `*** Begin Patch
*** Add File: empty.txt
*** End Patch`
expected := &AddFile{FilePath: "empty.txt", Lines: []string{}}

// TC3: Absolute path rejected
input := `*** Begin Patch
*** Add File: /etc/passwd
+malicious
*** End Patch`
expectError := "absolute paths not allowed"

// TC4: Path traversal rejected
input := `*** Begin Patch
*** Add File: ../../../etc/passwd
+malicious
*** End Patch`
expectError := "path traversal detected"
```

---

#### FR3: Parse Delete File Operations

**Description:** Parse file deletion operations

**Format:**
```
*** Delete File: path/to/file.txt
```

**Acceptance Criteria:**
- ✅ Recognizes `*** Delete File:` marker
- ✅ Extracts file path after marker
- ✅ Validates path is relative
- ✅ Returns `*DeleteFile` with path

**Test Cases:**
```go
// TC1: Valid delete
input := `*** Begin Patch
*** Delete File: old/file.go
*** End Patch`
expected := &DeleteFile{FilePath: "old/file.go"}

// TC2: Absolute path rejected
input := `*** Begin Patch
*** Delete File: /etc/passwd
*** End Patch`
expectError := "absolute paths not allowed"

// TC3: Path traversal rejected
input := `*** Begin Patch
*** Delete File: ../../etc/passwd
*** End Patch`
expectError := "path traversal detected"
```

---

#### FR4: Parse Update File Operations

**Description:** Parse file update operations with hunks

**Format:**
```
*** Update File: path/to/file.go
@@ context header
 context line
-old line
+new line
 context line
```

**Acceptance Criteria:**
- ✅ Recognizes `*** Update File:` marker
- ✅ Extracts file path
- ✅ Parses optional `*** Move to:` for renames
- ✅ Parses multiple hunks
- ✅ Each hunk has optional `@@` header
- ✅ Collects context (` `), delete (`-`), insert (`+`) lines
- ✅ Validates path is relative
- ✅ Returns `*UpdateFile` with path, optional newPath, and hunks

**Test Cases:**
```go
// TC1: Simple update
input := `*** Begin Patch
*** Update File: main.go
@@
 func main() {
-    fmt.Println("old")
+    fmt.Println("new")
 }
*** End Patch`
expected := &UpdateFile{
    FilePath: "main.go",
    Hunks: []Hunk{{
        Header: "",
        Changes: []LineChange{
            {Type: LineContext, Text: "func main() {"},
            {Type: LineDelete, Text: "    fmt.Println(\"old\")"},
            {Type: LineInsert, Text: "    fmt.Println(\"new\")"},
            {Type: LineContext, Text: "}"},
        },
    }},
}

// TC2: Update with function context
input := `*** Begin Patch
*** Update File: handler.go
@@ func (h *Handler) Process
 func (h *Handler) Process(data string) error {
-    return oldValue
+    return newValue
 }
*** End Patch`
expected := &UpdateFile{
    FilePath: "handler.go",
    Hunks: []Hunk{{
        Header: "func (h *Handler) Process",
        Changes: []LineChange{
            {Type: LineContext, Text: "func (h *Handler) Process(data string) error {"},
            {Type: LineDelete, Text: "    return oldValue"},
            {Type: LineInsert, Text: "    return newValue"},
            {Type: LineContext, Text: "}"},
        },
    }},
}

// TC3: Move/rename operation
input := `*** Begin Patch
*** Update File: old/path.go
*** Move to: new/path.go
@@
 content
*** End Patch`
expected := &UpdateFile{
    FilePath: "old/path.go",
    NewPath: "new/path.go",
    Hunks: []Hunk{{Changes: []LineChange{{Type: LineContext, Text: "content"}}}},
}

// TC4: Multiple hunks
input := `*** Begin Patch
*** Update File: multi.go
@@
-old1
+new1
@@
-old2
+new2
*** End Patch`
expected := &UpdateFile{
    FilePath: "multi.go",
    Hunks: []Hunk{
        {Changes: []LineChange{
            {Type: LineDelete, Text: "old1"},
            {Type: LineInsert, Text: "new1"},
        }},
        {Changes: []LineChange{
            {Type: LineDelete, Text: "old2"},
            {Type: LineInsert, Text: "new2"},
        }},
    },
}
```

---

#### FR5: Error Reporting with Line Numbers

**Description:** Provide detailed error messages with line numbers

**Acceptance Criteria:**
- ✅ All errors include line number where error occurred
- ✅ Errors include helpful context about what was expected
- ✅ Path validation errors include the invalid path
- ✅ Syntax errors describe the malformed input

**Test Cases:**
```go
// TC1: Line number in error
input := `*** Begin Patch
*** Add File: test.txt
+line 1
invalid line without prefix
+line 2
*** End Patch`
expectError := "line 4: invalid line format: expected '+' prefix"

// TC2: Unknown operation type
input := `*** Begin Patch
*** Unknown Operation: test.txt
*** End Patch`
expectError := "line 2: unknown operation: '*** Unknown Operation: test.txt'"

// TC3: Path validation in error
input := `*** Begin Patch
*** Add File: /etc/passwd
+content
*** End Patch`
expectError := "line 2: invalid path '/etc/passwd': absolute paths not allowed"
```

---

### Non-Functional Requirements

#### NFR1: Performance

**Target:** Parse 10,000 line patch in <100ms

**Rationale:** Large refactorings may generate large patches. Parser should not be a bottleneck.

**Measurement:**
```go
func BenchmarkParseLargePatch(b *testing.B) {
    patch := generatePatch(10000) // 10k lines
    for i := 0; i < b.N; i++ {
        p := NewParser(patch)
        p.Parse()
    }
}
// Target: <100ms per operation
```

---

#### NFR2: Memory Efficiency

**Target:** <1MB memory overhead for 10k line patch

**Rationale:** Parser should not consume excessive memory for large patches.

**Measurement:** Use `testing.AllocsPerRun()` and memory profiling

---

#### NFR3: Complexity

**Target:** Cyclomatic complexity ≤10 per function

**Rationale:** Parser functions should be simple and testable.

**Measurement:** Use `gocyclo -over 10 ./internal/patchapply/`

---

### Security Requirements

#### SR1: Path Validation

**Requirement:** All file paths must be validated using `pkg/pathutil.ValidateRelativePath()`

**Threat Model:**
- Path traversal attacks: `../../etc/passwd`
- Absolute path injection: `/etc/passwd`
- Symlink escape: `link_to_etc/passwd` (validated by applier, not parser)

**Validation:**
```go
func validatePath(path string) error {
    if err := pathutil.ValidateRelativePath(path); err != nil {
        return fmt.Errorf("invalid path %q: %w", path, err)
    }
    return nil
}
```

---

#### SR2: Input Validation

**Requirement:** Reject malformed input gracefully without panics

**Threat Model:**
- Malicious patches designed to crash parser
- Extremely long lines (>1MB)
- Invalid UTF-8
- Control characters

**Validation:**
- No panics on malformed input (test with fuzzing)
- Reject lines >100k characters
- Handle invalid UTF-8 gracefully

---

#### SR3: Resource Limits

**Requirement:** Prevent resource exhaustion attacks

**Threat Model:**
- Patches with millions of operations
- Patches with extremely long lines

**Limits:**
- Max patch size: 100MB
- Max operations: 10,000
- Max line length: 100k characters

---

## Design

### Architecture

```
Parser
  ├── NewParser(text string) *Parser
  ├── Parse() (*Patch, error)
  └── Internal methods:
      ├── expectLine(expected string) bool
      ├── parseOperation(line string) (FileOperation, error)
      ├── parseAddFile(path string) (*AddFile, error)
      ├── parseDeleteFile(path string) (*DeleteFile, error)
      ├── parseUpdateFile(path string) (*UpdateFile, error)
      └── parseHunk() (*Hunk, error)
```

### Data Structures

```go
// Patch represents a complete patch
type Patch struct {
    Operations []FileOperation
}

// FileOperation is a union type for all file operations
type FileOperation interface {
    isFileOperation()
    Path() string
}

// AddFile adds a new file
type AddFile struct {
    FilePath string
    Lines    []string
}

func (a *AddFile) isFileOperation() {}
func (a *AddFile) Path() string     { return a.FilePath }

// DeleteFile deletes a file
type DeleteFile struct {
    FilePath string
}

func (d *DeleteFile) isFileOperation() {}
func (d *DeleteFile) Path() string     { return d.FilePath }

// UpdateFile updates an existing file
type UpdateFile struct {
    FilePath string
    NewPath  string // Optional, for move operations
    Hunks    []Hunk
}

func (u *UpdateFile) isFileOperation() {}
func (u *UpdateFile) Path() string     { return u.FilePath }

// Hunk represents a change section
type Hunk struct {
    Header  string      // Optional context (e.g., "func MyFunc")
    Changes []LineChange
}

// LineChange represents a single line operation
type LineChange struct {
    Type LineChangeType
    Text string
}

type LineChangeType int

const (
    LineContext LineChangeType = iota // " " prefix
    LineDelete                        // "-" prefix
    LineInsert                        // "+" prefix
)

func (t LineChangeType) String() string {
    switch t {
    case LineContext:
        return "context"
    case LineDelete:
        return "delete"
    case LineInsert:
        return "insert"
    default:
        return "unknown"
    }
}
```

### Parser Implementation

```go
package patchapply

import (
    "bufio"
    "fmt"
    "strings"

    "github.com/dmytrogajewski/spin/pkg/pathutil"
)

// Parser parses patch text into structured Patch
type Parser struct {
    scanner *bufio.Scanner
    lineNum int
}

// NewParser creates a new patch parser
func NewParser(text string) *Parser {
    return &Parser{
        scanner: bufio.NewScanner(strings.NewReader(text)),
        lineNum: 0,
    }
}

// Parse parses the entire patch
func (p *Parser) Parse() (*Patch, error) {
    if !p.expectLine("*** Begin Patch") {
        return nil, fmt.Errorf("line %d: expected '*** Begin Patch'", p.lineNum)
    }

    var ops []FileOperation
    for p.scanner.Scan() {
        line := p.scanner.Text()
        p.lineNum++

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

// expectLine checks if the next line matches expected text
func (p *Parser) expectLine(expected string) bool {
    if !p.scanner.Scan() {
        return false
    }
    p.lineNum++
    return strings.TrimSpace(p.scanner.Text()) == expected
}

// parseOperation parses a single file operation
func (p *Parser) parseOperation(line string) (FileOperation, error) {
    line = strings.TrimSpace(line)

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

// parseAddFile parses an add file operation
func (p *Parser) parseAddFile(path string) (*AddFile, error) {
    // Validate path
    if err := pathutil.ValidateRelativePath(path); err != nil {
        return nil, fmt.Errorf("invalid path %q: %w", path, err)
    }

    lines := make([]string, 0)
    for p.scanner.Scan() {
        line := p.scanner.Text()
        p.lineNum++

        // Check for next operation or end
        if strings.HasPrefix(line, "***") {
            // Put line back (conceptually - we'll handle in Parse)
            // For now, we need to track this differently
            break
        }

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

// parseDeleteFile parses a delete file operation
func (p *Parser) parseDeleteFile(path string) (*DeleteFile, error) {
    // Validate path
    if err := pathutil.ValidateRelativePath(path); err != nil {
        return nil, fmt.Errorf("invalid path %q: %w", path, err)
    }

    return &DeleteFile{
        FilePath: path,
    }, nil
}

// parseUpdateFile parses an update file operation
func (p *Parser) parseUpdateFile(path string) (*UpdateFile, error) {
    // Validate path
    if err := pathutil.ValidateRelativePath(path); err != nil {
        return nil, fmt.Errorf("invalid path %q: %w", path, err)
    }

    update := &UpdateFile{
        FilePath: path,
        Hunks:    make([]Hunk, 0),
    }

    // Check for optional move operation
    if p.scanner.Scan() {
        line := p.scanner.Text()
        p.lineNum++

        if strings.HasPrefix(line, "*** Move to: ") {
            newPath := strings.TrimSpace(strings.TrimPrefix(line, "*** Move to: "))
            if err := pathutil.ValidateRelativePath(newPath); err != nil {
                return nil, fmt.Errorf("invalid new path %q: %w", newPath, err)
            }
            update.NewPath = newPath
        } else if strings.HasPrefix(line, "@@") {
            // Start of first hunk
            hunk, err := p.parseHunk(line)
            if err != nil {
                return nil, err
            }
            update.Hunks = append(update.Hunks, *hunk)
        }
    }

    // Parse remaining hunks
    for p.scanner.Scan() {
        line := p.scanner.Text()
        p.lineNum++

        if strings.HasPrefix(line, "***") {
            // Next operation
            break
        }

        if strings.HasPrefix(line, "@@") {
            hunk, err := p.parseHunk(line)
            if err != nil {
                return nil, err
            }
            update.Hunks = append(update.Hunks, *hunk)
        }
    }

    return update, nil
}

// parseHunk parses a single hunk starting with @@
func (p *Parser) parseHunk(firstLine string) (*Hunk, error) {
    hunk := &Hunk{
        Header:  strings.TrimSpace(strings.TrimPrefix(firstLine, "@@")),
        Changes: make([]LineChange, 0),
    }

    for p.scanner.Scan() {
        line := p.scanner.Text()
        p.lineNum++

        // Check for end of hunk
        if strings.HasPrefix(line, "@@") || strings.HasPrefix(line, "***") {
            // Next hunk or operation
            break
        }

        if len(line) == 0 {
            // Empty line is context
            hunk.Changes = append(hunk.Changes, LineChange{
                Type: LineContext,
                Text: "",
            })
            continue
        }

        prefix := line[0]
        text := ""
        if len(line) > 1 {
            text = line[1:]
        }

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
            return nil, fmt.Errorf("invalid line prefix: expected ' ', '-', or '+', got %q", prefix)
        }
    }

    return hunk, nil
}
```

---

## Testing Strategy

### Unit Tests

**Coverage Target:** ≥95%

#### Test Categories

1. **Valid Patches**
   - Empty patch
   - Single add file
   - Single delete file
   - Single update file
   - Multiple operations
   - Complex hunks with context headers

2. **Syntax Errors**
   - Missing begin marker
   - Missing end marker
   - Unknown operation type
   - Invalid line prefixes
   - Malformed headers

3. **Path Validation**
   - Absolute paths
   - Path traversal attempts
   - Empty paths
   - Valid relative paths

4. **Edge Cases**
   - Empty files
   - Files with only whitespace
   - Very long lines
   - Unicode content
   - Mixed line endings

5. **Error Messages**
   - Line numbers are accurate
   - Error messages are descriptive
   - Path shown in errors

### Table-Driven Tests

```go
func TestParser_Parse(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    *Patch
        wantErr string
    }{
        {
            name: "empty patch",
            input: "*** Begin Patch\n*** End Patch\n",
            want: &Patch{Operations: []FileOperation{}},
        },
        {
            name: "add file",
            input: `*** Begin Patch
*** Add File: test.txt
+hello
+world
*** End Patch`,
            want: &Patch{
                Operations: []FileOperation{
                    &AddFile{
                        FilePath: "test.txt",
                        Lines:    []string{"hello", "world"},
                    },
                },
            },
        },
        {
            name: "missing begin marker",
            input: "*** End Patch\n",
            wantErr: "expected '*** Begin Patch'",
        },
        {
            name: "absolute path rejected",
            input: `*** Begin Patch
*** Add File: /etc/passwd
+content
*** End Patch`,
            wantErr: "absolute paths not allowed",
        },
        // ... more test cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            p := NewParser(tt.input)
            got, err := p.Parse()

            if tt.wantErr != "" {
                if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
                    t.Errorf("Parse() error = %v, wantErr %q", err, tt.wantErr)
                }
                return
            }

            if err != nil {
                t.Errorf("Parse() unexpected error = %v", err)
                return
            }

            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("Parse() = %+v, want %+v", got, tt.want)
            }
        })
    }
}
```

### Benchmark Tests

```go
func BenchmarkParser_Parse_Small(b *testing.B) {
    patch := generateSmallPatch() // 100 lines
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        p := NewParser(patch)
        _, _ = p.Parse()
    }
}

func BenchmarkParser_Parse_Large(b *testing.B) {
    patch := generateLargePatch() // 10k lines
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        p := NewParser(patch)
        _, _ = p.Parse()
    }
}
```

### Fuzzing Tests

```go
func FuzzParser(f *testing.F) {
    // Seed corpus
    f.Add("*** Begin Patch\n*** End Patch\n")
    f.Add("*** Begin Patch\n*** Add File: test.txt\n+content\n*** End Patch\n")

    f.Fuzz(func(t *testing.T, input string) {
        p := NewParser(input)
        _, _ = p.Parse() // Should not panic
    })
}
```

---

## Implementation Plan

### Phase 1: Core Types (1 day)

**File:** `internal/patchapply/types.go`

- [ ] Define `Patch` struct
- [ ] Define `FileOperation` interface
- [ ] Define `AddFile`, `DeleteFile`, `UpdateFile` structs
- [ ] Define `Hunk` and `LineChange` structs
- [ ] Define `LineChangeType` constants
- [ ] Add godoc comments
- [ ] Add String() methods for debugging

### Phase 2: Parser Structure (1 day)

**File:** `internal/patchapply/parser.go`

- [ ] Define `Parser` struct
- [ ] Implement `NewParser()`
- [ ] Implement `Parse()` skeleton
- [ ] Implement `expectLine()` helper
- [ ] Implement `parseOperation()` dispatcher

### Phase 3: Operation Parsers (2 days)

**File:** `internal/patchapply/parser.go`

- [ ] Implement `parseAddFile()`
- [ ] Implement `parseDeleteFile()`
- [ ] Implement `parseUpdateFile()`
- [ ] Implement `parseHunk()`
- [ ] Add path validation using `pkg/pathutil`
- [ ] Add line number tracking
- [ ] Add error messages with context

### Phase 4: Testing (2 days)

**File:** `internal/patchapply/parser_test.go`

- [ ] Write test fixtures
- [ ] Write table-driven tests for valid patches
- [ ] Write tests for syntax errors
- [ ] Write tests for path validation
- [ ] Write tests for edge cases
- [ ] Write benchmark tests
- [ ] Write fuzz tests
- [ ] Achieve ≥95% coverage

### Phase 5: Analysis & Refinement (1 day)

- [ ] Run `uast parse parser.go | herr analyze`
- [ ] Run `make lint`
- [ ] Run `gocyclo -over 10 ./internal/patchapply/`
- [ ] Refactor complex functions
- [ ] Optimize hot paths
- [ ] Final test pass

**Total Estimated Time:** 7 days

---

## Acceptance Criteria

### Functionality

- [ ] ✅ Parses all operation types: Add, Delete, Update
- [ ] ✅ Handles optional move operations
- [ ] ✅ Parses multiple hunks per update
- [ ] ✅ Extracts context headers correctly
- [ ] ✅ Validates all file paths
- [ ] ✅ Rejects absolute paths
- [ ] ✅ Rejects path traversal attempts

### Error Handling

- [ ] ✅ All errors include line numbers
- [ ] ✅ Error messages are descriptive
- [ ] ✅ Invalid paths shown in errors
- [ ] ✅ No panics on malformed input

### Quality

- [ ] ✅ Test coverage ≥95%
- [ ] ✅ `make lint` passes (zero errors)
- [ ] ✅ Cyclomatic complexity ≤10 per function
- [ ] ✅ `go test -race` passes
- [ ] ✅ All godoc comments present

### Performance

- [ ] ✅ Parses 10k line patch in <100ms
- [ ] ✅ Memory overhead <1MB for 10k lines
- [ ] ✅ No performance regressions from baseline

### Security

- [ ] ✅ Path validation using `pkg/pathutil`
- [ ] ✅ No panics on fuzzing input
- [ ] ✅ Resource limits enforced

---

## Dependencies

### Internal Packages

- `pkg/pathutil` - Path validation (REQUIRED)
- `pkg/strutil` - String utilities (optional, for future enhancements)

### Standard Library

- `bufio` - Line-by-line scanning
- `strings` - String manipulation
- `fmt` - Error formatting

---

## Risks & Mitigations

### Risk 1: Complex Parser Logic

**Impact:** High complexity → bugs and maintenance burden

**Probability:** Medium

**Mitigation:**
- Use table-driven tests for all cases
- Keep functions small (≤50 LOC)
- Use `gocyclo` to enforce complexity limits
- Code review with focus on edge cases

### Risk 2: Performance Issues

**Impact:** Slow parsing blocks file modifications

**Probability:** Low

**Mitigation:**
- Use `bufio.Scanner` for efficient line reading
- Benchmark early and often
- Profile hot paths with `pprof`
- Add performance regression tests

### Risk 3: Security Vulnerabilities

**Impact:** Path traversal or DoS attacks

**Probability:** Low

**Mitigation:**
- Use `pkg/pathutil` for all path validation
- Fuzz testing to find crashes
- Resource limits (max patch size, max operations)
- Security review before release

---

## Success Metrics

### Coverage

- **Target:** ≥95% test coverage
- **Measurement:** `go test -cover ./internal/patchapply/`

### Quality

- **Target:** Zero lint errors
- **Measurement:** `make lint`

### Performance

- **Target:** <100ms for 10k line patch
- **Measurement:** `go test -bench=. ./internal/patchapply/`

### Complexity

- **Target:** Cyclomatic complexity ≤10
- **Measurement:** `gocyclo -over 10 ./internal/patchapply/`

---

## Documentation

### Package Documentation

**File:** `internal/patchapply/doc.go`

```go
// Package patchapply provides file modification via structured patches.
//
// The package implements Spin's custom patch format designed for AI models
// to generate correct, safe file modifications. Unlike standard diff formats,
// Spin's format is simple, unambiguous, and resistant to generation errors.
//
// # Patch Format
//
// A patch consists of operations enclosed in Begin/End markers:
//
//     *** Begin Patch
//     *** Add File: path/to/file.txt
//     +line 1
//     +line 2
//     *** End Patch
//
// Supported operations:
//   - Add File: Create new file with content
//   - Delete File: Remove existing file
//   - Update File: Modify file with hunks
//
// # Security
//
// All file paths are validated using pkg/pathutil to prevent:
//   - Path traversal attacks (../../etc/passwd)
//   - Absolute path injection (/etc/passwd)
//   - Symlink escapes (link_to_etc/passwd)
//
// # Usage
//
//     parser := patchapply.NewParser(patchText)
//     patch, err := parser.Parse()
//     if err != nil {
//         log.Fatal(err)
//     }
//
//     applier := patchapply.NewApplier("/workspace")
//     if err := applier.Apply(patch); err != nil {
//         log.Fatal(err)
//     }
//
// # Error Handling
//
// Parse errors include line numbers for debugging:
//
//     line 5: invalid path "/etc/passwd": absolute paths not allowed
//
// This helps AI models understand and correct their patch generation.
package patchapply
```

### Function Documentation

All exported functions must have godoc comments following Go conventions:

```go
// NewParser creates a new patch parser for the given text.
//
// The parser uses a streaming approach with bufio.Scanner for memory efficiency.
// It can handle large patches (>10k lines) without loading everything into memory.
func NewParser(text string) *Parser { ... }

// Parse parses the complete patch and returns a structured Patch AST.
//
// Returns an error if:
//   - Begin/End markers are missing or malformed
//   - Any file paths are invalid (absolute, traversal, etc.)
//   - Syntax is incorrect (invalid prefixes, unknown operations)
//
// Errors include line numbers for precise debugging.
func (p *Parser) Parse() (*Patch, error) { ... }
```

---

## References

1. [Spin Tools & Utility Modules](../tools-modules/tools-modules.md)
2. [Tools-Modules ROADMAP](../tools-modules/ROADMAP.md)
3. [pkg/pathutil Documentation](../../docs/packages/pathutil.md)
4. [pkg/strutil Documentation](../../docs/packages/strutil.md)
5. [AGENTS.md](../../AGENTS.md) - Development workflow

---

## Appendix A: Patch Format Grammar

```ebnf
Patch     = Begin Operation* End
Begin     = "*** Begin Patch" NEWLINE
End       = "*** End Patch" NEWLINE

Operation = AddFile | DeleteFile | UpdateFile

AddFile   = "*** Add File: " Path NEWLINE AddLine*
AddLine   = "+" TEXT NEWLINE

DeleteFile = "*** Delete File: " Path NEWLINE

UpdateFile = "*** Update File: " Path NEWLINE MoveTo? Hunk+
MoveTo    = "*** Move to: " Path NEWLINE

Hunk      = "@@" Header? NEWLINE HunkLine*
Header    = TEXT
HunkLine  = ContextLine | DeleteLine | InsertLine

ContextLine = " " TEXT NEWLINE
DeleteLine  = "-" TEXT NEWLINE
InsertLine  = "+" TEXT NEWLINE

Path      = RELATIVE_PATH
TEXT      = [^\n]*
NEWLINE   = "\n"
```

---

## Appendix B: Example Patches

### Example 1: Add New File

```
*** Begin Patch
*** Add File: src/handler/new_handler.go
+package handler
+
+import "context"
+
+// NewHandler creates a new handler
+func NewHandler() *Handler {
+    return &Handler{}
+}
+
+// Handler processes requests
+type Handler struct {}
*** End Patch
```

### Example 2: Delete File

```
*** Begin Patch
*** Delete File: src/deprecated/old_code.go
*** End Patch
```

### Example 3: Update with Context

```
*** Begin Patch
*** Update File: src/config/config.go
@@ type Config struct
 type Config struct {
     Host string
     Port int
-    Timeout time.Duration
+    Timeout int // seconds
 }
*** End Patch
```

### Example 4: Rename and Update

```
*** Begin Patch
*** Update File: src/handler.go
*** Move to: src/handler/handler.go
@@
 package handler

 import (
-    "log"
+    "github.com/sirupsen/logrus"
 )
*** End Patch
```

### Example 5: Multiple Hunks

```
*** Begin Patch
*** Update File: src/server.go
@@ func (s *Server) Start
 func (s *Server) Start() error {
-    log.Println("Starting server")
+    s.logger.Info("Starting server")
     return s.listenAndServe()
 }

@@ func (s *Server) Stop
 func (s *Server) Stop() error {
-    log.Println("Stopping server")
+    s.logger.Info("Stopping server")
     return s.shutdown()
 }
*** End Patch
```

---

**END OF FRD**
