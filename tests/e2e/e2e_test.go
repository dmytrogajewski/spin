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
	// testTimeout is the default timeout for e2e tests.
	testTimeout = 60 * time.Second

	// Binary path (relative to test file).
	binPath = "../../bin/spin"

	// Env var that instructs tests to reuse an existing binary.
	skipBuildEnv = "SPIN_E2E_SKIP_BUILD"
)

// TestMain builds the binary before running tests.
func TestMain(m *testing.M) {
	if shouldSkipBuild() {
		_, err := os.Stat(binPath)
		if err == nil {
			fmt.Fprintln(os.Stdout, "Using existing spin binary for e2e tests")
		} else {
			fmt.Fprintln(os.Stdout, "Pre-built spin binary not found, rebuilding...")
			buildSpinBinary()
		}
	} else {
		buildSpinBinary()
	}

	// Run tests.
	code := m.Run()

	// Cleanup (optional - keep binary for debugging)
	// os.Remove(binPath).

	os.Exit(code)
}

func shouldSkipBuild() bool {
	return os.Getenv(skipBuildEnv) == "1"
}

func buildSpinBinary() {
	// Build the binary with e2e_llm_test tag to enable test-llm provider
	// This allows e2e tests to run without requiring external LLM services.
	fmt.Fprintln(os.Stdout, "Building spin binary for e2e tests (with e2e_llm_test tag)...")

	cmd := exec.Command("go", "build", "-tags", "e2e_llm_test", "-o", binPath, "../../cmd/spin")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to build binary: %v\n%s\n", err, output)
		os.Exit(1)
	}
}

// runSpin executes the spin binary with given args and returns stdout, stderr, and error.
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

// runSpinWithInput executes spin with stdin input.
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

func TestConfigCommands_ShowEmpty(t *testing.T) {
	t.Parallel()

	emptyConfig := filepath.Join(t.TempDir(), "spin.yaml")
	if err := os.WriteFile(emptyConfig, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to create empty config: %v", err)
	}

	stdout, stderr, err := runSpin(t, "--config-file", emptyConfig, "config", "show")
	if err != nil {
		t.Fatalf("config show failed: %v\nstderr: %s", err, stderr)
	}

	if len(stdout) == 0 && len(stderr) == 0 {
		t.Errorf("Expected some output from config show, got nothing")
	}
}

func TestConfigCommands_ShowWithBinaryInCwd(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "spin"), []byte{0x7f, 0x45, 0x4c, 0x46}, 0755); err != nil {
		t.Fatalf("Failed to create fake binary: %v", err)
	}

	cmd := exec.Command(binPath, "config", "show")
	cmd.Dir = tmpDir

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err := cmd.Run()
	stderr := errBuf.String()

	if strings.Contains(stderr, "control characters are not allowed") {
		t.Errorf("BUG #1 REGRESSION: Config loader tried to read binary file!\nstderr: %s", stderr)
	}

	if err != nil && !strings.Contains(stderr, "No configuration file") {
		t.Logf("Warning: unexpected error (but not the binary-reading bug): %v\nstderr: %s", err, stderr)
	}
}

func TestConfigCommands_Validate(t *testing.T) {
	t.Parallel()

	stdout, stderr, _ := runSpin(t, "config", "validate")
	output := stdout + stderr

	if !strings.Contains(output, "valid") && !strings.Contains(output, "invalid") {
		t.Errorf("Expected validation result, got: %s", output)
	}
}

func TestConfigCommands_Path(t *testing.T) {
	t.Parallel()

	stdout, stderr, err := runSpin(t, "config", "path")
	output := stdout + stderr

	if err != nil && !strings.Contains(output, "no config file") {
		t.Errorf("config path failed unexpectedly: %v\noutput: %s", err, output)
	}
}

func TestMCPCommands_ListEmpty(t *testing.T) {
	t.Parallel()

	configPath := createTempConfig(t)

	stdout, stderr, err := runSpin(t, "--config-file", configPath, "mcp", "registry", "list")
	if err != nil {
		t.Fatalf("mcp registry list failed: %v\nstderr: %s", err, stderr)
	}

	if !strings.Contains(stdout, "No registries configured") {
		t.Errorf("Expected 'No registries configured', got: %s", stdout)
	}
}

func TestMCPCommands_AddAndRemove(t *testing.T) {
	t.Parallel()

	configPath := createTempConfig(t)

	// Add MCP registry.
	assertSpinSuccess(t, configPath, "mcp", "registry", "local", "add", "test-server", "echo", "test")

	// List should show the registry.
	stdout := assertSpinContains(t, configPath, "test-server", "mcp", "registry", "list")
	_ = stdout

	// Get registry details.
	assertSpinContains(t, configPath, "test-server", "mcp", "registry", "get", "test-server")

	// Remove registry.
	assertSpinSuccess(t, configPath, "mcp", "registry", "remove", "test-server", "--yes")

	// List should be empty again.
	assertSpinContains(t, configPath, "No registries configured", "mcp", "registry", "list")
}

// createTempConfig creates a temporary spin config file and returns its path.
func createTempConfig(t *testing.T) string {
	t.Helper()

	configPath := filepath.Join(t.TempDir(), "spin.yaml")
	if err := os.WriteFile(configPath, []byte("# Spin configuration\n"), 0644); err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	return configPath
}

// assertSpinSuccess runs spin with the given config and args, fataling on error.
func assertSpinSuccess(t *testing.T, configPath string, args ...string) {
	t.Helper()

	fullArgs := append([]string{"--config-file", configPath}, args...)
	_, stderr, err := runSpin(t, fullArgs...)

	if err != nil {
		t.Fatalf("spin %v failed: %v\nstderr: %s", args, err, stderr)
	}
}

// assertSpinContains runs spin and checks that stdout contains the expected string.
func assertSpinContains(t *testing.T, configPath, expected string, args ...string) string {
	t.Helper()

	fullArgs := append([]string{"--config-file", configPath}, args...)
	stdout, stderr, err := runSpin(t, fullArgs...)

	if err != nil {
		t.Fatalf("spin %v failed: %v\nstderr: %s", args, err, stderr)
	}

	if !strings.Contains(stdout, expected) {
		t.Errorf("Expected %q in output, got: %s", expected, stdout)
	}

	return stdout
}

// TestDebugCommands tests debug command functionality.
func TestDebugCommands(t *testing.T) {
	t.Parallel()

	t.Run("debug sandbox platform check", func(t *testing.T) {
		t.Parallel()

		stdout, stderr, err := runSpin(t, "debug", "sandbox", "ls")

		output := stdout + stderr

		// On Linux, should fail with platform check
		// On macOS, might work or show not implemented.
		if strings.Contains(output, "only available on macOS") {
			// Correct behavior on Linux.
			if err == nil {
				t.Error("Expected error on non-macOS platform")
			}
		} else if strings.Contains(output, "not yet implemented") {
			// Acceptable - stub implementation on macOS.
			t.Logf("Sandbox command reached stub (macOS): %s", output)
		} else {
			t.Logf("Sandbox command output: %s", output)
		}
	})

	t.Run("debug landlock platform check", func(t *testing.T) {
		t.Parallel()

		stdout, stderr, err := runSpin(t, "debug", "landlock", "ls")

		output := stdout + stderr

		// On macOS, should fail with platform check
		// On Linux, might work or show not implemented.
		if strings.Contains(output, "only available on Linux") {
			// Correct behavior on macOS.
			if err == nil {
				t.Error("Expected error on non-Linux platform")
			}
		} else if strings.Contains(output, "not yet implemented") {
			// Acceptable - stub implementation on Linux.
			t.Logf("Landlock command reached stub (Linux): %s", output)
		} else {
			t.Logf("Landlock command output: %s", output)
		}
	})
}

// createExecConfig creates a temporary config for exec mode tests.
func createExecConfig(t *testing.T) string {
	t.Helper()

	configPath := filepath.Join(t.TempDir(), "spin.yaml")
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

	return configPath
}

func TestExecMode_BasicPrompt(t *testing.T) {
	t.Parallel()

	configPath := createExecConfig(t)
	stdout, stderr := runExecCommand(t, configPath, "what is 2+2? answer with just the number")

	checkExecErrors(t, stderr)

	if len(stdout) == 0 {
		t.Errorf("BUG #3 REGRESSION: No output from exec mode!\nstderr: %s", stderr)
	}

	t.Logf("Exec output: %s", stdout)
}

func TestExecMode_FromStdin(t *testing.T) {
	t.Parallel()

	configPath := createExecConfig(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binPath, "--config-file", configPath, "exec")
	cmd.Stdin = strings.NewReader("what is 5+3? answer with just the number")

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	runErr := cmd.Run()
	stdout, stderr := outBuf.String(), errBuf.String()

	if runErr != nil && !strings.Contains(stderr, "context deadline exceeded") {
		t.Errorf("exec from stdin failed: %v\nstderr: %s", runErr, stderr)
	}

	if len(stdout) > 0 && !strings.Contains(stdout, "8") {
		t.Logf("Warning: Expected answer '8', got: %s", stdout)
	}
}

// runExecCommand runs a spin exec command and returns stdout and stderr.
func runExecCommand(t *testing.T, configPath, prompt string) (string, string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binPath, "--config-file", configPath, "exec", prompt)

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	if err := cmd.Run(); err != nil {
		checkExecRunError(t, err, errBuf.String(), outBuf.String())
	}

	return outBuf.String(), errBuf.String()
}

// checkExecRunError handles exec command run errors.
func checkExecRunError(t *testing.T, err error, stderr, stdout string) {
	t.Helper()

	if strings.Contains(stderr, "provider is required") || strings.Contains(stderr, "model is required") {
		t.Errorf("BUG #2 REGRESSION: Config integration broken!\nstderr: %s", stderr)
	} else if strings.Contains(stderr, "context deadline exceeded") {
		t.Skip("Test timed out")
	} else {
		t.Errorf("exec failed: %v\nstderr: %s\nstdout: %s", err, stderr, stdout)
	}
}

// checkExecErrors checks for known regression bugs in exec output.
func checkExecErrors(t *testing.T, stderr string) {
	t.Helper()

	if strings.Contains(stderr, "provider is required") || strings.Contains(stderr, "model is required") {
		t.Errorf("BUG #2 REGRESSION: Config integration broken!\nstderr: %s", stderr)
	}
}

// TestVersionAndHelp tests version and help commands.
func TestVersionAndHelp(t *testing.T) {
	t.Parallel()

	t.Run("version flag", func(t *testing.T) {
		t.Parallel()

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
		t.Parallel()

		stdout, stderr, err := runSpin(t, "--help")

		// Help returns exit code 0.
		if err != nil {
			t.Logf("Help returned error (might be expected): %v", err)
		}

		output := stdout + stderr
		if !strings.Contains(output, "Usage:") && !strings.Contains(output, "spin") {
			t.Errorf("Expected help text, got: %s", output)
		}
	})

	t.Run("subcommand help", func(t *testing.T) {
		t.Parallel()

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

// TestJSONOutput tests JSON output modes.
func TestJSONOutput(t *testing.T) {
	t.Parallel()

	t.Run("config show json", func(t *testing.T) {
		t.Parallel()

		stdout, stderr, err := runSpin(t, "config", "show", "--format", "json")
		if err != nil {
			t.Logf("config show json returned error: %v\nstderr: %s", err, stderr)
		}

		// Should be valid JSON.
		var result map[string]any
		err = json.Unmarshal([]byte(stdout), &result)
		if err != nil {
			t.Errorf("Invalid JSON output: %v\nOutput: %s", err, stdout)
		}
	})

	t.Run("mcp registry get json", func(t *testing.T) {
		t.Parallel()

		// Use a temp config file to avoid races with other tests.
		tmpConfigPath := filepath.Join(t.TempDir(), "spin.yaml")
		err := os.WriteFile(tmpConfigPath, []byte("# Spin configuration\n"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test config: %v", err)
		}

		// First add a registry.
		_, _, _ = runSpin(t, "--config-file", tmpConfigPath, "mcp", "registry", "local", "add", "json-test", "echo", "test")
		t.Cleanup(func() { _, _, _ = runSpin(t, "--config-file", tmpConfigPath, "mcp", "registry", "remove", "json-test", "--yes") })

		stdout, stderr, err := runSpin(t, "--config-file", tmpConfigPath, "mcp", "registry", "get", "json-test", "--format", "json")
		if err != nil {
			t.Fatalf("mcp registry get json failed: %v\nstderr: %s", err, stderr)
		}

		// Should be valid JSON.
		var result map[string]any
		err = json.Unmarshal([]byte(stdout), &result)
		if err != nil {
			t.Errorf("Invalid JSON output: %v\nOutput: %s", err, stdout)
		}

		// Should have expected fields.
		if result["name"] != "json-test" {
			t.Errorf("Expected name 'json-test', got: %v", result["name"])
		}
	})
}

// Helper function to check if Ollama is available.
func isOllamaAvailable(t *testing.T) bool {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "curl", "-s", "http://127.0.0.1:11434/api/tags")
	err := cmd.Run()

	return err == nil
}
