// Package main provides a TUI streaming example.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/dmytrogajewski/spin/internal/ui/adapters"
)

// check panics if err is non-nil. Used in example/demo programs only.
func check(err error) {
	if err != nil {
		log.Fatalf("TUI error: %v", err)
	}
}

const (
	tokenArrivalDelay = 100 * time.Millisecond
	fastTokenDelay    = 20 * time.Millisecond
	lineBreakPause    = 200 * time.Millisecond
	spacePause        = 30 * time.Millisecond
	normalTokenDelay  = 60 * time.Millisecond
	msPerSecondFloat  = 1000.0
	chunkBufferSize   = 100
	fastChunkCount    = 1000
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
		check(ui.PrintLine(line))
	}
}

// runDemos runs all four streaming demonstrations.
func runDemos(ctx context.Context, ui *adapters.PureTTY) {
	check(ui.PrintLine("━━━ Demo 1: Word-by-word streaming ━━━"))
	check(ui.PrintLine(""))
	streamWords(ctx, ui, "The quick brown fox jumps over the lazy dog.")
	check(ui.PrintLine(""))

	check(ui.PrintLine("━━━ Demo 2: Character-by-character streaming ━━━"))
	check(ui.PrintLine(""))
	streamChars(ctx, ui, "Hello, world! This is character-level streaming.")
	check(ui.PrintLine(""))

	check(ui.PrintLine("━━━ Demo 3: Simulated LLM response ━━━"))
	check(ui.PrintLine(""))
	check(ui.PrintLine("User: Write a haiku about coding"))
	check(ui.PrintLine(""))
	streamLLM(ctx, ui)
	check(ui.PrintLine(""))

	check(ui.PrintLine("━━━ Demo 4: Fast streaming (1000 chunks, shows coalescing) ━━━"))
	check(ui.PrintLine(""))
	streamFast(ctx, ui)
	check(ui.PrintLine(""))
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
		check(ui.PrintLine(line))
	}
}

// runInputLoop handles user input until the context is canceled.
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

			if line == "quit" || line == "exit" {
				cancel()
			} else if line != "" {
				check(ui.PrintLine(fmt.Sprintf("Echo: %s", line)))
			}
		}
	}
}

// streamWords streams text word by word.
func streamWords(ctx context.Context, ui *adapters.PureTTY, text string) {
	chunks := make(chan string, chunkBufferSize)

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

				time.Sleep(tokenArrivalDelay) // Simulate token arrival.
			}
		}

		chunks <- "\n"
	}()

	err := ui.PrintChunks(ctx, chunks)
	if err != nil && !errors.Is(err, context.Canceled) {
		check(ui.PrintLine(fmt.Sprintf("Streaming error: %v", err)))
	}
}

// streamChars streams text character by character.
func streamChars(ctx context.Context, ui *adapters.PureTTY, text string) {
	chunks := make(chan string, chunkBufferSize)

	go func() {
		defer close(chunks)

		for _, ch := range text {
			select {
			case <-ctx.Done():
				return
			case chunks <- string(ch):
				time.Sleep(fastTokenDelay)
			}
		}

		chunks <- "\n"
	}()

	err := ui.PrintChunks(ctx, chunks)
	if err != nil && !errors.Is(err, context.Canceled) {
		check(ui.PrintLine(fmt.Sprintf("Streaming error: %v", err)))
	}
}

// streamLLM simulates an LLM response with variable token timing.
func streamLLM(ctx context.Context, ui *adapters.PureTTY) {
	chunks := make(chan string, chunkBufferSize)

	// Simulated haiku response.
	response := []string{
		"Code", " flows", " like", " a", " stream,\n",
		"Bugs", " hide", " in", " shadows", " deep,\n",
		"Tests", " bring", " peace", " of", " mind", ".\n",
	}

	go func() {
		defer close(chunks)

		check(ui.SetStatus("generating...")) // Show status during generation.

		for _, token := range response {
			select {
			case <-ctx.Done():
				return
			case chunks <- token:
				// Variable delay to simulate realistic LLM token timing.
				switch {
				case strings.HasSuffix(token, "\n"):
					time.Sleep(lineBreakPause) // Pause at line breaks.
				case token == " ":
					time.Sleep(spacePause) // Short pause for spaces.
				default:
					time.Sleep(normalTokenDelay) // Normal token delay.
				}
			}
		}

		check(ui.SetStatus("")) // Clear status when done.
	}()

	err := ui.PrintChunks(ctx, chunks)
	if err != nil && !errors.Is(err, context.Canceled) {
		check(ui.PrintLine(fmt.Sprintf("Streaming error: %v", err)))
	}
}

// streamFast demonstrates coalescing with many rapid chunks.
func streamFast(ctx context.Context, ui *adapters.PureTTY) {
	chunks := make(chan string, fastChunkCount)

	go func() {
		defer close(chunks)

		for i := range fastChunkCount {
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
		check(ui.PrintLine(fmt.Sprintf("Streaming error: %v", err)))
	}

	elapsed := time.Since(start)

	check(ui.PrintLine(fmt.Sprintf("Streamed 1000 chunks in %v (throughput: %.0f chunks/sec)",
		elapsed, msPerSecondFloat/elapsed.Seconds())))
}
