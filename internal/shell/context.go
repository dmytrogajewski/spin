// Package shell provides shell context management.
package shell

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/dmytrogajewski/spin/internal/process"
)

var (
	// ErrShellIntegrationDisabled is a sentinel error.
	ErrShellIntegrationDisabled = errors.New("shell integration disabled")
	// ErrNoShellAvailable is a sentinel error.
	ErrNoShellAvailable = errors.New("no shell available")
	// ErrShellCommandTimedOutAfter is a sentinel error.
	ErrShellCommandTimedOutAfter = errors.New("shell command timed out after")
	// ErrExecutionFailed is a sentinel error.
	ErrExecutionFailed = errors.New("execution failed")
)

// Context provides shell-aware functionality for the agent.
type Context struct {
	enabled   bool
	workDir   string
	logger    *slog.Logger
	mu        sync.RWMutex
	shell     string
	shellPath string
	envVars   map[string]string
	timeout   time.Duration
}

// NewContext creates a new Shell context.
func NewContext(enabled bool, workDir string, logger *slog.Logger, timeout time.Duration) *Context {
	return &Context{
		enabled: enabled,
		workDir: workDir,
		logger:  logger,
		envVars: make(map[string]string),
		timeout: timeout,
	}
}

// Initialize sets up Shell context.
func (s *Context) Initialize(ctx context.Context) error {
	if !s.enabled {
		s.logger.DebugContext(ctx, "Shell integration disabled")

		return nil
	}

	// Detect shell.
	shell, shellPath := s.detectShell()
	if shell == "" {
		s.logger.DebugContext(ctx, "No shell detected")

		return nil
	}

	s.mu.Lock()
	s.shell = shell
	s.shellPath = shellPath
	s.mu.Unlock()

	// Gather shell environment variables.
	s.gatherEnvironmentVars()

	s.logger.InfoContext(ctx, "Shell integration initialized",
		"shell", shell,
		"path", shellPath,
		"workDir", s.workDir)

	return nil
}

// IsEnabled returns true if Shell context is enabled.
func (s *Context) IsEnabled() bool {
	return s.enabled
}

// GetShell returns the detected shell name.
func (s *Context) GetShell() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.shell
}

// GetShellPath returns the detected shell path.
func (s *Context) GetShellPath() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.shellPath
}

// GetEnvironmentVars returns shell-specific environment variables.
func (s *Context) GetEnvironmentVars() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy to avoid race conditions.
	result := make(map[string]string)
	maps.Copy(result, s.envVars)

	return result
}

// ContextInfo contains shell context information for the agent.
type ContextInfo struct {
	ShellEnabled bool              `json:"shell_enabled"`
	Shell        string            `json:"shell,omitempty"`
	ShellPath    string            `json:"shell_path,omitempty"`
	ShellEnv     map[string]string `json:"shell_env,omitempty"`
}

// IsShellEnabled returns whether shell is enabled.
func (c ContextInfo) IsShellEnabled() bool { return c.ShellEnabled }

// GetShell returns the shell name.
func (c ContextInfo) GetShell() string { return c.Shell }

// GetShellPath returns the shell path.
func (c ContextInfo) GetShellPath() string { return c.ShellPath }

// GetShellEnv returns the shell environment variables.
func (c ContextInfo) GetShellEnv() map[string]string { return c.ShellEnv }

// GetContextInfo returns shell context information for the agent.
func (s *Context) GetContextInfo() ContextInfo {
	if !s.IsEnabled() {
		return ContextInfo{
			ShellEnabled: false,
		}
	}

	return ContextInfo{
		ShellEnabled: true,
		Shell:        s.GetShell(),
		ShellPath:    s.GetShellPath(),
		ShellEnv:     s.GetEnvironmentVars(),
	}
}

// ExecuteShellCommand executes a command using the detected shell.
func (s *Context) ExecuteShellCommand(ctx context.Context, command string) (string, error) {
	if !s.IsEnabled() {
		return "", ErrShellIntegrationDisabled
	}

	shellPath := s.GetShellPath()
	if shellPath == "" {
		return "", ErrNoShellAvailable
	}

	cmdCtx, cancel := context.WithTimeout(ctx, s.effectiveTimeout(ctx))
	defer cancel()

	args := s.shellArgs(command)

	cmd := exec.CommandContext(cmdCtx, shellPath, args...)
	cmd.Dir = s.workDir
	cmd.Env = s.buildEnvironment()

	process.SetGroup(cmd)

	cmd.Cancel = func() error {
		return process.KillGroup(cmd)
	}

	var stdout, stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return s.handleRunError(cmdCtx, err, command, &stdout, &stderr)
	}

	output := stdout.String()
	if stderr.String() != "" {
		output += stderr.String()
	}

	return output, nil
}

// effectiveTimeout returns the timeout to use, respecting parent context deadline.
func (s *Context) effectiveTimeout(ctx context.Context) time.Duration {
	timeout := s.timeout

	deadline, hasDeadline := ctx.Deadline()
	if !hasDeadline {
		return timeout
	}

	remaining := time.Until(deadline)
	if remaining < timeout && remaining > 0 {
		return remaining
	}

	return timeout
}

// shellArgs returns the appropriate command-line arguments for the detected shell.
func (s *Context) shellArgs(command string) []string {
	switch s.GetShell() {
	case "cmd":
		return []string{"/c", command}
	case "powershell":
		return []string{"-Command", command}
	default:
		return []string{"-c", command}
	}
}

// handleRunError processes errors from cmd.Run and returns an appropriate error.
func (s *Context) handleRunError(
	cmdCtx context.Context, err error, command string,
	stdout, stderr *bytes.Buffer,
) (string, error) {
	if errors.Is(cmdCtx.Err(), context.DeadlineExceeded) {
		return "", fmt.Errorf("shell command timed out after %v: %s: %w", s.timeout, command, ErrShellCommandTimedOutAfter)
	}

	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return "", fmt.Errorf("shell command failed: %w", err)
	}

	return "", s.formatExitError(exitErr, stdout, stderr)
}

// formatExitError builds the error message for a non-zero exit code.
func (s *Context) formatExitError(exitErr *exec.ExitError, stdout, stderr *bytes.Buffer) error {
	exitCode := exitErr.ExitCode()
	stdoutStr := strings.TrimSpace(stdout.String())
	stderrStr := strings.TrimSpace(stderr.String())

	output := mergeOutputs(stdoutStr, stderrStr)

	if output != "" {
		return fmt.Errorf("execution failed: exit status %d\n%s: %w", exitCode, output, ErrExecutionFailed)
	}

	return fmt.Errorf("execution failed: exit status %d: %w", exitCode, ErrExecutionFailed)
}

// mergeOutputs combines stdout and stderr, deduplicating when identical.
func mergeOutputs(stdoutStr, stderrStr string) string {
	if stdoutStr == stderrStr {
		return stderrStr
	}

	if stderrStr != "" && stdoutStr != "" {
		return fmt.Sprintf("Error: %s\nOutput: %s", stderrStr, stdoutStr)
	}

	if stderrStr != "" {
		return stderrStr
	}

	return stdoutStr
}

// IsShellCommand checks if a command should be executed through the shell.
func (s *Context) IsShellCommand(command string) bool {
	if !s.IsEnabled() {
		return false
	}

	// Commands that typically need shell interpretation.
	shellCommands := []string{
		"cd", "pwd", "export", "unset", "alias", "unalias",
		"source", ".", "eval", "exec", "history", "jobs",
		"fg", "bg", "kill", "wait", "trap", "exit",
	}

	// Check if command starts with shell operators.
	shellOperators := []string{
		"&&", "||", "|", ">", ">>", "<", "<<", "&", ";",
		"$(", "$", "`", "~", "*", "?", "[", "]", "{", "}",
	}

	cmd := strings.TrimSpace(command)

	// Check for shell operators.
	for _, op := range shellOperators {
		if strings.Contains(cmd, op) {
			return true
		}
	}

	// Check for shell commands.
	parts := strings.Fields(cmd)
	if len(parts) > 0 {
		cmdName := parts[0]
		if slices.Contains(shellCommands, cmdName) {
			return true
		}
	}

	// Check for environment variable expansion.
	if strings.Contains(cmd, "$") || strings.Contains(cmd, "${") {
		return true
	}

	// Check for wildcards.
	if strings.Contains(cmd, "*") || strings.Contains(cmd, "?") {
		return true
	}

	return false
}

// GetWorkingDirectory returns the current working directory.
func (s *Context) GetWorkingDirectory() string {
	return s.workDir
}

// SetWorkingDirectory sets the working directory.
func (s *Context) SetWorkingDirectory(workDir string) {
	s.mu.Lock()
	s.workDir = workDir
	s.mu.Unlock()
}

// Close cleans up shell context resources.
func (s *Context) Close() error {
	// No resources to clean up.
	return nil
}

// detectShell detects the current shell.
func (s *Context) detectShell() (shellName, shellPath string) {
	// Check SHELL environment variable first.
	if shell := os.Getenv("SHELL"); shell != "" {
		_, err := exec.LookPath(shell)
		if err == nil {
			return s.extractShellName(shell), shell
		}
	}

	// Check common shell paths.
	commonShells := map[string][]string{
		"bash": {"/bin/bash", "/usr/bin/bash", "/usr/local/bin/bash"},
		"zsh":  {"/bin/zsh", "/usr/bin/zsh", "/usr/local/bin/zsh"},
		"fish": {"/bin/fish", "/usr/bin/fish", "/usr/local/bin/fish"},
		"sh":   {"/bin/sh", "/usr/bin/sh"},
	}

	for shellName, paths := range commonShells {
		for _, path := range paths {
			_, err := exec.LookPath(path)
			if err == nil {
				return shellName, path
			}
		}
	}

	// Windows-specific shells.
	if runtime.GOOS == "windows" {
		cmdPath, err := exec.LookPath("cmd.exe")
		if err == nil {
			return "cmd", cmdPath
		}

		psPath, err := exec.LookPath("powershell.exe")
		if err == nil {
			return "powershell", psPath
		}
	}

	return "", ""
}

// extractShellName extracts the shell name from the path.
func (s *Context) extractShellName(shellPath string) string {
	baseName := filepath.Base(shellPath)

	// Remove .exe extension on Windows.
	if runtime.GOOS == "windows" && strings.HasSuffix(baseName, ".exe") {
		baseName = strings.TrimSuffix(baseName, ".exe")
	}

	return baseName
}

// shellSpecificVars maps shell names to their specific environment variables.
var shellSpecificVars = map[string][]string{
	"bash": {"BASH_VERSION", "BASHOPTS", "BASHPID", "BASH_SOURCE"},
	"zsh":  {"ZSH_VERSION", "ZDOTDIR", "ZSH_CACHE_DIR"},
	"fish": {"FISH_VERSION", "XDG_CONFIG_HOME"},
}

// gatherEnvironmentVars gathers shell-specific environment variables.
func (s *Context) gatherEnvironmentVars() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Common shell environment variables.
	commonVars := []string{
		"SHELL", "TERM", "COLUMNS", "LINES", "PS1", "PS2", "PS3", "PS4",
		"HISTSIZE", "HISTFILESIZE", "HISTFILE", "HISTCONTROL", "HISTIGNORE",
		"PATH", "HOME", "USER", "LOGNAME", "HOSTNAME", "PWD", "OLDPWD",
		"EDITOR", "VISUAL", "PAGER", "MANPAGER", "LANG", "LC_ALL", "LC_CTYPE",
	}

	s.collectEnvVars(commonVars)
	s.collectEnvVars(shellSpecificVars[s.shell])
}

// collectEnvVars reads the given environment variable names and stores non-empty values.
// Must be called with s.mu held.
func (s *Context) collectEnvVars(varNames []string) {
	for _, varName := range varNames {
		if value := os.Getenv(varName); value != "" {
			s.envVars[varName] = value
		}
	}
}

// buildEnvironment builds the environment for shell commands.
func (s *Context) buildEnvironment() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Start with current environment.
	env := os.Environ()

	// Add shell-specific variables.
	for key, value := range s.envVars {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}

	return env
}
