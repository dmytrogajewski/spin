package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/dmytrogajewski/spin/internal/config"
	"github.com/spf13/cobra"
)

// newMCPCmd creates the MCP management command.
func newMCPCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Manage MCP (Model Context Protocol) servers",
		Long: `Manage MCP (Model Context Protocol) servers.

MCP servers extend Spin with additional capabilities like filesystem
access, database queries, and API integrations.

Examples:
  # Add a filesystem server
  spin mcp add filesystem npx -y @modelcontextprotocol/server-filesystem /workspace

  # List all configured servers
  spin mcp list

  # Show details of a server
  spin mcp get filesystem

  # Remove a server
  spin mcp remove filesystem`,
	}

	cmd.AddCommand(newMCPAddCmd())
	cmd.AddCommand(newMCPListCmd())
	cmd.AddCommand(newMCPGetCmd())
	cmd.AddCommand(newMCPRemoveCmd())

	return cmd
}

// newMCPAddCmd creates the command for adding a new MCP server.
func newMCPAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <name> <command> [args...]",
		Short: "Add a new MCP server",
		Long: `Add a new MCP server configuration.

The server is added to your configuration file (usually ~/.spin/spin.yaml).

Examples:
  # Add a filesystem server
  spin mcp add filesystem npx -y @modelcontextprotocol/server-filesystem /workspace

  # Add a GitHub server
  spin mcp add github mcp-server-github --token-file ~/.github-token

  # Add a PostgreSQL server
  spin mcp add postgres mcp-server-postgres --connection-string "postgresql://localhost/mydb"`,
		Args: cobra.MinimumNArgs(2),
		RunE: runMCPAdd,
	}
	return cmd
}

// newMCPListCmd creates the command for listing configured MCP servers.
func newMCPListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all configured MCP servers",
		Long: `List all configured MCP servers.

Displays the name, command, and configuration status of each server.

Examples:
  # List all servers
  spin mcp list

  # List with JSON output
  spin mcp list --format json`,
		RunE: runMCPList,
	}
	cmd.Flags().String("format", "table", "Output format (table, json)")
	return cmd
}

// newMCPGetCmd creates the command for getting details of a specific MCP server.
func newMCPGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <name>",
		Short: "Show details of an MCP server",
		Long: `Show detailed configuration of a specific MCP server.

Displays the full configuration including command, args, environment variables,
and the source configuration file.

Examples:
  # Show server details
  spin mcp get filesystem

  # Show details with JSON output
  spin mcp get filesystem --format json`,
		Args: cobra.ExactArgs(1),
		RunE: runMCPGet,
	}
	cmd.Flags().String("format", "text", "Output format (text, json)")
	return cmd
}

// newMCPRemoveCmd creates the command for removing an MCP server.
func newMCPRemoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove an MCP server",
		Long: `Remove an MCP server configuration.

The server is removed from your configuration file.
By default, you are prompted for confirmation.

Examples:
  # Remove a server (with confirmation)
  spin mcp remove filesystem

  # Remove without confirmation
  spin mcp remove filesystem --yes`,
		Args: cobra.ExactArgs(1),
		RunE: runMCPRemove,
	}
	cmd.Flags().Bool("yes", false, "Skip confirmation prompt")
	return cmd
}

// runMCPAdd handles the execution of the MCP add command.
func runMCPAdd(cmd *cobra.Command, args []string) error {
	name := args[0]
	command := args[1]
	var serverArgs []string
	if len(args) > 2 {
		serverArgs = args[2:]
	}

	// Load config
	loader := config.NewLoaderV2()
	if _, err := loader.LoadFromFile(flagConfigFile); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Create MCP manager
	mgr := config.NewMCPManager(loader)

	// Create server
	server := config.MCPServer{
		Name:    name,
		Command: command,
		Args:    serverArgs,
	}

	// Add server
	if err := mgr.Add(server); err != nil {
		return err
	}

	// Determine config file location
	configFile := loader.ConfigFileUsed()
	if configFile == "" {
		homeDir, _ := os.UserHomeDir()
		configFile = homeDir + "/.spin/spin.yaml"
	}

	fmt.Printf("Added MCP server '%s' to %s\n", name, configFile)
	return nil
}

// runMCPList handles the execution of the MCP list command.
func runMCPList(cmd *cobra.Command, args []string) error {
	format, _ := cmd.Flags().GetString("format")

	// Load config
	loader := config.NewLoaderV2()
	if _, err := loader.LoadFromFile(flagConfigFile); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Create MCP manager
	mgr := config.NewMCPManager(loader)

	// List servers
	servers, err := mgr.List()
	if err != nil {
		return err
	}

	if len(servers) == 0 {
		fmt.Println("No MCP servers configured.")
		return nil
	}

	// Output based on format
	if format == "json" {
		return outputJSON(servers)
	}

	// Table format
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tCOMMAND\tSTATUS")
	for _, server := range servers {
		cmdStr := formatCommand(server)
		fmt.Fprintf(w, "%s\t%s\tconfigured\n", server.Name, cmdStr)
	}
	w.Flush()

	return nil
}

// runMCPGet handles the execution of the MCP get command.
func runMCPGet(cmd *cobra.Command, args []string) error {
	name := args[0]
	format, _ := cmd.Flags().GetString("format")

	// Load config
	loader := config.NewLoaderV2()
	if _, err := loader.LoadFromFile(flagConfigFile); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Create MCP manager
	mgr := config.NewMCPManager(loader)

	// Get server
	server, err := mgr.Get(name)
	if err != nil {
		return err
	}

	// Output based on format
	if format == "json" {
		return outputJSON(server)
	}

	// Text format
	fmt.Printf("Name: %s\n", server.Name)
	fmt.Printf("Command: %s\n", server.Command)
	if len(server.Args) > 0 {
		fmt.Println("Args:")
		for _, arg := range server.Args {
			fmt.Printf("  - %s\n", arg)
		}
	}
	if len(server.Env) > 0 {
		fmt.Println("Environment:")
		for key, value := range server.Env {
			fmt.Printf("  %s=%s\n", key, value)
		}
	} else {
		fmt.Println("Environment: (none)")
	}

	// Show source config file
	configFile := loader.ConfigFileUsed()
	if configFile != "" {
		fmt.Printf("Source: %s\n", configFile)
	}

	return nil
}

// runMCPRemove handles the execution of the MCP remove command.
func runMCPRemove(cmd *cobra.Command, args []string) error {
	name := args[0]
	yes, _ := cmd.Flags().GetBool("yes")

	// Load config
	loader := config.NewLoaderV2()
	if _, err := loader.LoadFromFile(flagConfigFile); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Create MCP manager
	mgr := config.NewMCPManager(loader)

	// Check if server exists
	_, err := mgr.Get(name)
	if err != nil {
		return err
	}

	// Confirm unless --yes flag
	if !yes {
		fmt.Printf("Remove MCP server '%s'? (y/N): ", name)
		var response string
		fmt.Scanln(&response)
		response = strings.ToLower(strings.TrimSpace(response))
		if response != "y" && response != "yes" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	// Remove server
	if err := mgr.Remove(name); err != nil {
		return err
	}

	// Determine config file location
	configFile := loader.ConfigFileUsed()
	if configFile == "" {
		homeDir, _ := os.UserHomeDir()
		configFile = homeDir + "/.spin/spin.yaml"
	}

	fmt.Printf("Removed MCP server '%s' from %s\n", name, configFile)
	return nil
}

// formatCommand formats a command string for display (truncates if too long)
func formatCommand(server config.MCPServer) string {
	parts := []string{server.Command}
	parts = append(parts, server.Args...)
	cmdStr := strings.Join(parts, " ")
	if len(cmdStr) > 50 {
		cmdStr = cmdStr[:47] + "..."
	}
	return cmdStr
}

// outputJSON outputs data as JSON
func outputJSON[T any](data T) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}
