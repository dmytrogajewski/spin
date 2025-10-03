package core

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestContext_Structure tests Context struct serialization
func TestContext_Structure(t *testing.T) {
	ctx := &Context{
		OS: OSInfo{
			OS:   "linux",
			Arch: "amd64",
		},
		WorkDir:     "/test",
		ProjectType: "go",
		Languages:   []string{"Go"},
	}

	// Can serialize to JSON
	data, err := json.Marshal(ctx)
	if err != nil {
		t.Fatalf("Marshal() failed: %v", err)
	}

	// Can deserialize from JSON
	var decoded Context
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Unmarshal() failed: %v", err)
	}

	if decoded.OS.OS != ctx.OS.OS {
		t.Errorf("OS mismatch: got %s, want %s", decoded.OS.OS, ctx.OS.OS)
	}
}

// TestGatherOSInfo tests OS information gathering
func TestGatherOSInfo(t *testing.T) {
	osInfo := gatherOSInfo()

	// Must have OS and Arch
	if osInfo.OS == "" {
		t.Error("OS should not be empty")
	}
	if osInfo.Arch == "" {
		t.Error("Arch should not be empty")
	}

	// OS should match runtime.GOOS
	if osInfo.OS != runtime.GOOS {
		t.Errorf("OS mismatch: got %s, want %s", osInfo.OS, runtime.GOOS)
	}

	// Arch should match runtime.GOARCH
	if osInfo.Arch != runtime.GOARCH {
		t.Errorf("Arch mismatch: got %s, want %s", osInfo.Arch, runtime.GOARCH)
	}
}

// TestGatherOSInfo_Shell tests shell detection
func TestGatherOSInfo_Shell(t *testing.T) {
	// Save original SHELL
	originalShell := os.Getenv("SHELL")
	defer func() {
		if originalShell != "" {
			os.Setenv("SHELL", originalShell)
		} else {
			os.Unsetenv("SHELL")
		}
	}()

	// With SHELL env var
	os.Setenv("SHELL", "/bin/zsh")

	osInfo := gatherOSInfo()
	if osInfo.Shell != "/bin/zsh" {
		t.Errorf("Shell mismatch: got %s, want /bin/zsh", osInfo.Shell)
	}
}

// TestGatherGitInfo_ValidRepo tests Git info gathering in a valid repository
func TestGatherGitInfo_ValidRepo(t *testing.T) {
	// Skip if git is not available
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not available")
	}

	// Setup: Create temp git repo
	dir := setupTestGitRepo(t)

	gitInfo, err := gatherGitInfo(dir)
	if err != nil {
		t.Fatalf("gatherGitInfo() failed: %v", err)
	}

	if gitInfo == nil {
		t.Fatal("gitInfo should not be nil for valid repo")
	}

	// Should detect branch
	if gitInfo.Branch == "" {
		t.Error("Branch should not be empty")
	}

	// Should have root path
	if gitInfo.Root == "" {
		t.Error("Root should not be empty")
	}
}

// TestGatherGitInfo_NotGitRepo tests behavior in non-git directory
func TestGatherGitInfo_NotGitRepo(t *testing.T) {
	dir := t.TempDir()

	gitInfo, err := gatherGitInfo(dir)
	if err != nil {
		t.Fatalf("gatherGitInfo() should not error on non-git dir: %v", err)
	}

	// Should return nil for non-git directories
	if gitInfo != nil {
		t.Error("gitInfo should be nil for non-git directory")
	}
}

// TestGatherGitInfo_WithChanges tests detection of uncommitted changes
func TestGatherGitInfo_WithChanges(t *testing.T) {
	// Skip if git is not available
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not available")
	}

	dir := setupTestGitRepoWithChanges(t)

	gitInfo, err := gatherGitInfo(dir)
	if err != nil {
		t.Fatalf("gatherGitInfo() failed: %v", err)
	}

	if !gitInfo.HasChanges {
		t.Error("HasChanges should be true for dirty repo")
	}
}

// TestScanProjectFiles tests project file scanning
func TestScanProjectFiles(t *testing.T) {
	dir := setupTestProject(t, map[string]string{
		"main.go":       "package main\n\nfunc main() {}\n",
		"utils/util.go": "package utils\n",
		"README.md":     "# Test\n",
	})

	files, err := scanProjectFiles(dir, 1000, 10)
	if err != nil {
		t.Fatalf("scanProjectFiles() failed: %v", err)
	}

	if len(files) != 3 {
		t.Errorf("Expected 3 files, got %d", len(files))
	}

	// Check Go file detected
	goFile := findFile(files, "main.go")
	if goFile == nil {
		t.Fatal("main.go not found in scanned files")
	}

	if goFile.Language != "Go" {
		t.Errorf("Language mismatch: got %s, want Go", goFile.Language)
	}

	if goFile.Lines != 3 {
		t.Errorf("Lines mismatch: got %d, want 3", goFile.Lines)
	}
}

// TestScanProjectFiles_SkipsHidden tests that hidden directories are skipped
func TestScanProjectFiles_SkipsHidden(t *testing.T) {
	dir := setupTestProject(t, map[string]string{
		"main.go":             "package main\n",
		".git/config":         "test",
		"node_modules/pkg.js": "test",
		".hidden/file.go":     "test",
	})

	files, err := scanProjectFiles(dir, 1000, 10)
	if err != nil {
		t.Fatalf("scanProjectFiles() failed: %v", err)
	}

	// Should only include main.go
	if len(files) != 1 {
		t.Errorf("Expected 1 file, got %d", len(files))
	}

	if len(files) > 0 && files[0].Path != "main.go" {
		t.Errorf("Expected main.go, got %s", files[0].Path)
	}
}

// TestDetectProjectType_Go tests Go project detection
func TestDetectProjectType_Go(t *testing.T) {
	files := []FileInfo{
		{Path: "go.mod"},
		{Path: "main.go", Language: "Go"},
	}

	projectType := detectProjectType(files)
	if projectType != "go" {
		t.Errorf("Expected 'go', got '%s'", projectType)
	}
}

// TestDetectProjectType_Python tests Python project detection
func TestDetectProjectType_Python(t *testing.T) {
	files := []FileInfo{
		{Path: "setup.py"},
		{Path: "main.py", Language: "Python"},
	}

	projectType := detectProjectType(files)
	if projectType != "python" {
		t.Errorf("Expected 'python', got '%s'", projectType)
	}
}

// TestDetectProjectType_NodeJS tests Node.js project detection
func TestDetectProjectType_NodeJS(t *testing.T) {
	files := []FileInfo{
		{Path: "package.json"},
		{Path: "index.js", Language: "JavaScript"},
	}

	projectType := detectProjectType(files)
	if projectType != "nodejs" {
		t.Errorf("Expected 'nodejs', got '%s'", projectType)
	}
}

// TestDetectProjectType_Rust tests Rust project detection
func TestDetectProjectType_Rust(t *testing.T) {
	files := []FileInfo{
		{Path: "Cargo.toml"},
		{Path: "main.rs", Language: "Rust"},
	}

	projectType := detectProjectType(files)
	if projectType != "rust" {
		t.Errorf("Expected 'rust', got '%s'", projectType)
	}
}

// TestDetectProjectType_Unknown tests unknown project type
func TestDetectProjectType_Unknown(t *testing.T) {
	files := []FileInfo{
		{Path: "readme.txt"},
	}

	projectType := detectProjectType(files)
	if projectType != "unknown" {
		t.Errorf("Expected 'unknown', got '%s'", projectType)
	}
}

// TestDetectLanguages tests language detection from files
func TestDetectLanguages(t *testing.T) {
	files := []FileInfo{
		{Path: "main.go", Language: "Go"},
		{Path: "utils.go", Language: "Go"},
		{Path: "script.py", Language: "Python"},
	}

	languages := detectLanguages(files)

	if len(languages) != 2 {
		t.Errorf("Expected 2 languages, got %d", len(languages))
	}

	hasGo := false
	hasPython := false
	for _, lang := range languages {
		if lang == "Go" {
			hasGo = true
		}
		if lang == "Python" {
			hasPython = true
		}
	}

	if !hasGo {
		t.Error("Expected Go in languages")
	}
	if !hasPython {
		t.Error("Expected Python in languages")
	}
}

// TestFilterEnvironment_IncludesSafe tests that safe vars are included
func TestFilterEnvironment_IncludesSafe(t *testing.T) {
	env := []string{
		"PATH=/usr/bin",
		"HOME=/home/user",
		"USER=testuser",
	}

	filtered := filterEnvironment(env)

	if filtered["PATH"] != "/usr/bin" {
		t.Errorf("PATH mismatch: got %s, want /usr/bin", filtered["PATH"])
	}
	if filtered["HOME"] != "/home/user" {
		t.Errorf("HOME mismatch: got %s, want /home/user", filtered["HOME"])
	}
	if filtered["USER"] != "testuser" {
		t.Errorf("USER mismatch: got %s, want testuser", filtered["USER"])
	}
}

// TestFilterEnvironment_ExcludesSensitive tests that sensitive vars are excluded
func TestFilterEnvironment_ExcludesSensitive(t *testing.T) {
	env := []string{
		"AWS_ACCESS_KEY_ID=secret123",
		"OPENAI_API_KEY=sk-xyz",
		"GITHUB_TOKEN=ghp_abc",
		"PATH=/usr/bin",
	}

	filtered := filterEnvironment(env)

	// Should not contain sensitive vars
	if _, exists := filtered["AWS_ACCESS_KEY_ID"]; exists {
		t.Error("AWS_ACCESS_KEY_ID should be filtered out")
	}
	if _, exists := filtered["OPENAI_API_KEY"]; exists {
		t.Error("OPENAI_API_KEY should be filtered out")
	}
	if _, exists := filtered["GITHUB_TOKEN"]; exists {
		t.Error("GITHUB_TOKEN should be filtered out")
	}

	// Should contain safe vars
	if _, exists := filtered["PATH"]; !exists {
		t.Error("PATH should not be filtered out")
	}
}

// TestGather_FullContext tests full context gathering
func TestGather_FullContext(t *testing.T) {
	dir := setupTestProject(t, map[string]string{
		"go.mod":  "module test\n",
		"main.go": "package main\n",
	})

	ctx, err := Gather(dir)
	if err != nil {
		t.Fatalf("Gather() failed: %v", err)
	}

	// OS info populated
	if ctx.OS.OS == "" {
		t.Error("OS should not be empty")
	}
	if ctx.OS.Arch == "" {
		t.Error("Arch should not be empty")
	}

	// WorkDir set
	if ctx.WorkDir != dir {
		t.Errorf("WorkDir mismatch: got %s, want %s", ctx.WorkDir, dir)
	}

	// Files scanned
	if len(ctx.Files) == 0 {
		t.Error("Files should not be empty")
	}

	// Project type detected
	if ctx.ProjectType != "go" {
		t.Errorf("ProjectType mismatch: got %s, want go", ctx.ProjectType)
	}

	// Languages detected
	hasGo := false
	for _, lang := range ctx.Languages {
		if lang == "Go" {
			hasGo = true
		}
	}
	if !hasGo {
		t.Error("Expected Go in languages")
	}

	// Environment filtered
	if len(ctx.Environment) == 0 {
		t.Error("Environment should not be empty")
	}
}

// TestGather_NonGitDirectory tests gathering in non-git directory
func TestGather_NonGitDirectory(t *testing.T) {
	dir := t.TempDir()

	ctx, err := Gather(dir)
	if err != nil {
		t.Fatalf("Gather() failed: %v", err)
	}

	// Git info should be nil
	if ctx.Git != nil {
		t.Error("Git should be nil for non-git directory")
	}
}

// TestGather_WithOptions tests gathering with options
func TestGather_WithOptions(t *testing.T) {
	// Create directory with many files
	dir := t.TempDir()
	for i := 0; i < 200; i++ {
		filename := filepath.Join(dir, "file"+string(rune('0'+i%10))+".txt")
		if err := os.WriteFile(filename, []byte("test"), 0644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}
	}

	ctx, err := Gather(dir, WithMaxFiles(50))
	if err != nil {
		t.Fatalf("Gather() failed: %v", err)
	}

	// Should respect max files
	if len(ctx.Files) > 50 {
		t.Errorf("Expected <= 50 files, got %d", len(ctx.Files))
	}
}

// TestGather_WithMaxDepth tests max depth option
func TestGather_WithMaxDepth(t *testing.T) {
	// Create nested directory structure
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "a/b/c/d/e"), 0755); err != nil {
		t.Fatalf("failed to create directories: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "root.txt"), []byte("root"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a/level1.txt"), []byte("level1"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a/b/level2.txt"), []byte("level2"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a/b/c/level3.txt"), []byte("level3"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a/b/c/d/level4.txt"), []byte("level4"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	ctx, err := Gather(dir, WithMaxDepth(2))
	if err != nil {
		t.Fatalf("Gather() failed: %v", err)
	}

	// Should not include files deeper than level 2
	for _, file := range ctx.Files {
		depth := strings.Count(file.Path, string(filepath.Separator))
		if depth > 2 {
			t.Errorf("File %s exceeds max depth of 2 (depth: %d)", file.Path, depth)
		}
	}
}

// TestGather_WithSkipGit tests skip git option
func TestGather_WithSkipGit(t *testing.T) {
	// Skip if git is not available
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not available")
	}

	dir := setupTestGitRepo(t)

	ctx, err := Gather(dir, WithSkipGit(true))
	if err != nil {
		t.Fatalf("Gather() failed: %v", err)
	}

	// Git info should be nil
	if ctx.Git != nil {
		t.Error("Git should be nil when WithSkipGit(true) is used")
	}
}

// TestContext_String tests context serialization
func TestContext_String(t *testing.T) {
	ctx := &Context{
		OS: OSInfo{
			OS:     "linux",
			Arch:   "amd64",
			Kernel: "5.15.0",
			Shell:  "/bin/bash",
		},
		WorkDir:     "/test",
		ProjectType: "go",
		Languages:   []string{"Go", "Python"},
		Git: &GitInfo{
			Root:           "/test",
			Branch:         "main",
			HasChanges:     true,
			UntrackedFiles: []string{"new.txt"},
			Remotes: []Remote{
				{Name: "origin", URL: "git@github.com:user/repo.git"},
			},
		},
		Files: []FileInfo{
			{Path: "main.go", Language: "Go", Lines: 100},
			{Path: "utils.py", Language: "Python", Lines: 50},
		},
	}

	s := ctx.String()

	// Should contain key information
	if s == "" {
		t.Error("String() should not be empty")
	}

	// Check for specific content
	expectedStrings := []string{
		"linux", "amd64", "/test", "go",
		"Go", "Python", "main", "dirty",
		"5.15.0", "/bin/bash",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(s, expected) {
			t.Errorf("String() missing expected content: %s", expected)
		}
	}
}

// TestContext_String_CleanGit tests context string with clean git repo
func TestContext_String_CleanGit(t *testing.T) {
	ctx := &Context{
		OS: OSInfo{
			OS:   "linux",
			Arch: "amd64",
		},
		WorkDir:     "/test",
		ProjectType: "go",
		Languages:   []string{"Go"},
		Git: &GitInfo{
			Branch:     "main",
			HasChanges: false,
		},
	}

	s := ctx.String()
	if !strings.Contains(s, "clean") {
		t.Error("String() should indicate clean git status")
	}
}

// TestContext_NoSensitiveData tests that no sensitive data leaks
func TestContext_NoSensitiveData(t *testing.T) {
	// Save original env
	originalKey := os.Getenv("OPENAI_API_KEY")
	defer func() {
		if originalKey != "" {
			os.Setenv("OPENAI_API_KEY", originalKey)
		} else {
			os.Unsetenv("OPENAI_API_KEY")
		}
	}()

	os.Setenv("OPENAI_API_KEY", "sk-secret123")

	ctx, err := Gather(t.TempDir())
	if err != nil {
		t.Fatalf("Gather() failed: %v", err)
	}

	// Should not contain API key
	for key := range ctx.Environment {
		if key == "OPENAI_API_KEY" {
			t.Error("OPENAI_API_KEY should be filtered out")
		}
	}
}

// BenchmarkGather benchmarks context gathering performance
func BenchmarkGather(b *testing.B) {
	// Setup test project
	dir := b.TempDir()
	for i := 0; i < 100; i++ {
		filename := filepath.Join(dir, "file"+string(rune('0'+i%10))+".go")
		if err := os.WriteFile(filename, []byte("package main\n"), 0644); err != nil {
			b.Fatalf("failed to write file: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Gather(dir)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Helper: setupTestProject creates a test project with given files
func setupTestProject(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()

	for path, content := range files {
		fullPath := filepath.Join(dir, path)

		// Create parent directories if needed
		parentDir := filepath.Dir(fullPath)
		if err := os.MkdirAll(parentDir, 0755); err != nil {
			t.Fatalf("failed to create directory %s: %v", parentDir, err)
		}

		// Write file
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write file %s: %v", fullPath, err)
		}
	}

	return dir
}

// Helper: setupTestGitRepo creates a git repository for testing
func setupTestGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git init failed: %v", err)
	}

	// Configure git
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git config user.email failed: %v", err)
	}

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git config user.name failed: %v", err)
	}

	// Create initial commit
	testFile := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	cmd = exec.Command("git", "add", "test.txt")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git add failed: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", "initial commit")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git commit failed: %v", err)
	}

	return dir
}

// Helper: setupTestGitRepoWithChanges creates a repo with uncommitted changes
func setupTestGitRepoWithChanges(t *testing.T) string {
	t.Helper()
	dir := setupTestGitRepo(t)

	// Create uncommitted file
	newFile := filepath.Join(dir, "new.txt")
	if err := os.WriteFile(newFile, []byte("new"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	// Also modify existing file to ensure changes are detected
	testFile := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(testFile, []byte("modified"), 0644); err != nil {
		t.Fatalf("failed to modify file: %v", err)
	}

	return dir
}

// Helper: findFile finds a file by path in FileInfo slice
func findFile(files []FileInfo, path string) *FileInfo {
	for i := range files {
		if files[i].Path == path {
			return &files[i]
		}
	}
	return nil
}
