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

	"github.com/dmytrogajewski/spin/pkg/alg/execx"
)

const (
	defaultEnvMaxFiles  = 1000
	defaultEnvMaxDepth  = 10
	envDiscoveryTimeout = 5 * time.Second
	minSplitParts       = 2
	languageUnknown     = "Unknown"
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
func GatherEnvironment(ctx context.Context, workDir string, opts ...EnvironmentOption) (*Environment, error) {
	// Apply options.
	cfg := &environmentConfig{
		maxFiles: defaultEnvMaxFiles,
		maxDepth: defaultEnvMaxDepth,
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
	osInfo := gatherOSInfo(ctx)

	// Gather Git information (if not skipped).
	var gitInfo *GitInfo
	if !cfg.skipGit {
		gitInfo, _ = gatherGitInfo(ctx, workDir) // Ignore errors, Git may not be available.
	}

	// Scan project files.
	files, err := scanProjectFiles(ctx, workDir, cfg.maxFiles, cfg.maxDepth)
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
func gatherOSInfo(ctx context.Context) OSInfo {
	info := OSInfo{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
	}

	// Get shell from environment.
	info.Shell = os.Getenv("SHELL")

	// Get kernel version on Linux/Unix.
	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" || runtime.GOOS == "freebsd" {
		output, err := exec.CommandContext(ctx, "uname", "-r").Output()
		if err == nil {
			info.Kernel = strings.TrimSpace(string(output))
		}
	}

	return info
}

// gatherGitInfo collects Git repository information.
// Returns nil if the directory is not in a Git repository.
func gatherGitInfo(parentCtx context.Context, workDir string) (*GitInfo, error) {
	_, err := exec.LookPath("git")
	if err != nil {
		return nil, ErrGitNotAvailable
	}

	ctx, cancel := context.WithTimeout(parentCtx, envDiscoveryTimeout)
	defer cancel()

	root, err := gitCommand(ctx, workDir, "rev-parse", "--show-toplevel")
	if err != nil {
		return nil, ErrNotGitRepository
	}

	info := &GitInfo{Root: strings.TrimSpace(root)}

	branch, err := gitCommand(ctx, workDir, "rev-parse", "--abbrev-ref", "HEAD")
	if err == nil {
		info.Branch = strings.TrimSpace(branch)
	}

	gatherGitStatus(ctx, workDir, info)
	gatherGitRemotes(ctx, workDir, info)

	return info, nil
}

// gitCommand runs a git command and returns its output.
func gitCommand(ctx context.Context, workDir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = workDir

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return string(output), nil
}

// gatherGitStatus collects git status (changes and untracked files).
func gatherGitStatus(ctx context.Context, workDir string, info *GitInfo) {
	output, err := gitCommand(ctx, workDir, "status", "--porcelain")
	if err != nil {
		return
	}

	info.HasChanges = strings.TrimSpace(output) != ""

	for line := range strings.SplitSeq(output, "\n") {
		if strings.HasPrefix(line, "??") {
			info.UntrackedFiles = append(info.UntrackedFiles, strings.TrimSpace(line[2:]))
		}
	}
}

// gatherGitRemotes collects git remote information.
func gatherGitRemotes(ctx context.Context, workDir string, info *GitInfo) {
	output, err := gitCommand(ctx, workDir, "remote", "-v")
	if err != nil {
		return
	}

	remoteMap := make(map[string]string)

	for line := range strings.SplitSeq(output, "\n") {
		parts := strings.Fields(line)
		if len(parts) >= minSplitParts {
			if _, exists := remoteMap[parts[0]]; !exists {
				remoteMap[parts[0]] = parts[1]
			}
		}
	}

	for name, url := range remoteMap {
		info.Remotes = append(info.Remotes, Remote{Name: name, URL: url})
	}
}

// fileScanner holds state for scanning project files.
type fileScanner struct {
	workDir  string
	maxFiles int
	maxDepth int
	files    []FileInfo
	count    int
	skipDirs map[string]bool
}

// newFileScanner creates a new file scanner.
func newFileScanner(workDir string, maxFiles, maxDepth int) *fileScanner {
	return &fileScanner{
		workDir:  workDir,
		maxFiles: maxFiles,
		maxDepth: maxDepth,
		skipDirs: map[string]bool{
			".git": true, ".hg": true, ".svn": true,
			"node_modules": true, "vendor": true,
			".idea": true, ".vscode": true,
			"__pycache__": true, ".pytest_cache": true,
			"target": true, "build": true, "dist": true,
		},
	}
}

// scanProjectFiles scans the project directory and collects file information.
func scanProjectFiles(ctx context.Context, workDir string, maxFiles, maxDepth int) ([]FileInfo, error) {
	scanner := newFileScanner(workDir, maxFiles, maxDepth)

	err := filepath.WalkDir(workDir, func(path string, d fs.DirEntry, walkErr error) error {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}

		return scanner.visit(path, d, walkErr)
	})
	if err != nil {
		return nil, fmt.Errorf("walking directory: %w", err)
	}

	return scanner.files, nil
}

// visit is the WalkDir callback for scanning files.
func (s *fileScanner) visit(path string, d fs.DirEntry, walkErr error) error {
	if isNonNil(walkErr) {
		return nil
	}

	relPath, relErr := filepath.Rel(s.workDir, path)
	if isNonNil(relErr) {
		return nil
	}

	depth := strings.Count(relPath, string(filepath.Separator))
	if depth > s.maxDepth {
		if d.IsDir() {
			return filepath.SkipDir
		}

		return nil
	}

	if d.IsDir() {
		return s.handleDirectory(d)
	}

	return s.handleFile(path, relPath, d)
}

// handleDirectory decides whether to skip a directory.
func (s *fileScanner) handleDirectory(d fs.DirEntry) error {
	name := d.Name()
	if strings.HasPrefix(name, ".") && name != "." {
		return filepath.SkipDir
	}

	if s.skipDirs[name] {
		return filepath.SkipDir
	}

	return nil
}

// handleFile processes a single file entry.
func (s *fileScanner) handleFile(path, relPath string, d fs.DirEntry) error {
	if strings.HasPrefix(d.Name(), ".") {
		return nil
	}

	if s.count >= s.maxFiles {
		return filepath.SkipAll
	}

	info, infoErr := d.Info()
	if isNonNil(infoErr) {
		return nil
	}

	s.files = append(s.files, FileInfo{
		Path:     relPath,
		Size:     info.Size(),
		Language: detectLanguageFromExt(filepath.Ext(path)),
		Lines:    countLines(path),
	})

	s.count++

	return nil
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

	return languageUnknown
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
		if file.Language != languageUnknown && file.Language != "Text" &&
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

// sensitivePrefixes lists env var prefixes that indicate sensitive data.
var sensitivePrefixes = []string{
	"AWS_", "GCP_", "AZURE_", "OPENAI_", "ANTHROPIC_", "HUGGINGFACE_",
}

// sensitiveSubstrings lists env var key substrings that indicate sensitive data.
var sensitiveSubstrings = []string{
	"TOKEN", "KEY", "SECRET", "PASSWORD", "AUTH", "CREDENTIAL", "PRIVATE",
}

// filterEnvironment filters environment variables to exclude sensitive information.
func filterEnvironment(env []string) map[string]string {
	return execx.FilterEnvironment(env, sensitivePrefixes, sensitiveSubstrings)
}

// String returns a human-readable representation of the context.
// The format is optimized for LLM consumption.
func (c *Environment) String() string {
	var sb strings.Builder

	sb.WriteString("Environment Context:\n")
	c.writeOSInfo(&sb)
	fmt.Fprintf(&sb, "- Working Directory: %s\n", c.WorkDir)
	c.writeProjectInfo(&sb)
	c.writeGitInfo(&sb)
	c.writeFilesSummary(&sb)

	return sb.String()
}

// writeOSInfo writes OS information to the builder.
func (c *Environment) writeOSInfo(sb *strings.Builder) {
	fmt.Fprintf(sb, "- OS: %s (%s)\n", c.OS.OS, c.OS.Arch)

	if c.OS.Kernel != "" {
		fmt.Fprintf(sb, "- Kernel: %s\n", c.OS.Kernel)
	}

	if c.OS.Shell != "" {
		fmt.Fprintf(sb, "- Shell: %s\n", c.OS.Shell)
	}
}

// writeProjectInfo writes project type and languages to the builder.
func (c *Environment) writeProjectInfo(sb *strings.Builder) {
	if c.ProjectType != "unknown" {
		fmt.Fprintf(sb, "- Project Type: %s\n", c.ProjectType)
	}

	if len(c.Languages) > 0 {
		fmt.Fprintf(sb, "- Languages: %s\n", strings.Join(c.Languages, ", "))
	}
}

// writeGitInfo writes git information to the builder.
func (c *Environment) writeGitInfo(sb *strings.Builder) {
	if c.Git == nil {
		return
	}

	status := "clean"
	if c.Git.HasChanges {
		status = "dirty"
	}

	fmt.Fprintf(sb, "- Git Branch: %s (%s)\n", c.Git.Branch, status)

	if len(c.Git.UntrackedFiles) > 0 {
		fmt.Fprintf(sb, "- Untracked Files: %d\n", len(c.Git.UntrackedFiles))
	}

	if len(c.Git.Remotes) > 0 {
		fmt.Fprintf(sb, "- Git Remotes: %d\n", len(c.Git.Remotes))
	}
}

// writeFilesSummary writes the project structure summary to the builder.
func (c *Environment) writeFilesSummary(sb *strings.Builder) {
	if len(c.Files) == 0 {
		return
	}

	fmt.Fprintf(sb, "\nProject Structure: %d files\n", len(c.Files))

	maxShow := 20
	for i, file := range c.Files {
		if i >= maxShow {
			fmt.Fprintf(sb, "... and %d more files\n", len(c.Files)-maxShow)

			break
		}

		if file.Language != languageUnknown {
			fmt.Fprintf(sb, "- %s (%s, %d lines)\n", file.Path, file.Language, file.Lines)
		} else {
			fmt.Fprintf(sb, "- %s\n", file.Path)
		}
	}
}
