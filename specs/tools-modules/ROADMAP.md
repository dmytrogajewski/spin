# Tools & Utility Modules Implementation Roadmap

**Project:** Spin - Golang AI Coding Agent
**Spec:** [tools-modules.md](tools-modules.md)
**Created:** 2025-10-11
**Status:** Planning

---

## Overview

This roadmap details the implementation of core tool and utility modules that provide file manipulation, search, Git operations, and string/path utilities for the Spin AI coding agent.

**Current State:**
- ✅ `internal/filesearch` exists (basic scanner + fuzzy matcher)
- ✅ `internal/ui/term/ansi` exists (basic ANSI handling)
- ❌ `internal/patchapply` - NOT implemented
- ❌ `internal/git` - NOT implemented
- ❌ `internal/gitpatch` - NOT implemented
- ❌ `pkg/pathutil` - NOT implemented
- ❌ `pkg/strutil` - NOT implemented
- ❌ Complete `pkg/ansi` - NOT implemented

**Implementation Strategy:**
1. Foundation first: `pkg/pathutil`, `pkg/strutil`, `pkg/ansi` (reusable utilities)
2. Core functionality: `internal/patchapply` (critical for file modifications)
3. Enhanced search: Upgrade `internal/filesearch` with gitignore support
4. Git integration: `internal/git`, `internal/gitpatch`
5. CLI tools and documentation

---

## Phase 1: Foundation - Public Utility Packages (pkg/)

### Feature 1.1: pkg/pathutil - Path Utilities and Validation

**Priority:** P0 (Blocker for all file operations)
**Estimated Effort:** 2-3 days

#### Definition of Ready (DoR)
- [ ] Read and understand path security requirements
- [ ] Review path traversal attack vectors
- [ ] Study Go's `filepath` package capabilities
- [ ] Review existing Spin path handling code

#### Implementation Tasks
1. **Create package structure**
   - `pkg/pathutil/pathutil.go` - Main implementation
   - `pkg/pathutil/pathutil_test.go` - Comprehensive tests
   - `pkg/pathutil/doc.go` - Package documentation

2. **Core Functions**
   ```go
   - ValidateRelativePath(path string) error
   - SafeJoin(root, relPath string) (string, error)
   - NormalizePath(path string) string
   - RelativePath(root, path string) (string, error)
   - IsWithinRoot(root, path string) bool
   ```

3. **Security Features**
   - Detect and reject absolute paths
   - Detect and reject `..` traversal
   - Symlink validation (ensure doesn't escape workspace)
   - Canonical path resolution

4. **Error Types**
   ```go
   - ErrAbsolutePath
   - ErrPathTraversal
   - ErrEmptyPath
   - ErrSymlinkEscape
   ```

#### Definition of Done (DoD)
- [ ] All functions implemented with godoc comments
- [ ] Table-driven tests covering:
  - Valid relative paths
  - Absolute path rejection
  - Parent directory traversal rejection (`../`, `../../`)
  - Hidden traversal patterns (`foo/../../../etc`)
  - Symlink escape attempts
  - Edge cases (empty, `.`, `./`, trailing slashes)
- [ ] Test coverage ≥95%
- [ ] `make lint` passes (zero errors)
- [ ] Race detector clean: `go test -race ./pkg/pathutil`
- [ ] Benchmark tests for performance-critical functions
- [ ] Package documentation in `docs/packages/pathutil.md`

#### Acceptance Criteria
- Can validate any path safely without crashes
- Rejects all known path traversal vectors
- Performance: <1μs for typical path validation
- Zero allocations in hot path (NormalizePath)

---

### Feature 1.2: pkg/strutil - String Manipulation Utilities

**Priority:** P1 (Required for patchapply)
**Estimated Effort:** 2-3 days

#### Definition of Ready (DoR)
- [ ] Review string manipulation needs from spec
- [ ] Study Go's `strings` and `unicode` packages
- [ ] Identify performance-critical operations

#### Implementation Tasks
1. **Line Operations**
   ```go
   - SplitLines(text string) []string
   - JoinLines(lines []string) string
   - TrimEmptyLines(lines []string) []string
   ```

2. **Indentation Detection**
   ```go
   - DetectIndentation(text string) (useTabs bool, size int)
   - NormalizeIndentation(text string, useTabs bool, size int) string
   ```

3. **Whitespace Handling**
   ```go
   - NormalizeWhitespace(text string) string
   - TrimWhitespace(text string) string
   ```

4. **Similarity Algorithms**
   ```go
   - LevenshteinDistance(a, b string) int
   - Similarity(a, b string) float64
   - FuzzyMatch(query, target string) float64
   ```

5. **Case Utilities**
   ```go
   - ToSnakeCase(s string) string
   - ToCamelCase(s string) string
   - ToPascalCase(s string) string
   ```

#### Definition of Done (DoD)
- [ ] All functions implemented with godoc
- [ ] Table-driven tests for each function
- [ ] Edge case coverage:
  - Empty strings
  - Unicode characters
  - Mixed line endings (CRLF, LF, CR)
  - Very long strings (>1MB)
- [ ] Test coverage ≥90%
- [ ] `make lint` passes
- [ ] Race detector clean
- [ ] Benchmark tests showing:
  - LevenshteinDistance: O(n*m) time
  - SplitLines: O(n) time
  - Similarity: <100μs for typical strings
- [ ] Package documentation in `docs/packages/strutil.md`

#### Acceptance Criteria
- Handles all line ending formats correctly
- Indentation detection accuracy ≥95% on real codebases
- Similarity function works for fuzzy matching
- No panics on malformed input

---

### Feature 1.3: pkg/ansi - ANSI Escape Sequence Handling

**Priority:** P2 (Required for TUI polish)
**Estimated Effort:** 2 days

#### Definition of Ready (DoR)
- [ ] Review existing `internal/ui/term/ansi.go`
- [ ] Study ANSI escape sequence specification
- [ ] Identify public API surface

#### Implementation Tasks
1. **Move and refactor existing code**
   - Extract from `internal/ui/term/ansi.go`
   - Create public API in `pkg/ansi/`
   - Maintain backward compatibility

2. **Core Functions**
   ```go
   - Strip(text string) string
   - Length(text string) int  // Visual length excluding escapes
   - Wrap(text string, width int) []string
   ```

3. **Color Constants**
   ```go
   - Reset, Black, Red, Green, Yellow, Blue, Magenta, Cyan, White
   - Bold, Dim, Italic, Underline
   ```

4. **Style Builder**
   ```go
   type Style struct { ... }
   - New(text string) *Style
   - Red(), Green(), Yellow(), Blue(), Bold(), Underline()
   - String() string
   ```

5. **Parser**
   ```go
   type Segment struct { ... }
   - Parse(text string) []Segment
   ```

#### Definition of Done (DoD)
- [ ] All functions implemented with godoc
- [ ] Tests covering:
  - Strip removes all ANSI codes
  - Length returns correct visual length
  - Style builder chains correctly
  - Parser handles nested styles
- [ ] Test coverage ≥90%
- [ ] `make lint` passes
- [ ] Race detector clean
- [ ] Migrate `internal/ui/term` to use `pkg/ansi`
- [ ] Package documentation in `docs/packages/ansi.md`

#### Acceptance Criteria
- Strip removes all ANSI sequences correctly
- Style builder produces valid ANSI codes
- Parser handles complex terminal output
- No breaking changes to existing TUI

---

## Phase 2: Core File Manipulation - internal/patchapply

### Feature 2.1: Patch Parser

**Priority:** P0 (Critical for file modifications)
**Estimated Effort:** 3-4 days

#### Definition of Ready (DoR)
- [ ] Read patch format specification from spec
- [ ] Design AST for patch operations
- [ ] Define error messages for parse failures
- [ ] Review Go parsing patterns (bufio.Scanner)

#### Implementation Tasks
1. **Types (`types.go`)**
   ```go
   type Patch struct { Operations []FileOperation }
   type FileOperation interface { isFileOperation(); Path() string }
   type AddFile struct { FilePath string; Lines []string }
   type DeleteFile struct { FilePath string }
   type UpdateFile struct { FilePath string; NewPath string; Hunks []Hunk }
   type Hunk struct { Header string; Changes []LineChange }
   type LineChange struct { Type LineChangeType; Text string }
   type LineChangeType int  // Context, Delete, Insert
   ```

2. **Parser (`parser.go`)**
   ```go
   type Parser struct { ... }
   - NewParser(text string) *Parser
   - Parse() (*Patch, error)
   - parseOperation(line string) (FileOperation, error)
   - parseAddFile(path string) (*AddFile, error)
   - parseDeleteFile(path string) (*DeleteFile, error)
   - parseUpdateFile(path string) (*UpdateFile, error)
   - parseHunk() (*Hunk, error)
   ```

3. **Validation**
   - Syntax validation (Begin/End markers)
   - Path validation (use `pkg/pathutil`)
   - Context validation (minimum context lines)
   - Hunk validation (balanced +/- lines)

#### Definition of Done (DoD)
- [ ] Parser handles all patch operations:
  - `*** Add File:`
  - `*** Delete File:`
  - `*** Update File:` with hunks
  - `*** Move to:` (rename operation)
- [ ] Comprehensive error messages with line numbers
- [ ] Table-driven tests for:
  - Valid patches (all operation types)
  - Syntax errors (missing markers, invalid format)
  - Edge cases (empty files, large patches)
- [ ] Test coverage ≥95%
- [ ] `make lint` passes
- [ ] Complexity ≤10 per function (gocyclo)

#### Acceptance Criteria
- Parses all patch formats from spec
- Clear error messages with line/column info
- No panics on malformed input
- Handles patches >10,000 lines efficiently

---

### Feature 2.2: Fuzzy Matcher

**Priority:** P0 (Critical for robust patch application)
**Estimated Effort:** 3 days

#### Definition of Ready (DoR)
- [ ] Review fuzzy matching algorithms
- [ ] Study diff3/patch algorithms
- [ ] Define similarity threshold requirements

#### Implementation Tasks
1. **Matcher (`matcher.go`)**
   ```go
   type Matcher struct { ... }
   - NewMatcher(content []string) *Matcher
   - FindContext(contextLines []string) int
   - SetThreshold(threshold float64)
   - similarity(a, b []string) float64
   - linesMatch(a, b string) bool
   ```

2. **Matching Strategies**
   - Exact match (first priority)
   - Fuzzy match with whitespace normalization
   - Context header matching (`@@` markers)
   - Multi-occurrence handling

3. **Similarity Algorithm**
   - Line-by-line comparison
   - Configurable threshold (default 85%)
   - Whitespace tolerance
   - Performance optimization (early exit)

#### Definition of Done (DoD)
- [ ] Implements sliding window search
- [ ] Handles multiple occurrences (uses context headers)
- [ ] Whitespace-tolerant matching
- [ ] Tests covering:
  - Exact match success
  - Fuzzy match within threshold
  - Multiple occurrences with disambiguation
  - Context not found scenarios
- [ ] Test coverage ≥90%
- [ ] Benchmark: <1ms for 10k line files
- [ ] `make lint` passes

#### Acceptance Criteria
- Finds context in 99% of real-world cases
- Handles indentation changes gracefully
- Disambiguates multiple occurrences correctly
- Performance acceptable for large files (<100ms for 100k lines)

---

### Feature 2.3: Patch Applier

**Priority:** P0 (Critical - brings everything together)
**Estimated Effort:** 4-5 days

#### Definition of Ready (DoR)
- [ ] Complete Feature 2.1 (Parser)
- [ ] Complete Feature 2.2 (Matcher)
- [ ] Complete Feature 1.1 (pathutil)
- [ ] Design rollback strategy

#### Implementation Tasks
1. **Applier (`applier.go`)**
   ```go
   type Applier struct { ... }
   - NewApplier(workspaceRoot string) *Applier
   - Apply(patch *Patch) error
   - SetDryRun(enabled bool)
   - SetBackup(enabled bool)
   - validatePath(relPath string) error
   - applyOperation(op FileOperation) error
   - applyAddFile(op *AddFile) error
   - applyDeleteFile(op *DeleteFile) error
   - applyUpdateFile(op *UpdateFile) error
   ```

2. **Safety Features**
   - Path validation (workspace confinement)
   - Backup creation before modifications
   - Atomic operations (all-or-nothing)
   - Dry-run mode (preview changes)

3. **File Operations**
   - Add: Create file with content
   - Delete: Remove file
   - Update: Apply hunks with fuzzy matching
   - Move: Rename file (during update)

4. **Error Handling**
   - Context not found errors (detailed)
   - File not found errors
   - Permission errors
   - Partial application errors

#### Definition of Done (DoD)
- [ ] All operations implemented
- [ ] Dry-run mode works correctly
- [ ] Backup creation optional
- [ ] Comprehensive tests:
  - Successful application (all operations)
  - Path traversal attempts (rejected)
  - Context not found (clear errors)
  - Permission errors (handled)
  - Atomic rollback on failure
- [ ] Test coverage ≥90%
- [ ] E2E tests with real patch files
- [ ] `make lint` passes
- [ ] Complexity ≤15 (refactor if needed)

#### Acceptance Criteria
- Applies patches successfully in normal cases
- Rejects all path traversal attempts
- Provides detailed error messages on failure
- Dry-run accurately predicts outcome
- Backup/restore works correctly

---

### Feature 2.4: patchapply CLI Tool

**Priority:** P1 (Useful for testing and manual use)
**Estimated Effort:** 1 day

#### Definition of Ready (DoR)
- [ ] Complete Feature 2.3 (Applier)
- [ ] Review CLI patterns in existing Spin tools

#### Implementation Tasks
1. **CLI Tool (`cmd/spin-apply-patch/main.go`)**
   ```go
   func main() {
     flags: --dry-run, --backup, --workspace
     read from stdin or file
     apply patch
     print results
   }
   ```

2. **Features**
   - Read patch from stdin or file
   - Workspace directory flag
   - Dry-run mode
   - Verbose output

#### Definition of Done (DoD)
- [ ] CLI tool implemented
- [ ] Help text written
- [ ] Manual testing with various patches
- [ ] Error handling and exit codes
- [ ] Integration test script
- [ ] Documentation in `docs/cli/spin-apply-patch.md`

#### Acceptance Criteria
- Works with stdin: `cat patch.txt | spin-apply-patch`
- Works with file: `spin-apply-patch -f patch.txt`
- Dry-run shows what would change
- Clear error messages

---

## Phase 3: Enhanced File Search - internal/filesearch

### Feature 3.1: Gitignore Handler

**Priority:** P1 (Essential for production use)
**Estimated Effort:** 2-3 days

#### Definition of Ready (DoR)
- [ ] Review existing `internal/filesearch/scanner.go`
- [ ] Study gitignore pattern syntax
- [ ] Evaluate `doublestar` library

#### Implementation Tasks
1. **IgnoreHandler (`ignore.go`)**
   ```go
   type IgnoreHandler struct { ... }
   - NewIgnoreHandler(root string) (*IgnoreHandler, error)
   - loadIgnoreFile(path string) error
   - IsIgnored(path string, isDir bool) bool
   ```

2. **Pattern Support**
   - Load `.gitignore`
   - Load `.spinignore`
   - Default patterns (`.git/`, `node_modules/`, `.spin/`)
   - Directory-specific ignores

3. **Integration**
   - Update `Scanner` to use `IgnoreHandler`
   - Respect ignore rules during traversal

#### Definition of Done (DoD)
- [ ] Loads `.gitignore` and `.spinignore`
- [ ] Supports glob patterns via `doublestar`
- [ ] Tests covering:
  - Basic patterns (`*.log`, `*.tmp`)
  - Directory patterns (`build/`, `dist/`)
  - Negation patterns (`!important.log`)
  - Nested gitignores
- [ ] Test coverage ≥90%
- [ ] Benchmark: minimal overhead on large repos
- [ ] `make lint` passes

#### Acceptance Criteria
- Respects all gitignore patterns correctly
- Handles nested gitignore files
- Performance acceptable (<100ms for 10k files)

---

### Feature 3.2: Advanced Search and Ranking

**Priority:** P2 (Nice to have)
**Estimated Effort:** 2 days

#### Definition of Ready (DoR)
- [ ] Complete Feature 3.1 (IgnoreHandler)
- [ ] Review search scoring algorithm from spec

#### Implementation Tasks
1. **Enhanced Searcher (`searcher.go`)**
   ```go
   type Searcher struct { ... }
   - NewSearcher(root string) (*Searcher, error)
   - IndexAsync(ctx context.Context) error
   - Search(query string, limit int) []SearchResult
   - scoreMatch(query string, entry FileEntry) float64
   ```

2. **Scoring Algorithm**
   - Exact filename match: 100 points
   - Filename starts with query: 90 points
   - Filename contains query: 80-70 points (position-weighted)
   - Path contains query: 60-50 points
   - Fuzzy match: 40+ points

3. **Async Indexing**
   - Context-aware traversal
   - Cancellation support
   - Progress reporting

#### Definition of Done (DoD)
- [ ] Implements advanced scoring
- [ ] Async indexing with context
- [ ] Tests covering:
  - Various query patterns
  - Ranking accuracy
  - Cancellation
- [ ] Test coverage ≥85%
- [ ] Benchmark: <10ms for 100k file index search
- [ ] `make lint` passes

#### Acceptance Criteria
- Search results ranked accurately
- Async indexing cancellable
- Performance meets targets

---

### Feature 3.3: filesearch CLI Tool

**Priority:** P2 (Useful for testing)
**Estimated Effort:** 1 day

#### Implementation Tasks
1. **CLI Tool (`cmd/spin-file-search/main.go`)**
   ```bash
   spin-file-search <query> [--limit N] [--json]
   ```

#### Definition of Done (DoD)
- [ ] CLI implemented with flags
- [ ] JSON output option
- [ ] Help text and examples
- [ ] Manual testing
- [ ] Documentation

---

## Phase 4: Git Integration

### Feature 4.1: internal/git - Repository Operations

**Priority:** P1 (Required for context gathering)
**Estimated Effort:** 3-4 days

#### Definition of Ready (DoR)
- [ ] Study `go-git` library vs shelling out
- [ ] Define required Git operations
- [ ] Review security considerations

#### Implementation Tasks
1. **Repository (`git.go`)**
   ```go
   type Repository struct { ... }
   - Discover(startPath string) (*Repository, error)
   - Root() string
   - Status(ctx context.Context) (*Status, error)
   - CurrentBranch(ctx context.Context) (string, error)
   - ListBranches(ctx context.Context) ([]string, error)
   - DiffToBranch(ctx context.Context, branch string) (*Diff, error)
   - RemoteURL(ctx context.Context, remote string) (string, error)
   ```

2. **Status Parsing**
   - Parse `git status --porcelain`
   - Extract branch info
   - Extract ahead/behind counts
   - File status (modified, untracked)

3. **Safety**
   - Use `exec.CommandContext` (no shell injection)
   - Validate all inputs
   - Proper error handling

#### Definition of Done (DoD)
- [ ] All operations implemented
- [ ] Tests with test repositories:
  - Status parsing accuracy
  - Branch detection
  - Diff parsing
- [ ] Test coverage ≥85%
- [ ] `make lint` passes
- [ ] Race detector clean
- [ ] Documentation in `docs/packages/git.md`

#### Acceptance Criteria
- Discovers Git repositories correctly
- Parses status accurately
- Handles non-Git directories gracefully

---

### Feature 4.2: internal/gitpatch - Git Patch Application

**Priority:** P2 (Useful for external patches)
**Estimated Effort:** 2 days

#### Implementation Tasks
1. **Applier (`gitpatch.go`)**
   ```go
   type Applier struct { ... }
   - NewApplier(workspaceRoot string) *Applier
   - Apply(patchText string) error
   - Check(patchText string) error
   ```

2. **Implementation**
   - Write patch to temp file
   - Use `git apply` command
   - Check mode with `git apply --check`

#### Definition of Done (DoD)
- [ ] Applies Git unified diffs
- [ ] Check mode validates without applying
- [ ] Tests with real Git patches
- [ ] Test coverage ≥85%
- [ ] `make lint` passes
- [ ] Documentation

#### Acceptance Criteria
- Applies standard Git patches successfully
- Check mode works correctly
- Clear error messages on failure

---

## Phase 5: Integration and Polish

### Feature 5.1: Core Integration

**Priority:** P0 (Essential for production)
**Estimated Effort:** 2-3 days

#### Implementation Tasks
1. **Integrate patchapply into Core**
   - Update tool registry
   - Add patch application tool
   - Wire up AI -> Core -> patchapply

2. **Integrate filesearch into Core**
   - Update file search tool
   - Add real-time indexing

3. **Integrate git into Core**
   - Context gathering (branch, status)
   - Include in system prompt

#### Definition of Done (DoD)
- [ ] All packages integrated
- [ ] E2E tests with AI generating patches
- [ ] Test coverage maintained
- [ ] `make lint` passes
- [ ] Manual testing with real scenarios

---

### Feature 5.2: Documentation

**Priority:** P1 (Required for maintenance)
**Estimated Effort:** 2 days

#### Implementation Tasks
1. **Package Documentation**
   - `docs/packages/patchapply.md`
   - `docs/packages/filesearch.md`
   - `docs/packages/git.md`
   - `docs/packages/gitpatch.md`
   - Update `docs/packages/README.md`

2. **Examples**
   - Usage examples in each doc
   - CLI tool examples
   - Integration examples

3. **Architecture Documentation**
   - Update architecture overview
   - Add data flow diagrams

#### Definition of Done (DoD)
- [ ] All package docs written
- [ ] Examples tested and working
- [ ] Architecture docs updated
- [ ] README.md updated

---

### Feature 5.3: E2E Testing and Hardening

**Priority:** P0 (Production readiness)
**Estimated Effort:** 3-4 days

#### Implementation Tasks
1. **E2E Test Suite**
   - Real AI patch generation and application
   - File search in real projects
   - Git context gathering
   - Error scenarios

2. **Hardening**
   - Security audit (path traversal, injection)
   - Performance optimization
   - Error message improvement
   - Edge case handling

3. **Chaos Testing**
   - Concurrent patch application
   - Large file handling (>100k lines)
   - Deep directory trees
   - Permission errors
   - Disk full scenarios

#### Definition of Done (DoD)
- [ ] E2E test suite passing
- [ ] Security audit complete (no vulns)
- [ ] Performance benchmarks met
- [ ] Chaos tests passing
- [ ] Production deployment checklist

---

## Dependencies and Sequencing

```
Phase 1 (Foundation)
├── 1.1 pathutil ───┐
├── 1.2 strutil ────┤
└── 1.3 ansi ───────┘
        │
        ↓
Phase 2 (patchapply)
├── 2.1 Parser ─────┐
├── 2.2 Matcher ────┤→ 2.3 Applier → 2.4 CLI
└── Depends on 1.1, 1.2
        │
        ↓
Phase 3 (filesearch)
├── 3.1 Gitignore ──→ 3.2 Search → 3.3 CLI
└── Can run parallel with Phase 4
        │
        ↓
Phase 4 (git)
├── 4.1 Repository ─→ 4.2 GitPatch
└── Can run parallel with Phase 3
        │
        ↓
Phase 5 (Integration)
└── 5.1 Core → 5.2 Docs → 5.3 E2E/Hardening
```

---

## Success Metrics

### Code Quality
- [ ] Test coverage ≥85% overall
- [ ] Test coverage ≥90% for critical paths (patchapply, pathutil)
- [ ] Zero `make lint` errors
- [ ] Cyclomatic complexity ≤15 per function
- [ ] Race detector clean on all packages

### Performance
- [ ] Path validation: <1μs per operation
- [ ] Patch parsing: <100ms for 10k line patch
- [ ] Patch application: <1s for typical patch
- [ ] File search indexing: <1s for 10k files
- [ ] File search query: <10ms per query

### Security
- [ ] All path traversal vectors blocked
- [ ] No command injection vulnerabilities
- [ ] Safe concurrency (no data races)
- [ ] Input validation on all external data

### Reliability
- [ ] E2E tests passing consistently
- [ ] No crashes on malformed input
- [ ] Graceful error handling
- [ ] Clear error messages

---

## Risk Management

### High Risk Items
1. **Fuzzy matching accuracy** - May not find context in edge cases
   - Mitigation: Extensive testing, configurable threshold, fallback to exact match

2. **Path security** - Path traversal vulnerabilities
   - Mitigation: Defense in depth (pathutil + applier validation), security audit

3. **Performance** - Large files/repos may be slow
   - Mitigation: Benchmarking, optimization, streaming where possible

4. **Git compatibility** - Different Git versions may behave differently
   - Mitigation: Test with multiple Git versions, use stable commands

### Medium Risk Items
1. **Gitignore pattern complexity** - Edge cases may not work
   - Mitigation: Use well-tested library (doublestar), extensive tests

2. **Integration complexity** - Wiring into Core may surface issues
   - Mitigation: Incremental integration, thorough E2E testing

---

## Delivery Schedule

**Phase 1:** Days 1-5 (1 week)
**Phase 2:** Days 6-15 (2 weeks)
**Phase 3:** Days 16-20 (1 week)
**Phase 4:** Days 21-25 (1 week)
**Phase 5:** Days 26-32 (1.5 weeks)

**Total Estimated Duration:** 6-7 weeks

**Milestone Dates:**
- Week 2: Foundation complete (Phase 1)
- Week 4: Patchapply working (Phase 2)
- Week 5: Search enhanced (Phase 3)
- Week 6: Git integrated (Phase 4)
- Week 7: Production ready (Phase 5)

---

## Notes

- Follow **ALL 14 steps** from AGENTS.md workflow
- Read **docs/** before each feature
- Write **FRD** for each feature (specs/frds/FRD-{datetime}.md)
- **TDD**: Tests first, then implementation
- **Analysis**: Run `uast parse | herr analyze` after each implementation
- **Quality gates**: Must pass before moving to next feature
- No feature merges without **e2e coverage**

---

**Last Updated:** 2025-10-11
**Next Review:** After Phase 1 completion
