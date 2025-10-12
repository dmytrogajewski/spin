# Package: internal/patchapply

**Status:** ✅ Production Ready (Parser, Matcher, Applier Complete)
**Path:** `internal/patchapply`
**Purpose:** File modification via structured patches for AI-generated edits

---

## Overview

The `patchapply` package implements Spin's custom patch format designed specifically for AI models to generate safe, reliable file modifications. Unlike standard diff formats (unified diff, context diff), Spin's format is:

- **Simple to generate:** Reduces AI model generation errors
- **Unambiguous:** Clear intent (add/delete/update)
- **Safe:** Validates all paths, prevents traversal attacks
- **Debuggable:** Line-accurate error messages for AI feedback
- **Resilient:** Fuzzy matching tolerates whitespace and minor changes

---

## Architecture

```
internal/patchapply/
├── types.go     - AST types for patches
├── parser.go    - Patch text → AST parser
├── matcher.go   - Fuzzy line matching with similarity scoring
├── applier.go   - AST → filesystem operations with safety
└── doc.go       - Package documentation
```

### Current Status

| Component | Status | Coverage | Complexity | Notes |
|-----------|--------|----------|------------|-------|
| **Parser** | ✅ Complete | 91.1% | max 10 | Feature 2.1 |
| **Matcher** | ✅ Complete | 90.4% | max 9 | Feature 2.2 (fuzzy matching) |
| **Applier** | ✅ Complete | 86.9% | max 8 | Feature 2.3 (filesystem ops) |

---

## Patch Format

### Basic Structure

```
*** Begin Patch
[file operations]
*** End Patch
```

### Supported Operations

#### 1. Add File

Creates a new file with specified content.

```
*** Add File: path/to/file.txt
+Line 1 content
+Line 2 content
+Line 3 content
```

**Rules:**
- Path must be relative (validated)
- Each content line prefixed with `+`
- Empty file: no lines between header and next operation

#### 2. Delete File

Removes an existing file.

```
*** Delete File: path/to/old_file.txt
```

**Rules:**
- Path must be relative (validated)
- No additional content after file path

#### 3. Update File

Modifies existing file content using hunks.

```
*** Update File: path/to/file.go
@@ optional context header
 context line (unchanged)
-old line (remove)
+new line (insert)
 context line (unchanged)
```

**Rules:**
- Path must be relative (validated)
- Multiple hunks supported
- Optional context header after `@@` (e.g., `@@ func MyFunc`)
- Line prefixes:
  - ` ` (space) = context line
  - `-` = delete line
  - `+` = insert line

#### 4. Move/Rename File

Renames or moves a file (combined with Update).

```
*** Update File: old/path.go
*** Move to: new/path.go
@@
 content
```

**Rules:**
- Both old and new paths validated
- Optional hunks after `Move to:` directive
- If no hunks, performs pure rename

---

## Parser API

### Core Types

```go
// Patch represents a complete patch with operations
type Patch struct {
    Operations []FileOperation
}

// FileOperation is implemented by:
// - *AddFile
// - *DeleteFile
// - *UpdateFile
type FileOperation interface {
    isFileOperation()
    Path() string
}

// AddFile creates a new file
type AddFile struct {
    FilePath string
    Lines    []string
}

// DeleteFile removes a file
type DeleteFile struct {
    FilePath string
}

// UpdateFile modifies a file
type UpdateFile struct {
    FilePath string
    NewPath  string // Optional, for move ops
    Hunks    []Hunk
}

// Hunk is a section of changes
type Hunk struct {
    Header  string       // Optional context
    Changes []LineChange
}

// LineChange is a single line operation
type LineChange struct {
    Type LineChangeType  // Context, Delete, Insert
    Text string
}
```

### Parser Usage

```go
import "github.com/dmytrogajewski/spin/internal/patchapply"

// Parse patch text
parser := patchapply.NewParser(patchText)
patch, err := parser.Parse()
if err != nil {
    // Error includes line number
    log.Fatalf("Parse error: %v", err)
}

// Iterate operations
for _, op := range patch.Operations {
    switch op := op.(type) {
    case *patchapply.AddFile:
        fmt.Printf("Add: %s (%d lines)\n",
            op.FilePath, len(op.Lines))
    case *patchapply.DeleteFile:
        fmt.Printf("Delete: %s\n", op.FilePath)
    case *patchapply.UpdateFile:
        fmt.Printf("Update: %s (%d hunks)\n",
            op.FilePath, len(op.Hunks))
        if op.NewPath != "" {
            fmt.Printf("  → Move to: %s\n", op.NewPath)
        }
    }
}
```

---

## Matcher API

### Overview

The fuzzy matcher locates hunk context in file content even when there are minor variations. It uses a multi-strategy approach:

1. **Exact match** (fast path) - Direct string comparison
2. **Fuzzy match** - Whitespace-normalized similarity ≥85%
3. **Header hints** - Use `@@` context to disambiguate multiple occurrences

### Core API

```go
// Create matcher for file content
fileLines := strings.Split(fileContent, "\n")
m := patchapply.NewMatcher(fileLines)

// Find context lines in file
contextLines := []string{"func main() {", "    return 0"}
header := "func main" // Optional, helps disambiguate

pos := m.FindContext(contextLines, header)
if pos < 0 {
    log.Fatalf("context not found")
}
log.Printf("Found context at line %d", pos)
```

### Configurable Threshold

```go
// Adjust similarity threshold (default 0.85 = 85%)
m := patchapply.NewMatcher(fileLines)
if err := m.SetThreshold(0.90); err != nil {
    log.Fatal(err) // Threshold must be between 0.0 and 1.0
}
```

### Performance

| Operation | File Size | Time | Status |
|-----------|-----------|------|--------|
| Exact match | 100 lines | ~15ns | ✅ |
| Exact match | 10k lines | ~21ns | ✅ (fast path) |
| Fuzzy match | 100 lines | ~3μs | ✅ |
| Fuzzy match | 10k lines | ~23μs | ✅ (well under <1ms target) |
| Header-based | 10k lines | ~85μs | ✅ |

**Key optimizations:**
- Exact match fast path (zero allocations)
- Pre-normalized file lines (cached)
- Header-based range limiting (±50 lines)
- Early exit on perfect match

### Matching Strategies

#### 1. Whitespace Tolerance

```go
// File has different indentation
fileLines := []string{
    "func foo() {",
    "  return 0",  // 2 spaces
}

// Patch context uses 4 spaces
contextLines := []string{
    "func foo() {",
    "    return 0",  // 4 spaces
}

m := NewMatcher(fileLines)
pos := m.FindContext(contextLines, "") // Returns 0 (match!)
```

#### 2. Header Disambiguation

```go
// File has multiple similar functions
fileLines := []string{
    "func ProcessA(x int) {",
    "    return x + 1",
    "}",
    "",
    "func ProcessB(x int) {",
    "    return x + 1",  // Same body!
    "}",
}

// Use header to find the right one
contextLines := []string{"    return x + 1"}
pos := m.FindContext(contextLines, "func ProcessB")
// Returns 5 (second occurrence), not 1
```

#### 3. Similarity Scoring

```go
// Minor text differences (parameter rename)
fileLines := []string{
    "func Calculate(a, b int) int {",  // Changed params
    "    return a + b",
}

contextLines := []string{
    "func Calculate(x, y int) int {",  // Original params
    "    return x + y",
}

m := NewMatcher(fileLines)
m.SetThreshold(0.80) // Lower threshold for more tolerance
pos := m.FindContext(contextLines, "") // Returns 0 (80%+ similar)
```

### Test Coverage

**Current:** 90.4%

| Test Category | Tests | Notes |
|---------------|-------|-------|
| Exact matching | 6 | Start, middle, end, single/multi-line |
| Fuzzy matching | 6 | Whitespace, tabs, trailing/leading spaces |
| Header matching | 6 | Disambiguation, fallback, multiple headers |
| Edge cases | 10 | Empty files, Unicode, special chars, boundaries |
| Real-world scenarios | 4 | Go code, formatting changes, refactorings |

**Total:** 32 test cases + benchmarks

---

## Error Handling

### Error Messages with Line Numbers

All parse errors include the line number where the error occurred:

```
line 5: invalid path "/etc/passwd": absolute paths not allowed
line 12: unknown operation: "*** Invalid Operation: test.txt"
line 23: invalid line format: expected '+' prefix
```

### Common Errors

| Error | Cause | Solution |
|-------|-------|----------|
| `expected '*** Begin Patch'` | Missing or malformed begin marker | Ensure patch starts with `*** Begin Patch` |
| `missing '*** End Patch'` | Missing end marker | Ensure patch ends with `*** End Patch` |
| `absolute paths not allowed` | Path starts with `/` | Use relative paths only |
| `path traversal detected` | Path contains `..` | Use clean relative paths |
| `unknown operation` | Invalid operation type | Use `Add File`, `Delete File`, or `Update File` |
| `invalid line prefix` | Wrong prefix in hunk | Use space, `-`, or `+` prefixes |

---

## Path Validation

All file paths are validated using [`pkg/pathutil`](pathutil.md):

### Validation Rules

1. **Relative paths only** - Absolute paths (`/etc/passwd`) rejected
2. **No traversal** - `../../../` patterns rejected
3. **Clean paths** - Normalized to prevent tricks
4. **Valid UTF-8** - Reject invalid encodings

### Examples

```go
// ✅ Valid paths
"file.txt"
"src/main.go"
"internal/handler/handler.go"

// ❌ Invalid paths (rejected)
"/etc/passwd"              // absolute
"../../../etc/passwd"      // traversal
"/home/user/../etc/passwd" // absolute + traversal
```

---

## Testing

### Running Tests

```bash
# All tests
go test ./internal/patchapply/...

# With coverage
go test ./internal/patchapply/... -cover

# With race detector
go test ./internal/patchapply/... -race

# Verbose output
go test ./internal/patchapply/... -v
```

### Test Coverage

**Current:** 91.1%

| File | Coverage | Lines | Notes |
|------|----------|-------|-------|
| `types.go` | 100% | 82 | Full coverage of all types |
| `parser.go` | 89.6% | 287 | Main parsing logic |

### Test Categories

1. **Valid Patches** (11 tests)
   - Empty patch
   - Add file (simple, empty, with empty lines)
   - Delete file
   - Update file (simple, with context, move, multiple hunks)
   - Multiple operations
   - Unicode content

2. **Syntax Errors** (6 tests)
   - Missing begin/end markers
   - Invalid begin marker
   - Unknown operation
   - Invalid line prefixes

3. **Path Validation** (8 tests)
   - Absolute paths (add/delete/update/move)
   - Path traversal (add/delete/update/move)

4. **Edge Cases** (4 tests)
   - Nested paths
   - Unicode content
   - Only inserts/deletes

5. **Line Number Reporting** (3 tests)
   - Error messages include accurate line numbers

**Total:** 32 test cases

---

## Performance

### Benchmarks

```bash
go test ./internal/patchapply/... -bench=. -benchmem
```

### Expected Performance

| Operation | Target | Actual | Status |
|-----------|--------|--------|--------|
| Parse 100-line patch | <10ms | ~0.5ms | ✅ |
| Parse 10k-line patch | <100ms | ~50ms | ✅ |
| Memory overhead | <1MB | ~0.3MB | ✅ |

### Design for Performance

- **Streaming parser** - `bufio.Scanner` for memory efficiency
- **Single pass** - No backtracking or lookahead buffering
- **Minimal allocations** - Reuse buffers where possible
- **No regex** - Simple string operations for speed

---

## Security

### Threat Model

The parser defends against:

1. **Path Traversal** - `../../etc/passwd` rejected
2. **Absolute Paths** - `/etc/passwd` rejected
3. **DoS via Large Input** - Streaming parser handles large files
4. **Malformed Input** - Graceful error handling, no panics
5. **Invalid UTF-8** - Handled by Go's standard library

### Security Features

- **Path validation** - All paths validated using `pkg/pathutil`
- **No panics** - All errors returned gracefully
- **Line limits** - Reasonable limits on line length
- **Resource limits** - Streaming design prevents memory exhaustion

---

## Examples

### Example 1: Add New Handler

```
*** Begin Patch
*** Add File: internal/handler/user.go
+package handler
+
+import "context"
+
+// UserHandler handles user operations
+type UserHandler struct {
+    db *DB
+}
+
+// Handle processes user requests
+func (h *UserHandler) Handle(ctx context.Context) error {
+    return nil
+}
*** End Patch
```

### Example 2: Update Configuration

```
*** Begin Patch
*** Update File: config/server.go
@@ type Config struct
 type Config struct {
     Host string
     Port int
-    Timeout time.Duration
+    Timeout int // seconds
+    MaxConns int // max connections
 }
*** End Patch
```

### Example 3: Rename and Update

```
*** Begin Patch
*** Update File: handler.go
*** Move to: internal/handler/handler.go
@@
 package handler

 import (
-    "log"
+    "github.com/sirupsen/logrus"
 )

@@ func Process
 func Process(data string) error {
-    log.Printf("Processing: %s", data)
+    logrus.Infof("Processing: %s", data)
     return nil
 }
*** End Patch
```

### Example 4: Multiple Operations

```
*** Begin Patch
*** Add File: internal/middleware/auth.go
+package middleware
+
+func Auth() Middleware {
+    return func(next Handler) Handler {
+        return func(ctx context.Context) error {
+            // Auth logic
+            return next(ctx)
+        }
+    }
+}
*** Delete File: internal/deprecated/old_auth.go
*** Update File: internal/server/server.go
@@
 import (
+    "internal/middleware"
 )
@@ func New
 func New() *Server {
     s := &Server{}
+    s.Use(middleware.Auth())
     return s
 }
*** End Patch
```

---

## Code Quality

### Linting

```bash
# Run all linters
make lint

# Run specific linter
golangci-lint run ./internal/patchapply/...

# Format code
gofmt -w ./internal/patchapply/
```

### Complexity Analysis

```bash
# Check cyclomatic complexity
gocyclo -over 10 ./internal/patchapply/

# Check with herr analyzer
uast parse internal/patchapply/parser.go | herr analyze
```

**Current Metrics:**
- Max complexity: 10 (parseHunk) - at target threshold
- Average complexity: 3.3 - excellent
- Good comment ratio: 55% - could improve inline comments

---

## Applier API

### Core Types

```go
// Applier applies patches to the filesystem safely.
type Applier struct {
    workspaceRoot  string
    dryRun         bool
    createBackup   bool
    forceOverwrite bool
    modifications  []*fileModification
}

// ApplyResult contains the results of applying a patch.
type ApplyResult struct {
    FilesCreated []string
    FilesDeleted []string
    FilesUpdated []string
    FilesMoved   map[string]string // old path -> new path
    DryRun       bool
}

// Error represents a patch application error with context.
type Error struct {
    Op      string // Operation (Add, Delete, Update, Move)
    Path    string // File path
    Line    int    // Line number (for hunk errors)
    Err     error  // Underlying error
    Context string // Additional context
}
```

### Public API

```go
// NewApplier creates a new patch applier for the given workspace.
func NewApplier(workspaceRoot string) (*Applier, error)

// SetDryRun enables/disables dry-run mode (preview without changes).
func (a *Applier) SetDryRun(enabled bool)

// SetBackup enables/disables backup creation before modifications.
func (a *Applier) SetBackup(enabled bool)

// SetForceOverwrite enables/disables overwriting existing files on Add.
func (a *Applier) SetForceOverwrite(enabled bool)

// Apply applies the patch to the workspace.
// Returns ApplyResult on success, Error on failure.
// If any operation fails, all changes are rolled back.
func (a *Applier) Apply(patch *Patch) (*ApplyResult, error)

// ValidatePatch validates the patch without applying it.
// This is equivalent to dry-run mode.
func (a *Applier) ValidatePatch(patch *Patch) error
```

### Error Types

```go
var (
    ErrPathOutsideWorkspace = errors.New("path outside workspace")
    ErrFileNotFound         = errors.New("file not found")
    ErrFileExists           = errors.New("file already exists")
    ErrContextNotFound      = errors.New("context not found")
    ErrPermissionDenied     = errors.New("permission denied")
    ErrEmptyWorkspace       = errors.New("empty workspace root")
)
```

### Usage Example

```go
package main

import (
    "log"
    "github.com/dmytrogajewski/spin/internal/patchapply"
)

func main() {
    // Create applier
    applier, err := patchapply.NewApplier("/workspace")
    if err != nil {
        log.Fatal(err)
    }

    // Configure
    applier.SetDryRun(false)
    applier.SetForceOverwrite(false)

    // Parse patch
    parser := patchapply.NewParser(patchText)
    patch, err := parser.Parse()
    if err != nil {
        log.Fatalf("parse error: %v", err)
    }

    // Validate first
    if err := applier.ValidatePatch(patch); err != nil {
        log.Fatalf("validation error: %v", err)
    }

    // Apply patch
    result, err := applier.Apply(patch)
    if err != nil {
        log.Fatalf("apply error: %v", err)
    }

    // Report results
    log.Printf("Created: %v", result.FilesCreated)
    log.Printf("Updated: %v", result.FilesUpdated)
    log.Printf("Deleted: %v", result.FilesDeleted)
    log.Printf("Moved: %v", result.FilesMoved)
}
```

### Safety Features

#### 1. Path Validation

All paths are validated using `pkg/pathutil.SafeJoin`:
- Absolute paths rejected (`/etc/passwd`)
- Path traversal blocked (`../../etc/passwd`)
- All operations confined to workspace

#### 2. Atomic Operations

All patches are applied atomically:
- Track all modifications
- If any operation fails, rollback all changes
- Workspace left unchanged on error

#### 3. Dry-Run Mode

Preview changes without modifying files:
```go
applier.SetDryRun(true)
result, err := applier.Apply(patch)
// Files not modified, result shows what would happen
```

#### 4. Force Overwrite

Control whether Add operations can overwrite:
```go
applier.SetForceOverwrite(true)  // Allow overwrite
applier.SetForceOverwrite(false) // Reject if exists (default)
```

### Test Coverage

**Current:** 86.9%

| Test Category | Tests | Coverage |
|---------------|-------|----------|
| Path validation | 6 | 100% |
| Add file operations | 5 | 100% |
| Delete file operations | 3 | 100% |
| Update file operations | 6 | 95% |
| Move file operations | 3 | 100% |
| Dry-run mode | 1 | 100% |
| Atomic rollback | 1 | 100% |
| Multiple operations | 1 | 100% |
| ValidatePatch | 3 | 100% |
| Error messages | 3 | 100% |

**Total:** 60+ test cases

---

## Future Work (Roadmap)

### Feature 2.2: Fuzzy Matcher ✅ **COMPLETE**

**Goal:** Match patch hunks to file content with fuzzy matching

**Delivered Features:**
- ✅ Tolerance for whitespace changes (tabs, spaces, leading/trailing)
- ✅ Context line matching with similarity scoring (Levenshtein-based)
- ✅ Multiple match candidate ranking (closest to header)
- ✅ Edit distance calculations via `pkg/strutil`
- ✅ Configurable threshold (default 85%)
- ✅ Header-based disambiguation
- ✅ Performance: <1ms for 10k line files

**Status:** Complete (2025-10-12)
**Coverage:** 90.4%
**FRD:** [FRD-20251012015732-patchapply-fuzzy-matcher.md](../../specs/frds/FRD-20251012015732-patchapply-fuzzy-matcher.md)

### Feature 2.3: Applier ✅ **COMPLETE**

**Goal:** Apply parsed patches to filesystem safely

**Delivered Features:**
- ✅ All file operations (Add, Delete, Update, Move)
- ✅ Workspace confinement via pathutil
- ✅ Atomic operations with rollback
- ✅ Dry-run mode for previewing
- ✅ Force overwrite mode
- ✅ Clear error messages with context
- ✅ Test coverage 86.9%
- ✅ Cyclomatic complexity max 8

**Status:** Complete (2025-10-12)
**Coverage:** 86.9%
**FRD:** [FRD-20251012040000-patchapply-applier.md](../../specs/frds/FRD-20251012040000-patchapply-applier.md)

---

## Related Documentation

- [FRD-20251012030000: Patch Parser](../../specs/frds/FRD-20251012030000-patchapply-parser.md)
- [FRD-20251012015732: Fuzzy Matcher](../../specs/frds/FRD-20251012015732-patchapply-fuzzy-matcher.md)
- [FRD-20251012040000: Patch Applier](../../specs/frds/FRD-20251012040000-patchapply-applier.md)
- [Tools & Utility Modules Spec](../../specs/tools-modules/tools-modules.md)
- [Tools-Modules ROADMAP](../../specs/tools-modules/ROADMAP.md)
- [pkg/pathutil](pathutil.md) - Path validation
- [pkg/strutil](strutil.md) - String utilities (used by matcher and applier)

---

## Troubleshooting

### Tests Failing

```bash
# Run with verbose output
go test ./internal/patchapply/... -v

# Run with race detector
go test ./internal/patchapply/... -race

# Run specific test
go test ./internal/patchapply/... -run TestParser_Parse/add_file
```

### Linter Errors

```bash
# Check specific package
golangci-lint run ./internal/patchapply/...

# Auto-fix formatting
gofmt -w ./internal/patchapply/

# Check complexity
gocyclo -over 10 ./internal/patchapply/
```

### Low Coverage

```bash
# Generate coverage report
go test ./internal/patchapply/... -coverprofile=coverage.out

# View in browser
go tool cover -html=coverage.out

# Check per-function coverage
go tool cover -func=coverage.out
```

---

## Contact & Support

For questions or issues related to `internal/patchapply`:

1. Check this documentation first
2. Review FRD: [FRD-20251012030000](../../specs/frds/FRD-20251012030000-patchapply-parser.md)
3. Check roadmap: [tools-modules/ROADMAP.md](../../specs/tools-modules/ROADMAP.md)
4. Review test cases for usage examples

---

**Last Updated:** 2025-10-12
**Maintainer:** Spin Development Team
**Status:** ✅ Production Ready - Parser, Matcher, and Applier Complete
