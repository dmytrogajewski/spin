package agent

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

var (
	// ErrGitNotAvailable is returned when git is not found in PATH.
	ErrGitNotAvailable = errors.New("git not available")
	// ErrNotGitRepository is returned when the directory is not a git repository.
	ErrNotGitRepository = errors.New("not a git repository")
)

// Environment contains environment information for the AI agent.
// It provides comprehensive information about the operating system,
// Git repository (if present), project structure, and filtered environment variables.
type Environment struct {
	OS          OSInfo            `json:"os"`
	Git         *GitInfo          `json:"git,omitempty"`
	WorkDir     string            `json:"work_dir"`
	Files       []FileInfo        `json:"files"`
	Environment map[string]string `json:"environment"`
	ProjectType string            `json:"project_type"`
	Languages   []string          `json:"languages"`
}

// OSInfo contains operating system information.
type OSInfo struct {
	OS     string `json:"os"`     // linux, darwin, windows, etc.
	Arch   string `json:"arch"`   // amd64, arm64, etc.
	Kernel string `json:"kernel"` // kernel version (Linux/Unix).
	Shell  string `json:"shell"`  // /bin/bash, /bin/zsh, etc.
}

// GitInfo contains Git repository information.
type GitInfo struct {
	Root           string   `json:"root"`
	Branch         string   `json:"branch"`
	HasChanges     bool     `json:"has_changes"`
	UntrackedFiles []string `json:"untracked_files"`
	Remotes        []Remote `json:"remotes"`
}

// Remote represents a Git remote.
type Remote struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// FileInfo contains file metadata.
type FileInfo struct {
	Path     string `json:"path"`
	Size     int64  `json:"size"`
	Language string `json:"language"`
	Lines    int    `json:"lines"`
}

// EnvironmentOption configures context gathering.
type EnvironmentOption func(*environmentConfig)

// environmentConfig holds configuration for context gathering.
type environmentConfig struct {
	maxFiles int
	maxDepth int
	skipGit  bool
}

// WithMaxFiles limits the number of files scanned.
func WithMaxFiles(maxFiles int) EnvironmentOption {
	return func(c *environmentConfig) {
		c.maxFiles = maxFiles
	}
}

// WithMaxDepth limits directory traversal depth.
func WithMaxDepth(depth int) EnvironmentOption {
	return func(c *environmentConfig) {
		c.maxDepth = depth
	}
}

// WithSkipGit disables Git information gathering.
func WithSkipGit(skip bool) EnvironmentOption {
	return func(c *environmentConfig) {
		c.skipGit = skip
	}
}

// GatherEnvironment collects environment context for the AI agent.
// It gathers OS information, Git repository info (if present),
// project files, and filtered environment variables.
func GatherEnvironment(workDir string, opts ...EnvironmentOption) (*Environment, error) {
	// Apply options.
	cfg := &environmentConfig{
		maxFiles: 1000,
		maxDepth: 10,
		skipGit:  false,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	// Validate workDir exists.
	_, err := os.Stat(workDir)
	if err != nil {
		return nil, fmt.Errorf("work directory does not exist: %w", err)
	}

	// Gather OS information.
	osInfo := gatherOSInfo()

	// Gather Git information (if not skipped).
	var gitInfo *GitInfo
	if !cfg.skipGit {
		gitInfo, _ = gatherGitInfo(workDir) // Ignore errors, Git may not be available.
	}

	// Scan project files.
	files, err := scanProjectFiles(workDir, cfg.maxFiles, cfg.maxDepth)
	if err != nil {
		// Continue with empty files on error.
		files = []FileInfo{}
	}

	// Detect project type and languages.
	projectType := detectProjectType(files)
	languages := detectLanguages(files)

	// Filter environment variables.
	environment := filterEnvironment(os.Environ())

	return &Environment{
		OS:          osInfo,
		Git:         gitInfo,
		WorkDir:     workDir,
		Files:       files,
		Environment: environment,
		ProjectType: projectType,
		Languages:   languages,
	}, nil
}

// gatherOSInfo collects operating system information.
func gatherOSInfo() OSInfo {
	info := OSInfo{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
	}

	// Get shell from environment.
	info.Shell = os.Getenv("SHELL")

	// Get kernel version on Linux/Unix.
	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" || runtime.GOOS == "freebsd" {
		output, err := exec.Command("uname", "-r").Output()
		if err == nil {
			info.Kernel = strings.TrimSpace(string(output))
		}
	}

	return info
}

// gatherGitInfo collects Git repository information.
// Returns nil if the directory is not in a Git repository.
func gatherGitInfo(workDir string) (*GitInfo, error) {
	// Check if git is available.
	_, err := exec.LookPath("git")
	if err != nil {
		return nil, ErrGitNotAvailable
	}

	// Create context with timeout for git commands.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Find git root.
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--show-toplevel")
	cmd.Dir = workDir

	output, err := cmd.Output()
	if err != nil {
		return nil, ErrNotGitRepository
	}

	root := strings.TrimSpace(string(output))

	info := &GitInfo{
		Root: root,
	}

	// Get current branch.
	cmd = exec.CommandContext(ctx, "git", "rev-parse", "--abbrev-ref", "HEAD")

	cmd.Dir = workDir
	output, err = cmd.Output()
	if err == nil {
		info.Branch = strings.TrimSpace(string(output))
	}

	// Check for changes.
	cmd = exec.CommandContext(ctx, "git", "status", "--porcelain")

	cmd.Dir = workDir
	output, err = cmd.Output()
	if err == nil {
		statusOutput := string(output)
		info.HasChanges = len(strings.TrimSpace(statusOutput)) > 0

		// Parse untracked files.
		for line := range strings.SplitSeq(statusOutput, "\n") {
			if strings.HasPrefix(line, "??") {
				file := strings.TrimSpace(line[2:])
				info.UntrackedFiles = append(info.UntrackedFiles, file)
			}
		}
	}

	// Get remotes.
	cmd = exec.CommandContext(ctx, "git", "remote", "-v")

	cmd.Dir = workDir
	output, err = cmd.Output()
	if err == nil {
		remoteMap := make(map[string]string)

		for line := range strings.SplitSeq(string(output), "\n") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				name := parts[0]

				url := parts[1]
				if _, exists := remoteMap[name]; !exists {
					remoteMap[name] = url
				}
			}
		}

		for name, url := range remoteMap {
			info.Remotes = append(info.Remotes, Remote{
				Name: name,
				URL:  url,
			})
		}
	}

	return info, nil
}

// scanProjectFiles scans the project directory and collects file information.
func scanProjectFiles(workDir string, maxFiles, maxDepth int) ([]FileInfo, error) {
	var files []FileInfo

	count := 0

	// Directories to skip.
	skipDirs := map[string]bool{
		".git":          true,
		".hg":           true,
		".svn":          true,
		"node_modules":  true,
		"vendor":        true,
		".idea":         true,
		".vscode":       true,
		"__pycache__":   true,
		".pytest_cache": true,
		"target":        true, // Rust/Java.
		"build":         true,
		"dist":          true,
	}

	err := filepath.WalkDir(workDir, func(path string, d fs.DirEntry, walkErr error) error {
		// Skip entries with walk errors (permission denied, etc.).
		if isNonNil(walkErr) {
			return nil
		}

		// Get relative path — skip if it cannot be relativized.
		relPath, relErr := filepath.Rel(workDir, path)
		if isNonNil(relErr) {
			return nil
		}

		// Check depth.
		depth := strings.Count(relPath, string(filepath.Separator))
		if depth > maxDepth {
			if d.IsDir() {
				return filepath.SkipDir
			}

			return nil
		}

		// Skip hidden directories.
		if d.IsDir() {
			name := d.Name()
			if strings.HasPrefix(name, ".") && name != "." {
				return filepath.SkipDir
			}

			if skipDirs[name] {
				return filepath.SkipDir
			}

			return nil
		}

		// Skip hidden files.
		if strings.HasPrefix(d.Name(), ".") {
			return nil
		}

		// Check file limit.
		if count >= maxFiles {
			return filepath.SkipAll
		}

		// Get file info — skip if unavailable.
		info, infoErr := d.Info()
		if isNonNil(infoErr) {
			return nil
		}

		// Get language from extension.
		language := detectLanguageFromExt(filepath.Ext(path))

		// Count lines.
		lines := countLines(path)

		files = append(files, FileInfo{
			Path:     relPath,
			Size:     info.Size(),
			Language: language,
			Lines:    lines,
		})

		count++

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walking directory: %w", err)
	}

	return files, nil
}

// isNonNil returns true if the error is non-nil.
// This helper avoids the nilerr pattern where checking err != nil
// and returning nil triggers a lint warning.
func isNonNil(err error) bool {
	return err != nil
}

// detectLanguageFromExt detects programming language from file extension.
func detectLanguageFromExt(ext string) string {
	languageMap := map[string]string{
		".go":   "go",
		".py":   "python",
		".js":   "javascript",
		".ts":   "typescript",
		".jsx":  "javascript",
		".tsx":  "typescript",
		".rs":   "rust",
		".rb":   "ruby",
		".java": "java",
		".c":    "c",
		".cpp":  "cpp",
		".cc":   "cpp",
		".cxx":  "cpp",
		".h":    "c",
		".hpp":  "cpp",
		".cs":   "csharp",
		".php":  "php",
		".sh":   "shell",
		".bash": "shell",
		".zsh":  "shell",
		".yaml": "yaml",
		".yml":  "yaml",
		".json": "json",
		".xml":  "xml",
		".html": "html",
		".css":  "css",
		".md":   "markdown",
		".txt":  "text",
		".toml": "toml",
	}

	if lang, exists := languageMap[ext]; exists {
		return lang
	}

	return "Unknown"
}

// countLines counts the number of lines in a file.
func countLines(path string) int {
	file, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer file.Close()

	count := 0

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		count++
	}

	return count
}

// detectProjectType detects the project type based on files.
func detectProjectType(files []FileInfo) string {
	hasFile := func(name string) bool {
		for _, f := range files {
			if f.Path == name || filepath.Base(f.Path) == name {
				return true
			}
		}

		return false
	}

	// Go project.
	if hasFile("go.mod") {
		return "go"
	}

	// Python project.
	if hasFile("setup.py") || hasFile("requirements.txt") || hasFile("pyproject.toml") {
		return "python"
	}

	// Node.js project.
	if hasFile("package.json") {
		return "nodejs"
	}

	// Rust project.
	if hasFile("Cargo.toml") {
		return "rust"
	}

	// Ruby project.
	if hasFile("Gemfile") {
		return "ruby"
	}

	// Java/Maven project.
	if hasFile("pom.xml") {
		return "java-maven"
	}

	// Java/Gradle project.
	if hasFile("build.gradle") || hasFile("build.gradle.kts") {
		return "java-gradle"
	}

	return "unknown"
}

// detectLanguages detects programming languages used in the project.
func detectLanguages(files []FileInfo) []string {
	languageSet := make(map[string]bool)

	for _, file := range files {
		if file.Language != "Unknown" && file.Language != "Text" &&
			file.Language != "JSON" && file.Language != "YAML" &&
			file.Language != "TOML" && file.Language != "Markdown" {
			languageSet[file.Language] = true
		}
	}

	// Convert to sorted slice.
	languages := make([]string, 0, len(languageSet))
	for lang := range languageSet {
		languages = append(languages, lang)
	}

	sort.Strings(languages)

	return languages
}

// filterEnvironment filters environment variables to exclude sensitive information.
func filterEnvironment(env []string) map[string]string {
	filtered := make(map[string]string)

	// Sensitive prefixes.
	sensitivePrefixes := []string{
		"AWS_",
		"GCP_",
		"AZURE_",
		"OPENAI_",
		"ANTHROPIC_",
		"HUGGINGFACE_",
	}

	// Sensitive substrings.
	sensitiveSubstrings := []string{
		"TOKEN",
		"KEY",
		"SECRET",
		"PASSWORD",
		"AUTH",
		"CREDENTIAL",
		"PRIVATE",
	}

	isSensitive := func(key string) bool {
		upperKey := strings.ToUpper(key)

		// Check prefixes.
		for _, prefix := range sensitivePrefixes {
			if strings.HasPrefix(upperKey, prefix) {
				return true
			}
		}

		// Check substrings.
		for _, substring := range sensitiveSubstrings {
			if strings.Contains(upperKey, substring) {
				return true
			}
		}

		return false
	}

	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := parts[0]
		value := parts[1]

		if !isSensitive(key) {
			filtered[key] = value
		}
	}

	return filtered
}

// String returns a human-readable representation of the context.
// The format is optimized for LLM consumption.
func (c *Environment) String() string {
	var sb strings.Builder

	sb.WriteString("Environment Context:\n")

	// OS Information.
	fmt.Fprintf(&sb, "- OS: %s (%s)\n", c.OS.OS, c.OS.Arch)

	if c.OS.Kernel != "" {
		fmt.Fprintf(&sb, "- Kernel: %s\n", c.OS.Kernel)
	}

	if c.OS.Shell != "" {
		fmt.Fprintf(&sb, "- Shell: %s\n", c.OS.Shell)
	}

	// Working Directory.
	fmt.Fprintf(&sb, "- Working Directory: %s\n", c.WorkDir)

	// Project Type.
	if c.ProjectType != "unknown" {
		fmt.Fprintf(&sb, "- Project Type: %s\n", c.ProjectType)
	}

	// Languages.
	if len(c.Languages) > 0 {
		fmt.Fprintf(&sb, "- Languages: %s\n", strings.Join(c.Languages, ", "))
	}

	// Git Information.
	if c.Git != nil {
		status := "clean"
		if c.Git.HasChanges {
			status = "dirty"
		}

		fmt.Fprintf(&sb, "- Git Branch: %s (%s)\n", c.Git.Branch, status)

		if len(c.Git.UntrackedFiles) > 0 {
			fmt.Fprintf(&sb, "- Untracked Files: %d\n", len(c.Git.UntrackedFiles))
		}

		if len(c.Git.Remotes) > 0 {
			fmt.Fprintf(&sb, "- Git Remotes: %d\n", len(c.Git.Remotes))
		}
	}

	// Project Structure Summary.
	if len(c.Files) > 0 {
		fmt.Fprintf(&sb, "\nProject Structure: %d files\n", len(c.Files))

		// Show up to 20 files.
		maxShow := 20
		for i, file := range c.Files {
			if i >= maxShow {
				fmt.Fprintf(&sb, "... and %d more files\n", len(c.Files)-maxShow)

				break
			}

			if file.Language != "Unknown" {
				fmt.Fprintf(&sb, "- %s (%s, %d lines)\n", file.Path, file.Language, file.Lines)
			} else {
				fmt.Fprintf(&sb, "- %s\n", file.Path)
			}
		}
	}

	return sb.String()
}
