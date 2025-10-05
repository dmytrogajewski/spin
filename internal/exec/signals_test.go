package exec

import (
	"context"
	"os"
	"syscall"
	"testing"
	"time"
)

func TestSignalHandlerSIGINT(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handler
	sigChan := make(chan os.Signal, 1)
	done := make(chan bool, 1)

	go func() {
		handleSignals(ctx, cancel, sigChan)
		done <- true
	}()

	// Send SIGINT
	sigChan <- syscall.SIGINT

	// Wait for context cancellation
	select {
	case <-ctx.Done():
		// Success - context was cancelled
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Context not cancelled after SIGINT")
	}

	// Cleanup
	close(sigChan)
	<-done
}

func TestSignalHandlerSIGTERM(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	done := make(chan bool, 1)

	go func() {
		handleSignals(ctx, cancel, sigChan)
		done <- true
	}()

	// Send SIGTERM
	sigChan <- syscall.SIGTERM

	select {
	case <-ctx.Done():
		// Success
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Context not cancelled after SIGTERM")
	}

	close(sigChan)
	<-done
}

func TestSignalHandlerContextAlreadyCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	sigChan := make(chan os.Signal, 1)
	done := make(chan bool, 1)

	go func() {
		handleSignals(ctx, cancel, sigChan)
		done <- true
	}()

	// Signal handler should exit immediately since context is done
	select {
	case <-done:
		// Success
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Signal handler didn't exit for cancelled context")
	}

	close(sigChan)
}

func TestSetupSignalHandler(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// This should not panic
	sigChan := setupSignalHandler(ctx, cancel)
	if sigChan == nil {
		t.Fatal("setupSignalHandler returned nil channel")
	}

	// Send a signal
	sigChan <- syscall.SIGINT

	// Verify context cancellation
	select {
	case <-ctx.Done():
		// Success
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Context not cancelled via setupSignalHandler")
	}
}
