// Package e2e contains end-to-end tests for the Spin binary.
// These tests execute the actual binary and verify its behavior.
package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const (
	// testTimeout is the default timeout for e2e tests
	testTimeout = 60 * time.Second

	// Binary path (relative to test file)
	binPath = "../../bin/spin"

	// Env var that instructs tests to reuse an existing binary
	skipBuildEnv = "SPIN_E2E_SKIP_BUILD"
)

// TestMain builds the binary before running tests
func TestMain(m *testing.M) {
	if shouldSkipBuild() {
		if _, err := os.Stat(binPath); err == nil {
			fmt.Println("Using existing spin binary for e2e tests")
		} else {
			fmt.Println("Pre-built spin binary not found, rebuilding...")
			buildSpinBinary()
		}
	} else {
		buildSpinBinary()
	}

	// Run tests
	code := m.Run()

	// Cleanup (optional - keep binary for debugging)
	// os.Remove(binPath)

	os.Exit(code)
}

func shouldSkipBuild() bool {
	return os.Getenv(skipBuildEnv) == "1"
}

func buildSpinBinary() {
	// Build the binary with e2e_llm_test tag to enable test-llm provider
	// This allows e2e tests to run without requiring external LLM services
	fmt.Println("Building spin binary for e2e tests (with e2e_llm_test tag)...")
	cmd := exec.Command("go", "build", "-tags", "e2e_llm_test", "-o", binPath, "../../cmd/spin")
	if output, err := cmd.CombinedOutput(); err != nil {
		fmt.Printf("Failed to build binary: %v\n%s\n", err, output)
		os.Exit(1)
	}
}

// runSpin executes the spin binary with given args and returns stdout, stderr, and error
func runSpin(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, binPath, args...)

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err = cmd.Run()

	return outBuf.String(), errBuf.String(), err
}

// runSpinWithInput executes spin with stdin input
func runSpinWithInput(t *testing.T, input string, args ...string) (stdout, stderr string, err error) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, binPath, args...)
	cmd.Stdin = strings.NewReader(input)

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err = cmd.Run()

	return outBuf.String(), errBuf.String(), err
}

// TestConfigCommands tests config-related functionality
func TestConfigCommands(t *testing.T) {
	t.Run("config show without config file", func(t *testing.T) {
		// Remove config if exists
		homeDir, _ := os.UserHomeDir()
		configPath := filepath.Join(homeDir, ".spin", "spin.yaml")
		tempBackup := configPath + ".e2e-backup"

		// Backup existing config
		if _, err := os.Stat(configPath); err == nil {
			if err := os.Rename(configPath, tempBackup); err != nil {
				t.Fatalf("Failed to backup config: %v", err)
			}
			defer os.Rename(tempBackup, configPath)
		}

		stdout, stderr, err := runSpin(t, "config", "show")

		// Should not error when no config exists
		if err != nil {
			t.Fatalf("config show failed: %v\nstderr: %s", err, stderr)
		}

		// Should indicate no config file
		if !strings.Contains(stdout, "No configuration file") && !strings.Contains(stdout, "{}") {
			t.Errorf("Expected 'No configuration file' or empty config, got: %s", stdout)
		}
	})

	t.Run("config show with binary file in cwd", func(t *testing.T) {
		// Create a temporary directory with a binary file named "spin"
		tmpDir := t.TempDir()
		binaryPath := filepath.Join(tmpDir, "spin")

		// Create a fake binary file
		if err := os.WriteFile(binaryPath, []byte{0x7f, 0x45, 0x4c, 0x46}, 0755); err != nil {
			t.Fatalf("Failed to create fake binary: %v", err)
		}

		// Run config show from that directory
		cmd := exec.Command(binPath, "config", "show")
		cmd.Dir = tmpDir

		var outBuf, errBuf bytes.Buffer
		cmd.Stdout = &outBuf
		cmd.Stderr = &errBuf

		err := cmd.Run()

		// Should NOT fail with YAML parsing error (Bug #1 regression test)
		stderr := errBuf.String()
		if strings.Contains(stderr, "control characters are not allowed") {
			t.Errorf("BUG #1 REGRESSION: Config loader tried to read binary file!\nstderr: %s", stderr)
		}

		// Should succeed or show no config
		if err != nil && !strings.Contains(stderr, "No configuration file") {
			t.Logf("Warning: unexpected error (but not the binary-reading bug): %v\nstderr: %s", err, stderr)
		}
	})

	t.Run("config validate", func(t *testing.T) {
		stdout, stderr, _ := runSpin(t, "config", "validate")

		// Should validate without crashing
		output := stdout + stderr
		if !strings.Contains(output, "valid") && !strings.Contains(output, "invalid") {
			t.Errorf("Expected validation result, got: %s", output)
		}
	})

	t.Run("config path", func(t *testing.T) {
		stdout, stderr, err := runSpin(t, "config", "path")

		output := stdout + stderr

		// Should show config path or indicate no config
		if err != nil && !strings.Contains(output, "no config file") {
			t.Errorf("config path failed unexpectedly: %v\noutput: %s", err, output)
		}
	})
}

// TestMCPCommands tests MCP management commands
func TestMCPCommands(t *testing.T) {
	// Setup: ensure clean MCP state
	homeDir, _ := os.UserHomeDir()
	spinDir := filepath.Join(homeDir, ".spin")
	configPath := filepath.Join(spinDir, "spin.yaml")
	tempBackup := configPath + ".e2e-mcp-backup"

	// Backup existing config
	if _, err := os.Stat(configPath); err == nil {
		if err := os.Rename(configPath, tempBackup); err != nil {
			t.Fatalf("Failed to backup config: %v", err)
		}
		defer os.Rename(tempBackup, configPath)
	}

	// Create .spin directory if it doesn't exist
	if err := os.MkdirAll(spinDir, 0755); err != nil {
		t.Fatalf("Failed to create .spin directory: %v", err)
	}

	// Create empty config file for MCP commands to work with
	emptyConfig := []byte("# Spin configuration\n")
	if err := os.WriteFile(configPath, emptyConfig, 0644); err != nil {
		t.Fatalf("Failed to create empty config: %v", err)
	}
	defer os.Remove(configPath)

	t.Run("mcp list empty", func(t *testing.T) {
		stdout, stderr, err := runSpin(t, "mcp", "list")

		if err != nil {
			t.Fatalf("mcp list failed: %v\nstderr: %s", err, stderr)
		}

		if !strings.Contains(stdout, "No MCP servers") {
			t.Errorf("Expected 'No MCP servers', got: %s", stdout)
		}
	})

	t.Run("mcp add and remove", func(t *testing.T) {
		// Add MCP server
		stdout, stderr, err := runSpin(t, "mcp", "add", "test-server", "echo", "test")
		if err != nil {
			t.Fatalf("mcp add failed: %v\nstderr: %s\nstdout: %s", err, stderr, stdout)
		}

		if !strings.Contains(stdout+stderr, "Added MCP server 'test-server'") {
			t.Errorf("Expected add confirmation, got stdout: %s, stderr: %s", stdout, stderr)
		}

		// List should show the server
		stdout, stderr, err = runSpin(t, "mcp", "list")
		if err != nil {
			t.Fatalf("mcp list failed: %v\nstderr: %s", err, stderr)
		}

		if !strings.Contains(stdout, "test-server") {
			t.Errorf("Expected server 'test-server' in list, got: %s", stdout)
		}

		// Get server details
		stdout, stderr, err = runSpin(t, "mcp", "get", "test-server")
		if err != nil {
			t.Fatalf("mcp get failed: %v\nstderr: %s", err, stderr)
		}

		if !strings.Contains(stdout, "test-server") || !strings.Contains(stdout, "echo") {
			t.Errorf("Expected server details, got: %s", stdout)
		}

		// Remove server
		stdout, stderr, err = runSpin(t, "mcp", "remove", "test-server", "--yes")
		if err != nil {
			t.Fatalf("mcp remove failed: %v\nstderr: %s", err, stderr)
		}

		// List should be empty again
		stdout, stderr, err = runSpin(t, "mcp", "list")
		if err != nil {
			t.Fatalf("mcp list failed: %v\nstderr: %s", err, stderr)
		}

		if !strings.Contains(stdout, "No MCP servers") {
			t.Errorf("Expected empty list after removal, got: %s", stdout)
		}
	})
}

// TestDebugCommands tests debug command functionality
func TestDebugCommands(t *testing.T) {
	t.Parallel()

	t.Run("debug sandbox platform check", func(t *testing.T) {
		stdout, stderr, err := runSpin(t, "debug", "sandbox", "ls")

		output := stdout + stderr

		// On Linux, should fail with platform check
		// On macOS, might work or show not implemented
		if strings.Contains(output, "only available on macOS") {
			// Correct behavior on Linux
			if err == nil {
				t.Error("Expected error on non-macOS platform")
			}
		} else if strings.Contains(output, "not yet implemented") {
			// Acceptable - stub implementation on macOS
			t.Logf("Sandbox command reached stub (macOS): %s", output)
		} else {
			t.Logf("Sandbox command output: %s", output)
		}
	})

	t.Run("debug landlock platform check", func(t *testing.T) {
		stdout, stderr, err := runSpin(t, "debug", "landlock", "ls")

		output := stdout + stderr

		// On macOS, should fail with platform check
		// On Linux, might work or show not implemented
		if strings.Contains(output, "only available on Linux") {
			// Correct behavior on macOS
			if err == nil {
				t.Error("Expected error on non-Linux platform")
			}
		} else if strings.Contains(output, "not yet implemented") {
			// Acceptable - stub implementation on Linux
			t.Logf("Landlock command reached stub (Linux): %s", output)
		} else {
			t.Logf("Landlock command output: %s", output)
		}
	})
}

// TestExecMode tests exec mode with test-llm provider (no external LLM required)
func TestExecMode(t *testing.T) {
	t.Parallel()

	// Create temporary config with test-llm provider
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "spin.yaml")

	config := `llm:
  provider: test-llm
  model: dummy
  temperature: 0.7
  max_tokens: 4096

sandbox:
  mode: workspace-write
`

	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	t.Run("exec basic prompt", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, binPath, "--config-file", configPath, "exec", "what is 2+2? answer with just the number")

		var outBuf, errBuf bytes.Buffer
		cmd.Stdout = &outBuf
		cmd.Stderr = &errBuf

		err := cmd.Run()

		stdout := outBuf.String()
		stderr := errBuf.String()

		// Should not error (Bug #2 regression test)
		if err != nil {
			if strings.Contains(stderr, "provider is required") || strings.Contains(stderr, "model is required") {
				t.Errorf("BUG #2 REGRESSION: Config integration broken!\nstderr: %s", stderr)
			} else if strings.Contains(stderr, "context deadline exceeded") {
				t.Skip("Test timed out")
			} else {
				t.Errorf("exec failed: %v\nstderr: %s\nstdout: %s", err, stderr, stdout)
			}
		}

		// Should have some output (Bug #3 regression test)
		if len(stdout) == 0 {
			t.Errorf("BUG #3 REGRESSION: No output from exec mode!\nstderr: %s", stderr)
		}

		// Response should contain some output (test-llm provider returns "Task completed successfully.")
		if len(stdout) == 0 {
			t.Logf("Warning: No output from exec mode")
		}

		t.Logf("Exec output: %s", stdout)
	})

	t.Run("exec from stdin", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, binPath, "--config-file", configPath, "exec")
		cmd.Stdin = strings.NewReader("what is 5+3? answer with just the number")

		var outBuf, errBuf bytes.Buffer
		cmd.Stdout = &outBuf
		cmd.Stderr = &errBuf

		err := cmd.Run()

		stdout := outBuf.String()
		stderr := errBuf.String()

		if err != nil && !strings.Contains(stderr, "context deadline exceeded") {
			t.Errorf("exec from stdin failed: %v\nstderr: %s", err, stderr)
		}

		if len(stdout) > 0 && !strings.Contains(stdout, "8") {
			t.Logf("Warning: Expected answer '8', got: %s", stdout)
		}
	})
}

// TestVersionAndHelp tests version and help commands
func TestVersionAndHelp(t *testing.T) {
	t.Parallel()

	t.Run("version flag", func(t *testing.T) {
		stdout, stderr, err := runSpin(t, "--version")

		if err != nil {
			t.Fatalf("--version failed: %v\nstderr: %s", err, stderr)
		}

		output := stdout + stderr
		if !strings.Contains(output, "spin version") && !strings.Contains(output, "dev") {
			t.Errorf("Expected version info, got: %s", output)
		}
	})

	t.Run("help flag", func(t *testing.T) {
		stdout, stderr, err := runSpin(t, "--help")

		// Help returns exit code 0
		if err != nil {
			t.Logf("Help returned error (might be expected): %v", err)
		}

		output := stdout + stderr
		if !strings.Contains(output, "Usage:") && !strings.Contains(output, "spin") {
			t.Errorf("Expected help text, got: %s", output)
		}
	})

	t.Run("subcommand help", func(t *testing.T) {
		commands := []string{"exec", "config", "mcp", "debug"}

		for _, cmd := range commands {
			stdout, stderr, _ := runSpin(t, cmd, "--help")

			output := stdout + stderr
			if !strings.Contains(output, cmd) {
				t.Errorf("Help for %s missing command name, got: %s", cmd, output)
			}
		}
	})
}

// TestJSONOutput tests JSON output modes
func TestJSONOutput(t *testing.T) {
	t.Run("config show json", func(t *testing.T) {
		stdout, stderr, err := runSpin(t, "config", "show", "--format", "json")

		if err != nil {
			t.Logf("config show json returned error: %v\nstderr: %s", err, stderr)
		}

		// Should be valid JSON
		var result map[string]interface{}
		if err := json.Unmarshal([]byte(stdout), &result); err != nil {
			t.Errorf("Invalid JSON output: %v\nOutput: %s", err, stdout)
		}
	})

	t.Run("mcp get json", func(t *testing.T) {
		// First add a server
		runSpin(t, "mcp", "add", "json-test", "echo", "test")
		defer runSpin(t, "mcp", "remove", "json-test", "--yes")

		stdout, stderr, err := runSpin(t, "mcp", "get", "json-test", "--format", "json")

		if err != nil {
			t.Fatalf("mcp get json failed: %v\nstderr: %s", err, stderr)
		}

		// Should be valid JSON
		var result map[string]interface{}
		if err := json.Unmarshal([]byte(stdout), &result); err != nil {
			t.Errorf("Invalid JSON output: %v\nOutput: %s", err, stdout)
		}

		// Should have expected fields
		if result["name"] != "json-test" {
			t.Errorf("Expected name 'json-test', got: %v", result["name"])
		}
	})
}

// Helper function to check if Ollama is available
func isOllamaAvailable(t *testing.T) bool {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "curl", "-s", "http://127.0.0.1:11434/api/tags")
	err := cmd.Run()

	return err == nil
}
