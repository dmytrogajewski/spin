// Package main provides a TUI demo example.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/dmytrogajewski/spin/internal/ui/adapters"
)

// check panics if err is non-nil. Used in example/demo programs only.
func check(err error) {
	if err != nil {
		log.Fatalf("TUI error: %v", err)
	}
}

func main() {
	ui, err := adapters.NewPureTTY(os.Stdout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create TUI: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	startSignalHandler(cancel)
	startTUI(ctx, ui, cancel)
	printWelcome(ui)
	runInputLoop(ctx, ui, cancel)
}

// startSignalHandler listens for interrupt signals and cancels the context.
func startSignalHandler(cancel context.CancelFunc) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		cancel()
	}()
}

// startTUI runs the TUI adapter in a background goroutine.
func startTUI(ctx context.Context, ui *adapters.PureTTY, cancel context.CancelFunc) {
	go func() {
		runErr := ui.Run(ctx)
		if runErr != nil {
			fmt.Fprintf(os.Stderr, "TUI error: %v\n", runErr)
			cancel()
		}
	}()
}

// printWelcome prints the welcome message and available commands.
func printWelcome(ui *adapters.PureTTY) {
	lines := []string{
		"╔══════════════════════════════════════════╗",
		"║   Spin TUI Demo - Minimal Example       ║",
		"╚══════════════════════════════════════════╝",
		"",
		"Welcome to the Spin TUI minimal demo!",
		"This example shows the simplest possible TUI usage.",
		"",
		"Commands:",
		"  help    - Show this message",
		"  hello   - Print a greeting",
		"  quit    - Exit the demo",
		"",
		"Type a command and press Enter:",
	}

	for _, line := range lines {
		check(ui.PrintLine(line))
	}
}

// printHelp prints the help message.
func printHelp(ui *adapters.PureTTY) {
	lines := []string{
		"",
		"Available commands:",
		"  help    - Show this message",
		"  hello   - Print a greeting",
		"  quit    - Exit the demo",
		"",
	}

	for _, line := range lines {
		check(ui.PrintLine(line))
	}
}

// runInputLoop handles user input until context is canceled.
func runInputLoop(ctx context.Context, ui *adapters.PureTTY, cancel context.CancelFunc) {
	for {
		select {
		case <-ctx.Done():
			check(ui.Stop())

			_, _ = fmt.Fprintln(os.Stdout, "\nGoodbye!")

			return

		case line, ok := <-ui.RequestInput():
			if !ok {
				check(ui.Stop())

				return
			}

			handleCommand(ui, line, cancel)
		}
	}
}

// handleCommand processes a single user command.
func handleCommand(ui *adapters.PureTTY, line string, cancel context.CancelFunc) {
	switch line {
	case "quit", "exit", "q":
		check(ui.PrintLine("Exiting demo..."))

		cancel()

	case "help", "h", "?":
		printHelp(ui)

	case "hello":
		check(ui.PrintLine(""))
		check(ui.PrintLine("Hello from Spin TUI! 👋"))
		check(ui.PrintLine("The quick brown fox jumps over the lazy dog."))
		check(ui.PrintLine(""))

	case "":
		// Empty line, do nothing.

	default:
		check(ui.PrintLine(""))
		check(ui.PrintLine(fmt.Sprintf("Unknown command: '%s'", line)))
		check(ui.PrintLine("Type 'help' for available commands."))
		check(ui.PrintLine(""))
	}
}
