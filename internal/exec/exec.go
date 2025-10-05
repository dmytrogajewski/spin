// Package exec provides non-interactive execution mode for Spin.
package exec

import (
	"context"
	"io"
	"os"
)

// Parse parses command-line arguments for exec mode.
// This is the public wrapper for parseArgs.
func Parse(args []string, stdin io.Reader) (*ExecArgs, error) {
	return parseArgs(args, stdin)
}

// Run executes the task in exec mode.
// This is the public wrapper for runTask.
func Run(ctx context.Context, args *ExecArgs) error {
	return runTask(ctx, args)
}

// SetupSignals sets up signal handling for exec mode.
// This is the public wrapper for setupSignalHandler.
func SetupSignals(ctx context.Context, cancel context.CancelFunc) chan os.Signal {
	return setupSignalHandler(ctx, cancel)
}

// FormatError formats an error for display.
// This is the public wrapper for formatError.
func FormatError(err error) string {
	return formatError(err)
}

// ExitCode extracts exit code from an error.
// This is the public wrapper for exitCodeFromError.
func GetExitCode(err error) ExitCode {
	return exitCodeFromError(err)
}
