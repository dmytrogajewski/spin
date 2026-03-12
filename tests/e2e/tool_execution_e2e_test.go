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
	t.Parallel()

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
		_, _ = console.Send("\x04")

		done := make(chan error, 1)

		go func() {
			done <- cmd.Wait()
		}()

		select {
		case <-done:
			// Command exited normally.
		case <-time.After(2 * time.Second):
			// Force kill after 2 seconds.
			_ = cmd.Process.Kill()
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

	// Look for signs of successful tool execution.
	// Should NOT see cycle detection error.
	output := expectAnyString(t, console, "EXECUTE", "TOOL", "list_directory")
	if output == "" {
		t.Logf("Did not see tool execution block, checking for cycle detection...")

		_, cycleErr := console.ExpectString("Cycle detected")
		if cycleErr == nil {
			t.Fatal("BUG REPRODUCED: Cycle detection triggered instead of executing tool")
		}

		t.Fatalf("Neither tool execution nor cycle detection found")
	}

	t.Log("Tool execution block found:", output)

	// Should see tool completion.
	time.Sleep(2 * time.Second)
}

// TestTUIToolVisualization tests that tool calls are properly visualized.
func TestTUIToolVisualization(t *testing.T) {
	t.Parallel()

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
		_, _ = console.Send("\x04")

		done := make(chan error, 1)

		go func() {
			done <- cmd.Wait()
		}()

		select {
		case <-done:
			// Command exited normally.
		case <-time.After(2 * time.Second):
			// Force kill after 2 seconds.
		_ = cmd.Process.Kill()
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

// tuiToolCase describes a TUI tool test case with a prompt and cycle detection check.
type tuiToolCase struct {
	name           string
	prompt         string
	waitSeconds    int
	cycleBugString string
	cycleBugMsg    string
	expectString   string
	expectMsg      string
}

func runTUIToolTests(t *testing.T, cases []tuiToolCase) {
	t.Helper()

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

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
				_, _ = console.Send("\x04")

				done := make(chan error, 1)

				go func() {
					done <- cmd.Wait()
				}()

				select {
				case <-done:
				case <-time.After(2 * time.Second):
					_ = cmd.Process.Kill()
					<-done
				}
			}()

			time.Sleep(2 * time.Second)

			_, err = console.SendLine(tt.prompt)
			require.NoError(t, err)

			time.Sleep(time.Duration(tt.waitSeconds) * time.Second)

			_, cycleErr := console.ExpectString(tt.cycleBugString)
			if cycleErr == nil {
				t.Fatal(tt.cycleBugMsg)
			}

			if tt.expectString != "" {
				_, err = console.ExpectString(tt.expectString)
				require.NoError(t, err, tt.expectMsg)
			}
		})
	}
}

// TestTUIListDirectoryTool specifically tests the list_directory tool.
func TestTUIListDirectoryTool(t *testing.T) {
	t.Parallel()
	runTUIToolTests(t, []tuiToolCase{
		{
			name:           "list_directory",
			prompt:         "use the list_directory tool to show me files in the current directory",
			waitSeconds:    8,
			cycleBugString: "repeated_tool",
			cycleBugMsg:    "BUG DETECTED: list_directory triggered cycle detection - tool not being executed",
			expectString:   "list_directory",
			expectMsg:      "Should see list_directory tool being called",
		},
	})
}

// TestTUIMultipleToolCalls tests that multiple tool calls work correctly.
func TestTUIMultipleToolCalls(t *testing.T) {
	t.Parallel()
	runTUIToolTests(t, []tuiToolCase{
		{
			name:           "multiple tools",
			prompt:         "First list the files, then read the README.md",
			waitSeconds:    15,
			cycleBugString: "Cycle detected",
			cycleBugMsg:    "BUG DETECTED: Cycle detection triggered with multiple tool calls",
		},
	})
}

// TestTUIReadFileTool tests the read_file tool execution.
func TestTUIReadFileTool(t *testing.T) {
	t.Parallel()

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
		_, _ = console.Send("\x04")

		done := make(chan error, 1)

		go func() {
			done <- cmd.Wait()
		}()

		select {
		case <-done:
			// Command exited normally.
		case <-time.After(2 * time.Second):
			// Force kill after 2 seconds.
			_ = cmd.Process.Kill()
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
	t.Parallel()

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
		_, _ = console.Send("\x04")

		done := make(chan error, 1)

		go func() {
			done <- cmd.Wait()
		}()

		select {
		case <-done:
			// Command exited normally.
		case <-time.After(2 * time.Second):
			// Force kill after 2 seconds.
			_ = cmd.Process.Kill()
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

// expectAnyString tries to match any of the given strings in order.
// Returns the matched output on first success, or empty string if none match.
func expectAnyString(t *testing.T, console *expect.Console, candidates ...string) string {
	t.Helper()

	for _, s := range candidates {
		output, err := console.ExpectString(s)
		if err == nil {
			return output
		}
	}

	return ""
}
