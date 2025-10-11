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
	// Create PureTTY adapter
	ui, err := adapters.NewPureTTY(os.Stdout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create TUI: %v\n", err)
		os.Exit(1)
	}

	// Set up context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle OS signals for clean shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	// Start TUI in background
	go func() {
		if err := ui.Run(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
			cancel()
		}
	}()

	// Print welcome message
	ui.PrintLine("╔══════════════════════════════════════════╗")
	ui.PrintLine("║   Spin TUI Demo - Minimal Example       ║")
	ui.PrintLine("╚══════════════════════════════════════════╝")
	ui.PrintLine("")
	ui.PrintLine("Welcome to the Spin TUI minimal demo!")
	ui.PrintLine("This example shows the simplest possible TUI usage.")
	ui.PrintLine("")
	ui.PrintLine("Commands:")
	ui.PrintLine("  help    - Show this message")
	ui.PrintLine("  hello   - Print a greeting")
	ui.PrintLine("  quit    - Exit the demo")
	ui.PrintLine("")
	ui.PrintLine("Type a command and press Enter:")

	// Main input loop
	for {
		select {
		case <-ctx.Done():
			// Clean shutdown
			ui.Stop()
			fmt.Println("\nGoodbye!")
			return

		case line, ok := <-ui.RequestInput():
			if !ok {
				// Input channel closed
				ui.Stop()
				return
			}

			// Handle commands
			switch line {
			case "quit", "exit", "q":
				ui.PrintLine("Exiting demo...")
				cancel()

			case "help", "h", "?":
				ui.PrintLine("")
				ui.PrintLine("Available commands:")
				ui.PrintLine("  help    - Show this message")
				ui.PrintLine("  hello   - Print a greeting")
				ui.PrintLine("  quit    - Exit the demo")
				ui.PrintLine("")

			case "hello":
				ui.PrintLine("")
				ui.PrintLine("Hello from Spin TUI! 👋")
				ui.PrintLine("The quick brown fox jumps over the lazy dog.")
				ui.PrintLine("")

			case "":
				// Empty line, do nothing

			default:
				ui.PrintLine("")
				ui.PrintLine(fmt.Sprintf("Unknown command: '%s'", line))
				ui.PrintLine("Type 'help' for available commands.")
				ui.PrintLine("")
			}
		}
	}
}
