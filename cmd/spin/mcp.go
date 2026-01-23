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

Supported transports:
  - stdio: Local process communication (default)
  - sse: Server-Sent Events for remote servers
  - streamable-http: HTTP streaming for remote servers
  - smithery: Smithery's connection-based API

Examples:
  # Add a local stdio server
  spin mcp add filesystem npx -y @modelcontextprotocol/server-filesystem /workspace

  # Add a remote SSE server (e.g., Smithery)
  spin mcp add smithery-memory --transport sse --url https://server.smithery.ai/sse

  # Add a remote server with authentication
  spin mcp add remote-api --transport sse --url https://api.example.com/mcp \
    --header "Authorization=Bearer token"

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
		Use:   "add <name> [command] [args...]",
		Short: "Add a new MCP server",
		Long: `Add a new MCP server configuration.

The server is added to your configuration file (usually ~/.spin/spin.yaml).

For stdio transport (default), provide the command and arguments.
For remote transports (sse, streamable-http), use --url flag.

Examples:
  # Add a local stdio server
  spin mcp add filesystem npx -y @modelcontextprotocol/server-filesystem /workspace

  # Add a remote SSE server
  spin mcp add smithery --transport sse --url https://server.smithery.ai/sse

  # Add a remote server with headers
  spin mcp add remote-api --transport sse --url https://api.example.com/mcp \
    --header "Authorization=Bearer token" --header "X-Custom=value"

  # Add a streamable HTTP server
  spin mcp add http-server --transport streamable-http --url https://mcp.example.com/v1

  # Add a server with OAuth
  spin mcp add protected --transport sse --url https://protected.example.com/mcp \
    --oauth-client-id "my-client" --oauth-client-secret "secret"

  # Add a Smithery server
  spin mcp add papersearch --transport smithery \
    --url https://server.smithery.ai/@adamamer20/paper-search-mcp-openai \
    --smithery-api-key "your-smithery-api-key" --smithery-namespace "your-namespace"`,
		Args: cobra.MinimumNArgs(1),
		RunE: runMCPAdd,
	}

	// Transport flags
	cmd.Flags().String("transport", "stdio", "Transport type: stdio, sse, streamable-http, smithery")
	cmd.Flags().String("url", "", "URL for remote MCP server (required for sse/streamable-http/smithery)")
	cmd.Flags().StringArray("header", nil, "HTTP headers for remote servers (format: Key=Value)")

	// OAuth flags
	cmd.Flags().String("oauth-client-id", "", "OAuth client ID")
	cmd.Flags().String("oauth-client-secret", "", "OAuth client secret")
	cmd.Flags().String("oauth-redirect-url", "", "OAuth redirect URL")
	cmd.Flags().StringArray("oauth-scope", nil, "OAuth scopes")

	// Smithery flags
	cmd.Flags().String("smithery-api-key", "", "Smithery API key (required for smithery transport)")
	cmd.Flags().String("smithery-namespace", "", "Smithery namespace (required for smithery transport)")

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

	// Get transport flags
	transport, _ := cmd.Flags().GetString("transport")
	url, _ := cmd.Flags().GetString("url")
	headerFlags, _ := cmd.Flags().GetStringArray("header")

	// Get OAuth flags
	oauthClientID, _ := cmd.Flags().GetString("oauth-client-id")
	oauthClientSecret, _ := cmd.Flags().GetString("oauth-client-secret")
	oauthRedirectURL, _ := cmd.Flags().GetString("oauth-redirect-url")
	oauthScopes, _ := cmd.Flags().GetStringArray("oauth-scope")

	// Get Smithery flags
	smitheryAPIKey, _ := cmd.Flags().GetString("smithery-api-key")
	smitheryNamespace, _ := cmd.Flags().GetString("smithery-namespace")

	// Load config
	loader := config.NewLoaderV2()
	if _, err := loader.LoadFromFile(flagConfigFile); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Create MCP manager
	mgr := config.NewMCPConfigStore(loader)

	// Create server based on transport type
	server := config.MCPServer{
		Name:      name,
		Transport: config.MCPTransportType(transport),
	}

	// Set transport-specific fields
	transportType := config.MCPTransportType(transport)
	if transportType == config.MCPTransportSmithery {
		// Smithery transport: use URL, smithery_api_key, and smithery_namespace
		if url == "" {
			return fmt.Errorf("--url is required for smithery transport")
		}
		if smitheryAPIKey == "" {
			return fmt.Errorf("--smithery-api-key is required for smithery transport")
		}
		if smitheryNamespace == "" {
			return fmt.Errorf("--smithery-namespace is required for smithery transport")
		}
		server.URL = url
		server.SmitheryAPIKey = smitheryAPIKey
		server.SmitheryNamespace = smitheryNamespace
	} else if transportType.IsRemote() {
		// Remote transport: use URL and headers
		if url == "" {
			return fmt.Errorf("--url is required for %s transport", transport)
		}
		server.URL = url
		server.Headers = parseHeaders(headerFlags)
	} else {
		// Stdio transport: use command and args
		if len(args) < 2 {
			return fmt.Errorf("command is required for stdio transport")
		}
		server.Command = args[1]
		if len(args) > 2 {
			server.Args = args[2:]
		}
	}

	// Set OAuth if provided
	if oauthClientID != "" {
		server.OAuth = &config.MCPOAuthConfigV2{
			ClientID:     oauthClientID,
			ClientSecret: oauthClientSecret,
			RedirectURL:  oauthRedirectURL,
			Scopes:       oauthScopes,
		}
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

// parseHeaders parses header flags in format "Key=Value" into a map.
func parseHeaders(headers []string) map[string]string {
	if len(headers) == 0 {
		return nil
	}
	result := make(map[string]string)
	for _, h := range headers {
		parts := strings.SplitN(h, "=", 2)
		if len(parts) == 2 {
			result[parts[0]] = parts[1]
		}
	}
	return result
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
	mgr := config.NewMCPConfigStore(loader)

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
	fmt.Fprintln(w, "NAME\tTRANSPORT\tURL/COMMAND\tSTATUS")
	for _, server := range servers {
		transport := string(server.Transport)
		if transport == "" {
			transport = "stdio"
		}
		endpoint := formatEndpoint(server)
		fmt.Fprintf(w, "%s\t%s\t%s\tconfigured\n", server.Name, transport, endpoint)
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
	mgr := config.NewMCPConfigStore(loader)

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

	// Transport
	transport := string(server.Transport)
	if transport == "" {
		transport = "stdio"
	}
	fmt.Printf("Transport: %s\n", transport)

	// Transport-specific fields
	if server.Transport.IsRemote() {
		fmt.Printf("URL: %s\n", server.URL)
		if len(server.Headers) > 0 {
			fmt.Println("Headers:")
			for key, value := range server.Headers {
				// Mask sensitive headers
				displayValue := value
				if strings.EqualFold(key, "authorization") || strings.Contains(strings.ToLower(key), "secret") {
					displayValue = "***"
				}
				fmt.Printf("  %s: %s\n", key, displayValue)
			}
		}
	} else {
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
		}
	}

	// OAuth
	if server.OAuth != nil {
		fmt.Println("OAuth:")
		fmt.Printf("  Client ID: %s\n", server.OAuth.ClientID)
		if server.OAuth.ClientSecret != "" {
			fmt.Printf("  Client Secret: ***\n")
		}
		if server.OAuth.RedirectURL != "" {
			fmt.Printf("  Redirect URL: %s\n", server.OAuth.RedirectURL)
		}
		if len(server.OAuth.Scopes) > 0 {
			fmt.Printf("  Scopes: %s\n", strings.Join(server.OAuth.Scopes, ", "))
		}
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
	mgr := config.NewMCPConfigStore(loader)

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

// formatEndpoint formats the server endpoint for display (URL or command).
func formatEndpoint(server config.MCPServer) string {
	if server.Transport.IsRemote() {
		// Remote: show URL
		url := server.URL
		if len(url) > 50 {
			url = url[:47] + "..."
		}
		return url
	}
	// Stdio: show command
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
