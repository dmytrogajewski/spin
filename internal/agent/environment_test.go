package agent

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmytrogajewski/spin/pkg/alg/execx"
)

func TestGatherEnvironment(t *testing.T) {
	t.Parallel()
	workDir := t.TempDir()

	// Create some test files.
	testFile := filepath.Join(workDir, "test.go")
	err := os.WriteFile(testFile, []byte("package main\n\nfunc main() {\n\tprintln(\"hello\")\n}"), 0o600)
	require.NoError(t, err)

	env, err := GatherEnvironment(context.Background(), workDir)
	require.NoError(t, err)
	assert.NotNil(t, env)
	assert.Equal(t, workDir, env.WorkDir)
	assert.NotNil(t, env.OS)
	assert.NotNil(t, env.Files)
	assert.NotNil(t, env.Environment)
	assert.NotNil(t, env.Languages)
}

func TestGatherEnvironment_WithOptions(t *testing.T) {
	t.Parallel()
	workDir := t.TempDir()

	// Create some test files.
	for i := range 5 {
		testFile := filepath.Join(workDir, "test"+string(rune(i+'0'))+".go")
		err := os.WriteFile(testFile, []byte("package main"), 0o600)
		require.NoError(t, err)
	}

	// Test with max files limit.
	env, err := GatherEnvironment(context.Background(), workDir, WithMaxFiles(2))
	require.NoError(t, err)
	assert.NotNil(t, env)
	assert.LessOrEqual(t, len(env.Files), 2)
}

func TestGatherEnvironment_WithSkipGit(t *testing.T) {
	t.Parallel()
	workDir := t.TempDir()

	env, err := GatherEnvironment(context.Background(), workDir, WithSkipGit(true))
	require.NoError(t, err)
	assert.NotNil(t, env)
	assert.Nil(t, env.Git)
}

func TestGatherEnvironment_WithMaxDepth(t *testing.T) {
	t.Parallel()
	workDir := t.TempDir()

	// Create nested directories.
	subDir := filepath.Join(workDir, "subdir")
	err := os.Mkdir(subDir, 0o750)
	require.NoError(t, err)

	testFile := filepath.Join(subDir, "test.go")
	err = os.WriteFile(testFile, []byte("package main"), 0o600)
	require.NoError(t, err)

	env, err := GatherEnvironment(context.Background(), workDir, WithMaxDepth(1))
	require.NoError(t, err)
	assert.NotNil(t, env)
}

func TestGatherEnvironment_EmptyDirectory(t *testing.T) {
	t.Parallel()
	workDir := t.TempDir()

	env, err := GatherEnvironment(context.Background(), workDir)
	require.NoError(t, err)
	assert.NotNil(t, env)
	assert.Equal(t, workDir, env.WorkDir)
	assert.Empty(t, env.Files)
}

func TestGatherEnvironment_NonExistentDirectory(t *testing.T) {
	t.Parallel()

	env, err := GatherEnvironment(context.Background(), "/non/existent/directory")
	require.Error(t, err)
	assert.Nil(t, env)
}

func TestEnvironmentOption_WithMaxFiles(t *testing.T) {
	t.Parallel()

	cfg := &environmentConfig{}
	opt := WithMaxFiles(100)
	opt(cfg)
	assert.Equal(t, 100, cfg.maxFiles)
}

func TestEnvironmentOption_WithMaxDepth(t *testing.T) {
	t.Parallel()

	cfg := &environmentConfig{}
	opt := WithMaxDepth(5)
	opt(cfg)
	assert.Equal(t, 5, cfg.maxDepth)
}

func TestEnvironmentOption_WithSkipGit(t *testing.T) {
	t.Parallel()

	cfg := &environmentConfig{}
	opt := WithSkipGit(true)
	opt(cfg)
	assert.True(t, cfg.skipGit)
}

func TestGatherOSInfo(t *testing.T) {
	t.Parallel()

	osInfo := gatherOSInfo(context.Background())
	assert.NotEmpty(t, osInfo.OS)
	assert.NotEmpty(t, osInfo.Arch)
}

func TestDetectLanguage(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	// Create sample files so content-based detection works for ambiguous extensions.
	writeFile := func(name, content string) string {
		p := filepath.Join(dir, name)
		require.NoError(t, os.WriteFile(p, []byte(content), 0o600))

		return p
	}

	tests := []struct {
		name     string
		path     string
		filename string
		expected string
	}{
		{"Go file", writeFile("main.go", "package main"), "main.go", "Go"},
		{"Python file", writeFile("main.py", "print('hi')"), "main.py", "Python"},
		{"JavaScript file", writeFile("app.js", "console.log('hi')"), "app.js", "JavaScript"},
		{"TypeScript file", writeFile("app.ts", "const x: string = 'hi'"), "app.ts", "TypeScript"},
		{"Java file", writeFile("Main.java", "class Main {}"), "Main.java", "Java"},
		{"C file", writeFile("main.c", "#include <stdio.h>"), "main.c", "C"},
		{"C++ file", writeFile("main.cpp", "#include <iostream>"), "main.cpp", "C++"},
		{"Rust file", writeFile("main.rs", "fn main() {}"), "main.rs", "Rust"},
		{"Makefile", writeFile("Makefile", "all:\n\techo hi"), "Makefile", "Makefile"},
		{"Dockerfile", writeFile("Dockerfile", "FROM alpine"), "Dockerfile", "Dockerfile"},
		{"Unknown extension", writeFile("file.xyz", ""), "file.xyz", "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := detectLanguage(tt.path, tt.filename)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCountLines(t *testing.T) {
	t.Parallel()
	workDir := t.TempDir()

	// Create a test file with known line count.
	testFile := filepath.Join(workDir, "test.txt")
	content := "line1\nline2\nline3\n"
	err := os.WriteFile(testFile, []byte(content), 0o600)
	require.NoError(t, err)

	lines := countLines(testFile)
	assert.Equal(t, 3, lines)
}

func TestCountLines_NonExistentFile(t *testing.T) {
	t.Parallel()

	lines := countLines("/non/existent/file.txt")
	assert.Equal(t, 0, lines)
}

func TestDetectProjectType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		files    []FileInfo
		expected string
	}{
		{
			name: "Go project",
			files: []FileInfo{
				{Path: "main.go", Language: "Go"},
				{Path: "go.mod", Language: ""},
			},
			expected: "go",
		},
		{
			name: "Python project",
			files: []FileInfo{
				{Path: "main.py", Language: "Python"},
				{Path: "requirements.txt", Language: ""},
			},
			expected: "python",
		},
		{
			name: "JavaScript project",
			files: []FileInfo{
				{Path: "index.js", Language: "JavaScript"},
				{Path: "package.json", Language: ""},
			},
			expected: "nodejs",
		},
		{
			name: "Unknown project",
			files: []FileInfo{
				{Path: "file.txt", Language: ""},
			},
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := detectProjectType(tt.files)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDetectLanguages(t *testing.T) {
	t.Parallel()

	files := []FileInfo{
		{Path: "main.go", Language: "Go"},
		{Path: "test.py", Language: "Python"},
		{Path: "script.js", Language: "JavaScript"},
		{Path: "README.md", Language: "Markdown"},
	}

	languages := detectLanguages(files)
	assert.Contains(t, languages, "Go")
	assert.Contains(t, languages, "Python")
	assert.Contains(t, languages, "JavaScript")
	assert.NotContains(t, languages, "Markdown") // Markup, not programming.
	assert.Len(t, languages, 3)
}

func TestDetectLanguages_EmptyFiles(t *testing.T) {
	t.Parallel()

	languages := detectLanguages([]FileInfo{})
	assert.Empty(t, languages)
}

func TestFilterEnvironment(t *testing.T) {
	t.Parallel()

	env := []string{
		"PATH=/usr/bin:/bin",
		"HOME=/home/user",
		"USER=testuser",
		"SECRET_KEY=secret123",
		"API_KEY=api123",
		"DEBUG=true",
		"PORT=8080",
	}

	filtered := execx.FilterEnvironment(env, sensitivePrefixes, sensitiveSubstrings)

	// Should include non-sensitive variables.
	assert.Contains(t, filtered, "PATH")
	assert.Contains(t, filtered, "HOME")
	assert.Contains(t, filtered, "USER")
	assert.Contains(t, filtered, "DEBUG")
	assert.Contains(t, filtered, "PORT")

	// Should exclude sensitive variables.
	assert.NotContains(t, filtered, "SECRET_KEY")
	assert.NotContains(t, filtered, "API_KEY")
}

func TestEnvironment_String(t *testing.T) {
	t.Parallel()

	env := &Environment{
		WorkDir:     "/test/dir",
		ProjectType: "go",
		Languages:   []string{"go", "python"},
		OS: OSInfo{
			OS:   "linux",
			Arch: "amd64",
		},
	}

	str := env.String()
	assert.NotEmpty(t, str)
	assert.Contains(t, str, "/test/dir")
	assert.Contains(t, str, "go")
	assert.Contains(t, str, "linux")
}

func TestEnvironment_Structure(t *testing.T) {
	t.Parallel()

	env := &Environment{
		WorkDir:     "/test/dir",
		ProjectType: "go",
		Languages:   []string{"go"},
		Files: []FileInfo{
			{Path: "main.go", Size: 100, Language: "Go", Lines: 10},
		},
		Environment: map[string]string{
			"PATH": "/usr/bin",
		},
		OS: OSInfo{
			OS:     "linux",
			Arch:   "amd64",
			Kernel: "5.4.0",
			Shell:  "/bin/bash",
		},
		Git: &GitInfo{
			Root:       "/test/dir",
			Branch:     "main",
			HasChanges: false,
			Remotes: []Remote{
				{Name: "origin", URL: "https://github.com/user/repo.git"},
			},
		},
	}

	assert.Equal(t, "/test/dir", env.WorkDir)
	assert.Equal(t, "go", env.ProjectType)
	assert.Len(t, env.Languages, 1)
	assert.Len(t, env.Files, 1)
	assert.Len(t, env.Environment, 1)
	assert.NotNil(t, env.OS)
	assert.NotNil(t, env.Git)
}

func TestOSInfo_Structure(t *testing.T) {
	t.Parallel()

	osInfo := OSInfo{
		OS:     "linux",
		Arch:   "amd64",
		Kernel: "5.4.0",
		Shell:  "/bin/bash",
	}

	assert.Equal(t, "linux", osInfo.OS)
	assert.Equal(t, "amd64", osInfo.Arch)
	assert.Equal(t, "5.4.0", osInfo.Kernel)
	assert.Equal(t, "/bin/bash", osInfo.Shell)
}

func TestGitInfo_Structure(t *testing.T) {
	t.Parallel()

	gitInfo := GitInfo{
		Root:           "/path/to/repo",
		Branch:         "main",
		HasChanges:     true,
		UntrackedFiles: []string{"file1.txt", "file2.txt"},
		Remotes: []Remote{
			{Name: "origin", URL: "https://github.com/user/repo.git"},
		},
	}

	assert.Equal(t, "/path/to/repo", gitInfo.Root)
	assert.Equal(t, "main", gitInfo.Branch)
	assert.True(t, gitInfo.HasChanges)
	assert.Len(t, gitInfo.UntrackedFiles, 2)
	assert.Len(t, gitInfo.Remotes, 1)
	assert.Equal(t, "origin", gitInfo.Remotes[0].Name)
	assert.Equal(t, "https://github.com/user/repo.git", gitInfo.Remotes[0].URL)
}

func TestFileInfo_Structure(t *testing.T) {
	t.Parallel()

	fileInfo := FileInfo{
		Path:     "test.go",
		Size:     1024,
		Language: "Go",
		Lines:    50,
	}

	assert.Equal(t, "test.go", fileInfo.Path)
	assert.Equal(t, int64(1024), fileInfo.Size)
	assert.Equal(t, "Go", fileInfo.Language)
	assert.Equal(t, 50, fileInfo.Lines)
}

func TestRemote_Structure(t *testing.T) {
	t.Parallel()

	remote := Remote{
		Name: "origin",
		URL:  "https://github.com/user/repo.git",
	}

	assert.Equal(t, "origin", remote.Name)
	assert.Equal(t, "https://github.com/user/repo.git", remote.URL)
}
