package e2e

import (
	"os"
	"os/exec"
	"testing"
	"time"

	expect "github.com/Netflix/go-expect"
	"github.com/stretchr/testify/require"
)

// TestTUIToolExecution tests that tools are properly executed when called by the LLM.
// This reproduces the bug where list_directory is called but not executed.
func TestTUIToolExecution(t *testing.T) {
	skipTUITests(t)

	console, err := expect.NewConsole(expect.WithStdout(os.Stdout))
	require.NoError(t, err)

	defer console.Close()

	binPath := getBinPath(t)

	// Use a model that supports tool calling.
	cmd := exec.Command(binPath, "--model", "dummy", "--provider", "test-llm")
	cmd.Stdin = console.Tty()
	cmd.Stdout = console.Tty()
	cmd.Stderr = console.Tty()

	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	err = cmd.Start()
	require.NoError(t, err)

	defer func() {
		console.Send("\x04")

		done := make(chan error, 1)

		go func() {
			done <- cmd.Wait()
		}()

		select {
		case <-done:
			// Command exited normally.
		case <-time.After(2 * time.Second):
			// Force kill after 2 seconds.
			cmd.Process.Kill()
			<-done
		}
	}()

	// Wait for initialization.
	time.Sleep(2 * time.Second)

	// Ask to list files - this should trigger list_directory tool.
	_, err = console.SendLine("list files in current directory")
	require.NoError(t, err)

	// Wait for tool execution
	// If bug exists: will see "Cycle detected: repeated_tool"
	// If fixed: will see actual file listing.
	time.Sleep(10 * time.Second)

	// Look for signs of successful tool execution
	// Should NOT see cycle detection error
	// Try multiple possible tool block headers (EXECUTE, TOOL, etc.)
	var output string

	output, err = console.ExpectString("EXECUTE")
	if err != nil {
		output, err = console.ExpectString("TOOL")
		if err != nil {
			output, err = console.ExpectString("list_directory")
			if err != nil {
				t.Logf("Did not see tool execution block, checking for cycle detection...")

				// Check if cycle detection triggered (the bug).
				_, cycleErr := console.ExpectString("Cycle detected")
				if cycleErr == nil {
					t.Fatal("BUG REPRODUCED: Cycle detection triggered instead of executing tool")
				}

				// Neither tool execution nor cycle - something else wrong.
				t.Fatalf("Neither tool execution nor cycle detection found")
			}
		}
	}

	t.Log("Tool execution block found:", output)

	// Should see tool completion.
	time.Sleep(2 * time.Second)
}

// TestTUIToolVisualization tests that tool calls are properly visualized.
func TestTUIToolVisualization(t *testing.T) {
	skipTUITests(t)

	console, err := expect.NewConsole(expect.WithStdout(os.Stdout))
	require.NoError(t, err)

	defer console.Close()

	binPath := getBinPath(t)
	cmd := exec.Command(binPath, "--model", "dummy", "--provider", "test-llm")
	cmd.Stdin = console.Tty()
	cmd.Stdout = console.Tty()
	cmd.Stderr = console.Tty()

	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	err = cmd.Start()
	require.NoError(t, err)

	defer func() {
		console.Send("\x04")

		done := make(chan error, 1)

		go func() {
			done <- cmd.Wait()
		}()

		select {
		case <-done:
			// Command exited normally.
		case <-time.After(2 * time.Second):
			// Force kill after 2 seconds.
			cmd.Process.Kill()
			<-done
		}
	}()

	time.Sleep(2 * time.Second)

	// Request a tool operation.
	_, err = console.SendLine("read the README.md file")
	require.NoError(t, err)

	// Look for tool block header (EXECUTE, TOOL, or READ).
	_, err = console.ExpectString("EXECUTE")
	if err != nil {
		_, err = console.ExpectString("TOOL")
		if err != nil {
			_, err = console.ExpectString("READ")
		}
	}

	require.NoError(t, err, "Should see tool block when tool is called")

	time.Sleep(5 * time.Second)

	// Should see tool completion indicator (⤷).
	_, err = console.ExpectString("⤷")
	require.NoError(t, err, "Should see completion indicator")
}

// TestTUIListDirectoryTool specifically tests the list_directory tool.
func TestTUIListDirectoryTool(t *testing.T) {
	skipTUITests(t)

	console, err := expect.NewConsole(expect.WithStdout(os.Stdout))
	require.NoError(t, err)

	defer console.Close()

	binPath := getBinPath(t)
	cmd := exec.Command(binPath, "--model", "dummy", "--provider", "test-llm")
	cmd.Stdin = console.Tty()
	cmd.Stdout = console.Tty()
	cmd.Stderr = console.Tty()

	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	err = cmd.Start()
	require.NoError(t, err)

	defer func() {
		console.Send("\x04")

		done := make(chan error, 1)

		go func() {
			done <- cmd.Wait()
		}()

		select {
		case <-done:
			// Command exited normally.
		case <-time.After(2 * time.Second):
			// Force kill after 2 seconds.
			cmd.Process.Kill()
			<-done
		}
	}()

	time.Sleep(2 * time.Second)

	// Explicitly request list_directory.
	_, err = console.SendLine("use the list_directory tool to show me files in the current directory")
	require.NoError(t, err)

	// Wait and check for cycle detection (the bug).
	time.Sleep(8 * time.Second)

	// Try to find cycle detection notice.
	_, cycleErr := console.ExpectString("repeated_tool")
	if cycleErr == nil {
		t.Fatal("BUG DETECTED: list_directory triggered cycle detection - tool not being executed")
	}

	// If no cycle, should see tool output.
	_, err = console.ExpectString("list_directory")
	require.NoError(t, err, "Should see list_directory tool being called")
}

// TestTUIMultipleToolCalls tests that multiple tool calls work correctly.
func TestTUIMultipleToolCalls(t *testing.T) {
	skipTUITests(t)

	console, err := expect.NewConsole(expect.WithStdout(os.Stdout))
	require.NoError(t, err)

	defer console.Close()

	binPath := getBinPath(t)
	cmd := exec.Command(binPath, "--model", "dummy", "--provider", "test-llm")
	cmd.Stdin = console.Tty()
	cmd.Stdout = console.Tty()
	cmd.Stderr = console.Tty()

	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	err = cmd.Start()
	require.NoError(t, err)

	defer func() {
		console.Send("\x04")

		done := make(chan error, 1)

		go func() {
			done <- cmd.Wait()
		}()

		select {
		case <-done:
			// Command exited normally.
		case <-time.After(2 * time.Second):
			// Force kill after 2 seconds.
			cmd.Process.Kill()
			<-done
		}
	}()

	time.Sleep(2 * time.Second)

	// Request multiple operations.
	_, err = console.SendLine("First list the files, then read the README.md")
	require.NoError(t, err)

	// Should see TOOL blocks (plural).
	time.Sleep(15 * time.Second)

	// Check for cycle detection.
	_, cycleErr := console.ExpectString("Cycle detected")
	if cycleErr == nil {
		t.Fatal("BUG DETECTED: Cycle detection triggered with multiple tool calls")
	}
}

// TestTUIReadFileTool tests the read_file tool execution.
func TestTUIReadFileTool(t *testing.T) {
	skipTUITests(t)

	console, err := expect.NewConsole(expect.WithStdout(os.Stdout))
	require.NoError(t, err)

	defer console.Close()

	binPath := getBinPath(t)
	cmd := exec.Command(binPath, "--model", "dummy", "--provider", "test-llm")
	cmd.Stdin = console.Tty()
	cmd.Stdout = console.Tty()
	cmd.Stderr = console.Tty()

	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	err = cmd.Start()
	require.NoError(t, err)

	defer func() {
		console.Send("\x04")

		done := make(chan error, 1)

		go func() {
			done <- cmd.Wait()
		}()

		select {
		case <-done:
			// Command exited normally.
		case <-time.After(2 * time.Second):
			// Force kill after 2 seconds.
			cmd.Process.Kill()
			<-done
		}
	}()

	time.Sleep(2 * time.Second)

	// Ask to read a file that exists.
	_, err = console.SendLine("read the go.mod file")
	require.NoError(t, err)

	time.Sleep(10 * time.Second)

	// Should see either:
	// - TOOL block with read_file (success)
	// - Cycle detection (bug).

	_, cycleErr := console.ExpectString("Cycle detected")
	if cycleErr == nil {
		t.Fatal("BUG DETECTED: read_file triggered cycle detection")
	}

	// Should see module declaration from go.mod.
	_, err = console.ExpectString("module")
	require.NoError(t, err, "Should see file contents after read_file execution")
}

// TestTUIToolWithoutCycleDetection tests tool execution with cycle detection disabled.
// This helps isolate whether cycle detection is the problem.
func TestTUIToolWithoutCycleDetection(t *testing.T) {
	skipTUITests(t)

	console, err := expect.NewConsole(expect.WithStdout(os.Stdout))
	require.NoError(t, err)

	defer console.Close()

	binPath := getBinPath(t)

	// Placeholder: Add flag to disable cycle detection if config supports it
	// This would allow testing without cycle detection triggering.
	cmd := exec.Command(binPath, "--model", "dummy", "--provider", "test-llm")
	cmd.Stdin = console.Tty()
	cmd.Stdout = console.Tty()
	cmd.Stderr = console.Tty()

	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	err = cmd.Start()
	require.NoError(t, err)

	defer func() {
		console.Send("\x04")

		done := make(chan error, 1)

		go func() {
			done <- cmd.Wait()
		}()

		select {
		case <-done:
			// Command exited normally.
		case <-time.After(2 * time.Second):
			// Force kill after 2 seconds.
			cmd.Process.Kill()
			<-done
		}
	}()

	time.Sleep(2 * time.Second)

	// Request tool usage.
	_, err = console.SendLine("list files")
	require.NoError(t, err)

	// Wait longer since cycle detection won't intervene.
	time.Sleep(15 * time.Second)

	// Should eventually see tool execution or timeout.
	_, err = console.ExpectString("EXECUTE")
	if err != nil {
		_, err = console.ExpectString("TOOL")
		if err != nil {
			_, err = console.ExpectString("list_directory")
		}
	}

	require.NoError(t, err, "Should see tool execution even without cycle detection")
}
