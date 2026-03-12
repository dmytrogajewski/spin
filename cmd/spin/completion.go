package main

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

var ErrUnsupportedShell = errors.New("unsupported shell")

// newCompletionCmd creates the completion command.
func newCompletionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion script",
		Long: `Generate shell completion script for Spin.

To load completions:

Bash:
  $ source <(spin completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ spin completion bash > /etc/bash_completion.d/spin
  # macOS:
  $ spin completion bash > $(brew --prefix)/etc/bash_completion.d/spin

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following once:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ spin completion zsh > "${fpath[1]}/_spin"

  # You will need to start a new shell for this setup to take effect.

Fish:
  $ spin completion fish | source

  # To load completions for each session, execute once:
  $ spin completion fish > ~/.config/fish/completions/spin.fish

PowerShell:
  PS> spin completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> spin completion powershell > spin.ps1
  # and source this file from your PowerShell profile.
`,
		Args:                  cobra.ExactArgs(1),
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return cmd.Root().GenBashCompletion(cmd.OutOrStdout())
			case "zsh":
				return cmd.Root().GenZshCompletion(cmd.OutOrStdout())
			case "fish":
				return cmd.Root().GenFishCompletion(cmd.OutOrStdout(), true)
			case "powershell":
				return cmd.Root().GenPowerShellCompletionWithDesc(cmd.OutOrStdout())
			default:
return fmt.Errorf("unsupported shell: %s: %w", args[0], ErrUnsupportedShell)
			}
		},
	}

	return cmd
}
