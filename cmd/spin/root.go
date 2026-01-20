package main

import (
	"github.com/dmytrogajewski/spin/internal/version"
	"github.com/spf13/cobra"
)

// Global flags
var (
	flagModel      string
	flagProvider   string
	flagSandbox    string
	flagWorkDir    string
	flagConfigFile string
	flagConfig     []string
	flagTaskMode   string
	flagAgentsMD   string
)

// newRootCmd creates the root command for spin CLI.
func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "spin",
		Short: "AI-powered coding assistant",
		Long: `Spin is an open-source AI coding assistant that works with multiple LLM providers.

It provides an interactive terminal UI, non-interactive execution mode,
and integrates with IDEs via JSON-RPC.

Compatible with: Ollama, LMStudio, OpenAI, Anthropic, and any OpenAI-compatible API.`,
		Version: version.ShortVersion(),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Default behavior: launch TUI when no subcommand is provided
			return runTUI(cmd, args)
		},
		SilenceUsage:  true,
		SilenceErrors: true, // Errors are handled in main()
	}

	// Set custom version template
	cmd.SetVersionTemplate(version.String() + "\n")

	// Global flags
	cmd.PersistentFlags().StringVar(&flagModel, "model", "", "Model to use (e.g., llama3.1, mixtral, gpt-4o)")
	cmd.PersistentFlags().StringVar(&flagProvider, "provider", "", "Provider (ollama, lmstudio, openai, anthropic)")
	cmd.PersistentFlags().StringVar(&flagSandbox, "sandbox", "", "Sandbox mode (read-only, workspace-write, full-access)")
	cmd.PersistentFlags().StringVar(&flagWorkDir, "cd", "", "Working directory")
	cmd.PersistentFlags().StringVar(&flagConfigFile, "config-file", "", "Path to configuration file")
	cmd.PersistentFlags().StringSliceVarP(&flagConfig, "config", "c", nil, "Config overrides (key=value)")
	cmd.PersistentFlags().StringVarP(&flagTaskMode, "mode", "m", "regular", "Task mode: regular (full-featured, 16K tokens), review (read-only, 12K tokens), compact (minimal, 4K tokens), planning (context-only, 4K tokens)")
	cmd.PersistentFlags().StringVar(&flagAgentsMD, "agents-md", "", "Path to AGENTS.md file (overrides auto-discovery)")

	// Add commands
	cmd.AddCommand(newTUICmd()) // TUI mode (Phase 7.4 complete!)
	cmd.AddCommand(newVersionCmd())
	cmd.AddCommand(newCompletionCmd())
	cmd.AddCommand(newExecCmd())
	cmd.AddCommand(newACPCmd()) // ACP server mode
	cmd.AddCommand(newConfigCmd())
	cmd.AddCommand(newAuthCmd()) // Auth management
	cmd.AddCommand(newMCPCmd())
	cmd.AddCommand(newDebugCmd())
	cmd.AddCommand(newApplyPatchCmd()) // Apply patch CLI (Feature 2.4)
	cmd.AddCommand(newModeCmd())       // Mode management (P3.3)
	cmd.AddCommand(newApprovalCmd())   // Approval policy management (CLI, ACP-compliant)

	return cmd
}
