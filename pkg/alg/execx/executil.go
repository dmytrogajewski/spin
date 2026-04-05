// Package execx provides shared utilities for command execution,
// including output merging and context-aware timeout calculation.
package execx

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"time"
)

// MergeOutputs combines stdout and stderr into a single string.
// If both are non-empty, they are joined with a newline.
// If only one is non-empty, it is returned as-is.
// If both are empty, an empty string is returned.
func MergeOutputs(stdout, stderr string) string {
	if stderr == "" {
		return stdout
	}

	if stdout != "" {
		return stdout + "\n" + stderr
	}

	return stderr
}

// EffectiveTimeout returns the shorter of defaultTimeout or the
// time remaining until ctx's deadline. If ctx has no deadline or
// the remaining time is non-positive, defaultTimeout is returned.
func EffectiveTimeout(ctx context.Context, defaultTimeout time.Duration) time.Duration {
	deadline, hasDeadline := ctx.Deadline()
	if !hasDeadline {
		return defaultTimeout
	}

	remaining := time.Until(deadline)
	if remaining > 0 && remaining < defaultTimeout {
		return remaining
	}

	return defaultTimeout
}

// FindEditor returns the preferred text editor by checking $EDITOR, $VISUAL,
// and then looking for common editors (vi, vim, nano, emacs) on the PATH.
// Returns an empty string if no editor is found.
func FindEditor() string {
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor
	}

	if visual := os.Getenv("VISUAL"); visual != "" {
		return visual
	}

	// Try common editors.
	for _, editor := range []string{"vi", "vim", "nano", "emacs"} {
		if _, err := exec.LookPath(editor); err == nil {
			return editor
		}
	}

	return ""
}

// IsShellCommand checks if a command string requires shell interpretation.
// It returns true when the command contains shell metacharacters (|, >, <, $, &&, ||)
// or starts with a shell built-in (cd, export, source).
func IsShellCommand(cmdStr string) bool {
	shellChars := []string{"|", ">", "<", "$", "&&", "||"}
	for _, c := range shellChars {
		if strings.Contains(cmdStr, c) {
			return true
		}
	}

	shellPrefixes := []string{"cd ", "export ", "source "}
	for _, p := range shellPrefixes {
		if strings.HasPrefix(cmdStr, p) {
			return true
		}
	}

	return false
}
