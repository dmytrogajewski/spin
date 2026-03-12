// Package main provides a TUI streaming example.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/dmytrogajewski/spin/internal/ui/adapters"
)

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
	printHeader(ui)
	runDemos(ctx, ui)
	printFooter(ui)
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

// printHeader prints the streaming demo header.
func printHeader(ui *adapters.PureTTY) {
	lines := []string{
		"╔══════════════════════════════════════════╗",
		"║   Spin TUI - Streaming Demo             ║",
		"╚══════════════════════════════════════════╝",
		"",
		"This demo shows how to stream chunks (like LLM tokens).",
		"Watch how the output appears word-by-word with coalescing.",
		"",
	}

	for _, line := range lines {
		_ = ui.PrintLine(line)
	}
}

// runDemos runs all four streaming demonstrations.
func runDemos(ctx context.Context, ui *adapters.PureTTY) {
	_ = ui.PrintLine("━━━ Demo 1: Word-by-word streaming ━━━")
	_ = ui.PrintLine("")
	streamWords(ctx, ui, "The quick brown fox jumps over the lazy dog.")
	_ = ui.PrintLine("")

	_ = ui.PrintLine("━━━ Demo 2: Character-by-character streaming ━━━")
	_ = ui.PrintLine("")
	streamChars(ctx, ui, "Hello, world! This is character-level streaming.")
	_ = ui.PrintLine("")

	_ = ui.PrintLine("━━━ Demo 3: Simulated LLM response ━━━")
	_ = ui.PrintLine("")
	_ = ui.PrintLine("User: Write a haiku about coding")
	_ = ui.PrintLine("")
	streamLLM(ctx, ui)
	_ = ui.PrintLine("")

	_ = ui.PrintLine("━━━ Demo 4: Fast streaming (1000 chunks, shows coalescing) ━━━")
	_ = ui.PrintLine("")
	streamFast(ctx, ui)
	_ = ui.PrintLine("")
}

// printFooter prints the closing instructions.
func printFooter(ui *adapters.PureTTY) {
	lines := []string{
		"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━",
		"",
		"Streaming demo complete!",
		"Press Ctrl-D or Ctrl-C to exit, or type 'quit'",
		"",
	}

	for _, line := range lines {
		_ = ui.PrintLine(line)
	}
}

// runInputLoop handles user input until the context is cancelled.
func runInputLoop(ctx context.Context, ui *adapters.PureTTY, cancel context.CancelFunc) {
	for {
		select {
		case <-ctx.Done():
			_ = ui.Stop()
			_, _ = fmt.Fprintln(os.Stdout, "\nGoodbye!")

			return

		case line, ok := <-ui.RequestInput():
			if !ok {
				_ = ui.Stop()

				return
			}

			if line == "quit" || line == "exit" {
				cancel()
			} else if line != "" {
				_ = ui.PrintLine(fmt.Sprintf("Echo: %s", line))
			}
		}
	}
}

// streamWords streams text word by word.
func streamWords(ctx context.Context, ui *adapters.PureTTY, text string) {
	chunks := make(chan string, 100)

	go func() {
		defer close(chunks)

		words := strings.Fields(text)
		for i, word := range words {
			select {
			case <-ctx.Done():
				return
			case chunks <- word:
				if i < len(words)-1 {
					chunks <- " "
				}

				time.Sleep(100 * time.Millisecond) // Simulate token arrival.
			}
		}

		chunks <- "\n"
	}()

	err := ui.PrintChunks(ctx, chunks)
	if err != nil && !errors.Is(err, context.Canceled) {
		_ = ui.PrintLine(fmt.Sprintf("Streaming error: %v", err))
	}
}

// streamChars streams text character by character.
func streamChars(ctx context.Context, ui *adapters.PureTTY, text string) {
	chunks := make(chan string, 100)

	go func() {
		defer close(chunks)

		for _, ch := range text {
			select {
			case <-ctx.Done():
				return
			case chunks <- string(ch):
				time.Sleep(20 * time.Millisecond)
			}
		}

		chunks <- "\n"
	}()

	err := ui.PrintChunks(ctx, chunks)
	if err != nil && !errors.Is(err, context.Canceled) {
		_ = ui.PrintLine(fmt.Sprintf("Streaming error: %v", err))
	}
}

// streamLLM simulates an LLM response with variable token timing.
func streamLLM(ctx context.Context, ui *adapters.PureTTY) {
	chunks := make(chan string, 100)

	// Simulated haiku response.
	response := []string{
		"Code", " flows", " like", " a", " stream,\n",
		"Bugs", " hide", " in", " shadows", " deep,\n",
		"Tests", " bring", " peace", " of", " mind", ".\n",
	}

	go func() {
		defer close(chunks)

		_ = ui.SetStatus("generating...") // Show status during generation.

		for _, token := range response {
			select {
			case <-ctx.Done():
				return
			case chunks <- token:
				// Variable delay to simulate realistic LLM token timing.
				if strings.HasSuffix(token, "\n") {
					time.Sleep(200 * time.Millisecond) // Pause at line breaks.
				} else if token == " " {
					time.Sleep(30 * time.Millisecond) // Short pause for spaces.
				} else {
					time.Sleep(60 * time.Millisecond) // Normal token delay.
				}
			}
		}

		_ = ui.SetStatus("") // Clear status when done.
	}()

	err := ui.PrintChunks(ctx, chunks)
	if err != nil && !errors.Is(err, context.Canceled) {
		_ = ui.PrintLine(fmt.Sprintf("Streaming error: %v", err))
	}
}

// streamFast demonstrates coalescing with many rapid chunks.
func streamFast(ctx context.Context, ui *adapters.PureTTY) {
	chunks := make(chan string, 1000)

	go func() {
		defer close(chunks)

		for i := range 1000 {
			select {
			case <-ctx.Done():
				return
			case chunks <- fmt.Sprintf("%d ", i):
				// No delay - chunks arrive as fast as possible
				// Printer will coalesce them to reduce write syscalls.
			}
		}

		chunks <- "\n"
	}()

	start := time.Now()

	err := ui.PrintChunks(ctx, chunks)
	if err != nil && !errors.Is(err, context.Canceled) {
		_ = ui.PrintLine(fmt.Sprintf("Streaming error: %v", err))
	}

	elapsed := time.Since(start)

	_ = ui.PrintLine(fmt.Sprintf("Streamed 1000 chunks in %v (throughput: %.0f chunks/sec)",
		elapsed, 1000.0/elapsed.Seconds()))
}
