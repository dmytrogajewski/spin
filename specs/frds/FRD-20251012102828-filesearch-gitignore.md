# FRD-20251012102828: File Search Gitignore Handler

**Feature:** Enhanced File Search with Gitignore Support
**Package:** `internal/filesearch`
**Priority:** P1 (Essential for production use)
**Status:** Planning
**Created:** 2025-10-12
**Roadmap:** Phase 3, Feature 3.1

---

## 1. Overview

### 1.1 Purpose

Add comprehensive `.gitignore` and `.spinignore` pattern support to `internal/filesearch` to prevent scanning and returning files that should be ignored. This is essential for production use as it dramatically reduces noise in search results and improves performance by skipping large directories like `node_modules`, `.git`, `vendor`, etc.

### 1.2 Background

Currently, `internal/filesearch/scanner.go` only has basic hardcoded `.git` directory ignore support. Real-world projects use `.gitignore` files with complex patterns to exclude build artifacts, dependencies, IDE files, and other generated content. Spin needs to respect these patterns to provide relevant search results.

### 1.3 Goals

- ✅ Load and parse `.gitignore` files
- ✅ Load and parse `.spinignore` files (Spin-specific ignores)
- ✅ Support standard gitignore pattern syntax via `doublestar` library
- ✅ Support directory-specific ignore patterns
- ✅ Include sensible default ignore patterns
- ✅ Integrate seamlessly with existing `Scanner`
- ✅ Maintain performance (minimal overhead on large repos)

### 1.4 Non-Goals

- ❌ Full git semantics (no negation patterns `!` in initial version)
- ❌ `.gitignore` pattern caching across scanner instances
- ❌ Watching `.gitignore` files for changes
- ❌ Support for `.git/info/exclude`

---

## 2. Requirements

### 2.1 Functional Requirements

**FR-1: Pattern Loading**
- MUST load `.gitignore` from repository root
- MUST load `.spinignore` from repository root
- MUST handle missing files gracefully (no error)
- MUST skip lines starting with `#` (comments)
- MUST skip empty lines
- MUST trim whitespace from patterns

**FR-2: Pattern Matching**
- MUST support glob patterns via `github.com/bmatcuk/doublestar/v4`
  - `*.log` - match all `.log` files
  - `build/` - match `build` directory
  - `**/temp` - match `temp` anywhere
  - `node_modules/**` - match everything in `node_modules`
- MUST handle both file and directory paths
- MUST support patterns ending with `/` for directory-only matching
- MUST match patterns case-sensitively (Git default)

**FR-3: Default Patterns**
- MUST include default ignore patterns:
  - `.git/**` - Git internal files
  - `node_modules/**` - Node.js dependencies
  - `.spin/**` - Spin internal directory
  - `vendor/**` - Go vendor directory (common)
  - `__pycache__/**` - Python cache
  - `.vscode/**` - VS Code settings
  - `.idea/**` - JetBrains IDEs

**FR-4: Integration with Scanner**
- MUST integrate `IgnoreHandler` into `Scanner`
- MUST check ignore patterns during `filepath.WalkDir` traversal
- MUST return `filepath.SkipDir` for ignored directories
- MUST skip ignored files without returning them
- MUST maintain backward compatibility with existing `Scanner` API

**FR-5: Performance**
- MUST have minimal overhead (<10ms for 1000 patterns on 10k files)
- MUST short-circuit pattern matching when possible
- MUST NOT significantly slow down existing `Scanner.Scan()`

### 2.2 Non-Functional Requirements

**NFR-1: Code Quality**
- Test coverage ≥90%
- Cyclomatic complexity ≤15 per function
- Zero `make lint` errors
- Race detector clean (`go test -race`)

**NFR-2: Documentation**
- Godoc comments on all exported types
- Package documentation updated
- Usage examples in tests
- Integration documented in `docs/packages/filesearch.md`

**NFR-3: Maintainability**
- Clean separation of concerns (IgnoreHandler separate from Scanner)
- Clear error messages
- Testable design (dependency injection)

---

## 3. Design

### 3.1 Architecture

```
internal/filesearch/
├── scanner.go        # Existing scanner (enhanced)
├── matcher.go        # Existing matcher (unchanged)
├── ignore.go         # NEW: IgnoreHandler implementation
├── ignore_test.go    # NEW: IgnoreHandler tests
├── scanner_test.go   # Updated with ignore tests
└── doc.go            # Updated package doc
```

### 3.2 Data Structures

```go
// IgnoreHandler handles .gitignore and .spinignore patterns
type IgnoreHandler struct {
    patterns      []string // All loaded patterns
    rootDir       string   // Root directory for pattern resolution
}
```

### 3.3 API Design

#### New Type: IgnoreHandler

```go
package filesearch

import (
    "bufio"
    "os"
    "path/filepath"
    "strings"

    "github.com/bmatcuk/doublestar/v4"
)

// IgnoreHandler handles .gitignore and .spinignore pattern matching
type IgnoreHandler struct {
    patterns []string
    rootDir  string
}

// NewIgnoreHandler creates a new ignore handler for the given root directory.
// Loads .gitignore and .spinignore if they exist, plus default patterns.
// Returns error only on critical failures (file exists but unreadable).
func NewIgnoreHandler(rootDir string) (*IgnoreHandler, error)

// IsIgnored checks if a relative path should be ignored.
// The path parameter should be relative to rootDir.
// The isDir parameter indicates if the path is a directory.
func (h *IgnoreHandler) IsIgnored(relPath string, isDir bool) bool

// loadIgnoreFile loads patterns from an ignore file.
// Returns error only if file exists but cannot be read.
// Missing files are not an error.
func (h *IgnoreHandler) loadIgnoreFile(path string) error

// defaultPatterns returns the default ignore patterns.
func defaultPatterns() []string
```

#### Updated Type: Scanner

```go
// Scanner scans directories for files with gitignore support
type Scanner struct {
    baseDir      string
    ignoreGit    bool          // Deprecated: use IgnoreHandler instead
    maxDepth     int
    ignoreHandler *IgnoreHandler // NEW: gitignore support
}

// NewScanner creates a new file scanner with gitignore support.
// If ignoreGit is true, basic .git exclusion is enabled (backward compatible).
// For full gitignore support, the Scanner will auto-create an IgnoreHandler.
func NewScanner(baseDir string, ignoreGit bool) *Scanner

// NewScannerWithIgnore creates a scanner with a custom IgnoreHandler.
// This allows advanced configuration and testing.
func NewScannerWithIgnore(baseDir string, handler *IgnoreHandler) *Scanner
```

### 3.4 Implementation Details

**Pattern Matching Logic:**
```go
func (h *IgnoreHandler) IsIgnored(relPath string, isDir bool) bool {
    for _, pattern := range h.patterns {
        // Try exact match
        matched, err := doublestar.Match(pattern, relPath)
        if err == nil && matched {
            return true
        }

        // For directories, also check with trailing slash
        if isDir {
            matched, err = doublestar.Match(pattern, relPath+"/")
            if err == nil && matched {
                return true
            }
        }
    }
    return false
}
```

**Scanner Integration:**
```go
func (s *Scanner) Scan() ([]string, error) {
    var files []string

    // Auto-create IgnoreHandler if not provided
    if s.ignoreHandler == nil && s.baseDir != "" {
        handler, _ := NewIgnoreHandler(s.baseDir)
        s.ignoreHandler = handler
    }

    err := filepath.WalkDir(s.baseDir, func(path string, d os.DirEntry, err error) error {
        if err != nil {
            return nil // Skip errors
        }

        // Get relative path
        relPath, err := filepath.Rel(s.baseDir, path)
        if err != nil {
            return nil
        }

        // Convert to forward slashes for consistent matching
        relPath = filepath.ToSlash(relPath)

        // Check if ignored
        if s.ignoreHandler != nil && s.ignoreHandler.IsIgnored(relPath, d.IsDir()) {
            if d.IsDir() {
                return filepath.SkipDir
            }
            return nil
        }

        // Legacy ignoreGit support (for backward compatibility)
        if d.IsDir() {
            if s.ignoreGit && d.Name() == ".git" {
                return filepath.SkipDir
            }
            return nil
        }

        files = append(files, relPath)
        return nil
    })

    return files, err
}
```

**Default Patterns:**
```go
func defaultPatterns() []string {
    return []string{
        ".git/**",
        "node_modules/**",
        ".spin/**",
        "vendor/**",
        "__pycache__/**",
        ".vscode/**",
        ".idea/**",
        "*.pyc",
        "*.pyo",
        ".DS_Store",
        "Thumbs.db",
    }
}
```

---

## 4. Testing Strategy

### 4.1 Test Cases

#### Unit Tests for IgnoreHandler

1. **Basic Pattern Loading**
   - Load .gitignore with simple patterns
   - Load .spinignore with patterns
   - Handle missing files gracefully
   - Skip comments and empty lines

2. **Pattern Matching**
   - `*.log` matches `debug.log` but not `log.txt`
   - `build/` matches `build` directory
   - `node_modules/**` matches all files in `node_modules`
   - `**/temp` matches `temp` at any depth

3. **Directory Matching**
   - `dist/` matches only directories, not `dist.txt`
   - Trailing slash patterns work correctly

4. **Default Patterns**
   - `.git/**` excludes git internal files
   - `node_modules/**` excludes npm dependencies

5. **Edge Cases**
   - Empty .gitignore file
   - .gitignore with only comments
   - Malformed patterns (gracefully skip)
   - Very long pattern lists (1000+ patterns)

#### Integration Tests with Scanner

1. **Basic Integration**
   - Scanner uses IgnoreHandler automatically
   - Ignored files not returned in results
   - Ignored directories not traversed

2. **Real-World Scenarios**
   - Node.js project with `node_modules`
   - Go project with `vendor` and `.git`
   - Python project with `__pycache__`

3. **Performance Tests**
   - 10k files with 100 patterns: <100ms
   - 1k files with 1000 patterns: <200ms

4. **Backward Compatibility**
   - Existing `NewScanner(dir, true)` still works
   - `ignoreGit` flag still respected

### 4.2 Test Coverage Goals

- **IgnoreHandler:** ≥90% coverage
  - `NewIgnoreHandler`: All paths covered
  - `IsIgnored`: All matching logic paths
  - `loadIgnoreFile`: Error and success paths
- **Scanner Integration:** ≥85% coverage
  - Ignore checks during traversal
  - SkipDir handling
  - Backward compatibility paths

### 4.3 Benchmark Tests

```go
func BenchmarkIgnoreHandler_IsIgnored_10Patterns(b *testing.B)
func BenchmarkIgnoreHandler_IsIgnored_100Patterns(b *testing.B)
func BenchmarkIgnoreHandler_IsIgnored_1000Patterns(b *testing.B)
func BenchmarkScanner_Scan_WithIgnore_10kFiles(b *testing.B)
```

---

## 5. Implementation Plan

### 5.1 Step 1: Create IgnoreHandler (2-3 hours)

**Files:**
- `internal/filesearch/ignore.go`
- `internal/filesearch/ignore_test.go`

**Tasks:**
1. Define `IgnoreHandler` struct
2. Implement `NewIgnoreHandler`
3. Implement `loadIgnoreFile`
4. Implement `IsIgnored` with doublestar matching
5. Implement `defaultPatterns`
6. Write unit tests (20+ test cases)

### 5.2 Step 2: Integrate with Scanner (1-2 hours)

**Files:**
- `internal/filesearch/scanner.go` (modify)
- `internal/filesearch/scanner_test.go` (add tests)

**Tasks:**
1. Add `ignoreHandler *IgnoreHandler` field to Scanner
2. Create `NewScannerWithIgnore` constructor
3. Update `Scan()` to use IgnoreHandler
4. Add auto-creation logic for backward compatibility
5. Write integration tests (10+ test cases)

### 5.3 Step 3: Testing and Benchmarking (1-2 hours)

**Tasks:**
1. Run all tests: `go test -v ./internal/filesearch/...`
2. Check coverage: `go test -cover ./internal/filesearch/...`
3. Run benchmarks: `go test -bench=. ./internal/filesearch/...`
4. Run race detector: `go test -race ./internal/filesearch/...`
5. Validate performance meets goals

### 5.4 Step 4: Code Quality (1 hour)

**Tasks:**
1. Run `make lint` - fix all errors
2. Run `uast parse` and `herr analyze` - fix findings
3. Ensure complexity ≤15
4. Add/improve godoc comments
5. Update `doc.go` package documentation

### 5.5 Step 5: Documentation (30 min)

**Tasks:**
1. Create `docs/packages/filesearch.md` (if not exists)
2. Document IgnoreHandler API
3. Add usage examples
4. Update README if needed

---

## 6. Acceptance Criteria

### 6.1 Functionality

- ✅ `NewIgnoreHandler("/project")` loads `.gitignore` and `.spinignore`
- ✅ `IsIgnored("node_modules/pkg/file.js", false)` returns `true`
- ✅ `IsIgnored("src/main.go", false)` returns `false`
- ✅ Default patterns exclude `.git`, `node_modules`, `.spin`, `vendor`, `__pycache__`
- ✅ `Scanner.Scan()` does not return ignored files
- ✅ `Scanner.Scan()` does not traverse ignored directories

### 6.2 Quality

- ✅ Test coverage ≥90% for IgnoreHandler
- ✅ Test coverage ≥85% for Scanner integration
- ✅ All tests passing: `go test ./internal/filesearch/...`
- ✅ Race detector clean: `go test -race ./internal/filesearch/...`
- ✅ `make lint` passes with zero errors
- ✅ Cyclomatic complexity ≤15 per function
- ✅ Godoc coverage 100% for exported types

### 6.3 Performance

- ✅ `IsIgnored` with 100 patterns: <1μs per call
- ✅ `Scanner.Scan()` with ignore on 10k file repo: <100ms
- ✅ Memory overhead <1MB for typical ignore file

### 6.4 Backward Compatibility

- ✅ Existing `NewScanner(dir, true)` calls work unchanged
- ✅ `ignoreGit` flag still respected (even if deprecated)
- ✅ No breaking changes to `Scanner` public API

---

## 7. Risks and Mitigations

### 7.1 Risk: Pattern Performance

**Risk:** Complex glob patterns may slow down traversal on large repos.

**Mitigation:**
- Use efficient `doublestar` library (well-tested, optimized)
- Benchmark early and optimize hot paths
- Short-circuit on first match
- Document performance characteristics

### 7.2 Risk: Pattern Complexity

**Risk:** Gitignore syntax is complex, may have edge cases.

**Mitigation:**
- Start with most common patterns (glob, directory patterns)
- Defer complex features (negation `!`) to future version
- Comprehensive test suite with real-world patterns
- Clear documentation of supported patterns

### 7.3 Risk: Backward Compatibility

**Risk:** Changes to Scanner might break existing code.

**Mitigation:**
- Add `ignoreHandler` as optional field
- Keep `ignoreGit` flag working
- Auto-create IgnoreHandler internally
- Add `NewScannerWithIgnore` for explicit control
- Extensive integration tests

---

## 8. Future Enhancements

**Post-MVP Features:**
- Negation patterns (`!important.log`)
- Directory-specific `.gitignore` files (nested)
- `.git/info/exclude` support
- Custom pattern sources (not just files)
- Pattern compilation/caching for performance
- Incremental ignore list updates
- Support for `.ignore` files from other tools

---

## 9. References

**External Documentation:**
- [Git Gitignore Documentation](https://git-scm.com/docs/gitignore)
- [doublestar Library](https://github.com/bmatcuk/doublestar)

**Internal Documentation:**
- [specs/tools-modules/ROADMAP.md](../tools-modules/ROADMAP.md) - Phase 3, Feature 3.1
- [specs/tools-modules/tools-modules.md](../tools-modules/tools-modules.md) - Package 2: internal/filesearch
- [docs/packages/README.md](../../docs/packages/README.md) - Package index

**Related FRDs:**
- None (first FRD for filesearch)

---

## 10. Approval

**Status:** Draft
**Author:** Spin AI Agent
**Reviewers:** TBD
**Approval Date:** TBD

---

**Last Updated:** 2025-10-12
**Version:** 1.0
