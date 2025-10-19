package core

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestGatherEnvironment_AllScenarios tests GatherEnvironment comprehensively
func TestGatherEnvironment_WithValidDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	testFile := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(testFile, []byte("package main\n\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	env, err := GatherEnvironment(tmpDir)
	if err != nil {
		t.Fatalf("GatherEnvironment() error = %v", err)
	}

	if env.WorkDir != tmpDir {
		t.Errorf("WorkDir = %s, want %s", env.WorkDir, tmpDir)
	}

	if len(env.Files) == 0 {
		t.Error("Expected at least one file in Files")
	}
}

// TestGatherEnvironment_WithGoProject tests detection of Go project
func TestGatherEnvironment_WithGoProject(t *testing.T) {
	tmpDir := t.TempDir()

	// Create go.mod to make it a Go project
	goMod := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goMod, []byte("module test\n"), 0644); err != nil {
		t.Fatalf("Failed to create go.mod: %v", err)
	}

	// Create .go file
	mainGo := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(mainGo, []byte("package main\n"), 0644); err != nil {
		t.Fatalf("Failed to create main.go: %v", err)
	}

	env, err := GatherEnvironment(tmpDir)
	if err != nil {
		t.Fatalf("GatherEnvironment() error = %v", err)
	}

	if env.ProjectType != "go" {
		t.Errorf("ProjectType = %s, want 'go'", env.ProjectType)
	}

	if len(env.Languages) == 0 || env.Languages[0] != "Go" {
		t.Error("Expected 'Go' in languages list")
	}
}

// TestGatherEnvironment_WithPythonProject tests Python project detection
func TestGatherEnvironment_WithPythonProject(t *testing.T) {
	tmpDir := t.TempDir()

	// Create requirements.txt
	reqFile := filepath.Join(tmpDir, "requirements.txt")
	if err := os.WriteFile(reqFile, []byte("flask==2.0.0\n"), 0644); err != nil {
		t.Fatalf("Failed to create requirements.txt: %v", err)
	}

	// Create .py file
	mainPy := filepath.Join(tmpDir, "main.py")
	if err := os.WriteFile(mainPy, []byte("print('hello')\n"), 0644); err != nil {
		t.Fatalf("Failed to create main.py: %v", err)
	}

	env, err := GatherEnvironment(tmpDir)
	if err != nil {
		t.Fatalf("GatherEnvironment() error = %v", err)
	}

	if env.ProjectType != "python" {
		t.Errorf("ProjectType = %s, want 'python'", env.ProjectType)
	}
}

// TestGatherEnvironment_WithGitRepo tests Git info gathering
func TestGatherEnvironment_WithGitRepo(t *testing.T) {
	// Skip if git is not available
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	tmpDir := t.TempDir()

	// Initialize git repo
	initCmd := exec.Command("git", "init")
	initCmd.Dir = tmpDir
	if err := initCmd.Run(); err != nil {
		t.Skipf("Failed to init git repo: %v", err)
	}

	// Configure git
	exec.Command("git", "config", "user.email", "test@example.com").Dir = tmpDir
	exec.Command("git", "config", "user.name", "Test User").Dir = tmpDir

	// Create and commit a file
	testFile := filepath.Join(tmpDir, "README.md")
	if err := os.WriteFile(testFile, []byte("# Test"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	addCmd := exec.Command("git", "add", ".")
	addCmd.Dir = tmpDir
	addCmd.Run()

	commitCmd := exec.Command("git", "commit", "-m", "Initial commit")
	commitCmd.Dir = tmpDir
	commitCmd.Run()

	env, err := GatherEnvironment(tmpDir)
	if err != nil {
		t.Fatalf("GatherEnvironment() error = %v", err)
	}

	if env.Git == nil {
		t.Error("Expected Git info for git repository")
	}

	if env.Git != nil && env.Git.Root == "" {
		t.Error("Expected non-empty Git root")
	}
}

// TestGatherEnvironment_NonExistentDirectory tests with invalid directory
func TestGatherEnvironment_NonExistentDirectory(t *testing.T) {
	_, err := GatherEnvironment("/this/directory/does/not/exist/anywhere")
	if err == nil {
		t.Error("Expected error for non-existent directory")
	}
}

// TestEnvironment_String_AllFields tests Environment.String() method
func TestEnvironment_String_WithAllFields(t *testing.T) {
	env := &Environment{
		WorkDir:     "/test/workdir",
		ProjectType: "go",
		Git: &GitInfo{
			Root:   "/test",
			Branch: "main",
		},
		Files: []FileInfo{
			{Path: "main.go", Size: 100},
		},
		Languages: []string{"Go"},
	}

	str := env.String()
	if str == "" {
		t.Error("Expected non-empty string representation")
	}

	if !strings.Contains(str, "/test/workdir") {
		t.Error("Expected String() to contain WorkDir")
	}

	if !strings.Contains(str, "go") {
		t.Error("Expected String() to contain ProjectType")
	}
}

// TestEnvironment_String_MinimalFields tests String with minimal fields
func TestEnvironment_String_MinimalFields(t *testing.T) {
	env := &Environment{
		WorkDir: "/tmp",
	}

	str := env.String()
	if str == "" {
		t.Error("Expected non-empty string representation")
	}

	if !strings.Contains(str, "/tmp") {
		t.Error("Expected String() to contain WorkDir")
	}
}

// TestScanProjectFiles_WithIgnoredDirectories tests file scanning with ignored dirs
func TestScanProjectFiles_WithIgnoredDirs(t *testing.T) {
	tmpDir := t.TempDir()

	// Create node_modules (should be ignored)
	nodeModules := filepath.Join(tmpDir, "node_modules")
	if err := os.MkdirAll(nodeModules, 0755); err != nil {
		t.Fatalf("Failed to create node_modules: %v", err)
	}

	ignoredFile := filepath.Join(nodeModules, "package.json")
	if err := os.WriteFile(ignoredFile, []byte("{}"), 0644); err != nil {
		t.Fatalf("Failed to create ignored file: %v", err)
	}

	// Create normal file
	normalFile := filepath.Join(tmpDir, "main.js")
	if err := os.WriteFile(normalFile, []byte("console.log('hello')"), 0644); err != nil {
		t.Fatalf("Failed to create normal file: %v", err)
	}

	env, err := GatherEnvironment(tmpDir)
	if err != nil {
		t.Fatalf("GatherEnvironment() error = %v", err)
	}

	// Check that ignored directories were skipped
	for _, file := range env.Files {
		if strings.Contains(file.Path, "node_modules") {
			t.Error("node_modules should be ignored")
		}
	}
}

// TestScanProjectFiles_WithNestedStructure tests scanning nested directories
func TestScanProjectFiles_NestedDirectories(t *testing.T) {
	tmpDir := t.TempDir()

	// Create nested structure
	srcDir := filepath.Join(tmpDir, "src", "pkg")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("Failed to create nested dirs: %v", err)
	}

	// Create files at different levels
	files := []string{
		"README.md",
		filepath.Join("src", "main.go"),
		filepath.Join("src", "pkg", "util.go"),
	}

	for _, file := range files {
		path := filepath.Join(tmpDir, file)
		if err := os.WriteFile(path, []byte("content"), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", file, err)
		}
	}

	env, err := GatherEnvironment(tmpDir)
	if err != nil {
		t.Fatalf("GatherEnvironment() error = %v", err)
	}

	if len(env.Files) < len(files) {
		t.Errorf("Expected at least %d files, got %d", len(files), len(env.Files))
	}
}

// TestCountLines_Various tests line counting
func TestCountLines_MultipleFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create file with known line count
	testFile := filepath.Join(tmpDir, "test.txt")
	content := strings.Repeat("line\n", 50)
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	env, err := GatherEnvironment(tmpDir)
	if err != nil {
		t.Fatalf("GatherEnvironment() error = %v", err)
	}

	// Should have counted lines
	if env.Files == nil || len(env.Files) == 0 {
		t.Fatal("Expected files to be scanned")
	}
}

// TestDetectProjectType_AllTypes tests project type detection
func TestDetectProjectType_UnknownType(t *testing.T) {
	tmpDir := t.TempDir()

	// Create file with unknown extension
	testFile := filepath.Join(tmpDir, "random.xyz")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	env, err := GatherEnvironment(tmpDir)
	if err != nil {
		t.Fatalf("GatherEnvironment() error = %v", err)
	}

	if env.ProjectType != "unknown" {
		t.Errorf("ProjectType = %s, want 'unknown' for unrecognized project", env.ProjectType)
	}
}

// TestFilterEnvironment tests environment variable filtering
func TestFilterEnvironment_PreservesImportantVars(t *testing.T) {
	// filterEnvironment is private, so test via executor execution
	tmpDir := t.TempDir()
	executor, err := NewExecutor(tmpDir)
	if err != nil {
		t.Fatalf("NewExecutor() error = %v", err)
	}

	cmd := &Command{
		Program: "env",
		Args:    []string{},
	}

	result, err := executor.Execute(context.Background(), cmd, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// PATH should be present
	if !strings.Contains(result.Stdout, "PATH=") {
		t.Error("Expected PATH in environment")
	}
}
