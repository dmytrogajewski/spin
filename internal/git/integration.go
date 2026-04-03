package git

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

const minDiffContextLines = 5

var (
	// ErrNoGitStatusAvailable is a sentinel error.
	ErrNoGitStatusAvailable = errors.New("no Git status available")
	// ErrNotAGitRepository is a sentinel error.
	ErrNotAGitRepository = errors.New("not a Git repository")
)

// Integration provides Git-aware functionality for the agent.
type Integration struct {
	enabled    bool
	workDir    string
	logger     *slog.Logger
	mu         sync.RWMutex
	repo       *Repository
	lastStatus *Status
}

// NewIntegration creates a new Git integration.
func NewIntegration(enabled bool, workDir string, logger *slog.Logger) *Integration {
	return &Integration{
		enabled: enabled,
		workDir: workDir,
		logger:  logger,
	}
}

// Initialize sets up Git integration.
func (g *Integration) Initialize(ctx context.Context) error {
	if !g.enabled {
		g.logger.DebugContext(ctx, "Git integration disabled")

		return nil
	}

	// Check if workDir is a Git repository.
	repo, err := Discover(ctx, g.workDir)
	if err != nil {
		g.logger.DebugContext(ctx, "Not a Git repository", "workDir", g.workDir, "error", err)

		return nil // Not an error, just not a Git repo.
	}

	g.mu.Lock()
	g.repo = repo
	g.mu.Unlock()

	g.logger.InfoContext(ctx, "Git integration initialized",
		"repo", repo.Root(),
		"workDir", g.workDir)

	// Get initial status.
	err = g.updateStatus(ctx)
	if err != nil {
		g.logger.WarnContext(ctx, "Failed to get initial Git status", "error", err)
	}

	return nil
}

// IsEnabled returns true if Git integration is enabled.
func (g *Integration) IsEnabled() bool {
	return g.enabled
}

// IsRepository returns true if the working directory is a Git repository.
func (g *Integration) IsRepository() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return g.repo != nil
}

// GetRepository returns the Git repository if available.
func (g *Integration) GetRepository() *Repository {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return g.repo
}

// GetStatus returns the current Git status.
func (g *Integration) GetStatus() *Status {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return g.lastStatus
}

// RefreshStatus updates the Git status.
func (g *Integration) RefreshStatus(ctx context.Context) error {
	return g.updateStatus(ctx)
}

// updateStatus updates the Git status.
func (g *Integration) updateStatus(ctx context.Context) error {
	if !g.IsRepository() {
		return nil
	}

	status, err := g.repo.Status(ctx)
	if err != nil {
		return fmt.Errorf("failed to get Git status: %w", err)
	}

	g.mu.Lock()
	g.lastStatus = status
	g.mu.Unlock()

	return nil
}

// GetBranch returns the current branch name.
func (g *Integration) GetBranch() (string, error) {
	status := g.GetStatus()
	if status == nil {
		return "", ErrNoGitStatusAvailable
	}

	return status.Branch, nil
}

// GetRemoteURL returns the remote URL for the current branch.
func (g *Integration) GetRemoteURL() (string, error) {
	status := g.GetStatus()
	if status == nil {
		return "", ErrNoGitStatusAvailable
	}

	return status.RemoteBranch, nil
}

// GetCommitHash returns the current commit hash.
func (g *Integration) GetCommitHash() (string, error) {
	status := g.GetStatus()
	if status == nil {
		return "", ErrNoGitStatusAvailable
	}

	return status.Hash, nil
}

// GetModifiedFiles returns the list of modified files.
func (g *Integration) GetModifiedFiles() ([]string, error) {
	status := g.GetStatus()
	if status == nil {
		return nil, ErrNoGitStatusAvailable
	}

	var files []string

	for _, file := range status.ModifiedFiles {
		if file.Worktree == Modified || file.Worktree == Added ||
			file.Worktree == Deleted || file.Worktree == Renamed {
			files = append(files, file.Path)
		}
	}

	return files, nil
}

// GetStagedFiles returns the list of staged files.
func (g *Integration) GetStagedFiles() ([]string, error) {
	status := g.GetStatus()
	if status == nil {
		return nil, ErrNoGitStatusAvailable
	}

	var files []string

	for _, file := range status.ModifiedFiles {
		if file.Staging == Added || file.Staging == Modified ||
			file.Staging == Deleted || file.Staging == Renamed {
			files = append(files, file.Path)
		}
	}

	return files, nil
}

// GetUntrackedFiles returns the list of untracked files.
func (g *Integration) GetUntrackedFiles() ([]string, error) {
	status := g.GetStatus()
	if status == nil {
		return nil, ErrNoGitStatusAvailable
	}

	return status.UntrackedFiles, nil
}

// IsClean returns true if the working directory is clean.
func (g *Integration) IsClean() bool {
	status := g.GetStatus()
	if status == nil {
		return true // Assume clean if no status.
	}

	return len(status.ModifiedFiles) == 0 && len(status.UntrackedFiles) == 0
}

// ContextInfo holds git context information for the agent.
type ContextInfo struct {
	GitEnabled     bool   `json:"git_enabled"`
	IsRepo         bool   `json:"is_repo"`
	Branch         string `json:"branch,omitempty"`
	Remote         string `json:"remote,omitempty"`
	Commit         string `json:"commit,omitempty"`
	IsClean        bool   `json:"is_clean,omitempty"`
	ModifiedFiles  int    `json:"modified_files,omitempty"`
	UntrackedFiles int    `json:"untracked_files,omitempty"`
	Ahead          int    `json:"ahead,omitempty"`
	Behind         int    `json:"behind,omitempty"`
	Detached       bool   `json:"detached,omitempty"`
}

// GetContextInfo returns Git context information for the agent.
func (g *Integration) GetContextInfo() ContextInfo {
	if !g.IsRepository() {
		return ContextInfo{
			GitEnabled: false,
			IsRepo:     false,
		}
	}

	info := ContextInfo{
		GitEnabled: true,
		IsRepo:     true,
	}

	branch, err := g.GetBranch()
	if err == nil {
		info.Branch = branch
	}

	remote, err := g.GetRemoteURL()
	if err == nil {
		info.Remote = remote
	}

	commit, err := g.GetCommitHash()
	if err == nil {
		info.Commit = commit
	}

	if status := g.GetStatus(); status != nil {
		info.IsClean = g.IsClean()
		info.ModifiedFiles = len(status.ModifiedFiles)
		info.UntrackedFiles = len(status.UntrackedFiles)
		info.Ahead = status.Ahead
		info.Behind = status.Behind
		info.Detached = status.Detached
	}

	return info
}

// GetDiff returns the diff for a specific file or all changes.
func (g *Integration) GetDiff(ctx context.Context, filePath string) (string, error) {
	if !g.IsRepository() {
		return "", ErrNotAGitRepository
	}

	// Use "git diff HEAD" to show both staged and unstaged changes.
	// Plain "git diff" only shows unstaged, missing anything after "git add".
	var args []string
	if filePath != "" {
		args = []string{"diff", "HEAD", "--", filePath}
	} else {
		args = []string{"diff", "HEAD"}
	}

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = g.workDir

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get diff: %w", err)
	}

	return string(output), nil
}

// GetLog returns recent commit history.
func (g *Integration) GetLog(ctx context.Context, limit int) ([]CommitInfo, error) {
	if !g.IsRepository() {
		return nil, ErrNotAGitRepository
	}

	if limit <= 0 {
		limit = 10 // Default limit.
	}

	// Use git log with format to get commit info.
	cmd := exec.CommandContext(ctx, "git", "log",
		fmt.Sprintf("-%d", limit),
		"--format=%H%n%an%n%ae%n%at%n%s%n%b%n---END---")
	cmd.Dir = g.workDir

	output, logErr := cmd.Output()
	if logErr == nil {
		return parseGitLog(string(output)), nil
	}

	// Empty repo or no commits is not an error.
	return []CommitInfo{}, nil
}

// runGitCmd executes a git command in the working directory, guarding on IsRepository.
func (g *Integration) runGitCmd(ctx context.Context, errPrefix string, args ...string) error {
	if !g.IsRepository() {
		return ErrNotAGitRepository
	}

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = g.workDir

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %w", errPrefix, err)
	}

	return nil
}

// StageFile stages a file for commit.
func (g *Integration) StageFile(ctx context.Context, filePath string) error {
	return g.runGitCmd(ctx, "git add", "add", filePath)
}

// UnstageFile unstages a file.
func (g *Integration) UnstageFile(ctx context.Context, filePath string) error {
	return g.runGitCmd(ctx, "git reset", "reset", "HEAD", filePath)
}

// Commit creates a commit with the given message.
func (g *Integration) Commit(ctx context.Context, message string) error {
	return g.runGitCmd(ctx, "git commit", "commit", "-m", message)
}

// Push pushes changes to the remote repository.
func (g *Integration) Push(ctx context.Context) error {
	return g.runGitCmd(ctx, "git push", "push")
}

// Pull pulls changes from the remote repository.
func (g *Integration) Pull(ctx context.Context) error {
	return g.runGitCmd(ctx, "git pull", "pull")
}

// CreateBranch creates a new branch.
func (g *Integration) CreateBranch(ctx context.Context, branchName string) error {
	return g.runGitCmd(ctx, "git checkout -b", "checkout", "-b", branchName)
}

// SwitchBranch switches to an existing branch.
func (g *Integration) SwitchBranch(ctx context.Context, branchName string) error {
	return g.runGitCmd(ctx, "git checkout", "checkout", branchName)
}

// ListBranches returns the list of local and remote branches.
func (g *Integration) ListBranches(ctx context.Context) ([]string, error) {
	if !g.IsRepository() {
		return nil, ErrNotAGitRepository
	}

	cmd := exec.CommandContext(ctx, "git", "branch", "--format=%(refname:short)")
	cmd.Dir = g.workDir

	output, branchErr := cmd.Output()
	if branchErr == nil {
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")

		branches := make([]string, 0, len(lines))
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" {
				branches = append(branches, line)
			}
		}

		return branches, nil
	}

	// No branches is not an error.
	return []string{}, nil
}

// ListRemotes returns the list of remote repositories.
func (g *Integration) ListRemotes(ctx context.Context) ([]string, error) {
	if !g.IsRepository() {
		return nil, ErrNotAGitRepository
	}

	cmd := exec.CommandContext(ctx, "git", "remote")
	cmd.Dir = g.workDir

	output, remoteErr := cmd.Output()
	if remoteErr == nil {
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")

		remotes := make([]string, 0, len(lines))
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" {
				remotes = append(remotes, line)
			}
		}

		return remotes, nil
	}

	// No remotes or error is not an error.
	return []string{}, nil
}

// GetWorkingDirectory returns the working directory.
func (g *Integration) GetWorkingDirectory() string {
	return g.workDir
}

// SetWorkingDirectory sets the working directory.
func (g *Integration) SetWorkingDirectory(workDir string) {
	g.mu.Lock()
	g.workDir = workDir
	g.mu.Unlock()
}

// Close cleans up Git integration resources.
func (g *Integration) Close() error {
	// No resources to clean up.
	return nil
}

// parseGitLog parses git log output into CommitInfo structs.
func parseGitLog(output string) []CommitInfo {
	commits := make([]CommitInfo, 0)

	// Split by commit delimiter.
	commitBlocks := strings.SplitSeq(output, "---END---")

	for block := range commitBlocks {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}

		lines := strings.Split(block, "\n")
		if len(lines) < minDiffContextLines {
			continue // Invalid commit block.
		}

		commit := CommitInfo{
			Hash:   strings.TrimSpace(lines[0]),
			Author: strings.TrimSpace(lines[1]),
			Email:  strings.TrimSpace(lines[2]),
		}

		// Parse timestamp.
		timestamp := strings.TrimSpace(lines[3])

		ts, err := strconv.ParseInt(timestamp, 10, 64)
		if err == nil {
			commit.Timestamp = time.Unix(ts, 0).Format(time.RFC3339)
		} else {
			commit.Timestamp = timestamp
		}

		// Subject and body.
		subject := strings.TrimSpace(lines[4])

		var body string

		if len(lines) > minDiffContextLines {
			bodyLines := lines[5:]
			body = strings.TrimSpace(strings.Join(bodyLines, "\n"))
		}

		// Combine subject and body into Message.
		if body != "" {
			commit.Message = subject + "\n\n" + body
		} else {
			commit.Message = subject
		}

		commits = append(commits, commit)
	}

	return commits
}
