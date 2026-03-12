package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/dmytrogajewski/spin/internal/config"
)

var (
	ErrUnsupportedFormat = errors.New("unsupported format")
	ErrValidationFailed = errors.New("validation failed")
	ErrValidationFailed2 = errors.New("validation failed")
	ErrNoConfigFile = errors.New("no config file")
	ErrNoConfigFile2 = errors.New("no config file")
	ErrNoEditorFoundSetEditorOr = errors.New("no editor found. Set $EDITOR or $VISUAL environment variable")
	ErrKeyNotFound = errors.New("key not found")
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
	cmd.AddCommand(newConfigGetCmd())
	cmd.AddCommand(newConfigSetCmd())

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
func runConfigShow(cmd *cobra.Command, _ []string) error {
	format, _ := cmd.Flags().GetString("format")

	// Use V2 loader with env override support.
	loaderV2 := config.NewLoaderV2()

	var (
		cfgV2 *config.V2
		errV2 error
	)

	if cf := flagConfigFile(cmd); cf != "" {
		cfgV2, errV2 = loaderV2.LoadFromFileWithEnv(cf)
	} else {
		cfgV2, errV2 = loaderV2.Load()
	}

	if errV2 != nil {
		return fmt.Errorf("failed to load config: %w", errV2)
	}

	// Successfully loaded V2 config, show it.
	if format == "text" || format == "yaml" {
		fmt.Fprintf(cmd.OutOrStdout(), "# Configuration V2\n\n")
	}

	switch format {
	case "json":
		return printJSON(cmd.OutOrStdout(), cfgV2)
	case "yaml", "text":
		return printYAML(cmd.OutOrStdout(), cfgV2)
	default:
return fmt.Errorf("unsupported format: %s (use: text, json, yaml): %w", format, ErrUnsupportedFormat)
	}
}

// runConfigValidate implements the 'config validate' command.
func runConfigValidate(cmd *cobra.Command, _ []string) error {
	file, _ := cmd.Flags().GetString("file")

	// Determine which file to load.
	configPath := flagConfigFile(cmd)
	if file != "" {
		configPath = file
	}

	// Use V2 loader.
	loaderV2 := config.NewLoaderV2()

	var (
		cfgV2 *config.V2
		errV2 error
	)

	if configPath != "" {
		cfgV2, errV2 = loaderV2.LoadFromFile(configPath)
	} else {
		// Try loading with LoadWithEnv which uses default search paths.
		cfgV2, errV2 = loaderV2.LoadWithEnv()
		if errV2 != nil {
			// Try plain Load() without env.
			cfgV2, errV2 = loaderV2.Load()
		}
	}

	if errV2 != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "✗ Configuration is invalid\n\n")
		fmt.Fprintf(cmd.ErrOrStderr(), "Error: %v\n", errV2)

		return ErrValidationFailed
	}

	// V2 config loaded successfully (possibly migrated from V1), validate it.
	err := cfgV2.Validate()
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "✗ Configuration V2 is invalid\n\n")
		fmt.Fprintf(cmd.ErrOrStderr(), "Error: %v\n", err)

		return ErrValidationFailed2
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✓ Configuration V2 is valid (version: %s)\n", cfgV2.Version)

	return nil
}

// runConfigPath implements the 'config path' command.
func runConfigPath(cmd *cobra.Command, _ []string) error {
	showAll, _ := cmd.Flags().GetBool("all")

	loaderV2 := config.NewLoaderV2()

	var errV2 error

	if cf := flagConfigFile(cmd); cf != "" {
		_, errV2 = loaderV2.LoadFromFile(cf)
	} else {
		_, errV2 = loaderV2.Load()
	}

	if errV2 != nil {
		// Config not found - show search paths.
		if showAll {
			fmt.Fprintf(cmd.OutOrStdout(), "No configuration file found. Search paths:\n")

			for _, path := range getConfigSearchPaths() {
				fmt.Fprintf(cmd.OutOrStdout(), "  - %s\n", path)
			}
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "No configuration file found. Using defaults.\n")
			fmt.Fprintf(cmd.OutOrStdout(), "Run 'spin config path --all' to see search paths.\n")
		}

		return ErrNoConfigFile
	}

	configPath := loaderV2.ConfigFileUsed()
	if configPath == "" {
		fmt.Fprintf(cmd.OutOrStdout(), "No configuration file found. Using defaults.\n")

		return ErrNoConfigFile2
	}

	fmt.Fprintf(cmd.OutOrStdout(), "%s\n", configPath)

	return nil
}

// runConfigEdit implements the 'config edit' command.
func runConfigEdit(cmd *cobra.Command, _ []string) error {
	noValidate, _ := cmd.Flags().GetBool("no-validate")

	// Load or determine config path.
	loaderV2 := config.NewLoaderV2()
	_, _ = loaderV2.Load()

	configPath := loaderV2.ConfigFileUsed()
	if configPath == "" {
		// Create default config in home directory.
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}

		configDir := filepath.Join(homeDir, ".spin")
		configPath = filepath.Join(configDir, "spin.yaml")

		// Create directory if needed.
		err = os.MkdirAll(configDir, 0755)
		if err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}

		// Create default config.
		err = createDefaultConfig(configPath)
		if err != nil {
			return fmt.Errorf("failed to create default config: %w", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Created new configuration file: %s\n", configPath)
	}

	// Find editor.
	editor := getEditor()
	if editor == "" {
		return ErrNoEditorFoundSetEditorOr
	}

	// Open editor.
	editorCmd := exec.Command(editor, configPath)
	editorCmd.Stdin = os.Stdin
	editorCmd.Stdout = os.Stdout
	editorCmd.Stderr = os.Stderr

	err := editorCmd.Run()
	if err != nil {
		return fmt.Errorf("editor failed: %w", err)
	}

	// Validate after editing.
	if !noValidate {
		newLoaderV2 := config.NewLoaderV2()
		var cfgV2 *config.V2
		cfgV2, err = newLoaderV2.LoadFromFile(configPath)
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "⚠ Warning: configuration has errors:\n%v\n", err)
			fmt.Fprintf(cmd.ErrOrStderr(), "\nRun 'spin config validate' for details.\n")
		}

		err = cfgV2.Validate()
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "⚠ Warning: configuration validation failed:\n%v\n", err)
			fmt.Fprintf(cmd.ErrOrStderr(), "\nRun 'spin config validate' for details.\n")
		}
	}

	return nil
}

// Helper functions.

// printJSON prints data as JSON.
func printJSON[T any](out io.Writer, data T) error {
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("encoding JSON: %w", err)
	}

	return nil
}

// printYAML prints data as YAML.
func printYAML[T any](out io.Writer, data T) error {
	encoder := yaml.NewEncoder(out)

	encoder.SetIndent(2)
	defer encoder.Close()

	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("encoding YAML: %w", err)
	}

	return nil
}

// getEditor returns the preferred editor.
func getEditor() string {
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor
	}

	if visual := os.Getenv("VISUAL"); visual != "" {
		return visual
	}

	// Try common editors.
	for _, editor := range []string{"vi", "vim", "nano", "emacs"} {
		_, err := exec.LookPath(editor)
		if err == nil {
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

func newConfigGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <key>",
		Short: "Get a configuration value",
		Long: `Get a specific configuration value by key.

Examples:
  spin config get llm.model
  spin config get ace.enabled`,
		Args: cobra.ExactArgs(1),
		RunE: runConfigGet,
	}
}

func newConfigSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Long: `Set a specific configuration value by key.

Examples:
  spin config set llm.model gpt-4
  spin config set ace.enabled true`,
		Args: cobra.ExactArgs(2),
		RunE: runConfigSet,
	}
}

// runConfigGet implements the 'config get' command.
func runConfigGet(cmd *cobra.Command, args []string) error {
	key := args[0]

	loaderV2 := config.NewLoaderV2()
	_, err := loaderV2.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	value := loaderV2.Get(key)
	if value == nil {
return fmt.Errorf("key not found: %s: %w", key, ErrKeyNotFound)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "%v\n", value)

	return nil
}

// runConfigSet implements the 'config set' command.
func runConfigSet(cmd *cobra.Command, args []string) error {
	key := args[0]
	value := args[1]

	loaderV2 := config.NewLoaderV2()
	_, err := loaderV2.Load()
	if err != nil {
		// If config doesn't exist, that's okay - we'll create it.
		_ = err
	}

	loaderV2.Set(key, value)

	fmt.Fprintf(cmd.OutOrStdout(), "Set %s = %s\n", key, value)
	fmt.Fprintf(cmd.OutOrStdout(), "Note: Changes are in-memory only. Use 'spin config edit' to persist.\n")

	return nil
}

// createDefaultConfig creates a default configuration file.
func createDefaultConfig(path string) error {
	defaultConfig := `# Spin Configuration File V2
# See: https://docs.spin.dev/configuration

version: "2.0"

llm:
  # Provider: openai, anthropic, ollama, lmstudio, openai-compatible
  provider: ollama

  # Model name
  model: qwen2.5-coder:7b

  # Temperature (0.0-2.0)
  temperature: 0.7

  # Max tokens per request
  max_tokens: 8192

  # Request timeout
  timeout: 5m

  # API endpoint (optional, uses provider default)
  # base_url: http://localhost:11434/v1

  # Authentication (recommended: use keystore or environment variable)
  # api_key: ""

agent:
  # Maximum conversation turns
  max_turns: 50

  # Agent execution timeout
  timeout: 60m

  # Working directory
  work_dir: "."

  # Require approval for dangerous operations
  require_approval: false

  # Logging
  log_level: info  # debug, info, warn, error
  log_format: text  # text, json
  debug: false

  # Cycle detection
  cycle_detection:
    enabled: true
    window_size: 3
    similarity_thresh: 0.8
    tool_repeat_limit: 3
    error_repeat_limit: 3

ace:
  # Enable Agentic Context Engineering
  enabled: false

  # Paths
  playbook_path: "~/.spin/ace/playbooks/default.json"
  trajectory_path: "~/.spin/ace/trajectories/"

  # Retrieval settings
  top_k: 5
  min_score: 0.3

security:
  # Sandbox mode: none, workspace-only
  sandbox_mode: workspace-only

  # Policy file path
  # policy_file: ""

  # Allowed commands
  # allowed_commands: []

protocol:
  # Enable MCP (Model Context Protocol)
  enable_mcp: false

  # MCP Servers
  # mcp_servers:
  #   - name: filesystem
  #     command: npx
  #     args:
  #       - -y
  #       - @modelcontextprotocol/server-filesystem
  #       - /workspace

  # Enable Git integration
  enable_git: true

  # Enable Shell integration
  enable_shell: true

  # Shell timeout
  shell_timeout: 5m
`

	if err := os.WriteFile(path, []byte(defaultConfig), 0644); err != nil {
		return fmt.Errorf("writing default config: %w", err)
	}

	return nil
}
