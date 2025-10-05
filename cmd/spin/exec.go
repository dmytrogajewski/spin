package main

import (
	"context"
	"fmt"
	"os"

	execpkg "github.com/dmytrogajewski/spin/internal/exec"
	"github.com/spf13/cobra"
)

// newExecCmd creates the exec command for non-interactive execution.
func newExecCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exec [prompt]",
		Short: "Non-interactive execution mode",
		Long: `Execute Spin in non-interactive mode for CI/CD and automation.

Examples:
  spin exec "run all tests and fix failures"
  echo "refactor authentication" | spin exec
  spin exec --timeout 5m "deploy to staging"
  spin exec --format json "analyze code" | jq`,
		RunE: runExec,
	}

	// Exec-specific flags
	cmd.Flags().Bool("auto-approve", false, "Automatically approve all operations (DANGEROUS)")
	cmd.Flags().String("timeout", "", "Maximum execution time (e.g., 5m, 1h)")
	cmd.Flags().String("format", "text", "Output format (text, json)")
	cmd.Flags().Bool("no-stream", false, "Disable streaming output")
	cmd.Flags().Bool("exit-on-error", true, "Exit immediately on first error")

	return cmd
}

// runExec executes the exec mode.
func runExec(cmd *cobra.Command, args []string) error {
	// Parse arguments using internal/exec package
	execArgs, err := execpkg.Parse(args, os.Stdin)
	if err != nil {
		return err
	}

	// Create context
	ctx := context.Background()
	var cancel context.CancelFunc

	if execArgs.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, execArgs.Timeout)
	} else {
		ctx, cancel = context.WithCancel(ctx)
	}
	defer cancel()

	// Setup signal handling
	_ = execpkg.SetupSignals(ctx, cancel)

	// Execute task
	if err := execpkg.Run(ctx, execArgs); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", execpkg.FormatError(err))
		os.Exit(int(execpkg.GetExitCode(err)))
	}

	return nil
}
