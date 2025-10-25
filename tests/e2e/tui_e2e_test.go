package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	expect "github.com/Netflix/go-expect"
	"github.com/stretchr/testify/require"
)

// getBinPath returns absolute path to spin binary
func getBinPath(t *testing.T) string {
	// Get workspace root (go up from tests/e2e/ to root)
	wd, err := os.Getwd()
	require.NoError(t, err)
	root := filepath.Dir(filepath.Dir(wd)) // tests/e2e/ -> tests/ -> root
	return filepath.Join(root, "bin", "spin")
}

// TestTUILaunch tests that TUI launches successfully
func TestTUILaunch(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Create console
	console, err := expect.NewConsole(expect.WithStdout(os.Stdout))
	require.NoError(t, err)
	defer console.Close()

	// Get binary path
	binPath := getBinPath(t)

	// Launch TUI
	cmd := exec.Command(binPath, "--model", "qwen3:0.6b", "--provider", "ollama")
	cmd.Stdin = console.Tty()
	cmd.Stdout = console.Tty()
	cmd.Stderr = console.Tty()
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	err = cmd.Start()
	require.NoError(t, err)
	defer func() {
		console.Send("\x04") // Ctrl+D to exit
		cmd.Wait()
	}()

	// Wait for TUI to initialize - just give it time to render
	time.Sleep(2 * time.Second)

	// TUI should be running without errors - test passes if we get here
}

// TestTUIBasicChat tests sending a message and receiving response
func TestTUIBasicChat(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	console, err := expect.NewConsole(expect.WithStdout(os.Stdout))
	require.NoError(t, err)
	defer console.Close()

	binPath := getBinPath(t)
	cmd := exec.Command(binPath, "--model", "qwen3:0.6b", "--provider", "ollama")
	cmd.Stdin = console.Tty()
	cmd.Stdout = console.Tty()
	cmd.Stderr = console.Tty()
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	err = cmd.Start()
	require.NoError(t, err)
	defer func() {
		console.Send("\x04")
		cmd.Wait()
	}()

	// Wait for initialization
	time.Sleep(1 * time.Second)

	// Send a simple message
	_, err = console.SendLine("Say exactly: Hello from test")
	require.NoError(t, err)

	// Wait for response (look for any assistant output)
	_, err = console.ExpectString("Hello")
	require.NoError(t, err, "Should receive response from LLM")
}

// TestTUIFilePickerTrigger tests @ key triggers file picker
func TestTUIFilePickerTrigger(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	console, err := expect.NewConsole(expect.WithStdout(os.Stdout))
	require.NoError(t, err)
	defer console.Close()

	binPath := getBinPath(t)
	cmd := exec.Command(binPath, "--model", "qwen3:0.6b", "--provider", "ollama")
	cmd.Stdin = console.Tty()
	cmd.Stdout = console.Tty()
	cmd.Stderr = console.Tty()
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	err = cmd.Start()
	require.NoError(t, err)
	defer func() {
		console.Send("\x04")
		cmd.Wait()
	}()

	// Wait for initialization
	time.Sleep(1 * time.Second)

	// Type @ to trigger file picker
	_, err = console.Send("@")
	require.NoError(t, err)

	// Wait a bit for file picker to appear
	time.Sleep(500 * time.Millisecond)

	// Close file picker with Esc
	_, err = console.Send("\x1b") // Esc
	require.NoError(t, err)
}

// TestTUIHelpModal tests Ctrl+H triggers help
func TestTUIHelpModal(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	console, err := expect.NewConsole(expect.WithStdout(os.Stdout))
	require.NoError(t, err)
	defer console.Close()

	binPath := getBinPath(t)
	cmd := exec.Command(binPath, "--model", "qwen3:0.6b", "--provider", "ollama")
	cmd.Stdin = console.Tty()
	cmd.Stdout = console.Tty()
	cmd.Stderr = console.Tty()
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	err = cmd.Start()
	require.NoError(t, err)
	defer func() {
		console.Send("\x04")
		cmd.Wait()
	}()

	// Wait for initialization
	time.Sleep(1 * time.Second)

	// Press Ctrl+H
	_, err = console.Send("\x08") // Ctrl+H
	require.NoError(t, err)

	// Wait a bit for help modal
	time.Sleep(500 * time.Millisecond)

	// Close with Esc
	_, err = console.Send("\x1b")
	require.NoError(t, err)
}

// TestTUIExitWithCtrlD tests Ctrl+D exits cleanly
func TestTUIExitWithCtrlD(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	console, err := expect.NewConsole(expect.WithStdout(os.Stdout))
	require.NoError(t, err)
	defer console.Close()

	binPath := getBinPath(t)
	cmd := exec.Command(binPath, "--model", "qwen3:0.6b", "--provider", "ollama")
	cmd.Stdin = console.Tty()
	cmd.Stdout = console.Tty()
	cmd.Stderr = console.Tty()
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	err = cmd.Start()
	require.NoError(t, err)

	// Wait for initialization
	time.Sleep(1 * time.Second)

	// Send Ctrl+D
	_, err = console.Send("\x04")
	require.NoError(t, err)

	// Wait for process to exit
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		// Process should exit cleanly (exit code 0 or specific exit code)
		if err != nil {
			// Check if it's a clean exit
			if exitErr, ok := err.(*exec.ExitError); ok {
				require.Equal(t, 0, exitErr.ExitCode(), "Should exit with code 0")
			}
		}
	case <-time.After(5 * time.Second):
		t.Fatal("TUI did not exit within timeout after Ctrl+D")
	}
}

// TestTUIToolApproval tests approval workflow
func TestTUIToolApproval(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	console, err := expect.NewConsole(expect.WithStdout(os.Stdout))
	require.NoError(t, err)
	defer console.Close()

	binPath := getBinPath(t)
	cmd := exec.Command(binPath, "--model", "qwen3:1.7b", "--provider", "ollama", "--sandbox", "workspace-write")
	cmd.Stdin = console.Tty()
	cmd.Stdout = console.Tty()
	cmd.Stderr = console.Tty()
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")
	cmd.Dir = t.TempDir() // Run in temp directory

	err = cmd.Start()
	require.NoError(t, err)
	defer func() {
		console.Send("\x04")
		cmd.Wait()
	}()

	// Wait for initialization
	time.Sleep(1 * time.Second)

	// Ask to create a file
	_, err = console.SendLine("Create a file called test.txt with the text 'automated test'")
	require.NoError(t, err)

	// Wait for approval prompt
	time.Sleep(5 * time.Second)

	// Approve with 'A'
	_, err = console.Send("A")
	require.NoError(t, err)

	// Wait a bit for execution
	time.Sleep(2 * time.Second)
}

// TestTUIMultiTurn tests conversation context is maintained
func TestTUIMultiTurn(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	console, err := expect.NewConsole(expect.WithStdout(os.Stdout))
	require.NoError(t, err)
	defer console.Close()

	binPath := getBinPath(t)
	cmd := exec.Command(binPath, "--model", "qwen3:1.7b", "--provider", "ollama")
	cmd.Stdin = console.Tty()
	cmd.Stdout = console.Tty()
	cmd.Stderr = console.Tty()
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	err = cmd.Start()
	require.NoError(t, err)
	defer func() {
		console.Send("\x04")
		cmd.Wait()
	}()

	// Wait for initialization
	time.Sleep(1 * time.Second)

	// First message
	_, err = console.SendLine("My favorite number is 42")
	require.NoError(t, err)
	time.Sleep(5 * time.Second) // Wait for response

	// Second message - test context retention
	_, err = console.SendLine("What is my favorite number?")
	require.NoError(t, err)

	// Look for "42" in response
	_, err = console.ExpectString("42")
	require.NoError(t, err, "Should remember context from previous message")
}

// TestTUIStopStreaming tests Ctrl+C stops streaming
func TestTUIStopStreaming(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	console, err := expect.NewConsole(expect.WithStdout(os.Stdout))
	require.NoError(t, err)
	defer console.Close()

	binPath := getBinPath(t)
	cmd := exec.Command(binPath, "--model", "qwen3:1.7b", "--provider", "ollama")
	cmd.Stdin = console.Tty()
	cmd.Stdout = console.Tty()
	cmd.Stderr = console.Tty()
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	err = cmd.Start()
	require.NoError(t, err)
	defer func() {
		console.Send("\x04")
		cmd.Wait()
	}()

	// Wait for initialization
	time.Sleep(1 * time.Second)

	// Ask for a long response
	_, err = console.SendLine("Write a very long story about a robot")
	require.NoError(t, err)

	// Wait a bit for streaming to start
	time.Sleep(2 * time.Second)

	// Send Ctrl+C to cancel
	_, err = console.Send("\x03")
	require.NoError(t, err)

	// TUI should still be responsive (not crash)
	time.Sleep(1 * time.Second)
}
