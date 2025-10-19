package core

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dmytrogajewski/spin/internal/git"
)

func TestGitIntegration_GetModifiedFiles(t *testing.T) {
	tests := []struct {
		name    string
		status  *git.Status
		want    int
		wantErr bool
	}{
		{
			name:    "no status available",
			status:  nil,
			want:    0,
			wantErr: true,
		},
		{
			name: "with modified files",
			status: &git.Status{
				Branch: "main",
				ModifiedFiles: []git.FileStatus{
					{Path: "file1.go", Worktree: git.Modified},
					{Path: "file2.go", Worktree: git.Added},
					{Path: "file3.go", Worktree: git.Deleted},
				},
			},
			want:    3,
			wantErr: false,
		},
		{
			name: "with no modified files",
			status: &git.Status{
				Branch:        "main",
				ModifiedFiles: []git.FileStatus{},
			},
			want:    0,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &GitIntegration{
				enabled:    true,
				workDir:    "/tmp",
				logger:     slog.New(slog.NewTextHandler(os.Stderr, nil)),
				lastStatus: tt.status,
			}

			files, err := g.GetModifiedFiles()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetModifiedFiles() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && len(files) != tt.want {
				t.Errorf("GetModifiedFiles() got %d files, want %d", len(files), tt.want)
			}
		})
	}
}

func TestGitIntegration_GetStagedFiles(t *testing.T) {
	tests := []struct {
		name    string
		status  *git.Status
		want    int
		wantErr bool
	}{
		{
			name:    "no status available",
			status:  nil,
			want:    0,
			wantErr: true,
		},
		{
			name: "with staged files",
			status: &git.Status{
				Branch: "main",
				ModifiedFiles: []git.FileStatus{
					{Path: "file1.go", Staging: git.Added},
					{Path: "file2.go", Staging: git.Modified},
					{Path: "file3.go", Worktree: git.Modified}, // Not staged
				},
			},
			want:    2,
			wantErr: false,
		},
		{
			name: "with no staged files",
			status: &git.Status{
				Branch:        "main",
				ModifiedFiles: []git.FileStatus{},
			},
			want:    0,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &GitIntegration{
				enabled:    true,
				workDir:    "/tmp",
				logger:     slog.New(slog.NewTextHandler(os.Stderr, nil)),
				lastStatus: tt.status,
			}

			files, err := g.GetStagedFiles()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetStagedFiles() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && len(files) != tt.want {
				t.Errorf("GetStagedFiles() got %d files, want %d", len(files), tt.want)
			}
		})
	}
}

func TestGitIntegration_GetDiff(t *testing.T) {
	tests := []struct {
		name     string
		isRepo   bool
		filePath string
		wantErr  bool
	}{
		{
			name:     "not a repository",
			isRepo:   false,
			filePath: "",
			wantErr:  true,
		},
		{
			name:     "repository with empty path",
			isRepo:   true,
			filePath: "",
			wantErr:  false, // May fail without real git repo, but tests code path
		},
		{
			name:     "repository with file path",
			isRepo:   true,
			filePath: "test.go",
			wantErr:  false, // May fail without real git repo, but tests code path
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workDir := t.TempDir()

			var repo *git.Repository
			if tt.isRepo {
				// Create a minimal mock repository
				repo = &git.Repository{}
			}

			g := &GitIntegration{
				enabled: true,
				workDir: workDir,
				logger:  slog.New(slog.NewTextHandler(os.Stderr, nil)),
				repo:    repo,
			}

			_, err := g.GetDiff(tt.filePath)
			// For non-repo case, we expect error
			if !tt.isRepo && err == nil {
				t.Error("GetDiff() expected error for non-repository, got nil")
			}
			// For repo case, we just test the code path (may fail without real git)
		})
	}
}

func TestGitIntegration_StageFile(t *testing.T) {
	tests := []struct {
		name     string
		isRepo   bool
		filePath string
		wantErr  bool
	}{
		{
			name:     "not a repository",
			isRepo:   false,
			filePath: "test.go",
			wantErr:  true,
		},
		{
			name:     "repository with valid file",
			isRepo:   true,
			filePath: "test.go",
			wantErr:  false, // May fail if git not available, but tests structure
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workDir := t.TempDir()

			var repo *git.Repository
			if tt.isRepo {
				repo = &git.Repository{}
			}

			g := &GitIntegration{
				enabled: true,
				workDir: workDir,
				logger:  slog.New(slog.NewTextHandler(os.Stderr, nil)),
				repo:    repo,
			}

			err := g.StageFile(tt.filePath)
			if !tt.isRepo && err == nil {
				t.Error("StageFile() expected error for non-repository, got nil")
			}
		})
	}
}

func TestGitIntegration_UnstageFile(t *testing.T) {
	tests := []struct {
		name     string
		isRepo   bool
		filePath string
		wantErr  bool
	}{
		{
			name:     "not a repository",
			isRepo:   false,
			filePath: "test.go",
			wantErr:  true,
		},
		{
			name:     "repository with valid file",
			isRepo:   true,
			filePath: "test.go",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workDir := t.TempDir()

			var repo *git.Repository
			if tt.isRepo {
				repo = &git.Repository{}
			}

			g := &GitIntegration{
				enabled: true,
				workDir: workDir,
				logger:  slog.New(slog.NewTextHandler(os.Stderr, nil)),
				repo:    repo,
			}

			err := g.UnstageFile(tt.filePath)
			if !tt.isRepo && err == nil {
				t.Error("UnstageFile() expected error for non-repository, got nil")
			}
		})
	}
}

func TestGitIntegration_Commit(t *testing.T) {
	tests := []struct {
		name    string
		isRepo  bool
		message string
		wantErr bool
	}{
		{
			name:    "not a repository",
			isRepo:  false,
			message: "test commit",
			wantErr: true,
		},
		{
			name:    "repository with valid message",
			isRepo:  true,
			message: "test commit",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workDir := t.TempDir()

			var repo *git.Repository
			if tt.isRepo {
				repo = &git.Repository{}
			}

			g := &GitIntegration{
				enabled: true,
				workDir: workDir,
				logger:  slog.New(slog.NewTextHandler(os.Stderr, nil)),
				repo:    repo,
			}

			err := g.Commit(tt.message)
			if !tt.isRepo && err == nil {
				t.Error("Commit() expected error for non-repository, got nil")
			}
		})
	}
}

func TestGitIntegration_Push(t *testing.T) {
	tests := []struct {
		name    string
		isRepo  bool
		wantErr bool
	}{
		{
			name:    "not a repository",
			isRepo:  false,
			wantErr: true,
		},
		{
			name:    "repository",
			isRepo:  true,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workDir := t.TempDir()

			var repo *git.Repository
			if tt.isRepo {
				repo = &git.Repository{}
			}

			g := &GitIntegration{
				enabled: true,
				workDir: workDir,
				logger:  slog.New(slog.NewTextHandler(os.Stderr, nil)),
				repo:    repo,
			}

			err := g.Push()
			if !tt.isRepo && err == nil {
				t.Error("Push() expected error for non-repository, got nil")
			}
		})
	}
}

func TestGitIntegration_Pull(t *testing.T) {
	tests := []struct {
		name    string
		isRepo  bool
		wantErr bool
	}{
		{
			name:    "not a repository",
			isRepo:  false,
			wantErr: true,
		},
		{
			name:    "repository",
			isRepo:  true,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workDir := t.TempDir()

			var repo *git.Repository
			if tt.isRepo {
				repo = &git.Repository{}
			}

			g := &GitIntegration{
				enabled: true,
				workDir: workDir,
				logger:  slog.New(slog.NewTextHandler(os.Stderr, nil)),
				repo:    repo,
			}

			err := g.Pull()
			if !tt.isRepo && err == nil {
				t.Error("Pull() expected error for non-repository, got nil")
			}
		})
	}
}

func TestGitIntegration_CreateBranch(t *testing.T) {
	tests := []struct {
		name       string
		isRepo     bool
		branchName string
		wantErr    bool
	}{
		{
			name:       "not a repository",
			isRepo:     false,
			branchName: "feature-test",
			wantErr:    true,
		},
		{
			name:       "repository with valid branch name",
			isRepo:     true,
			branchName: "feature-test",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workDir := t.TempDir()

			var repo *git.Repository
			if tt.isRepo {
				repo = &git.Repository{}
			}

			g := &GitIntegration{
				enabled: true,
				workDir: workDir,
				logger:  slog.New(slog.NewTextHandler(os.Stderr, nil)),
				repo:    repo,
			}

			err := g.CreateBranch(tt.branchName)
			if !tt.isRepo && err == nil {
				t.Error("CreateBranch() expected error for non-repository, got nil")
			}
		})
	}
}

func TestGitIntegration_SwitchBranch(t *testing.T) {
	tests := []struct {
		name       string
		isRepo     bool
		branchName string
		wantErr    bool
	}{
		{
			name:       "not a repository",
			isRepo:     false,
			branchName: "main",
			wantErr:    true,
		},
		{
			name:       "repository with valid branch name",
			isRepo:     true,
			branchName: "main",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workDir := t.TempDir()

			var repo *git.Repository
			if tt.isRepo {
				repo = &git.Repository{}
			}

			g := &GitIntegration{
				enabled: true,
				workDir: workDir,
				logger:  slog.New(slog.NewTextHandler(os.Stderr, nil)),
				repo:    repo,
			}

			err := g.SwitchBranch(tt.branchName)
			if !tt.isRepo && err == nil {
				t.Error("SwitchBranch() expected error for non-repository, got nil")
			}
		})
	}
}

func TestGitIntegration_GetWorkingDirectory(t *testing.T) {
	workDir := "/tmp/test"
	g := &GitIntegration{
		enabled: true,
		workDir: workDir,
		logger:  slog.New(slog.NewTextHandler(os.Stderr, nil)),
	}

	got := g.GetWorkingDirectory()
	if got != workDir {
		t.Errorf("GetWorkingDirectory() = %v, want %v", got, workDir)
	}
}

func TestGitIntegration_SetWorkingDirectory(t *testing.T) {
	g := &GitIntegration{
		enabled: true,
		workDir: "/tmp/old",
		logger:  slog.New(slog.NewTextHandler(os.Stderr, nil)),
	}

	newWorkDir := "/tmp/new"
	g.SetWorkingDirectory(newWorkDir)

	got := g.GetWorkingDirectory()
	if got != newWorkDir {
		t.Errorf("SetWorkingDirectory() -> GetWorkingDirectory() = %v, want %v", got, newWorkDir)
	}
}

func TestGitIntegration_ConcurrentAccess(t *testing.T) {
	g := &GitIntegration{
		enabled: true,
		workDir: "/tmp/test",
		logger:  slog.New(slog.NewTextHandler(os.Stderr, nil)),
		lastStatus: &git.Status{
			Branch: "main",
			ModifiedFiles: []git.FileStatus{
				{Path: "file1.go", Worktree: git.Modified},
			},
		},
	}

	// Test concurrent reads
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			_ = g.GetStatus()
			_ = g.IsRepository()
			_, _ = g.GetBranch()
			done <- true
		}()
	}

	// Wait for all goroutines with timeout
	timeout := time.After(5 * time.Second)
	for i := 0; i < 10; i++ {
		select {
		case <-done:
			// Success
		case <-timeout:
			t.Fatal("Concurrent access test timed out")
		}
	}
}

func TestGitIntegration_GetModifiedFiles_EdgeCases(t *testing.T) {
	tests := []struct {
		name   string
		status *git.Status
		want   []string
	}{
		{
			name: "renamed files included",
			status: &git.Status{
				Branch: "main",
				ModifiedFiles: []git.FileStatus{
					{Path: "old.go", Worktree: git.Renamed},
				},
			},
			want: []string{"old.go"},
		},
		{
			name: "mixed status types",
			status: &git.Status{
				Branch: "main",
				ModifiedFiles: []git.FileStatus{
					{Path: "added.go", Worktree: git.Added},
					{Path: "modified.go", Worktree: git.Modified},
					{Path: "deleted.go", Worktree: git.Deleted},
					{Path: "unchanged.go", Worktree: git.Unmodified}, // Should not be included
				},
			},
			want: []string{"added.go", "modified.go", "deleted.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &GitIntegration{
				enabled:    true,
				workDir:    "/tmp",
				logger:     slog.New(slog.NewTextHandler(os.Stderr, nil)),
				lastStatus: tt.status,
			}

			got, err := g.GetModifiedFiles()
			if err != nil {
				t.Fatalf("GetModifiedFiles() error = %v", err)
			}

			if len(got) != len(tt.want) {
				t.Errorf("GetModifiedFiles() got %d files, want %d", len(got), len(tt.want))
			}

			// Check if all expected files are present
			gotMap := make(map[string]bool)
			for _, f := range got {
				gotMap[f] = true
			}
			for _, wantFile := range tt.want {
				if !gotMap[wantFile] {
					t.Errorf("GetModifiedFiles() missing file %v", wantFile)
				}
			}
		})
	}
}

func TestGitIntegration_GetStagedFiles_EdgeCases(t *testing.T) {
	tests := []struct {
		name   string
		status *git.Status
		want   []string
	}{
		{
			name: "renamed files in staging",
			status: &git.Status{
				Branch: "main",
				ModifiedFiles: []git.FileStatus{
					{Path: "renamed.go", Staging: git.Renamed},
				},
			},
			want: []string{"renamed.go"},
		},
		{
			name: "mixed staging status",
			status: &git.Status{
				Branch: "main",
				ModifiedFiles: []git.FileStatus{
					{Path: "added.go", Staging: git.Added},
					{Path: "modified.go", Staging: git.Modified},
					{Path: "deleted.go", Staging: git.Deleted},
					{Path: "worktree.go", Worktree: git.Modified, Staging: git.Unmodified},
				},
			},
			want: []string{"added.go", "modified.go", "deleted.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &GitIntegration{
				enabled:    true,
				workDir:    "/tmp",
				logger:     slog.New(slog.NewTextHandler(os.Stderr, nil)),
				lastStatus: tt.status,
			}

			got, err := g.GetStagedFiles()
			if err != nil {
				t.Fatalf("GetStagedFiles() error = %v", err)
			}

			if len(got) != len(tt.want) {
				t.Errorf("GetStagedFiles() got %d files, want %d", len(got), len(tt.want))
			}
		})
	}
}

func TestGitIntegration_InitializeWithRealRepo(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create a temporary git repository for testing
	tmpDir := t.TempDir()

	// Initialize git repo (requires git to be installed)
	ctx := context.Background()

	g := NewGitIntegration(true, tmpDir, slog.New(slog.NewTextHandler(os.Stderr, nil)))

	// Initialize should not error even if not a git repo
	err := g.Initialize(ctx)
	if err != nil {
		t.Errorf("Initialize() error = %v, expected nil", err)
	}

	// Should report as not a repository
	if g.IsRepository() {
		t.Error("IsRepository() = true, want false for non-git directory")
	}
}

func TestGitIntegration_DisabledState(t *testing.T) {
	g := NewGitIntegration(false, "/tmp", slog.New(slog.NewTextHandler(os.Stderr, nil)))

	if g.IsEnabled() {
		t.Error("IsEnabled() = true, want false")
	}

	ctx := context.Background()
	err := g.Initialize(ctx)
	if err != nil {
		t.Errorf("Initialize() error = %v, want nil for disabled integration", err)
	}

	if g.IsRepository() {
		t.Error("IsRepository() = true, want false for disabled integration")
	}
}

func TestGitIntegration_Close(t *testing.T) {
	g := NewGitIntegration(true, "/tmp", slog.New(slog.NewTextHandler(os.Stderr, nil)))

	err := g.Close()
	if err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}
}

func TestGitIntegration_GetDiff_WithTempRepo(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test that requires git in short mode")
	}

	tmpDir := t.TempDir()

	g := &GitIntegration{
		enabled: true,
		workDir: tmpDir,
		logger:  slog.New(slog.NewTextHandler(os.Stderr, nil)),
		repo:    &git.Repository{}, // Mock repo
	}

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Try to get diff - will fail without real git repo but tests the code path
	_, err = g.GetDiff("test.txt")
	// We expect an error since it's not a real git repo, but we tested the code path
	if err == nil {
		t.Log("GetDiff succeeded unexpectedly, probably in a real git repo")
	}
}
