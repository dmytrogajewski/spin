package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/dmytrogajewski/spin/internal/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// newConfigCmd creates the config management command.
func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
		Long: `Manage Spin configuration files and settings.

Examples:
  # Show current configuration
  spin config show

  # Validate configuration file
  spin config validate

  # Show configuration file path
  spin config path

  # Edit configuration in $EDITOR
  spin config edit`,
	}

	cmd.AddCommand(newConfigShowCmd())
	cmd.AddCommand(newConfigValidateCmd())
	cmd.AddCommand(newConfigEditCmd())
	cmd.AddCommand(newConfigPathCmd())

	return cmd
}

func newConfigShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show current configuration",
		Long: `Show current configuration from active config file.

Displays the merged configuration including defaults, file settings,
and environment variable overrides. Sensitive values are redacted.

Examples:
  # Show as YAML (default)
  spin config show

  # Show as JSON
  spin config show --format json

  # Show as YAML
  spin config show --format yaml`,
		RunE: runConfigShow,
	}
	cmd.Flags().String("format", "yaml", "Output format (text, json, yaml)")
	return cmd
}

func newConfigValidateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate configuration file",
		Long: `Validate configuration file syntax and semantics.

Checks YAML/JSON syntax and validates field types, required fields,
and enum values. Returns exit code 0 on success, 1 on failure.

Examples:
  # Validate default config
  spin config validate

  # Validate specific file
  spin config validate --file /path/to/config.yaml`,
		RunE: runConfigValidate,
	}
	cmd.Flags().String("file", "", "Config file to validate (default: use search paths)")
	return cmd
}

func newConfigEditCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edit",
		Short: "Edit configuration file",
		Long: `Edit configuration file in $EDITOR.

Opens the configuration file in your preferred editor ($EDITOR or $VISUAL).
Creates a default config file if none exists. Validates after editing.

Examples:
  # Edit config in $EDITOR
  spin config edit

  # Edit without validation
  spin config edit --no-validate`,
		RunE: runConfigEdit,
	}
	cmd.Flags().Bool("no-validate", false, "Skip validation after editing")
	return cmd
}

func newConfigPathCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "path",
		Short: "Show configuration file path",
		Long: `Show path to active configuration file.

Displays the path to the configuration file being used.
Shows search paths if no config file is found.

Examples:
  # Show config file path
  spin config path

  # Show all search paths
  spin config path --all`,
		RunE: runConfigPath,
	}
	cmd.Flags().Bool("all", false, "Show all search paths")
	return cmd
}

// runConfigShow implements the 'config show' command.
func runConfigShow(cmd *cobra.Command, args []string) error {
	format, _ := cmd.Flags().GetString("format")

	// Load config
	loader := config.NewLoader()
	if err := loader.Load(flagConfigFile); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get all settings
	settings := loader.AllSettings()

	// Redact sensitive values
	redactSensitiveValues(settings)

	// Show config file source if using text format
	if format == "text" || format == "yaml" {
		configPath := loader.ConfigFileUsed()
		if configPath != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "# Configuration: %s\n\n", configPath)
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "# No configuration file (using defaults)\n\n")
		}
	}

	// Display based on format
	switch format {
	case "json":
		return printJSON(cmd.OutOrStdout(), settings)
	case "yaml", "text":
		return printYAML(cmd.OutOrStdout(), settings)
	default:
		return fmt.Errorf("unsupported format: %s (use: text, json, yaml)", format)
	}
}

// runConfigValidate implements the 'config validate' command.
func runConfigValidate(cmd *cobra.Command, args []string) error {
	file, _ := cmd.Flags().GetString("file")

	// Determine which file to load
	configPath := flagConfigFile
	if file != "" {
		configPath = file
	}

	// Load config
	loader := config.NewLoader()
	var err error
	if configPath != "" {
		err = loader.LoadFromFile(configPath)
	} else {
		err = loader.Load("")
	}

	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "✗ Configuration is invalid\n\n")
		fmt.Fprintf(cmd.ErrOrStderr(), "Error: %v\n", err)
		return fmt.Errorf("validation failed")
	}

	// Get the actual config file used
	actualPath := loader.ConfigFileUsed()
	if actualPath == "" {
		fmt.Fprintf(cmd.ErrOrStderr(), "✗ No configuration file found\n")
		return fmt.Errorf("no config file")
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✓ Configuration is valid: %s\n", actualPath)
	return nil
}

// runConfigPath implements the 'config path' command.
func runConfigPath(cmd *cobra.Command, args []string) error {
	showAll, _ := cmd.Flags().GetBool("all")

	loader := config.NewLoader()
	if err := loader.Load(flagConfigFile); err != nil {
		// Config not found - show search paths
		if showAll {
			fmt.Fprintf(cmd.OutOrStdout(), "No configuration file found. Search paths:\n")
			for _, path := range getConfigSearchPaths() {
				fmt.Fprintf(cmd.OutOrStdout(), "  - %s\n", path)
			}
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "No configuration file found. Using defaults.\n")
			fmt.Fprintf(cmd.OutOrStdout(), "Run 'spin config path --all' to see search paths.\n")
		}
		return fmt.Errorf("no config file")
	}

	configPath := loader.ConfigFileUsed()
	if configPath == "" {
		fmt.Fprintf(cmd.OutOrStdout(), "No configuration file found. Using defaults.\n")
		return fmt.Errorf("no config file")
	}

	fmt.Fprintf(cmd.OutOrStdout(), "%s\n", configPath)
	return nil
}

// runConfigEdit implements the 'config edit' command.
func runConfigEdit(cmd *cobra.Command, args []string) error {
	noValidate, _ := cmd.Flags().GetBool("no-validate")

	// Load or determine config path
	loader := config.NewLoader()
	_ = loader.Load(flagConfigFile)

	configPath := loader.ConfigFileUsed()
	if configPath == "" {
		// Create default config in home directory
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}

		configDir := filepath.Join(homeDir, ".spin")
		configPath = filepath.Join(configDir, "spin.yaml")

		// Create directory if needed
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}

		// Create default config
		if err := createDefaultConfig(configPath); err != nil {
			return fmt.Errorf("failed to create default config: %w", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Created new configuration file: %s\n", configPath)
	}

	// Find editor
	editor := getEditor()
	if editor == "" {
		return fmt.Errorf("no editor found. Set $EDITOR or $VISUAL environment variable")
	}

	// Open editor
	editorCmd := exec.Command(editor, configPath)
	editorCmd.Stdin = os.Stdin
	editorCmd.Stdout = os.Stdout
	editorCmd.Stderr = os.Stderr

	if err := editorCmd.Run(); err != nil {
		return fmt.Errorf("editor failed: %w", err)
	}

	// Validate after editing
	if !noValidate {
		newLoader := config.NewLoader()
		if err := newLoader.LoadFromFile(configPath); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "⚠ Warning: configuration has errors:\n%v\n", err)
			fmt.Fprintf(cmd.ErrOrStderr(), "\nRun 'spin config validate' for details.\n")
		}
	}

	return nil
}

// Helper functions

// redactSensitiveValues redacts sensitive values in the config map.
func redactSensitiveValues(m map[string]interface{}) {
	sensitiveKeys := []string{
		"api_key", "apikey", "api-key",
		"secret", "password", "token",
		"credentials", "credential",
		"key", "private_key", "private-key",
	}

	for k, v := range m {
		lowerKey := strings.ToLower(k)

		// Check if key is sensitive
		for _, sensitive := range sensitiveKeys {
			if strings.Contains(lowerKey, sensitive) {
				m[k] = "<redacted>"
				continue
			}
		}

		// Recursively redact nested maps
		if nested, ok := v.(map[string]interface{}); ok {
			redactSensitiveValues(nested)
		}
	}
}

// printJSON prints data as JSON.
func printJSON[T any](out io.Writer, data T) error {
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// printYAML prints data as YAML.
func printYAML[T any](out io.Writer, data T) error {
	encoder := yaml.NewEncoder(out)
	encoder.SetIndent(2)
	defer encoder.Close()
	return encoder.Encode(data)
}

// getEditor returns the preferred editor.
func getEditor() string {
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor
	}
	if visual := os.Getenv("VISUAL"); visual != "" {
		return visual
	}

	// Try common editors
	for _, editor := range []string{"vi", "vim", "nano", "emacs"} {
		if _, err := exec.LookPath(editor); err == nil {
			return editor
		}
	}

	return ""
}

// getConfigSearchPaths returns the list of config search paths.
func getConfigSearchPaths() []string {
	homeDir, _ := os.UserHomeDir()

	return []string{
		"./spin.yaml",
		filepath.Join(homeDir, ".spin", "spin.yaml"),
		"/etc/spin/spin.yaml",
	}
}

// createDefaultConfig creates a default configuration file.
func createDefaultConfig(path string) error {
	defaultConfig := `# Spin Configuration File
# See: https://docs.spin.dev/configuration

llm:
  # Provider: openai, anthropic, ollama, lmstudio, openai-compatible
  provider: openai

  # Model name
  model: gpt-4o

  # API endpoint (optional, uses provider default)
  # base_url: https://api.openai.com/v1

  # Request timeout
  timeout: 60s

  # Authentication (recommended: use keystore)
  # Store key: spin config set-key my-key sk-...
  # Then reference: key_name: my-key
  #
  # Or use environment variable:
  # export OPENAI_API_KEY=sk-...

# Sandbox settings
sandbox:
  # Mode: read-only, workspace-only, full-access
  mode: workspace-only

# Appearance (TUI only)
appearance:
  theme: auto  # auto, dark, light, plain
  no_color: false

# Logging
logging:
  level: info  # debug, info, warn, error
  format: text  # text, json

# MCP Servers (Model Context Protocol)
# mcp_servers:
#   - name: filesystem
#     command: npx
#     args:
#       - -y
#       - @modelcontextprotocol/server-filesystem
#       - /workspace
`

	return os.WriteFile(path, []byte(defaultConfig), 0644)
}
