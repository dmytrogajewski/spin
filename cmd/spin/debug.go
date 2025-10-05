package main

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/dmytrogajewski/spin/internal/debug"
	"github.com/spf13/cobra"
)

// newDebugCmd creates the debug command with subcommands.
func newDebugCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "debug",
		Short: "Debug and development utilities",
		Long:  `Tools for testing, debugging, and profiling Spin.`,
	}

	cmd.AddCommand(
		newDebugEventsCmd(),
		newDebugSandboxCmd(),
		newDebugLandlockCmd(),
	)

	return cmd
}

// newDebugEventsCmd creates the events debugging command.
func newDebugEventsCmd() *cobra.Command {
	var format string
	var filterStr string

	cmd := &cobra.Command{
		Use:   "events <prompt>",
		Short: "Execute a task and log all core events",
		Long: `Execute a task and print all core events to stderr for debugging.

Events are logged in real-time as they occur. Use --format json for
machine-readable output, or --filter to show specific event types only.`,
		Example: `  # Show all events
  spin debug events "list files in current directory"

  # Show only tool events
  spin debug events --filter tool "run tests"

  # JSON output for parsing
  spin debug events --format json "fix linting" | jq`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDebugEvents(cmd.Context(), strings.Join(args, " "), format, filterStr)
		},
	}

	cmd.Flags().StringVar(&format, "format", "text", "Output format (text|json)")
	cmd.Flags().StringVar(&filterStr, "filter", "", "Event type filter (comma-separated, e.g. 'tool,stream')")

	return cmd
}

// runDebugEvents executes the events debugging command.
func runDebugEvents(ctx context.Context, prompt, format, filterStr string) error {
	// Parse filter
	var filter []string
	if filterStr != "" {
		rawFilters := strings.Split(filterStr, ",")
		for _, f := range rawFilters {
			f = strings.TrimSpace(f)
			// Map short names to full event names
			switch f {
			case "tool":
				filter = append(filter, "tool_call_start", "tool_call_progress", "tool_call_complete")
			case "stream":
				filter = append(filter, "content_delta", "content_complete")
			case "turn":
				filter = append(filter, "turn_start", "turn_complete", "turn_failed")
			case "approval":
				filter = append(filter, "command_approval", "command_approved", "command_denied")
			default:
				filter = append(filter, f)
			}
		}
	}

	// Create event logger
	logger := debug.NewEventLogger(format, filter)

	// Run with timeout
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	return logger.Run(ctx, prompt)
}

// newDebugSandboxCmd creates the macOS sandbox testing command.
func newDebugSandboxCmd() *cobra.Command {
	var mode string
	var workspace string

	cmd := &cobra.Command{
		Use:   "sandbox <command> [args...]",
		Short: "Test macOS sandbox behavior (macOS only)",
		Long: `Execute a command in the macOS sandbox to verify behavior.

This command is only available on macOS and uses sandbox-exec to
test different sandbox modes before deploying in production.`,
		Example: `  # Test read-only mode
  spin debug sandbox --mode read-only ls -la

  # Test workspace-write mode
  spin debug sandbox --mode workspace-write touch test.txt`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Check platform
			if runtime.GOOS != "darwin" {
				return fmt.Errorf("sandbox command is only available on macOS (current: %s)", runtime.GOOS)
			}
			return runDebugSandbox(cmd.Context(), args[0], args[1:], mode, workspace)
		},
	}

	cmd.Flags().StringVar(&mode, "mode", "workspace-write", "Sandbox mode (read-only|workspace-write|full-access)")
	cmd.Flags().StringVar(&workspace, "workspace", ".", "Workspace directory")

	return cmd
}

// runDebugSandbox executes the sandbox testing command.
func runDebugSandbox(ctx context.Context, command string, args []string, mode, workspace string) error {
	fmt.Fprintf(os.Stderr, "⚠️  Sandbox mode: %s\n", mode)
	fmt.Fprintf(os.Stderr, "⚠️  Workspace: %s\n", workspace)
	fmt.Fprintf(os.Stderr, "⚠️  Command: %s %s\n\n", command, strings.Join(args, " "))

	// TODO: Implement actual sandbox execution via internal/security/sandbox
	// For now, just return unimplemented
	return fmt.Errorf("sandbox testing not yet implemented")
}

// newDebugLandlockCmd creates the Linux Landlock testing command.
func newDebugLandlockCmd() *cobra.Command {
	var mode string
	var workspace string

	cmd := &cobra.Command{
		Use:   "landlock <command> [args...]",
		Short: "Test Linux Landlock restrictions (Linux only)",
		Long: `Execute a command with Landlock LSM restrictions to verify behavior.

This command is only available on Linux and uses Landlock to test
filesystem restrictions before deploying in production.`,
		Example: `  # Test Landlock restrictions
  spin debug landlock --mode read-only ls /etc

  # Test workspace access
  spin debug landlock --mode workspace-write touch file.txt`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Check platform
			if runtime.GOOS != "linux" {
				return fmt.Errorf("landlock command is only available on Linux (current: %s)", runtime.GOOS)
			}
			return runDebugLandlock(cmd.Context(), args[0], args[1:], mode, workspace)
		},
	}

	cmd.Flags().StringVar(&mode, "mode", "workspace-write", "Sandbox mode (read-only|workspace-write|full-access)")
	cmd.Flags().StringVar(&workspace, "workspace", ".", "Workspace directory")

	return cmd
}

// runDebugLandlock executes the Landlock testing command.
func runDebugLandlock(ctx context.Context, command string, args []string, mode, workspace string) error {
	fmt.Fprintf(os.Stderr, "⚠️  Landlock mode: %s\n", mode)
	fmt.Fprintf(os.Stderr, "⚠️  Workspace: %s\n", workspace)
	fmt.Fprintf(os.Stderr, "⚠️  Command: %s %s\n\n", command, strings.Join(args, " "))

	// TODO: Implement actual Landlock execution via internal/security/sandbox
	// For now, just return unimplemented
	return fmt.Errorf("landlock testing not yet implemented")
}
