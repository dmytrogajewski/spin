package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

// newDebugCmd creates the debug command.
func newDebugCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "debug",
		Short: "Debug and testing commands",
		Long:  `Commands for debugging and testing Spin functionality.`,
	}

	cmd.AddCommand(newDebugSandboxCmd())
	cmd.AddCommand(newDebugLandlockCmd())

	return cmd
}

func newDebugSandboxCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sandbox <command>",
		Short: "Test macOS sandbox (sandbox-exec)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("debug sandbox not yet implemented")
		},
	}
}

func newDebugLandlockCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "landlock <command>",
		Short: "Test Linux landlock LSM",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("debug landlock not yet implemented")
		},
	}
}
