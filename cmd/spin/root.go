package main

import (
	"fmt"

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
			// Default behavior: would launch TUI (not implemented yet)
			return fmt.Errorf("TUI mode not yet implemented. Try: spin --help")
		},
		SilenceUsage: true,
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

	// Add commands
	cmd.AddCommand(newVersionCmd())
	cmd.AddCommand(newCompletionCmd())
	cmd.AddCommand(newExecCmd())
	cmd.AddCommand(newServeCmd())
	cmd.AddCommand(newConfigCmd())
	cmd.AddCommand(newMCPCmd())
	cmd.AddCommand(newDebugCmd())

	return cmd
}
