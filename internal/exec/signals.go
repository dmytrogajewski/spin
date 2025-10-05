package exec

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

// setupSignalHandler sets up signal handling for SIGINT and SIGTERM.
// Returns the signal channel for testing purposes.
func setupSignalHandler(ctx context.Context, cancel context.CancelFunc) chan os.Signal {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go handleSignals(ctx, cancel, sigChan)

	return sigChan
}

// handleSignals handles OS signals and cancels the context.
func handleSignals(ctx context.Context, cancel context.CancelFunc, sigChan chan os.Signal) {
	select {
	case sig := <-sigChan:
		fmt.Fprintf(os.Stderr, "\nReceived signal: %v\n", sig)
		fmt.Fprintf(os.Stderr, "Cancelling execution... (press Ctrl+C again to force quit)\n")
		cancel()
	case <-ctx.Done():
		// Context already cancelled, exit
		return
	}
}

