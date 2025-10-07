package main

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/dmytrogajewski/spin/internal/core"
	"github.com/dmytrogajewski/spin/internal/llm/builder"
	"github.com/dmytrogajewski/spin/internal/tui"
)

// tuiCmd represents the TUI mode command.
// This is the default command when spin is run without arguments.
var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Start interactive terminal UI (default)",
	Long: `Launch Spin in interactive TUI mode with full Bubble Tea interface.

The TUI provides:
- Interactive chat with AI
- Tool approval dialogs
- File picker (@-trigger)
- Backtrack mode (Esc-Esc)
- Syntax-highlighted code blocks
- Real-time streaming responses

This is the default mode when running 'spin' without arguments.`,
	RunE: runTUI,
}

// Note: tuiCmd is added to rootCmd in root.go newRootCmd() function
// TUI-specific flags will be added in future phases:
// - --no-color: Disable colors
// - --theme: Color theme (auto, dark, light)

// runTUI starts the Bubble Tea TUI application.
func runTUI(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Load configuration
	configLoader, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Apply flag overrides to config loader
	if flagModel != "" {
		configLoader.Set("model", flagModel)
	}
	if flagProvider != "" {
		configLoader.Set("provider", flagProvider)
	}
	if flagSandbox != "" {
		configLoader.Set("sandbox_mode", flagSandbox)
	}
	if flagWorkDir != "" {
		configLoader.Set("work_dir", flagWorkDir)
	}

	// Unmarshal to TUI Config (for display)
	var tuiCfg tui.Config
	if err := configLoader.Unmarshal(&tuiCfg); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Apply defaults to TUI config
	if tuiCfg.SandboxMode == "" {
		tuiCfg.SandboxMode = "workspace-write"
	}
	if tuiCfg.WorkDir == "" {
		wd, err := os.Getwd()
		if err == nil {
			tuiCfg.WorkDir = wd
		}
	}

	// Start with defaults and overlay config
	coreCfg := core.DefaultConfig()
	if err := configLoader.Unmarshal(coreCfg); err != nil {
		return fmt.Errorf("failed to unmarshal core config: %w", err)
	}

	// Set working directory if not already set
	if coreCfg.WorkDir == "" {
		wd, err := os.Getwd()
		if err == nil {
			coreCfg.WorkDir = wd
		}
	}

	// Create auth manager and build LLM provider
	authMgr := createAuthManager()
	b := builder.NewBuilder(configLoader, authMgr)
	providerCfg := builder.Config{
		Provider: flagProvider,
		Model:    flagModel,
	}
	provider, err := b.Build(ctx, providerCfg)
	if err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}
	defer provider.Close()

	// Initialize debug logging
	if err := tui.InitDebugLogging(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to init debug logging: %v\n", err)
	}

	// Create TUI model with config and provider
	m, err := tui.NewModelWithLLM(&tuiCfg, coreCfg, provider)
	if err != nil {
		return fmt.Errorf("failed to create TUI model: %w", err)
	}
	defer m.Close()

	// Configure Bubble Tea program options
	p := tea.NewProgram(
		m,
		tea.WithAltScreen(), // Use alternate screen buffer (preserves terminal history)
		// Mouse support disabled to prevent scroll wheel from generating escape sequences
		// tea.WithMouseCellMotion(), // Commented out to fix input corruption on scroll
	)

	// Run the program
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	// Check for errors in final model
	if model, ok := finalModel.(tui.Model); ok {
		if model.Err() != nil {
			return model.Err()
		}
	}

	return nil
}
