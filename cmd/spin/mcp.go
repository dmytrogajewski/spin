package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

// newMCPCmd creates the MCP management command.
func newMCPCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Manage MCP server configurations",
		Long:  `Manage Model Context Protocol server configurations.`,
	}

	cmd.AddCommand(newMCPAddCmd())
	cmd.AddCommand(newMCPListCmd())
	cmd.AddCommand(newMCPGetCmd())
	cmd.AddCommand(newMCPRemoveCmd())

	return cmd
}

func newMCPAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add <name> <command> [args...]",
		Short: "Add MCP server configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("mcp add not yet implemented")
		},
	}
}

func newMCPListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List MCP server configurations",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("mcp list not yet implemented")
		},
	}
}

func newMCPGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <name>",
		Short: "Get MCP server configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("mcp get not yet implemented")
		},
	}
}

func newMCPRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove MCP server configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("mcp remove not yet implemented")
		},
	}
}
