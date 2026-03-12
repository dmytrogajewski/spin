// Package main provides a TUI demo example.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/dmytrogajewski/spin/internal/ui/adapters"
)

func main() {
	// Create PureTTY adapter.
	ui, err := adapters.NewPureTTY(os.Stdout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create TUI: %v\n", err)
		os.Exit(1)
	}

	// Set up context with cancellation.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle OS signals for clean shutdown.
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		cancel()
	}()

	// Start TUI in background.
	go func() {
		runErr := ui.Run(ctx)
		if runErr != nil {
			fmt.Fprintf(os.Stderr, "TUI error: %v\n", runErr)
			cancel()
		}
	}()

	// Print welcome message.
	_ = ui.PrintLine("╔══════════════════════════════════════════╗")
	_ = ui.PrintLine("║   Spin TUI Demo - Minimal Example       ║")
	_ = ui.PrintLine("╚══════════════════════════════════════════╝")
	_ = ui.PrintLine("")
	_ = ui.PrintLine("Welcome to the Spin TUI minimal demo!")
	_ = ui.PrintLine("This example shows the simplest possible TUI usage.")
	_ = ui.PrintLine("")
	_ = ui.PrintLine("Commands:")
	_ = ui.PrintLine("  help    - Show this message")
	_ = ui.PrintLine("  hello   - Print a greeting")
	_ = ui.PrintLine("  quit    - Exit the demo")
	_ = ui.PrintLine("")
	_ = ui.PrintLine("Type a command and press Enter:")

	// Main input loop.
	for {
		select {
		case <-ctx.Done():
			// Clean shutdown.
			_ = ui.Stop()
			_, _ = fmt.Fprintln(os.Stdout, "\nGoodbye!")

			return

		case line, ok := <-ui.RequestInput():
			if !ok {
				// Input channel closed.
				_ = ui.Stop()

				return
			}

			// Handle commands.
			switch line {
			case "quit", "exit", "q":
				_ = ui.PrintLine("Exiting demo...")
				cancel()

			case "help", "h", "?":
				_ = ui.PrintLine("")
				_ = ui.PrintLine("Available commands:")
				_ = ui.PrintLine("  help    - Show this message")
				_ = ui.PrintLine("  hello   - Print a greeting")
				_ = ui.PrintLine("  quit    - Exit the demo")
				_ = ui.PrintLine("")

			case "hello":
				_ = ui.PrintLine("")
				_ = ui.PrintLine("Hello from Spin TUI! 👋")
				_ = ui.PrintLine("The quick brown fox jumps over the lazy dog.")
				_ = ui.PrintLine("")

			case "":
				// Empty line, do nothing.

			default:
				_ = ui.PrintLine("")
				_ = ui.PrintLine(fmt.Sprintf("Unknown command: '%s'", line))
				_ = ui.PrintLine("Type 'help' for available commands.")
				_ = ui.PrintLine("")
			}
		}
	}
}
