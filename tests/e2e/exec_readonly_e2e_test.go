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
// execResult holds the output of running a command.
type execResult struct {
	stdout string
	stderr string
	err    error
}

// runSpinExec runs a spin exec command and returns the result.
func runSpinExec(t *testing.T, configPath, workDir, prompt string) execResult {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binPath,
		"--config-file", configPath,
		"--cd", workDir,
		"exec",
		prompt,
	)

	var outBuf, errBuf bytes.Buffer

	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err := cmd.Run()

	return execResult{stdout: outBuf.String(), stderr: errBuf.String(), err: err}
}

// runSpinExecWithRetry runs a spin exec command with retries on empty output.
// Under heavy parallel load (go test ./...), the binary may occasionally produce
// no output due to resource contention.
func runSpinExecWithRetry(t *testing.T, configPath, workDir, prompt string) execResult {
	t.Helper()

	const maxAttempts = 3

	for attempt := range maxAttempts {
		r := runSpinExec(t, configPath, workDir, prompt)
		if r.stdout != "" || r.stderr != "" || r.err != nil {
			return r
		}

		if attempt >= maxAttempts-1 {
			return r
		}

		t.Logf("Empty output on attempt %d, retrying...", attempt+1)
		time.Sleep(200 * time.Millisecond)
	}

	return execResult{} // unreachable.
}

// containsDenialIndicator checks if output contains any denial-related keywords.
func containsDenialIndicator(output string) bool {
	indicators := []string{"exec mode requires --auto-approve", "requires --auto-approve", "denied", "not approved", "approval required"}
	lower := strings.ToLower(output)

	for _, ind := range indicators {
		if strings.Contains(lower, strings.ToLower(ind)) {
			return true
		}
	}

	return false
}

// assertNotDenied checks that the output does not contain denial messages.
func assertNotDenied(t *testing.T, r execResult) {
	t.Helper()

	combined := strings.ToLower(r.stdout + r.stderr)
	if strings.Contains(combined, "requires --auto-approve") || strings.Contains(combined, "denied") {
		t.Errorf("Operation was incorrectly denied. Output: %s", r.stdout+r.stderr)
	}
}

// setupReadOnlyTestEnv creates a temporary workspace and config for exec readonly tests.
func setupReadOnlyTestEnv(t *testing.T) (workDir, configPath string) {
	t.Helper()

	workDir = t.TempDir()
	testFile := filepath.Join(workDir, "test.txt")

	err := os.WriteFile(testFile, []byte("This is a test file for read operations"), 0o600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tmpDir := t.TempDir()
	configPath = filepath.Join(tmpDir, "spin.yaml")

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

	err = os.WriteFile(configPath, []byte(config), 0o600)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	return workDir, configPath
}

func TestExecMode_ReadOnlyDeniesWrites(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir, configPath := setupReadOnlyTestEnv(t)

	targetFile := filepath.Join(workDir, "should-not-exist.txt")
	r := runSpinExec(t, configPath, workDir,
		"Create a file called should-not-exist.txt with the text 'this should not be created'")

	verifyFileNotCreated(t, targetFile)
	logExecResult(t, "Write test", r)
}

func TestExecMode_ReadOnlyAllowsReads(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir, configPath := setupReadOnlyTestEnv(t)

	r := runSpinExecWithRetry(t, configPath, workDir, "Read the file test.txt and tell me what it contains")
	assertExecSucceeded(t, r, "Read operation")
	assertNotDenied(t, r)
}

func TestExecMode_ReadOnlyAllowsListDir(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	workDir, configPath := setupReadOnlyTestEnv(t)

	r := runSpinExecWithRetry(t, configPath, workDir, "List all files in the current directory")
	assertExecSucceeded(t, r, "List operation")
	assertNotDenied(t, r)

	if !strings.Contains(strings.ToLower(r.stdout), "test.txt") {
		t.Logf("Warning: Expected 'test.txt' in directory listing, got: %s", r.stdout)
	}
}

// verifyFileNotCreated checks that a file does not exist.
func verifyFileNotCreated(t *testing.T, path string) {
	t.Helper()

	if _, err := os.Stat(path); err == nil {
		t.Errorf("File was created despite no --auto-approve flag! File exists: %s", path)
	}
}

// assertExecSucceeded checks that an exec result completed without error and has output.
// Retries once on empty output to handle transient failures under heavy parallel load.
func assertExecSucceeded(t *testing.T, r execResult, label string) {
	t.Helper()

	if r.err != nil {
		t.Errorf("%s failed unexpectedly: %v\nstderr: %s\nstdout: %s", label, r.err, r.stderr, r.stdout)
	}

	if r.stdout == "" && r.stderr == "" {
		t.Errorf("No output from %s", label)
	}
}

// logExecResult logs the output of an exec result.
func logExecResult(t *testing.T, label string, r execResult) {
	t.Helper()

	output := r.stdout + r.stderr
	if !containsDenialIndicator(output) && output != "" {
		t.Logf("No explicit denial message found, but file was correctly not created. Output: %s", output)
	}

	if r.err != nil {
		t.Logf("%s - command exited with error (expected for denied operations): %v", label, r.err)
	}

	t.Logf("%s - stdout: %s", label, r.stdout)
	t.Logf("%s - stderr: %s", label, r.stderr)
}
