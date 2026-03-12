package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/dmytrogajewski/spin/internal/config"
)

var (
	ErrInvalidTransport = errors.New("invalid transport")
	ErrAPIKeyIsRequired = errors.New("API key is required")
	ErrRegistryNotFound = errors.New("registry '' not found")
	ErrAPIErrorStatus = errors.New("API error (status )")
	ErrInvalidServerPathFormat = errors.New("invalid server path format")
)

// ============================================================================
// Smithery API Types
// ============================================================================.

// smitheryToolsResponse represents the response from the Smithery tools API.
type smitheryToolsResponse struct {
	Tools      []smitheryToolWithServer `json:"tools"`
	Pagination smitheryPagination       `json:"pagination"`
}

// smitheryToolWithServer represents a tool with its server information.
type smitheryToolWithServer struct {
	ID     string             `json:"id"`
	Tool   smitheryTool       `json:"tool"`
	Server smitheryToolServer `json:"server"`
}

// smitheryTool represents an MCP tool from Smithery.
type smitheryTool struct {
	Name        string `json:"name"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
}

// smitheryToolServer represents a server that provides a tool.
type smitheryToolServer struct {
	ID            string `json:"id"`
	QualifiedName string `json:"qualifiedName"`
	DisplayName   string `json:"displayName"`
	IconURL       string `json:"iconUrl,omitempty"`
	Verified      bool   `json:"verified"`
	IsDeployed    bool   `json:"isDeployed"`
}

// smitheryPagination represents pagination info from the API.
type smitheryPagination struct {
	CurrentPage int `json:"currentPage"`
	PageSize    int `json:"pageSize"`
	TotalPages  int `json:"totalPages"`
	TotalCount  int `json:"totalCount"`
}

// ============================================================================
// Main MCP Command
// ============================================================================.

// newMCPCmd creates the MCP management command.
func newMCPCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Manage MCP (Model Context Protocol) registries and tools",
		Long: `Manage MCP (Model Context Protocol) registries and tools.

MCP registries extend Spin with additional capabilities like filesystem
access, database queries, and API integrations.

Registry types:
  - local: Local process communication via stdio
  - remote: Remote servers via SSE or HTTP streaming
  - smithery: Smithery's hosted MCP servers

Examples:
  # Add a local registry
  spin mcp registry local add filesystem npx -y @modelcontextprotocol/server-filesystem /workspace

  # Add a Smithery registry
  spin mcp registry smithery add paper-search @adamamer20/paper-search-mcp

  # List all registries
  spin mcp registry list

  # Search for tools across registries
  spin mcp search github

  # List all tools
  spin mcp list`,
	}

	// Registry management.
	cmd.AddCommand(newMCPRegistryCmd())

	// Tool operations.
	cmd.AddCommand(newMCPSearchCmd())
	cmd.AddCommand(newMCPListToolsCmd())

	return cmd
}

// ============================================================================
// Registry Command Group
// ============================================================================.

// newMCPRegistryCmd creates the registry management command group.
func newMCPRegistryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "registry",
		Short: "Manage MCP registries",
		Long: `Manage MCP registries.

Registries are sources of MCP tools. Each registry can provide multiple tools
that extend Spin's capabilities.

Registry types:
  - local: Local process (stdio) - runs a command locally
  - remote: Remote server (SSE/HTTP) - connects to a remote endpoint
  - smithery: Smithery platform - uses Smithery's hosted servers

Examples:
  # Add a local registry
  spin mcp registry local add filesystem npx -y @modelcontextprotocol/server-filesystem /workspace

  # Add a remote registry
  spin mcp registry remote add api-server --url https://api.example.com/mcp

  # Add a Smithery registry
  spin mcp registry smithery add paper-search @adamamer20/paper-search-mcp

  # List all registries
  spin mcp registry list

  # Remove a registry
  spin mcp registry remove filesystem`,
	}

	// Type-specific add commands.
	cmd.AddCommand(newMCPRegistryLocalCmd())
	cmd.AddCommand(newMCPRegistryRemoteCmd())
	cmd.AddCommand(newMCPRegistrySmitheryCmd())

	// General registry operations.
	cmd.AddCommand(newMCPRegistryListCmd())
	cmd.AddCommand(newMCPRegistryGetCmd())
	cmd.AddCommand(newMCPRegistryRemoveCmd())

	return cmd
}

// ============================================================================
// Local Registry Commands
// ============================================================================.

// newMCPRegistryLocalCmd creates the local registry command group.
func newMCPRegistryLocalCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "local",
		Short: "Manage local (stdio) registries",
	}
	cmd.AddCommand(newMCPRegistryLocalAddCmd())

	return cmd
}

// newMCPRegistryLocalAddCmd creates the command for adding a local registry.
func newMCPRegistryLocalAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <name> <command> [args...]",
		Short: "Add a local stdio registry",
		Long: `Add a local MCP registry that communicates via stdio.

The command is executed as a subprocess, and Spin communicates
with it using the MCP protocol over stdin/stdout.

Examples:
  # Add a filesystem server
  spin mcp registry local add filesystem npx -y @modelcontextprotocol/server-filesystem /workspace

  # Add with environment variables
  spin mcp registry local add my-server ./my-mcp-server --env DEBUG=true --env PORT=8080`,
		Args: cobra.MinimumNArgs(2),
		RunE: runMCPRegistryLocalAdd,
	}
	cmd.Flags().StringToString("env", nil, "Environment variables (KEY=VALUE)")

	return cmd
}

func runMCPRegistryLocalAdd(cmd *cobra.Command, args []string) error {
	name := args[0]
	command := args[1]
	cmdArgs := args[2:]

	env, _ := cmd.Flags().GetStringToString("env")

	// Load config.
	loader := config.NewLoaderV2()
	_, err := loader.LoadFromFile(flagConfigFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	mgr := config.NewMCPConfigStore(loader)

	server := config.MCPServer{
		Name:      name,
		Transport: config.MCPTransportStdio,
		Command:   command,
		Args:      cmdArgs,
		Env:       env,
	}

	err = mgr.Add(server)
	if err != nil {
		return err
	}

	configFile := loader.ConfigFileUsed()
	if configFile == "" {
		homeDir, _ := os.UserHomeDir()
		configFile = homeDir + "/.spin/spin.yaml"
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Added local registry '%s' to %s\n", name, configFile)

	return nil
}

// ============================================================================
// Remote Registry Commands
// ============================================================================.

// newMCPRegistryRemoteCmd creates the remote registry command group.
func newMCPRegistryRemoteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remote",
		Short: "Manage remote (SSE/HTTP) registries",
	}
	cmd.AddCommand(newMCPRegistryRemoteAddCmd())

	return cmd
}

// newMCPRegistryRemoteAddCmd creates the command for adding a remote registry.
func newMCPRegistryRemoteAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Add a remote SSE/HTTP registry",
		Long: `Add a remote MCP registry that communicates via SSE or HTTP streaming.

Examples:
  # Add a remote SSE server
  spin mcp registry remote add api-server --url https://api.example.com/mcp

  # Add with custom headers
  spin mcp registry remote add protected --url https://api.example.com/mcp \
    --header "Authorization=Bearer token"

  # Add with HTTP streaming transport
  spin mcp registry remote add http-api --url https://api.example.com/mcp \
    --transport streamable-http

  # Enable dynamic tool discovery
  spin mcp registry remote add dynamic-api --url https://api.example.com/mcp --dynamic`,
		Args: cobra.ExactArgs(1),
		RunE: runMCPRegistryRemoteAdd,
	}
	cmd.Flags().String("url", "", "Server URL (required)")
	cmd.Flags().String("transport", "sse", "Transport type: sse, streamable-http")
	cmd.Flags().StringArray("header", nil, "HTTP headers (Key=Value)")
	cmd.Flags().Bool("dynamic", false, "Enable dynamic tool discovery")
	cmd.Flags().String("oauth-client-id", "", "OAuth client ID")
	cmd.Flags().String("oauth-client-secret", "", "OAuth client secret")
	cmd.Flags().String("oauth-redirect-url", "", "OAuth redirect URL")
	cmd.Flags().StringArray("oauth-scope", nil, "OAuth scopes")
	_ = cmd.MarkFlagRequired("url")

	return cmd
}

func runMCPRegistryRemoteAdd(cmd *cobra.Command, args []string) error {
	name := args[0]

	urlStr, _ := cmd.Flags().GetString("url")
	transport, _ := cmd.Flags().GetString("transport")
	headerFlags, _ := cmd.Flags().GetStringArray("header")
	dynamic, _ := cmd.Flags().GetBool("dynamic")
	oauthClientID, _ := cmd.Flags().GetString("oauth-client-id")
	oauthClientSecret, _ := cmd.Flags().GetString("oauth-client-secret")
	oauthRedirectURL, _ := cmd.Flags().GetString("oauth-redirect-url")
	oauthScopes, _ := cmd.Flags().GetStringArray("oauth-scope")

	// Validate transport.
	var transportType config.MCPTransportType

	switch transport {
	case "sse":
		transportType = config.MCPTransportSSE
	case "streamable-http":
		transportType = config.MCPTransportStreamableHTTP
	default:
return fmt.Errorf("invalid transport: %s (use 'sse' or 'streamable-http'): %w", transport, ErrInvalidTransport)
	}

	// Load config.
	loader := config.NewLoaderV2()
	_, err := loader.LoadFromFile(flagConfigFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	mgr := config.NewMCPConfigStore(loader)

	server := config.MCPServer{
		Name:           name,
		Transport:      transportType,
		URL:            urlStr,
		Headers:        parseHeaders(headerFlags),
		DynamicLoadout: dynamic,
	}

	// Set OAuth if provided.
	if oauthClientID != "" {
		server.OAuth = &config.MCPOAuthConfigV2{
			ClientID:     oauthClientID,
			ClientSecret: oauthClientSecret,
			RedirectURL:  oauthRedirectURL,
			Scopes:       oauthScopes,
		}
	}

	err = mgr.Add(server)
	if err != nil {
		return err
	}

	configFile := loader.ConfigFileUsed()
	if configFile == "" {
		homeDir, _ := os.UserHomeDir()
		configFile = homeDir + "/.spin/spin.yaml"
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Added remote registry '%s' to %s\n", name, configFile)

	return nil
}

// ============================================================================
// Smithery Registry Commands
// ============================================================================.

// newMCPRegistrySmitheryCmd creates the Smithery registry command group.
func newMCPRegistrySmitheryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "smithery",
		Short: "Manage Smithery registries",
	}
	cmd.AddCommand(newMCPRegistrySmitheryAddCmd())

	return cmd
}

// newMCPRegistrySmitheryAddCmd creates the command for adding a Smithery registry.
func newMCPRegistrySmitheryAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <name> <server-path>",
		Short: "Add a Smithery registry",
		Long: `Add a Smithery MCP registry.

Server path formats:
  - @namespace/server-name (short form)
  - namespace/server-name (without @)
  - https://server.smithery.ai/@namespace/server-name (full URL)

The API key can be provided via:
  - --api-key flag
  - SMITHERY_API_KEY environment variable
  - Interactive prompt (if neither is set)

Examples:
  # Add using short path
  spin mcp registry smithery add paper-search @adamamer20/paper-search-mcp

  # Add with explicit API key
  spin mcp registry smithery add paper-search @adamamer20/paper-search-mcp --api-key sk_...

  # Enable dynamic tool discovery
  spin mcp registry smithery add paper-search @adamamer20/paper-search-mcp --dynamic`,
		Args: cobra.ExactArgs(2),
		RunE: runMCPRegistrySmitheryAdd,
	}
	cmd.Flags().String("api-key", "", "Smithery API key (or use SMITHERY_API_KEY env var)")
	cmd.Flags().String("namespace", "", "Smithery namespace (auto-detected from path)")
	cmd.Flags().Bool("dynamic", false, "Enable dynamic tool discovery")

	return cmd
}

func runMCPRegistrySmitheryAdd(cmd *cobra.Command, args []string) error {
	name := args[0]
	serverPath := args[1]

	apiKey, _ := cmd.Flags().GetString("api-key")
	namespace, _ := cmd.Flags().GetString("namespace")
	dynamic, _ := cmd.Flags().GetBool("dynamic")

	// Parse server path.
	serverURL, extractedNamespace, err := parseSmitheryPath(serverPath)
	if err != nil {
		return err
	}

	if namespace == "" {
		namespace = extractedNamespace
	}

	// Get API key from env if not provided.
	if apiKey == "" {
		apiKey = os.Getenv("SMITHERY_API_KEY")
	}

	// Prompt for API key if still not set.
	if apiKey == "" {
		fmt.Fprint(cmd.OutOrStdout(), "Enter Smithery API key: ")

		reader := bufio.NewReader(os.Stdin)

		var input string
		input, err = reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read API key: %w", err)
		}

		apiKey = strings.TrimSpace(input)
		if apiKey == "" {
			return ErrAPIKeyIsRequired
		}
	}

	// Load config.
	loader := config.NewLoaderV2()
	_, err = loader.LoadFromFile(flagConfigFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	mgr := config.NewMCPConfigStore(loader)

	server := config.MCPServer{
		Name:              name,
		Transport:         config.MCPTransportSmithery,
		URL:               serverURL,
		SmitheryAPIKey:    apiKey,
		SmitheryNamespace: namespace,
		DynamicLoadout:    dynamic,
	}

	err = mgr.Add(server)
	if err != nil {
		return err
	}

	configFile := loader.ConfigFileUsed()
	if configFile == "" {
		homeDir, _ := os.UserHomeDir()
		configFile = homeDir + "/.spin/spin.yaml"
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Added Smithery registry '%s' to %s\n", name, configFile)
	fmt.Fprintf(cmd.OutOrStdout(), "  URL: %s\n", serverURL)
	fmt.Fprintf(cmd.OutOrStdout(), "  Namespace: %s\n", namespace)

	return nil
}

// ============================================================================
// Registry List/Get/Remove Commands
// ============================================================================.

// newMCPRegistryListCmd creates the command for listing registries.
func newMCPRegistryListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all configured registries",
		Long: `List all configured MCP registries.

Examples:
  # List all registries
  spin mcp registry list

  # List with JSON output
  spin mcp registry list --format json`,
		RunE: runMCPRegistryList,
	}
	cmd.Flags().String("format", "table", "Output format (table, json)")

	return cmd
}

func runMCPRegistryList(cmd *cobra.Command, _ []string) error {
	format, _ := cmd.Flags().GetString("format")

	loader := config.NewLoaderV2()
	_, err := loader.LoadFromFile(flagConfigFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	mgr := config.NewMCPConfigStore(loader)

	servers, err := mgr.List()
	if err != nil {
		return err
	}

	if len(servers) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No registries configured.")
		fmt.Fprintln(cmd.OutOrStdout(), "\nTo add a registry:")
		fmt.Fprintln(cmd.OutOrStdout(), "  spin mcp registry local add <name> <command> [args...]")
		fmt.Fprintln(cmd.OutOrStdout(), "  spin mcp registry remote add <name> --url <url>")
		fmt.Fprintln(cmd.OutOrStdout(), "  spin mcp registry smithery add <name> @namespace/server")

		return nil
	}

	if format == "json" {
		return outputJSON(servers)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tTYPE\tDYNAMIC\tENDPOINT")

	for _, server := range servers {
		regType := config.GetRegistryTypeName(server)

		dynamic := ""
		if server.DynamicLoadout {
			dynamic = "yes"
		}

		endpoint := formatEndpoint(server)
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", server.Name, regType, dynamic, endpoint)
	}

	w.Flush()

	return nil
}

// newMCPRegistryGetCmd creates the command for getting registry details.
func newMCPRegistryGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <name>",
		Short: "Show details of a registry",
		Long: `Show detailed configuration of a specific registry.

Examples:
  # Show registry details
  spin mcp registry get filesystem

  # Show with JSON output
  spin mcp registry get filesystem --format json`,
		Args: cobra.ExactArgs(1),
		RunE: runMCPRegistryGet,
	}
	cmd.Flags().String("format", "text", "Output format (text, json)")

	return cmd
}

func runMCPRegistryGet(cmd *cobra.Command, args []string) error {
	name := args[0]
	format, _ := cmd.Flags().GetString("format")

	loader := config.NewLoaderV2()
	_, err := loader.LoadFromFile(flagConfigFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	mgr := config.NewMCPConfigStore(loader)

	server, err := mgr.Get(name)
	if err != nil {
		return err
	}

	if format == "json" {
		return outputJSON(server)
	}

	// Text format.
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Name: %s\n", server.Name)
	fmt.Fprintf(out, "Type: %s\n", config.GetRegistryTypeName(*server))
	fmt.Fprintf(out, "Dynamic Loadout: %v\n", server.DynamicLoadout)

	transport := string(server.Transport)
	if transport == "" {
		transport = "stdio"
	}

	fmt.Fprintf(out, "Transport: %s\n", transport)

	if server.Transport.IsRemote() {
		fmt.Fprintf(out, "URL: %s\n", server.URL)

		if len(server.Headers) > 0 {
			fmt.Fprintln(out, "Headers:")

			for key, value := range server.Headers {
				displayValue := value
				if strings.EqualFold(key, "authorization") || strings.Contains(strings.ToLower(key), "secret") {
					displayValue = "***"
				}

				fmt.Fprintf(out, "  %s: %s\n", key, displayValue)
			}
		}

		if server.Transport == config.MCPTransportSmithery {
			fmt.Fprintf(out, "Smithery Namespace: %s\n", server.SmitheryNamespace)
			fmt.Fprintf(out, "Smithery API Key: %s\n", maskAPIKey(server.SmitheryAPIKey))
		}
	} else {
		fmt.Fprintf(out, "Command: %s\n", server.Command)

		if len(server.Args) > 0 {
			fmt.Fprintf(out, "Args: %s\n", strings.Join(server.Args, " "))
		}

		if len(server.Env) > 0 {
			fmt.Fprintln(out, "Environment:")

			for key, value := range server.Env {
				fmt.Fprintf(out, "  %s=%s\n", key, value)
			}
		}
	}

	if server.OAuth != nil {
		fmt.Fprintln(out, "OAuth:")
		fmt.Fprintf(out, "  Client ID: %s\n", server.OAuth.ClientID)

		if server.OAuth.ClientSecret != "" {
			fmt.Fprintln(out, "  Client Secret: ***")
		}
	}

	configFile := loader.ConfigFileUsed()
	if configFile != "" {
		fmt.Fprintf(out, "Source: %s\n", configFile)
	}

	return nil
}

// newMCPRegistryRemoveCmd creates the command for removing a registry.
func newMCPRegistryRemoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove a registry",
		Long: `Remove an MCP registry from your configuration.

Examples:
  # Remove with confirmation
  spin mcp registry remove filesystem

  # Remove without confirmation
  spin mcp registry remove filesystem --yes`,
		Args: cobra.ExactArgs(1),
		RunE: runMCPRegistryRemove,
	}
	cmd.Flags().Bool("yes", false, "Skip confirmation prompt")

	return cmd
}

func runMCPRegistryRemove(cmd *cobra.Command, args []string) error {
	name := args[0]
	yes, _ := cmd.Flags().GetBool("yes")

	loader := config.NewLoaderV2()
	_, err := loader.LoadFromFile(flagConfigFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	mgr := config.NewMCPConfigStore(loader)

	_, err = mgr.Get(name)
	if err != nil {
		return err
	}

	if !yes {
		fmt.Fprintf(cmd.OutOrStdout(), "Remove registry '%s'? (y/N): ", name)

		var response string
		_, _ = fmt.Fscanln(os.Stdin, &response)

		response = strings.ToLower(strings.TrimSpace(response))
		if response != "y" && response != "yes" {
			fmt.Fprintln(cmd.OutOrStdout(), "Canceled.")

			return nil
		}
	}

	err = mgr.Remove(name)
	if err != nil {
		return err
	}

	configFile := loader.ConfigFileUsed()
	if configFile == "" {
		homeDir, _ := os.UserHomeDir()
		configFile = homeDir + "/.spin/spin.yaml"
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Removed registry '%s' from %s\n", name, configFile)

	return nil
}

// ============================================================================
// Search Command
// ============================================================================.

// newMCPSearchCmd creates the command for searching tools.
func newMCPSearchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search for tools across registries",
		Long: `Search for MCP tools across all configured registries.

For Smithery registries, this searches the Smithery API for available tools.
For other registries, it searches the locally configured tools.

Examples:
  # Search all registries
  spin mcp search github

  # Search specific registry
  spin mcp search filesystem --registry=local-fs

  # Search with more results
  spin mcp search database --limit 20

  # Search only verified Smithery servers
  spin mcp search api --verified`,
		Args: cobra.ExactArgs(1),
		RunE: runMCPSearch,
	}
	cmd.Flags().String("registry", "", "Filter by registry name")
	cmd.Flags().Int("limit", 10, "Maximum results")
	cmd.Flags().Bool("verified", false, "Only show tools from verified servers (Smithery)")
	cmd.Flags().String("format", "table", "Output format (table, json)")
	cmd.Flags().String("api-key", "", "Smithery API key (or use SMITHERY_API_KEY env var)")

	return cmd
}

func runMCPSearch(cmd *cobra.Command, args []string) error {
	query := args[0]
	registryFilter, _ := cmd.Flags().GetString("registry")
	limit, _ := cmd.Flags().GetInt("limit")
	verified, _ := cmd.Flags().GetBool("verified")
	format, _ := cmd.Flags().GetString("format")
	apiKey, _ := cmd.Flags().GetString("api-key")

	// Get API key from env if not provided.
	if apiKey == "" {
		apiKey = os.Getenv("SMITHERY_API_KEY")
	}

	loader := config.NewLoaderV2()
	_, err := loader.LoadFromFile(flagConfigFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	mgr := config.NewMCPConfigStore(loader)

	servers, err := mgr.List()
	if err != nil {
		return err
	}

	// Filter by registry if specified.
	if registryFilter != "" {
		var filtered []config.MCPServer

		for _, s := range servers {
			if s.Name == registryFilter {
				filtered = append(filtered, s)
			}
		}

		if len(filtered) == 0 {
return fmt.Errorf("registry '%s' not found: %w", registryFilter, ErrRegistryNotFound)
		}

		servers = filtered
	}

	// Collect search results.
	type searchResult struct {
		ToolName    string `json:"tool_name"`
		Registry    string `json:"registry"`
		Type        string `json:"type"`
		Verified    bool   `json:"verified,omitempty"`
		Description string `json:"description,omitempty"`
	}

	var results []searchResult

	for _, server := range servers {
		if server.Transport == config.MCPTransportSmithery {
			// Search Smithery API.
			key := apiKey
			if key == "" {
				key = server.SmitheryAPIKey
			}

			if key == "" {
				fmt.Fprintf(os.Stderr, "Warning: No API key for Smithery registry '%s', skipping\n", server.Name)

				continue
			}

			smitheryResults, searchErr := searchSmitheryAPI(query, key, limit, verified)
			if searchErr != nil {
				fmt.Fprintf(os.Stderr, "Warning: Failed to search Smithery registry '%s': %v\n", server.Name, searchErr)

				continue
			}

			for _, r := range smitheryResults.Tools {
				results = append(results, searchResult{
					ToolName:    r.Tool.Name,
					Registry:    server.Name,
					Type:        "smithery",
					Verified:    r.Server.Verified,
					Description: r.Tool.Description,
				})
			}
		}
		// Local/remote registries require initialization to list tools.
		// Currently only Smithery API search is implemented.
	}

	if len(results) == 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "No tools found for query: %s\n", query)

		return nil
	}

	if format == "json" {
		return outputJSON(results)
	}

	// Table format.
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "TOOL\tREGISTRY\tTYPE\tVERIFIED\tDESCRIPTION")

	for _, r := range results {
		verifiedStr := ""
		if r.Verified {
			verifiedStr = "yes"
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", r.ToolName, r.Registry, r.Type, verifiedStr, r.Description)
	}

	w.Flush()

	fmt.Fprintf(cmd.OutOrStdout(), "\nFound %d tools.\n", len(results))

	return nil
}

func searchSmitheryAPI(query, apiKey string, limit int, verified bool) (*smitheryToolsResponse, error) {
	apiURL := fmt.Sprintf("https://api.smithery.ai/tools?q=%s&pageSize=%d",
		url.QueryEscape(query), limit)

	if verified {
		apiURL += "&serverVerified=true"
	}

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)

return nil, fmt.Errorf("API error (status %d): %s: %w", resp.StatusCode, string(body), ErrAPIErrorStatus)
	}

	var result smitheryToolsResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// ============================================================================
// List Tools Command
// ============================================================================.

// newMCPListToolsCmd creates the command for listing all tools.
func newMCPListToolsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"tools"},
		Short:   "List all tools from configured registries",
		Long: `List all tools from configured MCP registries.

Shows tools with their source registry and loadout type (static or dynamic).

Examples:
  # List all tools
  spin mcp list

  # Filter by registry
  spin mcp list --registry=smithery

  # JSON output
  spin mcp list --format json`,
		RunE: runMCPListTools,
	}
	cmd.Flags().String("registry", "", "Filter by registry name")
	cmd.Flags().String("format", "table", "Output format (table, json)")

	return cmd
}

func runMCPListTools(cmd *cobra.Command, _ []string) error {
	registryFilter, _ := cmd.Flags().GetString("registry")
	format, _ := cmd.Flags().GetString("format")

	loader := config.NewLoaderV2()
	_, err := loader.LoadFromFile(flagConfigFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	mgr := config.NewMCPConfigStore(loader)

	servers, err := mgr.List()
	if err != nil {
		return err
	}

	// Filter by registry if specified.
	if registryFilter != "" {
		var filtered []config.MCPServer

		for _, s := range servers {
			if s.Name == registryFilter {
				filtered = append(filtered, s)
			}
		}

		servers = filtered
	}

	if len(servers) == 0 {
		if registryFilter != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "No registry found with name '%s'.\n", registryFilter)
		} else {
			fmt.Fprintln(cmd.OutOrStdout(), "No registries configured.")
		}

		return nil
	}

	// Show registries as tool sources (tools are discovered at runtime when registry is initialized).
	type toolInfo struct {
		Tool   string `json:"tool"`
		Source string `json:"source"`
		Type   string `json:"type"`
		Status string `json:"status"`
	}

	var tools []toolInfo
	for _, server := range servers {
		// Placeholder: show registry as a tool source.
		tools = append(tools, toolInfo{
			Tool:   fmt.Sprintf("(tools from %s)", server.Name),
			Source: config.FormatSource(server),
			Type:   config.GetRegistryTypeName(server),
			Status: "configured",
		})
	}

	if format == "json" {
		return outputJSON(tools)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "TOOL\tSOURCE\tTYPE\tSTATUS")

	for _, t := range tools {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", t.Tool, t.Source, t.Type, t.Status)
	}

	w.Flush()

	fmt.Fprintf(cmd.OutOrStdout(), "\n%d registries configured.\n", len(servers))
	fmt.Fprintln(cmd.OutOrStdout(), "Note: Use 'spin mcp search <query>' to discover available tools.")

	return nil
}

// ============================================================================
// Helper Functions
// ============================================================================.

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

// formatEndpoint formats the server endpoint for display (URL or command).
func formatEndpoint(server config.MCPServer) string {
	if server.Transport.IsRemote() {
		return server.URL
	}

	parts := []string{server.Command}
	parts = append(parts, server.Args...)

	return strings.Join(parts, " ")
}

// parseSmitheryPath parses Smithery server path formats.
func parseSmitheryPath(path string) (serverURL string, namespace string, err error) {
	const baseURL = "https://server.smithery.ai"

	if strings.HasPrefix(path, "https://") || strings.HasPrefix(path, "http://") {
		var parsedURL *url.URL
		parsedURL, err = url.Parse(path)
		if err != nil {
			return "", "", fmt.Errorf("invalid URL: %w", err)
		}

		pathParts := strings.Split(strings.TrimPrefix(parsedURL.Path, "/"), "/")
		if len(pathParts) >= 1 {
			ns := pathParts[0]
			if after, ok := strings.CutPrefix(ns, "@"); ok {
				ns = after
			}

			namespace = ns
		}

		return path, namespace, nil
	}

	cleanPath := path
	if after, ok := strings.CutPrefix(path, "@"); ok {
		cleanPath = after
	}

	parts := strings.SplitN(cleanPath, "/", 2)
	if len(parts) != 2 {
return "", "", fmt.Errorf("invalid server path format: %s (expected @namespace/server-name): %w", path, ErrInvalidServerPathFormat)
	}

	namespace = parts[0]
	serverName := parts[1]
	serverURL = fmt.Sprintf("%s/@%s/%s", baseURL, namespace, serverName)

	return serverURL, namespace, nil
}

// maskAPIKey masks an API key for display.
func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return "***"
	}

	return key[:4] + "..." + key[len(key)-4:]
}

// outputJSON outputs data as JSON.
func outputJSON[T any](data T) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("encoding JSON output: %w", err)
	}

	return nil
}
