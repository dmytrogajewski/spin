# filesearch - File Scanning and Fuzzy Matching with Advanced Ranking

**Package:** `internal/filesearch`
**Status:** ✅ Stable
**Test Coverage:** 92.5%
**Complexity:** Low (max: 5)

---

## Overview

The `filesearch` package provides fast file discovery and intelligent fuzzy matching capabilities for the Spin TUI file picker. It includes comprehensive `.gitignore` and `.spinignore` support, advanced 7-tier ranking algorithm, and asynchronous indexing with context cancellation.

**Key Features:**
- Recursive directory scanning with ignore pattern support
- **NEW:** Advanced 7-tier scoring algorithm for intelligent ranking
- **NEW:** Async file indexing with context cancellation
- **NEW:** High-level Searcher API for easy integration
- Gitignore pattern matching (doublestar glob syntax)
- Fuzzy file path matching with position-aware scoring
- Default ignore patterns for common files (.git, node_modules, vendor, etc.)
- High performance (10k files in ~12ms with ignore rules)
- Thread-safe concurrent search operation

---

## Components

### Scanner

Recursively scans directories and returns file paths while respecting ignore patterns.

```go
type Scanner struct {
    baseDir       string
    ignoreGit     bool          // Deprecated: use IgnoreHandler instead
    maxDepth      int
    ignoreHandler *IgnoreHandler
}
```

**Constructor:**
```go
scanner := filesearch.NewScanner("/workspace", false)
files, err := scanner.Scan()
```

The Scanner automatically creates an `IgnoreHandler` that loads:
- `.gitignore` from workspace root
- `.spinignore` from workspace root
- Default ignore patterns

### IgnoreHandler

Handles gitignore-style pattern matching to determine if files should be excluded.

```go
type IgnoreHandler struct {
    patterns []string
    rootDir  string
}
```

**Constructor:**
```go
handler, err := filesearch.NewIgnoreHandler("/workspace")
isIgnored := handler.IsIgnored("node_modules/pkg/index.js", false)
// Returns: true
```

**Default Ignore Patterns:**
- `.git/**` - Git internal files
- `.gitignore` - Gitignore file itself
- `.spinignore` - Spinignore file itself
- `node_modules/**` - Node.js dependencies
- `.spin/**` - Spin internal directory
- `vendor/**` - Go vendor directory
- `__pycache__/**` - Python cache
- `.vscode/**`, `.idea/**` - IDE settings
- `*.pyc`, `*.pyo` - Python bytecode
- `.DS_Store`, `Thumbs.db` - OS-specific files

### Matcher (Enhanced with Advanced Scoring)

Provides fuzzy matching for file paths with **advanced 7-tier intelligent ranking**.

```go
type Matcher struct {
    caseSensitive bool
}

type Match struct {
    Path    string
    Score   int      // Higher is better
    Indices []int    // Matched character positions
}
```

**Constructor:**
```go
matcher := filesearch.NewMatcher(false) // case-insensitive (recommended)
matches := matcher.Match("test", files)
```

**Advanced Scoring Algorithm (7-tier):**

| Priority | Match Type | Score | Example |
|----------|------------|-------|---------|
| 1 | Exact filename | 100 | `test` → `test` |
| 2 | Filename prefix | 90 | `test` → `test.go` |
| 3 | Filename contains (early) | 80-70 | `test` → `my_test.go` |
| 4 | Path segment exact | 60 | `src` → `src/main.go` |
| 5 | Path segment prefix | 50 | `int` → `internal/core.go` |
| 6 | Fuzzy consecutive | 40+ | `cfg` → `config.go` |
| 7 | Fuzzy scattered | 20+ | `mgo` → `main.go` |

**Additional Bonuses:**
- Consecutive character matches: +15 points
- Match after separator (`/`, `_`, `-`, `.`): +10 points
- Shorter paths: +50 (< 20 chars), +25 (< 40 chars), +10 (else)
- Match in filename vs path: +30 points
- Results always sorted by score (highest first)

**Performance:**
- Score calculation: ~1μs per path
- Search 10k files: <10ms

### Searcher (High-Level API) ⭐ NEW

High-level API that combines Scanner and Matcher with async indexing and intelligent ranking.

```go
type Searcher struct {
    root    string
    scanner *Scanner
    matcher *Matcher
    index   []string
    indexed bool
}
```

**Constructor:**
```go
searcher, err := filesearch.NewSearcher("/workspace")
if err != nil {
    log.Fatal(err)
}
```

**Core Methods:**

```go
// IndexAsync indexes files asynchronously with context cancellation
func (s *Searcher) IndexAsync(ctx context.Context) error

// Search performs ranked search on indexed files
func (s *Searcher) Search(query string, limit int) []Match

// IsIndexed returns true if indexing is complete
func (s *Searcher) IsIndexed() bool
```

**Features:**
- ✅ Async indexing with context cancellation
- ✅ Thread-safe concurrent searches (sync.RWMutex)
- ✅ Idempotent IndexAsync (safe to call multiple times)
- ✅ Automatic .gitignore/.spinignore support
- ✅ Advanced 7-tier ranking for best results
- ✅ Efficient memory usage (<10MB for 100k files)

---

## Usage Examples

### Quick Start with Searcher (Recommended)

```go
package main

import (
    "context"
    "fmt"
    "log"
    "github.com/dmytrogajewski/spin/internal/filesearch"
)

func main() {
    // Create searcher
    searcher, err := filesearch.NewSearcher(".")
    if err != nil {
        log.Fatal(err)
    }

    // Index files asynchronously
    ctx := context.Background()
    if err := searcher.IndexAsync(ctx); err != nil {
        log.Fatal(err)
    }

    // Search with intelligent ranking
    results := searcher.Search("test", 10) // top 10 matches
    for i, match := range results {
        fmt.Printf("%d. %s (score: %d)\n", i+1, match.Path, match.Score)
    }
}
```

### Async Indexing with Cancellation

```go
package main

import (
    "context"
    "fmt"
    "time"
    "github.com/dmytrogajewski/spin/internal/filesearch"
)

func main() {
    searcher, _ := filesearch.NewSearcher(".")

    // Create context with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    // Index with cancellation support
    if err := searcher.IndexAsync(ctx); err != nil {
        if err == context.Canceled {
            fmt.Println("Indexing cancelled by user")
        } else if err == context.DeadlineExceeded {
            fmt.Println("Indexing timed out")
        } else {
            fmt.Printf("Error: %v\n", err)
        }
        return
    }

    // Search only if fully indexed
    if searcher.IsIndexed() {
        results := searcher.Search("main", 5)
        // Process results...
    }
}
```

### Concurrent Searches

```go
package main

import (
    "context"
    "fmt"
    "sync"
    "github.com/dmytrogajewski/spin/internal/filesearch"
)

func main() {
    searcher, _ := filesearch.NewSearcher(".")
    searcher.IndexAsync(context.Background())

    // Multiple concurrent searches (thread-safe)
    var wg sync.WaitGroup
    queries := []string{"test", "main", "config"}

    for _, query := range queries {
        wg.Add(1)
        go func(q string) {
            defer wg.Done()
            results := searcher.Search(q, 10)
            fmt.Printf("%s: %d matches\n", q, len(results))
        }(query)
    }

    wg.Wait()
}
```

### Basic File Scanning

```go
package main

import (
    "fmt"
    "github.com/dmytrogajewski/spin/internal/filesearch"
)

func main() {
    // Create scanner - automatically loads .gitignore and .spinignore
    scanner := filesearch.NewScanner(".", false)

    // Scan for all files (respecting ignore patterns)
    files, err := scanner.Scan()
    if err != nil {
        panic(err)
    }

    fmt.Printf("Found %d files\n", len(files))
    for _, file := range files {
        fmt.Println(file)
    }
}
```

### Custom Ignore Handler

```go
// Create custom ignore handler
handler, _ := filesearch.NewIgnoreHandler("/workspace")

// Create scanner with custom handler
scanner := filesearch.NewScannerWithIgnore("/workspace", handler)
files, _ := scanner.Scan()
```

### Fuzzy Matching

```go
// Scan files
scanner := filesearch.NewScanner(".", false)
files, _ := scanner.Scan()

// Create matcher
matcher := filesearch.NewMatcher(false) // case-insensitive

// Find matches for "test"
matches := matcher.Match("test", files)

// Print top 10 matches
for i, match := range matches {
    if i >= 10 {
        break
    }
    fmt.Printf("%s (score: %d)\n", match.Path, match.Score)
}
```

### Manual Ignore Checking

```go
handler, _ := filesearch.NewIgnoreHandler("/workspace")

// Check if path should be ignored
if handler.IsIgnored("node_modules/pkg/index.js", false) {
    fmt.Println("File is ignored")
}

if handler.IsIgnored("build", true) {  // directory
    fmt.Println("Directory is ignored")
}
```

---

## Gitignore Pattern Syntax

The IgnoreHandler uses the `doublestar` library for pattern matching, supporting standard gitignore syntax:

### Wildcards
- `*` - Matches any characters except `/`
- `**` - Matches any characters including `/` (recursive)
- `?` - Matches exactly one character

### Examples
```gitignore
# Ignore all .log files
*.log
**/*.log

# Ignore build directory and all contents
build/
build/**

# Ignore at any depth
**/temp

# Ignore node_modules everywhere
node_modules/**

# Ignore Python bytecode
*.pyc
*.pyo
__pycache__/**
```

### Directory-Only Patterns
Patterns ending with `/` only match directories:
```gitignore
dist/       # Matches dist directory, not dist file
```

---

## Performance

### Benchmarks
```
BenchmarkIgnoreHandler_IsIgnored_100Patterns    200k ops   5.6 μs/op
BenchmarkIgnoreHandler_IsIgnored_1000Patterns    22k ops  55.0 μs/op
BenchmarkScanner_WithIgnore_10kFiles              90 ops  12.5 ms/op
```

**Characteristics:**
- Pattern matching: O(n*m) where n = patterns, m = path depth
- Memory: ~0 allocations for IsIgnored (patterns pre-loaded)
- Scanning: ~12ms for 10k files with ignore rules
- Well under performance targets (<100ms for typical projects)

---

## Integration with Spin

### TUI File Picker
The Scanner is used by Spin's TUI file picker to provide fast, relevant file suggestions:

```go
// In TUI code
scanner := filesearch.NewScanner(workspaceDir, false)
files, _ := scanner.Scan()

matcher := filesearch.NewMatcher(false)
matches := matcher.Match(userQuery, files)

// Display top matches to user
```

### Automatic Exclusions
With gitignore support, the file picker automatically excludes:
- Build artifacts (dist/, build/, out/)
- Dependencies (node_modules/, vendor/)
- IDE files (.vscode/, .idea/)
- Git internal files (.git/)
- OS files (.DS_Store, Thumbs.db)

This dramatically reduces noise and improves search relevance.

---

## Configuration

### Custom .spinignore
Create a `.spinignore` file in your workspace root to add custom ignore patterns:

```gitignore
# Custom patterns
*.tmp
tmp/
experimental/
```

### Combining Patterns
Patterns from multiple sources are combined:
1. Default patterns (always applied)
2. .gitignore patterns (if file exists)
3. .spinignore patterns (if file exists)

All patterns are OR'ed together - if any pattern matches, the file is ignored.

---

## Error Handling

### Graceful Degradation
- Missing `.gitignore` or `.spinignore` files are not errors
- Malformed patterns are silently skipped
- Permission errors during scan skip the file/directory
- IgnoreHandler creation errors are ignored in Scanner (falls back to defaults)

### Error Cases
The only errors returned are:
- `Scanner.Scan()`: Catastrophic filesystem errors (rare)
- `IgnoreHandler.loadIgnoreFile()`: File exists but cannot be read

---

## Testing

### Test Coverage: 93.6%

**Test Categories:**
- **IgnoreHandler:** 30+ test cases
  - Pattern loading (.gitignore, .spinignore)
  - Pattern matching (wildcards, directories, defaults)
  - Edge cases (empty paths, comments, whitespace)
  - Performance tests (100-1000 patterns)

- **Scanner Integration:** 10+ test cases
  - Integration with IgnoreHandler
  - Real-world project structures (Node.js, Go, Python)
  - Backward compatibility

### Running Tests
```bash
go test ./internal/filesearch/
go test -race ./internal/filesearch/
go test -bench=. ./internal/filesearch/
go test -cover ./internal/filesearch/
```

---

## Dependencies

**External:**
- `github.com/bmatcuk/doublestar/v4` - Gitignore pattern matching

**Internal:**
- `os`, `path/filepath` - Filesystem operations
- `bufio` - Efficient file reading

---

## API Reference

### Scanner

#### NewScanner
```go
func NewScanner(baseDir string, ignoreGit bool) *Scanner
```
Creates a new scanner. The `ignoreGit` parameter is deprecated but maintained for backward compatibility.

#### NewScannerWithIgnore
```go
func NewScannerWithIgnore(baseDir string, handler *IgnoreHandler) *Scanner
```
Creates a scanner with a custom IgnoreHandler.

#### Scan
```go
func (s *Scanner) Scan() ([]string, error)
```
Scans the directory recursively and returns relative file paths. Respects ignore patterns.

### IgnoreHandler

#### NewIgnoreHandler
```go
func NewIgnoreHandler(rootDir string) (*IgnoreHandler, error)
```
Creates a new ignore handler, loads patterns from .gitignore and .spinignore.

#### IsIgnored
```go
func (h *IgnoreHandler) IsIgnored(relPath string, isDir bool) bool
```
Checks if a path should be ignored. Returns true if any pattern matches.

### Matcher

#### NewMatcher
```go
func NewMatcher(caseSensitive bool) *Matcher
```
Creates a new fuzzy matcher.

#### Match
```go
func (m *Matcher) Match(query string, paths []string) []Match
```
Finds all fuzzy matches for query in paths, sorted by score.

#### Score
```go
func (m *Matcher) Score(query, path string) (int, []int)
```
Calculates the fuzzy match score for a single path.

---

## Changelog

### 2025-10-12 - v2.0
- ✅ Added IgnoreHandler with .gitignore and .spinignore support
- ✅ Default ignore patterns for common files
- ✅ Integration with Scanner (automatic)
- ✅ 93.6% test coverage
- ✅ Performance: <15ms for 10k files with ignore

### Previous
- v1.0 - Basic Scanner and Matcher with hardcoded .git ignore

---

## Known Limitations

1. **No Negation Patterns:** The `!pattern` syntax to un-ignore files is not supported yet
2. **Single .gitignore:** Only loads .gitignore from workspace root, not nested .gitignore files
3. **No .git/info/exclude:** Does not read Git's exclude file
4. **Case Sensitivity:** Patterns are case-sensitive (Git default behavior)

**Future Enhancements:**
- Negation patterns (`!important.log`)
- Nested .gitignore support
- .git/info/exclude support
- Case-insensitive pattern option

---

## Troubleshooting

### Files Not Being Ignored

**Problem:** File should be ignored but appears in results.

**Solutions:**
1. Check pattern syntax in `.gitignore` or `.spinignore`
2. Use `**/*.ext` for files at any depth
3. Use `dir/**` to ignore entire directory tree
4. Verify file is not matched by earlier non-ignore pattern

### Too Many Files Excluded

**Problem:** Legitimate files are being ignored.

**Solutions:**
1. Check default patterns match your file
2. Create `.spinignore` to override (future: use negation)
3. Use custom IgnoreHandler without defaults

### Slow Performance

**Problem:** Scanning takes too long.

**Solutions:**
1. Add more directories to ignore patterns
2. Limit scan to subdirectory instead of root
3. Check for extremely deep directory structures (>20 levels)

---

**Last Updated:** 2025-10-12
**Maintainer:** Spin Development Team
