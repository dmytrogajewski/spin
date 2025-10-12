package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

// modeInfo contains detailed information about a task mode.
type modeInfo struct {
	name        string
	description string
	maxTokens   int
	tools       []string
	bestFor     []string
}

// allModes contains detailed information for all available modes.
var allModes = map[string]modeInfo{
	"regular": {
		name:        "regular",
		description: "Full-featured interactive coding mode with access to all tools",
		maxTokens:   16384,
		tools: []string{
			"read_file",
			"write_file",
			"list_directory",
			"execute_command",
			"get_context",
			"file_search",
			"apply_patch",
			"git_context",
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
		maxTokens:   12288,
		tools: []string{
			"read_file",
			"list_directory",
			"get_context",
			"file_search",
			"git_context",
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
		maxTokens:   4096,
		tools: []string{
			"read_file",
			"get_context",
			"file_search",
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
		maxTokens:   4096,
		tools: []string{
			"get_context",
			"file_search",
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

	// Add subcommands
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
func runModeList(cmd *cobra.Command, args []string) error {
	fmt.Println("Available task modes:")
	fmt.Println()

	// Print modes in a consistent order
	modeOrder := []string{"regular", "review", "compact", "planning"}
	for _, name := range modeOrder {
		info := allModes[name]
		fmt.Printf("  %s\n", name)
		fmt.Printf("    %s\n", info.description)
		fmt.Printf("    Token budget: %d | Tools: %d\n", info.maxTokens, len(info.tools))
		fmt.Println()
	}

	fmt.Println("Use 'spin mode describe <mode-name>' for detailed information.")
	fmt.Println("Use 'spin --mode <mode-name>' to start with a specific mode.")

	return nil
}

// runModeDescribe handles the 'spin mode describe <mode-name>' command.
func runModeDescribe(cmd *cobra.Command, args []string) error {
	modeName := args[0]

	// Validate mode name
	info, exists := allModes[modeName]
	if !exists {
		return fmt.Errorf("unknown mode: %s (valid modes: regular, review, compact, planning)", modeName)
	}

	// Print detailed mode information
	fmt.Printf("Mode: %s\n", info.name)
	fmt.Println()
	fmt.Printf("Description:\n  %s\n", info.description)
	fmt.Println()
	fmt.Printf("Token Budget: %d tokens\n", info.maxTokens)
	fmt.Println()

	// Print tools
	fmt.Printf("Available Tools (%d):\n", len(info.tools))
	for _, tool := range info.tools {
		fmt.Printf("  - %s\n", tool)
	}
	fmt.Println()

	// Print best use cases
	fmt.Println("Best For:")
	for _, useCase := range info.bestFor {
		fmt.Printf("  • %s\n", useCase)
	}
	fmt.Println()

	// Print usage examples
	fmt.Println("Usage:")
	fmt.Printf("  spin --mode %s              # Start TUI in %s mode\n", modeName, modeName)
	fmt.Printf("  spin --mode %s exec <task>  # Execute task in %s mode\n", modeName, modeName)
	fmt.Println()

	return nil
}
