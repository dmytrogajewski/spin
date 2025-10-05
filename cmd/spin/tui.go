package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

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

	// Unmarshal to TUI Config
	var cfg tui.Config
	if err := configLoader.Unmarshal(&cfg); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Create TUI model with config
	m := tui.NewModelWithConfig(&cfg)

	// Configure Bubble Tea program options
	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),       // Use alternate screen buffer (preserves terminal history)
		tea.WithMouseCellMotion(), // Enable mouse support for future phases
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
