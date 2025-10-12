# FRD-20251012110000: Git Repository Operations

**Feature:** internal/git - Git Repository Operations
**Priority:** P1 (Required for context gathering)
**Status:** Draft
**Created:** 2025-10-12
**Epic:** Phase 4: Git Integration (tools-modules roadmap)

---

## Overview

Implement `internal/git` package to provide Git repository operations and information gathering for the Spin AI coding agent using the `go-git` pure Go library. This package enables Spin to understand the Git context of the workspace, including current branch, status, commit history, and diff operations.

## Motivation

### Problem Statement

Spin needs to understand the Git context of the workspace to:
1. Provide AI with awareness of current branch and uncommitted changes
2. Include repository status in system prompts for better context
3. Enable intelligent suggestions based on Git state
4. Support future features like automated commit operations
5. Gather repository information for pull request context

### Goals

- **G1**: Discover Git repositories starting from any directory
- **G2**: Query repository status (branch, modified/untracked files, ahead/behind)
- **G3**: List branches and get current branch
- **G4**: Compute diffs between branches or commits
- **G5**: Query remote repository URLs
- **G6**: Pure Go implementation (no shell commands)
- **G7**: Context-aware operations (cancellable)
- **G8**: Clear error messages for non-Git directories

### Non-Goals

- **NG1**: Direct Git write operations (commit, push, pull) - deferred to future
- **NG2**: Submodule management - deferred to future
- **NG3**: Branch creation/switching - deferred to future
- **NG4**: Stash operations - deferred to future
- **NG5**: Git LFS support - deferred to future

## Requirements

### Functional Requirements

**FR1: Repository Discovery**
- Must discover Git repository by walking up directory tree from startPath
- Must find `.git` directory and return repository root
- Must return clear error if not a Git repository
- Must handle symlinks safely
- Must work with both `.git` directory and `.git` file (worktrees)

**FR2: Repository Status**
- Must return current branch name
- Must return remote branch name if tracking
- Must return ahead/behind commit counts
- Must return list of modified files (working tree changes)
- Must return list of staged files
- Must return list of untracked files
- Must handle detached HEAD state

**FR3: Branch Operations**
- Must return current branch name
- Must list all local branches
- Must list remote branches
- Must handle detached HEAD state
- Must return empty list if no branches (bare repo)

**FR4: Diff Operations**
- Must compute diff to specified branch
- Must compute diff to specified commit
- Must return file changes with status (A/M/D/R)
- Must handle empty diffs gracefully
- Must support working tree diffs

**FR5: Remote Operations**
- Must return remote URL for specified remote (default: "origin")
- Must handle repositories without remotes
- Must return clear error if remote doesn't exist
- Must list all configured remotes

**FR6: Safety & Security**
- Must use pure Go (no shell invocation)
- Must respect context cancellation
- Must validate all inputs
- Must never expose credentials in errors or logs
- Must operate within repository boundaries

### Non-Functional Requirements

**NFR1: Performance**
- Repository discovery: <10ms for typical depth (<10 directories)
- Status query: <100ms for typical repository (<1000 files)
- Diff operations: <200ms for typical diff (<500 files)
- Memory efficient (no full tree loading unless needed)

**NFR2: Reliability**
- Must handle repository state gracefully
- Must provide clear, actionable error messages
- Must work with Git repositories created by Git 2.0+
- Must not corrupt repository state
- Must be safe for concurrent read operations

**NFR3: Testability**
- All operations must be testable with mock Git repositories
- Must support integration tests with real Git
- Must achieve ≥85% test coverage

**NFR4: Maintainability**
- Clear package structure following Go conventions
- Comprehensive godoc comments on all exports
- Follow Spin's error handling patterns
- Minimal external dependencies

## Design

### Architecture

```
internal/git/
├── git.go           # Main package interface (Repository type)
├── discover.go      # Repository discovery
├── status.go        # Status operations
├── branch.go        # Branch operations
├── diff.go          # Diff operations
├── remote.go        # Remote operations
├── types.go         # Shared types (Status, Diff, FileChange, etc.)
├── errors.go        # Error types and messages
├── doc.go           # Package documentation
├── git_test.go      # Main tests
├── status_test.go   # Status tests
├── branch_test.go   # Branch operation tests
├── diff_test.go     # Diff tests
└── remote_test.go   # Remote operation tests
```

### Dependencies

**Primary Dependency:**
```go
import (
    "github.com/go-git/go-git/v5"
    "github.com/go-git/go-git/v5/plumbing"
    "github.com/go-git/go-git/v5/plumbing/object"
    "github.com/go-git/go-git/v5/plumbing/storer"
)
```

**Why go-git:**
- Pure Go implementation (no git binary dependency)
- Complete Git implementation
- Well-tested and maintained
- Used by major projects (GitHub CLI, etc.)
- Better error handling and type safety
- Cross-platform compatibility guaranteed

### Core Types

```go
package git

import (
    "context"
    "time"

    git "github.com/go-git/go-git/v5"
    "github.com/go-git/go-git/v5/plumbing"
)

// Repository represents a Git repository
type Repository struct {
    repo *git.Repository // go-git repository
    root string          // Absolute path to repository root
}

// Status represents repository status
type Status struct {
    Branch         string       // Current branch name
    RemoteBranch   string       // Upstream branch (e.g., "origin/main")
    Ahead          int          // Commits ahead of remote
    Behind         int          // Commits behind remote
    ModifiedFiles  []FileStatus // Modified/staged files
    UntrackedFiles []string     // Untracked files
    Detached       bool         // True if in detached HEAD state
    Hash           string       // Current commit hash
}

// FileStatus represents a file's status
type FileStatus struct {
    Path    string     // File path
    Staging StatusCode // Staging area status
    Worktree StatusCode // Working tree status
}

// StatusCode represents file status
type StatusCode int

const (
    Unmodified StatusCode = iota
    Modified
    Added
    Deleted
    Renamed
    Copied
    Untracked
)

// Diff represents changes between commits/branches
type Diff struct {
    From  string       // Source ref/commit
    To    string       // Target ref/commit
    Files []FileChange // Changed files
}

// FileChange represents a single file change
type FileChange struct {
    Status  string // A=added, M=modified, D=deleted, R=renamed, C=copied
    Path    string // File path (relative to repo root)
    OldPath string // Old path (for renames/copies)
    Patch   string // Optional: unified diff patch
}

// BranchInfo represents branch information
type BranchInfo struct {
    Name     string // Branch name (short form: "main")
    FullName string // Full ref name: "refs/heads/main"
    Hash     string // Current commit hash
    Remote   string // Upstream remote (if tracking)
}

// RemoteInfo represents remote information
type RemoteInfo struct {
    Name string   // Remote name (e.g., "origin")
    URLs []string // Remote URLs (fetch/push)
}

// Error types
var (
    ErrNotRepository   = errors.New("not a git repository")
    ErrInvalidRemote   = errors.New("remote not found")
    ErrInvalidBranch   = errors.New("branch not found")
    ErrInvalidRef      = errors.New("invalid reference")
    ErrDetachedHead    = errors.New("detached HEAD state")
)
```

### API Design

```go
// Discover finds a git repository starting from the given path
// Walks up directory tree until .git directory is found
func Discover(ctx context.Context, startPath string) (*Repository, error)

// DiscoverWithContext same as Discover, respects context cancellation
func DiscoverWithContext(ctx context.Context, startPath string) (*Repository, error)

// Root returns the repository root path
func (r *Repository) Root() string

// Status returns current repository status
func (r *Repository) Status(ctx context.Context) (*Status, error)

// CurrentBranch returns the current branch info
// Returns ErrDetachedHead if in detached HEAD state
func (r *Repository) CurrentBranch(ctx context.Context) (*BranchInfo, error)

// ListBranches returns all local branches
func (r *Repository) ListBranches(ctx context.Context) ([]*BranchInfo, error)

// ListRemoteBranches returns all remote branches
func (r *Repository) ListRemoteBranches(ctx context.Context) ([]*BranchInfo, error)

// DiffToBranch returns diff from current HEAD to specified branch
func (r *Repository) DiffToBranch(ctx context.Context, branch string) (*Diff, error)

// DiffToCommit returns diff from current HEAD to specified commit
func (r *Repository) DiffToCommit(ctx context.Context, commit string) (*Diff, error)

// DiffBetween returns diff between two refs/commits
func (r *Repository) DiffBetween(ctx context.Context, from, to string) (*Diff, error)

// RemoteURL returns the URL for specified remote
func (r *Repository) RemoteURL(ctx context.Context, remoteName string) (string, error)

// ListRemotes returns all configured remotes
func (r *Repository) ListRemotes(ctx context.Context) ([]*RemoteInfo, error)

// Log returns commit log
func (r *Repository) Log(ctx context.Context, ref string, limit int) ([]*CommitInfo, error)

// CommitInfo represents commit information
type CommitInfo struct {
    Hash      string    // Commit hash
    Author    string    // Author name
    Email     string    // Author email
    Message   string    // Commit message
    Timestamp time.Time // Commit timestamp
}
```

### go-git Usage Patterns

**Repository Discovery:**
```go
func Discover(ctx context.Context, startPath string) (*Repository, error) {
    // Normalize path
    absPath, err := filepath.Abs(startPath)
    if err != nil {
        return nil, fmt.Errorf("resolve path: %w", err)
    }

    // Walk up directory tree
    for {
        // Check context cancellation
        select {
        case <-ctx.Done():
            return nil, ctx.Err()
        default:
        }

        // Try to open repository at current path
        repo, err := git.PlainOpen(absPath)
        if err == nil {
            worktree, err := repo.Worktree()
            if err != nil {
                return nil, fmt.Errorf("get worktree: %w", err)
            }

            return &Repository{
                repo: repo,
                root: worktree.Filesystem.Root(),
            }, nil
        }

        // Not a repo, try parent directory
        parent := filepath.Dir(absPath)
        if parent == absPath {
            return nil, fmt.Errorf("%w: %s", ErrNotRepository, startPath)
        }
        absPath = parent
    }
}
```

**Status Query:**
```go
func (r *Repository) Status(ctx context.Context) (*Status, error) {
    worktree, err := r.repo.Worktree()
    if err != nil {
        return nil, fmt.Errorf("get worktree: %w", err)
    }

    // Get worktree status
    gitStatus, err := worktree.Status()
    if err != nil {
        return nil, fmt.Errorf("get status: %w", err)
    }

    // Get current HEAD
    head, err := r.repo.Head()
    if err != nil {
        return &Status{Detached: true}, nil // Detached HEAD
    }

    status := &Status{
        Branch:        head.Name().Short(),
        ModifiedFiles: make([]FileStatus, 0),
        UntrackedFiles: make([]string, 0),
        Hash:          head.Hash().String(),
    }

    // Parse file statuses
    for path, fileStatus := range gitStatus {
        if fileStatus.Worktree == git.Untracked {
            status.UntrackedFiles = append(status.UntrackedFiles, path)
        } else {
            status.ModifiedFiles = append(status.ModifiedFiles, FileStatus{
                Path:     path,
                Staging:  mapGoGitStatus(fileStatus.Staging),
                Worktree: mapGoGitStatus(fileStatus.Worktree),
            })
        }
    }

    // Get tracking branch and ahead/behind
    if head.Name().IsBranch() {
        if remote, err := r.getRemoteBranch(head.Name()); err == nil {
            status.RemoteBranch = remote
            status.Ahead, status.Behind = r.calculateAheadBehind(head.Hash(), remote)
        }
    }

    return status, nil
}
```

**Branch Operations:**
```go
func (r *Repository) ListBranches(ctx context.Context) ([]*BranchInfo, error) {
    branches := make([]*BranchInfo, 0)

    refs, err := r.repo.Branches()
    if err != nil {
        return nil, fmt.Errorf("list branches: %w", err)
    }

    err = refs.ForEach(func(ref *plumbing.Reference) error {
        // Check context cancellation
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
        }

        branch := &BranchInfo{
            Name:     ref.Name().Short(),
            FullName: ref.Name().String(),
            Hash:     ref.Hash().String(),
        }

        // Get remote tracking info
        if remote, err := r.getRemoteBranch(ref.Name()); err == nil {
            branch.Remote = remote
        }

        branches = append(branches, branch)
        return nil
    })

    if err != nil {
        return nil, fmt.Errorf("iterate branches: %w", err)
    }

    return branches, nil
}
```

**Diff Operations:**
```go
func (r *Repository) DiffToBranch(ctx context.Context, branch string) (*Diff, error) {
    // Resolve current HEAD
    head, err := r.repo.Head()
    if err != nil {
        return nil, fmt.Errorf("get HEAD: %w", err)
    }

    // Resolve target branch
    branchRef, err := r.repo.Reference(plumbing.NewBranchReferenceName(branch), true)
    if err != nil {
        return nil, fmt.Errorf("%w: %s", ErrInvalidBranch, branch)
    }

    // Get commits
    headCommit, err := r.repo.CommitObject(head.Hash())
    if err != nil {
        return nil, fmt.Errorf("get head commit: %w", err)
    }

    branchCommit, err := r.repo.CommitObject(branchRef.Hash())
    if err != nil {
        return nil, fmt.Errorf("get branch commit: %w", err)
    }

    // Compute diff
    patch, err := headCommit.Patch(branchCommit)
    if err != nil {
        return nil, fmt.Errorf("compute diff: %w", err)
    }

    // Parse patch into file changes
    return r.parsePatch(patch, head.Hash().String(), branchRef.Hash().String())
}
```

**Remote Operations:**
```go
func (r *Repository) ListRemotes(ctx context.Context) ([]*RemoteInfo, error) {
    remotes, err := r.repo.Remotes()
    if err != nil {
        return nil, fmt.Errorf("list remotes: %w", err)
    }

    result := make([]*RemoteInfo, 0, len(remotes))
    for _, remote := range remotes {
        config := remote.Config()
        urls := make([]string, 0, len(config.URLs))
        urls = append(urls, config.URLs...)

        result = append(result, &RemoteInfo{
            Name: config.Name,
            URLs: urls,
        })
    }

    return result, nil
}

func (r *Repository) RemoteURL(ctx context.Context, remoteName string) (string, error) {
    remote, err := r.repo.Remote(remoteName)
    if err != nil {
        return "", fmt.Errorf("%w: %s", ErrInvalidRemote, remoteName)
    }

    config := remote.Config()
    if len(config.URLs) == 0 {
        return "", fmt.Errorf("no URLs for remote %s", remoteName)
    }

    return config.URLs[0], nil
}
```

### Error Handling

```go
// Wrap go-git errors with context
func wrapRepoError(err error, op string) error {
    if err == plumbing.ErrReferenceNotFound {
        return fmt.Errorf("%w: %s", ErrInvalidRef, op)
    }
    if err == git.ErrRepositoryNotExists {
        return fmt.Errorf("%w: %s", ErrNotRepository, op)
    }
    return fmt.Errorf("%s: %w", op, err)
}

// Check for detached HEAD
func (r *Repository) isDetachedHead() (bool, error) {
    head, err := r.repo.Head()
    if err != nil {
        return false, err
    }
    return !head.Name().IsBranch(), nil
}
```

### Integration with Core

```go
// In internal/core/context.go or similar:

// GatherGitContext collects Git context for system prompt
func GatherGitContext(ctx context.Context, workspace string) string {
    repo, err := git.Discover(ctx, workspace)
    if err != nil {
        return "" // Not a Git repository
    }

    status, err := repo.Status(ctx)
    if err != nil {
        return fmt.Sprintf("Git repository at %s (status unavailable)", repo.Root())
    }

    var sb strings.Builder
    sb.WriteString(fmt.Sprintf("Git repository: %s\n", repo.Root()))

    if status.Detached {
        sb.WriteString(fmt.Sprintf("Current commit: %s (detached HEAD)\n", status.Hash[:7]))
    } else {
        sb.WriteString(fmt.Sprintf("Current branch: %s\n", status.Branch))
    }

    if status.RemoteBranch != "" {
        sb.WriteString(fmt.Sprintf("Tracking: %s", status.RemoteBranch))
        if status.Ahead > 0 || status.Behind > 0 {
            sb.WriteString(fmt.Sprintf(" [ahead %d, behind %d]", status.Ahead, status.Behind))
        }
        sb.WriteString("\n")
    }

    if len(status.ModifiedFiles) > 0 {
        sb.WriteString(fmt.Sprintf("Modified/staged: %d files\n", len(status.ModifiedFiles)))
    }
    if len(status.UntrackedFiles) > 0 {
        sb.WriteString(fmt.Sprintf("Untracked: %d files\n", len(status.UntrackedFiles)))
    }

    return sb.String()
}
```

## Implementation Plan

### Phase 1: Core Structure & Discovery (Day 1, 4 hours)
1. Create package structure with all files
2. Add go-git dependency: `go get github.com/go-git/go-git/v5`
3. Define types in `types.go`
4. Define errors in `errors.go`
5. Write package documentation in `doc.go`
6. Implement `Repository` struct in `git.go`
7. Implement `Discover()` in `discover.go`
8. Write tests for discovery (valid repo, nested dirs, not a repo)

### Phase 2: Status Operations (Day 1-2, 6 hours)
1. Implement `Status()` in `status.go`
2. Implement status mapping from go-git types
3. Implement ahead/behind calculation
4. Write comprehensive status tests
5. Test with real repositories (clean, dirty, ahead/behind)
6. Test detached HEAD state

### Phase 3: Branch Operations (Day 2, 4 hours)
1. Implement `CurrentBranch()` in `branch.go`
2. Implement `ListBranches()` in `branch.go`
3. Implement `ListRemoteBranches()` in `branch.go`
4. Implement remote tracking resolution
5. Write tests for branch operations
6. Test with multiple branches

### Phase 4: Diff Operations (Day 3, 6 hours)
1. Implement `DiffToBranch()` in `diff.go`
2. Implement `DiffToCommit()` in `diff.go`
3. Implement `DiffBetween()` in `diff.go`
4. Implement patch parser (`parsePatch()`)
5. Write comprehensive diff tests
6. Test with real repositories

### Phase 5: Remote Operations (Day 3, 3 hours)
1. Implement `RemoteURL()` in `remote.go`
2. Implement `ListRemotes()` in `remote.go`
3. Write tests for remote operations
4. Test with repos without remotes
5. Test with multiple remotes

### Phase 6: Polish and Documentation (Day 4, 8 hours)
1. Implement `Log()` function for commit history
2. Run `uast parse | herr analyze` on all files
3. Run `make lint` and fix all issues
4. Achieve ≥85% test coverage
5. Run race detector on all tests
6. Write `docs/packages/git.md`
7. Update roadmap
8. Update AGENTS.md if needed
9. Write integration tests

## Testing Strategy

### Unit Tests

**Test Repository Discovery:**
```go
func TestDiscover(t *testing.T) {
    tests := []struct {
        name    string
        setup   func(t *testing.T) string // Returns startPath
        wantErr error
    }{
        {"valid repo root", setupValidRepo, nil},
        {"nested directory", setupNestedDir, nil},
        {"not a repo", setupNonRepo, ErrNotRepository},
        {"nonexistent path", setupInvalidPath, ErrNotRepository},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            startPath := tt.setup(t)
            repo, err := git.Discover(context.Background(), startPath)

            if tt.wantErr != nil {
                if !errors.Is(err, tt.wantErr) {
                    t.Errorf("want error %v, got %v", tt.wantErr, err)
                }
                return
            }

            if err != nil {
                t.Fatalf("unexpected error: %v", err)
            }

            if repo == nil {
                t.Fatal("expected repo, got nil")
            }
        })
    }
}
```

**Test Status:**
```go
func TestStatus(t *testing.T) {
    tests := []struct {
        name   string
        setup  func(t *testing.T) *Repository
        verify func(t *testing.T, status *Status)
    }{
        {
            name: "clean repo",
            setup: setupCleanRepo,
            verify: func(t *testing.T, s *Status) {
                if len(s.ModifiedFiles) != 0 {
                    t.Error("expected no modified files")
                }
                if len(s.UntrackedFiles) != 0 {
                    t.Error("expected no untracked files")
                }
            },
        },
        {
            name: "modified files",
            setup: setupModifiedRepo,
            verify: func(t *testing.T, s *Status) {
                if len(s.ModifiedFiles) == 0 {
                    t.Error("expected modified files")
                }
            },
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            repo := tt.setup(t)
            status, err := repo.Status(context.Background())
            if err != nil {
                t.Fatalf("Status failed: %v", err)
            }

            tt.verify(t, status)
        })
    }
}
```

### Integration Tests

```go
func TestIntegrationFullWorkflow(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    // Create temporary Git repository using go-git
    tmpDir := t.TempDir()
    repo, err := gogit.PlainInit(tmpDir, false)
    if err != nil {
        t.Fatal(err)
    }

    // Make initial commit
    w, _ := repo.Worktree()
    filename := filepath.Join(tmpDir, "README.md")
    ioutil.WriteFile(filename, []byte("# Test\n"), 0644)
    w.Add("README.md")
    w.Commit("Initial commit", &gogit.CommitOptions{
        Author: &object.Signature{
            Name:  "Test User",
            Email: "test@example.com",
            When:  time.Now(),
        },
    })

    // Test our package
    discovered, err := git.Discover(context.Background(), tmpDir)
    if err != nil {
        t.Fatalf("Discover failed: %v", err)
    }

    status, err := discovered.Status(context.Background())
    if err != nil {
        t.Fatalf("Status failed: %v", err)
    }

    if status.Branch != "master" && status.Branch != "main" {
        t.Errorf("unexpected branch: %s", status.Branch)
    }

    branches, err := discovered.ListBranches(context.Background())
    if err != nil {
        t.Fatalf("ListBranches failed: %v", err)
    }

    if len(branches) == 0 {
        t.Error("expected at least one branch")
    }
}
```

### Benchmark Tests

```go
func BenchmarkDiscover(b *testing.B) {
    tmpDir := setupBenchRepo(b)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := git.Discover(context.Background(), tmpDir)
        if err != nil {
            b.Fatal(err)
        }
    }
}

func BenchmarkStatus(b *testing.B) {
    repo := setupBenchRepo(b)
    discovered, _ := git.Discover(context.Background(), repo)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := discovered.Status(context.Background())
        if err != nil {
            b.Fatal(err)
        }
    }
}

func BenchmarkListBranches(b *testing.B) {
    repo := setupBenchRepoWithBranches(b, 10)
    discovered, _ := git.Discover(context.Background(), repo)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := discovered.ListBranches(context.Background())
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

### Test Coverage Goals

- Overall package: ≥85%
- Critical paths (Discover, Status): ≥90%
- Branch/Diff operations: ≥85%
- Error paths: ≥80%

## Acceptance Criteria

### Functional
- [ ] Can discover Git repository from any subdirectory
- [ ] Returns clear error for non-Git directories
- [ ] Returns complete status (branch, ahead/behind, modified/staged/untracked)
- [ ] Lists all local and remote branches accurately
- [ ] Computes diffs correctly (all change types)
- [ ] Returns remote URLs correctly
- [ ] Handles repositories without remotes gracefully
- [ ] Handles detached HEAD state correctly
- [ ] Respects context cancellation in all operations

### Quality
- [ ] Test coverage ≥85%
- [ ] `make lint` passes (zero errors)
- [ ] Race detector clean: `go test -race ./internal/git`
- [ ] Cyclomatic complexity ≤15 per function
- [ ] All exported types have godoc comments
- [ ] `uast parse | herr analyze` findings addressed
- [ ] No dead code

### Performance
- [ ] Discover: <10ms (typical case)
- [ ] Status: <100ms (repo with <1000 files)
- [ ] ListBranches: <50ms (repo with <100 branches)
- [ ] Diff: <200ms (diff with <500 files)

### Integration
- [ ] Integrates with Core for context gathering
- [ ] Works with repositories created by Git 2.0+
- [ ] Context cancellation respected
- [ ] Clear error messages for all failure modes

### Documentation
- [ ] Package documentation in `doc.go`
- [ ] Full documentation in `docs/packages/git.md`
- [ ] Usage examples in documentation
- [ ] Roadmap updated
- [ ] AGENTS.md updated if needed

## Risks and Mitigations

### Risk 1: go-git Performance on Large Repos
**Impact:** Medium
**Probability:** Low
**Mitigation:**
- Lazy loading (only load what's needed)
- Context timeouts on all operations
- Benchmark with large repos
- Document performance characteristics

### Risk 2: go-git Feature Parity
**Impact:** Medium
**Probability:** Low
**Mitigation:**
- go-git is mature and feature-complete for read operations
- Fallback: can add git command shelling for specific operations if needed
- Test with real-world repositories

### Risk 3: Memory Usage
**Impact:** Medium
**Probability:** Low
**Mitigation:**
- Don't load full object database
- Stream where possible
- Implement resource cleanup
- Add memory benchmarks

### Risk 4: go-git Breaking Changes
**Impact:** Low
**Probability:** Very Low
**Mitigation:**
- Use stable v5 API
- Pin dependency version in go.mod
- Monitor for updates

## Dependencies

### External Dependencies
- `github.com/go-git/go-git/v5` - Pure Go Git implementation
  - License: Apache 2.0
  - Size: ~2MB
  - Maintained: Yes (active development)
  - Used by: GitHub CLI, many major Go projects

### Internal Dependencies
- `pkg/pathutil` - Path validation (for root path handling)
- Standard library: `context`, `path/filepath`, `strings`, `time`

### Reverse Dependencies (Future)
- `internal/core` - Context gathering
- `internal/gitpatch` - Git patch application
- Future commit/branch management tools

## Alternatives Considered

### 1. Shell Out to git Command
**Pros:**
- Simple implementation
- Always up-to-date with git
- No external Go dependencies

**Cons:**
- Security risks (command injection)
- Platform-specific (requires git installation)
- Harder to test
- Less type-safe
- Parsing fragility

**Decision:** Use go-git for safety, reliability, and Go idioms.

### 2. Minimal git Wrapper (libgit2 bindings)
**Pros:**
- Faster than pure Go
- C library is well-tested

**Cons:**
- CGO dependency (cross-compilation issues)
- Build complexity
- Less portable

**Decision:** go-git provides better Go integration.

## Future Enhancements

### Phase 2 Features (Deferred)
- Branch creation and switching
- Commit operations with signing
- Push/pull operations
- Stash management
- Submodule support
- Reflog operations
- Git LFS support

### Performance Optimizations
- Cache repository object
- Lazy loading of status/branches
- Parallel remote operations
- Incremental status updates

### Enhanced Features
- Full unified diff with context
- Conflict detection
- Merge status information
- Cherry-pick detection
- Rebase information

## References

- go-git Documentation: https://pkg.go.dev/github.com/go-git/go-git/v5
- go-git Examples: https://github.com/go-git/go-git/tree/master/_examples
- Git Internals: https://git-scm.com/book/en/v2/Git-Internals-Plumbing-and-Porcelain
- Spin AGENTS.md: 14-step workflow
- Roadmap: specs/tools-modules/ROADMAP.md
- Related FRDs: patchapply series, pathutil, strutil

## Changelog

| Date | Author | Changes |
|------|--------|---------|
| 2025-10-12 | Spin Agent | Initial FRD draft with go-git implementation |

---

**Status:** ✅ Ready for Implementation
**Estimated Effort:** 3-4 days
**Target Completion:** 2025-10-15
