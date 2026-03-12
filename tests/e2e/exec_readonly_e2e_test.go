// Package e2e contains end-to-end tests for the Spin binary.
package e2e

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestExecMode_ReadOnlyDeniesWrites tests that exec mode without --auto-approve
// denies write operations while allowing read operations.
// This test covers the scenario documented in docs/job-ci-automation.md Flow 3.
// This test uses the test-llm provider (requires e2e_llm_test build tag).
func TestExecMode_ReadOnlyDeniesWrites(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	t.Parallel()

	// Create temporary workspace.
	workDir := t.TempDir()

	// Create a test file for read operations.
	testFile := filepath.Join(workDir, "test.txt")

	testContent := "This is a test file for read operations"
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create temporary config with test-llm provider (no external LLM required).
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "spin.yaml")

	config := `version: "2.0"
llm:
  provider: test-llm
  model: dummy
  temperature: 0.7
  max_tokens: 4096

protocol:
  enable_shell: true
  enable_git: true

security:
  sandbox:
    mode: workspace-write
`

	err = os.WriteFile(configPath, []byte(config), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	t.Run("write operation denied without auto-approve", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()

		// Target file that should NOT be created.
		targetFile := filepath.Join(workDir, "should-not-exist.txt")

		// Run exec without --auto-approve, asking to create a file.
		cmd := exec.CommandContext(ctx, binPath,
			"--config-file", configPath,
			"--cd", workDir,
			"exec",
			"Create a file called should-not-exist.txt with the text 'this should not be created'",
		)

		var outBuf, errBuf bytes.Buffer

		cmd.Stdout = &outBuf
		cmd.Stderr = &errBuf

		runErr := cmd.Run()
		stdout := outBuf.String()
		stderr := errBuf.String()

		// Check if file was created (it should NOT be).
		_, statErr := os.Stat(targetFile)
		if statErr == nil {
			t.Errorf("File was created despite no --auto-approve flag! File exists: %s", targetFile)
		}

		// Check for denial message in output
		// The denial might appear in stdout (agent response) or stderr (error message).
		output := stdout + stderr
		denialIndicators := []string{
			"exec mode requires --auto-approve",
			"requires --auto-approve",
			"denied",
			"not approved",
			"approval required",
		}

		foundDenial := false

		for _, indicator := range denialIndicators {
			if strings.Contains(strings.ToLower(output), strings.ToLower(indicator)) {
				foundDenial = true

				break
			}
		}

		if !foundDenial && len(output) > 0 {
			// If we got output but no clear denial message, log it for debugging
			// The agent might have responded differently, but the file should still not exist.
			t.Logf("No explicit denial message found, but file was correctly not created. Output: %s", output)
		}

		// The command may succeed (agent responds) or fail (approval denied)
		// What matters is that the file was not created.
		if runErr != nil {
			t.Logf("Write test - command exited with error (expected for denied operations): %v", runErr)
		}

		t.Logf("Write test - stdout: %s", stdout)
		t.Logf("Write test - stderr: %s", stderr)
	})

	t.Run("read operation works without auto-approve", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()

		// Run exec without --auto-approve, asking to read the test file.
		cmd := exec.CommandContext(ctx, binPath,
			"--config-file", configPath,
			"--cd", workDir,
			"exec",
			"Read the file test.txt and tell me what it contains",
		)

		var outBuf, errBuf bytes.Buffer

		cmd.Stdout = &outBuf
		cmd.Stderr = &errBuf

		runErr := cmd.Run()
		stdout := outBuf.String()
		stderr := errBuf.String()

		// Read operations should succeed.
		if runErr != nil {
			t.Errorf("Read operation failed unexpectedly: %v\nstderr: %s\nstdout: %s", runErr, stderr, stdout)
		}

		// Should have output.
		if len(stdout) == 0 {
			t.Errorf("No output from read operation\nstderr: %s", stderr)
		}

		// Should contain the test file content (or at least part of it)
		// The agent might paraphrase, but should mention the content.
		if !strings.Contains(strings.ToLower(stdout), strings.ToLower("test file")) &&
			!strings.Contains(strings.ToLower(stdout), strings.ToLower("read operations")) {
			t.Logf("Warning: Expected test file content in response, got: %s", stdout)
		}

		// Should NOT contain denial messages for read operations.
		if strings.Contains(strings.ToLower(stdout+stderr), "requires --auto-approve") ||
			strings.Contains(strings.ToLower(stdout+stderr), "denied") {
			t.Errorf("Read operation was incorrectly denied. Output: %s", stdout+stderr)
		}

		t.Logf("Read test - stdout: %s", stdout)
		t.Logf("Read test - stderr: %s", stderr)
	})

	t.Run("list directory works without auto-approve", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()

		// Run exec without --auto-approve, asking to list directory.
		cmd := exec.CommandContext(ctx, binPath,
			"--config-file", configPath,
			"--cd", workDir,
			"exec",
			"List all files in the current directory",
		)

		var outBuf, errBuf bytes.Buffer

		cmd.Stdout = &outBuf
		cmd.Stderr = &errBuf

		runErr := cmd.Run()
		stdout := outBuf.String()
		stderr := errBuf.String()

		// List operations should succeed.
		if runErr != nil {
			t.Errorf("List operation failed unexpectedly: %v\nstderr: %s\nstdout: %s", runErr, stderr, stdout)
		}

		// Should have output.
		if len(stdout) == 0 {
			t.Errorf("No output from list operation\nstderr: %s", stderr)
		}

		// Should mention the test file.
		if !strings.Contains(strings.ToLower(stdout), "test.txt") {
			t.Logf("Warning: Expected 'test.txt' in directory listing, got: %s", stdout)
		}

		// Should NOT contain denial messages for list operations.
		if strings.Contains(strings.ToLower(stdout+stderr), "requires --auto-approve") ||
			strings.Contains(strings.ToLower(stdout+stderr), "denied") {
			t.Errorf("List operation was incorrectly denied. Output: %s", stdout+stderr)
		}

		t.Logf("List test - stdout: %s", stdout)
	})
}
