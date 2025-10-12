# FRD-20251012040000: Patch Applier

**Feature:** Patch Applier - Safe Filesystem Operations
**Package:** `internal/patchapply`
**Priority:** P0 (Critical - brings everything together)
**Status:** Specification
**Created:** 2025-10-12
**Author:** Spin Agent
**Depends On:** FRD-20251012030000 (Parser), FRD-20251012015732 (Matcher)

---

## Overview

The Patch Applier is the final component of the `internal/patchapply` package that safely applies parsed patches to the filesystem. It brings together the parser (which converts patch text to AST) and the matcher (which finds hunk context) to perform actual file operations with robust safety guarantees.

### Problem Statement

AI-generated patches need to be applied to real files with these requirements:

1. **Safety First**: Prevent path traversal, validate all operations
2. **Atomic Operations**: All-or-nothing semantics (rollback on failure)
3. **Conflict Detection**: Detect when context cannot be found
4. **Dry-Run Support**: Preview changes without modifying files
5. **Clear Errors**: Detailed error messages for AI feedback
6. **Backup Support**: Optional backup before modification

### Goals

- ✅ Apply all patch operations (Add, Delete, Update, Move)
- ✅ Workspace confinement via `pkg/pathutil`
- ✅ Atomic application with rollback on failure
- ✅ Dry-run mode for previewing changes
- ✅ Optional backup creation before modification
- ✅ Clear error messages with context
- ✅ Test coverage ≥90%
- ✅ Cyclomatic complexity ≤15

---

## Requirements

### Functional Requirements

#### FR-1: File Operations
- **FR-1.1**: Support Add File operation (create new file with content)
- **FR-1.2**: Support Delete File operation (remove existing file)
- **FR-1.3**: Support Update File operation (apply hunks to existing file)
- **FR-1.4**: Support Move File operation (rename/move with optional content changes)

#### FR-2: Safety and Validation
- **FR-2.1**: Validate all paths using `pkg/pathutil.SafeJoin`
- **FR-2.2**: Ensure all operations confined to workspace root
- **FR-2.3**: Reject absolute paths and path traversal attempts
- **FR-2.4**: Validate file existence before Update/Delete
- **FR-2.5**: Prevent overwriting existing files on Add (unless force flag)

#### FR-3: Atomic Operations
- **FR-3.1**: Apply all operations atomically (all succeed or all fail)
- **FR-3.2**: Track all file modifications for rollback
- **FR-3.3**: Rollback all changes if any operation fails
- **FR-3.4**: Clean up temporary files on error

#### FR-4: Hunk Application
- **FR-4.1**: Use fuzzy matcher to find context for each hunk
- **FR-4.2**: Apply changes in correct order (delete then insert)
- **FR-4.3**: Preserve file line endings (CRLF vs LF)
- **FR-4.4**: Handle multiple hunks per file correctly
- **FR-4.5**: Validate context lines match before/after changes

#### FR-5: Operating Modes
- **FR-5.1**: Normal mode - apply changes to filesystem
- **FR-5.2**: Dry-run mode - validate without modifying files
- **FR-5.3**: Backup mode - create backups before modification
- **FR-5.4**: Force mode - overwrite existing files on Add

#### FR-6: Error Handling
- **FR-6.1**: Return detailed errors with file path and line number
- **FR-6.2**: Distinguish between:
  - Path validation errors
  - File not found errors
  - Context not found errors
  - Permission errors
  - IO errors
- **FR-6.3**: Include context in error messages (what failed and why)

### Non-Functional Requirements

#### NFR-1: Performance
- Apply typical patch (<100 lines) in <100ms
- Apply large patch (>1000 lines) in <1s
- Memory overhead <10MB for large files

#### NFR-2: Reliability
- Zero data loss - rollback must be bulletproof
- No partial modifications on failure
- Handle concurrent access gracefully (fail fast)

#### NFR-3: Quality
- Test coverage ≥90%
- Cyclomatic complexity ≤15 per function
- Zero linter errors
- Race detector clean

#### NFR-4: Maintainability
- Clear separation of concerns (validation, application, rollback)
- Well-documented error types
- Comprehensive godoc comments
- Integration test coverage

---

## Design

### Architecture

```
Applier
├── Validation Layer
│   ├── Path validation (SafeJoin, ValidateRelativePath)
│   ├── Operation validation (file existence, conflicts)
│   └── Pre-flight checks (dry-run)
│
├── Application Layer
│   ├── applyAddFile()      - Create new file
│   ├── applyDeleteFile()   - Remove file
│   ├── applyUpdateFile()   - Apply hunks
│   └── applyHunk()         - Apply single hunk using Matcher
│
└── Rollback Layer
    ├── Track modifications (original content, created files)
    ├── Restore on failure
    └── Clean up backups on success
```

### Core Types

```go
// Applier applies patches to the filesystem safely.
type Applier struct {
    workspaceRoot string
    dryRun        bool
    createBackup  bool
    forceOverwrite bool
    modifications map[string]*fileModification  // For rollback
}

// fileModification tracks a file change for rollback.
type fileModification struct {
    path         string
    operation    modOperation
    originalContent []byte  // For Update/Delete
    created      bool       // For Add
}

type modOperation int
const (
    opCreate modOperation = iota
    opUpdate
    opDelete
    opMove
)

// ApplyResult contains the results of applying a patch.
type ApplyResult struct {
    FilesCreated []string
    FilesDeleted []string
    FilesUpdated []string
    FilesMoved   map[string]string  // old -> new
    DryRun       bool
}

// Error types
type Error struct {
    Op      string  // Operation (Add, Delete, Update)
    Path    string  // File path
    Line    int     // Line number (for hunk errors)
    Err     error   // Underlying error
    Context string  // Additional context
}

var (
    ErrPathOutsideWorkspace = errors.New("path outside workspace")
    ErrFileNotFound         = errors.New("file not found")
    ErrFileExists           = errors.New("file already exists")
    ErrContextNotFound      = errors.New("context not found")
    ErrPermissionDenied     = errors.New("permission denied")
)
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
// This is equivalent to SetDryRun(true) + Apply().
func (a *Applier) ValidatePatch(patch *Patch) error
```

### Implementation Flow

#### 1. Pre-flight Validation

```go
func (a *Applier) Apply(patch *Patch) (*ApplyResult, error) {
    // Phase 1: Validate all operations
    for _, op := range patch.Operations {
        if err := a.validateOperation(op); err != nil {
            return nil, err
        }
    }

    if a.dryRun {
        return &ApplyResult{DryRun: true}, nil
    }

    // Phase 2: Apply operations
    // ...
}
```

#### 2. Apply Operations (with tracking)

```go
func (a *Applier) applyOperations(patch *Patch) (*ApplyResult, error) {
    result := &ApplyResult{}

    for _, op := range patch.Operations {
        switch op := op.(type) {
        case *AddFile:
            if err := a.applyAddFile(op, result); err != nil {
                a.rollback()
                return nil, err
            }
        case *DeleteFile:
            if err := a.applyDeleteFile(op, result); err != nil {
                a.rollback()
                return nil, err
            }
        case *UpdateFile:
            if err := a.applyUpdateFile(op, result); err != nil {
                a.rollback()
                return nil, err
            }
        }
    }

    return result, nil
}
```

#### 3. Add File Operation

```go
func (a *Applier) applyAddFile(op *AddFile, result *ApplyResult) error {
    // Validate and resolve path
    fullPath, err := pathutil.SafeJoin(a.workspaceRoot, op.FilePath)
    if err != nil {
        return &Error{Op: "Add", Path: op.FilePath, Err: err}
    }

    // Check if file exists
    if _, err := os.Stat(fullPath); err == nil {
        if !a.forceOverwrite {
            return &Error{
                Op: "Add",
                Path: op.FilePath,
                Err: ErrFileExists,
                Context: "use force mode to overwrite",
            }
        }
    }

    // Create parent directories
    if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
        return &Error{Op: "Add", Path: op.FilePath, Err: err}
    }

    // Write file content
    content := strings.Join(op.Lines, "\n")
    if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
        return &Error{Op: "Add", Path: op.FilePath, Err: err}
    }

    // Track for rollback
    a.trackModification(op.FilePath, opCreate, nil)

    result.FilesCreated = append(result.FilesCreated, op.FilePath)
    return nil
}
```

#### 4. Delete File Operation

```go
func (a *Applier) applyDeleteFile(op *DeleteFile, result *ApplyResult) error {
    fullPath, err := pathutil.SafeJoin(a.workspaceRoot, op.FilePath)
    if err != nil {
        return &Error{Op: "Delete", Path: op.FilePath, Err: err}
    }

    // Read original content for rollback
    originalContent, err := os.ReadFile(fullPath)
    if err != nil {
        if os.IsNotExist(err) {
            return &Error{
                Op: "Delete",
                Path: op.FilePath,
                Err: ErrFileNotFound,
            }
        }
        return &Error{Op: "Delete", Path: op.FilePath, Err: err}
    }

    // Delete file
    if err := os.Remove(fullPath); err != nil {
        return &Error{Op: "Delete", Path: op.FilePath, Err: err}
    }

    // Track for rollback
    a.trackModification(op.FilePath, opDelete, originalContent)

    result.FilesDeleted = append(result.FilesDeleted, op.FilePath)
    return nil
}
```

#### 5. Update File Operation (with hunks)

```go
func (a *Applier) applyUpdateFile(op *UpdateFile, result *ApplyResult) error {
    fullPath, err := pathutil.SafeJoin(a.workspaceRoot, op.FilePath)
    if err != nil {
        return &Error{Op: "Update", Path: op.FilePath, Err: err}
    }

    // Read file content
    originalContent, err := os.ReadFile(fullPath)
    if err != nil {
        if os.IsNotExist(err) {
            return &Error{
                Op: "Update",
                Path: op.FilePath,
                Err: ErrFileNotFound,
                Context: "file must exist for update",
            }
        }
        return &Error{Op: "Update", Path: op.FilePath, Err: err}
    }

    // Parse lines
    lines := strutil.SplitLines(string(originalContent))

    // Apply each hunk
    for i, hunk := range op.Hunks {
        if err := a.applyHunk(lines, hunk, op.FilePath, i); err != nil {
            return err
        }
    }

    // Handle rename/move
    targetPath := fullPath
    if op.NewPath != "" {
        targetPath, err = pathutil.SafeJoin(a.workspaceRoot, op.NewPath)
        if err != nil {
            return &Error{Op: "Move", Path: op.NewPath, Err: err}
        }

        // Ensure parent directories exist
        if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
            return &Error{Op: "Move", Path: op.NewPath, Err: err}
        }

        result.FilesMoved[op.FilePath] = op.NewPath
    }

    // Write modified content
    newContent := strutil.JoinLines(lines)
    if err := os.WriteFile(targetPath, []byte(newContent), 0644); err != nil {
        return &Error{Op: "Update", Path: op.FilePath, Err: err}
    }

    // If moved, delete original
    if op.NewPath != "" && targetPath != fullPath {
        if err := os.Remove(fullPath); err != nil {
            return &Error{Op: "Move", Path: op.FilePath, Err: err}
        }
    }

    // Track for rollback
    a.trackModification(op.FilePath, opUpdate, originalContent)

    result.FilesUpdated = append(result.FilesUpdated, op.FilePath)
    return nil
}
```

#### 6. Apply Hunk (using Matcher)

```go
func (a *Applier) applyHunk(
    lines []string,
    hunk Hunk,
    filePath string,
    hunkIdx int,
) error {
    // Extract context lines from hunk
    contextLines := a.extractContextLines(hunk)
    if len(contextLines) == 0 {
        return &Error{
            Op: "Update",
            Path: filePath,
            Line: hunkIdx,
            Err: errors.New("no context lines in hunk"),
            Context: "hunks must have context lines for matching",
        }
    }

    // Find context in file using fuzzy matcher
    matcher := NewMatcher(lines)
    pos := matcher.FindContext(contextLines, hunk.Header)
    if pos < 0 {
        return &Error{
            Op: "Update",
            Path: filePath,
            Line: hunkIdx,
            Err: ErrContextNotFound,
            Context: fmt.Sprintf(
                "could not find context for hunk %d (header: %q)",
                hunkIdx,
                hunk.Header,
            ),
        }
    }

    // Apply changes at found position
    newLines := make([]string, 0, len(lines))
    newLines = append(newLines, lines[:pos]...)  // Before context

    offset := 0
    for _, change := range hunk.Changes {
        switch change.Type {
        case LineContext:
            // Verify context matches
            if offset >= len(lines) ||
               !a.linesMatch(lines[pos+offset], change.Text) {
                return &Error{
                    Op: "Update",
                    Path: filePath,
                    Line: hunkIdx,
                    Err: errors.New("context mismatch"),
                    Context: fmt.Sprintf(
                        "expected %q, got %q",
                        change.Text,
                        lines[pos+offset],
                    ),
                }
            }
            newLines = append(newLines, change.Text)
            offset++

        case LineDelete:
            // Skip this line (delete)
            offset++

        case LineInsert:
            // Add this line
            newLines = append(newLines, change.Text)
        }
    }

    // Add remaining lines after hunk
    newLines = append(newLines, lines[pos+offset:]...)

    // Replace lines slice content
    copy(lines, newLines)
    return nil
}
```

#### 7. Rollback on Failure

```go
func (a *Applier) rollback() error {
    // Reverse order of modifications
    for i := len(a.modifications) - 1; i >= 0; i-- {
        mod := a.modifications[i]
        fullPath, _ := pathutil.SafeJoin(a.workspaceRoot, mod.path)

        switch mod.operation {
        case opCreate:
            // Remove created file
            os.Remove(fullPath)

        case opUpdate:
            // Restore original content
            os.WriteFile(fullPath, mod.originalContent, 0644)

        case opDelete:
            // Recreate deleted file
            os.WriteFile(fullPath, mod.originalContent, 0644)

        case opMove:
            // Move back to original location
            // (handled via Update rollback)
        }
    }

    return nil
}
```

---

## Test Plan

### Unit Tests

#### 1. Path Validation Tests
```go
func TestApplier_ValidatePaths(t *testing.T) {
    tests := []struct {
        name    string
        path    string
        wantErr error
    }{
        {"valid relative", "src/main.go", nil},
        {"absolute path", "/etc/passwd", ErrPathOutsideWorkspace},
        {"traversal", "../../../etc/passwd", ErrPathOutsideWorkspace},
    }
    // ...
}
```

#### 2. Add File Tests
```go
func TestApplier_AddFile(t *testing.T) {
    tests := []struct {
        name       string
        filePath   string
        content    []string
        existing   bool
        force      bool
        wantErr    error
    }{
        {"new file", "test.txt", []string{"line1"}, false, false, nil},
        {"exists no force", "test.txt", []string{"line1"}, true, false, ErrFileExists},
        {"exists with force", "test.txt", []string{"line1"}, true, true, nil},
    }
    // ...
}
```

#### 3. Delete File Tests
```go
func TestApplier_DeleteFile(t *testing.T) {
    tests := []struct {
        name     string
        filePath string
        exists   bool
        wantErr  error
    }{
        {"existing file", "test.txt", true, nil},
        {"missing file", "missing.txt", false, ErrFileNotFound},
    }
    // ...
}
```

#### 4. Update File Tests
```go
func TestApplier_UpdateFile(t *testing.T) {
    tests := []struct {
        name        string
        original    string
        hunks       []Hunk
        expected    string
        wantErr     error
    }{
        {"simple update", "a\nb\nc", hunks, "a\nX\nc", nil},
        {"context not found", "a\nb\nc", badHunks, "", ErrContextNotFound},
    }
    // ...
}
```

#### 5. Hunk Application Tests
```go
func TestApplier_ApplyHunk(t *testing.T) {
    tests := []struct {
        name     string
        lines    []string
        hunk     Hunk
        expected []string
        wantErr  error
    }{
        {"insert line", []string{"a", "b"}, insertHunk, []string{"a", "X", "b"}, nil},
        {"delete line", []string{"a", "b", "c"}, deleteHunk, []string{"a", "c"}, nil},
        {"replace line", []string{"a", "b", "c"}, replaceHunk, []string{"a", "X", "c"}, nil},
    }
    // ...
}
```

#### 6. Rollback Tests
```go
func TestApplier_Rollback(t *testing.T) {
    tests := []struct {
        name       string
        operations []FileOperation
        failAt     int  // Which operation to fail
        verify     func(t *testing.T, workspace string)
    }{
        {"rollback add", addOps, 1, verifyNoFiles},
        {"rollback update", updateOps, 0, verifyOriginalContent},
    }
    // ...
}
```

### Integration Tests

#### 1. Complete Patch Application
```go
func TestApplier_Integration_CompletePatch(t *testing.T) {
    // Create test workspace
    // Apply patch with Add, Update, Delete, Move
    // Verify all changes applied correctly
    // Verify result summary
}
```

#### 2. Dry-Run Mode
```go
func TestApplier_Integration_DryRun(t *testing.T) {
    // Create test workspace
    // Apply patch in dry-run mode
    // Verify no files changed
    // Verify result summary correct
}
```

#### 3. Backup and Restore
```go
func TestApplier_Integration_Backup(t *testing.T) {
    // Create test workspace
    // Enable backup mode
    // Apply patch
    // Verify backups created
    // Force failure and verify restore
}
```

#### 4. Atomic Failure
```go
func TestApplier_Integration_AtomicFailure(t *testing.T) {
    // Apply patch with multiple operations
    // Force failure on 3rd operation
    // Verify all changes rolled back
    // Verify workspace unchanged
}
```

### Performance Tests

```go
func BenchmarkApplier_SmallPatch(b *testing.B) {
    // Benchmark <100 line patch
    // Target: <100ms
}

func BenchmarkApplier_LargePatch(b *testing.B) {
    // Benchmark >1000 line patch
    // Target: <1s
}
```

---

## Acceptance Criteria

### Must Have
- ✅ All file operations (Add, Delete, Update, Move) work correctly
- ✅ Path validation prevents traversal and escapes
- ✅ Atomic operations with rollback on failure
- ✅ Dry-run mode works without side effects
- ✅ Context not found errors are clear and actionable
- ✅ Test coverage ≥90%
- ✅ All tests pass with `-race`
- ✅ `make lint` passes (zero errors)
- ✅ Cyclomatic complexity ≤15

### Should Have
- ✅ Backup mode creates recoverable backups
- ✅ Performance: <100ms for typical patches
- ✅ Memory efficient for large files
- ✅ Clear error messages with file/line context

### Nice to Have
- ⭕ Progress reporting for large patches
- ⭕ Conflict resolution strategies
- ⭕ Partial application mode (continue on error)

---

## Dependencies

### Required Packages
- `pkg/pathutil` - Path validation and security
- `pkg/strutil` - Line operations and whitespace handling
- `internal/patchapply/parser.go` - Patch parsing
- `internal/patchapply/matcher.go` - Fuzzy context matching

### Standard Library
- `os` - File operations
- `io/ioutil` - File reading
- `path/filepath` - Path manipulation
- `fmt` - Error formatting
- `errors` - Error wrapping

---

## Risks and Mitigations

### Risk 1: Data Loss on Rollback Failure
**Impact:** High
**Probability:** Low
**Mitigation:**
- Test rollback extensively
- Keep modifications list ordered
- Use atomic file operations where possible

### Risk 2: Context Not Found (False Negatives)
**Impact:** Medium
**Probability:** Medium
**Mitigation:**
- Use fuzzy matcher with configurable threshold
- Provide clear error messages with context
- Allow AI to retry with more context

### Risk 3: Concurrent Modifications
**Impact:** Medium
**Probability:** Low
**Mitigation:**
- Fail fast on file lock errors
- Document that applier is not thread-safe
- Add file locking in future version if needed

### Risk 4: Performance on Large Files
**Impact:** Low
**Probability:** Low
**Mitigation:**
- Benchmark with large files
- Use efficient line slicing
- Consider streaming for very large files

---

## Future Enhancements

### V2: Advanced Features
- File locking for concurrent safety
- Incremental backup (only changed sections)
- Conflict resolution strategies
- Progress reporting with callbacks
- Partial application mode

### V3: Performance Optimizations
- Memory-mapped file I/O for large files
- Parallel application of independent operations
- Incremental hunk matching (cache context positions)

---

## References

1. [Tools & Modules Spec](../tools-modules/tools-modules.md)
2. [ROADMAP Phase 2.3](../tools-modules/ROADMAP.md#feature-23-patch-applier)
3. [FRD-20251012030000: Parser](FRD-20251012030000-patchapply-parser.md)
4. [FRD-20251012015732: Fuzzy Matcher](FRD-20251012015732-patchapply-fuzzy-matcher.md)
5. [pkg/pathutil](../../docs/packages/pathutil.md)
6. [pkg/strutil](../../docs/packages/strutil.md)

---

**Status:** Ready for Implementation
**Next Steps:** Write tests, implement, analyze, lint, iterate
