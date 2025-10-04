// Package hardening provides process-level security measures for the Spin
// autonomous coding agent.
//
// The hardening package automatically applies security measures at process
// startup to reduce attack surface:
//
//   - Disable core dumps (prevent credential leakage)
//   - Disable ptrace (prevent debugger attachment)
//   - Remove dangerous environment variables (prevent library injection)
//
// # Usage
//
// Import the package to automatically apply hardening via init():
//
//	import _ "github.com/dmytrogajewski/spin/internal/security/hardening"
//
// Or manually call Apply():
//
//	if err := hardening.Apply(); err != nil {
//	    log.Warn("Hardening failed", "error", err)
//	}
//
// # Platform Support
//
//   - Linux: Full support (core dumps, ptrace, env sanitization)
//   - macOS: Full support (core dumps, ptrace, env sanitization)
//   - Windows: Partial support (env sanitization only)
//
// # Best Practices
//
// Hardening is best-effort and failures are logged but not fatal.
// The application continues even if hardening fails, as the security
// is defense-in-depth and other layers (policy, sandbox) provide
// additional protection.
package hardening

import (
	"fmt"
	"log/slog"
	"os"
)

// Apply applies all hardening measures.
// Should be called as early as possible in main() or via init().
// Returns error if any hardening measure fails, but continues applying
// all measures regardless.
func Apply() error {
	var errs []error

	// 1. Disable core dumps
	if err := disableCoreDumps(); err != nil {
		errs = append(errs, fmt.Errorf("disable core dumps: %w", err))
		slog.Warn("failed to disable core dumps", "error", err)
	} else {
		slog.Debug("disabled core dumps")
	}

	// 2. Disable ptrace
	if err := disablePtrace(); err != nil {
		errs = append(errs, fmt.Errorf("disable ptrace: %w", err))
		slog.Warn("failed to disable ptrace", "error", err)
	} else {
		slog.Debug("disabled ptrace")
	}

	// 3. Remove dangerous env vars
	if err := sanitizeEnvironment(); err != nil {
		errs = append(errs, fmt.Errorf("sanitize environment: %w", err))
		slog.Warn("failed to sanitize environment", "error", err)
	} else {
		slog.Debug("sanitized environment")
	}

	if len(errs) > 0 {
		return fmt.Errorf("hardening errors: %v", errs)
	}

	return nil
}

// sanitizeEnvironment removes dangerous environment variables that could
// be used for library injection or other attacks.
func sanitizeEnvironment() error {
	dangerous := []string{
		// Linux
		"LD_PRELOAD",
		"LD_LIBRARY_PATH",
		"LD_AUDIT",
		"LD_BIND_NOW",

		// macOS
		"DYLD_INSERT_LIBRARIES",
		"DYLD_LIBRARY_PATH",
		"DYLD_FRAMEWORK_PATH",
		"DYLD_FALLBACK_LIBRARY_PATH",
		"DYLD_FALLBACK_FRAMEWORK_PATH",
	}

	for _, key := range dangerous {
		if _, exists := os.LookupEnv(key); exists {
			slog.Warn("removing dangerous environment variable", "key", key)
			os.Unsetenv(key)
		}
	}

	return nil
}
