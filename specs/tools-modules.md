# Spin Tools & Utility Modules - Technical Documentation

## Overview

This document covers the tool and utility modules that provide core functionality for code manipulation, file operations, and project analysis for the Spin AI coding agent.

**Key Packages:**
1. **internal/patchapply** - File modification via diff patches
2. **internal/filesearch** - Fuzzy file searching
3. **internal/git** - Git repository operations
4. **internal/gitpatch** - Git patch application
5. **pkg/pathutil** - Path utilities and validation
6. **pkg/strutil** - String manipulation utilities
7. **pkg/ansi** - ANSI escape sequence handling

---

## Package 1: internal/patchapply

**Path:** `internal/patchapply/`  
**Purpose:** Apply file modifications via structured patch format

### Overview

`patchapply` is Spin's primary file manipulation tool. It uses a custom, simplified patch format designed to be:
- Easy for AI models to generate correctly
- Safe to parse and apply
- Clear in intent (add/delete/update)
- Resistant to ambiguity

### Patch Format

#### Structure

```
*** Begin Patch
[file operations]
*** End Patch
```

#### File Operations

**1. Add File**
```
*** Add File: path/to/new_file.txt
+Line 1 of content
+Line 2 of content
+Line 3 of content
```

**2. Delete File**
```
*** Delete File: path/to/old_file.txt
```

**3. Update File**
```
*** Update File: path/to/existing_file.go
@@ type MyStruct struct
 type MyStruct struct {
-    OldField string
+    NewField string
     // Context after
 }
```

**4. Move File (during update)**
```
*** Update File: old/path.txt
*** Move to: new/path.txt
@@ 
+New content if needed
```

### Patch Grammar

```
Patch := Begin { FileOp } End
Begin := "*** Begin Patch" NEWLINE
End := "*** End Patch" NEWLINE

FileOp := AddFile | DeleteFile | UpdateFile

AddFile := "*** Add File: " path NEWLINE { "+" line NEWLINE }

DeleteFile := "*** Delete File: " path NEWLINE

UpdateFile := "*** Update File: " path NEWLINE 
              [ MoveTo ]
              { Hunk }

MoveTo := "*** Move to: " newPath NEWLINE

Hunk := "@@" [ header ] NEWLINE
        { HunkLine }
        [ "*** End of File" NEWLINE ]

HunkLine := (" " | "-" | "+") text NEWLINE
```

### Context Rules

**Context Requirements:**
- **Default:** 3 lines before and 3 lines after each change
- **Ambiguity:** Use `@@` header to specify function/type context
- **Multiple Occurrences:** Use nested `@@` for deeper specificity

**Example with Function Context:**
```
*** Update File: handler.go
@@ func (h *BaseHandler) Process
 func (h *BaseHandler) Process(data string) error {
     // Line before
-    return oldValue
+    return newValue
     // Line after
 }
```

**Example with Nested Context:**
```
*** Update File: complex.go
@@ type OuterStruct struct
@@     func (o *OuterStruct) InnerMethod
         // Specific context
-        oldCode()
+        newCode()
         // More context
```

### Architecture

```
internal/patchapply/
├── patchapply.go       # Main package interface
├── parser.go           # Patch parsing
├── matcher.go          # Fuzzy matching for hunks
├── applier.go          # Patch application logic
└── types.go            # Core data structures
```

### Implementation Details

#### Core Types (`types.go`)

```go
package patchapply

import "io/fs"

// Patch represents a complete patch operation
type Patch struct {
    Operations []FileOperation
}

// FileOperation is a union type for file operations
type FileOperation interface {
    isFileOperation()
    Path() string
}

// AddFile represents adding a new file
type AddFile struct {
    FilePath string
    Lines    []string
}

// DeleteFile represents deleting a file
type DeleteFile struct {
    FilePath string
}

// UpdateFile represents updating an existing file
type UpdateFile struct {
    FilePath    string
    NewPath     string // Optional, for move operations
    Hunks       []Hunk
}

// Hunk represents a change section within a file
type Hunk struct {
    Header  string      // Optional context header (e.g., "func MyFunc")
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
```

#### Parser (`parser.go`)

```go
package patchapply

import (
    "bufio"
    "fmt"
    "strings"
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
    switch {
    case strings.HasPrefix(line, "*** Add File: "):
        return p.parseAddFile(strings.TrimPrefix(line, "*** Add File: "))
    case strings.HasPrefix(line, "*** Delete File: "):
        return p.parseDeleteFile(strings.TrimPrefix(line, "*** Delete File: "))
    case strings.HasPrefix(line, "*** Update File: "):
        return p.parseUpdateFile(strings.TrimPrefix(line, "*** Update File: "))
    default:
        return nil, fmt.Errorf("unknown operation: %s", line)
    }
}

// Additional parsing methods...
```

#### Fuzzy Matcher (`matcher.go`)

```go
package patchapply

import (
    "strings"
)

// Matcher finds context in file content using fuzzy matching
type Matcher struct {
    content      []string
    threshold    float64 // Similarity threshold (0.0-1.0)
}

// NewMatcher creates a new fuzzy matcher
func NewMatcher(content []string) *Matcher {
    return &Matcher{
        content:   content,
        threshold: 0.85, // 85% similarity required
    }
}

// FindContext searches for the best match of context lines
// Returns the starting line number or -1 if not found
func (m *Matcher) FindContext(contextLines []string) int {
    if len(contextLines) == 0 {
        return 0
    }
    
    bestScore := 0.0
    bestLine := -1
    
    // Sliding window search
    for i := 0; i <= len(m.content)-len(contextLines); i++ {
        score := m.similarity(contextLines, m.content[i:i+len(contextLines)])
        if score > bestScore && score >= m.threshold {
            bestScore = score
            bestLine = i
        }
    }
    
    return bestLine
}

// similarity calculates similarity between two line slices
func (m *Matcher) similarity(a, b []string) float64 {
    if len(a) != len(b) {
        return 0.0
    }
    
    matches := 0
    for i := range a {
        if m.linesMatch(a[i], b[i]) {
            matches++
        }
    }
    
    return float64(matches) / float64(len(a))
}

// linesMatch checks if two lines match (with whitespace tolerance)
func (m *Matcher) linesMatch(a, b string) bool {
    // Normalize whitespace
    a = strings.Join(strings.Fields(a), " ")
    b = strings.Join(strings.Fields(b), " ")
    return a == b
}

// SetThreshold sets the similarity threshold
func (m *Matcher) SetThreshold(threshold float64) {
    if threshold >= 0.0 && threshold <= 1.0 {
        m.threshold = threshold
    }
}
```

#### Applier (`applier.go`)

```go
package patchapply

import (
    "fmt"
    "os"
    "path/filepath"
    "strings"
    
    "spin/pkg/pathutil"
)

// Applier applies patches to the filesystem
type Applier struct {
    workspaceRoot string
    dryRun        bool
    backup        bool
}

// NewApplier creates a new patch applier
func NewApplier(workspaceRoot string) *Applier {
    return &Applier{
        workspaceRoot: workspaceRoot,
        dryRun:        false,
        backup:        true,
    }
}

// SetDryRun enables or disables dry-run mode
func (a *Applier) SetDryRun(enabled bool) {
    a.dryRun = enabled
}

// SetBackup enables or disables backup creation
func (a *Applier) SetBackup(enabled bool) {
    a.backup = enabled
}

// Apply applies a patch to the workspace
func (a *Applier) Apply(patch *Patch) error {
    // Validate all paths first
    for _, op := range patch.Operations {
        if err := a.validatePath(op.Path()); err != nil {
            return fmt.Errorf("invalid path %q: %w", op.Path(), err)
        }
    }
    
    // Apply operations
    for _, op := range patch.Operations {
        if err := a.applyOperation(op); err != nil {
            return fmt.Errorf("failed to apply operation on %q: %w", op.Path(), err)
        }
    }
    
    return nil
}

// validatePath ensures path is safe and within workspace
func (a *Applier) validatePath(relPath string) error {
    // Use pathutil for validation
    if err := pathutil.ValidateRelativePath(relPath); err != nil {
        return err
    }
    
    // Ensure it doesn't escape workspace
    absPath := filepath.Join(a.workspaceRoot, relPath)
    absPath, err := filepath.Abs(absPath)
    if err != nil {
        return err
    }
    
    absRoot, err := filepath.Abs(a.workspaceRoot)
    if err != nil {
        return err
    }
    
    if !strings.HasPrefix(absPath, absRoot) {
        return fmt.Errorf("path escapes workspace")
    }
    
    return nil
}

// applyOperation applies a single file operation
func (a *Applier) applyOperation(op FileOperation) error {
    switch v := op.(type) {
    case *AddFile:
        return a.applyAddFile(v)
    case *DeleteFile:
        return a.applyDeleteFile(v)
    case *UpdateFile:
        return a.applyUpdateFile(v)
    default:
        return fmt.Errorf("unknown operation type: %T", op)
    }
}

// Additional apply methods...
```

### Usage

#### As Library

```go
package main

import (
    "fmt"
    "log"
    
    "spin/internal/patchapply"
)

func main() {
    patchText := `*** Begin Patch
*** Update File: hello.txt
@@
-Hello World
+Hello Spin
*** End Patch`
    
    parser := patchapply.NewParser(patchText)
    patch, err := parser.Parse()
    if err != nil {
        log.Fatalf("Parse error: %v", err)
    }
    
    applier := patchapply.NewApplier("/workspace")
    if err := applier.Apply(patch); err != nil {
        log.Fatalf("Apply error: %v", err)
    }
    
    fmt.Println("Patch applied successfully")
}
```

#### As CLI Tool

```bash
# Apply patch from stdin
echo "*** Begin Patch..." | spin-apply-patch

# Apply patch from file
spin-apply-patch < changes.patch

# Dry-run mode
spin-apply-patch --dry-run < changes.patch
```

### Integration with Core

**In Spin Core:**
1. AI generates patch in tool call
2. Core validates patch syntax
3. Core invokes patchapply package
4. Applier returns success/failure with detailed errors
5. Core reports result to AI

### Safety Features

1. **Relative Paths Only:** Absolute paths rejected
2. **Workspace Confinement:** Operations limited to workspace via `pathutil`
3. **Atomic Application:** All changes or none (transaction-like)
4. **Backup Support:** Optional backup before apply
5. **Dry-Run Mode:** Preview changes without applying
6. **Symlink Protection:** Validates symlinks don't escape workspace

### Error Messages

**Clear Error Reporting:**
```
Error: Failed to apply hunk at line 45 in file.go
Reason: Context not found
Expected:
  func oldFunction() {
      return value
  }
Actual:
  func oldFunction(ctx context.Context) {  // <- signature changed
      return value
  }
```

### Testing

**Test Coverage:**
- Successful patch application
- Syntax error handling
- Context mismatch scenarios
- File not found cases
- Edge cases (empty files, binary files)
- Path traversal attacks

```bash
go test ./internal/patchapply/...
go test ./internal/patchapply -v -race
```

---

## Package 2: internal/filesearch

**Path:** `internal/filesearch/`  
**Purpose:** Fast fuzzy file searching in project directories

### Overview

Provides fuzzy filename search with:
- Substring matching
- Path-aware scoring
- .gitignore respect
- Fast indexing with caching

### Implementation

**Core Dependencies:**
- Standard library `io/fs`, `path/filepath`
- Third-party: `github.com/bmatcuk/doublestar/v4` for gitignore patterns

**Features:**
- Recursive directory traversal
- .gitignore/.spinignore parsing
- Concurrent file discovery with `sync.WaitGroup`
- In-memory file index with mutex protection

### Package Structure

```go
package filesearch

import (
    "context"
    "io/fs"
    "path/filepath"
    "sync"
)

// Searcher provides fuzzy file searching
type Searcher struct {
    root    string
    index   []FileEntry
    mu      sync.RWMutex
    ignorer *IgnoreHandler
}

// FileEntry represents a file in the index
type FileEntry struct {
    Path         string  // Relative path from root
    Name         string  // Base filename
    ModTime      int64   // Unix timestamp
}

// SearchResult represents a search match
type SearchResult struct {
    Path  string
    Score float64
}

// NewSearcher creates a new file searcher
func NewSearcher(root string) (*Searcher, error) {
    ignorer, err := NewIgnoreHandler(root)
    if err != nil {
        return nil, err
    }
    
    s := &Searcher{
        root:    root,
        index:   make([]FileEntry, 0, 1000),
        ignorer: ignorer,
    }
    
    return s, nil
}

// IndexAsync builds the file index asynchronously
func (s *Searcher) IndexAsync(ctx context.Context) error {
    entries := make([]FileEntry, 0, 1000)
    
    err := filepath.WalkDir(s.root, func(path string, d fs.DirEntry, err error) error {
        if err != nil {
            return err
        }
        
        // Check context cancellation
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
        }
        
        relPath, err := filepath.Rel(s.root, path)
        if err != nil {
            return err
        }
        
        // Check if ignored
        if s.ignorer.IsIgnored(relPath, d.IsDir()) {
            if d.IsDir() {
                return fs.SkipDir
            }
            return nil
        }
        
        if !d.IsDir() {
            info, err := d.Info()
            if err != nil {
                return nil // Skip files we can't stat
            }
            
            entries = append(entries, FileEntry{
                Path:    relPath,
                Name:    d.Name(),
                ModTime: info.ModTime().Unix(),
            })
        }
        
        return nil
    })
    
    if err != nil {
        return err
    }
    
    s.mu.Lock()
    s.index = entries
    s.mu.Unlock()
    
    return nil
}

// Search performs fuzzy search and returns top results
func (s *Searcher) Search(query string, limit int) []SearchResult {
    s.mu.RLock()
    defer s.mu.RUnlock()
    
    if query == "" {
        return nil
    }
    
    results := make([]SearchResult, 0, limit)
    
    for _, entry := range s.index {
        score := s.scoreMatch(query, entry)
        if score > 0 {
            results = append(results, SearchResult{
                Path:  entry.Path,
                Score: score,
            })
        }
    }
    
    // Sort by score descending
    sortByScore(results)
    
    // Limit results
    if len(results) > limit {
        results = results[:limit]
    }
    
    return results
}
```

### Scoring Algorithm

```go
// scoreMatch calculates match score for a file entry
func (s *Searcher) scoreMatch(query string, entry FileEntry) float64 {
    queryLower := strings.ToLower(query)
    nameLower := strings.ToLower(entry.Name)
    pathLower := strings.ToLower(entry.Path)
    
    score := 0.0
    
    // Exact filename match
    if nameLower == queryLower {
        score = 100.0
        return score
    }
    
    // Filename starts with query
    if strings.HasPrefix(nameLower, queryLower) {
        score = 90.0
        return score
    }
    
    // Filename contains query
    if idx := strings.Index(nameLower, queryLower); idx >= 0 {
        // Earlier in filename is better
        score = 80.0 - float64(idx)
        return score
    }
    
    // Path contains query
    if idx := strings.Index(pathLower, queryLower); idx >= 0 {
        score = 60.0 - float64(idx)/10.0
        return score
    }
    
    // Fuzzy match (consecutive characters)
    if fuzzyScore := fuzzyMatch(queryLower, nameLower); fuzzyScore > 0 {
        score = 40.0 + fuzzyScore
        return score
    }
    
    return 0.0
}

// fuzzyMatch calculates fuzzy match score
func fuzzyMatch(query, target string) float64 {
    if len(query) == 0 {
        return 0
    }
    
    matches := 0
    targetIdx := 0
    
    for _, ch := range query {
        found := false
        for targetIdx < len(target) {
            if rune(target[targetIdx]) == ch {
                matches++
                found = true
                targetIdx++
                break
            }
            targetIdx++
        }
        if !found {
            break
        }
    }
    
    return float64(matches) / float64(len(query)) * 30.0
}
```

### Ignore Handling

```go
package filesearch

import (
    "bufio"
    "os"
    "path/filepath"
    "strings"
    
    "github.com/bmatcuk/doublestar/v4"
)

// IgnoreHandler handles .gitignore and .spinignore patterns
type IgnoreHandler struct {
    patterns []string
}

// NewIgnoreHandler creates a new ignore handler
func NewIgnoreHandler(root string) (*IgnoreHandler, error) {
    h := &IgnoreHandler{
        patterns: make([]string, 0),
    }
    
    // Load .gitignore
    gitignorePath := filepath.Join(root, ".gitignore")
    if err := h.loadIgnoreFile(gitignorePath); err != nil && !os.IsNotExist(err) {
        return nil, err
    }
    
    // Load .spinignore
    spinignorePath := filepath.Join(root, ".spinignore")
    if err := h.loadIgnoreFile(spinignorePath); err != nil && !os.IsNotExist(err) {
        return nil, err
    }
    
    // Add default patterns
    h.patterns = append(h.patterns, ".git/**", "node_modules/**", ".spin/**")
    
    return h, nil
}

// loadIgnoreFile loads patterns from an ignore file
func (h *IgnoreHandler) loadIgnoreFile(path string) error {
    f, err := os.Open(path)
    if err != nil {
        return err
    }
    defer f.Close()
    
    scanner := bufio.NewScanner(f)
    for scanner.Scan() {
        line := strings.TrimSpace(scanner.Text())
        if line == "" || strings.HasPrefix(line, "#") {
            continue
        }
        h.patterns = append(h.patterns, line)
    }
    
    return scanner.Err()
}

// IsIgnored checks if a path should be ignored
func (h *IgnoreHandler) IsIgnored(path string, isDir bool) bool {
    for _, pattern := range h.patterns {
        matched, err := doublestar.Match(pattern, path)
        if err == nil && matched {
            return true
        }
        
        // For directories, also check with trailing slash
        if isDir {
            matched, err = doublestar.Match(pattern, path+"/")
            if err == nil && matched {
                return true
            }
        }
    }
    return false
}
```

### CLI Tool

```go
package main

import (
    "context"
    "flag"
    "fmt"
    "log"
    "os"
    
    "spin/internal/filesearch"
)

func main() {
    limit := flag.Int("limit", 10, "Maximum number of results")
    flag.Parse()
    
    if flag.NArg() < 1 {
        log.Fatal("Usage: spin-file-search [--limit N] <query>")
    }
    
    query := flag.Arg(0)
    cwd, err := os.Getwd()
    if err != nil {
        log.Fatal(err)
    }
    
    searcher, err := filesearch.NewSearcher(cwd)
    if err != nil {
        log.Fatal(err)
    }
    
    ctx := context.Background()
    if err := searcher.IndexAsync(ctx); err != nil {
        log.Fatal(err)
    }
    
    results := searcher.Search(query, *limit)
    for _, r := range results {
        fmt.Printf("%s (score: %.1f)\n", r.Path, r.Score)
    }
}
```

### Integration

**Used by:**
- Interactive mode `@` file search feature
- HTTP API `search_files` endpoint
- MCP server file discovery

---

## Package 3: internal/git

**Path:** `internal/git/`  
**Purpose:** Git repository operations and information gathering

### Overview

Provides Git operations using the `go-git` library (pure Go implementation) or shelling out to `git` command for complex operations.

### Package Structure

```go
package git

import (
    "context"
    "fmt"
    "os/exec"
    "path/filepath"
    "strings"
)

// Repository represents a Git repository
type Repository struct {
    root string
}

// Discover finds a git repository starting from the given path
func Discover(startPath string) (*Repository, error) {
    absPath, err := filepath.Abs(startPath)
    if err != nil {
        return nil, err
    }
    
    // Walk up directory tree looking for .git
    for {
        gitPath := filepath.Join(absPath, ".git")
        if stat, err := os.Stat(gitPath); err == nil && stat.IsDir() {
            return &Repository{root: absPath}, nil
        }
        
        parent := filepath.Dir(absPath)
        if parent == absPath {
            return nil, fmt.Errorf("not a git repository")
        }
        absPath = parent
    }
}

// Root returns the repository root path
func (r *Repository) Root() string {
    return r.root
}

// Status returns current repository status
func (r *Repository) Status(ctx context.Context) (*Status, error) {
    cmd := exec.CommandContext(ctx, "git", "status", "--porcelain", "--branch")
    cmd.Dir = r.root
    
    output, err := cmd.Output()
    if err != nil {
        return nil, fmt.Errorf("git status failed: %w", err)
    }
    
    return parseStatus(string(output))
}

// Status information
type Status struct {
    Branch        string
    RemoteBranch  string
    Ahead         int
    Behind        int
    ModifiedFiles []string
    UntrackedFiles []string
}

// parseStatus parses git status --porcelain output
func parseStatus(output string) (*Status, error) {
    status := &Status{
        ModifiedFiles:  make([]string, 0),
        UntrackedFiles: make([]string, 0),
    }
    
    lines := strings.Split(strings.TrimSpace(output), "\n")
    if len(lines) == 0 {
        return status, nil
    }
    
    // First line is branch info
    branchLine := lines[0]
    if strings.HasPrefix(branchLine, "## ") {
        branchInfo := strings.TrimPrefix(branchLine, "## ")
        parts := strings.Split(branchInfo, "...")
        status.Branch = parts[0]
        
        if len(parts) > 1 {
            remoteParts := strings.Fields(parts[1])
            status.RemoteBranch = remoteParts[0]
            
            // Parse ahead/behind
            for _, part := range remoteParts[1:] {
                if strings.HasPrefix(part, "[ahead") {
                    fmt.Sscanf(part, "[ahead %d]", &status.Ahead)
                } else if strings.HasPrefix(part, "behind") {
                    fmt.Sscanf(part, "behind %d]", &status.Behind)
                }
            }
        }
    }
    
    // Remaining lines are file status
    for _, line := range lines[1:] {
        if len(line) < 4 {
            continue
        }
        
        statusCode := line[:2]
        filePath := line[3:]
        
        switch {
        case statusCode == "??":
            status.UntrackedFiles = append(status.UntrackedFiles, filePath)
        default:
            status.ModifiedFiles = append(status.ModifiedFiles, filePath)
        }
    }
    
    return status, nil
}

// CurrentBranch returns the current branch name
func (r *Repository) CurrentBranch(ctx context.Context) (string, error) {
    cmd := exec.CommandContext(ctx, "git", "branch", "--show-current")
    cmd.Dir = r.root
    
    output, err := cmd.Output()
    if err != nil {
        return "", fmt.Errorf("git branch failed: %w", err)
    }
    
    return strings.TrimSpace(string(output)), nil
}

// ListBranches returns all branches
func (r *Repository) ListBranches(ctx context.Context) ([]string, error) {
    cmd := exec.CommandContext(ctx, "git", "branch", "--list")
    cmd.Dir = r.root
    
    output, err := cmd.Output()
    if err != nil {
        return nil, fmt.Errorf("git branch failed: %w", err)
    }
    
    branches := make([]string, 0)
    for _, line := range strings.Split(string(output), "\n") {
        branch := strings.TrimPrefix(strings.TrimSpace(line), "* ")
        if branch != "" {
            branches = append(branches, branch)
        }
    }
    
    return branches, nil
}

// DiffToBranch returns diff to specified branch
func (r *Repository) DiffToBranch(ctx context.Context, branch string) (*Diff, error) {
    cmd := exec.CommandContext(ctx, "git", "diff", "--name-status", branch)
    cmd.Dir = r.root
    
    output, err := cmd.Output()
    if err != nil {
        return nil, fmt.Errorf("git diff failed: %w", err)
    }
    
    return parseDiff(string(output))
}

// Diff represents changes between commits
type Diff struct {
    Files []FileChange
}

// FileChange represents a single file change
type FileChange struct {
    Status string // A, M, D, R
    Path   string
}

// parseDiff parses git diff --name-status output
func parseDiff(output string) (*Diff, error) {
    diff := &Diff{
        Files: make([]FileChange, 0),
    }
    
    for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
        if line == "" {
            continue
        }
        
        parts := strings.Fields(line)
        if len(parts) >= 2 {
            diff.Files = append(diff.Files, FileChange{
                Status: parts[0],
                Path:   parts[1],
            })
        }
    }
    
    return diff, nil
}

// RemoteURL returns the remote URL
func (r *Repository) RemoteURL(ctx context.Context, remote string) (string, error) {
    cmd := exec.CommandContext(ctx, "git", "remote", "get-url", remote)
    cmd.Dir = r.root
    
    output, err := cmd.Output()
    if err != nil {
        return "", fmt.Errorf("git remote failed: %w", err)
    }
    
    return strings.TrimSpace(string(output)), nil
}
```

### Integration with Core

**Context Gathering:**
Core uses `internal/git` to provide AI with:
- Current branch name
- Uncommitted changes status
- Remote repository URL
- Recent commit history

**Example Context:**
```
Repository: github.com/user/project
Branch: feature/new-api
Status: 3 modified files, 1 untracked file
```

---

## Package 4: internal/gitpatch

**Path:** `internal/gitpatch/`  
**Purpose:** Apply Git-format patches

### Overview

Handles standard Git unified diff format (different from `patchapply`).

**Use Cases:**
- Applying patches from `git diff`
- Integrating external patches
- Importing changes from Git

### Format Support

**Git Unified Diff:**
```
diff --git a/file.txt b/file.txt
index 1234567..abcdef0 100644
--- a/file.txt
+++ b/file.txt
@@ -1,3 +1,3 @@
 line 1
-old line
+new line
 line 3
```

### Implementation

```go
package gitpatch

import (
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
)

// Applier applies Git-format patches
type Applier struct {
    workspaceRoot string
}

// NewApplier creates a new git patch applier
func NewApplier(workspaceRoot string) *Applier {
    return &Applier{
        workspaceRoot: workspaceRoot,
    }
}

// Apply applies a git-format patch
func (a *Applier) Apply(patchText string) error {
    // Write patch to temporary file
    tmpFile, err := os.CreateTemp("", "spin-patch-*.patch")
    if err != nil {
        return fmt.Errorf("create temp file: %w", err)
    }
    defer os.Remove(tmpFile.Name())
    defer tmpFile.Close()
    
    if _, err := tmpFile.WriteString(patchText); err != nil {
        return fmt.Errorf("write patch: %w", err)
    }
    tmpFile.Close()
    
    // Apply using git apply
    cmd := exec.Command("git", "apply", "--whitespace=fix", tmpFile.Name())
    cmd.Dir = a.workspaceRoot
    
    output, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("git apply failed: %w\nOutput: %s", err, output)
    }
    
    return nil
}

// Check checks if a patch can be applied without applying it
func (a *Applier) Check(patchText string) error {
    tmpFile, err := os.CreateTemp("", "spin-patch-*.patch")
    if err != nil {
        return fmt.Errorf("create temp file: %w", err)
    }
    defer os.Remove(tmpFile.Name())
    defer tmpFile.Close()
    
    if _, err := tmpFile.WriteString(patchText); err != nil {
        return fmt.Errorf("write patch: %w", err)
    }
    tmpFile.Close()
    
    // Check using git apply --check
    cmd := exec.Command("git", "apply", "--check", tmpFile.Name())
    cmd.Dir = a.workspaceRoot
    
    output, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("patch would fail: %w\nOutput: %s", err, output)
    }
    
    return nil
}
```

---

## Package 5: pkg/pathutil

**Path:** `pkg/pathutil/`  
**Purpose:** Path utilities and validation (safe for external use)

### Overview

Provides secure path manipulation and validation to prevent path traversal attacks.

### Implementation

```go
package pathutil

import (
    "errors"
    "fmt"
    "path/filepath"
    "strings"
)

var (
    ErrAbsolutePath     = errors.New("absolute paths not allowed")
    ErrPathTraversal    = errors.New("path traversal detected")
    ErrEmptyPath        = errors.New("empty path not allowed")
)

// ValidateRelativePath validates that a path is relative and safe
func ValidateRelativePath(path string) error {
    if path == "" {
        return ErrEmptyPath
    }
    
    if filepath.IsAbs(path) {
        return ErrAbsolutePath
    }
    
    // Check for parent directory references
    cleaned := filepath.Clean(path)
    if strings.HasPrefix(cleaned, "..") {
        return ErrPathTraversal
    }
    
    // Check for hidden path traversal
    parts := strings.Split(cleaned, string(filepath.Separator))
    for _, part := range parts {
        if part == ".." {
            return ErrPathTraversal
        }
    }
    
    return nil
}

// SafeJoin joins paths and validates the result stays within root
func SafeJoin(root, relPath string) (string, error) {
    if err := ValidateRelativePath(relPath); err != nil {
        return "", err
    }
    
    joined := filepath.Join(root, relPath)
    absJoined, err := filepath.Abs(joined)
    if err != nil {
        return "", fmt.Errorf("resolve path: %w", err)
    }
    
    absRoot, err := filepath.Abs(root)
    if err != nil {
        return "", fmt.Errorf("resolve root: %w", err)
    }
    
    // Ensure joined path is under root
    if !strings.HasPrefix(absJoined, absRoot+string(filepath.Separator)) &&
       absJoined != absRoot {
        return "", ErrPathTraversal
    }
    
    return absJoined, nil
}

// NormalizePath normalizes a path for consistent comparison
func NormalizePath(path string) string {
    return filepath.Clean(path)
}

// RelativePath returns path relative to root
func RelativePath(root, path string) (string, error) {
    return filepath.Rel(root, path)
}
```

---

## Package 6: pkg/strutil

**Path:** `pkg/strutil/`  
**Purpose:** String manipulation utilities

### Overview

Advanced string operations for code manipulation.

### Implementation

```go
package strutil

import (
    "strings"
    "unicode"
)

// SplitLines splits text into lines, handling different line endings
func SplitLines(text string) []string {
    // Normalize line endings
    text = strings.ReplaceAll(text, "\r\n", "\n")
    text = strings.ReplaceAll(text, "\r", "\n")
    return strings.Split(text, "\n")
}

// DetectIndentation detects the indentation type and size
func DetectIndentation(text string) (useTabs bool, size int) {
    lines := SplitLines(text)
    
    tabCount := 0
    spaceCount := 0
    spaceSizes := make(map[int]int)
    
    for _, line := range lines {
        if len(line) == 0 {
            continue
        }
        
        // Count leading whitespace
        indent := 0
        for i, ch := range line {
            if ch == '\t' {
                tabCount++
                break
            } else if ch == ' ' {
                indent++
            } else {
                if indent > 0 {
                    spaceCount++
                    spaceSizes[indent]++
                }
                break
            }
            
            if i > 20 { // Don't check too far
                break
            }
        }
    }
    
    // Determine if tabs or spaces
    if tabCount > spaceCount {
        return true, 1
    }
    
    // Find most common space indentation size
    maxCount := 0
    commonSize := 4 // Default
    for size, count := range spaceSizes {
        if count > maxCount {
            maxCount = count
            commonSize = size
        }
    }
    
    return false, commonSize
}

// NormalizeWhitespace normalizes whitespace in text
func NormalizeWhitespace(text string) string {
    return strings.Join(strings.Fields(text), " ")
}

// TrimEmptyLines removes leading and trailing empty lines
func TrimEmptyLines(lines []string) []string {
    start := 0
    for start < len(lines) && strings.TrimSpace(lines[start]) == "" {
        start++
    }
    
    end := len(lines)
    for end > start && strings.TrimSpace(lines[end-1]) == "" {
        end--
    }
    
    if start >= end {
        return []string{}
    }
    
    return lines[start:end]
}

// LevenshteinDistance calculates edit distance between strings
func LevenshteinDistance(a, b string) int {
    if len(a) == 0 {
        return len(b)
    }
    if len(b) == 0 {
        return len(a)
    }
    
    // Create matrix
    matrix := make([][]int, len(a)+1)
    for i := range matrix {
        matrix[i] = make([]int, len(b)+1)
    }
    
    // Initialize first row and column
    for i := 0; i <= len(a); i++ {
        matrix[i][0] = i
    }
    for j := 0; j <= len(b); j++ {
        matrix[0][j] = j
    }
    
    // Fill matrix
    for i := 1; i <= len(a); i++ {
        for j := 1; j <= len(b); j++ {
            cost := 1
            if a[i-1] == b[j-1] {
                cost = 0
            }
            
            matrix[i][j] = min3(
                matrix[i-1][j]+1,      // deletion
                matrix[i][j-1]+1,      // insertion
                matrix[i-1][j-1]+cost, // substitution
            )
        }
    }
    
    return matrix[len(a)][len(b)]
}

func min3(a, b, c int) int {
    if a < b {
        if a < c {
            return a
        }
        return c
    }
    if b < c {
        return b
    }
    return c
}

// Similarity calculates similarity ratio between strings (0.0 to 1.0)
func Similarity(a, b string) float64 {
    maxLen := max(len(a), len(b))
    if maxLen == 0 {
        return 1.0
    }
    
    distance := LevenshteinDistance(a, b)
    return 1.0 - float64(distance)/float64(maxLen)
}

func max(a, b int) int {
    if a > b {
        return a
    }
    return b
}
```

---

## Package 7: pkg/ansi

**Path:** `pkg/ansi/`  
**Purpose:** ANSI escape sequence handling

### Overview

Parse, strip, and generate ANSI escape sequences for terminal formatting.

### Implementation

```go
package ansi

import (
    "fmt"
    "regexp"
    "strings"
)

var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// Strip removes all ANSI escape sequences from text
func Strip(text string) string {
    return ansiRegex.ReplaceAllString(text, "")
}

// Color codes
const (
    Reset = "\x1b[0m"
    
    Black   = "\x1b[30m"
    Red     = "\x1b[31m"
    Green   = "\x1b[32m"
    Yellow  = "\x1b[33m"
    Blue    = "\x1b[34m"
    Magenta = "\x1b[35m"
    Cyan    = "\x1b[36m"
    White   = "\x1b[37m"
    
    Bold      = "\x1b[1m"
    Dim       = "\x1b[2m"
    Italic    = "\x1b[3m"
    Underline = "\x1b[4m"
)

// Style represents text styling
type Style struct {
    text  string
    codes []string
}

// New creates a new styled string
func New(text string) *Style {
    return &Style{
        text:  text,
        codes: make([]string, 0),
    }
}

// Red applies red color
func (s *Style) Red() *Style {
    s.codes = append(s.codes, Red)
    return s
}

// Green applies green color
func (s *Style) Green() *Style {
    s.codes = append(s.codes, Green)
    return s
}

// Yellow applies yellow color
func (s *Style) Yellow() *Style {
    s.codes = append(s.codes, Yellow)
    return s
}

// Blue applies blue color
func (s *Style) Blue() *Style {
    s.codes = append(s.codes, Blue)
    return s
}

// Bold applies bold styling
func (s *Style) Bold() *Style {
    s.codes = append(s.codes, Bold)
    return s
}

// Underline applies underline styling
func (s *Style) Underline() *Style {
    s.codes = append(s.codes, Underline)
    return s
}

// String returns the styled string
func (s *Style) String() string {
    if len(s.codes) == 0 {
        return s.text
    }
    
    var b strings.Builder
    for _, code := range s.codes {
        b.WriteString(code)
    }
    b.WriteString(s.text)
    b.WriteString(Reset)
    
    return b.String()
}

// Sprintf formats and styles text
func Sprintf(format string, style *Style, args ...interface{}) string {
    text := fmt.Sprintf(format, args...)
    style.text = text
    return style.String()
}

// Parse parses ANSI text into structured format
type Segment struct {
    Text       string
    Foreground string
    Background string
    Bold       bool
    Underline  bool
}

// Parse parses ANSI text into segments
func Parse(text string) []Segment {
    segments := make([]Segment, 0)
    
    current := Segment{}
    inEscape := false
    escapeSeq := ""
    
    for i := 0; i < len(text); i++ {
        ch := text[i]
        
        if ch == '\x1b' {
            if len(current.Text) > 0 {
                segments = append(segments, current)
                current = Segment{}
            }
            inEscape = true
            escapeSeq = string(ch)
            continue
        }
        
        if inEscape {
            escapeSeq += string(ch)
            if (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') {
                // End of escape sequence
                parseEscapeSequence(escapeSeq, &current)
                inEscape = false
                escapeSeq = ""
            }
            continue
        }
        
        current.Text += string(ch)
    }
    
    if len(current.Text) > 0 {
        segments = append(segments, current)
    }
    
    return segments
}

func parseEscapeSequence(seq string, seg *Segment) {
    // Simple parser for common sequences
    if strings.Contains(seq, "[0m") {
        // Reset
        seg.Foreground = ""
        seg.Background = ""
        seg.Bold = false
        seg.Underline = false
    } else if strings.Contains(seq, "[1m") {
        seg.Bold = true
    } else if strings.Contains(seq, "[4m") {
        seg.Underline = true
    } else if strings.Contains(seq, "[31m") {
        seg.Foreground = "red"
    } else if strings.Contains(seq, "[32m") {
        seg.Foreground = "green"
    }
    // Add more as needed
}
```

---

## Cross-Module Integration

### File Modification Pipeline

```
AI generates patch text
        ↓
internal/patchapply parses patch
        ↓
pkg/pathutil validates paths
        ↓
internal/git checks git status
        ↓
internal/patchapply applies changes
        ↓
internal/filesearch updates index
```

### Search and Edit Flow

```
User types "@config"
        ↓
internal/filesearch fuzzy matches
        ↓
Returns ["config.toml", "src/config.go"]
        ↓
User selects config.toml
        ↓
AI generates patch
        ↓
internal/patchapply modifies file
```

---

## Testing Strategies

### Unit Tests

Each package has comprehensive unit tests:
```bash
go test ./internal/patchapply/...
go test ./internal/filesearch/...
go test ./internal/git/...
go test ./pkg/...
```

### Integration Tests

Test cross-module interactions:
```bash
go test ./...
go test -race ./...
```

### Table-Driven Tests

Go idiomatic approach:
```go
func TestApplyPatch(t *testing.T) {
    tests := []struct {
        name    string
        patch   string
        want    string
        wantErr bool
    }{
        {
            name: "simple update",
            patch: `*** Begin Patch
*** Update File: test.txt
@@
-old
+new
*** End Patch`,
            want:    "new\n",
            wantErr: false,
        },
        // More test cases...
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### Benchmark Tests

Performance testing:
```bash
go test -bench=. ./internal/filesearch/
go test -bench=. ./internal/patchapply/
```

---

## Performance Characteristics

| Package | Operation | Time Complexity | Space Complexity |
|---------|-----------|----------------|------------------|
| patchapply | Parse | O(n) | O(n) |
| patchapply | Apply | O(n*m) | O(n) |
| filesearch | Index | O(n) | O(n) |
| filesearch | Search | O(n*log(k)) | O(k) |
| git | Status | O(n) | O(n) |
| git | Diff | O(n) | O(n) |

*n = file size/count, m = context search window, k = result limit*

---

## Security Considerations

### Path Traversal Prevention

**All packages validate paths using `pkg/pathutil`:**
```go
// GOOD: Relative path within workspace
"src/main.go" ✓

// BAD: Absolute path
"/etc/passwd" ✗

// BAD: Parent directory escape
"../../../etc/passwd" ✗

// BAD: Symlink escape
"link_to_etc/passwd" ✗ (if link points outside workspace)
```

### Input Validation

- **patchapply:** Validates all paths before file operations
- **filesearch:** Filters results to workspace only
- **git:** Uses safe command invocation with proper escaping

### Command Injection Prevention

```go
// SAFE: Using exec.Command with separate args
cmd := exec.Command("git", "status", "--porcelain")

// NEVER: Using shell string concatenation
// cmd := exec.Command("sh", "-c", "git status " + userInput)
```

---

## Best Practices

### For Developers

1. **Use `pkg/pathutil` Always:** Never manipulate paths without validation
2. **Error Wrapping:** Use `fmt.Errorf` with `%w` for error chains
3. **Context Propagation:** Pass `context.Context` for cancellation
4. **Table-Driven Tests:** Use Go's idiomatic testing patterns
5. **Documentation:** Write godoc comments for all exported types

### For AI Models

1. **Context Matters:** Provide sufficient context in patches (3+ lines)
2. **Be Specific:** Use `@@` headers for ambiguous locations
3. **One Change At A Time:** Break large changes into multiple patches
4. **Verify Paths:** Double-check file paths before generating patches
5. **Go Conventions:** Follow Go style (gofmt, effective go)

---

## Project Layout

Following [golang-standards/project-layout](https://github.com/golang-standards/project-layout):

```
spin/
├── cmd/
│   ├── spin/                  # Main application
│   ├── spin-apply-patch/      # Patch application tool
│   └── spin-file-search/      # File search tool
├── internal/
│   ├── patchapply/            # Patch application (internal only)
│   ├── filesearch/            # File searching (internal only)
│   ├── git/                   # Git operations (internal only)
│   └── gitpatch/              # Git patch handling (internal only)
├── pkg/
│   ├── pathutil/              # Path utilities (public API)
│   ├── strutil/               # String utilities (public API)
│   └── ansi/                  # ANSI handling (public API)
├── test/
│   └── fixtures/              # Test fixtures
├── go.mod
├── go.sum
└── README.md
```

**Key Principles:**
- `internal/` - Private packages, cannot be imported by external projects
- `pkg/` - Public packages, safe for external use
- `cmd/` - Executable entry points

---

## Dependencies

### Standard Library Only (Preferred)
- `io/fs`, `path/filepath`, `os`, `bufio`
- `context`, `sync`
- `os/exec`
- `strings`, `regexp`

### Third-Party (Minimal)
- `github.com/bmatcuk/doublestar/v4` - Gitignore pattern matching
- Consider: `github.com/go-git/go-git/v5` - Pure Go git (optional alternative to shelling out)

### Philosophy
- Prefer standard library when possible
- Minimize dependencies for security and maintenance
- No vendor lock-in (compatible with ollama, lmstudio, etc.)

---

## CLI Tools

### spin-apply-patch

```bash
# Apply patch from stdin
cat changes.patch | spin-apply-patch

# Apply with dry-run
spin-apply-patch --dry-run < changes.patch

# Apply with backup
spin-apply-patch --backup < changes.patch
```

### spin-file-search

```bash
# Search for files
spin-file-search config

# Limit results
spin-file-search --limit 20 handler

# JSON output
spin-file-search --json test
```

---

## Future Enhancements

### patchapply
- [ ] Partial patch application (apply some hunks, skip others)
- [ ] Interactive conflict resolution
- [ ] Undo/redo stack with history
- [ ] Streaming patch application for large files

### filesearch
- [ ] Content-based search using ripgrep integration
- [ ] Semantic search with embeddings
- [ ] Smart ranking (recent files, commonly edited, git history)
- [ ] Regex pattern support

### git
- [ ] Branch creation/switching operations
- [ ] Commit operations with signing
- [ ] Stash management
- [ ] Submodule support
- [ ] Pure Go implementation (no git command dependency)

---

## Conclusion

The tools and utilities packages provide the foundation for Spin's file manipulation and project analysis capabilities. By following Go best practices, maintaining clear boundaries with proper use of `internal/` and `pkg/`, comprehensive path validation, and extensive testing, these packages enable safe and efficient code modifications by autonomous AI agents.

The design philosophy emphasizes:
- **Simplicity:** Following KISS and DRY principles
- **Safety:** Comprehensive input validation and error handling
- **Performance:** Efficient algorithms with proper concurrency
- **Testability:** Table-driven tests and clear interfaces
- **Maintainability:** Clean architecture and effective Go patterns
- **Openness:** No vendor lock-in, compatible with any LLM provider

