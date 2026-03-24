package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// ErrUnknownMode is a sentinel error.
var ErrUnknownMode = errors.New("unknown mode")

// modeInfo contains detailed information about a task mode.
type modeInfo struct {
	name        string
	description string
	maxTokens   int
	tools       []string
	bestFor     []string
}

// allModes contains detailed information for all available modes.
const (
	regularModeMaxTokens  = 16384
	reviewModeMaxTokens   = 12288
	compactModeMaxTokens  = 4096
	planningModeMaxTokens = 4096
)

var allModes = map[string]modeInfo{
	"regular": {
		name:        "regular",
		description: "Full-featured interactive coding mode with access to all tools",
		maxTokens:   regularModeMaxTokens,
		tools: []string{
			"read_file",
			"write_file",
			"edit_file",
			"apply_patch",
			"list_directory",
			"file_search",
			"find_symbol",
			"find_references",
			"rename_symbol",
			"get_context",
			"git_context",
			"git_operation",
			"execute_command",
			"start_process",
			"list_processes",
			"get_process_output",
			"kill_process",
			"memory",
			"scratchpad",
		},
		bestFor: []string{
			"Implementing new features",
			"Refactoring code",
			"Complex multi-step tasks",
			"Full development workflows",
		},
	},
	"review": {
		name:        "review",
		description: "Read-only code analysis and review mode",
		maxTokens:   reviewModeMaxTokens,
		tools: []string{
			"read_file",
			"list_directory",
			"file_search",
			"find_symbol",
			"find_references",
			"get_context",
			"git_context",
			"git_operation",
		},
		bestFor: []string{
			"Code reviews and PR analysis",
			"Security audits",
			"Understanding codebase",
			"Documentation review",
		},
	},
	"compact": {
		name:        "compact",
		description: "Quick queries with minimal context and tool access",
		maxTokens:   compactModeMaxTokens,
		tools: []string{
			"read_file",
			"list_directory",
			"file_search",
			"find_symbol",
			"get_context",
		},
		bestFor: []string{
			"Quick questions",
			"Fast iteration",
			"Debugging specific issues",
			"Low-latency interactions",
		},
	},
	"planning": {
		name:        "planning",
		description: "Task decomposition and planning mode with context-only tools",
		maxTokens:   planningModeMaxTokens,
		tools: []string{
			"read_file",
			"list_directory",
			"file_search",
			"find_symbol",
			"find_references",
			"get_context",
			"git_context",
		},
		bestFor: []string{
			"Breaking down large tasks",
			"Architecture planning",
			"Project roadmapping",
			"High-level design",
		},
	},
}

// newModeCmd creates the 'spin mode' command and its subcommands.
func newModeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mode [command]",
		Short: "Manage and inspect task modes",
		Long: `Manage task modes for the Spin agent.

Task modes control which tools are available and how much context
the agent can use. Different modes are optimized for different workflows.

Available commands:
  list      List all available task modes
  describe  Show detailed information about a specific mode`,
		SilenceUsage: true,
	}

	// Add subcommands.
	cmd.AddCommand(newModeListCmd())
	cmd.AddCommand(newModeDescribeCmd())

	return cmd
}

// newModeListCmd creates the 'spin mode list' subcommand.
func newModeListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all available task modes",
		Long: `List all available task modes with brief descriptions.

Task modes:
  - regular:  Full-featured interactive coding (default)
  - review:   Read-only code analysis
  - compact:  Quick queries with minimal context
  - planning: Task decomposition and planning`,
		RunE:         runModeList,
		SilenceUsage: true,
	}

	return cmd
}

// newModeDescribeCmd creates the 'spin mode describe' subcommand.
func newModeDescribeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "describe <mode-name>",
		Short: "Show detailed information about a task mode",
		Long: `Show detailed information about a specific task mode.

This includes:
  - Full description
  - Token budget
  - Available tools
  - Best use cases

Example:
  spin mode describe review`,
		Args:         cobra.ExactArgs(1),
		RunE:         runModeDescribe,
		SilenceUsage: true,
	}

	return cmd
}

// runModeList handles the 'spin mode list' command.
func runModeList(_ *cobra.Command, _ []string) error {
	fmt.Fprintln(os.Stdout, "Available task modes:")
	fmt.Fprintln(os.Stdout)

	// Print modes in a consistent order.
	modeOrder := []string{"regular", "review", "compact", "planning"}
	for _, name := range modeOrder {
		info := allModes[name]
		fmt.Fprintf(os.Stdout, "  %s\n", name)
		fmt.Fprintf(os.Stdout, "    %s\n", info.description)
		fmt.Fprintf(os.Stdout, "    Token budget: %d | Tools: %d\n", info.maxTokens, len(info.tools))
		fmt.Fprintln(os.Stdout)
	}

	fmt.Fprintln(os.Stdout, "Use 'spin mode describe <mode-name>' for detailed information.")
	fmt.Fprintln(os.Stdout, "Use 'spin --mode <mode-name>' to start with a specific mode.")

	return nil
}

// runModeDescribe handles the 'spin mode describe <mode-name>' command.
func runModeDescribe(_ *cobra.Command, args []string) error {
	modeName := args[0]

	// Validate mode name.
	info, exists := allModes[modeName]
	if !exists {
		return fmt.Errorf("unknown mode: %s (valid modes: regular, review, compact, planning): %w", modeName, ErrUnknownMode)
	}

	// Print detailed mode information.
	fmt.Fprintf(os.Stdout, "Mode: %s\n", info.name)
	fmt.Fprintln(os.Stdout)
	fmt.Fprintf(os.Stdout, "Description:\n  %s\n", info.description)
	fmt.Fprintln(os.Stdout)
	fmt.Fprintf(os.Stdout, "Token Budget: %d tokens\n", info.maxTokens)
	fmt.Fprintln(os.Stdout)

	// Print tools.
	fmt.Fprintf(os.Stdout, "Available Tools (%d):\n", len(info.tools))

	for _, tool := range info.tools {
		fmt.Fprintf(os.Stdout, "  - %s\n", tool)
	}

	fmt.Fprintln(os.Stdout)

	// Print best use cases.
	fmt.Fprintln(os.Stdout, "Best For:")

	for _, useCase := range info.bestFor {
		fmt.Fprintf(os.Stdout, "  • %s\n", useCase)
	}

	fmt.Fprintln(os.Stdout)

	// Print usage examples.
	fmt.Fprintln(os.Stdout, "Usage:")
	fmt.Fprintf(os.Stdout, "  spin --mode %s              # Start TUI in %s mode\n", modeName, modeName)
	fmt.Fprintf(os.Stdout, "  spin --mode %s exec <task>  # Execute task in %s mode\n", modeName, modeName)
	fmt.Fprintln(os.Stdout)

	return nil
}
