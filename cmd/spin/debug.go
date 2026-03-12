package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/dmytrogajewski/spin/internal/dbg"
)

var (
	// ErrSandboxCommandIsOnlyAvailableOn is a sentinel error.
	ErrSandboxCommandIsOnlyAvailableOn = errors.New("sandbox command is only available on macOS (current")
	// ErrLandlockCommandIsOnlyAvailableOn is a sentinel error.
	ErrLandlockCommandIsOnlyAvailableOn = errors.New("landlock command is only available on Linux (current")
	// ErrSandboxTestingNotImplemented is a sentinel error.
	ErrSandboxTestingNotImplemented = errors.New("sandbox testing not implemented")
	// ErrLandlockTestingNotImplemented is a sentinel error.
	ErrLandlockTestingNotImplemented = errors.New("landlock testing not implemented")
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
	var (
		format    string
		filterStr string
	)

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
  spin debug events --format json "build project"`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDebugEvents(cmd.Context(), strings.Join(args, " "), format, filterStr)
		},
	}

	cmd.Flags().StringVar(&format, "format", formatText, "Output format (text|json)")
	cmd.Flags().StringVar(&filterStr, "filter", "", "Event type filter (comma-separated, e.g. 'tool,stream')")

	return cmd
}

// runDebugEvents executes the events debugging command.
const debugTimeout = 5 * time.Minute

func runDebugEvents(ctx context.Context, prompt, format, filterStr string) error {
	// Parse filter.
	var filter []string

	if filterStr != "" {
		rawFilters := strings.SplitSeq(filterStr, ",")
		for f := range rawFilters {
			f = strings.TrimSpace(f)
			// Map short names to full event names.
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

	// Create event logger.
	logger := dbg.NewEventLogger(format, filter)

	// Run with timeout.
	ctx, cancel := context.WithTimeout(ctx, debugTimeout)
	defer cancel()

	return logger.Run(ctx, prompt)
}

// debugIsolationConfig holds configuration for platform-specific isolation commands.
type debugIsolationConfig struct {
	use         string
	short       string
	long        string
	example     string
	requiredOS  string
	platformErr error
	runFunc     func(ctx context.Context, command string, args []string, mode, workspace string) error
	extraFlags  func(cmd *cobra.Command)
}

// newDebugIsolationCmd creates a platform-specific isolation debugging command.
func newDebugIsolationCmd(cfg debugIsolationConfig) *cobra.Command {
	var (
		mode      string
		workspace string
		timeout   string
	)

	cmd := &cobra.Command{
		Use:     cfg.use,
		Short:   cfg.short,
		Long:    cfg.long,
		Example: cfg.example,
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if runtime.GOOS != cfg.requiredOS {
				return fmt.Errorf("%s (current: %s): %w", cfg.platformErr.Error(), runtime.GOOS, cfg.platformErr)
			}

			return cfg.runFunc(cmd.Context(), args[0], args[1:], mode, workspace)
		},
	}

	cmd.Flags().StringVar(&mode, "mode", "workspace-write", "Sandbox mode (read-only|workspace-write|full-access)")
	cmd.Flags().StringVar(&workspace, "workspace", ".", "Workspace directory")
	cmd.Flags().StringVar(&timeout, "timeout", "30s", "Command timeout")

	if cfg.extraFlags != nil {
		cfg.extraFlags(cmd)
	}

	return cmd
}

// newDebugSandboxCmd creates the sandbox debugging command.
func newDebugSandboxCmd() *cobra.Command {
	return newDebugIsolationCmd(debugIsolationConfig{
		use:   "sandbox <command>",
		short: "Execute a command in a sandboxed environment",
		long: `Execute a command with sandbox restrictions to verify behavior.

This command is only available on macOS and uses sandbox-exec to test
filesystem restrictions before deploying in production.`,
		example: `  # Test sandbox restrictions
  spin debug sandbox "ls -la"

  # Test network access
  spin debug sandbox --network "curl https://example.com"

  # Test write access
  spin debug sandbox --read-only=false "touch test.txt"`,
		requiredOS:  "darwin",
		platformErr: ErrSandboxCommandIsOnlyAvailableOn,
		runFunc:     runDebugSandbox,
		extraFlags: func(cmd *cobra.Command) {
			cmd.Flags().Bool("read-only", true, "Enable read-only mode")
			cmd.Flags().Bool("network", false, "Enable network access")
		},
	})
}

// newDebugLandlockCmd creates the Landlock debugging command.
func newDebugLandlockCmd() *cobra.Command {
	return newDebugIsolationCmd(debugIsolationConfig{
		use:   "landlock <command>",
		short: "Execute a command with Landlock restrictions",
		long: `Execute a command with Landlock LSM restrictions to verify behavior.

This command is only available on Linux and uses Landlock to test
filesystem restrictions before deploying in production.`,
		example: `  # Test Landlock restrictions
  spin debug landlock "ls -la"

  # Test write access
  spin debug landlock --allow-write "touch test.txt"

  # Test read-only mode
  spin debug landlock --allow-read=false "cat file.txt"`,
		requiredOS:  "linux",
		platformErr: ErrLandlockCommandIsOnlyAvailableOn,
		runFunc:     runDebugLandlock,
		extraFlags: func(cmd *cobra.Command) {
			cmd.Flags().Bool("allow-read", true, "Allow read access")
			cmd.Flags().Bool("allow-write", false, "Allow write access")
		},
	})
}

// runDebugSandbox executes the sandbox testing command.

func runDebugSandbox(_ context.Context, command string, args []string, mode, workspace string) error {
	fmt.Fprintf(os.Stderr, "⚠️  Sandbox mode: %s\n", mode)
	fmt.Fprintf(os.Stderr, "⚠️  Workspace: %s\n", workspace)
	fmt.Fprintf(os.Stderr, "⚠️  Command: %s %s\n\n", command, strings.Join(args, " "))

	// Placeholder: Sandbox execution requires proper sandbox implementation
	// via internal/security/sandbox with appropriate isolation (namespaces, chroot, etc.)
	// This is a complex feature that requires OS-specific implementations.
	return ErrSandboxTestingNotImplemented
}

// runDebugLandlock executes the Landlock testing command.

func runDebugLandlock(_ context.Context, command string, args []string, mode, workspace string) error {
	fmt.Fprintf(os.Stderr, "⚠️  Landlock mode: %s\n", mode)
	fmt.Fprintf(os.Stderr, "⚠️  Workspace: %s\n", workspace)
	fmt.Fprintf(os.Stderr, "⚠️  Command: %s %s\n\n", command, strings.Join(args, " "))

	// Placeholder: Landlock execution requires Linux-specific implementation
	// via internal/security/sandbox using Landlock ABI
	// This is a kernel feature that requires appropriate system calls.
	return ErrLandlockTestingNotImplemented
}
