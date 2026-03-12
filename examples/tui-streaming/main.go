package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

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

	// Handle OS signals.
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		cancel()
	}()

	// Start TUI.
	go func() {
		err := ui.Run(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
			cancel()
		}
	}()

	// Print header.
	ui.PrintLine("╔══════════════════════════════════════════╗")
	ui.PrintLine("║   Spin TUI - Streaming Demo             ║")
	ui.PrintLine("╚══════════════════════════════════════════╝")
	ui.PrintLine("")
	ui.PrintLine("This demo shows how to stream chunks (like LLM tokens).")
	ui.PrintLine("Watch how the output appears word-by-word with coalescing.")
	ui.PrintLine("")

	// Demo 1: Word-by-word streaming.
	ui.PrintLine("━━━ Demo 1: Word-by-word streaming ━━━")
	ui.PrintLine("")
	streamWords(ctx, ui, "The quick brown fox jumps over the lazy dog.")
	ui.PrintLine("")

	// Demo 2: Character-by-character streaming.
	ui.PrintLine("━━━ Demo 2: Character-by-character streaming ━━━")
	ui.PrintLine("")
	streamChars(ctx, ui, "Hello, world! This is character-level streaming.")
	ui.PrintLine("")

	// Demo 3: Simulated LLM response.
	ui.PrintLine("━━━ Demo 3: Simulated LLM response ━━━")
	ui.PrintLine("")
	ui.PrintLine("User: Write a haiku about coding")
	ui.PrintLine("")
	streamLLM(ctx, ui)
	ui.PrintLine("")

	// Demo 4: Fast streaming (shows coalescing).
	ui.PrintLine("━━━ Demo 4: Fast streaming (1000 chunks, shows coalescing) ━━━")
	ui.PrintLine("")
	streamFast(ctx, ui)
	ui.PrintLine("")

	// Done.
	ui.PrintLine("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	ui.PrintLine("")
	ui.PrintLine("Streaming demo complete!")
	ui.PrintLine("Press Ctrl-D or Ctrl-C to exit, or type 'quit'")
	ui.PrintLine("")

	// Wait for exit.
	for {
		select {
		case <-ctx.Done():
			ui.Stop()
			fmt.Println("\nGoodbye!")

			return

		case line, ok := <-ui.RequestInput():
			if !ok {
				ui.Stop()

				return
			}

			if line == "quit" || line == "exit" {
				cancel()
			} else if line != "" {
				ui.PrintLine(fmt.Sprintf("Echo: %s", line))
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
	if err != nil && err != context.Canceled {
		ui.PrintLine(fmt.Sprintf("Streaming error: %v", err))
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
	if err != nil && err != context.Canceled {
		ui.PrintLine(fmt.Sprintf("Streaming error: %v", err))
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

		ui.SetStatus("generating...") // Show status during generation.

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

		ui.SetStatus("") // Clear status when done.
	}()

	err := ui.PrintChunks(ctx, chunks)
	if err != nil && err != context.Canceled {
		ui.PrintLine(fmt.Sprintf("Streaming error: %v", err))
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
	if err != nil && err != context.Canceled {
		ui.PrintLine(fmt.Sprintf("Streaming error: %v", err))
	}

	elapsed := time.Since(start)

	ui.PrintLine(fmt.Sprintf("Streamed 1000 chunks in %v (throughput: %.0f chunks/sec)",
		elapsed, 1000.0/elapsed.Seconds()))
}
