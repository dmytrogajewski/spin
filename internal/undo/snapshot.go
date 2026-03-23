package undo

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/dmytrogajewski/spin/pkg/alg/hashx"
	"github.com/dmytrogajewski/spin/pkg/alg/pathx"
)

const (
	// snapshotDirName is the directory under the user's home for snapshot repos.
	snapshotDirName = ".spin/snapshots"
	// projectHashLen is the number of hex characters used for the project hash.
	projectHashLen = 16
	// gitCommandTimeout is the maximum time allowed for any git command.
	gitCommandTimeout = 30 * time.Second
	// gcPruneAge is the age threshold for garbage collection.
	gcPruneAge = "7.days.ago"
	// shadowRepoPerm is the permission for the shadow repo directory.
	shadowRepoPerm = 0o750
	// excludeFilePerm is the permission for the info/exclude file.
	excludeFilePerm = 0o640
	// diffTreeFieldCount is the expected number of fields in git diff-tree output.
	diffTreeFieldCount = 6
	// diffTreeStatusField is the index of the status field in git diff-tree output.
	diffTreeStatusField = 4
	// diffTreePathField is the index of the path field in git diff-tree output.
	diffTreePathField = 5
)

// Snapshot errors.
var (
	// ErrGitNotFound is returned when the git binary is not on PATH.
	ErrGitNotFound = errors.New("git binary not found on PATH")
	// ErrShadowRepoNotInitialized is returned when Snapshot/Restore is called before Init.
	ErrShadowRepoNotInitialized = errors.New("shadow repo not initialized; call Init() first")
	// ErrSnapshotNotFound is returned when a snapshot hash is not found.
	ErrSnapshotNotFound = errors.New("snapshot not found")
	// ErrUnsafeWorkDir is returned when the work directory is too broad to snapshot safely
	// (e.g., $HOME, /, /tmp). Snapshotting such directories would traverse thousands of
	// files including container storage, caches, and other unrelated data.
	ErrUnsafeWorkDir = errors.New("work directory is too broad for snapshotting")
)

// SnapshotManager maintains a shadow bare git repo that captures full working-tree
// state at each agent step. Enables comprehensive rollback including shell side-effects.
// Thread-safe via [sync.Mutex].
type SnapshotManager struct {
	mu          sync.Mutex
	workDir     string
	shadowDir   string
	initialized bool
	snapshots   []string
}

// NewSnapshotManager creates a new SnapshotManager for the given work directory.
func NewSnapshotManager(workDir string) *SnapshotManager {
	return &SnapshotManager{
		workDir:   workDir,
		snapshots: make([]string, 0),
	}
}

// Init creates the shadow bare git repo if it doesn't exist.
// Returns [ErrUnsafeWorkDir] if the work directory is $HOME, /, or /tmp.
func (m *SnapshotManager) Init() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if pathx.IsUnsafeWorkDir(m.workDir) {
		return ErrUnsafeWorkDir
	}

	if _, err := exec.LookPath("git"); err != nil {
		return ErrGitNotFound
	}

	shadowDir, dirErr := resolveShadowDir(m.workDir)
	if dirErr != nil {
		return fmt.Errorf("resolve shadow dir: %w", dirErr)
	}

	m.shadowDir = shadowDir

	if err := os.MkdirAll(m.shadowDir, shadowRepoPerm); err != nil {
		return fmt.Errorf("create shadow dir: %w", err)
	}

	// Initialize bare repo if not already done.
	if _, statErr := os.Stat(filepath.Join(m.shadowDir, "HEAD")); os.IsNotExist(statErr) {
		if _, err := m.runGit("init", "--bare", m.shadowDir); err != nil {
			return fmt.Errorf("init shadow repo: %w", err)
		}
	}

	// Sync .gitignore to info/exclude.
	if err := m.syncGitignore(); err != nil {
		// Best-effort: log and continue.
		_ = err
	}

	m.initialized = true

	return nil
}

// Snapshot captures the current working tree state and returns a tree hash.
func (m *SnapshotManager) Snapshot() (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.initialized {
		return "", ErrShadowRepoNotInitialized
	}

	// Stage all files in the shadow index.
	// Nested .git directories (e.g., `cargo new` projects) cause `git add -A`
	// to fail with "does not have a commit checked out". We tolerate this
	// specific error since the nested repo's files are still added individually.
	if _, err := m.runGitWithEnvAllowNestedGit("add", "-A"); err != nil {
		return "", fmt.Errorf("git add: %w", err)
	}

	// Write the tree object.
	output, err := m.runGitWithEnv("write-tree")
	if err != nil {
		return "", fmt.Errorf("git write-tree: %w", err)
	}

	treeHash := strings.TrimSpace(output)
	m.snapshots = append(m.snapshots, treeHash)

	return treeHash, nil
}

// Restore reverts the working tree to the state captured by the given tree hash.
func (m *SnapshotManager) Restore(treeHash string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.initialized {
		return ErrShadowRepoNotInitialized
	}

	// Get the current tree hash for comparison.
	currentHash, err := m.runGitWithEnv("write-tree")
	if err != nil {
		return fmt.Errorf("get current tree: %w", err)
	}

	currentHash = strings.TrimSpace(currentHash)

	// Find changed files between current and target.
	diffOutput, diffErr := m.runGitWithEnv("diff-tree", "-r", "--no-commit-id", currentHash, treeHash)
	if diffErr != nil {
		return fmt.Errorf("diff-tree: %w", diffErr)
	}

	// Parse diff-tree output and restore each file.
	return m.applyDiffTree(diffOutput, treeHash)
}

// Cleanup runs git garbage collection on the shadow repo.
func (m *SnapshotManager) Cleanup() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.initialized {
		return ErrShadowRepoNotInitialized
	}

	if _, err := m.runGit("--git-dir", m.shadowDir, "gc", "--prune="+gcPruneAge); err != nil {
		return fmt.Errorf("git gc: %w", err)
	}

	return nil
}

// SnapshotCount returns the number of snapshots taken.
func (m *SnapshotManager) SnapshotCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return len(m.snapshots)
}

// GetSnapshot returns the snapshot hash at the given step index (zero-based).
func (m *SnapshotManager) GetSnapshot(step int) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if step < 0 || step >= len(m.snapshots) {
		return "", ErrSnapshotNotFound
	}

	return m.snapshots[step], nil
}

// ShadowDir returns the path to the shadow repo directory.
func (m *SnapshotManager) ShadowDir() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.shadowDir
}

// nestedGitExclude prevents `git add -A` from failing on nested .git directories.
// Without this, directories containing their own .git (e.g., `cargo new` projects)
// cause "does not have a commit checked out" errors.
const nestedGitExclude = "**/.git\n"

// syncGitignore copies .gitignore from workDir to shadow repo's info/exclude,
// prepending rules to ignore nested .git directories.
func (m *SnapshotManager) syncGitignore() error {
	infoDir := filepath.Join(m.shadowDir, "info")

	if mkErr := os.MkdirAll(infoDir, shadowRepoPerm); mkErr != nil {
		return fmt.Errorf("create info dir: %w", mkErr)
	}

	// Always exclude nested .git dirs to prevent submodule-like errors.
	var content []byte

	content = append(content, []byte(nestedGitExclude)...)

	gitignorePath := filepath.Join(m.workDir, ".gitignore")

	if userIgnore, err := os.ReadFile(gitignorePath); err == nil {
		content = append(content, userIgnore...)
	}

	excludePath := filepath.Join(infoDir, "exclude")

	if writeErr := os.WriteFile(excludePath, content, excludeFilePerm); writeErr != nil {
		return fmt.Errorf("write exclude: %w", writeErr)
	}

	return nil
}

// applyDiffTree parses git diff-tree output and restores files from the target tree.
func (m *SnapshotManager) applyDiffTree(diffOutput, treeHash string) error {
	for line := range strings.SplitSeq(strings.TrimSpace(diffOutput), "\n") {
		if line == "" {
			continue
		}

		if err := m.restoreDiffLine(line, treeHash); err != nil {
			return err
		}
	}

	return nil
}

// restoreDiffLine processes a single diff-tree output line and restores the file.
func (m *SnapshotManager) restoreDiffLine(line, treeHash string) error {
	// Diff-tree format: :oldmode newmode oldhash newhash status path.
	parts := strings.Fields(line)
	if len(parts) < diffTreeFieldCount {
		return nil
	}

	status := parts[diffTreeStatusField]
	path := parts[diffTreePathField]
	fullPath := filepath.Join(m.workDir, path)

	switch {
	case strings.HasPrefix(status, "D"):
		// File was deleted in target tree — delete from working tree.
		if err := os.Remove(fullPath); err != nil {
			return fmt.Errorf("remove %s: %w", path, err)
		}

		return nil
	default:
		// File was added or modified — restore from target tree.
		return m.restoreFileFromTree(treeHash, path, fullPath)
	}
}

// restoreFileFromTree reads a file from the target tree and writes it to the working tree.
func (m *SnapshotManager) restoreFileFromTree(treeHash, relativePath, fullPath string) error {
	content, err := m.runGitWithEnv("cat-file", "-p", treeHash+":"+relativePath)
	if err != nil {
		return fmt.Errorf("cat-file %s: %w", relativePath, err)
	}

	dir := filepath.Dir(fullPath)
	if mkErr := os.MkdirAll(dir, shadowRepoPerm); mkErr != nil {
		return fmt.Errorf("create dir for %s: %w", relativePath, mkErr)
	}

	if writeErr := os.WriteFile(fullPath, []byte(content), excludeFilePerm); writeErr != nil {
		return fmt.Errorf("write %s: %w", relativePath, writeErr)
	}

	return nil
}

// runGit executes a git command with timeout.
func (m *SnapshotManager) runGit(args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", args...)

	var stdout, stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s: %w", stderr.String(), err)
	}

	return stdout.String(), nil
}

// nestedGitErrSubstring is the error message git produces for nested repos without commits.
const nestedGitErrSubstring = "does not have a commit checked out"

// runGitWithEnvAllowNestedGit runs a git command, tolerating errors caused by
// nested .git directories. If the only errors are about gitlink/submodule entries,
// the command is considered successful (the other files were still staged).
func (m *SnapshotManager) runGitWithEnvAllowNestedGit(args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = m.workDir
	cmd.Env = append(os.Environ(),
		"GIT_DIR="+m.shadowDir,
		"GIT_WORK_TREE="+m.workDir,
	)

	var stdout, stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrStr := stderr.String()

		// Tolerate errors caused solely by nested .git directories.
		// Git produces lines like:
		//   error: 'sub/' does not have a commit checked out
		//   error: unable to index file 'sub/'
		//   fatal: adding files failed  (or localized equivalent)
		// These are safe to ignore — the other files were still staged.
		if isOnlyNestedGitError(stderrStr) {
			return stdout.String(), nil
		}

		return "", fmt.Errorf("%s: %w", stderrStr, err)
	}

	return stdout.String(), nil
}

// isOnlyNestedGitError returns true if all stderr lines are about nested .git dirs.
// Returns false if there are any unrecognized error lines.
func isOnlyNestedGitError(stderr string) bool {
	if !strings.Contains(stderr, nestedGitErrSubstring) {
		return false
	}

	hasNestedErr := false

	for line := range strings.SplitSeq(stderr, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.Contains(line, nestedGitErrSubstring) {
			hasNestedErr = true

			continue
		}

		// These always accompany the nested git error.
		if strings.Contains(line, "unable to index file") ||
			strings.HasPrefix(line, "fatal:") ||
			strings.HasPrefix(line, "error:") {
			continue
		}

		// Unrecognized line — not safe to ignore.
		return false
	}

	return hasNestedErr
}

// runGitWithEnv executes a git command with GIT_DIR and GIT_WORK_TREE set.
func (m *SnapshotManager) runGitWithEnv(args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = m.workDir
	cmd.Env = append(os.Environ(),
		"GIT_DIR="+m.shadowDir,
		"GIT_WORK_TREE="+m.workDir,
	)

	var stdout, stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s: %w", stderr.String(), err)
	}

	return stdout.String(), nil
}

// ProjectHash returns a deterministic hash of the given work directory path.
// Uses SHA-256 truncated to projectHashLen hex characters.
func ProjectHash(workDir string) string {
	absPath, err := filepath.Abs(workDir)
	if err != nil {
		absPath = workDir
	}

	return hashx.SHA256Hex([]byte(absPath))[:projectHashLen]
}

// resolveShadowDir computes the shadow repo directory path.
func resolveShadowDir(workDir string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}

	hash := ProjectHash(workDir)

	return filepath.Join(homeDir, snapshotDirName, hash), nil
}
