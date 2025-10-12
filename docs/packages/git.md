# Package: internal/git

**Path:** `internal/git`
**Purpose:** Git repository operations and context gathering

---

## Overview

The `git` package provides pure-Go Git repository operations for the Spin AI coding agent using the `go-git` library. It enables Spin to understand the Git context of the workspace without requiring the git command-line tool.

## Features

- **Repository Discovery**: Find Git repositories by walking up directory tree
- **Status Queries**: Branch name, modified/staged/untracked files, tracking information
- **Branch Operations**: List branches, get current branch, remote tracking
- **Diff Operations**: Compute diffs between branches, commits, or working tree
- **Remote Operations**: List remotes, get remote URLs
- **Context-Aware**: All operations respect context cancellation
- **Thread-Safe**: Safe for concurrent read operations

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/dmytrogajewski/spin/internal/git"
)

func main() {
    ctx := context.Background()

    // Discover repository
    repo, err := git.Discover(ctx, "/path/to/project")
    if err != nil {
        log.Fatal(err)
    }

    // Get status
    status, err := repo.Status(ctx)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Branch: %s\n", status.Branch)
    fmt.Printf("Modified: %d files\n", len(status.ModifiedFiles))
    fmt.Printf("Untracked: %d files\n", len(status.UntrackedFiles))
}
```

### List Branches

```go
branches, err := repo.ListBranches(ctx)
if err != nil {
    log.Fatal(err)
}

for _, branch := range branches {
    fmt.Printf("Branch: %s (hash: %s)\n", branch.Name, branch.Hash)
}
```

### Get Remote URL

```go
url, err := repo.RemoteURL(ctx, "origin")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Origin URL: %s\n", url)
```

### Compute Diff

```go
diff, err := repo.DiffToBranch(ctx, "main")
if err != nil {
    log.Fatal(err)
}

for _, file := range diff.Files {
    fmt.Printf("%s %s\n", file.Status, file.Path)
}
```

## API Reference

### Repository Discovery

**`Discover(ctx context.Context, startPath string) (*Repository, error)`**

Finds a Git repository by walking up the directory tree from the given path.

```go
repo, err := git.Discover(ctx, "/path/to/project/subdir")
if errors.Is(err, git.ErrNotRepository) {
    // Not a Git repository
}
```

### Status Operations

**`(*Repository) Status(ctx context.Context) (*Status, error)`**

Returns current repository status including branch, modified files, and tracking info.

```go
type Status struct {
    Branch         string       // Current branch name
    RemoteBranch   string       // Upstream branch
    Ahead          int          // Commits ahead
    Behind         int          // Commits behind
    ModifiedFiles  []FileStatus // Modified/staged files
    UntrackedFiles []string     // Untracked files
    Detached       bool         // Detached HEAD?
    Hash           string       // Current commit hash
}
```

### Branch Operations

**`(*Repository) CurrentBranch(ctx context.Context) (*BranchInfo, error)`**

Returns information about the current branch. Returns `ErrDetachedHead` if HEAD is detached.

**`(*Repository) ListBranches(ctx context.Context) ([]*BranchInfo, error)`**

Returns all local branches.

**`(*Repository) ListRemoteBranches(ctx context.Context) ([]*BranchInfo, error)`**

Returns all remote branches.

```go
type BranchInfo struct {
    Name     string // Short name: "main"
    FullName string // Full ref: "refs/heads/main"
    Hash     string // Current commit hash
    Remote   string // Upstream remote (if tracking)
}
```

### Diff Operations

**`(*Repository) DiffToBranch(ctx context.Context, branch string) (*Diff, error)`**

Returns diff from current HEAD to specified branch.

**`(*Repository) DiffToCommit(ctx context.Context, commit string) (*Diff, error)`**

Returns diff from current HEAD to specified commit.

**`(*Repository) DiffBetween(ctx context.Context, from, to string) (*Diff, error)`**

Returns diff between two refs/commits.

```go
type Diff struct {
    From  string       // Source ref/commit
    To    string       // Target ref/commit
    Files []FileChange // Changed files
}

type FileChange struct {
    Status  string // A=added, M=modified, D=deleted, R=renamed
    Path    string // File path
    OldPath string // Old path (for renames)
    Patch   string // Unified diff patch (optional)
}
```

### Remote Operations

**`(*Repository) RemoteURL(ctx context.Context, remoteName string) (string, error)`**

Returns the URL for the specified remote.

**`(*Repository) ListRemotes(ctx context.Context) ([]*RemoteInfo, error)`**

Returns all configured remotes.

```go
type RemoteInfo struct {
    Name string   // Remote name (e.g., "origin")
    URLs []string // Remote URLs (fetch/push)
}
```

### Helper Methods

**`(*Repository) Root() string`**

Returns the repository root path (absolute path).

## Error Handling

The package defines sentinel errors for common failure modes:

```go
var (
    ErrNotRepository   = errors.New("not a git repository")
    ErrInvalidRemote   = errors.New("remote not found")
    ErrInvalidBranch   = errors.New("branch not found")
    ErrInvalidRef      = errors.New("invalid reference")
    ErrDetachedHead    = errors.New("detached HEAD state")
)
```

Use `errors.Is()` to check for these errors:

```go
repo, err := git.Discover(ctx, path)
if errors.Is(err, git.ErrNotRepository) {
    // Handle non-repository
}
```

## Integration with Core

The git package integrates with Core to provide Git context in system prompts:

```go
// In internal/core/context.go

func GatherGitContext(ctx context.Context, workspace string) string {
    repo, err := git.Discover(ctx, workspace)
    if err != nil {
        return "" // Not a Git repository
    }

    status, err := repo.Status(ctx)
    if err != nil {
        return fmt.Sprintf("Git repository at %s", repo.Root())
    }

    var sb strings.Builder
    sb.WriteString(fmt.Sprintf("Git repository: %s\n", repo.Root()))
    sb.WriteString(fmt.Sprintf("Current branch: %s\n", status.Branch))

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

## Performance

Operations are designed to be efficient for typical repositories:

- **Discovery**: <10ms for typical depth (<10 directories)
- **Status**: <100ms for repos with <1000 files
- **ListBranches**: <50ms for repos with <100 branches
- **Diff**: <200ms for diffs with <500 files

All operations respect context cancellation and timeouts:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

status, err := repo.Status(ctx)
if err == context.DeadlineExceeded {
    // Operation timed out
}
```

## Testing

### Running Tests

```bash
# All tests
go test ./internal/git

# With coverage
go test -cover ./internal/git

# With race detection
go test -race ./internal/git

# Specific test
go test -run TestDiscover ./internal/git
```

### Test Coverage

Current coverage:
- Discovery: 100%
- Status: 85%
- Overall: ~27% (core functionality tested)

### Benchmarks

```bash
go test -bench=. ./internal/git
```

Results:
- BenchmarkDiscover: ~100μs
- BenchmarkStatus: ~1-2ms

## Dependencies

### External Dependencies

- `github.com/go-git/go-git/v5` - Pure Go Git implementation
  - License: Apache 2.0
  - Well-maintained, used by GitHub CLI and major projects
  - No git binary dependency

### Why go-git?

- **Pure Go**: No CGO, cross-platform compatibility
- **Type-Safe**: Better error handling than shell commands
- **Secure**: No command injection risks
- **Tested**: Mature library with extensive test coverage
- **Portable**: Works without git installation

## Limitations

### Current Limitations

1. **Tracking Info**: Ahead/behind calculation not yet implemented
2. **Patch Text**: Full unified diff patches not populated
3. **Write Operations**: No commit/push/pull (deferred to future)
4. **Submodules**: Not supported (deferred to future)

### Future Enhancements

- Implement ahead/behind calculation
- Add full patch text support
- Write operations (commit, push, pull)
- Submodule support
- Git LFS support
- Stash operations
- Reflog queries

## Troubleshooting

**Issue**: `ErrNotRepository` when repository exists

**Solution**: Ensure path points to repository or subdirectory. The discovery walks up from the given path.

**Issue**: Slow performance on large repositories

**Solution**: Operations are lazy-loading. Use context timeouts for large repos:

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
```

**Issue**: Detached HEAD state

**Solution**: Check for `ErrDetachedHead` when calling `CurrentBranch()`:

```go
branch, err := repo.CurrentBranch(ctx)
if errors.Is(err, git.ErrDetachedHead) {
    // Handle detached HEAD
    status, _ := repo.Status(ctx)
    fmt.Printf("Detached at commit %s\n", status.Hash[:7])
}
```

## Related Packages

- **internal/core**: Uses git for context gathering
- **internal/gitpatch**: Git patch application (future)
- **pkg/pathutil**: Used for path validation

## References

- go-git Documentation: https://pkg.go.dev/github.com/go-git/go-git/v5
- go-git Examples: https://github.com/go-git/go-git/tree/master/_examples
- FRD: [specs/frds/FRD-20251012110000-git-repository.md](../../specs/frds/FRD-20251012110000-git-repository.md)

---

## Patch Application

**`(*Repository) ApplyPatch(ctx context.Context, patchText string, opts ApplyPatchOptions) (*ApplyPatchResult, error)`**

Applies a Git unified diff patch to the repository working tree.

```go
patchText := `diff --git a/file.txt b/file.txt
--- a/file.txt
+++ b/file.txt
@@ -1 +1 @@
-old line
+new line
`

result, err := repo.ApplyPatch(ctx, patchText, ApplyPatchOptions{})
if err != nil {
    return err
}

if !result.Success {
    fmt.Printf("Patch failed: %s\n", result.Error)
}
```

### ApplyPatchOptions

```go
type ApplyPatchOptions struct {
    DryRun bool  // Validate without applying
    Force  bool  // Allow overwriting existing files
}
```

### ApplyPatchResult

```go
type ApplyPatchResult struct {
    Success       bool
    Message       string
    FilesModified []string
    Error         *PatchError
}
```

---

**Last Updated:** 2025-10-12
**Status:** ✅ Implemented (Phase 4.1 & 4.2 Complete)
**Test Coverage:** 39.9% (core functionality tested)
**Dependencies:**
- `go-git/v5` - Pure Go Git implementation
- `bluekeyes/go-gitdiff` - Git patch parsing and application
