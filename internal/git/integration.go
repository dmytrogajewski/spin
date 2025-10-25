package git

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"sync"
)

// GitIntegration provides Git-aware functionality for the agent.
type GitIntegration struct {
	enabled    bool
	workDir    string
	logger     *slog.Logger
	mu         sync.RWMutex
	repo       *Repository
	lastStatus *Status
}

// NewGitIntegration creates a new Git integration.
func NewGitIntegration(enabled bool, workDir string, logger *slog.Logger) *GitIntegration {
	return &GitIntegration{
		enabled: enabled,
		workDir: workDir,
		logger:  logger,
	}
}

// Initialize sets up Git integration.
func (g *GitIntegration) Initialize(ctx context.Context) error {
	if !g.enabled {
		g.logger.Debug("Git integration disabled")
		return nil
	}

	// Check if workDir is a Git repository
	repo, err := Discover(ctx, g.workDir)
	if err != nil {
		g.logger.Debug("Not a Git repository", "workDir", g.workDir, "error", err)
		return nil // Not an error, just not a Git repo
	}

	g.mu.Lock()
	g.repo = repo
	g.mu.Unlock()

	g.logger.Info("Git integration initialized",
		"repo", repo.Root(),
		"workDir", g.workDir)

	// Get initial status
	if err := g.refreshStatus(ctx); err != nil {
		g.logger.Warn("Failed to get initial Git status", "error", err)
	}

	return nil
}

// IsEnabled returns true if Git integration is enabled.
func (g *GitIntegration) IsEnabled() bool {
	return g.enabled
}

// IsRepository returns true if the working directory is a Git repository.
func (g *GitIntegration) IsRepository() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.repo != nil
}

// GetRepository returns the Git repository if available.
func (g *GitIntegration) GetRepository() *Repository {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.repo
}

// GetStatus returns the current Git status.
func (g *GitIntegration) GetStatus() *Status {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.lastStatus
}

// RefreshStatus updates the Git status.
func (g *GitIntegration) RefreshStatus(ctx context.Context) error {
	return g.refreshStatus(ctx)
}

// refreshStatus updates the Git status.
func (g *GitIntegration) refreshStatus(ctx context.Context) error {
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
func (g *GitIntegration) GetBranch() (string, error) {
	status := g.GetStatus()
	if status == nil {
		return "", fmt.Errorf("no Git status available")
	}

	return status.Branch, nil
}

// GetRemoteURL returns the remote URL for the current branch.
func (g *GitIntegration) GetRemoteURL() (string, error) {
	status := g.GetStatus()
	if status == nil {
		return "", fmt.Errorf("no Git status available")
	}

	return status.RemoteBranch, nil
}

// GetCommitHash returns the current commit hash.
func (g *GitIntegration) GetCommitHash() (string, error) {
	status := g.GetStatus()
	if status == nil {
		return "", fmt.Errorf("no Git status available")
	}

	return status.Hash, nil
}

// GetModifiedFiles returns the list of modified files.
func (g *GitIntegration) GetModifiedFiles() ([]string, error) {
	status := g.GetStatus()
	if status == nil {
		return nil, fmt.Errorf("no Git status available")
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
func (g *GitIntegration) GetStagedFiles() ([]string, error) {
	status := g.GetStatus()
	if status == nil {
		return nil, fmt.Errorf("no Git status available")
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
func (g *GitIntegration) GetUntrackedFiles() ([]string, error) {
	status := g.GetStatus()
	if status == nil {
		return nil, fmt.Errorf("no Git status available")
	}

	return status.UntrackedFiles, nil
}

// IsClean returns true if the working directory is clean.
func (g *GitIntegration) IsClean() bool {
	status := g.GetStatus()
	if status == nil {
		return true // Assume clean if no status
	}

	return len(status.ModifiedFiles) == 0 && len(status.UntrackedFiles) == 0
}

// GetContextInfo returns Git context information for the agent.
func (g *GitIntegration) GetContextInfo() map[string]interface{} {
	if !g.IsRepository() {
		return map[string]interface{}{
			"git_enabled": false,
			"is_repo":     false,
		}
	}

	info := map[string]interface{}{
		"git_enabled": true,
		"is_repo":     true,
	}

	if branch, err := g.GetBranch(); err == nil {
		info["branch"] = branch
	}

	if remote, err := g.GetRemoteURL(); err == nil {
		info["remote"] = remote
	}

	if commit, err := g.GetCommitHash(); err == nil {
		info["commit"] = commit
	}

	if status := g.GetStatus(); status != nil {
		info["is_clean"] = g.IsClean()
		info["modified_files"] = len(status.ModifiedFiles)
		info["untracked_files"] = len(status.UntrackedFiles)
		info["ahead"] = status.Ahead
		info["behind"] = status.Behind
		info["detached"] = status.Detached
	}

	return info
}

// GetDiff returns the diff for a specific file or all changes.
func (g *GitIntegration) GetDiff(filePath string) (string, error) {
	if !g.IsRepository() {
		return "", fmt.Errorf("not a Git repository")
	}

	var args []string
	if filePath != "" {
		args = []string{"diff", filePath}
	} else {
		args = []string{"diff"}
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = g.workDir
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get diff: %w", err)
	}

	return string(output), nil
}

// GetLog returns recent commit history.
func (g *GitIntegration) GetLog(limit int) ([]CommitInfo, error) {
	if !g.IsRepository() {
		return nil, fmt.Errorf("not a Git repository")
	}

	// For now, return empty list - would need to implement log functionality
	return []CommitInfo{}, nil
}

// StageFile stages a file for commit.
func (g *GitIntegration) StageFile(filePath string) error {
	if !g.IsRepository() {
		return fmt.Errorf("not a Git repository")
	}

	cmd := exec.Command("git", "add", filePath)
	cmd.Dir = g.workDir
	return cmd.Run()
}

// UnstageFile unstages a file.
func (g *GitIntegration) UnstageFile(filePath string) error {
	if !g.IsRepository() {
		return fmt.Errorf("not a Git repository")
	}

	cmd := exec.Command("git", "reset", "HEAD", filePath)
	cmd.Dir = g.workDir
	return cmd.Run()
}

// Commit creates a commit with the given message.
func (g *GitIntegration) Commit(message string) error {
	if !g.IsRepository() {
		return fmt.Errorf("not a Git repository")
	}

	cmd := exec.Command("git", "commit", "-m", message)
	cmd.Dir = g.workDir
	return cmd.Run()
}

// Push pushes changes to the remote repository.
func (g *GitIntegration) Push() error {
	if !g.IsRepository() {
		return fmt.Errorf("not a Git repository")
	}

	cmd := exec.Command("git", "push")
	cmd.Dir = g.workDir
	return cmd.Run()
}

// Pull pulls changes from the remote repository.
func (g *GitIntegration) Pull() error {
	if !g.IsRepository() {
		return fmt.Errorf("not a Git repository")
	}

	cmd := exec.Command("git", "pull")
	cmd.Dir = g.workDir
	return cmd.Run()
}

// CreateBranch creates a new branch.
func (g *GitIntegration) CreateBranch(branchName string) error {
	if !g.IsRepository() {
		return fmt.Errorf("not a Git repository")
	}

	cmd := exec.Command("git", "checkout", "-b", branchName)
	cmd.Dir = g.workDir
	return cmd.Run()
}

// SwitchBranch switches to an existing branch.
func (g *GitIntegration) SwitchBranch(branchName string) error {
	if !g.IsRepository() {
		return fmt.Errorf("not a Git repository")
	}

	cmd := exec.Command("git", "checkout", branchName)
	cmd.Dir = g.workDir
	return cmd.Run()
}

// ListBranches returns the list of local and remote branches.
func (g *GitIntegration) ListBranches() ([]string, error) {
	if !g.IsRepository() {
		return nil, fmt.Errorf("not a Git repository")
	}

	// For now, return empty list - would need to implement branch listing
	return []string{}, nil
}

// ListRemotes returns the list of remote repositories.
func (g *GitIntegration) ListRemotes() ([]string, error) {
	if !g.IsRepository() {
		return nil, fmt.Errorf("not a Git repository")
	}

	// For now, return empty list - would need to implement remote listing
	return []string{}, nil
}

// GetWorkingDirectory returns the working directory.
func (g *GitIntegration) GetWorkingDirectory() string {
	return g.workDir
}

// SetWorkingDirectory sets the working directory.
func (g *GitIntegration) SetWorkingDirectory(workDir string) {
	g.mu.Lock()
	g.workDir = workDir
	g.mu.Unlock()
}

// Close cleans up Git integration resources.
func (g *GitIntegration) Close() error {
	// No resources to clean up
	return nil
}
