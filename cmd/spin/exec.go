package main

import (
	"context"
	"fmt"
	"os"

	"github.com/dmytrogajewski/spin/internal/auth"
	"github.com/dmytrogajewski/spin/internal/config"
	execpkg "github.com/dmytrogajewski/spin/internal/exec"
	"github.com/dmytrogajewski/spin/internal/llm"
	"github.com/dmytrogajewski/spin/internal/llm/builder"
	"github.com/spf13/cobra"
)

// newExecCmd creates the exec command for non-interactive execution.
func newExecCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exec [prompt]",
		Short: "Non-interactive execution mode",
		Long: `Execute Spin in non-interactive mode for CI/CD and automation.

Examples:
  spin exec "run all tests and fix failures"
  echo "refactor authentication" | spin exec
  spin exec --timeout 5m "deploy to staging"
  spin exec --format json "analyze code" | jq`,
		RunE: runExec,
	}

	// Exec-specific flags
	cmd.Flags().Bool("auto-approve", false, "Automatically approve all operations (DANGEROUS)")
	cmd.Flags().String("timeout", "", "Maximum execution time (e.g., 5m, 1h)")
	cmd.Flags().String("format", "text", "Output format (text, json)")
	cmd.Flags().Bool("no-stream", false, "Disable streaming output")
	cmd.Flags().Bool("exit-on-error", true, "Exit immediately on first error")

	return cmd
}

// runExec executes the exec mode.
func runExec(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Parse arguments using internal/exec package
	execArgs, err := execpkg.Parse(args, os.Stdin)
	if err != nil {
		return err
	}

	// Load configuration
	configLoader, err := loadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Create auth manager
	authMgr := createAuthManager()

	// Build LLM provider
	provider, err := buildProvider(ctx, configLoader, authMgr)
	if err != nil {
		return fmt.Errorf("create provider: %w", err)
	}
	defer provider.Close()

	// Create context with timeout
	var cancel context.CancelFunc
	if execArgs.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, execArgs.Timeout)
	} else {
		ctx, cancel = context.WithCancel(ctx)
	}
	defer cancel()

	// Setup signal handling
	_ = execpkg.SetupSignals(ctx, cancel)

	// Execute task with real provider
	if err := execpkg.RunWithProvider(ctx, execArgs, provider); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", execpkg.FormatError(err))
		os.Exit(int(execpkg.GetExitCode(err)))
	}

	return nil
}

// loadConfig loads configuration from file or defaults.
func loadConfig() (*config.Loader, error) {
	configLoader := config.NewLoader()

	// Load from explicit config file if provided
	if flagConfigFile != "" {
		if err := configLoader.LoadFromFile(flagConfigFile); err != nil {
			return nil, err
		}
	} else {
		// Try to load from default locations (ignore error if not found)
		_ = configLoader.Load("")
	}

	return configLoader, nil
}

// createAuthManager creates an auth manager with platform-specific keystore.
func createAuthManager() *auth.Manager {
	keystore := auth.NewKeystore()
	return auth.NewManager(keystore)
}

// buildProvider creates an LLM provider from configuration.
func buildProvider(ctx context.Context, configLoader *config.Loader, authMgr *auth.Manager) (llm.Provider, error) {
	// Create builder
	b := builder.NewBuilder(configLoader, authMgr)

	// Build provider with flags as overrides
	cfg := builder.Config{
		Provider: flagProvider,
		Model:    flagModel,
		// Additional flags can be added here in the future
		// BaseURL:  flagBaseURL,
		// KeyName:  flagKeyName,
	}

	return b.Build(ctx, cfg)
}
