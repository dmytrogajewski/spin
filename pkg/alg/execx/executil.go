// Package execx provides shared utilities for command execution,
// including output merging and context-aware timeout calculation.
package execx

import (
	"context"
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
