package shell

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// ShellIntegration provides shell-aware functionality for the agent.
type ShellIntegration struct {
	enabled   bool
	workDir   string
	logger    *slog.Logger
	mu        sync.RWMutex
	shell     string
	shellPath string
	envVars   map[string]string
	timeout   time.Duration
}

// NewShellIntegration creates a new Shell integration.
func NewShellIntegration(enabled bool, workDir string, logger *slog.Logger, timeout time.Duration) *ShellIntegration {
	return &ShellIntegration{
		enabled: enabled,
		workDir: workDir,
		logger:  logger,
		envVars: make(map[string]string),
		timeout: timeout,
	}
}

// Initialize sets up Shell integration.
func (s *ShellIntegration) Initialize(ctx context.Context) error {
	if !s.enabled {
		s.logger.Debug("Shell integration disabled")
		return nil
	}

	// Detect shell
	shell, shellPath := s.detectShell()
	if shell == "" {
		s.logger.Debug("No shell detected")
		return nil
	}

	s.mu.Lock()
	s.shell = shell
	s.shellPath = shellPath
	s.mu.Unlock()

	// Gather shell environment variables
	s.gatherEnvironmentVars()

	s.logger.Info("Shell integration initialized",
		"shell", shell,
		"path", shellPath,
		"workDir", s.workDir)

	return nil
}

// IsEnabled returns true if Shell integration is enabled.
func (s *ShellIntegration) IsEnabled() bool {
	return s.enabled
}

// GetShell returns the detected shell name.
func (s *ShellIntegration) GetShell() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.shell
}

// GetShellPath returns the detected shell path.
func (s *ShellIntegration) GetShellPath() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.shellPath
}

// GetEnvironmentVars returns shell-specific environment variables.
func (s *ShellIntegration) GetEnvironmentVars() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy to avoid race conditions
	result := make(map[string]string)
	for k, v := range s.envVars {
		result[k] = v
	}
	return result
}

// GetContextInfo returns Shell context information for the agent.
func (s *ShellIntegration) GetContextInfo() map[string]interface{} {
	if !s.IsEnabled() {
		return map[string]interface{}{
			"shell_enabled": false,
		}
	}

	info := map[string]interface{}{
		"shell_enabled": true,
		"shell":         s.GetShell(),
		"shell_path":    s.GetShellPath(),
	}

	// Add shell-specific environment variables
	envVars := s.GetEnvironmentVars()
	if len(envVars) > 0 {
		info["shell_env"] = envVars
	}

	return info
}

// ExecuteShellCommand executes a command using the detected shell.
func (s *ShellIntegration) ExecuteShellCommand(ctx context.Context, command string) (string, error) {
	if !s.IsEnabled() {
		return "", fmt.Errorf("shell integration disabled")
	}

	shellPath := s.GetShellPath()
	if shellPath == "" {
		return "", fmt.Errorf("no shell available")
	}

	// Create a context with timeout for the shell command
	// Use the shorter of the two timeouts (integration timeout vs context timeout)
	var cmdCtx context.Context
	var cancel context.CancelFunc

	if deadline, ok := ctx.Deadline(); ok {
		// Context already has a deadline, use the shorter timeout
		integrationDeadline := time.Now().Add(s.timeout)
		if deadline.Before(integrationDeadline) {
			cmdCtx, cancel = context.WithTimeout(ctx, time.Until(deadline))
		} else {
			cmdCtx, cancel = context.WithTimeout(ctx, s.timeout)
		}
	} else {
		// No existing deadline, use integration timeout
		cmdCtx, cancel = context.WithTimeout(ctx, s.timeout)
	}
	defer cancel()

	// Determine shell arguments based on shell type
	var args []string
	switch s.GetShell() {
	case "bash":
		args = []string{"-c", command}
	case "zsh":
		args = []string{"-c", command}
	case "fish":
		args = []string{"-c", command}
	case "sh":
		args = []string{"-c", command}
	case "cmd":
		args = []string{"/c", command}
	case "powershell":
		args = []string{"-Command", command}
	default:
		args = []string{"-c", command}
	}

	cmd := exec.CommandContext(cmdCtx, shellPath, args...)
	cmd.Dir = s.workDir

	// Set environment variables
	cmd.Env = s.buildEnvironment()

	// Use CombinedOutput to capture both stdout and stderr
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if it's a timeout error
		if cmdCtx.Err() == context.DeadlineExceeded {
			// Determine which timeout was actually used
			var timeoutUsed time.Duration
			if deadline, ok := ctx.Deadline(); ok {
				integrationDeadline := time.Now().Add(s.timeout)
				if deadline.Before(integrationDeadline) {
					timeoutUsed = time.Until(deadline)
				} else {
					timeoutUsed = s.timeout
				}
			} else {
				timeoutUsed = s.timeout
			}
			return "", fmt.Errorf("shell command timed out after %v: %s", timeoutUsed, command)
		}
		// Check if it's an ExitError to get exit code and stderr
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode := exitErr.ExitCode()
			// Output already contains stderr since we used CombinedOutput
			outputStr := strings.TrimSpace(string(output))
			if outputStr != "" {
				return "", fmt.Errorf("shell command failed (exit %d): %s", exitCode, outputStr)
			}
			return "", fmt.Errorf("shell command failed (exit %d)", exitCode)
		}
		return "", fmt.Errorf("shell command failed: %w", err)
	}

	return string(output), nil
}

// IsShellCommand checks if a command should be executed through the shell.
func (s *ShellIntegration) IsShellCommand(command string) bool {
	if !s.IsEnabled() {
		return false
	}

	// Commands that typically need shell interpretation
	shellCommands := []string{
		"cd", "pwd", "export", "unset", "alias", "unalias",
		"source", ".", "eval", "exec", "history", "jobs",
		"fg", "bg", "kill", "wait", "trap", "exit",
	}

	// Check if command starts with shell operators
	shellOperators := []string{
		"&&", "||", "|", ">", ">>", "<", "<<", "&", ";",
		"$(", "$", "`", "~", "*", "?", "[", "]", "{", "}",
	}

	cmd := strings.TrimSpace(command)

	// Check for shell operators
	for _, op := range shellOperators {
		if strings.Contains(cmd, op) {
			return true
		}
	}

	// Check for shell commands
	parts := strings.Fields(cmd)
	if len(parts) > 0 {
		cmdName := parts[0]
		for _, shellCmd := range shellCommands {
			if cmdName == shellCmd {
				return true
			}
		}
	}

	// Check for environment variable expansion
	if strings.Contains(cmd, "$") || strings.Contains(cmd, "${") {
		return true
	}

	// Check for wildcards
	if strings.Contains(cmd, "*") || strings.Contains(cmd, "?") {
		return true
	}

	return false
}

// GetWorkingDirectory returns the current working directory.
func (s *ShellIntegration) GetWorkingDirectory() string {
	return s.workDir
}

// SetWorkingDirectory sets the working directory.
func (s *ShellIntegration) SetWorkingDirectory(workDir string) {
	s.mu.Lock()
	s.workDir = workDir
	s.mu.Unlock()
}

// Close cleans up Shell integration resources.
func (s *ShellIntegration) Close() error {
	// No resources to clean up
	return nil
}

// detectShell detects the current shell.
func (s *ShellIntegration) detectShell() (string, string) {
	// Check SHELL environment variable first
	if shell := os.Getenv("SHELL"); shell != "" {
		if _, err := exec.LookPath(shell); err == nil {
			return s.extractShellName(shell), shell
		}
	}

	// Check common shell paths
	commonShells := map[string][]string{
		"bash": {"/bin/bash", "/usr/bin/bash", "/usr/local/bin/bash"},
		"zsh":  {"/bin/zsh", "/usr/bin/zsh", "/usr/local/bin/zsh"},
		"fish": {"/bin/fish", "/usr/bin/fish", "/usr/local/bin/fish"},
		"sh":   {"/bin/sh", "/usr/bin/sh"},
	}

	for shellName, paths := range commonShells {
		for _, path := range paths {
			if _, err := exec.LookPath(path); err == nil {
				return shellName, path
			}
		}
	}

	// Windows-specific shells
	if runtime.GOOS == "windows" {
		if cmdPath, err := exec.LookPath("cmd.exe"); err == nil {
			return "cmd", cmdPath
		}
		if psPath, err := exec.LookPath("powershell.exe"); err == nil {
			return "powershell", psPath
		}
	}

	return "", ""
}

// extractShellName extracts the shell name from the path.
func (s *ShellIntegration) extractShellName(shellPath string) string {
	baseName := filepath.Base(shellPath)

	// Remove .exe extension on Windows
	if runtime.GOOS == "windows" && strings.HasSuffix(baseName, ".exe") {
		baseName = strings.TrimSuffix(baseName, ".exe")
	}

	return baseName
}

// gatherEnvironmentVars gathers shell-specific environment variables.
func (s *ShellIntegration) gatherEnvironmentVars() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Common shell environment variables
	shellVars := []string{
		"SHELL", "TERM", "COLUMNS", "LINES", "PS1", "PS2", "PS3", "PS4",
		"HISTSIZE", "HISTFILESIZE", "HISTFILE", "HISTCONTROL", "HISTIGNORE",
		"PATH", "HOME", "USER", "LOGNAME", "HOSTNAME", "PWD", "OLDPWD",
		"EDITOR", "VISUAL", "PAGER", "MANPAGER", "LANG", "LC_ALL", "LC_CTYPE",
	}

	for _, varName := range shellVars {
		if value := os.Getenv(varName); value != "" {
			s.envVars[varName] = value
		}
	}

	// Shell-specific variables
	switch s.shell {
	case "bash":
		bashVars := []string{"BASH_VERSION", "BASHOPTS", "BASHPID", "BASH_SOURCE"}
		for _, varName := range bashVars {
			if value := os.Getenv(varName); value != "" {
				s.envVars[varName] = value
			}
		}
	case "zsh":
		zshVars := []string{"ZSH_VERSION", "ZDOTDIR", "ZSH_CACHE_DIR"}
		for _, varName := range zshVars {
			if value := os.Getenv(varName); value != "" {
				s.envVars[varName] = value
			}
		}
	case "fish":
		fishVars := []string{"FISH_VERSION", "XDG_CONFIG_HOME"}
		for _, varName := range fishVars {
			if value := os.Getenv(varName); value != "" {
				s.envVars[varName] = value
			}
		}
	}
}

// buildEnvironment builds the environment for shell commands.
func (s *ShellIntegration) buildEnvironment() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Start with current environment
	env := os.Environ()

	// Add shell-specific variables
	for key, value := range s.envVars {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}

	return env
}
