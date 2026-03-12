package main

import (
	"github.com/spf13/cobra"

	"github.com/dmytrogajewski/spin/internal/appinfo"
)

// flagModel returns the --model flag value from a cobra command.
func flagModel(cmd *cobra.Command) string {
	v, _ := cmd.Root().PersistentFlags().GetString("model")

	return v
}

// flagProvider returns the --provider flag value from a cobra command.
func flagProvider(cmd *cobra.Command) string {
	v, _ := cmd.Root().PersistentFlags().GetString("provider")

	return v
}

// flagWorkDir returns the --cd flag value from a cobra command.
func flagWorkDir(cmd *cobra.Command) string {
	v, _ := cmd.Root().PersistentFlags().GetString("cd")

	return v
}

// flagConfigFile returns the --config-file flag value from a cobra command.
func flagConfigFile(cmd *cobra.Command) string {
	v, _ := cmd.Root().PersistentFlags().GetString("config-file")

	return v
}

// flagAgentsMD returns the --agents-md flag value from a cobra command.
func flagAgentsMD(cmd *cobra.Command) string {
	v, _ := cmd.Root().PersistentFlags().GetString("agents-md")

	return v
}

// newRootCmd creates the root command for spin CLI.
func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "spin",
		Short: "AI-powered coding assistant",
		Long: `Spin is an open-source AI coding assistant that works with multiple LLM providers.

It provides an interactive terminal UI, non-interactive execution mode,
and integrates with IDEs via JSON-RPC.

Compatible with: Ollama, LMStudio, OpenAI, Anthropic, and any OpenAI-compatible API.`,
		Version:       appinfo.ShortVersion(),
		RunE:          runTUI,
		SilenceUsage:  true,
		SilenceErrors: true, // Errors are handled in main().
	}

	// Set custom version template.
	cmd.SetVersionTemplate(appinfo.String() + "\n")

	// Global flags (command-local, no package-level variables).
	cmd.PersistentFlags().String("model", "", "Model to use (e.g., llama3.1, mixtral, gpt-4o)")
	cmd.PersistentFlags().String("provider", "", "Provider (ollama, lmstudio, openai, anthropic)")
	cmd.PersistentFlags().String("sandbox", "", "Sandbox mode (read-only, workspace-write, full-access)")
	cmd.PersistentFlags().String("cd", "", "Working directory")
	cmd.PersistentFlags().String("config-file", "", "Path to configuration file")
	cmd.PersistentFlags().StringSliceP("config", "c", nil, "Config overrides (key=value)")

	modeHelp := "Task mode: regular (full-featured, 16K tokens), " +
		"review (read-only, 12K tokens), compact (minimal, 4K tokens), " +
		"planning (context-only, 4K tokens)"
	cmd.PersistentFlags().StringP("mode", "m", "regular", modeHelp)
	cmd.PersistentFlags().String("agents-md", "", "Path to AGENTS.md file (overrides auto-discovery)")

	// Add commands.
	cmd.AddCommand(newTUICmd()) // TUI mode (Phase 7.4 complete!)
	cmd.AddCommand(newVersionCmd())
	cmd.AddCommand(newCompletionCmd())
	cmd.AddCommand(newExecCmd())
	cmd.AddCommand(newACPCmd()) // ACP server mode.
	cmd.AddCommand(newConfigCmd())
	cmd.AddCommand(newAuthCmd()) // Auth management.
	cmd.AddCommand(newMCPCmd())
	cmd.AddCommand(newDebugCmd())
	cmd.AddCommand(newApplyPatchCmd()) // Apply patch CLI (Feature 2.4).
	cmd.AddCommand(newModeCmd())       // Mode management (P3.3).
	cmd.AddCommand(newApprovalCmd())   // Approval policy management (CLI, ACP-compliant).

	return cmd
}
